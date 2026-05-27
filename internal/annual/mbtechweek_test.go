package annual

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestMbTechWeek_Scrape(t *testing.T) {
	html, err := os.ReadFile("testdata/mbtechweek.html")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write(html)
	}))
	defer srv.Close()

	s := newMbTechWeekWith(srv.URL)
	got, err := s.Scrape(context.Background())
	if err != nil {
		t.Fatalf("scrape: %v", err)
	}
	if got.Name != "MbTech Week" {
		t.Errorf("name = %q", got.Name)
	}
	if got.URL != "https://mbtechweek.ca/" {
		t.Errorf("url = %q", got.URL)
	}
	// Fixture countdown points at the 2027 edition: data-countdown="2027-02-21 ...".
	want := time.Date(2027, 2, 21, 0, 0, 0, 0, time.UTC)
	if !got.Date.Equal(want) {
		t.Errorf("date = %v, want %v", got.Date, want)
	}
}
