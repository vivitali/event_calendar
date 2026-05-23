// Package telegram provides a minimal Telegram Bot API client used to send
// event digests, polls, and alerts.
package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Service is a thin HTTP client over the Telegram Bot API.
type Service struct {
	botToken string
	client   *http.Client
	baseURL  string
}

// NewService constructs a Service for the given bot token.
func NewService(botToken string) *Service {
	return &Service{
		botToken: botToken,
		client:   &http.Client{Timeout: 30 * time.Second},
		baseURL:  "https://api.telegram.org/bot" + botToken,
	}
}

type sendMessageRequest struct {
	ChatID                string `json:"chat_id"`
	Text                  string `json:"text"`
	ParseMode             string `json:"parse_mode,omitempty"`
	DisableWebPagePreview bool   `json:"disable_web_page_preview,omitempty"`
}

type apiResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
}

// SendMessage sends a Markdown-formatted message to chatID.
func (s *Service) SendMessage(chatID, message string) error {
	if s.botToken == "" {
		return fmt.Errorf("bot token not configured")
	}
	if chatID == "" {
		return fmt.Errorf("chat ID not provided")
	}
	if message == "" {
		return fmt.Errorf("message is empty")
	}
	if len(message) > 4096 {
		return fmt.Errorf("message too long (%d characters, max 4096)", len(message))
	}
	return s.post("/sendMessage", sendMessageRequest{
		ChatID:                chatID,
		Text:                  message,
		ParseMode:             "MarkdownV2",
		DisableWebPagePreview: true,
	})
}

// SendAlert wraps the message in a standard alert template.
func (s *Service) SendAlert(chatID, alertMessage string) error {
	body := fmt.Sprintf("🚨 *Winnipeg Tech Events Alert*\n\n%s\n\n_Time: %s_",
		alertMessage, time.Now().Format("2006-01-02 15:04:05 MST"))
	return s.SendMessage(chatID, body)
}

// TestConnection calls /getMe to verify the bot token works.
func (s *Service) TestConnection() error {
	if s.botToken == "" {
		return fmt.Errorf("bot token not configured")
	}
	resp, err := s.client.Get(s.baseURL + "/getMe")
	if err != nil {
		return fmt.Errorf("failed to test connection: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("connection test failed with status: %d", resp.StatusCode)
	}
	return nil
}

type sendPollRequest struct {
	ChatID                string   `json:"chat_id"`
	Question              string   `json:"question"`
	Options               []string `json:"options"`
	IsAnonymous           bool     `json:"is_anonymous"`
	Type                  string   `json:"type"`
	AllowsMultipleAnswers bool     `json:"allows_multiple_answers"`
}

// SendPoll sends a non-anonymous poll. 2-10 options required by Telegram.
func (s *Service) SendPoll(chatID, question string, options []string, allowMultiple bool) error {
	if s.botToken == "" {
		return fmt.Errorf("bot token not configured")
	}
	if chatID == "" {
		return fmt.Errorf("chat ID not provided")
	}
	if question == "" {
		return fmt.Errorf("question is empty")
	}
	if len(options) < 2 || len(options) > 10 {
		return fmt.Errorf("poll requires 2-10 options, got %d", len(options))
	}
	return s.post("/sendPoll", sendPollRequest{
		ChatID:                chatID,
		Question:              question,
		Options:               options,
		IsAnonymous:           false,
		Type:                  "regular",
		AllowsMultipleAnswers: allowMultiple,
	})
}

// SendMonthlyMeetupPoll posts the standard monthly day-of-week poll.
func (s *Service) SendMonthlyMeetupPoll(chatID string) error {
	return s.SendPoll(chatID,
		"Є бажаючі зустрітись - виберіть день тижня",
		[]string{"Понеділок", "Вівторок", "Середа", "Четвер", "П'ятниця", "Субота", "Неділя"},
		true,
	)
}

func (s *Service) post(path string, body any) error {
	jsonData, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	resp, err := s.client.Post(s.baseURL+path, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("HTTP request: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	var r apiResponse
	if err := json.Unmarshal(respBody, &r); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}
	if !r.OK {
		return fmt.Errorf("telegram API error: %s", r.Description)
	}
	return nil
}
