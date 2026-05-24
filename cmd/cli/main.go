// Command cli runs the same jobs the Lambda runs, from your terminal.
// Useful for local testing without invoking Lambda.
//
// Usage:
//
//	go run ./cmd/cli -action=events
//	go run ./cmd/cli -action=poll
package main

import (
	"encoding/json"
	"flag"
	"log"
	"os"
	"time"

	"event_calendar/internal/config"
	"event_calendar/internal/jobs"
	"event_calendar/pkg/scraping"
	"event_calendar/pkg/telegram"
)

func main() {
	action := flag.String("action", "events", "job to run: events or poll")
	flag.Parse()

	cfg := config.Load()
	var result jobs.Result
	switch *action {
	case "events":
		result = runEvents(cfg)
	case "poll":
		result = runPoll(cfg)
	default:
		log.Fatalf("unknown action %q (want events or poll)", *action)
	}

	out, _ := json.MarshalIndent(result, "", "  ")
	log.Printf("result:\n%s", out)
	if !result.Success {
		os.Exit(1)
	}
}

func runEvents(cfg config.Config) jobs.Result {
	factory := scraping.NewScrapingServiceFactory()
	scraper := factory.CreateDefaultService()
	sender := telegram.NewService(cfg.BotToken)
	return jobs.RunEventsDigest(jobs.EventsDeps{Scraper: scraper, Sender: sender, Now: time.Now()}, cfg, nil)
}

func runPoll(cfg config.Config) jobs.Result {
	poller := telegram.NewService(cfg.PollBotToken)
	return jobs.RunMonthlyPoll(jobs.PollDeps{Poller: poller}, cfg)
}
