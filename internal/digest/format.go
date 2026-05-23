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
	fmt.Fprintf(&sb, "🚀 *Winnipeg Tech Events* — %s · %d upcoming\n\n",
		now.Format("Mon Jan 2"), len(events))

	groups := groupEvents(events, now)
	for _, period := range periodOrder {
		bucket := groups[period]
		if len(bucket) == 0 {
			continue
		}
		fmt.Fprintf(&sb, "*%s*\n", bucketHeader(period, now))
		for _, e := range bucket {
			writeEvent(&sb, e)
		}
		sb.WriteString("\n")
	}

	sb.WriteString("_Shared via Winnipeg Tech Events Tracker_\n")
	sb.WriteString("#WinnipegTech #TechEvents")

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

// bucketHeader returns a Markdown header label with a date range so readers
// know at a glance what window each bucket covers. The "Later" bucket has no
// fixed range so the label stays bare.
func bucketHeader(period string, now time.Time) string {
	y, m, d := now.Date()
	today := time.Date(y, m, d, 0, 0, 0, 0, now.Location())
	startOfWeek := today.AddDate(0, 0, -int(today.Weekday()))
	endOfWeek := startOfWeek.AddDate(0, 0, 6)
	startOfNext := startOfWeek.AddDate(0, 0, 7)
	endOfNext := startOfNext.AddDate(0, 0, 6)

	switch period {
	case "Today":
		return fmt.Sprintf("Today (%s)", today.Format("Mon, Jan 2"))
	case "This Week":
		return fmt.Sprintf("This Week (%s – %s)",
			startOfWeek.Format("Jan 2"), endOfWeek.Format("Jan 2"))
	case "Next Week":
		return fmt.Sprintf("Next Week (%s – %s)",
			startOfNext.Format("Jan 2"), endOfNext.Format("Jan 2"))
	default:
		return period
	}
}

func writeEvent(sb *strings.Builder, e models.Event) {
	name := escapeMarkdown(strings.TrimSpace(e.Name))
	if e.URL != "" {
		fmt.Fprintf(sb, "• %s [%s](%s)\n", name, sourceName(e.Source), e.URL)
	} else {
		fmt.Fprintf(sb, "• %s %s\n", name, sourceLabel(e.Source))
	}

	var meta []string
	if !e.StartTime.IsZero() {
		meta = append(meta, "`"+e.StartTime.Format("Mon Jan 2")+"`")
	}
	if v := cleanVenue(e.Venue); v != "" {
		meta = append(meta, "`"+v+"`")
	}
	if e.Price != "" && e.Price != "Free" {
		meta = append(meta, "`"+e.Price+"`")
	}
	if len(meta) > 0 {
		fmt.Fprintf(sb, "  %s\n", strings.Join(meta, " · "))
	}
}

// cleanVenue drops placeholder strings that add no information.
func cleanVenue(v string) string {
	v = strings.TrimSpace(v)
	switch strings.ToLower(v) {
	case "", "event", "online", "tbd", "tba", "none":
		return ""
	}
	return v
}

// escapeMarkdown escapes the characters that would break Telegram MarkdownV1
// link text. Brackets in link text break the parser; underscores/asterisks
// inside titles toggle formatting unintentionally.
func escapeMarkdown(s string) string {
	r := strings.NewReplacer(
		"[", "(",
		"]", ")",
		"*", "·",
		"_", " ",
	)
	return r.Replace(s)
}

func sourceLabel(source string) string {
	return "`[" + sourceName(source) + "]`"
}

func sourceName(source string) string {
	switch source {
	case "meetup":
		return "Meetup"
	case "eventbrite":
		return "Eventbrite"
	default:
		return source
	}
}
