package luma

import (
	"os"
	"strings"
	"testing"
)

func TestParseICS_ExtractsEvents(t *testing.T) {
	data, err := os.ReadFile("testdata/winnipeg_tech_thursday.ics")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}

	events, err := parseICS(data)
	if err != nil {
		t.Fatalf("parseICS: %v", err)
	}
	if len(events) < 10 {
		t.Fatalf("want >= 10 events, got %d", len(events))
	}

	for i, e := range events {
		if e.Name == "" {
			t.Errorf("event %d: empty Name", i)
		}
		if e.Source != "luma" {
			t.Errorf("event %d: Source = %q, want luma", i, e.Source)
		}
		if e.StartTime.IsZero() {
			t.Errorf("event %d %q: StartTime zero", i, e.Name)
		}
		if e.URL == "" {
			t.Errorf("event %d %q: empty URL", i, e.Name)
		}
		// description should not bleed into name
		if strings.Contains(e.Name, "Address:") || strings.Contains(e.Name, "Hosted by") {
			t.Errorf("event %d name leaked description: %q", i, e.Name)
		}
	}
}

func TestParseICS_NoEvents(t *testing.T) {
	if _, err := parseICS([]byte("BEGIN:VCALENDAR\nEND:VCALENDAR\n")); err == nil {
		t.Errorf("want error when no events, got nil")
	}
}

func TestUnfoldICS_JoinsContinuationLines(t *testing.T) {
	// RFC 5545: CRLF followed by one whitespace char is removed entirely.
	in := "SUMMARY:Long \n title\nDESCRIPTION:Hello\n\tworld\n"
	got := unfoldICS([]byte(in))
	want := "SUMMARY:Long title\nDESCRIPTION:Helloworld\n"
	if string(got) != want {
		t.Errorf("unfoldICS = %q, want %q", got, want)
	}
}
