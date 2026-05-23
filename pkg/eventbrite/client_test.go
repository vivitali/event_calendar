package eventbrite

import (
	"os"
	"strings"
	"testing"
)

func TestParseServerData_ExtractsRealEvents(t *testing.T) {
	html, err := os.ReadFile("testdata/winnipeg_tech.html")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	events, err := parseServerData(html)
	if err != nil {
		t.Fatalf("parseServerData: %v", err)
	}

	if len(events) < 10 {
		t.Fatalf("want >= 10 events, got %d", len(events))
	}

	first := events[0]
	if first.Name == "" {
		t.Errorf("first event Name empty")
	}
	if !strings.HasPrefix(first.URL, "https://www.eventbrite.") {
		t.Errorf("first event URL not eventbrite: %q", first.URL)
	}
	if first.StartTime.IsZero() {
		t.Errorf("first event StartTime zero")
	}
	if first.Source != "eventbrite" {
		t.Errorf("first event Source = %q, want eventbrite", first.Source)
	}

	for _, e := range events {
		if strings.Contains(e.URL, "example") {
			t.Errorf("got placeholder sample URL, expected real scraped data: %q", e.URL)
		}
	}
}

func TestParseServerData_DedupesSpammyOrganizer(t *testing.T) {
	// Fixture contains 20 raw events but 11 of them are the same organizer
	// posting the same date with different keyword titles. Expect dedupe to one.
	html, err := os.ReadFile("testdata/winnipeg_tech.html")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	events, err := parseServerData(html)
	if err != nil {
		t.Fatalf("parseServerData: %v", err)
	}
	const wantMax = 11 // 20 raw - 10 dropped dupes + a little slack
	if len(events) > wantMax {
		t.Errorf("dedupe did not run: got %d events, want <= %d", len(events), wantMax)
	}
	// Should still keep at least 5 — the non-duplicated events plus 1 of the dup series.
	if len(events) < 5 {
		t.Errorf("dedupe too aggressive: got %d events, want >= 5", len(events))
	}
}

func TestParseServerData_NoMarker(t *testing.T) {
	if _, err := parseServerData([]byte("<html>no marker here</html>")); err == nil {
		t.Errorf("want error when SERVER_DATA marker missing, got nil")
	}
}
