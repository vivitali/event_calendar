package scraping

import (
	"time"

	"event_calendar/internal/models"
	"event_calendar/pkg/luma"
)

// LumaScraper wraps pkg/luma.Scraper so it satisfies EventScraper.
type LumaScraper struct {
	*BaseScraper
	inner *luma.Scraper
}

func NewLumaScraper() *LumaScraper {
	return &LumaScraper{
		BaseScraper: NewBaseScraper("luma", "https://luma.com"),
		inner:       luma.NewScraper(),
	}
}

func (l *LumaScraper) GetEvents(city, category string, period time.Duration) ([]models.Event, error) {
	events, err := l.inner.GetEvents(city, category, period)
	l.LogScrapingResult(events, err)
	return events, err
}
