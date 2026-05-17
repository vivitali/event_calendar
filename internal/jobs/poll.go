package jobs

import (
	"fmt"
	"log"

	"event_calendar/internal/config"
)

// Poller sends a pre-canned poll to a chat.
type Poller interface {
	SendMonthlyMeetupPoll(chatID string) error
}

// PollDeps holds the collaborators needed by RunMonthlyPoll.
type PollDeps struct {
	Poller Poller
}

// RunMonthlyPoll sends the monthly meetup poll. In TestMode the poll is
// skipped. Scheduling (i.e. "only on the 20th") is the responsibility of
// the caller (EventBridge in production).
func RunMonthlyPoll(deps PollDeps, cfg config.Config) Result {
	if cfg.PollBotToken == "" || cfg.PollChatID == "" {
		return Result{Error: "TELEGRAM_POLL_BOT_TOKEN or TELEGRAM_POLL_CHAT_ID not set"}
	}
	if cfg.TestMode {
		log.Printf("test mode: would have sent monthly poll to %s", cfg.PollChatID)
		return Result{Success: true, MessageSent: false}
	}
	if err := deps.Poller.SendMonthlyMeetupPoll(cfg.PollChatID); err != nil {
		return Result{Error: fmt.Sprintf("poll send failed: %v", err)}
	}
	return Result{Success: true, MessageSent: true}
}
