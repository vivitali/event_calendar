// Package digest formats event lists into Telegram messages.
package digest

import (
	"fmt"
	"strings"
	"time"

	"event_calendar/internal/models"
	"event_calendar/internal/timeutil"
)

const telegramMaxLen = 4096

var periodOrder = []string{"Today", "This Week", "Next Week", "Later"}

// FormatEventsMessage renders a Markdown-formatted Telegram digest of the events,
// grouped by time bucket relative to now. The result is guaranteed to be <= 4096 chars.
func FormatEventsMessage(events []models.Event, now time.Time) string {
	if len(events) == 0 {
		return "📅 *No upcoming events found* for Winnipeg tech community."
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "🚀 *Winnipeg Tech Events - %s*\n\n", now.Format("Monday, January 2, 2006"))

	groups := groupEvents(events, now)
	for _, period := range periodOrder {
		bucket := groups[period]
		if len(bucket) == 0 {
			continue
		}
		fmt.Fprintf(&sb, "*%s:*\n", period)
		for _, e := range bucket {
			writeEvent(&sb, e)
		}
	}
	sb.WriteString("\n_Shared via Winnipeg Tech Events Tracker_")

	out := sb.String()
	if len(out) > telegramMaxLen {
		out = out[:telegramMaxLen-3] + "..."
	}
	return out
}

func groupEvents(events []models.Event, now time.Time) map[string][]models.Event {
	groups := map[string][]models.Event{}
	for _, e := range events {
		switch {
		case timeutil.IsSameDay(e.StartTime, now):
			groups["Today"] = append(groups["Today"], e)
		case timeutil.IsThisWeek(e.StartTime, now):
			groups["This Week"] = append(groups["This Week"], e)
		case timeutil.IsNextWeek(e.StartTime, now):
			groups["Next Week"] = append(groups["Next Week"], e)
		default:
			groups["Later"] = append(groups["Later"], e)
		}
	}
	return groups
}

func writeEvent(sb *strings.Builder, e models.Event) {
	fmt.Fprintf(sb, "• %s %s\n", e.Name, sourceLabel(e.Source))
	if !e.StartTime.IsZero() {
		fmt.Fprintf(sb, "  📅 %s\n", e.StartTime.Format("Monday, Jan 2"))
	}
	if e.Venue != "" {
		fmt.Fprintf(sb, "  📍 %s\n", e.Venue)
	}
	if e.Price != "" && e.Price != "Free" {
		fmt.Fprintf(sb, "  💰 %s\n", e.Price)
	}
	if e.URL != "" {
		fmt.Fprintf(sb, "  🔗 [View Event](%s)\n", e.URL)
	}
	sb.WriteString("\n")
}

func sourceLabel(source string) string {
	switch source {
	case "meetup":
		return "`[Meetup]`"
	case "eventbrite":
		return "`[Eventbrite]`"
	default:
		return "`[" + source + "]`"
	}
}
