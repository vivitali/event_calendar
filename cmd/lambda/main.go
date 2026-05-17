// Command lambda is the AWS Lambda entry point. It dispatches to one of the
// job functions based on the "action" field in the invocation event.
package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/aws/aws-lambda-go/lambda"

	"event_calendar/internal/config"
	"event_calendar/internal/jobs"
	"event_calendar/internal/models"
	"event_calendar/pkg/devevents"
	"event_calendar/pkg/scraping"
	"event_calendar/pkg/telegram"
)

// Event is the invocation payload. EventBridge rules set "action".
type Event struct {
	Action string `json:"action"`
}

func main() {
	lambda.Start(handle)
}

func handle(ctx context.Context, evt Event) (jobs.Result, error) {
	defer recoverPanic()

	cfg := config.Load()
	log.Printf("lambda invoked: action=%q city=%s test_mode=%t", evt.Action, cfg.City, cfg.TestMode)

	switch evt.Action {
	case "events", "":
		return runEvents(cfg), nil
	case "poll":
		return runPoll(cfg), nil
	default:
		return jobs.Result{Error: fmt.Sprintf("unknown action %q", evt.Action)}, nil
	}
}

func runEvents(cfg config.Config) jobs.Result {
	factory := scraping.NewScrapingServiceFactory()
	scraper := combinedScraper{
		main: factory.CreateDefaultService(),
		dev:  devevents.NewScraper(),
	}
	sender := telegram.NewService(cfg.BotToken)
	return jobs.RunEventsDigest(jobs.EventsDeps{
		Scraper: scraper,
		Sender:  sender,
		Now:     time.Now(),
	}, cfg)
}

func runPoll(cfg config.Config) jobs.Result {
	poller := telegram.NewService(cfg.PollBotToken)
	return jobs.RunMonthlyPoll(jobs.PollDeps{Poller: poller}, cfg)
}

// combinedScraper merges the main scraping service with the devevents scraper,
// preserving the previous production behavior.
type combinedScraper struct {
	main *scraping.ScrapingService
	dev  *devevents.Scraper
}

func (c combinedScraper) ScrapeEvents(city, category string, period time.Duration) ([]models.Event, error) {
	main, mainErr := c.main.ScrapeEvents(city, category, period)
	if mainErr != nil {
		log.Printf("main scraper error: %v", mainErr)
	}
	dev, devErr := c.dev.GetEvents(city, category, period)
	if devErr != nil {
		log.Printf("devevents error: %v", devErr)
	}
	return append(main, dev...), nil
}

func recoverPanic() {
	if r := recover(); r != nil {
		log.Printf("PANIC: %v", r)
	}
}
