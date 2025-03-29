package aggregator

import (
	"event_calendar/internal/models"
	"time"
)

type EventProvider interface {
	GetEvents(city, category string, period time.Duration) ([]models.Event, error)
}

type Aggregator struct {
	providers []EventProvider
}

func NewAggregator(providers ...EventProvider) *Aggregator {
	return &Aggregator{providers: providers}
}

func (a *Aggregator) AggregateEvents(city, category string, period time.Duration) ([]models.Event, error) {
	var aggregated []models.Event
	for _, provider := range a.providers {
		events, err := provider.GetEvents(city, category, period)
		if err != nil {
			return nil, err
		}
		aggregated = append(aggregated, events...)
	}
	return aggregated, nil
}
