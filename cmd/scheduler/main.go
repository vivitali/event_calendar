package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"event_calendar/internal/models"
	"event_calendar/pkg/devevents"
	"event_calendar/pkg/scraping"
	"event_calendar/pkg/telegram"
)

type SchedulerConfig struct {
	BotToken    string `json:"bot_token"`
	ChatID      string `json:"chat_id"`
	TestMode    bool   `json:"test_mode"`
	City        string `json:"city"`
	Categories  string `json:"categories"`
	PeriodDays  int    `json:"period_days"`
}

type SchedulerResult struct {
	Success     bool      `json:"success"`
	EventsCount int       `json:"events_count"`
	MessageSent bool      `json:"message_sent"`
	Timestamp   time.Time `json:"timestamp"`
	Error       string    `json:"error,omitempty"`
	Logs        []string  `json:"logs"`
}

func main() {
	log.Println("🚀 Winnipeg Tech Events Scheduler Starting...")
	
	// Load configuration
	config := loadConfig()
	
	// Run the scheduler
	result := runScheduler(config)
	
	// Log results
	logResult(result)
	
	// Exit with appropriate code
	if result.Success {
		os.Exit(0)
	} else {
		os.Exit(1)
	}
}

func loadConfig() *SchedulerConfig {
	config := &SchedulerConfig{
		City:       "Winnipeg",
		Categories: "tech",
		PeriodDays: 30,
		TestMode:   false,
	}
	
	// Load from environment variables
	if botToken := os.Getenv("TELEGRAM_BOT_TOKEN"); botToken != "" {
		config.BotToken = botToken
	}
	
	if chatID := os.Getenv("TELEGRAM_CHAT_ID"); chatID != "" {
		config.ChatID = chatID
	}
	
	if testMode := os.Getenv("TEST_MODE"); testMode == "true" {
		config.TestMode = true
	}
	
	if city := os.Getenv("CITY"); city != "" {
		config.City = city
	}
	
	if categories := os.Getenv("CATEGORIES"); categories != "" {
		config.Categories = categories
	}
	
	if periodDays := os.Getenv("PERIOD_DAYS"); periodDays != "" {
		if pd, err := time.ParseDuration(periodDays + "h"); err == nil {
			config.PeriodDays = int(pd.Hours() / 24)
		}
	}
	
	log.Printf("📋 Configuration loaded: City=%s, Categories=%s, PeriodDays=%d, TestMode=%t", 
		config.City, config.Categories, config.PeriodDays, config.TestMode)
	
	return config
}

func runScheduler(config *SchedulerConfig) *SchedulerResult {
	result := &SchedulerResult{
		Timestamp: time.Now(),
		Logs:      []string{},
	}
	
	// Initialize scraping service
	log.Println("🔧 Initializing event scrapers...")
	factory := scraping.NewScrapingServiceFactory()
	scrapingService := factory.CreateDefaultService()
	
	// Also include devevents scraper for backward compatibility
	devEventsScraper := devevents.NewScraper()
	
	// Fetch events
	log.Println("📡 Fetching events from all sources...")
	period := time.Duration(config.PeriodDays) * 24 * time.Hour
	
	// Use the new scraping service
	events, err := scrapingService.ScrapeEvents(config.City, config.Categories, period)
	if err != nil {
		result.Error = fmt.Sprintf("Failed to scrape events: %v", err)
		result.Logs = append(result.Logs, result.Error)
		return result
	}
	
	// Also fetch from devevents
	devEvents, err := devEventsScraper.GetEvents(config.City, config.Categories, period)
	if err != nil {
		log.Printf("DevEvents scraping error: %v", err)
	} else {
		events = append(events, devEvents...)
	}
	
	result.EventsCount = len(events)
	result.Logs = append(result.Logs, fmt.Sprintf("Successfully aggregated %d events", len(events)))
	
	if len(events) == 0 {
		result.Logs = append(result.Logs, "No events found to post")
		result.Success = true
		return result
	}
	
	// Filter future events only
	futureEvents := filterFutureEventsFromModels(events)
	result.Logs = append(result.Logs, fmt.Sprintf("Filtered to %d future events", len(futureEvents)))
	
	if len(futureEvents) == 0 {
		result.Logs = append(result.Logs, "No future events found to post")
		result.Success = true
		return result
	}
	
	// Generate Telegram message
	log.Println("📝 Generating Telegram message...")
	message := generateTelegramMessageFromModels(futureEvents)
	result.Logs = append(result.Logs, fmt.Sprintf("Generated message with %d characters", len(message)))
	
	// Check if we should actually send
	if config.TestMode {
		result.Logs = append(result.Logs, "🧪 TEST MODE: Message would be sent but not actually posted")
		result.Success = true
		result.MessageSent = false
		return result
	}
	
	// Send to Telegram
	if config.BotToken == "" || config.ChatID == "" {
		result.Error = "Telegram bot token or chat ID not configured"
		result.Logs = append(result.Logs, result.Error)
		return result
	}
	
	log.Println("📤 Sending message to Telegram...")
	telegramService := telegram.NewService(config.BotToken)
	
	// Create vote keyboard for the overall message
	keyboard := telegramService.CreateVoteKeyboard()
	
	err = telegramService.SendMessageWithKeyboard(config.ChatID, message, keyboard)
	
	if err != nil {
		result.Error = fmt.Sprintf("Failed to send Telegram message: %v", err)
		result.Logs = append(result.Logs, result.Error)
		return result
	}
	
	result.Success = true
	result.MessageSent = true
	result.Logs = append(result.Logs, "✅ Message sent to Telegram successfully")
	
	return result
}

func filterFutureEventsFromModels(events []models.Event) []models.Event {
	now := time.Now()
	var future []models.Event
	
	for _, event := range events {
		if event.StartTime.After(now) {
			future = append(future, event)
		}
	}
	
	return future
}

func generateTelegramMessageFromModels(events []models.Event) string {
	now := time.Now()
	dateStr := now.Format("Monday, January 2, 2006")
	
	message := fmt.Sprintf("🚀 *Winnipeg Tech Events - %s*\n\n", dateStr)
	
	// Group events by time period
	groups := groupEventsByTimeFromModels(events)
	
	for period, periodEvents := range groups {
		if len(periodEvents) > 0 {
			message += fmt.Sprintf("*%s:*\n", period)
			for _, event := range periodEvents {
				// Event title with source label
				sourceLabel := getSourceLabel(event.Source)
				message += fmt.Sprintf("• %s %s\n", event.Name, sourceLabel)
				
				// Format date nicely
				if !event.StartTime.IsZero() {
					dateStr := event.StartTime.Format("Monday, Jan 2")
					message += fmt.Sprintf("  📅 %s\n", dateStr)
				}
				
				if event.Venue != "" {
					message += fmt.Sprintf("  📍 %s\n", event.Venue)
				}
				
				if event.Price != "" && event.Price != "Free" {
					message += fmt.Sprintf("  💰 %s\n", event.Price)
				}
				
				if event.URL != "" {
					message += fmt.Sprintf("  🔗 [View Event](%s)\n", event.URL)
				}
				
				message += "\n"
			}
		}
	}
	
	message += "\n_Shared via Winnipeg Tech Events Tracker_"
	
	return message
}

func getSourceLabel(source string) string {
	switch source {
	case "meetup":
		return "`[Meetup]`"
	case "eventbrite":
		return "`[Eventbrite]`"
	case "devevents":
		return "`[Dev.events]`"
	default:
		return "`[" + source + "]`"
	}
}

func generateTelegramMessage(events []interface{}) string {
	now := time.Now()
	dateStr := now.Format("Monday, January 2, 2006")
	
	message := fmt.Sprintf("🚀 *Winnipeg Tech Events - %s*\n\n", dateStr)
	
	// Group events by time period
	groups := groupEventsByTime(events)
	
	for period, periodEvents := range groups {
		if len(periodEvents) > 0 {
			message += fmt.Sprintf("*%s:*\n", period)
			for _, event := range periodEvents {
				if eventMap, ok := event.(map[string]interface{}); ok {
					name := getString(eventMap, "name")
					url := getString(eventMap, "url")
					startTime := getString(eventMap, "start_time")
					venue := getString(eventMap, "venue")
					price := getString(eventMap, "price")
					
					message += fmt.Sprintf("• %s\n", name)
					
					if startTime != "" {
						if t, err := time.Parse(time.RFC3339, startTime); err == nil {
							timeStr := t.Format("Jan 2 at 3:04 PM")
							message += fmt.Sprintf("  📅 %s\n", timeStr)
						}
					}
					
					if venue != "" {
						message += fmt.Sprintf("  📍 %s\n", venue)
					}
					
					if price != "" && price != "Free" {
						message += fmt.Sprintf("  💰 %s\n", price)
					}
					
					if url != "" {
						message += fmt.Sprintf("  🔗 [View Event](%s)\n", url)
					}
					
					message += "\n"
				}
			}
		}
	}
	
	message += "\n_Shared via Winnipeg Tech Events Tracker_"
	
	return message
}

func groupEventsByTimeFromModels(events []models.Event) map[string][]models.Event {
	now := time.Now()
	groups := map[string][]models.Event{
		"Today":     {},
		"This Week": {},
		"Next Week": {},
		"Later":     {},
	}
	
	for _, event := range events {
		if isSameDay(event.StartTime, now) {
			groups["Today"] = append(groups["Today"], event)
		} else if isThisWeek(event.StartTime) {
			groups["This Week"] = append(groups["This Week"], event)
		} else if isNextWeek(event.StartTime) {
			groups["Next Week"] = append(groups["Next Week"], event)
		} else {
			groups["Later"] = append(groups["Later"], event)
		}
	}
	
	// Remove empty groups
	for key, group := range groups {
		if len(group) == 0 {
			delete(groups, key)
		}
	}
	
	return groups
}

func groupEventsByTime(events []interface{}) map[string][]interface{} {
	now := time.Now()
	groups := map[string][]interface{}{
		"Today":     {},
		"This Week": {},
		"Next Week": {},
		"Later":     {},
	}
	
	for _, event := range events {
		if eventMap, ok := event.(map[string]interface{}); ok {
			if startTimeStr, ok := eventMap["start_time"].(string); ok {
				if startTime, err := time.Parse(time.RFC3339, startTimeStr); err == nil {
					if isSameDay(startTime, now) {
						groups["Today"] = append(groups["Today"], event)
					} else if isThisWeek(startTime) {
						groups["This Week"] = append(groups["This Week"], event)
					} else if isNextWeek(startTime) {
						groups["Next Week"] = append(groups["Next Week"], event)
					} else {
						groups["Later"] = append(groups["Later"], event)
					}
				}
			}
		}
	}
	
	// Remove empty groups
	for key, group := range groups {
		if len(group) == 0 {
			delete(groups, key)
		}
	}
	
	return groups
}

func isSameDay(date1, date2 time.Time) bool {
	return date1.Year() == date2.Year() && date1.YearDay() == date2.YearDay()
}

func isThisWeek(date time.Time) bool {
	now := time.Now()
	startOfWeek := now.AddDate(0, 0, -int(now.Weekday()))
	endOfWeek := startOfWeek.AddDate(0, 0, 6)
	return date.After(startOfWeek) && date.Before(endOfWeek.Add(24*time.Hour))
}

func isNextWeek(date time.Time) bool {
	now := time.Now()
	startOfNextWeek := now.AddDate(0, 0, 7-int(now.Weekday()))
	endOfNextWeek := startOfNextWeek.AddDate(0, 0, 6)
	return date.After(startOfNextWeek) && date.Before(endOfNextWeek.Add(24*time.Hour))
}

func getString(m map[string]interface{}, key string) string {
	if val, ok := m[key].(string); ok {
		return val
	}
	return ""
}

func logResult(result *SchedulerResult) {
	log.Printf("📊 Scheduler Result: Success=%t, Events=%d, MessageSent=%t", 
		result.Success, result.EventsCount, result.MessageSent)
	
	for _, logMsg := range result.Logs {
		log.Printf("📝 %s", logMsg)
	}
	
	if result.Error != "" {
		log.Printf("❌ Error: %s", result.Error)
	}
	
	// Output JSON result for external processing
	if jsonResult, err := json.MarshalIndent(result, "", "  "); err == nil {
		log.Printf("📋 JSON Result: %s", string(jsonResult))
	}
}
