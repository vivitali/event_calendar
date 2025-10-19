package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"time"
)

type CallbackQuery struct {
	ID      string `json:"id"`
	From    User   `json:"from"`
	Message struct {
		MessageID int    `json:"message_id"`
		Chat      Chat   `json:"chat"`
		Text      string `json:"text"`
	} `json:"message"`
	Data string `json:"data"`
}

type User struct {
	ID        int    `json:"id"`
	FirstName string `json:"first_name"`
	LastName  string `json:"last_name,omitempty"`
	Username  string `json:"username,omitempty"`
}

type Chat struct {
	ID    int64  `json:"id"`
	Title string `json:"title,omitempty"`
	Type  string `json:"type"`
}

type AnswerCallbackQueryRequest struct {
	CallbackQueryID string `json:"callback_query_id"`
	Text            string `json:"text,omitempty"`
	ShowAlert       bool   `json:"show_alert,omitempty"`
}

type VoteRecord struct {
	UserID   int       `json:"user_id"`
	Username string    `json:"username"`
	Vote     string    `json:"vote"`
	Timestamp time.Time `json:"timestamp"`
}

// Simple in-memory storage for votes (in production, use a database)
var voteStorage = make(map[string][]VoteRecord)

func (s *Service) HandleCallbackQuery(callbackQuery CallbackQuery) error {
	// Answer the callback query first
	err := s.answerCallbackQuery(callbackQuery.ID, "")
	if err != nil {
		return fmt.Errorf("failed to answer callback query: %v", err)
	}

	// Process the vote
	vote := callbackQuery.Data
	user := callbackQuery.From
	
	// Extract event ID if it's an event-specific vote
	if strings.HasPrefix(vote, "event_") {
		parts := strings.Split(vote, "_")
		if len(parts) >= 3 {
			eventID := parts[1]
			voteType := parts[2]
			
			// Record the vote
			voteKey := fmt.Sprintf("event_%s", eventID)
			record := VoteRecord{
				UserID:    user.ID,
				Username:  user.Username,
				Vote:      voteType,
				Timestamp: time.Now(),
			}
			
			// Remove existing vote from this user
			votes := voteStorage[voteKey]
			for i, existingVote := range votes {
				if existingVote.UserID == user.ID {
					votes = append(votes[:i], votes[i+1:]...)
					break
				}
			}
			
			// Add new vote
			votes = append(votes, record)
			voteStorage[voteKey] = votes
			
			// Update message with vote results
			err = s.updateMessageWithVotes(callbackQuery.Message.Chat.ID, callbackQuery.Message.MessageID, eventID, voteKey)
			if err != nil {
				return fmt.Errorf("failed to update message: %v", err)
			}
		}
	} else {
		// Handle general voting
		switch vote {
		case "vote_going", "vote_maybe", "vote_not_going":
			// Record general vote
			voteKey := "general_vote"
			record := VoteRecord{
				UserID:    user.ID,
				Username:  user.Username,
				Vote:      strings.TrimPrefix(vote, "vote_"),
				Timestamp: time.Now(),
			}
			
			// Remove existing vote from this user
			votes := voteStorage[voteKey]
			for i, existingVote := range votes {
				if existingVote.UserID == user.ID {
					votes = append(votes[:i], votes[i+1:]...)
					break
				}
			}
			
			// Add new vote
			votes = append(votes, record)
			voteStorage[voteKey] = votes
			
			// Send confirmation message
			confirmation := s.getVoteConfirmation(vote, user.FirstName)
			err = s.SendMessage(fmt.Sprintf("%d", callbackQuery.Message.Chat.ID), confirmation)
			if err != nil {
				return fmt.Errorf("failed to send confirmation: %v", err)
			}
			
		case "vote_results":
			// Show vote results
			results := s.getVoteResults("general_vote")
			err = s.SendMessage(fmt.Sprintf("%d", callbackQuery.Message.Chat.ID), results)
			if err != nil {
				return fmt.Errorf("failed to send results: %v", err)
			}
		}
	}

	return nil
}

func (s *Service) answerCallbackQuery(callbackQueryID, text string) error {
	request := AnswerCallbackQueryRequest{
		CallbackQueryID: callbackQueryID,
		Text:            text,
		ShowAlert:       false,
	}

	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal request: %v", err)
	}

	url := s.baseURL + "/answerCallbackQuery"
	resp, err := s.client.Post(url, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to send request: %v", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("failed to read response: %v", err)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(body, &response); err != nil {
		return fmt.Errorf("failed to parse response: %v", err)
	}

	if !response["ok"].(bool) {
		return fmt.Errorf("telegram API error: %v", response["description"])
	}

	return nil
}

func (s *Service) updateMessageWithVotes(chatID int64, messageID int, eventID, voteKey string) error {
	// This would update the original message with vote counts
	// For now, just send a new message with results
	votes := voteStorage[voteKey]
	if len(votes) == 0 {
		return nil
	}

	// Count votes
	goingCount := 0
	maybeCount := 0
	notGoingCount := 0
	
	for _, vote := range votes {
		switch vote.Vote {
		case "going":
			goingCount++
		case "maybe":
			maybeCount++
		case "not_going":
			notGoingCount++
		}
	}

	results := fmt.Sprintf("📊 **Vote Results for Event %s:**\n\n", eventID)
	results += fmt.Sprintf("👍 Going: %d\n", goingCount)
	results += fmt.Sprintf("🤔 Maybe: %d\n", maybeCount)
	results += fmt.Sprintf("❌ Not Going: %d\n", notGoingCount)
	results += fmt.Sprintf("\nTotal votes: %d", len(votes))

	return s.SendMessage(fmt.Sprintf("%d", chatID), results)
}

func (s *Service) getVoteConfirmation(vote, userName string) string {
	switch vote {
	case "vote_going":
		return fmt.Sprintf("👍 Thanks %s! You're going to the event!", userName)
	case "vote_maybe":
		return fmt.Sprintf("🤔 Thanks %s! You marked yourself as maybe for the event.", userName)
	case "vote_not_going":
		return fmt.Sprintf("❌ Thanks %s! You marked yourself as not going to the event.", userName)
	default:
		return "✅ Vote recorded!"
	}
}

func (s *Service) getVoteResults(voteKey string) string {
	votes := voteStorage[voteKey]
	if len(votes) == 0 {
		return "📊 No votes recorded yet."
	}

	// Count votes
	goingCount := 0
	maybeCount := 0
	notGoingCount := 0
	
	for _, vote := range votes {
		switch vote.Vote {
		case "going":
			goingCount++
		case "maybe":
			maybeCount++
		case "not_going":
			notGoingCount++
		}
	}

	results := "📊 **Overall Vote Results:**\n\n"
	results += fmt.Sprintf("👍 Going: %d\n", goingCount)
	results += fmt.Sprintf("🤔 Maybe: %d\n", maybeCount)
	results += fmt.Sprintf("❌ Not Going: %d\n", notGoingCount)
	results += fmt.Sprintf("\nTotal votes: %d", len(votes))

	return results
}
