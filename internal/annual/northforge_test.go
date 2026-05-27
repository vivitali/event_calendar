package annual

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestNorthForge_Scrape(t *testing.T) {
	html, err := os.ReadFile("testdata/northforge.html")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write(html)
	}))
	defer srv.Close()

	s := newNorthForgeWith(srv.URL)
	got, err := s.Scrape(context.Background())
	if err != nil {
		t.Fatalf("scrape: %v", err)
	}
	if got.Name != "North Forge RampUp Weekend" {
		t.Errorf("name = %q", got.Name)
	}
	if got.URL != "https://www.northforge.ca/rampup" {
		t.Errorf("url = %q", got.URL)
	}
	// Fixture shows "April 10-12, 2026"; we take the first day.
	want := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	if !got.Date.Equal(want) {
		t.Errorf("date = %v, want %v", got.Date, want)
	}
}
