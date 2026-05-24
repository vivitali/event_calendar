package annual

import (
	"context"
	"log"
	"sort"
	"sync"
	"time"
)

// SiteScraper fetches the next occurrence of one annual event from its
// canonical website.
type SiteScraper interface {
	// Name is a stable identifier used as the merge key against the baseline.
	Name() string
	// URL is the canonical event URL, surfaced in the digest footer.
	URL() string
	// Scrape fetches and parses the event's next-occurrence date.
	Scrape(ctx context.Context) (AnnualEvent, error)
}

// Scrape runs every scraper, merges results with the baseline from store
// (per-event last-known-good on error), drops past events, sorts by date,
// and writes back. Returns the number of events stored.
//
// Errors are intentionally swallowed inside the loop and logged; only a
// store read/write failure is returned. The weekly digest tolerates an
// empty SSM read, so a single failing scrape never breaks the user-facing
// path.
func Scrape(ctx context.Context, scrapers []SiteScraper, store Store, now time.Time) (int, error) {
	baseline, err := store.Get(ctx)
	if err != nil {
		log.Printf("annual: store.Get failed, starting from empty baseline: %v", err)
		baseline = nil
	}

	merged := make(map[string]AnnualEvent, len(scrapers)+len(baseline))
	for _, e := range baseline {
		merged[e.Name] = e
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, s := range scrapers {
		wg.Add(1)
		go func(s SiteScraper) {
			defer wg.Done()
			evt, err := s.Scrape(ctx)
			if err != nil {
				log.Printf("annual: scrape %s failed (retaining last-known): %v", s.Name(), err)
				return
			}
			mu.Lock()
			merged[s.Name()] = evt
			mu.Unlock()
		}(s)
	}
	wg.Wait()

	out := make([]AnnualEvent, 0, len(merged))
	for _, e := range merged {
		if e.Date.After(now) {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date.Before(out[j].Date) })

	if err := store.Put(ctx, out); err != nil {
		return 0, err
	}
	return len(out), nil
}
