package annual

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestWomenHack_Scrape(t *testing.T) {
	html, err := os.ReadFile("testdata/womenhack.html")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write(html)
	}))
	defer srv.Close()

	s := newWomenHackWith(srv.URL)
	got, err := s.Scrape(context.Background())
	if err != nil {
		t.Fatalf("scrape: %v", err)
	}
	if got.Name != "WomenHack Winnipeg" {
		t.Errorf("name = %q", got.Name)
	}
	if got.URL != "https://womenhack.com/events/" {
		t.Errorf("url = %q", got.URL)
	}
	// Fixture's first Winnipeg card: "Thu, Jun 11 · 7:00 PM", year from the
	// Eventbrite RSVP link (...june-11-2026-tickets...).
	want := time.Date(2026, 6, 11, 0, 0, 0, 0, time.UTC)
	if !got.Date.Equal(want) {
		t.Errorf("date = %v, want %v", got.Date, want)
	}
}
