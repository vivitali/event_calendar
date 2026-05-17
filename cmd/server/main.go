package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"event_calendar/pkg/devevents"
	"event_calendar/pkg/scraping"
)

// RequestParams defines incoming parameters
type RequestParams struct {
	City       string   `json:"city"`
	Categories []string `json:"categories"`
}

// Global scraping service instance
var scrapingService *scraping.ScrapingService

func main() {
	log.Printf("🚀 Starting Event Calendar Application...")
	
	// Initialize scraping service
	log.Printf("🔧 Initializing scraping service...")
	factory := scraping.NewScrapingServiceFactory()
	scrapingService = factory.CreateDefaultService()
	
	// Log service initialization
	scrapers := scrapingService.GetRegisteredScrapers()
	healthStatus := scrapingService.GetHealthStatus()
	log.Printf("✅ Scraping service initialized with %d scrapers: %v", len(scrapers), scrapers)
	log.Printf("📊 Scraper health status: %v", healthStatus)
	
	// Serve static files
	http.Handle("/", http.FileServer(http.Dir("./web")))
	
	// API endpoints
	http.HandleFunc("/api/events", aggregateEventsHandler)
	http.HandleFunc("/api/health", healthHandler)
	http.HandleFunc("/api/scrapers/health", scrapersHealthHandler)
	http.HandleFunc("/api/scrapers", scrapersInfoHandler)
	
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	log.Printf("🌐 Server starting on port %s", port)
	log.Printf("📡 Available endpoints:")
	log.Printf("   - GET /api/events - Scrape events from all sources")
	log.Printf("   - GET /api/health - Application health check")
	log.Printf("   - GET /api/scrapers/health - Scraper health status")
	log.Printf("   - GET /api/scrapers - Scraper information")
	log.Printf("   - GET / - Static web interface")
	
	log.Fatal(http.ListenAndServe(":"+port, nil))
}

func healthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"status": "healthy",
		"time":   time.Now().Format(time.RFC3339),
	})
}

func aggregateEventsHandler(w http.ResponseWriter, r *http.Request) {
	log.Printf("📡 [API] Received request to /api/events from %s", r.RemoteAddr)
	
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		log.Printf("📡 [API] Handling OPTIONS request")
		w.WriteHeader(http.StatusOK)
		return
	}

	params, err := parseRequestParams(r)
	if err != nil {
		log.Printf("❌ [API] Error parsing request parameters: %v", err)
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	log.Printf("📋 [API] Request parameters - City: %s, Categories: %v", params.City, params.Categories)

	// Default period
	periodDaysStr := os.Getenv("PERIOD_DAYS")
	periodDays, err := strconv.Atoi(periodDaysStr)
	if err != nil || periodDays <= 0 {
		periodDays = 30
	}

	period := time.Duration(periodDays) * 24 * time.Hour
	log.Printf("⏰ [API] Scraping period: %d days (%v)", periodDays, period)

	// Use the new scraping service
	log.Printf("🔄 [API] Starting scraping process...")
	startTime := time.Now()
	allEvents, err := scrapingService.ScrapeEvents(params.City, params.Categories[0], period)
	scrapingDuration := time.Since(startTime)
	
	if err != nil {
		log.Printf("❌ [API] Scraping error after %v: %v", scrapingDuration, err)
		http.Error(w, "Failed to scrape events", http.StatusInternalServerError)
		return
	}

	log.Printf("✅ [API] Scraping service completed in %v, found %d events", scrapingDuration, len(allEvents))

	// Also include devevents scraper for backward compatibility
	log.Printf("🔄 [API] Fetching events from devevents scraper...")
	devEventsScraper := devevents.NewScraper()
	devEvents, err := devEventsScraper.GetEvents(params.City, params.Categories[0], period)
	if err != nil {
		log.Printf("⚠️  [API] DevEvents scraping error: %v", err)
	} else {
		log.Printf("✅ [API] DevEvents scraper found %d events", len(devEvents))
		allEvents = append(allEvents, devEvents...)
	}

	log.Printf("📊 [API] Total events to return: %d", len(allEvents))
	
	// Log sample events for debugging
	if len(allEvents) > 0 {
		log.Printf("📋 [API] Sample events:")
		for i, event := range allEvents {
			if i >= 3 { // Log only first 3 events
				break
			}
			log.Printf("   %d. %s (%s) - %s", i+1, event.Name, event.Source, event.StartTime.Format("2006-01-02 15:04"))
		}
	}

	// Response
	log.Printf("📤 [API] Sending response with %d events", len(allEvents))
	json.NewEncoder(w).Encode(allEvents)
}

// scrapersHealthHandler returns the health status of all scrapers
func scrapersHealthHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	healthStatus := scrapingService.GetHealthStatus()
	json.NewEncoder(w).Encode(map[string]interface{}{
		"scrapers": healthStatus,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// scrapersInfoHandler returns information about registered scrapers
func scrapersInfoHandler(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Access-Control-Allow-Origin", "*")
	
	scrapers := scrapingService.GetRegisteredScrapers()
	healthStatus := scrapingService.GetHealthStatus()
	
	scraperInfo := make([]map[string]interface{}, len(scrapers))
	for i, name := range scrapers {
		scraperInfo[i] = map[string]interface{}{
			"name":   name,
			"healthy": healthStatus[name],
		}
	}
	
	json.NewEncoder(w).Encode(map[string]interface{}{
		"scrapers": scraperInfo,
		"count":    len(scrapers),
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

// parseRequestParams extracts parameters from HTTP request
func parseRequestParams(r *http.Request) (*RequestParams, error) {
	city := r.URL.Query().Get("city")
	if city == "" {
		city = "Winnipeg" // Default to Winnipeg
	}

	categoriesParam := r.URL.Query().Get("categories")
	if categoriesParam == "" {
		categoriesParam = "tech" // Default category
	}

	categories := strings.Split(categoriesParam, ",")

	return &RequestParams{
		City:       city,
		Categories: categories,
	}, nil
}
