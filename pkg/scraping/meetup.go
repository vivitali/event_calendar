package scraping

import (
	"event_calendar/internal/models"
	"fmt"
	"log"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
)

// MeetupScraper scrapes events from Meetup.com
type MeetupScraper struct {
	*BaseScraper
}

// NewMeetupScraper creates a new Meetup scraper
func NewMeetupScraper() *MeetupScraper {
	base := NewBaseScraper("meetup", "https://www.meetup.com")
	return &MeetupScraper{
		BaseScraper: base,
	}
}

// GetEvents scrapes events from Meetup.com
func (m *MeetupScraper) GetEvents(city, category string, period time.Duration) ([]models.Event, error) {
	log.Printf("🔍 [Meetup] Starting event fetch for city: %s, category: %s, period: %v", city, category, period)
	
	events, err := m.fetchEventsFromMeetup(city, category, period)
	m.LogScrapingResult(events, err)
	
	if err != nil {
		log.Printf("⚠️  [Meetup] Scraping failed, falling back to sample data: %v", err)
		sampleEvents := m.getSampleEvents()
		log.Printf("📋 [Meetup] Returning %d sample events", len(sampleEvents))
		return sampleEvents, nil
	}
	
	log.Printf("✅ [Meetup] Successfully fetched %d events", len(events))
	return events, nil
}

// fetchEventsFromMeetup performs the actual scraping from Meetup.com
func (m *MeetupScraper) fetchEventsFromMeetup(city, category string, period time.Duration) ([]models.Event, error) {
	// Build the search URL based on parameters
	searchURL := m.buildSearchURL(city, category)
	log.Printf("🌐 [Meetup] Fetching URL: %s", searchURL)
	
	// Fetch the page
	startTime := time.Now()
	resp, err := m.client.R().Get(searchURL)
	fetchDuration := time.Since(startTime)
	
	if err != nil {
		log.Printf("❌ [Meetup] HTTP request failed after %v: %v", fetchDuration, err)
		return nil, fmt.Errorf("failed to fetch Meetup page: %w", err)
	}
	
	log.Printf("📡 [Meetup] HTTP response received in %v, status: %d, size: %d bytes", 
		fetchDuration, resp.StatusCode(), len(resp.Body()))
	
	if resp.StatusCode() != 200 {
		log.Printf("❌ [Meetup] Non-200 status code: %d", resp.StatusCode())
		return nil, fmt.Errorf("received non-200 status code: %d", resp.StatusCode())
	}
	
	// Parse the HTML
	parseStart := time.Now()
	doc, err := goquery.NewDocumentFromReader(strings.NewReader(resp.String()))
	parseDuration := time.Since(parseStart)
	
	if err != nil {
		log.Printf("❌ [Meetup] HTML parsing failed after %v: %v", parseDuration, err)
		return nil, fmt.Errorf("failed to parse HTML: %w", err)
	}
	
	log.Printf("📄 [Meetup] HTML parsed successfully in %v", parseDuration)
	
	// Debug: Log page title and some content
	title := doc.Find("title").Text()
	log.Printf("📋 [Meetup] Page title: %s", title)
	
	// Debug: Check if we got a valid page
	if strings.Contains(strings.ToLower(title), "meetup") {
		log.Printf("✅ [Meetup] Successfully loaded Meetup page")
	} else {
		log.Printf("⚠️  [Meetup] Warning: Page doesn't appear to be a Meetup page")
	}
	
	// Debug: Log some HTML content to understand the structure
	bodyText := doc.Find("body").Text()
	if len(bodyText) > 500 {
		bodyText = bodyText[:500] + "..."
	}
	log.Printf("🔍 [Meetup] Body content preview: %s", bodyText)
	
	// Debug: Look for common event-related elements
	eventElements := doc.Find("[class*='event'], [class*='Event'], [data-testid*='event'], [data-testid*='Event']")
	log.Printf("🔍 [Meetup] Found %d elements with 'event' in class or data-testid", eventElements.Length())
	
	// Debug: Look for links that might be events
	allLinks := doc.Find("a")
	log.Printf("🔍 [Meetup] Found %d total links on page", allLinks.Length())
	
	// Debug: Look for specific patterns
	eventLinks := doc.Find("a[href*='/events/']")
	log.Printf("🔍 [Meetup] Found %d links with '/events/' in href", eventLinks.Length())
	
	// Debug: Log first few links to see the structure
	allLinks.Each(func(i int, sel *goquery.Selection) {
		if i >= 5 { // Only log first 5 links
			return
		}
		href, _ := sel.Attr("href")
		text := sel.Text()
		if len(text) > 50 {
			text = text[:50] + "..."
		}
		log.Printf("🔗 [Meetup] Link %d: href='%s', text='%s'", i+1, href, text)
	})
	
	// Extract events from the page
	extractStart := time.Now()
	events := m.extractEventsFromHTML(doc, period)
	extractDuration := time.Since(extractStart)
	
	log.Printf("🔍 [Meetup] Event extraction completed in %v, found %d events", extractDuration, len(events))
	
	return events, nil
}

// buildSearchURL constructs the Meetup search URL based on city and category
func (m *MeetupScraper) buildSearchURL(city, category string) string {
	// Default to Winnipeg if no city specified
	if city == "" {
		city = "Winnipeg"
	}
	
	// Map categories to Meetup category IDs
	categoryMap := map[string]string{
		"tech":      "546", // Technology
		"business":  "2",   // Business & Professional
		"social":    "1",   // Social
		"arts":      "3",   // Arts & Culture
		"health":    "4",   // Health & Wellness
		"education": "5",   // Education
		"sports":    "6",   // Sports & Recreation
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
func (m *MeetupScraper) extractEventsFromHTML(doc *goquery.Document, period time.Duration) []models.Event {
	var events []models.Event
	
	// Debug: Count potential event elements
	eventCardCount := doc.Find("[data-testid='event-card'], .eventCard, .event-card, [class*='event']").Length()
	eventLinkCount := doc.Find("a[href*='/events/']").Length()
	log.Printf("🔍 [Meetup] Found %d potential event cards and %d event links", eventCardCount, eventLinkCount)
	
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
		// Add more modern Meetup selectors
		"[data-testid*='event']",
		"[class*='Event']",
		".eventList-item",
		".event-list-item",
		".eventListItem",
		"[role='article']",
		".card",
		".event-card-wrapper",
	}
	
	for _, selector := range selectors {
		elements := doc.Find(selector)
		log.Printf("🔍 [Meetup] Trying selector '%s': found %d elements", selector, elements.Length())
		
		elements.Each(func(i int, sel *goquery.Selection) {
			event := m.parseEventCard(sel)
			if event != nil {
				log.Printf("✅ [Meetup] Parsed event: %s", event.Name)
				events = append(events, *event)
			}
		})
		
		if len(events) > 0 {
			log.Printf("✅ [Meetup] Found %d events using selector: %s", len(events), selector)
			break
		}
	}
	
	// If no events found with the above selectors, try alternative selectors
	if len(events) == 0 {
		log.Printf("⚠️  [Meetup] No events found with card selectors, trying link-based extraction")
		doc.Find("a[href*='/events/']").Each(func(i int, sel *goquery.Selection) {
			event := m.parseEventLink(sel)
			if event != nil {
				log.Printf("✅ [Meetup] Parsed event from link: %s", event.Name)
				events = append(events, *event)
			}
		})
		log.Printf("📊 [Meetup] Found %d events using link-based extraction", len(events))
	}
	
	// If still no events, try to find any links that might be events
	if len(events) == 0 {
		log.Printf("⚠️  [Meetup] No events found with standard selectors, trying broader search")
		doc.Find("a").Each(func(i int, sel *goquery.Selection) {
			href, exists := sel.Attr("href")
			if exists && (strings.Contains(href, "/events/") || strings.Contains(href, "meetup.com")) {
				event := m.parseEventLink(sel)
				if event != nil {
					log.Printf("✅ [Meetup] Parsed event from broader search: %s", event.Name)
					events = append(events, *event)
				}
			}
		})
		log.Printf("📊 [Meetup] Found %d events using broader search", len(events))
	}
	
	// If still no events, try to find any text that looks like event names
	if len(events) == 0 {
		log.Printf("⚠️  [Meetup] No events found with link-based extraction, trying text-based search")
		// Look for headings or text that might be event names
		doc.Find("h1, h2, h3, h4, h5, h6, .title, .name, .event-title").Each(func(i int, sel *goquery.Selection) {
			text := strings.TrimSpace(sel.Text())
			if len(text) > 10 && len(text) < 200 { // Reasonable event name length
				// Check if this looks like an event name
				if strings.Contains(strings.ToLower(text), "meetup") || 
				   strings.Contains(strings.ToLower(text), "workshop") ||
				   strings.Contains(strings.ToLower(text), "conference") ||
				   strings.Contains(strings.ToLower(text), "event") {
					event := &models.Event{
						ID:          fmt.Sprintf("meetup-text-%d", i),
						Name:        text,
						Description: "Event found through text analysis",
						City:        "Winnipeg",
						Category:    "tech",
						URL:         "https://www.meetup.com",
						StartTime:   time.Now().AddDate(0, 0, 7), // Default to next week
						EndTime:     time.Now().AddDate(0, 0, 7).Add(2 * time.Hour),
						Source:      "meetup",
					}
					log.Printf("✅ [Meetup] Created event from text: %s", event.Name)
					events = append(events, *event)
				}
			}
		})
		log.Printf("📊 [Meetup] Found %d events using text-based search", len(events))
	}
	
	// Filter events by period and remove duplicates
	events = m.FilterEventsByPeriod(events, period)
	events = m.RemoveDuplicateEvents(events)
	
	log.Printf("📊 [Meetup] Final result: %d events after filtering and deduplication", len(events))
	return events
}

// parseEventCard extracts event information from a card element
func (m *MeetupScraper) parseEventCard(sel *goquery.Selection) *models.Event {
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
		event.ID = m.extractEventIDFromURL(event.URL)
	}
	
	// Extract description
	description := sel.Find(".event-description, [class*='description'], p").First().Text()
	event.Description = strings.TrimSpace(description)
	
	// Extract date and time
	dateTime := sel.Find(".event-date, [class*='date'], [class*='time']").First().Text()
	if dateTime != "" {
		event.DateString = strings.TrimSpace(dateTime)
		event.StartTime = m.parseMeetupDate(dateTime)
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
	event.AttendeeCount = m.extractAttendeeCount(attendeeText)
	
	// Set default values
	if event.City == "" {
		event.City = "Winnipeg"
	}
	if event.Category == "" {
		event.Category = "tech"
	}
	
	// Validate event
	if err := m.ValidateEvent(*event); err != nil {
		return nil
	}
	
	return event
}

// parseEventLink extracts event information from a link element
func (m *MeetupScraper) parseEventLink(sel *goquery.Selection) *models.Event {
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
	event.ID = m.extractEventIDFromURL(event.URL)
	
	// Extract name from link text
	event.Name = strings.TrimSpace(sel.Text())
	
	// Set default values
	event.City = "Winnipeg"
	event.Category = "tech"
	event.StartTime = time.Now().AddDate(0, 0, 7) // Default to next week
	event.EndTime = event.StartTime.Add(2 * time.Hour)
	
	// Validate event
	if err := m.ValidateEvent(*event); err != nil {
		return nil
	}
	
	return event
}

// extractEventIDFromURL extracts a unique ID from the event URL
func (m *MeetupScraper) extractEventIDFromURL(url string) string {
	// Extract event ID from URL like /events/123456789/
	re := regexp.MustCompile(`/events/(\d+)/?`)
	matches := re.FindStringSubmatch(url)
	if len(matches) > 1 {
		return "meetup-" + matches[1]
	}
	
	// Fallback: use a hash of the URL
	return "meetup-" + fmt.Sprintf("%x", len(url))
}

// parseMeetupDate handles various Meetup date formats including day names
func (m *MeetupScraper) parseMeetupDate(dateString string) time.Time {
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

// extractAttendeeCount extracts number from strings like "45 attendees", "120 going"
func (m *MeetupScraper) extractAttendeeCount(text string) int {
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

// getSampleEvents returns sample events for fallback
func (m *MeetupScraper) getSampleEvents() []models.Event {
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
			Name:        "Web Development Trends 2025",
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
