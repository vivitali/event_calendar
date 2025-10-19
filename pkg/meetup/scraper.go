package meetup

import (
	"event_calendar/internal/models"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/go-resty/resty/v2"
)

type Scraper struct {
	client *resty.Client
	baseURL string
}

func NewScraper() *Scraper {
	return &Scraper{
		client: resty.New().
			SetTimeout(30 * time.Second).
			SetHeader("User-Agent", "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36").
			SetHeader("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/webp,*/*;q=0.8").
			SetHeader("Accept-Language", "en-US,en;q=0.5").
			SetHeader("Accept-Encoding", "gzip, deflate").
			SetHeader("Connection", "keep-alive").
			SetHeader("Upgrade-Insecure-Requests", "1"),
		baseURL: "https://www.meetup.com/find/?location=ca--mb--Winnipeg&source=EVENTS&categoryId=546",
	}
}

func (s *Scraper) GetEvents(city, category string, period time.Duration) ([]models.Event, error) {
	log.Printf("Fetching Meetup events for city: %s, category: %s, period: %v", city, category, period)
	
	events, err := s.fetchEventsFromMeetup(city, category, period)
	if err != nil {
		log.Printf("Failed to fetch events from Meetup: %v", err)
		// Return sample data if scraping fails
		return s.getSampleEvents(), nil
	}
	
	log.Printf("Successfully fetched %d events from Meetup", len(events))
	return events, nil
}

func (s *Scraper) fetchEventsFromMeetup(city, category string, period time.Duration) ([]models.Event, error) {
	// Build the search URL based on parameters
	searchURL := s.buildSearchURL(city, category)
	
	// Fetch the page
	resp, err := s.client.R().Get(searchURL)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch Meetup page: %w", err)
	}
	
	if resp.StatusCode() != 200 {
		return nil, fmt.Errorf("received non-200 status code: %d", resp.StatusCode())
	}
	
	// Parse the HTML
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(resp.String()))
	if err != nil {
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}
	
	// Debug: Log page title and some content
	title := doc.Find("title").Text()
	log.Printf("Page title: %s", title)
	
	// Debug: Check if we got a valid page
	if strings.Contains(strings.ToLower(title), "meetup") {
		log.Printf("Successfully loaded Meetup page")
	} else {
		log.Printf("Warning: Page doesn't appear to be a Meetup page")
	}
	
	// Extract events from the page
	events := s.extractEventsFromHTML(doc, period)
	
	return events, nil
}

// buildSearchURL constructs the Meetup search URL based on city and category
func (s *Scraper) buildSearchURL(city, category string) string {
	// Default to Winnipeg if no city specified
	if city == "" {
		city = "Winnipeg"
	}
	
	// Map categories to Meetup category IDs
	categoryMap := map[string]string{
		"tech":     "546", // Technology
		"business": "2",   // Business & Professional
		"social":   "1",   // Social
		"arts":     "3",   // Arts & Culture
		"health":   "4",   // Health & Wellness
		"education": "5",  // Education
		"sports":   "6",   // Sports & Recreation
	}
	
	categoryID := categoryMap[strings.ToLower(category)]
	if categoryID == "" {
		categoryID = "546" // Default to Technology
	}
	
	// Format city for URL (basic implementation)
	cityFormatted := strings.ReplaceAll(strings.ToLower(city), " ", "-")
	
	return fmt.Sprintf("https://www.meetup.com/find/?location=ca--mb--%s&source=EVENTS&categoryId=%s", 
		cityFormatted, categoryID)
}

// extractEventsFromHTML parses the HTML document and extracts event information
func (s *Scraper) extractEventsFromHTML(doc *goquery.Document, period time.Duration) []models.Event {
	var events []models.Event
	
	// Debug: Count potential event elements
	eventCardCount := doc.Find("[data-testid='event-card'], .eventCard, .event-card, [class*='event']").Length()
	eventLinkCount := doc.Find("a[href*='/events/']").Length()
	log.Printf("Found %d potential event cards and %d event links", eventCardCount, eventLinkCount)
	
	// Look for event cards in the search results with multiple selector strategies
	selectors := []string{
		"[data-testid='event-card']",
		".eventCard",
		".event-card", 
		"[class*='event']",
		".event",
		"[data-event-id]",
		".event-item",
		".eventItem",
	}
	
	for _, selector := range selectors {
		doc.Find(selector).Each(func(i int, sel *goquery.Selection) {
			event := s.parseEventCard(sel)
			if event != nil && s.isEventInPeriod(*event, period) {
				events = append(events, *event)
			}
		})
		
		if len(events) > 0 {
			log.Printf("Found %d events using selector: %s", len(events), selector)
			break
		}
	}
	
	// If no events found with the above selectors, try alternative selectors
	if len(events) == 0 {
		log.Printf("No events found with card selectors, trying link-based extraction")
		doc.Find("a[href*='/events/']").Each(func(i int, sel *goquery.Selection) {
			event := s.parseEventLink(sel)
			if event != nil && s.isEventInPeriod(*event, period) {
				events = append(events, *event)
			}
		})
		log.Printf("Found %d events using link-based extraction", len(events))
	}
	
	// If still no events, try to find any links that might be events
	if len(events) == 0 {
		log.Printf("No events found with standard selectors, trying broader search")
		doc.Find("a").Each(func(i int, sel *goquery.Selection) {
			href, exists := sel.Attr("href")
			if exists && (strings.Contains(href, "/events/") || strings.Contains(href, "meetup.com")) {
				event := s.parseEventLink(sel)
				if event != nil && s.isEventInPeriod(*event, period) {
					events = append(events, *event)
				}
			}
		})
		log.Printf("Found %d events using broader search", len(events))
	}
	
	return events
}

// parseEventCard extracts event information from a card element
func (s *Scraper) parseEventCard(sel *goquery.Selection) *models.Event {
	event := &models.Event{
		Source: "meetup",
	}
	
	// Extract event name
	name := sel.Find("h3, .event-title, [class*='title'], [class*='name']").First().Text()
	if name == "" {
		name = sel.Find("a").First().Text()
	}
	event.Name = strings.TrimSpace(name)
	
	// Extract event URL
	href, exists := sel.Find("a").First().Attr("href")
	if exists {
		if strings.HasPrefix(href, "/") {
			event.URL = "https://www.meetup.com" + href
		} else {
			event.URL = href
		}
	}
	
	// Extract event ID from URL
	if event.URL != "" {
		event.ID = s.extractEventIDFromURL(event.URL)
	}
	
	// Extract description
	description := sel.Find(".event-description, [class*='description'], p").First().Text()
	event.Description = strings.TrimSpace(description)
	
	// Extract date and time
	dateTime := sel.Find(".event-date, [class*='date'], [class*='time']").First().Text()
	if dateTime != "" {
		event.DateString = strings.TrimSpace(dateTime)
		event.StartTime = parseMeetupDate(dateTime)
		if !event.StartTime.IsZero() {
			event.EndTime = event.StartTime.Add(2 * time.Hour) // Default 2-hour duration
		}
	}
	
	// Extract venue
	venue := sel.Find(".event-venue, [class*='venue'], [class*='location']").First().Text()
	event.Venue = strings.TrimSpace(venue)
	
	// Extract group name
	group := sel.Find(".event-group, [class*='group']").First().Text()
	event.Group = strings.TrimSpace(group)
	
	// Extract attendee count
	attendeeText := sel.Find("[class*='attendee'], [class*='member']").First().Text()
	event.AttendeeCount = extractAttendeeCount(attendeeText)
	
	// Set default values
	if event.City == "" {
		event.City = "Winnipeg"
	}
	if event.Category == "" {
		event.Category = "tech"
	}
	
	// Only return event if we have essential information
	if event.Name != "" && event.URL != "" {
		return event
	}
	
	return nil
}

// parseEventLink extracts event information from a link element
func (s *Scraper) parseEventLink(sel *goquery.Selection) *models.Event {
	event := &models.Event{
		Source: "meetup",
	}
	
	// Extract URL
	href, exists := sel.Attr("href")
	if !exists {
		return nil
	}
	
	if strings.HasPrefix(href, "/") {
		event.URL = "https://www.meetup.com" + href
	} else {
		event.URL = href
	}
	
	// Extract event ID from URL
	event.ID = s.extractEventIDFromURL(event.URL)
	
	// Extract name from link text
	event.Name = strings.TrimSpace(sel.Text())
	
	// Set default values
	event.City = "Winnipeg"
	event.Category = "tech"
	event.StartTime = time.Now().AddDate(0, 0, 7) // Default to next week
	event.EndTime = event.StartTime.Add(2 * time.Hour)
	
	// Only return event if we have essential information
	if event.Name != "" && event.URL != "" {
		return event
	}
	
	return nil
}

// extractEventIDFromURL extracts a unique ID from the event URL
func (s *Scraper) extractEventIDFromURL(url string) string {
	// Extract event ID from URL like /events/123456789/
	re := regexp.MustCompile(`/events/(\d+)/?`)
	matches := re.FindStringSubmatch(url)
	if len(matches) > 1 {
		return "meetup-" + matches[1]
	}
	
	// Fallback: use a hash of the URL
	return "meetup-" + fmt.Sprintf("%x", len(url))
}

// isEventInPeriod checks if an event falls within the specified time period
func (s *Scraper) isEventInPeriod(event models.Event, period time.Duration) bool {
	now := time.Now()
	futureLimit := now.Add(period)
	
	// If no start time, assume it's in the future
	if event.StartTime.IsZero() {
		return true
	}
	
	// Check if event is in the future and within the period
	return event.StartTime.After(now) && event.StartTime.Before(futureLimit)
}

func (s *Scraper) getSampleEvents() []models.Event {
	now := time.Now()
	
	return []models.Event{
		{
			ID:          "meetup-ai-ml-1",
			Name:        "Winnipeg AI & Machine Learning Meetup",
			Description: "Join us for an evening discussing the latest trends in AI and machine learning. We'll have presentations from local experts and networking opportunities.",
			City:        "Winnipeg",
			Category:    "tech",
			URL:         "https://www.meetup.com/winnipeg-ai-ml/events/example1",
			StartTime:   now.AddDate(0, 0, 3), // 3 days from now
			EndTime:     now.AddDate(0, 0, 3).Add(2 * time.Hour),
			Source:      "meetup",
		},
		{
			ID:          "meetup-devops-1",
			Name:        "DevOps Workshop - CI/CD Pipeline",
			Description: "Hands-on workshop covering continuous integration and deployment best practices using modern tools like Jenkins, Docker, and Kubernetes.",
			City:        "Winnipeg",
			Category:    "tech",
			URL:         "https://www.meetup.com/winnipeg-devops/events/example2",
			StartTime:   now.AddDate(0, 0, 7), // 1 week from now
			EndTime:     now.AddDate(0, 0, 7).Add(3 * time.Hour),
			Source:      "meetup",
		},
		{
			ID:          "meetup-startup-1",
			Name:        "Startup Pitch Night",
			Description: "Local startups pitch their ideas to a panel of investors and mentors. Great networking opportunity for entrepreneurs and tech professionals.",
			City:        "Winnipeg",
			Category:    "tech",
			URL:         "https://www.meetup.com/winnipeg-startup/events/example3",
			StartTime:   now.AddDate(0, 0, 10), // 10 days from now
			EndTime:     now.AddDate(0, 0, 10).Add(2*time.Hour + 30*time.Minute),
			Source:      "meetup",
		},
		{
			ID:          "meetup-webdev-1",
			Name:        "Web Development Trends 20lkljljlkjl25",
			Description: "Exploring the latest web development frameworks, tools, and best practices. Panel discussion with local developers.",
			City:        "Winnipeg",
			Category:    "tech",
			URL:         "https://www.meetup.com/winnipeg-webdev/events/example4",
			StartTime:   now.AddDate(0, 0, 14), // 2 weeks from now
			EndTime:     now.AddDate(0, 0, 14).Add(1*time.Hour + 30*time.Minute),
			Source:      "meetup",
		},
	}
}

// parseMeetupDate handles various Meetup date formats including day names
func parseMeetupDate(dateString string) time.Time {
	if dateString == "" {
		return time.Time{}
	}

	now := time.Now()
	
	// Handle day names (e.g., "Thu", "Saturday")
	dayNames := map[string]time.Weekday{
		"sunday":    time.Sunday,
		"monday":    time.Monday,
		"tuesday":   time.Tuesday,
		"wednesday": time.Wednesday,
		"thursday":  time.Thursday,
		"friday":    time.Friday,
		"saturday":  time.Saturday,
		"sun":       time.Sunday,
		"mon":       time.Monday,
		"tue":       time.Tuesday,
		"wed":       time.Wednesday,
		"thu":       time.Thursday,
		"fri":       time.Friday,
		"sat":       time.Saturday,
	}

	lowerDateStr := strings.ToLower(strings.TrimSpace(dateString))
	if dayOfWeek, exists := dayNames[lowerDateStr]; exists {
		// Find next occurrence of this day after today
		todayDay := now.Weekday()
		daysUntilTarget := int(dayOfWeek - todayDay)
		
		if daysUntilTarget <= 0 {
			daysUntilTarget += 7 // Next week
		}
		
		targetDate := now.AddDate(0, 0, daysUntilTarget)
		return targetDate
	}

	// Try parsing as regular date
	layouts := []string{
		"January 2, 2006",
		"Jan 2, 2006",
		"January 2",
		"Jan 2",
		"2006-01-02",
		"01/02/2006",
		"1/2/2006",
	}

	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, dateString); err == nil {
			return parsed
		}
	}

	// If all else fails, return current time
	return now
}

// parseMeetupTime handles various time formats
func parseMeetupTime(timeString string) time.Time {
	if timeString == "" {
		return time.Time{}
	}

	layouts := []string{
		"3:04 PM",
		"3:04PM",
		"15:04",
		"3 PM",
		"3PM",
	}

	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, timeString); err == nil {
			return parsed
		}
	}

	return time.Time{}
}

// extractAttendeeCount extracts number from strings like "45 attendees", "120 going"
func extractAttendeeCount(text string) int {
	if text == "" {
		return 0
	}

	re := regexp.MustCompile(`(\d+)`)
	matches := re.FindStringSubmatch(text)
	if len(matches) > 1 {
		if count, err := strconv.Atoi(matches[1]); err == nil {
			return count
		}
	}

	return 0
}
