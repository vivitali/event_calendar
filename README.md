# Winnipeg Tech Events Scraper & Telegram Sharing Web App

Discovers, aggregates, and shares technology events happening in Winnipeg, Manitoba. Scrapes from multiple sources, posts weekly digests to Telegram, and runs a monthly meetup-day poll.

## Features

### Multi-Source Event Aggregation
- **Meetup.com** — schema.org JSON-LD parsing
- **Eventbrite** — `window.__SERVER_DATA__` SSR JSON, with same-organizer-same-day dedupe

### Telegram Integration
- Weekly events digest posted automatically
- Monthly meetup day-of-week poll on the 20th
- Optional second bot for the poll (falls back to main bot if not configured)

### Modern Web UI (local)
- Responsive design with dark mode
- Filter by date range, source, search
- Grouping by Today / This Week / Next Week / Later

### Scheduling
- AWS Lambda + EventBridge runs the jobs on schedule (independent of GitHub
  Actions, so it survives long periods of repo inactivity)

### Deploy
- Manual: `./deploy/deploy.sh`
- CI: `.github/workflows/deploy.yml` on push to `main` (OIDC, no long-lived AWS keys)

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

You need at minimum one bot + chat ID for the weekly events digest. You can
optionally add a **second** bot for the monthly poll (otherwise the main bot
posts both).

#### 1. Create the bot

1. In Telegram, open a chat with [@BotFather](https://t.me/botfather).
2. Send `/newbot` and follow the prompts. Pick:
   - A **name** (display, e.g. `Winnipeg Tech Events`)
   - A **username** ending in `bot` (e.g. `winnipeg_tech_events_bot`)
3. BotFather replies with a token like `8123456789:AAH...`. **This is `TELEGRAM_BOT_TOKEN`.** Save it.
4. (Optional) Send `/setdescription`, `/setabouttext`, `/setuserpic` to fill out the bot profile.

#### 2. Get the chat ID

For a group chat (recommended):

1. Add the bot to the target group as a member.
2. In Telegram, send any message in that group (e.g. `hi`).
3. Open the following URL in a browser, replacing `<TOKEN>`:
   ```
   https://api.telegram.org/bot<TOKEN>/getUpdates
   ```
4. Find the most recent update's `message.chat.id`. For groups this is a
   negative integer like `-1001234567890`. **This is `TELEGRAM_CHAT_ID`.**

For your own DM with the bot, click `/start` first, then the chat ID will be
your positive user ID (e.g. `123456789`).

#### 3. (Optional) Second bot for the poll

If you want the monthly meetup poll to come from a different bot:

1. Repeat step 1 with a new BotFather `/newbot` → `TELEGRAM_POLL_BOT_TOKEN`.
2. Add that bot to a chat (can be the same group) → `TELEGRAM_POLL_CHAT_ID`.

If you skip this, both jobs use `TELEGRAM_BOT_TOKEN` / `TELEGRAM_CHAT_ID`.

## Where secrets live

| Variable | `deploy/.env` (local) | GitHub Secrets (CI) | Lambda env |
|---|---|---|---|
| `TELEGRAM_BOT_TOKEN` | ✓ | ✓ | set by deploy |
| `TELEGRAM_CHAT_ID` | ✓ | ✓ | set by deploy |
| `TELEGRAM_POLL_BOT_TOKEN` (optional) | ✓ | ✓ | set by deploy |
| `TELEGRAM_POLL_CHAT_ID` (optional) | ✓ | ✓ | set by deploy |
| `AWS_ACCOUNT_ID` | — | ✓ (for OIDC role ARN) | — |

Non-secret tuning (`CITY`, `CATEGORIES`, `PERIOD_DAYS`, `TEST_MODE`) lives in
GitHub **Variables** (not Secrets) — see the workflow defaults.

`deploy/.env` is gitignored. Never commit it.

## Deploy via GitHub Actions

[`.github/workflows/deploy.yml`](.github/workflows/deploy.yml) auto-deploys
on push to `main` (and via manual dispatch). It uses **OIDC** — no long-lived
AWS keys stored in GitHub.

### One-time AWS setup

Run [`deploy/setup-gha.sh`](deploy/setup-gha.sh) — it creates the GitHub OIDC
provider, the deploy role with a trust policy scoped to your repo, and an
inline policy granting Lambda / EventBridge / IAM / CloudWatch Logs access.
Idempotent — safe to re-run.

```bash
aws login                              # or: aws configure sso
./deploy/setup-gha.sh                  # infers repo from "origin"
# or: REPO=owner/name ./deploy/setup-gha.sh
```

The script prints the `AWS_ACCOUNT_ID` value you need to put into GitHub
Secrets, and the role ARN the workflow assumes.

What it creates (under the hood):
- IAM OIDC provider for `token.actions.githubusercontent.com`
- IAM role `github-deploy-winnipeg-tech-events` with a trust policy locked to
  `repo:<owner>/<repo>:ref:refs/heads/main`
- Inline policy `deploy-permissions` with `lambda:*`, `events:*`, the IAM
  actions `deploy.sh` needs (`GetRole`/`CreateRole`/`AttachRolePolicy`/`PassRole`),
  and CloudWatch Logs retention setup

### One-time GitHub setup

In the repo on github.com: **Settings → Secrets and variables → Actions**.

**Secrets** (encrypted):

| Name | Value |
|---|---|
| `AWS_ACCOUNT_ID` | Your 12-digit AWS account id |
| `TELEGRAM_BOT_TOKEN` | From BotFather |
| `TELEGRAM_CHAT_ID` | From `getUpdates` |
| `TELEGRAM_POLL_BOT_TOKEN` | (Optional) second bot |
| `TELEGRAM_POLL_CHAT_ID` | (Optional) second chat |

**Variables** (plain text):

| Name | Default if unset | Purpose |
|---|---|---|
| `CITY` | `Winnipeg` | Scrape target |
| `CATEGORIES` | `tech` | Scrape category |
| `PERIOD_DAYS` | `30` | Days ahead to include |
| `TEST_MODE` | `false` | Build digest but don't send |

### Trigger a deploy

- **Automatic**: push to `main`.
- **Manual**: GitHub repo → **Actions** → **Deploy Lambda** → **Run workflow**.

The workflow ends with a smoke-test invocation of the deployed Lambda and
prints the JSON result in the run log.

## Architecture

```
EventBridge Scheduler (weekly Mon 14:00 UTC) ─┐
                                              ├──→ Lambda (Go, provided.al2023)
EventBridge Scheduler (monthly 20th 14:00 UTC)┘         │
                                                        ├──→ scrape Meetup (JSON-LD)
                                                        ├──→ scrape Eventbrite (SSR JSON)
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
  meetup/       # Meetup scraper (schema.org JSON-LD)
  eventbrite/   # Eventbrite scraper (window.__SERVER_DATA__)
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
