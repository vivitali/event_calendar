package scraping

import (
	"time"

	"event_calendar/internal/models"
	"event_calendar/pkg/meetup"
)

// MeetupScraper wraps pkg/meetup.Scraper so it satisfies EventScraper.
type MeetupScraper struct {
	*BaseScraper
	inner *meetup.Scraper
}

func NewMeetupScraper() *MeetupScraper {
	return &MeetupScraper{
		BaseScraper: NewBaseScraper("meetup", "https://www.meetup.com"),
		inner:       meetup.NewScraper(),
	}
}

func (m *MeetupScraper) GetEvents(city, category string, period time.Duration) ([]models.Event, error) {
	events, err := m.inner.GetEvents(city, category, period)
	m.LogScrapingResult(events, err)
	return events, err
}
