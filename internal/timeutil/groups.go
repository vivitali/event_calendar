// Package timeutil groups events into time buckets relative to a reference time.
package timeutil

import "time"

// IsSameDay returns true when a and b fall on the same calendar day.
func IsSameDay(a, b time.Time) bool {
	return a.Year() == b.Year() && a.YearDay() == b.YearDay()
}

// IsThisWeek returns true when t falls in the same Sun..Sat week as now.
func IsThisWeek(t, now time.Time) bool {
	startOfWeek := startOfWeek(now)
	endOfWeek := startOfWeek.AddDate(0, 0, 7)
	return !t.Before(startOfWeek) && t.Before(endOfWeek)
}

// IsNextWeek returns true when t falls in the Sun..Sat week after now's week.
func IsNextWeek(t, now time.Time) bool {
	startOfNext := startOfWeek(now).AddDate(0, 0, 7)
	endOfNext := startOfNext.AddDate(0, 0, 7)
	return !t.Before(startOfNext) && t.Before(endOfNext)
}

// GroupByPeriod buckets times into "Today", "This Week", "Next Week", "Later"
// using now as the reference. Empty buckets are omitted from the result.
func GroupByPeriod(times []time.Time, now time.Time) map[string][]time.Time {
	groups := map[string][]time.Time{}
	for _, t := range times {
		switch {
		case IsSameDay(t, now):
			groups["Today"] = append(groups["Today"], t)
		case IsThisWeek(t, now):
			groups["This Week"] = append(groups["This Week"], t)
		case IsNextWeek(t, now):
			groups["Next Week"] = append(groups["Next Week"], t)
		default:
			groups["Later"] = append(groups["Later"], t)
		}
	}
	return groups
}

func startOfWeek(now time.Time) time.Time {
	y, m, d := now.Date()
	midnight := time.Date(y, m, d, 0, 0, 0, 0, now.Location())
	return midnight.AddDate(0, 0, -int(midnight.Weekday()))
}
