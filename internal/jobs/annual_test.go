package jobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"event_calendar/internal/annual"
)

type fakeStore struct {
	events []annual.AnnualEvent
	putErr error
}

func (f *fakeStore) Get(ctx context.Context) ([]annual.AnnualEvent, error) {
	return f.events, nil
}
func (f *fakeStore) Put(ctx context.Context, e []annual.AnnualEvent) error {
	f.events = e
	return f.putErr
}

type fakeAnnualScraper struct {
	name string
	out  annual.AnnualEvent
	err  error
}

func (f *fakeAnnualScraper) Name() string { return f.name }
func (f *fakeAnnualScraper) URL() string  { return "https://x" }
func (f *fakeAnnualScraper) Scrape(ctx context.Context) (annual.AnnualEvent, error) {
	return f.out, f.err
}

func TestRunAnnualScrape_Success(t *testing.T) {
	now := time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC)
	store := &fakeStore{}
	deps := AnnualScrapeDeps{
		Store: store,
		Scrapers: []annual.SiteScraper{
			&fakeAnnualScraper{name: "A", out: annual.AnnualEvent{Name: "A", Date: now.AddDate(0, 2, 0), URL: "https://a"}},
		},
		Now: now,
	}
	res := RunAnnualScrape(context.Background(), deps)
	if !res.Success {
		t.Errorf("expected success, got %+v", res)
	}
	if res.EventsCount != 1 {
		t.Errorf("EventsCount = %d, want 1", res.EventsCount)
	}
	if len(store.events) != 1 {
		t.Errorf("store should have 1 event, has %d", len(store.events))
	}
}

func TestRunAnnualScrape_StoreError(t *testing.T) {
	now := time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC)
	store := &fakeStore{putErr: errors.New("ssm down")}
	deps := AnnualScrapeDeps{
		Store: store,
		Scrapers: []annual.SiteScraper{
			&fakeAnnualScraper{name: "A", out: annual.AnnualEvent{Name: "A", Date: now.AddDate(0, 2, 0), URL: "https://a"}},
		},
		Now: now,
	}
	res := RunAnnualScrape(context.Background(), deps)
	if res.Success {
		t.Errorf("expected failure on store error, got %+v", res)
	}
	if res.Error == "" {
		t.Errorf("expected non-empty Error message")
	}
}
