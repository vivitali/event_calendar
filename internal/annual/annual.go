// Package annual provides scraping and storage of once-a-year Winnipeg tech
// events that we surface in the weekly digest's "save the date" footer.
package annual

import "time"

// AnnualEvent is one curated annual event with its next-occurrence date.
type AnnualEvent struct {
	Name string    `json:"name"`
	Date time.Time `json:"date"`
	URL  string    `json:"url"`
}
