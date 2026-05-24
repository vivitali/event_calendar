package jobs

import (
	"errors"
	"testing"
	"time"

	"event_calendar/internal/config"
	"event_calendar/internal/models"
)

type fakeScraper struct {
	events []models.Event
	err    error
}

func (f *fakeScraper) ScrapeEvents(city, category string, period time.Duration) ([]models.Event, error) {
	return f.events, f.err
}

type fakeSender struct {
	chatID  string
	message string
	err     error
	calls   int
}

func (f *fakeSender) SendMessage(chatID, message string) error {
	f.calls++
	f.chatID = chatID
	f.message = message
	return f.err
}

func TestRunEventsDigest_Success(t *testing.T) {
	now := time.Date(2026, 1, 7, 12, 0, 0, 0, time.UTC)
	scraper := &fakeScraper{events: []models.Event{
		{Name: "Future", StartTime: now.Add(48 * time.Hour), Source: "meetup", URL: "https://e/1"},
		{Name: "Past", StartTime: now.Add(-48 * time.Hour), Source: "meetup", URL: "https://e/2"},
	}}
	sender := &fakeSender{}
	cfg := config.Config{BotToken: "tok", ChatID: "chat", City: "Winnipeg", Categories: "tech", PeriodDays: 30}

	res := RunEventsDigest(EventsDeps{Scraper: scraper, Sender: sender, Now: now}, cfg, nil)
	if !res.Success {
		t.Fatalf("expected success, got %+v", res)
	}
	if res.EventsCount != 1 {
		t.Errorf("expected 1 future event, got %d", res.EventsCount)
	}
	if !res.MessageSent {
		t.Errorf("expected message sent")
	}
	if sender.calls != 1 || sender.chatID != "chat" {
		t.Errorf("sender not called correctly: %+v", sender)
	}
}

func TestRunEventsDigest_NoEvents(t *testing.T) {
	now := time.Date(2026, 1, 7, 12, 0, 0, 0, time.UTC)
	scraper := &fakeScraper{events: nil}
	sender := &fakeSender{}
	cfg := config.Config{BotToken: "tok", ChatID: "chat"}

	res := RunEventsDigest(EventsDeps{Scraper: scraper, Sender: sender, Now: now}, cfg, nil)
	if !res.Success {
		t.Fatalf("expected success even with zero events, got %+v", res)
	}
	if res.MessageSent {
		t.Errorf("should not send when no events")
	}
	if sender.calls != 0 {
		t.Errorf("sender should not be called")
	}
}

func TestRunEventsDigest_TestMode(t *testing.T) {
	now := time.Date(2026, 1, 7, 12, 0, 0, 0, time.UTC)
	scraper := &fakeScraper{events: []models.Event{
		{Name: "Future", StartTime: now.Add(24 * time.Hour), Source: "meetup"},
	}}
	sender := &fakeSender{}
	cfg := config.Config{BotToken: "tok", ChatID: "chat", TestMode: true}

	res := RunEventsDigest(EventsDeps{Scraper: scraper, Sender: sender, Now: now}, cfg, nil)
	if !res.Success || res.MessageSent {
		t.Errorf("test mode should succeed without sending: %+v", res)
	}
	if sender.calls != 0 {
		t.Errorf("sender called in test mode")
	}
}

func TestRunEventsDigest_SendError_PreservesCount(t *testing.T) {
	now := time.Date(2026, 1, 7, 12, 0, 0, 0, time.UTC)
	scraper := &fakeScraper{events: []models.Event{
		{Name: "F1", StartTime: now.Add(24 * time.Hour), Source: "meetup", URL: "https://e/1"},
		{Name: "F2", StartTime: now.Add(48 * time.Hour), Source: "meetup", URL: "https://e/2"},
	}}
	sender := &fakeSender{err: errors.New("chat not found")}
	cfg := config.Config{BotToken: "tok", ChatID: "chat"}

	res := RunEventsDigest(EventsDeps{Scraper: scraper, Sender: sender, Now: now}, cfg, nil)
	if res.Success {
		t.Errorf("expected failure on send error: %+v", res)
	}
	if res.EventsCount != 2 {
		t.Errorf("expected EventsCount=2 to be preserved on send error, got %d", res.EventsCount)
	}
	if res.MessageSent {
		t.Errorf("MessageSent should be false on send error")
	}
}

func TestRunEventsDigest_ScraperError(t *testing.T) {
	now := time.Date(2026, 1, 7, 12, 0, 0, 0, time.UTC)
	scraper := &fakeScraper{err: errors.New("network down")}
	sender := &fakeSender{}
	cfg := config.Config{BotToken: "tok", ChatID: "chat"}

	res := RunEventsDigest(EventsDeps{Scraper: scraper, Sender: sender, Now: now}, cfg, nil)
	if res.Success {
		t.Errorf("expected failure on scraper error: %+v", res)
	}
	if res.Error == "" {
		t.Errorf("expected error message")
	}
}
