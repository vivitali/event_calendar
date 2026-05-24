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
	scraper := factory.CreateDefaultService()
	sender := telegram.NewService(cfg.BotToken)
	return jobs.RunEventsDigest(jobs.EventsDeps{
		Scraper: scraper,
		Sender:  sender,
		Now:     time.Now(),
	}, cfg, nil)
}

func runPoll(cfg config.Config) jobs.Result {
	poller := telegram.NewService(cfg.PollBotToken)
	return jobs.RunMonthlyPoll(jobs.PollDeps{Poller: poller}, cfg)
}

func recoverPanic() {
	if r := recover(); r != nil {
		log.Printf("PANIC: %v", r)
	}
}
