# Winnipeg Tech Events Scraper & Telegram Sharing Web App

Discovers, aggregates, and shares technology events happening in Winnipeg, Manitoba. Scrapes from multiple sources, posts weekly digests to Telegram, and runs a monthly meetup-day poll.

## Features

### Multi-Source Event Aggregation
- **Meetup.com** — tech events from Winnipeg area
- **Eventbrite** — tech events with smart datetime parsing
- **Dev.events** — developer events in Winnipeg/Manitoba

### Telegram Integration
- Weekly events digest posted automatically
- Monthly meetup day-of-week poll on the 20th
- Optional second bot for the poll (falls back to main bot if not configured)

### Modern Web UI (local)
- Responsive design with dark mode
- Filter by date range, source, search
- Grouping by Today / This Week / Next Week / Later

### Scheduling
- AWS Lambda + EventBridge runs the jobs on schedule
- No GitHub Actions (avoids the 60-day-inactivity workflow timeout)

## Quick Start

### Prerequisites
- Go 1.24+
- AWS CLI v2 (for deployment) configured via `aws login` or `aws configure sso`
- Telegram bot token + chat ID

### Run locally

```bash
go mod tidy

# Web UI on http://localhost:8080
go run ./cmd/server

# Trigger an events digest manually (test mode, no Telegram send):
TEST_MODE=true \
TELEGRAM_BOT_TOKEN=xxx TELEGRAM_CHAT_ID=yyy \
go run ./cmd/cli -action=events

# Trigger the monthly poll manually:
TEST_MODE=true \
TELEGRAM_POLL_BOT_TOKEN=xxx TELEGRAM_POLL_CHAT_ID=yyy \
go run ./cmd/cli -action=poll
```

### Deploy to AWS Lambda

The Lambda is invoked on a schedule by AWS EventBridge — no GitHub Actions, no
60-day inactivity timeout. Two schedules ship out of the box:

- `winnipeg-events-weekly` — Mondays at 14:00 UTC (9 AM CST)
- `winnipeg-poll-monthly` — 20th of each month at 14:00 UTC

```bash
# 1. Configure AWS credentials once
aws login                      # or: aws configure sso

# 2. Put your secrets in deploy/.env (file is gitignored)
cat > deploy/.env <<'EOF'
TELEGRAM_BOT_TOKEN=...
TELEGRAM_CHAT_ID=...
# Optional: separate bot for the monthly poll
TELEGRAM_POLL_BOT_TOKEN=...
TELEGRAM_POLL_CHAT_ID=...
EOF

# 3. Deploy
./deploy/deploy.sh

# 4. Smoke test
aws lambda invoke --function-name winnipeg-tech-events \
    --payload '{"action":"events"}' \
    --cli-binary-format raw-in-base64-out out.json && cat out.json

# 5. Tail logs
aws logs tail /aws/lambda/winnipeg-tech-events --follow
```

## Configuration

| Env var | Default | Purpose |
|---|---|---|
| `TELEGRAM_BOT_TOKEN` | — | Bot that posts the events digest |
| `TELEGRAM_CHAT_ID` | — | Chat that receives the events digest |
| `TELEGRAM_POLL_BOT_TOKEN` | falls back to `TELEGRAM_BOT_TOKEN` | Bot that posts the monthly poll |
| `TELEGRAM_POLL_CHAT_ID` | falls back to `TELEGRAM_CHAT_ID` | Chat that receives the monthly poll |
| `CITY` | `Winnipeg` | City passed to scrapers |
| `CATEGORIES` | `tech` | Category passed to scrapers |
| `PERIOD_DAYS` | `30` | How many days ahead to scrape |
| `TEST_MODE` | `false` | When `true`, format the message but don't send |
| `PORT` | `8080` | Web UI server port (local only) |

### Telegram Bot Setup

1. Message [@BotFather](https://t.me/botfather) → `/newbot` → save the token.
2. Add the bot to your group. Send any message in the group.
3. Visit `https://api.telegram.org/bot<TOKEN>/getUpdates` and copy the chat ID.

## Architecture

```
EventBridge Scheduler (weekly Mon 14:00 UTC) ─┐
                                              ├──→ Lambda (Go, provided.al2023)
EventBridge Scheduler (monthly 20th 14:00 UTC)┘         │
                                                        ├──→ scrape Meetup/Eventbrite/Dev.events
                                                        └──→ Telegram Bot API (digest or poll)
```

### Code layout

```
cmd/
  lambda/       # AWS Lambda entry point
  cli/          # Local CLI runner
  server/       # Local web UI server
internal/
  config/       # Env loading
  digest/       # Telegram message formatting
  jobs/         # Reusable events-digest + monthly-poll jobs
  timeutil/     # Date grouping helpers
  models/       # Event struct
pkg/
  scraping/     # Multi-source scraping service
  meetup/       eventbrite/   devevents/   # individual scrapers
  telegram/     # Telegram Bot API client
  aggregator/   # Result merging & deduping
deploy/
  deploy.sh     # Idempotent AWS deploy
web/            # Local web UI assets
```

## Testing

```bash
go test ./...     # unit tests for timeutil, digest, config, jobs
go vet ./...
```

## License

MIT — see [LICENSE](LICENSE).
