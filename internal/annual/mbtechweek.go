package annual

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"strconv"
	"time"
)

const mbTechWeekURL = "https://mbtechweek.ca/"

// MbTechWeek scrapes the next MbTech Week date from mbtechweek.ca.
type MbTechWeek struct {
	client   *http.Client
	fetchURL string
}

func NewMbTechWeek() *MbTechWeek { return newMbTechWeekWith(mbTechWeekURL) }

func newMbTechWeekWith(fetchURL string) *MbTechWeek {
	return &MbTechWeek{
		client:   &http.Client{Timeout: 30 * time.Second},
		fetchURL: fetchURL,
	}
}

func (m *MbTechWeek) Name() string { return "MbTech Week" }
func (m *MbTechWeek) URL() string  { return mbTechWeekURL }

func (m *MbTechWeek) Scrape(ctx context.Context) (AnnualEvent, error) {
	html, err := fetch(ctx, m.client, m.fetchURL)
	if err != nil {
		return AnnualEvent{}, err
	}
	d, err := parseMbTechWeekDate(html)
	if err != nil {
		return AnnualEvent{}, err
	}
	return AnnualEvent{Name: m.Name(), URL: mbTechWeekURL, Date: d}, nil
}

// mtwDateRe reads the homepage countdown attribute, which carries the next
// event's start date in ISO form (e.g. data-countdown="2027-02-21 09:00:24").
// The prose date ("February 21 – Saturday 27, 2027") is too irregular to parse.
var mtwDateRe = regexp.MustCompile(`data-countdown="(20\d{2})-(\d{2})-(\d{2})`)

func parseMbTechWeekDate(html string) (time.Time, error) {
	m := mtwDateRe.FindStringSubmatch(html)
	if m == nil {
		return time.Time{}, fmt.Errorf("mbtechweek: no countdown date found")
	}
	year, _ := strconv.Atoi(m[1])
	monthN, _ := strconv.Atoi(m[2])
	day, _ := strconv.Atoi(m[3])
	return time.Date(year, time.Month(monthN), day, 0, 0, 0, 0, time.UTC), nil
}
