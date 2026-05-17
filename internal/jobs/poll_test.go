package jobs

import (
	"errors"
	"testing"

	"event_calendar/internal/config"
)

type fakePoller struct {
	chatID string
	called int
	err    error
}

func (f *fakePoller) SendMonthlyMeetupPoll(chatID string) error {
	f.called++
	f.chatID = chatID
	return f.err
}

func TestRunMonthlyPoll_Success(t *testing.T) {
	poller := &fakePoller{}
	cfg := config.Config{PollBotToken: "tok", PollChatID: "pollchat"}

	res := RunMonthlyPoll(PollDeps{Poller: poller}, cfg)
	if !res.Success || !res.MessageSent {
		t.Fatalf("expected success+sent: %+v", res)
	}
	if poller.chatID != "pollchat" || poller.called != 1 {
		t.Errorf("poller not called correctly: %+v", poller)
	}
}

func TestRunMonthlyPoll_TestMode(t *testing.T) {
	poller := &fakePoller{}
	cfg := config.Config{PollBotToken: "tok", PollChatID: "pollchat", TestMode: true}

	res := RunMonthlyPoll(PollDeps{Poller: poller}, cfg)
	if !res.Success || res.MessageSent {
		t.Errorf("test mode should succeed without sending: %+v", res)
	}
	if poller.called != 0 {
		t.Errorf("poller called in test mode")
	}
}

func TestRunMonthlyPoll_MissingCreds(t *testing.T) {
	res := RunMonthlyPoll(PollDeps{Poller: &fakePoller{}}, config.Config{})
	if res.Success {
		t.Errorf("expected failure when creds missing: %+v", res)
	}
}

func TestRunMonthlyPoll_SendError(t *testing.T) {
	poller := &fakePoller{err: errors.New("kaboom")}
	cfg := config.Config{PollBotToken: "tok", PollChatID: "pollchat"}
	res := RunMonthlyPoll(PollDeps{Poller: poller}, cfg)
	if res.Success {
		t.Errorf("expected failure on send error: %+v", res)
	}
}
