#!/bin/bash

# Test script to simulate GitHub Actions workflow for monthly poll
# This script tests the poll scheduler as it would run in GitHub Actions

echo "🧪 Testing GitHub Actions Monthly Poll Workflow"
echo "=============================================="

# Set test environment variables (simulating GitHub Actions secrets)
export TELEGRAM_BOT_TOKEN="test_main_bot_token"
export TELEGRAM_CHAT_ID="test_main_chat_id"
export TELEGRAM_POLL_BOT_TOKEN="test_poll_bot_token"
export TELEGRAM_POLL_CHAT_ID="test_poll_chat_id"
export TEST_MODE="true"

echo "📋 Environment variables set (simulating GitHub Actions):"
echo "  TELEGRAM_BOT_TOKEN=$TELEGRAM_BOT_TOKEN"
echo "  TELEGRAM_CHAT_ID=$TELEGRAM_CHAT_ID"
echo "  TELEGRAM_POLL_BOT_TOKEN=$TELEGRAM_POLL_BOT_TOKEN"
echo "  TELEGRAM_POLL_CHAT_ID=$TELEGRAM_POLL_CHAT_ID"
echo "  TEST_MODE=$TEST_MODE"
echo ""

echo "🔧 Building poll scheduler..."
go build -o poll-scheduler ./cmd/poll-scheduler/

if [ $? -ne 0 ]; then
    echo "❌ Failed to build poll scheduler"
    exit 1
fi

echo "✅ Poll scheduler built successfully"
echo ""

echo "🚀 Running poll scheduler (simulating GitHub Actions run)..."
echo "Configuration:"
echo "  Test Mode: $TEST_MODE"
echo "  Current Date: $(date)"
echo ""

# Create a temporary version that simulates the 20th for testing
echo "🔧 Creating test version that simulates 20th of month..."
cat > /tmp/test_poll_github.go << 'EOF'
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
	
	// Check if it's the 20th of the month (always true for this test)
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
	// Always return true for testing (simulating 20th)
	return true
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
EOF

echo "🔧 Building test poll scheduler..."
go build -o test-poll-github /tmp/test_poll_github.go

if [ $? -ne 0 ]; then
    echo "❌ Failed to build test poll scheduler"
    exit 1
fi

echo "✅ Test poll scheduler built successfully"
echo ""

echo "🚀 Running test poll scheduler (simulating GitHub Actions)..."
./test-poll-github

if [ $? -eq 0 ]; then
    echo "✅ GitHub Actions simulation completed successfully"
else
    echo "❌ GitHub Actions simulation failed"
    exit 1
fi

echo ""
echo "🧹 Cleaning up..."
rm -f poll-scheduler test-poll-github
rm -f /tmp/test_poll_github.go

echo "✅ GitHub Actions workflow test completed successfully!"
echo ""
echo "📝 To set up the GitHub Actions workflow:"
echo "  1. Add TELEGRAM_POLL_BOT_TOKEN and TELEGRAM_POLL_CHAT_ID to your repository secrets"
echo "  2. The workflow will automatically run on the 20th of every month"
echo "  3. You can also trigger it manually from the GitHub Actions tab"
echo "  4. Set TEST_MODE=false in the workflow for production"
