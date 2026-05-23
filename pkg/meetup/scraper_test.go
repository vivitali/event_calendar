package meetup

import (
	"os"
	"strings"
	"testing"
)

func TestParseJSONLD_ExtractsCleanEvents(t *testing.T) {
	html, err := os.ReadFile("testdata/winnipeg_tech.html")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	events, err := parseJSONLD(html)
	if err != nil {
		t.Fatalf("parseJSONLD: %v", err)
	}

	if len(events) < 5 {
		t.Fatalf("want >= 5 events, got %d", len(events))
	}

	for i, e := range events {
		if e.Name == "" {
			t.Errorf("event %d: empty Name", i)
		}
		if !strings.HasPrefix(e.URL, "https://www.meetup.com/") {
			t.Errorf("event %d: URL %q does not start with meetup", i, e.URL)
		}
		if e.StartTime.IsZero() {
			t.Errorf("event %d %q: StartTime zero", i, e.Name)
		}
		if e.Source != "meetup" {
			t.Errorf("event %d: Source = %q", i, e.Source)
		}
		// Reject names containing concatenated card noise.
		lower := strings.ToLower(e.Name)
		if strings.Contains(lower, "attendee") || strings.Contains(lower, "attendees") {
			t.Errorf("event %d name contains 'attendee' (concatenated text): %q", i, e.Name)
		}
		if strings.Contains(lower, "monthly · ") || strings.Contains(lower, "weekly · ") {
			t.Errorf("event %d name contains date noise: %q", i, e.Name)
		}
	}
}

func TestParseJSONLD_NoEventsScript(t *testing.T) {
	if _, err := parseJSONLD([]byte("<html>no scripts</html>")); err == nil {
		t.Errorf("want error when no JSON-LD event script, got nil")
	}
}
