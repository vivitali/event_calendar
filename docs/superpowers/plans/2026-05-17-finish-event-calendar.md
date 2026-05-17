# Finish Event Calendar Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace GitHub Actions scheduling with AWS Lambda + EventBridge, remove dead code (Python Lambda stub, half-built callback voting), deduplicate date/format helpers, and ship a clean, deployable system.

**Architecture:** Single Go Lambda (`provided.al2023`) invoked by two EventBridge rules. Events digest runs weekly; monthly poll runs on the 20th. Job logic lives in `internal/jobs` and is shared by Lambda and a local CLI. Helpers extracted to `internal/timeutil` and `internal/digest`.

**Tech Stack:** Go 1.24.1, `github.com/aws/aws-lambda-go`, AWS CLI v2, EventBridge, Telegram Bot API.

**Spec:** [2026-05-17-finish-event-calendar-design.md](../specs/2026-05-17-finish-event-calendar-design.md)

---

## File Structure (target end state)

```
cmd/
  lambda/main.go        # Lambda entry (NEW)
  cli/main.go           # Local CLI for events/poll (NEW; replaces scheduler/ and poll-scheduler/)
  server/main.go        # Local web UI server (MOVED from cmd/main.go)
internal/
  config/config.go      # Env loading (NEW)
  config/config_test.go
  digest/format.go      # Telegram message formatting (NEW)
  digest/format_test.go
  jobs/events.go        # Events digest job (NEW)
  jobs/events_test.go
  jobs/poll.go          # Monthly poll job (NEW)
  jobs/poll_test.go
  models/event.go       # unchanged
  timeutil/groups.go    # Date grouping helpers (NEW)
  timeutil/groups_test.go
pkg/
  aggregator/aggregator.go    # MODIFIED: use internal/timeutil
  telegram/service.go          # MODIFIED: drop vote keyboards, use internal/timeutil/digest
  (scraping/, eventbrite/, meetup/, devevents/ unchanged)
deploy/
  deploy.sh             # AWS deploy script (NEW)
web/                    # unchanged
README.md               # MODIFIED
.gitignore              # MODIFIED (add deploy artifacts)
go.mod / go.sum         # MODIFIED (add aws-lambda-go)

DELETED:
  cmd/scheduler/        cmd/poll-scheduler/        cmd/webhook/
  pkg/telegram/callback_handler.go
  lambda/                                    # entire dir
  .github/workflows/                         # entire dir
  test_app.go test_poll.sh test_poll_20th.sh test_scheduler.sh
  test_github_actions.sh setup_github_secrets.sh validate.sh
  GITHUB_ACTIONS_SETUP.md SEPARATE_BOTS_SETUP.md POLL_INTEGRATION.md
  main scheduler                             # stray root binaries
```

---

## Task 1: Create `internal/timeutil` package

**Files:**
- Create: `internal/timeutil/groups.go`
- Test: `internal/timeutil/groups_test.go`

**Why:** `isSameDay`, `isThisWeek`, `isNextWeek` are duplicated in `pkg/telegram/service.go`, `pkg/aggregator/aggregator.go`, and `cmd/scheduler/main.go`. Current implementations call `time.Now()` internally, making them untestable. Refactor to take `now` as a parameter.

- [ ] **Step 1: Write the failing test**

Create `internal/timeutil/groups_test.go`:

```go
package timeutil

import (
	"testing"
	"time"
)

func TestIsSameDay(t *testing.T) {
	mon := time.Date(2026, 1, 5, 10, 0, 0, 0, time.UTC) // Monday
	tests := []struct {
		name string
		a, b time.Time
		want bool
	}{
		{"same day same time", mon, mon, true},
		{"same day different hour", mon, mon.Add(3 * time.Hour), true},
		{"next day", mon, mon.Add(25 * time.Hour), false},
		{"year boundary", time.Date(2025, 12, 31, 23, 0, 0, 0, time.UTC), time.Date(2026, 1, 1, 1, 0, 0, 0, time.UTC), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsSameDay(tt.a, tt.b); got != tt.want {
				t.Errorf("IsSameDay(%v, %v) = %v, want %v", tt.a, tt.b, got, tt.want)
			}
		})
	}
}

func TestIsThisWeek(t *testing.T) {
	// Anchor "now" to Wed 2026-01-07 (week is Sun 2026-01-04 .. Sat 2026-01-10)
	now := time.Date(2026, 1, 7, 12, 0, 0, 0, time.UTC)
	tests := []struct {
		name  string
		event time.Time
		want  bool
	}{
		{"sunday of this week", time.Date(2026, 1, 4, 9, 0, 0, 0, time.UTC), true},
		{"saturday of this week", time.Date(2026, 1, 10, 20, 0, 0, 0, time.UTC), true},
		{"previous saturday", time.Date(2026, 1, 3, 12, 0, 0, 0, time.UTC), false},
		{"next sunday", time.Date(2026, 1, 11, 12, 0, 0, 0, time.UTC), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsThisWeek(tt.event, now); got != tt.want {
				t.Errorf("IsThisWeek(%v, now=%v) = %v, want %v", tt.event, now, got, tt.want)
			}
		})
	}
}

func TestIsNextWeek(t *testing.T) {
	now := time.Date(2026, 1, 7, 12, 0, 0, 0, time.UTC) // Wed; next week = Sun 2026-01-11 .. Sat 2026-01-17
	tests := []struct {
		name  string
		event time.Time
		want  bool
	}{
		{"next sunday", time.Date(2026, 1, 11, 9, 0, 0, 0, time.UTC), true},
		{"next saturday", time.Date(2026, 1, 17, 20, 0, 0, 0, time.UTC), true},
		{"this saturday", time.Date(2026, 1, 10, 12, 0, 0, 0, time.UTC), false},
		{"week after next", time.Date(2026, 1, 18, 12, 0, 0, 0, time.UTC), false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := IsNextWeek(tt.event, now); got != tt.want {
				t.Errorf("IsNextWeek(%v, now=%v) = %v, want %v", tt.event, now, got, tt.want)
			}
		})
	}
}

func TestGroupByPeriod(t *testing.T) {
	now := time.Date(2026, 1, 7, 12, 0, 0, 0, time.UTC) // Wed
	events := []time.Time{
		now,                                           // Today
		time.Date(2026, 1, 9, 18, 0, 0, 0, time.UTC),  // Fri (This Week)
		time.Date(2026, 1, 14, 18, 0, 0, 0, time.UTC), // Wed (Next Week)
		time.Date(2026, 3, 1, 12, 0, 0, 0, time.UTC),  // Later
	}
	got := GroupByPeriod(events, now)
	if len(got["Today"]) != 1 {
		t.Errorf("Today: got %d, want 1", len(got["Today"]))
	}
	if len(got["This Week"]) != 1 {
		t.Errorf("This Week: got %d, want 1", len(got["This Week"]))
	}
	if len(got["Next Week"]) != 1 {
		t.Errorf("Next Week: got %d, want 1", len(got["Next Week"]))
	}
	if len(got["Later"]) != 1 {
		t.Errorf("Later: got %d, want 1", len(got["Later"]))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/timeutil/...
```
Expected: FAIL (package does not exist yet).

- [ ] **Step 3: Write minimal implementation**

Create `internal/timeutil/groups.go`:

```go
// Package timeutil groups events into time buckets relative to a reference time.
package timeutil

import "time"

// IsSameDay returns true when a and b fall on the same calendar day.
func IsSameDay(a, b time.Time) bool {
	return a.Year() == b.Year() && a.YearDay() == b.YearDay()
}

// IsThisWeek returns true when t falls in the same Sun..Sat week as now.
func IsThisWeek(t, now time.Time) bool {
	startOfWeek := startOfWeek(now)
	endOfWeek := startOfWeek.AddDate(0, 0, 7)
	return !t.Before(startOfWeek) && t.Before(endOfWeek)
}

// IsNextWeek returns true when t falls in the Sun..Sat week after now's week.
func IsNextWeek(t, now time.Time) bool {
	startOfNext := startOfWeek(now).AddDate(0, 0, 7)
	endOfNext := startOfNext.AddDate(0, 0, 7)
	return !t.Before(startOfNext) && t.Before(endOfNext)
}

// GroupByPeriod buckets times into "Today", "This Week", "Next Week", "Later"
// using now as the reference. Empty buckets are omitted from the result.
func GroupByPeriod(times []time.Time, now time.Time) map[string][]time.Time {
	groups := map[string][]time.Time{}
	for _, t := range times {
		switch {
		case IsSameDay(t, now):
			groups["Today"] = append(groups["Today"], t)
		case IsThisWeek(t, now):
			groups["This Week"] = append(groups["This Week"], t)
		case IsNextWeek(t, now):
			groups["Next Week"] = append(groups["Next Week"], t)
		default:
			groups["Later"] = append(groups["Later"], t)
		}
	}
	return groups
}

func startOfWeek(now time.Time) time.Time {
	y, m, d := now.Date()
	midnight := time.Date(y, m, d, 0, 0, 0, 0, now.Location())
	return midnight.AddDate(0, 0, -int(midnight.Weekday()))
}
```

- [ ] **Step 4: Run tests to verify they pass**

```bash
go test ./internal/timeutil/... -v
```
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/timeutil/
git commit -m "feat(timeutil): extract date grouping helpers with tests"
```

---

## Task 2: Create `internal/digest` package

**Files:**
- Create: `internal/digest/format.go`
- Test: `internal/digest/format_test.go`

**Why:** `FormatMessage` exists in 3 near-duplicate forms (`pkg/telegram/service.go:264`, `cmd/scheduler/main.go:234,292`). Centralize. Use the `timeutil` grouping from Task 1.

- [ ] **Step 1: Write the failing test**

Create `internal/digest/format_test.go`:

```go
package digest

import (
	"strings"
	"testing"
	"time"

	"event_calendar/internal/models"
)

func TestFormatEventsMessage_Empty(t *testing.T) {
	now := time.Date(2026, 1, 7, 12, 0, 0, 0, time.UTC)
	got := FormatEventsMessage(nil, now)
	if !strings.Contains(got, "No upcoming events") {
		t.Errorf("expected empty-state message, got: %s", got)
	}
}

func TestFormatEventsMessage_GroupsAndHeaders(t *testing.T) {
	now := time.Date(2026, 1, 7, 12, 0, 0, 0, time.UTC) // Wed
	events := []models.Event{
		{Name: "Today Event", URL: "https://e/1", Source: "meetup", StartTime: now.Add(3 * time.Hour), Venue: "Hub"},
		{Name: "Friday Event", URL: "https://e/2", Source: "eventbrite", StartTime: time.Date(2026, 1, 9, 18, 0, 0, 0, time.UTC)},
		{Name: "Next Week Event", URL: "https://e/3", Source: "devevents", StartTime: time.Date(2026, 1, 14, 18, 0, 0, 0, time.UTC)},
	}
	msg := FormatEventsMessage(events, now)
	for _, want := range []string{
		"Winnipeg Tech Events",
		"*Today:*",
		"*This Week:*",
		"*Next Week:*",
		"Today Event",
		"Friday Event",
		"Next Week Event",
		"[Meetup]",
		"[Eventbrite]",
		"[Dev.events]",
		"https://e/1",
	} {
		if !strings.Contains(msg, want) {
			t.Errorf("message missing %q.\n--- message ---\n%s", want, msg)
		}
	}
}

func TestFormatEventsMessage_TruncatesLong(t *testing.T) {
	now := time.Date(2026, 1, 7, 12, 0, 0, 0, time.UTC)
	var events []models.Event
	for i := 0; i < 200; i++ {
		events = append(events, models.Event{
			Name:      "Event with a fairly long name that pushes content",
			URL:       "https://example.com/very/long/url/path/segment",
			Source:    "meetup",
			StartTime: now.Add(time.Duration(i) * time.Hour),
		})
	}
	msg := FormatEventsMessage(events, now)
	if len(msg) > 4096 {
		t.Errorf("message exceeded Telegram limit: %d chars", len(msg))
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/digest/...
```
Expected: FAIL (package missing).

- [ ] **Step 3: Write minimal implementation**

Create `internal/digest/format.go`:

```go
// Package digest formats event lists into Telegram messages.
package digest

import (
	"fmt"
	"strings"
	"time"

	"event_calendar/internal/models"
	"event_calendar/internal/timeutil"
)

const telegramMaxLen = 4096

var periodOrder = []string{"Today", "This Week", "Next Week", "Later"}

// FormatEventsMessage renders a Markdown-formatted Telegram digest of the events,
// grouped by time bucket relative to now. The result is guaranteed to be <= 4096 chars.
func FormatEventsMessage(events []models.Event, now time.Time) string {
	if len(events) == 0 {
		return "📅 *No upcoming events found* for Winnipeg tech community."
	}

	var sb strings.Builder
	fmt.Fprintf(&sb, "🚀 *Winnipeg Tech Events - %s*\n\n", now.Format("Monday, January 2, 2006"))

	groups := groupEvents(events, now)
	for _, period := range periodOrder {
		bucket := groups[period]
		if len(bucket) == 0 {
			continue
		}
		fmt.Fprintf(&sb, "*%s:*\n", period)
		for _, e := range bucket {
			writeEvent(&sb, e)
		}
	}
	sb.WriteString("\n_Shared via Winnipeg Tech Events Tracker_")

	out := sb.String()
	if len(out) > telegramMaxLen {
		out = out[:telegramMaxLen-3] + "..."
	}
	return out
}

func groupEvents(events []models.Event, now time.Time) map[string][]models.Event {
	groups := map[string][]models.Event{}
	for _, e := range events {
		switch {
		case timeutil.IsSameDay(e.StartTime, now):
			groups["Today"] = append(groups["Today"], e)
		case timeutil.IsThisWeek(e.StartTime, now):
			groups["This Week"] = append(groups["This Week"], e)
		case timeutil.IsNextWeek(e.StartTime, now):
			groups["Next Week"] = append(groups["Next Week"], e)
		default:
			groups["Later"] = append(groups["Later"], e)
		}
	}
	return groups
}

func writeEvent(sb *strings.Builder, e models.Event) {
	fmt.Fprintf(sb, "• %s %s\n", e.Name, sourceLabel(e.Source))
	if !e.StartTime.IsZero() {
		fmt.Fprintf(sb, "  📅 %s\n", e.StartTime.Format("Monday, Jan 2"))
	}
	if e.Venue != "" {
		fmt.Fprintf(sb, "  📍 %s\n", e.Venue)
	}
	if e.Price != "" && e.Price != "Free" {
		fmt.Fprintf(sb, "  💰 %s\n", e.Price)
	}
	if e.URL != "" {
		fmt.Fprintf(sb, "  🔗 [View Event](%s)\n", e.URL)
	}
	sb.WriteString("\n")
}

func sourceLabel(source string) string {
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
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/digest/... -v
```
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/digest/
git commit -m "feat(digest): extract Telegram message formatting with tests"
```

---

## Task 3: Create `internal/config` package

**Files:**
- Create: `internal/config/config.go`
- Test: `internal/config/config_test.go`

**Why:** Config loading currently lives inside `cmd/scheduler/main.go:57` (and a partial copy in `cmd/poll-scheduler/main.go:47`). Hoist to a reusable package and add the fallback semantics from the spec (poll bot/chat default to main).

- [ ] **Step 1: Write the failing test**

Create `internal/config/config_test.go`:

```go
package config

import (
	"testing"
)

func TestLoad_DefaultsAndFallbacks(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "main-token")
	t.Setenv("TELEGRAM_CHAT_ID", "main-chat")
	// Poll envs unset — should fall back.
	cfg := Load()
	if cfg.BotToken != "main-token" || cfg.ChatID != "main-chat" {
		t.Fatalf("main creds wrong: %+v", cfg)
	}
	if cfg.PollBotToken != "main-token" || cfg.PollChatID != "main-chat" {
		t.Errorf("poll did not fall back to main: %+v", cfg)
	}
	if cfg.City != "Winnipeg" || cfg.Categories != "tech" || cfg.PeriodDays != 30 {
		t.Errorf("defaults wrong: %+v", cfg)
	}
	if cfg.TestMode {
		t.Errorf("TestMode should default to false")
	}
}

func TestLoad_OverridesAndExplicitPollCreds(t *testing.T) {
	t.Setenv("TELEGRAM_BOT_TOKEN", "main")
	t.Setenv("TELEGRAM_CHAT_ID", "mainchat")
	t.Setenv("TELEGRAM_POLL_BOT_TOKEN", "poll")
	t.Setenv("TELEGRAM_POLL_CHAT_ID", "pollchat")
	t.Setenv("CITY", "Toronto")
	t.Setenv("CATEGORIES", "ai")
	t.Setenv("PERIOD_DAYS", "14")
	t.Setenv("TEST_MODE", "true")

	cfg := Load()
	if cfg.PollBotToken != "poll" || cfg.PollChatID != "pollchat" {
		t.Errorf("explicit poll creds not honored: %+v", cfg)
	}
	if cfg.City != "Toronto" || cfg.Categories != "ai" || cfg.PeriodDays != 14 || !cfg.TestMode {
		t.Errorf("overrides not applied: %+v", cfg)
	}
}

func TestLoad_BadPeriodDays(t *testing.T) {
	t.Setenv("PERIOD_DAYS", "not-a-number")
	cfg := Load()
	if cfg.PeriodDays != 30 {
		t.Errorf("expected fallback to 30 on bad PERIOD_DAYS, got %d", cfg.PeriodDays)
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

```bash
go test ./internal/config/...
```
Expected: FAIL (package missing).

- [ ] **Step 3: Implement**

Create `internal/config/config.go`:

```go
// Package config loads runtime configuration from environment variables.
package config

import (
	"os"
	"strconv"
)

// Config holds all runtime settings. Poll fields default to the main bot/chat
// when their dedicated env vars are not set.
type Config struct {
	BotToken     string
	ChatID       string
	PollBotToken string
	PollChatID   string
	City         string
	Categories   string
	PeriodDays   int
	TestMode     bool
}

// Load reads environment variables and applies defaults.
func Load() Config {
	cfg := Config{
		BotToken:   os.Getenv("TELEGRAM_BOT_TOKEN"),
		ChatID:     os.Getenv("TELEGRAM_CHAT_ID"),
		City:       envOr("CITY", "Winnipeg"),
		Categories: envOr("CATEGORIES", "tech"),
		PeriodDays: envInt("PERIOD_DAYS", 30),
		TestMode:   os.Getenv("TEST_MODE") == "true",
	}
	cfg.PollBotToken = envOr("TELEGRAM_POLL_BOT_TOKEN", cfg.BotToken)
	cfg.PollChatID = envOr("TELEGRAM_POLL_CHAT_ID", cfg.ChatID)
	return cfg
}

func envOr(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}

func envInt(key string, fallback int) int {
	v := os.Getenv(key)
	if v == "" {
		return fallback
	}
	n, err := strconv.Atoi(v)
	if err != nil || n <= 0 {
		return fallback
	}
	return n
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/config/... -v
```
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat(config): centralize env loading with poll-creds fallback"
```

---

## Task 4: Refactor `pkg/aggregator` to use `internal/timeutil`

**Files:**
- Modify: `pkg/aggregator/aggregator.go`

**Why:** Remove duplicate `isSameDay`/`isThisWeek`/`isNextWeek` from this package; delegate to `timeutil`. `GroupEventsByTime` is unused outside the package now (the new `digest` package owns formatting), so delete it. Keep `Aggregator` struct, `removeDuplicates`, and `FilterFutureEvents` (still used).

- [ ] **Step 1: Replace file contents**

Overwrite `pkg/aggregator/aggregator.go`:

```go
// Package aggregator combines multiple event providers and filters/dedupes results.
package aggregator

import (
	"log"
	"sort"
	"time"

	"event_calendar/internal/models"
)

// EventProvider produces events from a single source.
type EventProvider interface {
	GetEvents(city, category string, period time.Duration) ([]models.Event, error)
}

// Aggregator fans out to multiple providers, merges results, and removes duplicates.
type Aggregator struct {
	providers []EventProvider
}

// NewAggregator constructs an aggregator over the supplied providers.
func NewAggregator(providers ...EventProvider) *Aggregator {
	return &Aggregator{providers: providers}
}

// AggregateEvents calls every provider, merges their results, sorts by start time,
// and removes duplicates. Per-provider errors are logged and skipped.
func (a *Aggregator) AggregateEvents(city, category string, period time.Duration) ([]models.Event, error) {
	var out []models.Event
	for _, p := range a.providers {
		events, err := p.GetEvents(city, category, period)
		if err != nil {
			log.Printf("provider error: %v", err)
			continue
		}
		out = append(out, events...)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].StartTime.Before(out[j].StartTime) })
	return removeDuplicates(out), nil
}

func removeDuplicates(events []models.Event) []models.Event {
	seen := make(map[string]bool)
	var unique []models.Event
	for _, e := range events {
		key := e.URL + "|" + e.Name
		if seen[key] {
			continue
		}
		seen[key] = true
		unique = append(unique, e)
	}
	return unique
}

// FilterFutureEvents returns only events whose StartTime is after now.
func FilterFutureEvents(events []models.Event, now time.Time) []models.Event {
	var future []models.Event
	for _, e := range events {
		if e.StartTime.After(now) {
			future = append(future, e)
		}
	}
	return future
}
```

- [ ] **Step 2: Verify it compiles**

```bash
go build ./pkg/aggregator/...
```
Expected: success.

- [ ] **Step 3: Verify no other code broke yet**

```bash
go build ./... 2>&1 | head -50
```
Expected: failures only in code that imported the deleted `GroupEventsByTime` or called `FilterFutureEvents` without a `now` argument (these will be fixed in later tasks).

- [ ] **Step 4: Commit**

```bash
git add pkg/aggregator/aggregator.go
git commit -m "refactor(aggregator): drop duplicate time helpers, take now as arg"
```

---

## Task 5: Slim `pkg/telegram/service.go` (remove vote keyboards + dedupe)

**Files:**
- Modify: `pkg/telegram/service.go`
- Delete: `pkg/telegram/callback_handler.go`

**Why:** Spec drops callback voting. Strip vote keyboards, `SendMessageWithKeyboard`, the old `FormatMessage` (replaced by `internal/digest`), and the local duplicates of `isSameDay`/`isThisWeek`/`isNextWeek`. Keep `SendMessage`, `SendAlert`, `SendPoll`, `SendMonthlyMeetupPoll`, `TestConnection`, `GetChatInfo`.

- [ ] **Step 1: Replace file contents**

Overwrite `pkg/telegram/service.go`:

```go
// Package telegram provides a minimal Telegram Bot API client used to send
// event digests, polls, and alerts.
package telegram

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// Service is a thin HTTP client over the Telegram Bot API.
type Service struct {
	botToken string
	client   *http.Client
	baseURL  string
}

// NewService constructs a Service for the given bot token.
func NewService(botToken string) *Service {
	return &Service{
		botToken: botToken,
		client:   &http.Client{Timeout: 30 * time.Second},
		baseURL:  "https://api.telegram.org/bot" + botToken,
	}
}

type sendMessageRequest struct {
	ChatID                string `json:"chat_id"`
	Text                  string `json:"text"`
	ParseMode             string `json:"parse_mode,omitempty"`
	DisableWebPagePreview bool   `json:"disable_web_page_preview,omitempty"`
}

type apiResponse struct {
	OK          bool   `json:"ok"`
	Description string `json:"description,omitempty"`
}

// SendMessage sends a Markdown-formatted message to chatID.
func (s *Service) SendMessage(chatID, message string) error {
	if s.botToken == "" {
		return fmt.Errorf("bot token not configured")
	}
	if chatID == "" {
		return fmt.Errorf("chat ID not provided")
	}
	if message == "" {
		return fmt.Errorf("message is empty")
	}
	if len(message) > 4096 {
		return fmt.Errorf("message too long (%d characters, max 4096)", len(message))
	}
	return s.post("/sendMessage", sendMessageRequest{
		ChatID:                chatID,
		Text:                  message,
		ParseMode:             "Markdown",
		DisableWebPagePreview: true,
	})
}

// SendAlert wraps the message in a standard alert template.
func (s *Service) SendAlert(chatID, alertMessage string) error {
	body := fmt.Sprintf("🚨 *Winnipeg Tech Events Alert*\n\n%s\n\n_Time: %s_",
		alertMessage, time.Now().Format("2006-01-02 15:04:05 MST"))
	return s.SendMessage(chatID, body)
}

// TestConnection calls /getMe to verify the bot token works.
func (s *Service) TestConnection() error {
	if s.botToken == "" {
		return fmt.Errorf("bot token not configured")
	}
	resp, err := s.client.Get(s.baseURL + "/getMe")
	if err != nil {
		return fmt.Errorf("failed to test connection: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("connection test failed with status: %d", resp.StatusCode)
	}
	return nil
}

type sendPollRequest struct {
	ChatID                string   `json:"chat_id"`
	Question              string   `json:"question"`
	Options               []string `json:"options"`
	IsAnonymous           bool     `json:"is_anonymous"`
	Type                  string   `json:"type"`
	AllowsMultipleAnswers bool     `json:"allows_multiple_answers"`
}

// SendPoll sends a non-anonymous poll. 2-10 options required by Telegram.
func (s *Service) SendPoll(chatID, question string, options []string, allowMultiple bool) error {
	if s.botToken == "" {
		return fmt.Errorf("bot token not configured")
	}
	if chatID == "" {
		return fmt.Errorf("chat ID not provided")
	}
	if question == "" {
		return fmt.Errorf("question is empty")
	}
	if len(options) < 2 || len(options) > 10 {
		return fmt.Errorf("poll requires 2-10 options, got %d", len(options))
	}
	return s.post("/sendPoll", sendPollRequest{
		ChatID:                chatID,
		Question:              question,
		Options:               options,
		IsAnonymous:           false,
		Type:                  "regular",
		AllowsMultipleAnswers: allowMultiple,
	})
}

// SendMonthlyMeetupPoll posts the standard monthly day-of-week poll.
func (s *Service) SendMonthlyMeetupPoll(chatID string) error {
	return s.SendPoll(chatID,
		"Є бажаючі зустрітись - виберіть день тижня",
		[]string{"Понеділок", "Вівторок", "Середа", "Четвер", "П'ятниця", "Субота", "Неділя"},
		true,
	)
}

func (s *Service) post(path string, body any) error {
	jsonData, err := json.Marshal(body)
	if err != nil {
		return fmt.Errorf("marshal request: %w", err)
	}
	resp, err := s.client.Post(s.baseURL+path, "application/json", bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("HTTP request: %w", err)
	}
	defer resp.Body.Close()
	respBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return fmt.Errorf("read response: %w", err)
	}
	var r apiResponse
	if err := json.Unmarshal(respBody, &r); err != nil {
		return fmt.Errorf("parse response: %w", err)
	}
	if !r.OK {
		return fmt.Errorf("telegram API error: %s", r.Description)
	}
	return nil
}
```

- [ ] **Step 2: Delete the callback handler**

```bash
rm pkg/telegram/callback_handler.go
```

- [ ] **Step 3: Build the telegram package**

```bash
go build ./pkg/telegram/...
```
Expected: success.

- [ ] **Step 4: Commit**

```bash
git add pkg/telegram/
git commit -m "refactor(telegram): drop callback voting and duplicated helpers"
```

---

## Task 6: Create `internal/jobs/events.go`

**Files:**
- Create: `internal/jobs/events.go`
- Test: `internal/jobs/events_test.go`

**Why:** Reusable events-digest logic for both Lambda and CLI. Takes dependencies as interfaces so tests can use fakes.

- [ ] **Step 1: Write the failing test**

Create `internal/jobs/events_test.go`:

```go
package jobs

import (
	"errors"
	"testing"
	"time"

	"event_calendar/internal/config"
	"event_calendar/internal/models"
)

type fakeScraper struct {
	events []models.Event
	err    error
}

func (f *fakeScraper) ScrapeEvents(city, category string, period time.Duration) ([]models.Event, error) {
	return f.events, f.err
}

type fakeSender struct {
	chatID  string
	message string
	err     error
	calls   int
}

func (f *fakeSender) SendMessage(chatID, message string) error {
	f.calls++
	f.chatID = chatID
	f.message = message
	return f.err
}

func TestRunEventsDigest_Success(t *testing.T) {
	now := time.Date(2026, 1, 7, 12, 0, 0, 0, time.UTC)
	scraper := &fakeScraper{events: []models.Event{
		{Name: "Future", StartTime: now.Add(48 * time.Hour), Source: "meetup", URL: "https://e/1"},
		{Name: "Past", StartTime: now.Add(-48 * time.Hour), Source: "meetup", URL: "https://e/2"},
	}}
	sender := &fakeSender{}
	cfg := config.Config{BotToken: "tok", ChatID: "chat", City: "Winnipeg", Categories: "tech", PeriodDays: 30}

	res := RunEventsDigest(EventsDeps{Scraper: scraper, Sender: sender, Now: now}, cfg)
	if !res.Success {
		t.Fatalf("expected success, got %+v", res)
	}
	if res.EventsCount != 1 {
		t.Errorf("expected 1 future event, got %d", res.EventsCount)
	}
	if !res.MessageSent {
		t.Errorf("expected message sent")
	}
	if sender.calls != 1 || sender.chatID != "chat" {
		t.Errorf("sender not called correctly: %+v", sender)
	}
}

func TestRunEventsDigest_NoEvents(t *testing.T) {
	now := time.Date(2026, 1, 7, 12, 0, 0, 0, time.UTC)
	scraper := &fakeScraper{events: nil}
	sender := &fakeSender{}
	cfg := config.Config{BotToken: "tok", ChatID: "chat"}

	res := RunEventsDigest(EventsDeps{Scraper: scraper, Sender: sender, Now: now}, cfg)
	if !res.Success {
		t.Fatalf("expected success even with zero events, got %+v", res)
	}
	if res.MessageSent {
		t.Errorf("should not send when no events")
	}
	if sender.calls != 0 {
		t.Errorf("sender should not be called")
	}
}

func TestRunEventsDigest_TestMode(t *testing.T) {
	now := time.Date(2026, 1, 7, 12, 0, 0, 0, time.UTC)
	scraper := &fakeScraper{events: []models.Event{
		{Name: "Future", StartTime: now.Add(24 * time.Hour), Source: "meetup"},
	}}
	sender := &fakeSender{}
	cfg := config.Config{BotToken: "tok", ChatID: "chat", TestMode: true}

	res := RunEventsDigest(EventsDeps{Scraper: scraper, Sender: sender, Now: now}, cfg)
	if !res.Success || res.MessageSent {
		t.Errorf("test mode should succeed without sending: %+v", res)
	}
	if sender.calls != 0 {
		t.Errorf("sender called in test mode")
	}
}

func TestRunEventsDigest_ScraperError(t *testing.T) {
	now := time.Date(2026, 1, 7, 12, 0, 0, 0, time.UTC)
	scraper := &fakeScraper{err: errors.New("network down")}
	sender := &fakeSender{}
	cfg := config.Config{BotToken: "tok", ChatID: "chat"}

	res := RunEventsDigest(EventsDeps{Scraper: scraper, Sender: sender, Now: now}, cfg)
	if res.Success {
		t.Errorf("expected failure on scraper error: %+v", res)
	}
	if res.Error == "" {
		t.Errorf("expected error message")
	}
}
```

- [ ] **Step 2: Verify test fails**

```bash
go test ./internal/jobs/...
```
Expected: FAIL (package missing).

- [ ] **Step 3: Write the implementation**

Create `internal/jobs/events.go`:

```go
// Package jobs contains the reusable scheduled-job logic for events digests
// and monthly polls. Each job takes its dependencies as interfaces so callers
// (Lambda, local CLI) can wire real or fake implementations.
package jobs

import (
	"fmt"
	"log"
	"time"

	"event_calendar/internal/config"
	"event_calendar/internal/digest"
	"event_calendar/internal/models"
)

// Scraper produces events from one or more sources.
type Scraper interface {
	ScrapeEvents(city, category string, period time.Duration) ([]models.Event, error)
}

// Sender posts a Markdown-formatted message to a Telegram chat.
type Sender interface {
	SendMessage(chatID, message string) error
}

// EventsDeps holds the collaborators needed by RunEventsDigest.
type EventsDeps struct {
	Scraper Scraper
	Sender  Sender
	Now     time.Time
}

// Result reports the outcome of a job execution.
type Result struct {
	Success     bool   `json:"success"`
	EventsCount int    `json:"events_count"`
	MessageSent bool   `json:"message_sent"`
	Error       string `json:"error,omitempty"`
}

// RunEventsDigest scrapes upcoming events, formats a digest, and sends it to
// the configured chat. In TestMode the digest is built but not sent.
func RunEventsDigest(deps EventsDeps, cfg config.Config) Result {
	period := time.Duration(cfg.PeriodDays) * 24 * time.Hour
	events, err := deps.Scraper.ScrapeEvents(cfg.City, cfg.Categories, period)
	if err != nil {
		return Result{Error: fmt.Sprintf("scrape failed: %v", err)}
	}

	future := filterFuture(events, deps.Now)
	log.Printf("events: scraped=%d future=%d", len(events), len(future))

	if len(future) == 0 {
		return Result{Success: true, EventsCount: 0}
	}

	message := digest.FormatEventsMessage(future, deps.Now)

	if cfg.TestMode {
		log.Printf("test mode: would have sent %d chars", len(message))
		return Result{Success: true, EventsCount: len(future), MessageSent: false}
	}

	if cfg.BotToken == "" || cfg.ChatID == "" {
		return Result{Error: "TELEGRAM_BOT_TOKEN or TELEGRAM_CHAT_ID not set"}
	}

	if err := deps.Sender.SendMessage(cfg.ChatID, message); err != nil {
		return Result{Error: fmt.Sprintf("send failed: %v", err)}
	}
	return Result{Success: true, EventsCount: len(future), MessageSent: true}
}

func filterFuture(events []models.Event, now time.Time) []models.Event {
	var out []models.Event
	for _, e := range events {
		if e.StartTime.After(now) {
			out = append(out, e)
		}
	}
	return out
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/jobs/... -v
```
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/jobs/events.go internal/jobs/events_test.go
git commit -m "feat(jobs): add events-digest job with tests"
```

---

## Task 7: Create `internal/jobs/poll.go`

**Files:**
- Create: `internal/jobs/poll.go`
- Test: `internal/jobs/poll_test.go`

**Why:** Reusable monthly-poll logic. Per spec §5.1, the `is20thOfMonth` gate is removed — EventBridge handles scheduling.

- [ ] **Step 1: Write the failing test**

Create `internal/jobs/poll_test.go`:

```go
package jobs

import (
	"errors"
	"testing"

	"event_calendar/internal/config"
)

type fakePoller struct {
	chatID string
	called int
	err    error
}

func (f *fakePoller) SendMonthlyMeetupPoll(chatID string) error {
	f.called++
	f.chatID = chatID
	return f.err
}

func TestRunMonthlyPoll_Success(t *testing.T) {
	poller := &fakePoller{}
	cfg := config.Config{PollBotToken: "tok", PollChatID: "pollchat"}

	res := RunMonthlyPoll(PollDeps{Poller: poller}, cfg)
	if !res.Success || !res.MessageSent {
		t.Fatalf("expected success+sent: %+v", res)
	}
	if poller.chatID != "pollchat" || poller.called != 1 {
		t.Errorf("poller not called correctly: %+v", poller)
	}
}

func TestRunMonthlyPoll_TestMode(t *testing.T) {
	poller := &fakePoller{}
	cfg := config.Config{PollBotToken: "tok", PollChatID: "pollchat", TestMode: true}

	res := RunMonthlyPoll(PollDeps{Poller: poller}, cfg)
	if !res.Success || res.MessageSent {
		t.Errorf("test mode should succeed without sending: %+v", res)
	}
	if poller.called != 0 {
		t.Errorf("poller called in test mode")
	}
}

func TestRunMonthlyPoll_MissingCreds(t *testing.T) {
	res := RunMonthlyPoll(PollDeps{Poller: &fakePoller{}}, config.Config{})
	if res.Success {
		t.Errorf("expected failure when creds missing: %+v", res)
	}
}

func TestRunMonthlyPoll_SendError(t *testing.T) {
	poller := &fakePoller{err: errors.New("kaboom")}
	cfg := config.Config{PollBotToken: "tok", PollChatID: "pollchat"}
	res := RunMonthlyPoll(PollDeps{Poller: poller}, cfg)
	if res.Success {
		t.Errorf("expected failure on send error: %+v", res)
	}
}
```

- [ ] **Step 2: Verify test fails**

```bash
go test ./internal/jobs/... -run RunMonthlyPoll
```
Expected: FAIL (functions not defined yet).

- [ ] **Step 3: Write the implementation**

Create `internal/jobs/poll.go`:

```go
package jobs

import (
	"fmt"
	"log"

	"event_calendar/internal/config"
)

// Poller sends a pre-canned poll to a chat.
type Poller interface {
	SendMonthlyMeetupPoll(chatID string) error
}

// PollDeps holds the collaborators needed by RunMonthlyPoll.
type PollDeps struct {
	Poller Poller
}

// RunMonthlyPoll sends the monthly meetup poll. In TestMode the poll is
// skipped. Scheduling (i.e. "only on the 20th") is the responsibility of
// the caller (EventBridge in production).
func RunMonthlyPoll(deps PollDeps, cfg config.Config) Result {
	if cfg.PollBotToken == "" || cfg.PollChatID == "" {
		return Result{Error: "TELEGRAM_POLL_BOT_TOKEN or TELEGRAM_POLL_CHAT_ID not set"}
	}
	if cfg.TestMode {
		log.Printf("test mode: would have sent monthly poll to %s", cfg.PollChatID)
		return Result{Success: true, MessageSent: false}
	}
	if err := deps.Poller.SendMonthlyMeetupPoll(cfg.PollChatID); err != nil {
		return Result{Error: fmt.Sprintf("poll send failed: %v", err)}
	}
	return Result{Success: true, MessageSent: true}
}
```

- [ ] **Step 4: Run tests**

```bash
go test ./internal/jobs/... -v
```
Expected: all PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/jobs/poll.go internal/jobs/poll_test.go
git commit -m "feat(jobs): add monthly-poll job with tests"
```

---

## Task 8: Add `aws-lambda-go` dependency and `cmd/lambda`

**Files:**
- Create: `cmd/lambda/main.go`
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: Add the Lambda SDK**

```bash
go get github.com/aws/aws-lambda-go@latest
```
Expected: `go.mod` and `go.sum` updated.

- [ ] **Step 2: Create the Lambda entry point**

Create `cmd/lambda/main.go`:

```go
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
```

- [ ] **Step 3: Build for local target to catch errors**

```bash
go build ./cmd/lambda/...
```
Expected: success.

- [ ] **Step 4: Build for Lambda target**

```bash
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 go build -tags lambda.norpc -o bootstrap ./cmd/lambda
ls -la bootstrap
```
Expected: a `bootstrap` binary appears.

- [ ] **Step 5: Add `bootstrap` and `*.zip` to `.gitignore`**

Append to `.gitignore`:

```
# AWS Lambda build artifacts
bootstrap
lambda.zip
deploy/lambda.zip
```

- [ ] **Step 6: Clean local build artifact**

```bash
rm bootstrap
```

- [ ] **Step 7: Commit**

```bash
git add go.mod go.sum cmd/lambda/main.go .gitignore
git commit -m "feat(lambda): add AWS Lambda entry point dispatching events and poll"
```

---

## Task 9: Create `cmd/cli` for local manual runs

**Files:**
- Create: `cmd/cli/main.go`

- [ ] **Step 1: Implement**

Create `cmd/cli/main.go`:

```go
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
	"event_calendar/internal/models"
	"event_calendar/pkg/devevents"
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
	scraper := combinedScraper{main: factory.CreateDefaultService(), dev: devevents.NewScraper()}
	sender := telegram.NewService(cfg.BotToken)
	return jobs.RunEventsDigest(jobs.EventsDeps{Scraper: scraper, Sender: sender, Now: time.Now()}, cfg)
}

func runPoll(cfg config.Config) jobs.Result {
	poller := telegram.NewService(cfg.PollBotToken)
	return jobs.RunMonthlyPoll(jobs.PollDeps{Poller: poller}, cfg)
}

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
```

- [ ] **Step 2: Build**

```bash
go build ./cmd/cli/...
```
Expected: success.

- [ ] **Step 3: Smoke test in test mode (no Telegram send)**

```bash
TEST_MODE=true CITY=Winnipeg CATEGORIES=tech TELEGRAM_BOT_TOKEN=x TELEGRAM_CHAT_ID=y go run ./cmd/cli -action=events
```
Expected: logs end with `"success": true`, `"message_sent": false`.

- [ ] **Step 4: Commit**

```bash
git add cmd/cli/main.go
git commit -m "feat(cli): add local CLI runner for events and poll jobs"
```

---

## Task 10: Move `cmd/main.go` → `cmd/server/main.go`

**Files:**
- Create: `cmd/server/main.go` (moved content)
- Delete: `cmd/main.go`

- [ ] **Step 1: Move the file**

```bash
mkdir -p cmd/server
git mv cmd/main.go cmd/server/main.go
```

- [ ] **Step 2: Verify build**

```bash
go build ./cmd/server/...
```
Expected: success.

- [ ] **Step 3: Commit**

```bash
git add cmd/server/main.go
git commit -m "refactor: move web UI server to cmd/server"
```

---

## Task 11: Delete obsolete `cmd/scheduler` and `cmd/poll-scheduler`

**Files:**
- Delete: `cmd/scheduler/`, `cmd/poll-scheduler/`

- [ ] **Step 1: Delete**

```bash
rm -rf cmd/scheduler cmd/poll-scheduler
```

- [ ] **Step 2: Verify build**

```bash
go build ./...
```
Expected: success.

- [ ] **Step 3: Commit**

```bash
git add -A cmd/
git commit -m "chore: remove obsolete scheduler binaries (replaced by cmd/lambda + cmd/cli)"
```

---

## Task 12: Delete `cmd/webhook` (callback voting server)

**Files:**
- Delete: `cmd/webhook/`

- [ ] **Step 1: Delete**

```bash
rm -rf cmd/webhook
```

- [ ] **Step 2: Verify build**

```bash
go build ./...
```
Expected: success.

- [ ] **Step 3: Commit**

```bash
git add -A cmd/
git commit -m "chore: remove webhook server (callback voting deferred)"
```

---

## Task 13: Delete Python Lambda

**Files:**
- Delete: `lambda/` directory

- [ ] **Step 1: Delete**

```bash
rm -rf lambda/
```

- [ ] **Step 2: Commit**

```bash
git add -A lambda/
git commit -m "chore: remove Python Lambda stub (replaced by Go Lambda)"
```

---

## Task 14: Delete GitHub Actions workflows

**Files:**
- Delete: `.github/workflows/`

- [ ] **Step 1: Delete**

```bash
rm -rf .github/workflows/
```

- [ ] **Step 2: Commit**

```bash
git add -A .github/
git commit -m "chore: remove GitHub Actions workflows (replaced by AWS EventBridge)"
```

---

## Task 15: Delete stale test scripts, docs, and stray binaries

**Files:**
- Delete: `test_app.go`, `test_poll.sh`, `test_poll_20th.sh`, `test_scheduler.sh`, `test_github_actions.sh`, `setup_github_secrets.sh`, `validate.sh`, `GITHUB_ACTIONS_SETUP.md`, `SEPARATE_BOTS_SETUP.md`, `POLL_INTEGRATION.md`, `main`, `scheduler` (root-level stray binaries)

- [ ] **Step 1: Delete**

```bash
rm -f test_app.go test_poll.sh test_poll_20th.sh test_scheduler.sh test_github_actions.sh \
      setup_github_secrets.sh validate.sh \
      GITHUB_ACTIONS_SETUP.md SEPARATE_BOTS_SETUP.md POLL_INTEGRATION.md \
      main scheduler
```

- [ ] **Step 2: Verify build still works**

```bash
go build ./... && go vet ./...
```
Expected: success, no vet warnings.

- [ ] **Step 3: Commit**

```bash
git add -A
git commit -m "chore: remove ad-hoc test scripts and stale docs"
```

---

## Task 16: Run the full test suite

- [ ] **Step 1: Test all packages**

```bash
go test ./... -v
```
Expected: all PASS. New packages (`internal/timeutil`, `internal/digest`, `internal/config`, `internal/jobs`) have tests; the rest compile.

- [ ] **Step 2: Vet**

```bash
go vet ./...
```
Expected: no output.

- [ ] **Step 3: No commit needed (verification only)**

---

## Task 17: Write `deploy/deploy.sh`

**Files:**
- Create: `deploy/deploy.sh`

**Why:** Idempotent script that builds the Lambda, creates/updates the function, IAM role, and EventBridge rules.

- [ ] **Step 1: Create the script**

Create `deploy/deploy.sh`:

```bash
#!/usr/bin/env bash
#
# deploy/deploy.sh — Build and deploy the Winnipeg Tech Events Lambda.
#
# Requirements:
#   - AWS CLI v2 configured (e.g. `aws login` or `aws configure sso`)
#   - Go 1.24+ installed
#
# Environment (read by Lambda; pass via shell or `deploy/.env`):
#   TELEGRAM_BOT_TOKEN
#   TELEGRAM_CHAT_ID
#   TELEGRAM_POLL_BOT_TOKEN   (optional; falls back to TELEGRAM_BOT_TOKEN)
#   TELEGRAM_POLL_CHAT_ID     (optional; falls back to TELEGRAM_CHAT_ID)
#   CITY=Winnipeg (default)
#   CATEGORIES=tech (default)
#   PERIOD_DAYS=30 (default)
#   TEST_MODE=false (default)
set -euo pipefail

FUNCTION_NAME="winnipeg-tech-events"
REGION="${AWS_REGION:-us-east-1}"
ROLE_NAME="winnipeg-tech-events-lambda-role"
RULE_EVENTS="winnipeg-events-weekly"
RULE_POLL="winnipeg-poll-monthly"
RUNTIME="provided.al2023"
HANDLER="bootstrap"
TIMEOUT=300
MEMORY=512
LOG_RETENTION_DAYS=14

cd "$(dirname "$0")/.."

# Load deploy/.env if present (does not commit secrets; .env is gitignored)
if [[ -f deploy/.env ]]; then
    set -a
    # shellcheck disable=SC1091
    source deploy/.env
    set +a
fi

echo "== Building Go Lambda =="
GOOS=linux GOARCH=amd64 CGO_ENABLED=0 \
    go build -tags lambda.norpc -o bootstrap ./cmd/lambda
zip -j deploy/lambda.zip bootstrap >/dev/null
rm bootstrap
echo "  -> deploy/lambda.zip"

ACCOUNT_ID=$(aws sts get-caller-identity --query Account --output text)
ROLE_ARN="arn:aws:iam::${ACCOUNT_ID}:role/${ROLE_NAME}"

echo "== Ensuring IAM role ${ROLE_NAME} =="
if ! aws iam get-role --role-name "$ROLE_NAME" >/dev/null 2>&1; then
    aws iam create-role \
        --role-name "$ROLE_NAME" \
        --assume-role-policy-document '{
            "Version": "2012-10-17",
            "Statement": [{"Effect":"Allow","Principal":{"Service":"lambda.amazonaws.com"},"Action":"sts:AssumeRole"}]
        }' >/dev/null
    aws iam attach-role-policy \
        --role-name "$ROLE_NAME" \
        --policy-arn arn:aws:iam::aws:policy/service-role/AWSLambdaBasicExecutionRole
    echo "  -> role created; waiting 10s for IAM propagation"
    sleep 10
fi

echo "== Deploying Lambda function =="
if aws lambda get-function --function-name "$FUNCTION_NAME" --region "$REGION" >/dev/null 2>&1; then
    aws lambda update-function-code \
        --function-name "$FUNCTION_NAME" \
        --zip-file fileb://deploy/lambda.zip \
        --region "$REGION" >/dev/null
    aws lambda wait function-updated --function-name "$FUNCTION_NAME" --region "$REGION"
    aws lambda update-function-configuration \
        --function-name "$FUNCTION_NAME" \
        --runtime "$RUNTIME" \
        --handler "$HANDLER" \
        --timeout "$TIMEOUT" \
        --memory-size "$MEMORY" \
        --region "$REGION" >/dev/null
    echo "  -> updated"
else
    aws lambda create-function \
        --function-name "$FUNCTION_NAME" \
        --runtime "$RUNTIME" \
        --role "$ROLE_ARN" \
        --handler "$HANDLER" \
        --zip-file fileb://deploy/lambda.zip \
        --timeout "$TIMEOUT" \
        --memory-size "$MEMORY" \
        --region "$REGION" >/dev/null
    echo "  -> created"
fi

echo "== Setting Lambda environment variables =="
aws lambda update-function-configuration \
    --function-name "$FUNCTION_NAME" \
    --region "$REGION" \
    --environment "Variables={
        TELEGRAM_BOT_TOKEN=${TELEGRAM_BOT_TOKEN:-},
        TELEGRAM_CHAT_ID=${TELEGRAM_CHAT_ID:-},
        TELEGRAM_POLL_BOT_TOKEN=${TELEGRAM_POLL_BOT_TOKEN:-},
        TELEGRAM_POLL_CHAT_ID=${TELEGRAM_POLL_CHAT_ID:-},
        CITY=${CITY:-Winnipeg},
        CATEGORIES=${CATEGORIES:-tech},
        PERIOD_DAYS=${PERIOD_DAYS:-30},
        TEST_MODE=${TEST_MODE:-false}
    }" >/dev/null

echo "== Setting CloudWatch log retention to ${LOG_RETENTION_DAYS} days =="
aws logs put-retention-policy \
    --log-group-name "/aws/lambda/${FUNCTION_NAME}" \
    --retention-in-days "$LOG_RETENTION_DAYS" \
    --region "$REGION" 2>/dev/null || true

create_or_update_rule() {
    local rule_name="$1"
    local schedule="$2"
    local input="$3"
    local description="$4"

    echo "== Ensuring EventBridge rule ${rule_name} =="
    aws events put-rule \
        --name "$rule_name" \
        --schedule-expression "$schedule" \
        --description "$description" \
        --region "$REGION" >/dev/null

    aws lambda add-permission \
        --function-name "$FUNCTION_NAME" \
        --statement-id "allow-${rule_name}" \
        --action lambda:InvokeFunction \
        --principal events.amazonaws.com \
        --source-arn "arn:aws:events:${REGION}:${ACCOUNT_ID}:rule/${rule_name}" \
        --region "$REGION" 2>/dev/null || true

    aws events put-targets \
        --rule "$rule_name" \
        --targets "Id=1,Arn=arn:aws:lambda:${REGION}:${ACCOUNT_ID}:function:${FUNCTION_NAME},Input='${input}'" \
        --region "$REGION" >/dev/null
}

# Mondays 14:00 UTC (9 AM CST)
create_or_update_rule "$RULE_EVENTS" \
    "cron(0 14 ? * MON *)" \
    '{"action":"events"}' \
    "Weekly Winnipeg tech events digest"

# 20th of each month, 14:00 UTC
create_or_update_rule "$RULE_POLL" \
    "cron(0 14 20 * ? *)" \
    '{"action":"poll"}' \
    "Monthly meetup day-of-week poll"

echo
echo "== Done =="
echo
echo "Manual invoke:"
echo "  aws lambda invoke --function-name ${FUNCTION_NAME} \\"
echo "      --payload '{\"action\":\"events\"}' --cli-binary-format raw-in-base64-out out.json && cat out.json"
echo
echo "Logs:"
echo "  aws logs tail /aws/lambda/${FUNCTION_NAME} --follow --region ${REGION}"
```

- [ ] **Step 2: Make executable**

```bash
chmod +x deploy/deploy.sh
```

- [ ] **Step 3: Commit (do NOT commit deploy/.env if you create one)**

```bash
git add deploy/deploy.sh
git commit -m "feat(deploy): add idempotent AWS Lambda + EventBridge deploy script"
```

---

## Task 18: Update `.gitignore` and `.dockerignore`

**Files:**
- Modify: `.gitignore`

- [ ] **Step 1: Append the deploy artifacts and `.env`**

Append to `.gitignore` (skip lines that already exist):

```
deploy/.env
deploy/lambda.zip
bootstrap
```

(`.env` is already covered by the existing `.env` rule; deploy artifacts get their own entries for clarity.)

- [ ] **Step 2: Commit**

```bash
git add .gitignore
git commit -m "chore: ignore Lambda deploy artifacts"
```

---

## Task 19: Update `README.md`

**Files:**
- Modify: `README.md`

- [ ] **Step 1: Rewrite the relevant sections**

Read the current README first to preserve project description and feature list. Then replace the entire **Quick Start**, **Installation**, deployment-related, and trigger-related sections with the block below. Keep the project pitch and the data-sources/features sections intact.

```markdown
## Quick Start

### Prerequisites
- Go 1.24+
- AWS CLI v2 (for deployment)
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

- `winnipeg-events-weekly` — Mondays at 14:00 UTC
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

### Configuration

| Env var | Default | Purpose |
|---|---|---|
| `TELEGRAM_BOT_TOKEN` | — | Bot that posts the events digest |
| `TELEGRAM_CHAT_ID` | — | Chat that receives the events digest |
| `TELEGRAM_POLL_BOT_TOKEN` | falls back to `TELEGRAM_BOT_TOKEN` | Bot that posts the monthly poll |
| `TELEGRAM_POLL_CHAT_ID` | falls back to `TELEGRAM_CHAT_ID` | Chat that receives the monthly poll |
| `CITY` | `Winnipeg` | City passed to scrapers |
| `CATEGORIES` | `tech` | Category passed to scrapers |
| `PERIOD_DAYS` | `30` | How many days ahead to scrape |
| `TEST_MODE` | `false` | When `true`, format the message but do not send |
```

- [ ] **Step 2: Commit**

```bash
git add README.md
git commit -m "docs: rewrite README around AWS Lambda deployment"
```

---

## Task 20: End-to-end smoke test (manual, after deploy)

**Files:** none

- [ ] **Step 1: Deploy**

```bash
./deploy/deploy.sh
```
Expected: ends with `== Done ==` and prints invoke + log commands.

- [ ] **Step 2: Verify both EventBridge rules exist**

```bash
aws events list-rules --name-prefix winnipeg- --query 'Rules[].{Name:Name,Sched:ScheduleExpression,State:State}' --output table
```
Expected: two rows for `winnipeg-events-weekly` and `winnipeg-poll-monthly`.

- [ ] **Step 3: Invoke events digest in test mode**

```bash
aws lambda update-function-configuration \
    --function-name winnipeg-tech-events \
    --environment "Variables={TEST_MODE=true,TELEGRAM_BOT_TOKEN=$TELEGRAM_BOT_TOKEN,TELEGRAM_CHAT_ID=$TELEGRAM_CHAT_ID}"
aws lambda invoke --function-name winnipeg-tech-events \
    --payload '{"action":"events"}' \
    --cli-binary-format raw-in-base64-out out.json && cat out.json
```
Expected: `out.json` contains `"success": true`, `"message_sent": false`.

- [ ] **Step 4: Flip off test mode and re-deploy env**

Re-run `./deploy/deploy.sh` with `TEST_MODE=false` in `deploy/.env`. Verify a digest message arrives in the Telegram chat.

- [ ] **Step 5: Tail logs to confirm clean run**

```bash
aws logs tail /aws/lambda/winnipeg-tech-events --since 5m
```
Expected: see `lambda invoked: action="events" ...` and a success line.

---

## Done Criteria (from spec §14)

- [ ] `./deploy/deploy.sh` runs end-to-end, idempotently
- [ ] `aws lambda invoke ... '{"action":"events"}'` posts a Telegram digest
- [ ] `aws lambda invoke ... '{"action":"poll"}'` posts a Telegram poll
- [ ] Both EventBridge rules visible with correct schedules
- [ ] `go build ./...` succeeds; `go vet ./...` clean; `go test ./...` passes
- [ ] No Python Lambda, no GH workflows, no callback voting code
- [ ] No duplicated date helpers or message builders
