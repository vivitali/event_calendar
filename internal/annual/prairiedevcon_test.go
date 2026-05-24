package annual

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestPrairieDevCon_Scrape(t *testing.T) {
	html, err := os.ReadFile("testdata/prairiedevcon.html")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write(html)
	}))
	defer srv.Close()

	s := newPrairieDevConWith(srv.URL)
	got, err := s.Scrape(context.Background())
	if err != nil {
		t.Fatalf("scrape: %v", err)
	}
	if got.Name != "Prairie Dev Con Winnipeg" {
		t.Errorf("name = %q", got.Name)
	}
	if got.URL != "https://www.prairiedevcon.com/" {
		t.Errorf("url = %q", got.URL)
	}
	// Fixture contains "Sept 21-22 2026" — first date is September 21 2026.
	want := time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC)
	if !got.Date.Equal(want) {
		t.Errorf("date = %v, want %v", got.Date, want)
	}
}
