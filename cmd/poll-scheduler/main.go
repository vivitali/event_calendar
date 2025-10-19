package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"event_calendar/pkg/telegram"
)

type PollSchedulerConfig struct {
	PollBotToken string `json:"poll_bot_token"`
	PollChatID   string `json:"poll_chat_id"`
	TestMode     bool   `json:"test_mode"`
}

type PollSchedulerResult struct {
	Success   bool      `json:"success"`
	PollSent  bool      `json:"poll_sent"`
	Timestamp time.Time `json:"timestamp"`
	Error     string    `json:"error,omitempty"`
	Logs      []string  `json:"logs"`
}

func main() {
	log.Println("📊 Monthly Meetup Poll Scheduler Starting...")
	
	// Load configuration
	config := loadConfig()
	
	// Run the poll scheduler
	result := runPollScheduler(config)
	
	// Log results
	logResult(result)
	
	// Exit with appropriate code
	if result.Success {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
}

func loadConfig() *PollSchedulerConfig {
	config := &PollSchedulerConfig{
		TestMode: false,
	}
	
	// Load from environment variables
	if pollBotToken := os.Getenv("TELEGRAM_POLL_BOT_TOKEN"); pollBotToken != "" {
		config.PollBotToken = pollBotToken
	}
	
	if pollChatID := os.Getenv("TELEGRAM_POLL_CHAT_ID"); pollChatID != "" {
		config.PollChatID = pollChatID
	}
	
	if testMode := os.Getenv("TEST_MODE"); testMode == "true" {
		config.TestMode = true
	}
	
	log.Printf("📋 Poll Configuration loaded: TestMode=%t", config.TestMode)
	
	return config
}

func runPollScheduler(config *PollSchedulerConfig) *PollSchedulerResult {
	result := &PollSchedulerResult{
		Timestamp: time.Now(),
		Logs:      []string{},
	}
	
	// Check if it's the 20th of the month
	if !is20thOfMonth() {
		result.Logs = append(result.Logs, "Not the 20th of the month, skipping poll")
		result.Success = true
		return result
	}
	
	result.Logs = append(result.Logs, "Today is the 20th of the month, proceeding with poll")
	
	// Check configuration
	if config.PollBotToken == "" || config.PollChatID == "" {
		result.Error = "Telegram poll bot token or poll chat ID not configured"
		result.Logs = append(result.Logs, result.Error)
		return result
	}
	
	// Check if we should actually send
	if config.TestMode {
		result.Logs = append(result.Logs, "🧪 TEST MODE: Poll would be sent but not actually posted")
		result.Success = true
		result.PollSent = false
		return result
	}
	
	// Send poll to Telegram
	log.Println("📤 Sending monthly meetup poll to Telegram...")
	telegramService := telegram.NewService(config.PollBotToken)
	
	err := telegramService.SendMonthlyMeetupPoll(config.PollChatID)
	
	if err != nil {
		result.Error = fmt.Sprintf("Failed to send monthly poll: %v", err)
		result.Logs = append(result.Logs, result.Error)
		return result
	}
	
	result.Success = true
	result.PollSent = true
	result.Logs = append(result.Logs, "✅ Monthly meetup poll sent successfully")
	
	return result
}

func is20thOfMonth() bool {
	now := time.Now()
	return now.Day() == 20
}

func logResult(result *PollSchedulerResult) {
	log.Printf("📊 Poll Scheduler Result: Success=%t, PollSent=%t", 
		result.Success, result.PollSent)
	
	for _, logMsg := range result.Logs {
		log.Printf("📝 %s", logMsg)
	}
	
	if result.Error != "" {
		log.Printf("❌ Error: %s", result.Error)
	}
	
	// Output JSON result for external processing
	if jsonResult, err := json.MarshalIndent(result, "", "  "); err == nil {
		log.Printf("📋 JSON Result: %s", string(jsonResult))
	}
}
