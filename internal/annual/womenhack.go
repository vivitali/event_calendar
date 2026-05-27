package annual

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

const womenHackURL = "https://womenhack.com/events/"

// WomenHack scrapes the next Winnipeg WomenHack event from the global events
// listing, filtering to the Winnipeg city card.
type WomenHack struct {
	client   *http.Client
	fetchURL string
}

func NewWomenHack() *WomenHack { return newWomenHackWith(womenHackURL) }

func newWomenHackWith(fetchURL string) *WomenHack {
	return &WomenHack{
		client:   &http.Client{Timeout: 30 * time.Second},
		fetchURL: fetchURL,
	}
}

func (w *WomenHack) Name() string { return "WomenHack Winnipeg" }
func (w *WomenHack) URL() string  { return womenHackURL }

func (w *WomenHack) Scrape(ctx context.Context) (AnnualEvent, error) {
	html, err := fetch(ctx, w.client, w.fetchURL)
	if err != nil {
		return AnnualEvent{}, err
	}
	d, err := parseWomenHackDate(html)
	if err != nil {
		return AnnualEvent{}, err
	}
	return AnnualEvent{Name: w.Name(), URL: womenHackURL, Date: d}, nil
}

// whYearRe pulls the year out of an Eventbrite RSVP link, the only place the
// listing carries it (e.g. ".../womenhack-winnipeg-employer-ticket-june-11-2026-tickets-...").
var whYearRe = regexp.MustCompile(`-(20\d{2})-tickets`)

// parseWomenHackDate reads the first Winnipeg event card. The date badge gives
// month + day; the year comes from the card's Eventbrite link. Cards are
// rendered soonest-first, so the first Winnipeg card is the next occurrence.
func parseWomenHackDate(html string) (time.Time, error) {
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(html))
	if err != nil {
		return time.Time{}, fmt.Errorf("womenhack: parse html: %w", err)
	}
	card := doc.Find(`.we-event-card[data-city="winnipeg"]`).First()
	if card.Length() == 0 {
		return time.Time{}, fmt.Errorf("womenhack: no Winnipeg event card found")
	}

	monthStr := strings.TrimSpace(card.Find(".we-event-date-badge .month").First().Text())
	month, ok := monthFromString(monthStr)
	if !ok {
		return time.Time{}, fmt.Errorf("womenhack: unknown month %q", monthStr)
	}

	dayStr := strings.TrimSpace(card.Find(".we-event-date-badge .day").First().Text())
	day, err := strconv.Atoi(dayStr)
	if err != nil {
		return time.Time{}, fmt.Errorf("womenhack: bad day %q: %w", dayStr, err)
	}

	year := 0
	card.Find("a[href]").EachWithBreak(func(_ int, s *goquery.Selection) bool {
		href, _ := s.Attr("href")
		if m := whYearRe.FindStringSubmatch(href); m != nil {
			year, _ = strconv.Atoi(m[1])
			return false
		}
		return true
	})
	if year == 0 {
		return time.Time{}, fmt.Errorf("womenhack: could not determine year")
	}

	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC), nil
}
