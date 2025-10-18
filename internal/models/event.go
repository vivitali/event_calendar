package models

import "time"

type Event struct {
	ID            string    `json:"id"`
	Name          string    `json:"name"`
	Description   string    `json:"description"`
	City          string    `json:"city"`
	Category      string    `json:"category"`
	URL           string    `json:"url"`
	StartTime     time.Time `json:"start_time"`
	EndTime       time.Time `json:"end_time"`
	Source        string    `json:"source"`
	Venue         string    `json:"venue,omitempty"`
	Group         string    `json:"group,omitempty"`
	AttendeeCount int       `json:"attendee_count,omitempty"`
	Price         string    `json:"price,omitempty"`
	DateString    string    `json:"date_string,omitempty"`
}
