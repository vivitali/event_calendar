package aggregator

import (
	"event_calendar/internal/models"
	"log"
	"sort"
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
	var errors []error
	
	for _, provider := range a.providers {
		events, err := provider.GetEvents(city, category, period)
		if err != nil {
			log.Printf("Provider error: %v", err)
			errors = append(errors, err)
			continue // Continue with other providers
		}
		aggregated = append(aggregated, events...)
	}
	
	// Sort events by start time
	sort.Slice(aggregated, func(i, j int) bool {
		return aggregated[i].StartTime.Before(aggregated[j].StartTime)
	})
	
	// Remove duplicates based on URL and name
	aggregated = removeDuplicates(aggregated)
	
	// Log results
	log.Printf("Aggregated %d events from %d providers", len(aggregated), len(a.providers))
	if len(errors) > 0 {
		log.Printf("Encountered %d provider errors", len(errors))
	}
	
	return aggregated, nil
}

// removeDuplicates removes duplicate events based on URL and name similarity
func removeDuplicates(events []models.Event) []models.Event {
	seen := make(map[string]bool)
	var unique []models.Event
	
	for _, event := range events {
		key := event.URL + "|" + event.Name
		if !seen[key] {
			seen[key] = true
			unique = append(unique, event)
		}
	}
	
	return unique
}

// FilterFutureEvents filters out past events
func FilterFutureEvents(events []models.Event) []models.Event {
	now := time.Now()
	var future []models.Event
	
	for _, event := range events {
		if event.StartTime.After(now) {
			future = append(future, event)
		}
	}
	
	return future
}

// GroupEventsByTime groups events by time periods
func GroupEventsByTime(events []models.Event) map[string][]models.Event {
	now := time.Now()
	groups := map[string][]models.Event{
		"Today":     {},
		"This Week": {},
		"Next Week": {},
		"Later":     {},
	}
	
	for _, event := range events {
		if isSameDay(event.StartTime, now) {
			groups["Today"] = append(groups["Today"], event)
		} else if isThisWeek(event.StartTime) {
			groups["This Week"] = append(groups["This Week"], event)
		} else if isNextWeek(event.StartTime) {
			groups["Next Week"] = append(groups["Next Week"], event)
		} else {
			groups["Later"] = append(groups["Later"], event)
		}
	}
	
	// Remove empty groups
	for key, group := range groups {
		if len(group) == 0 {
			delete(groups, key)
		}
	}
	
	return groups
}

func isSameDay(date1, date2 time.Time) bool {
	return date1.Year() == date2.Year() && 
		   date1.YearDay() == date2.YearDay()
}

func isThisWeek(date time.Time) bool {
	now := time.Now()
	startOfWeek := now.AddDate(0, 0, -int(now.Weekday()))
	endOfWeek := startOfWeek.AddDate(0, 0, 6)
	return date.After(startOfWeek) && date.Before(endOfWeek.Add(24*time.Hour))
}

func isNextWeek(date time.Time) bool {
	now := time.Now()
	startOfNextWeek := now.AddDate(0, 0, 7-int(now.Weekday()))
	endOfNextWeek := startOfNextWeek.AddDate(0, 0, 6)
	return date.After(startOfNextWeek) && date.Before(endOfNextWeek.Add(24*time.Hour))
}
