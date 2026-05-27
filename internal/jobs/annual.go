package jobs

import (
	"context"
	"fmt"
	"time"

	"event_calendar/internal/annual"
)

// AnnualScrapeDeps holds collaborators for RunAnnualScrape.
type AnnualScrapeDeps struct {
	Store    annual.Store
	Scrapers []annual.SiteScraper
	Now      time.Time
}

// RunAnnualScrape executes the monthly annual-events refresh: scrape each
// site, merge with last-known SSM state, persist back. Per-site failures
// are logged inside the orchestrator and do not fail the job; only a store
// read/write failure produces a non-success Result.
func RunAnnualScrape(ctx context.Context, deps AnnualScrapeDeps) Result {
	n, err := annual.Scrape(ctx, deps.Scrapers, deps.Store, deps.Now)
	if err != nil {
		return Result{Error: fmt.Sprintf("annual scrape failed: %v", err)}
	}
	return Result{Success: true, EventsCount: n}
}
