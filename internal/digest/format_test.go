package digest

import (
	"strings"
	"testing"
	"time"

	"event_calendar/internal/models"
)

func TestFormatEventsMessage_Empty(t *testing.T) {
	now := time.Date(2026, 1, 7, 12, 0, 0, 0, time.UTC)
	got := FormatEventsMessage(nil, now)
	if !strings.Contains(got, "No upcoming events") {
		t.Errorf("expected empty-state message, got: %s", got)
	}
}

func TestFormatEventsMessage_GroupsAndHeaders(t *testing.T) {
	now := time.Date(2026, 1, 7, 12, 0, 0, 0, time.UTC) // Wed
	events := []models.Event{
		{Name: "Today Event", URL: "https://e/1", Source: "meetup", StartTime: now.Add(3 * time.Hour), Venue: "Hub"},
		{Name: "Friday Event", URL: "https://e/2", Source: "eventbrite", StartTime: time.Date(2026, 1, 9, 18, 0, 0, 0, time.UTC)},
		{Name: "Next Week Event", URL: "https://e/3", Source: "devevents", StartTime: time.Date(2026, 1, 14, 18, 0, 0, 0, time.UTC)},
	}
	msg := FormatEventsMessage(events, now)
	for _, want := range []string{
		"Winnipeg Tech Events",
		"*Today:*",
		"*This Week:*",
		"*Next Week:*",
		"Today Event",
		"Friday Event",
		"Next Week Event",
		"[Meetup]",
		"[Eventbrite]",
		"[Dev.events]",
		"https://e/1",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("message missing %q.\n--- message ---\n%s", want, msg)
		}
	}
}

func TestFormatEventsMessage_TruncatesLong(t *testing.T) {
	now := time.Date(2026, 1, 7, 12, 0, 0, 0, time.UTC)
	var events []models.Event
	for i := 0; i < 200; i++ {
		events = append(events, models.Event{
			Name:      "Event with a fairly long name that pushes content",
			URL:       "https://example.com/very/long/url/path/segment",
			Source:    "meetup",
			StartTime: now.Add(time.Duration(i) * time.Hour),
		})
	}
	msg := FormatEventsMessage(events, now)
	if len(msg) > 4096 {
		t.Errorf("message exceeded Telegram limit: %d chars", len(msg))
	}
}
