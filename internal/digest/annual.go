package digest

import "time"

// AnnualEvent is a once-a-year Winnipeg tech event we want to highlight in
// the digest's expandable footer until it has passed.
type AnnualEvent struct {
	Name string
	Date time.Time
	URL  string
}

// annualEvents is the curated list of large, infrequent events we want to
// remind readers about. Edit this slice when dates are announced.
var annualEvents = []AnnualEvent{
	{
		Name: "Prairie Dev Con Winnipeg",
		Date: time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC),
		URL:  "https://www.prairiedevcon.com/",
	},
	{
		Name: "WomenHack Winnipeg",
		Date: time.Date(2026, 6, 11, 19, 0, 0, 0, time.UTC),
		URL:  "https://womenhack.com/cities/winnipeg/",
	},
	{
		Name: "North Forge RampUp Weekend",
		Date: time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC),
		URL:  "https://www.northforge.ca/rampup",
	},
	{
		Name: "MbTech Week",
		Date: time.Date(2027, 2, 1, 0, 0, 0, 0, time.UTC),
		URL:  "https://mbtechweek.ca/",
	},
}

// futureAnnualEvents returns annual events that have not yet started,
// in chronological order.
func futureAnnualEvents(now time.Time) []AnnualEvent {
	out := annualEvents[:0:0]
	for _, e := range annualEvents {
		if e.Date.After(now) {
			out = append(out, e)
		}
	}
	return out
}
