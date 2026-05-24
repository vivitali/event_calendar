package annual

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const prairieDevConURL = "https://www.prairiedevcon.com/"

// PrairieDevCon scrapes the next conference date from prairiedevcon.com.
type PrairieDevCon struct {
	client   *http.Client
	fetchURL string
}

// NewPrairieDevCon returns a scraper using the canonical URL.
func NewPrairieDevCon() *PrairieDevCon {
	return newPrairieDevConWith(prairieDevConURL)
}

func newPrairieDevConWith(fetchURL string) *PrairieDevCon {
	return &PrairieDevCon{
		client:   &http.Client{Timeout: 30 * time.Second},
		fetchURL: fetchURL,
	}
}

func (p *PrairieDevCon) Name() string { return "Prairie Dev Con Winnipeg" }
func (p *PrairieDevCon) URL() string  { return prairieDevConURL }

func (p *PrairieDevCon) Scrape(ctx context.Context) (AnnualEvent, error) {
	html, err := fetch(ctx, p.client, p.fetchURL)
	if err != nil {
		return AnnualEvent{}, err
	}
	d, err := parsePrairieDevConDate(html)
	if err != nil {
		return AnnualEvent{}, err
	}
	return AnnualEvent{Name: p.Name(), URL: prairieDevConURL, Date: d}, nil
}

// parsePrairieDevConDate looks for "Month D[-D2][,] YYYY" in the homepage
// text. Example matches:
//
//	"September 21-22 2026"
//	"September 21, 2026"
//	"Sept 21-22 2026"
var pdcDateRe = regexp.MustCompile(
	`(?i)(January|February|March|April|May|June|July|August|September|October|November|December|Sept?|Jan|Feb|Mar|Apr|Jun|Jul|Aug|Oct|Nov|Dec)\s+(\d{1,2})(?:\s*[-–]\s*\d{1,2})?[,\s]+\s*(20\d{2})`,
)

func parsePrairieDevConDate(html string) (time.Time, error) {
	m := pdcDateRe.FindStringSubmatch(html)
	if m == nil {
		return time.Time{}, fmt.Errorf("prairiedevcon: no date found")
	}
	monthStr, dayStr, yearStr := m[1], m[2], m[3]
	month, ok := monthFromString(monthStr)
	if !ok {
		return time.Time{}, fmt.Errorf("prairiedevcon: unknown month %q", monthStr)
	}
	var day, year int
	fmt.Sscanf(dayStr, "%d", &day)
	fmt.Sscanf(yearStr, "%d", &year)
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC), nil
}

// monthFromString maps full and short month names (case-insensitive) to time.Month.
func monthFromString(s string) (time.Month, bool) {
	switch strings.ToLower(s) {
	case "january", "jan":
		return time.January, true
	case "february", "feb":
		return time.February, true
	case "march", "mar":
		return time.March, true
	case "april", "apr":
		return time.April, true
	case "may":
		return time.May, true
	case "june", "jun":
		return time.June, true
	case "july", "jul":
		return time.July, true
	case "august", "aug":
		return time.August, true
	case "september", "sept", "sep":
		return time.September, true
	case "october", "oct":
		return time.October, true
	case "november", "nov":
		return time.November, true
	case "december", "dec":
		return time.December, true
	}
	return 0, false
}

// fetch is a shared GET-and-read used by all site scrapers in this package.
func fetch(ctx context.Context, c *http.Client, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; WinnipegTechEventsBot/1.0)")
	resp, err := c.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
