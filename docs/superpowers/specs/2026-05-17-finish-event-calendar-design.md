# Finish Event Calendar — Design Spec

**Date:** 2026-05-17
**Status:** Draft
**Goal:** Replace GitHub Actions scheduling with AWS Lambda + EventBridge, remove half-built/dead code, deduplicate helpers. Result: a simple, reliable production system that runs without 60-day inactivity timeouts and without unused subsystems.

---

## 1. Background

The project scrapes Winnipeg tech events from Meetup, Eventbrite, and Dev.events, then posts digests to Telegram. Current scheduling uses GitHub Actions, which **disables scheduled workflows after 60 days of repo inactivity**. The Python Lambda exists but only returns sample data — it never calls the real Go scrapers. A Telegram callback voting feature exists but uses in-memory storage and is not deployable. Helpers for date grouping and Telegram message formatting are duplicated across three files.

## 2. Goals

1. Scheduler runs reliably without depending on repo activity.
2. Real Go scrapers execute on the scheduled run (not sample data).
3. Codebase contains only working, deployable code.
4. Duplicated helpers extracted to single source of truth.
5. AWS-native deployment using already-configured credentials.

## 3. Non-goals

- Voting / RSVP feature (deleted; can be re-added later with proper storage).
- Migrating to a non-AWS platform.
- Frontend changes beyond what is required by backend refactors.
- Adding new scraper sources.
- Persistent storage for events (digests are stateless).

## 4. Architecture

```
EventBridge Scheduler (weekly Mon 14:00 UTC)  ──┐
                                                ├──→ Lambda (Go, provided.al2023)
EventBridge Scheduler (monthly 20th 14:00 UTC) ─┘         │
                                                          ├──→ scrape Meetup/Eventbrite/Dev.events
                                                          └──→ Telegram Bot API (digest or poll)
```

- **One Lambda function**, routed by event payload `{"action": "events"}` or `{"action": "poll"}`.
- **Two EventBridge schedule rules**, each invoking the Lambda with the appropriate payload.
- **No webhook**, **no API Gateway** (no inbound HTTP needed).
- **CloudWatch Logs** captures stdout/stderr automatically.
- **Telegram error alerts** sent from inside the Lambda when scraping or sending fails.

## 5. Code Layout (target)

```
cmd/
  lambda/main.go        # Lambda entry: parses event, dispatches to job
  cli/main.go           # Local CLI: runs the same jobs from terminal for testing
  server/main.go        # Local web UI server (renamed from cmd/main.go)
internal/
  config/config.go      # Env var loading, single source of truth
  jobs/
    events.go           # RunEventsDigest(cfg) -> Result
    poll.go             # RunMonthlyPoll(cfg) -> Result
  digest/
    format.go           # FormatEventsMessage(events []models.Event, now time.Time) string
  timeutil/
    groups.go           # IsSameDay, IsThisWeek, IsNextWeek, GroupByPeriod
internal/models/
  event.go              # unchanged
pkg/                    # unchanged: scraping/, telegram/, devevents/, meetup/, eventbrite/, aggregator/
deploy/
  deploy.sh             # AWS deploy: build Go for Lambda, zip, upload, set env, create/update EventBridge rules
web/                    # unchanged
```

### 5.1 What each new/changed unit does

- `cmd/lambda/main.go`: Receives `events.CloudWatchEvent` or generic `map[string]any`. Reads `action` field. Calls `jobs.RunEventsDigest` or `jobs.RunMonthlyPoll`. Returns `{success, eventsCount, messageSent}`.
- `cmd/cli/main.go`: Replaces `cmd/scheduler/main.go` + `cmd/poll-scheduler/main.go`. Accepts flag `-action=events|poll` for manual local trigger.
- `cmd/server/main.go`: Identical to current `cmd/main.go`, just moved. Run locally for the web UI.
- `internal/config/config.go`: One `Load()` function returning `Config{BotToken, ChatID, PollBotToken, PollChatID, TestMode, City, Categories, PeriodDays}`. Poll bot/chat default to main values if unset.
- `internal/jobs/events.go`: Owns scrape + filter + format + send sequence. Returns a `Result` struct. No `os.Exit`, no `log.Fatal`.
- `internal/jobs/poll.go`: Owns "send monthly poll" sequence. Does **not** include the `is20thOfMonth()` check — EventBridge handles scheduling; the job always runs when invoked.
- `internal/digest/format.go`: One canonical `FormatEventsMessage`. Replaces the three near-duplicate implementations in `pkg/telegram/service.go`, `cmd/scheduler/main.go`.
- `internal/timeutil/groups.go`: Exported `IsSameDay`, `IsThisWeek`, `IsNextWeek`, `GroupByPeriod`. Replaces the duplicates in `pkg/telegram/service.go`, `pkg/aggregator/aggregator.go`, `cmd/scheduler/main.go`.

## 6. Lambda Deployment

### 6.1 Build

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -tags lambda.norpc -o bootstrap cmd/lambda/main.go
zip lambda.zip bootstrap
```

Runtime: `provided.al2023` (custom runtime for Go, replaces deprecated `go1.x`).

### 6.2 IAM

Lambda execution role needs only `AWSLambdaBasicExecutionRole` (CloudWatch Logs). No VPC, no other AWS services accessed.

### 6.3 EventBridge rules

| Rule | Schedule (cron UTC) | Payload | Description |
|------|---------------------|---------|-------------|
| `winnipeg-events-weekly` | `0 14 ? * MON *` | `{"action":"events"}` | Mondays 14:00 UTC (9 AM CST) |
| `winnipeg-poll-monthly` | `0 14 20 * ? *` | `{"action":"poll"}` | 20th of each month, 14:00 UTC |

### 6.4 Environment variables on Lambda

```
TELEGRAM_BOT_TOKEN=...
TELEGRAM_CHAT_ID=...
TELEGRAM_POLL_BOT_TOKEN=...   # optional, falls back to TELEGRAM_BOT_TOKEN
TELEGRAM_POLL_CHAT_ID=...     # optional, falls back to TELEGRAM_CHAT_ID
CITY=Winnipeg
CATEGORIES=tech
PERIOD_DAYS=30
TEST_MODE=false
```

### 6.5 deploy.sh

Idempotent script that:
1. Builds Go binary for Lambda.
2. Creates/updates the Lambda function.
3. Creates/updates the IAM role if missing.
4. Creates/updates both EventBridge rules and targets.
5. Sets environment variables (read from local `.env` file or shell env).
6. Prints next steps and how to manually invoke.

## 7. Data Flow

### 7.1 Events digest (weekly)

```
EventBridge → Lambda.handler({"action":"events"})
  → config.Load()
  → scraping.NewScrapingServiceFactory().CreateDefaultService().ScrapeEvents(...)
  → devevents.NewScraper().GetEvents(...)            # appended
  → aggregator.FilterFutureEvents(events)
  → digest.FormatEventsMessage(events, time.Now())
  → telegram.Service.SendMessage(chatID, message)    # plain message, no keyboard
  → return Result
```

### 7.2 Monthly poll

```
EventBridge → Lambda.handler({"action":"poll"})
  → config.Load()
  → telegram.NewService(pollBotToken).SendMonthlyMeetupPoll(pollChatID)
  → return Result
```

## 8. Error Handling

- Each scraper failure: logged, skipped, others continue (existing behavior in `aggregator.Aggregator`).
- Telegram send failure: logged, Lambda returns `success=false`. CloudWatch alarm (out of scope; rely on Telegram self-alert).
- Lambda function failure (panic, timeout): inside `cmd/lambda/main.go` `defer recover()` catches panics and sends a Telegram alert to `TELEGRAM_CHAT_ID` via direct HTTP call before returning the error.
- Empty results (no events): logged, Lambda returns `success=true, messageSent=false, eventsCount=0`. No message posted.
- Telegram message length > 4096: existing truncation logic preserved in `digest.FormatEventsMessage`.

## 9. Code to Delete

- `cmd/scheduler/main.go` → replaced by `cmd/cli/main.go` + `internal/jobs/events.go`
- `cmd/poll-scheduler/main.go` → replaced by `cmd/cli/main.go` + `internal/jobs/poll.go`
- `cmd/webhook/main.go` → no callback voting
- `pkg/telegram/callback_handler.go` → no callback voting
- `pkg/telegram/service.go`: remove `CreateVoteKeyboard`, `CreateEventVoteKeyboard`, `SendMessageWithKeyboard` and related vote-button types. Keep `SendMessage`, `SendPoll`, `SendMonthlyMeetupPoll`, `SendAlert`, `TestConnection`.
- `lambda/handler.py`, `lambda/requirements.txt`, `lambda/deploy.sh` → replaced by Go Lambda in `cmd/lambda/` + `deploy/deploy.sh`
- `.github/workflows/winnipeg-tech-events.yml` → replaced by EventBridge
- `.github/workflows/monthly-poll.yml` → replaced by EventBridge
- `.github/workflows/deploy.yml` → review; if it only deploys to GH Actions infra, delete
- `test_app.go`, `test_poll.sh`, `test_poll_20th.sh`, `test_scheduler.sh`, `test_github_actions.sh`, `setup_github_secrets.sh`, `validate.sh`, `main`, `scheduler` (root binaries) → ad-hoc test scripts and stray build artifacts; delete after extracting any logic still needed
- `GITHUB_ACTIONS_SETUP.md`, `SEPARATE_BOTS_SETUP.md`, `POLL_INTEGRATION.md` → obsolete; replace with single section in `README.md` covering AWS deploy

## 10. README Updates

Replace deployment sections with one focused on AWS:
- Prerequisites: AWS CLI configured (`aws configure sso` or `aws login`).
- Single command: `./deploy/deploy.sh`.
- Environment variables documented.
- Local dev: `go run cmd/server/main.go` for the web UI, `go run cmd/cli/main.go -action=events` for a one-off digest test.

## 11. Testing

Limited surface for unit tests; focus where logic is non-trivial:

- `internal/timeutil/groups_test.go`: table-driven test for `IsSameDay`, `IsThisWeek`, `IsNextWeek`, `GroupByPeriod` with fixed `now` time.
- `internal/digest/format_test.go`: verifies message structure, empty-events case, markdown escaping, ordering by time bucket.
- `internal/jobs/events_test.go`: mock provider + mock Telegram sender, verify scrape → filter → format → send sequence and result fields.
- `internal/jobs/poll_test.go`: mock Telegram sender, verify poll request.
- Lambda handler not tested directly. Manual smoke test via `aws lambda invoke` after deploy.

Existing scraper packages have no tests; this spec does not add them (out of scope — they work in production).

## 12. Configuration Defaults

Match current behavior:
- `CITY=Winnipeg`, `CATEGORIES=tech`, `PERIOD_DAYS=30`, `TEST_MODE=false`
- Poll bot/chat fall back to main bot/chat when unset (one-bot setup works out of the box).
- `TEST_MODE=true` causes jobs to format the message and log it but skip the actual Telegram send.

## 13. Risks and Open Questions

- **Two-bot fallback semantics:** if main bot is in the events chat and user has not set poll-specific envs, the poll will post to the events chat. Documented in README; user can opt in to a separate poll bot by setting both `TELEGRAM_POLL_BOT_TOKEN` and `TELEGRAM_POLL_CHAT_ID`.
- **Eventbrite/Meetup scraper fragility:** these are HTML scrapers that may break when the target sites change. Out of scope here; existing fallback to sample data is preserved.
- **`provided.al2023` runtime:** requires the binary to be named `bootstrap` and packaged at the zip root. Build command handles this.
- **CloudWatch retention:** default is forever. `deploy.sh` should set retention to 14 days to keep costs at zero.

## 14. Success Criteria

1. `./deploy/deploy.sh` runs end-to-end, idempotently, with no manual AWS console steps.
2. `aws lambda invoke --function-name winnipeg-tech-events --payload '{"action":"events"}' out.json` produces a Telegram digest in the configured chat.
3. `aws lambda invoke --function-name winnipeg-tech-events --payload '{"action":"poll"}' out.json` produces a Telegram poll in the configured chat.
4. Both EventBridge rules visible in AWS console with the next-run time set correctly.
5. `go build ./...` succeeds. `go vet ./...` clean. Unit tests pass.
6. Repo contains no Python Lambda, no GH workflow files, no callback voting code.
7. No duplicated date helpers or message builders remain.
