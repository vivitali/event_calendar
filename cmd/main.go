package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"event_calendar/pkg/aggregator"
	"event_calendar/pkg/devevents"
	"event_calendar/pkg/eventbrite"
	"event_calendar/pkg/meetup"
)

// RequestParams defines incoming parameters
type RequestParams struct {
	City       string   `json:"city"`
	Categories []string `json:"categories"`
}

func main() {
	// Serve static files
	http.Handle("/", http.FileServer(http.Dir("./web")))
	
	// API endpoints
	http.HandleFunc("/api/events", aggregateEventsHandler)
	http.HandleFunc("/api/health", healthHandler)
	
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}
	
	log.Printf("Server starting on port %s", port)
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
	// Set CORS headers
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
	w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
	w.Header().Set("Content-Type", "application/json")

	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusOK)
		return
	}

	params, err := parseRequestParams(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Default period
	periodDaysStr := os.Getenv("PERIOD_DAYS")
	periodDays, err := strconv.Atoi(periodDaysStr)
	if err != nil || periodDays <= 0 {
		periodDays = 30
	}

	period := time.Duration(periodDays) * 24 * time.Hour

	// Initialize scrapers
	meetupScraper := meetup.NewScraper()
	eventbriteScraper := eventbrite.NewScraper()
	devEventsScraper := devevents.NewScraper()

	agg := aggregator.NewAggregator(meetupScraper, eventbriteScraper, devEventsScraper)

	// Collect events
	var allEvents []interface{}
	for _, category := range params.Categories {
		events, err := agg.AggregateEvents(params.City, category, period)
		if err != nil {
			log.Printf("Aggregation error for category %s: %v", category, err)
			continue
		}
		allEvents = append(allEvents, events)
	}

	// Response
	json.NewEncoder(w).Encode(allEvents)
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
