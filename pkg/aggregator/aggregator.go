// Package aggregator combines multiple event providers and filters/dedupes results.
package aggregator

import (
	"log"
	"sort"
	"time"

	"event_calendar/internal/models"
)

// EventProvider produces events from a single source.
type EventProvider interface {
	GetEvents(city, category string, period time.Duration) ([]models.Event, error)
}

// Aggregator fans out to multiple providers, merges results, and removes duplicates.
type Aggregator struct {
	providers []EventProvider
}

// NewAggregator constructs an aggregator over the supplied providers.
func NewAggregator(providers ...EventProvider) *Aggregator {
	return &Aggregator{providers: providers}
}

// AggregateEvents calls every provider, merges their results, sorts by start time,
// and removes duplicates. Per-provider errors are logged and skipped.
func (a *Aggregator) AggregateEvents(city, category string, period time.Duration) ([]models.Event, error) {
	var out []models.Event
	for _, p := range a.providers {
		events, err := p.GetEvents(city, category, period)
		if err != nil {
			log.Printf("provider error: %v", err)
			continue
		}
		out = append(out, events...)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartTime.Before(out[j].StartTime) })
	return removeDuplicates(out), nil
}

func removeDuplicates(events []models.Event) []models.Event {
	seen := make(map[string]bool)
	var unique []models.Event
	for _, e := range events {
		key := e.URL + "|" + e.Name
		if seen[key] {
			continue
		}
		seen[key] = true
		unique = append(unique, e)
	}
	return unique
}

// FilterFutureEvents returns only events whose StartTime is after now.
func FilterFutureEvents(events []models.Event, now time.Time) []models.Event {
	var future []models.Event
	for _, e := range events {
		if e.StartTime.After(now) {
			future = append(future, e)
		}
	}
	return future
}
