package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/GoogleCloudPlatform/functions-framework-go/functions"
	"github.com/joho/godotenv"

	"event_calendar/pkg/aggregator"
	"event_calendar/pkg/eventbrite"
)

// RequestParams defines incoming parameters
type RequestParams struct {
	City       string   `json:"city"`
	Categories []string `json:"categories"`
}

// Handler entry-point
func init() {
	functions.HTTP("AggregateEvents", aggregateEventsHandler)
}

func aggregateEventsHandler(w http.ResponseWriter, r *http.Request) {
	// Load .env
	if err := godotenv.Load(); err != nil {
		log.Println("Could not load .env, assuming env vars are already set.")
	}

	params, err := parseRequestParams(r)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}

	// Default period from YAML (or use constant if not in config)
	periodDaysStr := eventbrite.NewClient(os.Getenv("PERIOD_DAYS"))
	periodDays, err := strconv.Atoi(periodDaysStr)
	if err != nil || periodDays <= 0 {
		// If it's invalid (non-numeric or <= 0), use the default
		periodDays = 30
	}

	period := time.Duration(periodDays) * 24 * time.Hour

	// Initialize API clients
	ebClient := eventbrite.NewClient(os.Getenv("EVENTBRITE_API_KEY"))

	agg := aggregator.NewAggregator(ebClient)

	// Collect events
	var allEvents []interface{}
	for _, category := range params.Categories {
		events, err := agg.AggregateEvents(params.City, category, period)
		if err != nil {
			log.Println("Aggregation error:", err)
			continue
		}
		allEvents = append(allEvents, events)
	}

	// Response
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(allEvents)
}

// parseRequestParams extracts parameters from HTTP request
func parseRequestParams(r *http.Request) (*RequestParams, error) {
	city := r.URL.Query().Get("city")
	if city == "" {
		return nil, http.ErrMissingFile
	}

	categoriesParam := r.URL.Query().Get("categories")
	if categoriesParam == "" {
		return nil, http.ErrMissingFile
	}

	categories := strings.Split(categoriesParam, ",")

	return &RequestParams{
		City:       city,
		Categories: categories,
	}, nil
}
