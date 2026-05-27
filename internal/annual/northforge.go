package annual

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"time"
)

const northForgeURL = "https://www.northforge.ca/rampup"

// NorthForge scrapes the next RampUp Weekend date.
type NorthForge struct {
	client   *http.Client
	fetchURL string
}

func NewNorthForge() *NorthForge { return newNorthForgeWith(northForgeURL) }

func newNorthForgeWith(fetchURL string) *NorthForge {
	return &NorthForge{
		client:   &http.Client{Timeout: 30 * time.Second},
		fetchURL: fetchURL,
	}
}

func (n *NorthForge) Name() string { return "North Forge RampUp Weekend" }
func (n *NorthForge) URL() string  { return northForgeURL }

func (n *NorthForge) Scrape(ctx context.Context) (AnnualEvent, error) {
	html, err := fetch(ctx, n.client, n.fetchURL)
	if err != nil {
		return AnnualEvent{}, err
	}
	d, err := parseNorthForgeDate(html)
	if err != nil {
		return AnnualEvent{}, err
	}
	return AnnualEvent{Name: n.Name(), URL: northForgeURL, Date: d}, nil
}

// parseNorthForgeDate matches "Month D[-D2], YYYY" with optional en/em-dash.
var nfDateRe = regexp.MustCompile(
	`(?i)(January|February|March|April|May|June|July|August|September|October|November|December)\s+(\d{1,2})(?:\s*[-–—]\s*\d{1,2})?,\s+(20\d{2})`,
)

func parseNorthForgeDate(html string) (time.Time, error) {
	m := nfDateRe.FindStringSubmatch(html)
	if m == nil {
		return time.Time{}, fmt.Errorf("northforge: no date found")
	}
	month, ok := monthFromString(m[1])
	if !ok {
		return time.Time{}, fmt.Errorf("northforge: unknown month %q", m[1])
	}
	var day, year int
	fmt.Sscanf(m[2], "%d", &day)
	fmt.Sscanf(m[3], "%d", &year)
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC), nil
}
