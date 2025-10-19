package scraping

import (
	"event_calendar/internal/models"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"
)

// EventbriteScraper scrapes events from Eventbrite.com
type EventbriteScraper struct {
	*BaseScraper
}

// NewEventbriteScraper creates a new Eventbrite scraper
func NewEventbriteScraper() *EventbriteScraper {
	base := NewBaseScraper("eventbrite", "https://www.eventbrite.ca")
	return &EventbriteScraper{
		BaseScraper: base,
	}
}

// GetEvents scrapes events from Eventbrite.com
func (e *EventbriteScraper) GetEvents(city, category string, period time.Duration) ([]models.Event, error) {
	log.Printf("🔍 [Eventbrite] Starting event fetch for city: %s, category: %s, period: %v", city, category, period)
	
	events, err := e.fetchEventsFromEventbrite(city, category, period)
	e.LogScrapingResult(events, err)
	
	if err != nil {
		log.Printf("⚠️  [Eventbrite] Scraping failed, falling back to sample data: %v", err)
		sampleEvents := e.getSampleEvents()
		log.Printf("📋 [Eventbrite] Returning %d sample events", len(sampleEvents))
		return sampleEvents, nil
	}
	
	log.Printf("✅ [Eventbrite] Successfully fetched %d events", len(events))
	return events, nil
}

// fetchEventsFromEventbrite performs the actual scraping from Eventbrite.com
func (e *EventbriteScraper) fetchEventsFromEventbrite(city, category string, period time.Duration) ([]models.Event, error) {
	// Build the search URL based on parameters
	searchURL := e.buildSearchURL(city, category)
	log.Printf("🌐 [Eventbrite] Fetching URL: %s", searchURL)
	
	// Fetch the page
	startTime := time.Now()
	resp, err := e.client.R().Get(searchURL)
	fetchDuration := time.Since(startTime)
	
	if err != nil {
		log.Printf("❌ [Eventbrite] HTTP request failed after %v: %v", fetchDuration, err)
		return nil, fmt.Errorf("failed to fetch Eventbrite page: %w", err)
	}
	
	log.Printf("📡 [Eventbrite] HTTP response received in %v, status: %d, size: %d bytes", 
		fetchDuration, resp.StatusCode(), len(resp.Body()))
	
	if resp.StatusCode() != 200 {
		log.Printf("❌ [Eventbrite] Non-200 status code: %d", resp.StatusCode())
		return nil, fmt.Errorf("received non-200 status code: %d", resp.StatusCode())
	}
	
	// For now, return sample data since Eventbrite has strong anti-scraping measures
	// In a real implementation, you would parse the HTML here
	log.Printf("⚠️  [Eventbrite] Scraping not fully implemented due to anti-scraping measures")
	log.Printf("📋 [Eventbrite] Returning sample data instead")
	
	return e.getSampleEvents(), nil
}

// buildSearchURL constructs the Eventbrite search URL based on city and category
func (e *EventbriteScraper) buildSearchURL(city, category string) string {
	// Default to Winnipeg if no city specified
	if city == "" {
		city = "Winnipeg"
	}
	
	// Map categories to Eventbrite category paths
	categoryMap := map[string]string{
		"tech":      "tech-event",
		"business":  "business-event",
		"social":    "social-event",
		"arts":      "arts-event",
		"health":    "health-event",
		"education": "education-event",
		"sports":    "sports-event",
	}
	
	categoryPath := categoryMap[strings.ToLower(category)]
	if categoryPath == "" {
		categoryPath = "tech-event" // Default to Technology
	}
	
	// Format city for URL
	cityFormatted := strings.ReplaceAll(strings.ToLower(city), " ", "-")
	
	return fmt.Sprintf("https://www.eventbrite.ca/d/canada--%s/%s/", cityFormatted, categoryPath)
}

// getSampleEvents returns sample events for fallback
func (e *EventbriteScraper) getSampleEvents() []models.Event {
	now := time.Now()
	
	return []models.Event{
		{
			ID:          "eventbrite-conference-1",
			Name:        "Winnipeg Tech Conference 2025",
			Description: "Annual technology conference featuring local and international speakers discussing the latest trends in software development, AI, and digital transformation.",
			City:        "Winnipeg",
			Category:    "tech",
			URL:         "https://www.eventbrite.ca/e/winnipeg-tech-conference-2025-tickets-example1",
			StartTime:   time.Date(2025, 3, 15, 9, 0, 0, 0, time.FixedZone("CST", -6*3600)), // March 15, 2025 9:00 AM CST
			EndTime:     time.Date(2025, 3, 15, 17, 0, 0, 0, time.FixedZone("CST", -6*3600)), // March 15, 2025 5:00 PM CST
			Source:      "eventbrite",
		},
		{
			ID:          "eventbrite-workshop-1",
			Name:        "React Native Development Workshop",
			Description: "Learn to build mobile applications using React Native. This hands-on workshop covers the fundamentals and advanced concepts.",
			City:        "Winnipeg",
			Category:    "tech",
			URL:         "https://www.eventbrite.ca/e/react-native-workshop-winnipeg-tickets-example2",
			StartTime:   now.AddDate(0, 0, 5), // 5 days from now
			EndTime:     now.AddDate(0, 0, 5).Add(6 * time.Hour),
			Source:      "eventbrite",
		},
		{
			ID:          "eventbrite-networking-1",
			Name:        "Tech Networking Mixer",
			Description: "Connect with fellow tech professionals, entrepreneurs, and innovators in Winnipeg. Food and drinks provided.",
			City:        "Winnipeg",
			Category:    "tech",
			URL:         "https://www.eventbrite.ca/e/winnipeg-tech-networking-mixer-tickets-example3",
			StartTime:   now.AddDate(0, 0, 12), // 12 days from now
			EndTime:     now.AddDate(0, 0, 12).Add(2*time.Hour + 30*time.Minute),
			Source:      "eventbrite",
		},
		{
			ID:          "eventbrite-hackathon-1",
			Name:        "Winnipeg Hackathon 2025",
			Description: "48-hour coding competition bringing together developers, designers, and entrepreneurs to build innovative solutions.",
			City:        "Winnipeg",
			Category:    "tech",
			URL:         "https://www.eventbrite.ca/e/winnipeg-hackathon-2025-tickets-example4",
			StartTime:   time.Date(2025, 4, 20, 18, 0, 0, 0, time.FixedZone("CST", -6*3600)), // April 20, 2025 6:00 PM CST
			EndTime:     time.Date(2025, 4, 22, 18, 0, 0, 0, time.FixedZone("CST", -6*3600)), // April 22, 2025 6:00 PM CST
			Source:      "eventbrite",
		},
	}
}

// parseEventbriteDateTime parses ISO datetime strings from Eventbrite
// Format: "2025-11-05T17:00:00-06:00[America/Winnipeg]"
func (e *EventbriteScraper) parseEventbriteDateTime(dateTimeStr string) time.Time {
	if dateTimeStr == "" {
		return time.Time{}
	}

	// Remove timezone info in brackets
	re := regexp.MustCompile(`\[.*?\]`)
	cleanDateTime := re.ReplaceAllString(dateTimeStr, "")
	
	// Parse the ISO datetime
	layouts := []string{
		"2006-01-02T15:04:05-07:00",
		"2006-01-02T15:04:05Z07:00",
		"2006-01-02T15:04:05Z",
		time.RFC3339,
		time.RFC3339Nano,
	}

	for _, layout := range layouts {
		if parsed, err := time.Parse(layout, cleanDateTime); err == nil {
			return parsed
		}
	}

	// If all else fails, return current time
	return time.Now()
}

// extractPrice extracts price information from Eventbrite price strings
func (e *EventbriteScraper) extractPrice(priceStr string) string {
	if priceStr == "" {
		return "Free"
	}

	// Handle various price formats
	priceStr = strings.TrimSpace(priceStr)
	
	if strings.ToLower(priceStr) == "free" {
		return "Free"
	}
	
	// Extract price with currency
	re := regexp.MustCompile(`\$[\d,]+\.?\d*`)
	if match := re.FindString(priceStr); match != "" {
		return match
	}
	
	return priceStr
}

// extractVenue extracts venue information from Eventbrite venue strings
func (e *EventbriteScraper) extractVenue(venueStr string) string {
	if venueStr == "" {
		return ""
	}
	
	// Clean up venue string
	venueStr = strings.TrimSpace(venueStr)
	
	// Remove common suffixes
	suffixes := []string{
		", Winnipeg, MB, Canada",
		", Winnipeg",
		", MB",
		", Canada",
	}
	
	for _, suffix := range suffixes {
		if strings.HasSuffix(venueStr, suffix) {
			venueStr = strings.TrimSuffix(venueStr, suffix)
		}
	}
	
	return venueStr
}
