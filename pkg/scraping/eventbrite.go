package scraping

import (
	"time"

	"event_calendar/internal/models"
	"event_calendar/pkg/eventbrite"
)

// EventbriteScraper wraps pkg/eventbrite.Scraper so it satisfies EventScraper.
type EventbriteScraper struct {
	*BaseScraper
	inner *eventbrite.Scraper
}

func NewEventbriteScraper() *EventbriteScraper {
	return &EventbriteScraper{
		BaseScraper: NewBaseScraper("eventbrite", "https://www.eventbrite.ca"),
		inner:       eventbrite.NewScraper(),
	}
}

func (e *EventbriteScraper) GetEvents(city, category string, period time.Duration) ([]models.Event, error) {
	events, err := e.inner.GetEvents(city, category, period)
	e.LogScrapingResult(events, err)
	return events, err
}
