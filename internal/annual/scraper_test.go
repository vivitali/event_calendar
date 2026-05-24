package annual

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeScraper struct {
	name string
	url  string
	out  AnnualEvent
	err  error
}

func (f *fakeScraper) Name() string { return f.name }
func (f *fakeScraper) URL() string  { return f.url }
func (f *fakeScraper) Scrape(ctx context.Context) (AnnualEvent, error) {
	return f.out, f.err
}

type memStore struct {
	data []AnnualEvent
	get  func() ([]AnnualEvent, error)
}

func (m *memStore) Get(ctx context.Context) ([]AnnualEvent, error) {
	if m.get != nil {
		return m.get()
	}
	return m.data, nil
}
func (m *memStore) Put(ctx context.Context, events []AnnualEvent) error {
	m.data = events
	return nil
}

func TestScrape_WritesFreshResults(t *testing.T) {
	now := time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC)
	store := &memStore{}
	scrapers := []SiteScraper{
		&fakeScraper{name: "A", url: "https://a", out: AnnualEvent{Name: "A", Date: now.AddDate(0, 1, 0), URL: "https://a"}},
		&fakeScraper{name: "B", url: "https://b", out: AnnualEvent{Name: "B", Date: now.AddDate(0, 2, 0), URL: "https://b"}},
	}
	n, err := Scrape(context.Background(), scrapers, store, now)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if n != 2 {
		t.Errorf("want 2 events, got %d", n)
	}
	if len(store.data) != 2 {
		t.Fatalf("store should have 2, has %d", len(store.data))
	}
	if !store.data[0].Date.Before(store.data[1].Date) {
		t.Errorf("expected sorted by date asc, got %+v", store.data)
	}
}

func TestScrape_KeepsLastKnownOnError(t *testing.T) {
	now := time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC)
	old := AnnualEvent{Name: "A", Date: now.AddDate(0, 4, 0), URL: "https://a-old"}
	store := &memStore{data: []AnnualEvent{old}}
	scrapers := []SiteScraper{
		&fakeScraper{name: "A", url: "https://a", err: errors.New("boom")},
	}
	n, err := Scrape(context.Background(), scrapers, store, now)
	if err != nil {
		t.Fatalf("orchestrator should not return error, got %v", err)
	}
	if n != 1 {
		t.Errorf("want 1 event retained, got %d", n)
	}
	if len(store.data) != 1 || store.data[0].URL != "https://a-old" {
		t.Errorf("last-known not retained: %+v", store.data)
	}
}

func TestScrape_DropsPastEvents(t *testing.T) {
	now := time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC)
	store := &memStore{}
	scrapers := []SiteScraper{
		&fakeScraper{name: "A", url: "https://a", out: AnnualEvent{Name: "A", Date: now.AddDate(0, -1, 0), URL: "https://a"}},
		&fakeScraper{name: "B", url: "https://b", out: AnnualEvent{Name: "B", Date: now.AddDate(0, 1, 0), URL: "https://b"}},
	}
	n, _ := Scrape(context.Background(), scrapers, store, now)
	if n != 1 {
		t.Errorf("want 1 future event, got %d", n)
	}
	if len(store.data) != 1 || store.data[0].Name != "B" {
		t.Errorf("expected only B retained, got %+v", store.data)
	}
}

func TestScrape_EmptyBaselineNoFreshResults(t *testing.T) {
	now := time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC)
	store := &memStore{}
	scrapers := []SiteScraper{
		&fakeScraper{name: "A", url: "https://a", err: errors.New("boom")},
	}
	n, _ := Scrape(context.Background(), scrapers, store, now)
	if n != 0 {
		t.Errorf("want 0 events, got %d", n)
	}
	if len(store.data) != 0 {
		t.Errorf("store should remain empty, got %+v", store.data)
	}
}
