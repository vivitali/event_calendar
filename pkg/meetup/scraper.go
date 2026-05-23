package meetup

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"net/http"
	"regexp"
	"strings"
	"time"

	"event_calendar/internal/models"
)

type Scraper struct {
	client    *http.Client
	baseURL   string
	userAgent string
}

func NewScraper() *Scraper {
	return &Scraper{
		client:    &http.Client{Timeout: 30 * time.Second},
		baseURL:   "https://www.meetup.com/find/?location=ca--mb--winnipeg&source=EVENTS&categoryId=546",
		userAgent: "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0 Safari/537.36",
	}
}

func (s *Scraper) GetEvents(city, category string, period time.Duration) ([]models.Event, error) {
	html, err := s.fetchHTML(s.searchURL(city, category))
	if err != nil {
		log.Printf("[Meetup] fetch failed: %v", err)
		return nil, err
	}
	events, err := parseJSONLD(html)
	if err != nil {
		log.Printf("[Meetup] parse failed: %v", err)
		return nil, err
	}
	cutoff := time.Now().Add(period)
	out := events[:0]
	for _, e := range events {
		if !e.StartTime.IsZero() && e.StartTime.After(cutoff) {
			continue
		}
		if city != "" {
			e.City = city
		}
		if category != "" {
			e.Category = category
		}
		out = append(out, e)
	}
	log.Printf("[Meetup] parsed %d events, %d within %s", len(events), len(out), period)
	return out, nil
}

func (s *Scraper) searchURL(city, category string) string {
	if city == "" {
		city = "Winnipeg"
	}
	categoryMap := map[string]string{
		"tech":      "546",
		"business":  "2",
		"social":    "1",
		"arts":      "3",
		"health":    "4",
		"education": "5",
		"sports":    "6",
	}
	id := categoryMap[strings.ToLower(category)]
	if id == "" {
		id = "546"
	}
	return fmt.Sprintf("https://www.meetup.com/find/?location=ca--mb--%s&source=EVENTS&categoryId=%s",
		strings.ReplaceAll(strings.ToLower(city), " ", "-"), id)
}

func (s *Scraper) fetchHTML(url string) ([]byte, error) {
	req, err := http.NewRequest(http.MethodGet, url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("User-Agent", s.userAgent)
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	resp, err := s.client.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("meetup status %d", resp.StatusCode)
	}
	return io.ReadAll(resp.Body)
}

var jsonLDScriptRE = regexp.MustCompile(`(?s)<script type="application/ld\+json"[^>]*>(.*?)</script>`)

type ldEvent struct {
	Type        string          `json:"@type"`
	Name        string          `json:"name"`
	StartDate   string          `json:"startDate"`
	EndDate     string          `json:"endDate"`
	URL         string          `json:"url"`
	Description string          `json:"description"`
	Location    json.RawMessage `json:"location"`
	Organizer   json.RawMessage `json:"organizer"`
}

type ldNamed struct {
	Name string `json:"name"`
}

// firstName decodes a JSON-LD field that may be a single named object or an array of them.
func firstName(raw json.RawMessage) string {
	if len(raw) == 0 {
		return ""
	}
	var single ldNamed
	if err := json.Unmarshal(raw, &single); err == nil && single.Name != "" {
		return single.Name
	}
	var arr []ldNamed
	if err := json.Unmarshal(raw, &arr); err == nil {
		for _, n := range arr {
			if n.Name != "" {
				return n.Name
			}
		}
	}
	return ""
}

// parseJSONLD pulls all schema.org Event objects out of <script type="application/ld+json"> blocks.
func parseJSONLD(html []byte) ([]models.Event, error) {
	matches := jsonLDScriptRE.FindAllSubmatch(html, -1)
	if len(matches) == 0 {
		return nil, errors.New("no ld+json scripts found")
	}
	var out []models.Event
	for _, m := range matches {
		blob := m[1]
		var arr []ldEvent
		if err := json.Unmarshal(blob, &arr); err == nil {
			for _, e := range arr {
				if ev, ok := mapLDEvent(e); ok {
					out = append(out, ev)
				}
			}
			continue
		}
		var single ldEvent
		if err := json.Unmarshal(blob, &single); err == nil {
			if ev, ok := mapLDEvent(single); ok {
				out = append(out, ev)
			}
		}
	}
	if len(out) == 0 {
		return nil, errors.New("no Event objects in ld+json")
	}
	return out, nil
}

func mapLDEvent(e ldEvent) (models.Event, bool) {
	if e.Type != "Event" || e.Name == "" || e.URL == "" {
		return models.Event{}, false
	}
	return models.Event{
		ID:          eventID(e.URL),
		Name:        strings.TrimSpace(e.Name),
		Description: strings.TrimSpace(e.Description),
		URL:         cleanURL(e.URL),
		StartTime:   parseLDDate(e.StartDate),
		EndTime:     parseLDDate(e.EndDate),
		Source:      "meetup",
		Venue:       firstName(e.Location),
		Group:       firstName(e.Organizer),
	}, true
}

var eventIDRE = regexp.MustCompile(`/events/(\d+)/?`)

func eventID(url string) string {
	if m := eventIDRE.FindStringSubmatch(url); len(m) > 1 {
		return "meetup-" + m[1]
	}
	return "meetup-" + url
}

// cleanURL strips the noisy meetup recommendation/tracking query params.
func cleanURL(u string) string {
	if i := strings.Index(u, "?"); i > 0 {
		return u[:i]
	}
	return u
}

// parseLDDate parses ISO-8601 strings like "2026-06-03T22:00:00.000Z".
func parseLDDate(s string) time.Time {
	s = strings.TrimSpace(s)
	if s == "" {
		return time.Time{}
	}
	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02T15:04:05.000Z",
		"2006-01-02T15:04:05Z",
		"2006-01-02T15:04:05",
		"2006-01-02",
	} {
		if t, err := time.Parse(layout, s); err == nil {
			return t
		}
	}
	return time.Time{}
}
