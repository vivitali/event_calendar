# Annual Events Monthly Scrape — Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace the hardcoded `annualEvents` slice with a monthly Lambda job that scrapes 4 Winnipeg annual-event sites and persists results in SSM Parameter Store; the weekly digest reads from SSM at runtime.

**Architecture:** New `internal/annual` package owns the `AnnualEvent` type, a `Store` interface (with `SSMStore` impl), a `SiteScraper` interface, 4 site-specific parsers, and an orchestrator that merges fresh scrapes with last-known SSM state (per-event last-known-good on failure). A new Lambda action `scrape_annual` runs monthly via EventBridge; the existing `events` action reads SSM before formatting the digest.

**Tech Stack:** Go 1.24, `github.com/PuerkitoBio/goquery` (existing), `github.com/aws/aws-sdk-go-v2/{config,service/ssm}` (new), AWS Lambda, EventBridge, SSM Parameter Store.

**Spec:** [`docs/superpowers/specs/2026-05-23-annual-events-scrape-design.md`](../specs/2026-05-23-annual-events-scrape-design.md)

---

## File Map

**Create:**
- `internal/annual/annual.go` — `AnnualEvent` struct
- `internal/annual/store.go` — `Store` interface + `SSMStore` impl
- `internal/annual/store_test.go`
- `internal/annual/scraper.go` — `SiteScraper` interface + `Scrape` orchestrator
- `internal/annual/scraper_test.go`
- `internal/annual/prairiedevcon.go`
- `internal/annual/prairiedevcon_test.go`
- `internal/annual/womenhack.go`
- `internal/annual/womenhack_test.go`
- `internal/annual/northforge.go`
- `internal/annual/northforge_test.go`
- `internal/annual/mbtechweek.go`
- `internal/annual/mbtechweek_test.go`
- `internal/annual/testdata/prairiedevcon.html`
- `internal/annual/testdata/womenhack.html`
- `internal/annual/testdata/northforge.html`
- `internal/annual/testdata/mbtechweek.html`
- `internal/jobs/annual.go` — `RunAnnualScrape`
- `internal/jobs/annual_test.go`

**Modify:**
- `internal/digest/annual.go` — delete file (struct moves, hardcoded slice removed)
- `internal/digest/format.go` — `FormatEventsMessage` + `annualFooter` accept `[]annual.AnnualEvent`
- `internal/digest/format_test.go` — pass explicit slice
- `internal/jobs/events.go` — `RunEventsDigest` accepts `[]annual.AnnualEvent`
- `internal/jobs/events_test.go`
- `internal/config/config.go` — add `AnnualSSMParam`
- `internal/config/config_test.go`
- `cmd/lambda/main.go` — new `scrape_annual` case + SSM read on `events`
- `cmd/cli/main.go` — new `-action=scrape_annual` flag
- `deploy/deploy.sh` — IAM policy, SSM seed, EventBridge rule, env var
- `go.mod` / `go.sum` — AWS SDK deps
- `CHANGELOG.md` — release note

---

## Task 1: Create `internal/annual` package with `AnnualEvent` struct

**Files:**
- Create: `internal/annual/annual.go`

- [ ] **Step 1: Create the struct file**

Write `internal/annual/annual.go`:

```go
// Package annual provides scraping and storage of once-a-year Winnipeg tech
// events that we surface in the weekly digest's "save the date" footer.
package annual

import "time"

// AnnualEvent is one curated annual event with its next-occurrence date.
type AnnualEvent struct {
	Name string    `json:"name"`
	Date time.Time `json:"date"`
	URL  string    `json:"url"`
}
```

- [ ] **Step 2: Verify it compiles**

Run: `go build ./internal/annual/...`
Expected: no output, exit 0.

- [ ] **Step 3: Commit**

```bash
git add internal/annual/annual.go
git commit -m "feat(annual): add AnnualEvent type in new package"
```

---

## Task 2: Move `digest.annualFooter` to accept slice arg; delete hardcoded list

This task removes the hardcoded data and updates every caller in one coherent
change so the tree stays green.

**Files:**
- Delete: `internal/digest/annual.go`
- Modify: `internal/digest/format.go` (signatures of `FormatEventsMessage` and `annualFooter`)
- Modify: `internal/digest/format_test.go`
- Modify: `internal/jobs/events.go` (`RunEventsDigest` signature)
- Modify: `internal/jobs/events_test.go`
- Modify: `cmd/lambda/main.go` (pass empty slice for now)
- Modify: `cmd/cli/main.go` (pass empty slice for now)

- [ ] **Step 1: Update the digest test to drive the new signature**

In `internal/digest/format_test.go`, change `TestFormatEventsMessage_HasExpandableAnnualFooter` to pass an explicit annual slice. Replace the whole test with:

```go
func TestFormatEventsMessage_HasExpandableAnnualFooter(t *testing.T) {
	now := time.Date(2026, 1, 7, 12, 0, 0, 0, time.UTC)
	events := []models.Event{
		{Name: "X", URL: "https://e/x", Source: "meetup", StartTime: now.Add(24 * time.Hour)},
	}
	annualEvts := []annual.AnnualEvent{
		{Name: "Prairie Dev Con", Date: time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC), URL: "https://www.prairiedevcon.com/"},
	}
	msg := FormatEventsMessage(events, annualEvts, now)
	if !strings.Contains(msg, "**>📌 Save the date") {
		t.Errorf("missing expandable blockquote opener:\n%s", msg)
	}
	if !strings.HasSuffix(strings.TrimSpace(strings.Split(msg, "\n_Shared")[0]), "||") {
		t.Errorf("expandable blockquote should end with || before footer:\n%s", msg)
	}
}
```

Add the import:

```go
import (
	"strings"
	"testing"
	"time"

	"event_calendar/internal/annual"
	"event_calendar/internal/models"
)
```

Also update every other call to `FormatEventsMessage` in that file to pass `nil` (or `[]annual.AnnualEvent{}`) as the new second argument. The three other tests (`_Empty`, `_GroupsAndHeaders`, `_DropsGenericVenue`, `_RendersOnlineVenue`, `_TruncatesLong`) should all pass `nil` as the annual slice.

- [ ] **Step 2: Run tests to verify they fail to compile**

Run: `go test ./internal/digest/...`
Expected: compile error — `FormatEventsMessage` arity mismatch / undefined `annual` package usage.

- [ ] **Step 3: Update `FormatEventsMessage` signature**

In `internal/digest/format.go`:

Add import:
```go
import (
	// ...existing...
	"event_calendar/internal/annual"
)
```

Change function signature:
```go
func FormatEventsMessage(events []models.Event, annualEvts []annual.AnnualEvent, now time.Time) string {
```

Change the trailer-building block from `if af := annualFooter(now); af != "" {` to:
```go
if af := annualFooter(annualEvts, now); af != "" {
```

Change `annualFooter` to:
```go
// annualFooter renders future annual events as an expandable MarkdownV2 quote.
// Returns "" when there's nothing to show.
func annualFooter(events []annual.AnnualEvent, now time.Time) string {
	upcoming := futureOnly(events, now)
	if len(upcoming) == 0 {
		return ""
	}
	var sb strings.Builder
	sb.WriteString("**>📌 Save the date — annual events\n")
	for i, e := range upcoming {
		line := fmt.Sprintf(">• %s — %s [info](%s)",
			escMD(e.Name), escMD(e.Date.Format("Jan 2, 2006")), escMDURL(e.URL))
		if i == len(upcoming)-1 {
			line += "||"
		}
		sb.WriteString(line + "\n")
	}
	return sb.String()
}

// futureOnly returns annual events whose date is strictly after now,
// preserving caller-supplied order.
func futureOnly(events []annual.AnnualEvent, now time.Time) []annual.AnnualEvent {
	out := make([]annual.AnnualEvent, 0, len(events))
	for _, e := range events {
		if e.Date.After(now) {
			out = append(out, e)
		}
	}
	return out
}
```

- [ ] **Step 4: Delete the old hardcoded data file**

Run:
```bash
rm internal/digest/annual.go
```

- [ ] **Step 5: Update `jobs.RunEventsDigest` signature**

In `internal/jobs/events.go`:

Add import:
```go
import (
	// ...existing...
	"event_calendar/internal/annual"
)
```

Change function signature:
```go
func RunEventsDigest(deps EventsDeps, cfg config.Config, annualEvts []annual.AnnualEvent) Result {
```

Change the `digest.FormatEventsMessage(future, deps.Now)` call to:
```go
message := digest.FormatEventsMessage(future, annualEvts, deps.Now)
```

- [ ] **Step 6: Update `internal/jobs/events_test.go`**

Every call to `jobs.RunEventsDigest(deps, cfg)` becomes `jobs.RunEventsDigest(deps, cfg, nil)`. No other changes needed in tests.

- [ ] **Step 7: Update `cmd/lambda/main.go`**

In `runEvents`, change the return line to:
```go
return jobs.RunEventsDigest(jobs.EventsDeps{
    Scraper: scraper,
    Sender:  sender,
    Now:     time.Now(),
}, cfg, nil)
```

(SSM read comes in Task 9. For now pass `nil` so the build stays green.)

- [ ] **Step 8: Update `cmd/cli/main.go`**

In `runEvents`, change the return to:
```go
return jobs.RunEventsDigest(jobs.EventsDeps{Scraper: scraper, Sender: sender, Now: time.Now()}, cfg, nil)
```

- [ ] **Step 9: Run all tests**

Run: `go test ./...`
Expected: all green.

- [ ] **Step 10: Commit**

```bash
git add -A
git commit -m "refactor(digest): inject annual events instead of hardcoded slice

Delete internal/digest/annual.go; FormatEventsMessage and RunEventsDigest
now take []annual.AnnualEvent. All callers pass nil for now; the SSM-backed
store is wired in a later task."
```

---

## Task 3: `Store` interface + `SSMStore` skeleton (no SDK yet)

We add the interface and a fake-friendly shape first. Real SDK wiring comes
after deps are added (Task 11). This task defines `Store` and a thin SSMStore
that takes a tiny `ssmAPI` interface, so unit tests can drive it without AWS.

**Files:**
- Create: `internal/annual/store.go`
- Create: `internal/annual/store_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/annual/store_test.go`:

```go
package annual

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

type fakeSSM struct {
	getValue string
	getErr   error
	putValue string
	putErr   error
}

func (f *fakeSSM) GetParameter(ctx context.Context, name string) (string, error) {
	return f.getValue, f.getErr
}

func (f *fakeSSM) PutParameter(ctx context.Context, name, value string) error {
	f.putValue = value
	return f.putErr
}

func TestSSMStore_Get_ReturnsParsedEvents(t *testing.T) {
	want := []AnnualEvent{
		{Name: "X", Date: time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC), URL: "https://x"},
	}
	raw, _ := json.Marshal(want)
	store := &SSMStore{api: &fakeSSM{getValue: string(raw)}, name: "/p"}
	got, err := store.Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(got) != 1 || got[0].Name != "X" {
		t.Errorf("unexpected events: %+v", got)
	}
}

func TestSSMStore_Get_ReturnsEmptyOnNotFound(t *testing.T) {
	store := &SSMStore{api: &fakeSSM{getErr: ErrParameterNotFound}, name: "/p"}
	got, err := store.Get(context.Background())
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("expected empty slice, got %+v", got)
	}
}

func TestSSMStore_Get_PropagatesOtherErrors(t *testing.T) {
	want := errors.New("boom")
	store := &SSMStore{api: &fakeSSM{getErr: want}, name: "/p"}
	_, err := store.Get(context.Background())
	if !errors.Is(err, want) {
		t.Errorf("expected wrapped %v, got %v", want, err)
	}
}

func TestSSMStore_Put_MarshalsJSON(t *testing.T) {
	api := &fakeSSM{}
	store := &SSMStore{api: api, name: "/p"}
	events := []AnnualEvent{{Name: "Y", Date: time.Date(2026, 6, 11, 19, 0, 0, 0, time.UTC), URL: "https://y"}}
	if err := store.Put(context.Background(), events); err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	var round []AnnualEvent
	if err := json.Unmarshal([]byte(api.putValue), &round); err != nil {
		t.Fatalf("put value is not JSON: %v (%q)", err, api.putValue)
	}
	if len(round) != 1 || round[0].Name != "Y" {
		t.Errorf("unexpected round-trip: %+v", round)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/annual/...`
Expected: compile error — `SSMStore`, `ssmAPI`, `ErrParameterNotFound` not defined.

- [ ] **Step 3: Implement `internal/annual/store.go`**

```go
package annual

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
)

// ErrParameterNotFound is returned by ssmAPI.GetParameter when the parameter
// does not exist. SSMStore.Get translates this to an empty slice.
var ErrParameterNotFound = errors.New("parameter not found")

// Store persists the curated annual-events list between Lambda invocations.
type Store interface {
	Get(ctx context.Context) ([]AnnualEvent, error)
	Put(ctx context.Context, events []AnnualEvent) error
}

// ssmAPI is the slice of the AWS SSM client we need. Lets tests substitute
// a fake without pulling in the SDK.
type ssmAPI interface {
	GetParameter(ctx context.Context, name string) (string, error)
	PutParameter(ctx context.Context, name, value string) error
}

// SSMStore is a Store backed by AWS SSM Parameter Store.
type SSMStore struct {
	api  ssmAPI
	name string
}

// NewSSMStore returns a store that reads/writes the named parameter.
func NewSSMStore(api ssmAPI, name string) *SSMStore {
	return &SSMStore{api: api, name: name}
}

// Get returns the parsed list. Missing parameter → empty slice, nil error.
func (s *SSMStore) Get(ctx context.Context) ([]AnnualEvent, error) {
	raw, err := s.api.GetParameter(ctx, s.name)
	if err != nil {
		if errors.Is(err, ErrParameterNotFound) {
			return []AnnualEvent{}, nil
		}
		return nil, fmt.Errorf("ssm get %s: %w", s.name, err)
	}
	if raw == "" {
		return []AnnualEvent{}, nil
	}
	var out []AnnualEvent
	if err := json.Unmarshal([]byte(raw), &out); err != nil {
		return nil, fmt.Errorf("ssm parse %s: %w", s.name, err)
	}
	return out, nil
}

// Put serialises events to JSON and writes the parameter (overwriting).
func (s *SSMStore) Put(ctx context.Context, events []AnnualEvent) error {
	if events == nil {
		events = []AnnualEvent{}
	}
	raw, err := json.Marshal(events)
	if err != nil {
		return fmt.Errorf("ssm marshal: %w", err)
	}
	if err := s.api.PutParameter(ctx, s.name, string(raw)); err != nil {
		return fmt.Errorf("ssm put %s: %w", s.name, err)
	}
	return nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/annual/...`
Expected: all 4 tests pass.

- [ ] **Step 5: Commit**

```bash
git add internal/annual/store.go internal/annual/store_test.go
git commit -m "feat(annual): Store interface + SSMStore with fake-friendly ssmAPI"
```

---

## Task 4: `SiteScraper` interface + `Scrape` orchestrator

**Files:**
- Create: `internal/annual/scraper.go`
- Create: `internal/annual/scraper_test.go`

- [ ] **Step 1: Write the failing tests**

Create `internal/annual/scraper_test.go`:

```go
package annual

import (
	"context"
	"errors"
	"testing"
	"time"
)

type fakeScraper struct {
	name string
	url  string
	out  AnnualEvent
	err  error
}

func (f *fakeScraper) Name() string { return f.name }
func (f *fakeScraper) URL() string  { return f.url }
func (f *fakeScraper) Scrape(ctx context.Context) (AnnualEvent, error) {
	return f.out, f.err
}

type memStore struct {
	data []AnnualEvent
	get  func() ([]AnnualEvent, error)
}

func (m *memStore) Get(ctx context.Context) ([]AnnualEvent, error) {
	if m.get != nil {
		return m.get()
	}
	return m.data, nil
}
func (m *memStore) Put(ctx context.Context, events []AnnualEvent) error {
	m.data = events
	return nil
}

func TestScrape_WritesFreshResults(t *testing.T) {
	now := time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC)
	store := &memStore{}
	scrapers := []SiteScraper{
		&fakeScraper{name: "A", url: "https://a", out: AnnualEvent{Name: "A", Date: now.AddDate(0, 1, 0), URL: "https://a"}},
		&fakeScraper{name: "B", url: "https://b", out: AnnualEvent{Name: "B", Date: now.AddDate(0, 2, 0), URL: "https://b"}},
	}
	n, err := Scrape(context.Background(), scrapers, store, now)
	if err != nil {
		t.Fatalf("unexpected err: %v", err)
	}
	if n != 2 {
		t.Errorf("want 2 events, got %d", n)
	}
	if len(store.data) != 2 {
		t.Fatalf("store should have 2, has %d", len(store.data))
	}
	if !store.data[0].Date.Before(store.data[1].Date) {
		t.Errorf("expected sorted by date asc, got %+v", store.data)
	}
}

func TestScrape_KeepsLastKnownOnError(t *testing.T) {
	now := time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC)
	old := AnnualEvent{Name: "A", Date: now.AddDate(0, 4, 0), URL: "https://a-old"}
	store := &memStore{data: []AnnualEvent{old}}
	scrapers := []SiteScraper{
		&fakeScraper{name: "A", url: "https://a", err: errors.New("boom")},
	}
	n, err := Scrape(context.Background(), scrapers, store, now)
	if err != nil {
		t.Fatalf("orchestrator should not return error, got %v", err)
	}
	if n != 1 {
		t.Errorf("want 1 event retained, got %d", n)
	}
	if len(store.data) != 1 || store.data[0].URL != "https://a-old" {
		t.Errorf("last-known not retained: %+v", store.data)
	}
}

func TestScrape_DropsPastEvents(t *testing.T) {
	now := time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC)
	store := &memStore{}
	scrapers := []SiteScraper{
		&fakeScraper{name: "A", url: "https://a", out: AnnualEvent{Name: "A", Date: now.AddDate(0, -1, 0), URL: "https://a"}},
		&fakeScraper{name: "B", url: "https://b", out: AnnualEvent{Name: "B", Date: now.AddDate(0, 1, 0), URL: "https://b"}},
	}
	n, _ := Scrape(context.Background(), scrapers, store, now)
	if n != 1 {
		t.Errorf("want 1 future event, got %d", n)
	}
	if len(store.data) != 1 || store.data[0].Name != "B" {
		t.Errorf("expected only B retained, got %+v", store.data)
	}
}

func TestScrape_EmptyBaselineNoFreshResults(t *testing.T) {
	now := time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC)
	store := &memStore{}
	scrapers := []SiteScraper{
		&fakeScraper{name: "A", url: "https://a", err: errors.New("boom")},
	}
	n, _ := Scrape(context.Background(), scrapers, store, now)
	if n != 0 {
		t.Errorf("want 0 events, got %d", n)
	}
	if len(store.data) != 0 {
		t.Errorf("store should remain empty, got %+v", store.data)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/annual/...`
Expected: compile error — `SiteScraper`, `Scrape` not defined.

- [ ] **Step 3: Implement `internal/annual/scraper.go`**

```go
package annual

import (
	"context"
	"log"
	"sort"
	"sync"
	"time"
)

// SiteScraper fetches the next occurrence of one annual event from its
// canonical website.
type SiteScraper interface {
	// Name is a stable identifier used as the merge key against the baseline.
	Name() string
	// URL is the canonical event URL, surfaced in the digest footer.
	URL() string
	// Scrape fetches and parses the event's next-occurrence date.
	Scrape(ctx context.Context) (AnnualEvent, error)
}

// Scrape runs every scraper, merges results with the baseline from store
// (per-event last-known-good on error), drops past events, sorts by date,
// and writes back. Returns the number of events stored.
//
// Errors are intentionally swallowed inside the loop and logged; only a
// store read/write failure is returned. The weekly digest tolerates an
// empty SSM read, so a single failing scrape never breaks the user-facing
// path.
func Scrape(ctx context.Context, scrapers []SiteScraper, store Store, now time.Time) (int, error) {
	baseline, err := store.Get(ctx)
	if err != nil {
		log.Printf("annual: store.Get failed, starting from empty baseline: %v", err)
		baseline = nil
	}

	merged := make(map[string]AnnualEvent, len(scrapers)+len(baseline))
	for _, e := range baseline {
		merged[e.Name] = e
	}

	var mu sync.Mutex
	var wg sync.WaitGroup
	for _, s := range scrapers {
		wg.Add(1)
		go func(s SiteScraper) {
			defer wg.Done()
			evt, err := s.Scrape(ctx)
			if err != nil {
				log.Printf("annual: scrape %s failed (retaining last-known): %v", s.Name(), err)
				return
			}
			mu.Lock()
			merged[s.Name()] = evt
			mu.Unlock()
		}(s)
	}
	wg.Wait()

	out := make([]AnnualEvent, 0, len(merged))
	for _, e := range merged {
		if e.Date.After(now) {
			out = append(out, e)
		}
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Date.Before(out[j].Date) })

	if err := store.Put(ctx, out); err != nil {
		return 0, err
	}
	return len(out), nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/annual/...`
Expected: all tests pass (existing store tests + 4 new orchestrator tests).

- [ ] **Step 5: Commit**

```bash
git add internal/annual/scraper.go internal/annual/scraper_test.go
git commit -m "feat(annual): SiteScraper interface + Scrape orchestrator

Per-event last-known-good on failure, drops past events, sorts by date asc."
```

---

## Task 5: Prairie Dev Con scraper

The site URL is `https://www.prairiedevcon.com/`. The homepage prominently
displays the conference dates (e.g. "September 21-22 2026"). We extract the
first date.

**Files:**
- Create: `internal/annual/testdata/prairiedevcon.html`
- Create: `internal/annual/prairiedevcon.go`
- Create: `internal/annual/prairiedevcon_test.go`

- [ ] **Step 1: Capture the fixture**

Run:
```bash
curl -sSL -A "Mozilla/5.0" https://www.prairiedevcon.com/ \
    -o internal/annual/testdata/prairiedevcon.html
```

Inspect the file for the date string. Expected pattern: one of "September 21-22 2026", "Sep 21, 2026", or similar. Note the exact text — you'll target it in the parser.

- [ ] **Step 2: Write the failing test**

Create `internal/annual/prairiedevcon_test.go`:

```go
package annual

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestPrairieDevCon_Scrape(t *testing.T) {
	html, err := os.ReadFile("testdata/prairiedevcon.html")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write(html)
	}))
	defer srv.Close()

	s := newPrairieDevConWith(srv.URL)
	got, err := s.Scrape(context.Background())
	if err != nil {
		t.Fatalf("scrape: %v", err)
	}
	if got.Name != "Prairie Dev Con Winnipeg" {
		t.Errorf("name = %q", got.Name)
	}
	if got.URL != "https://www.prairiedevcon.com/" {
		t.Errorf("url = %q", got.URL)
	}
	// Verify the parsed date matches what you observed in the fixture.
	// Update this if the fixture you captured shows a different date.
	want := time.Date(2026, 9, 21, 0, 0, 0, 0, time.UTC)
	if !got.Date.Equal(want) {
		t.Errorf("date = %v, want %v", got.Date, want)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/annual/ -run TestPrairieDevCon_Scrape -v`
Expected: compile error — `newPrairieDevConWith` not defined.

- [ ] **Step 4: Implement the scraper**

Create `internal/annual/prairiedevcon.go`:

```go
package annual

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
	"time"
)

const prairieDevConURL = "https://www.prairiedevcon.com/"

// PrairieDevCon scrapes the next conference date from prairiedevcon.com.
type PrairieDevCon struct {
	client  *http.Client
	fetchURL string
}

// NewPrairieDevCon returns a scraper using the canonical URL.
func NewPrairieDevCon() *PrairieDevCon {
	return newPrairieDevConWith(prairieDevConURL)
}

func newPrairieDevConWith(fetchURL string) *PrairieDevCon {
	return &PrairieDevCon{
		client:   &http.Client{Timeout: 30 * time.Second},
		fetchURL: fetchURL,
	}
}

func (p *PrairieDevCon) Name() string { return "Prairie Dev Con Winnipeg" }
func (p *PrairieDevCon) URL() string  { return prairieDevConURL }

func (p *PrairieDevCon) Scrape(ctx context.Context) (AnnualEvent, error) {
	html, err := fetch(ctx, p.client, p.fetchURL)
	if err != nil {
		return AnnualEvent{}, err
	}
	d, err := parsePrairieDevConDate(html)
	if err != nil {
		return AnnualEvent{}, err
	}
	return AnnualEvent{Name: p.Name(), URL: prairieDevConURL, Date: d}, nil
}

// parsePrairieDevConDate looks for "Month D[-D2][,] YYYY" in the homepage
// text. Example matches:
//   "September 21-22 2026"
//   "September 21, 2026"
//   "Sept 21 2026"
var pdcDateRe = regexp.MustCompile(
	`(?i)(January|February|March|April|May|June|July|August|September|October|November|December|Sept?|Jan|Feb|Mar|Apr|Jun|Jul|Aug|Oct|Nov|Dec)\s+(\d{1,2})(?:\s*[-–]\s*\d{1,2})?[,\s]+\s*(20\d{2})`,
)

func parsePrairieDevConDate(html string) (time.Time, error) {
	m := pdcDateRe.FindStringSubmatch(html)
	if m == nil {
		return time.Time{}, fmt.Errorf("prairiedevcon: no date found")
	}
	monthStr, dayStr, yearStr := m[1], m[2], m[3]
	month, ok := monthFromString(monthStr)
	if !ok {
		return time.Time{}, fmt.Errorf("prairiedevcon: unknown month %q", monthStr)
	}
	var day, year int
	fmt.Sscanf(dayStr, "%d", &day)
	fmt.Sscanf(yearStr, "%d", &year)
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC), nil
}

// monthFromString maps full and short month names (case-insensitive) to time.Month.
func monthFromString(s string) (time.Month, bool) {
	switch strings.ToLower(s) {
	case "january", "jan":
		return time.January, true
	case "february", "feb":
		return time.February, true
	case "march", "mar":
		return time.March, true
	case "april", "apr":
		return time.April, true
	case "may":
		return time.May, true
	case "june", "jun":
		return time.June, true
	case "july", "jul":
		return time.July, true
	case "august", "aug":
		return time.August, true
	case "september", "sept", "sep":
		return time.September, true
	case "october", "oct":
		return time.October, true
	case "november", "nov":
		return time.November, true
	case "december", "dec":
		return time.December, true
	}
	return 0, false
}

// fetch is a shared GET-and-read used by all site scrapers in this package.
func fetch(ctx context.Context, c *http.Client, url string) (string, error) {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return "", err
	}
	req.Header.Set("User-Agent", "Mozilla/5.0 (compatible; WinnipegTechEventsBot/1.0)")
	resp, err := c.Do(req)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return "", fmt.Errorf("GET %s: status %d", url, resp.StatusCode)
	}
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	return string(body), nil
}
```

- [ ] **Step 5: Run test to verify it passes**

Run: `go test ./internal/annual/ -run TestPrairieDevCon_Scrape -v`
Expected: PASS. If the parsed date doesn't match the `want` in the test, update the test to match what the fixture actually contains, then commit. The regex is intentionally permissive; if it doesn't match the fixture text at all, inspect and broaden it.

- [ ] **Step 6: Run all package tests**

Run: `go test ./internal/annual/...`
Expected: all green.

- [ ] **Step 7: Commit**

```bash
git add internal/annual/prairiedevcon.go internal/annual/prairiedevcon_test.go internal/annual/testdata/prairiedevcon.html
git commit -m "feat(annual): Prairie Dev Con homepage date scraper"
```

---

## Task 6: WomenHack scraper

Site: `https://womenhack.com/cities/winnipeg/`. Page lists upcoming Winnipeg
events with a date heading (e.g. "June 11, 2026 - 7:00 pm").

**Files:**
- Create: `internal/annual/testdata/womenhack.html`
- Create: `internal/annual/womenhack.go`
- Create: `internal/annual/womenhack_test.go`

- [ ] **Step 1: Capture the fixture**

```bash
curl -sSL -A "Mozilla/5.0" https://womenhack.com/cities/winnipeg/ \
    -o internal/annual/testdata/womenhack.html
```

Inspect for the next event date. WomenHack typically renders as
"Month D, YYYY - H:MM pm" near the event listing.

- [ ] **Step 2: Write the failing test**

Create `internal/annual/womenhack_test.go`:

```go
package annual

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestWomenHack_Scrape(t *testing.T) {
	html, err := os.ReadFile("testdata/womenhack.html")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write(html)
	}))
	defer srv.Close()

	s := newWomenHackWith(srv.URL)
	got, err := s.Scrape(context.Background())
	if err != nil {
		t.Fatalf("scrape: %v", err)
	}
	if got.Name != "WomenHack Winnipeg" {
		t.Errorf("name = %q", got.Name)
	}
	if got.URL != "https://womenhack.com/cities/winnipeg/" {
		t.Errorf("url = %q", got.URL)
	}
	// Adjust to match whatever next-occurrence date the captured fixture shows.
	want := time.Date(2026, 6, 11, 0, 0, 0, 0, time.UTC)
	if !got.Date.Equal(want) {
		t.Errorf("date = %v, want %v", got.Date, want)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/annual/ -run TestWomenHack_Scrape -v`
Expected: compile error.

- [ ] **Step 4: Implement the scraper**

Create `internal/annual/womenhack.go`:

```go
package annual

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"time"
)

const womenHackURL = "https://womenhack.com/cities/winnipeg/"

// WomenHack scrapes the next Winnipeg WomenHack event date.
type WomenHack struct {
	client   *http.Client
	fetchURL string
}

func NewWomenHack() *WomenHack { return newWomenHackWith(womenHackURL) }

func newWomenHackWith(fetchURL string) *WomenHack {
	return &WomenHack{
		client:   &http.Client{Timeout: 30 * time.Second},
		fetchURL: fetchURL,
	}
}

func (w *WomenHack) Name() string { return "WomenHack Winnipeg" }
func (w *WomenHack) URL() string  { return womenHackURL }

func (w *WomenHack) Scrape(ctx context.Context) (AnnualEvent, error) {
	html, err := fetch(ctx, w.client, w.fetchURL)
	if err != nil {
		return AnnualEvent{}, err
	}
	d, err := parseWomenHackDate(html)
	if err != nil {
		return AnnualEvent{}, err
	}
	return AnnualEvent{Name: w.Name(), URL: womenHackURL, Date: d}, nil
}

// parseWomenHackDate matches "Month D, YYYY" (the listing format used on
// womenhack.com city pages). Picks the first match — listings are
// chronological, oldest-first.
var whDateRe = regexp.MustCompile(
	`(?i)(January|February|March|April|May|June|July|August|September|October|November|December)\s+(\d{1,2}),\s+(20\d{2})`,
)

func parseWomenHackDate(html string) (time.Time, error) {
	m := whDateRe.FindStringSubmatch(html)
	if m == nil {
		return time.Time{}, fmt.Errorf("womenhack: no date found")
	}
	month, ok := monthFromString(m[1])
	if !ok {
		return time.Time{}, fmt.Errorf("womenhack: unknown month %q", m[1])
	}
	var day, year int
	fmt.Sscanf(m[2], "%d", &day)
	fmt.Sscanf(m[3], "%d", &year)
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC), nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/annual/...`
Expected: all green. Adjust the `want` date in the test if the captured fixture differs.

- [ ] **Step 6: Commit**

```bash
git add internal/annual/womenhack.go internal/annual/womenhack_test.go internal/annual/testdata/womenhack.html
git commit -m "feat(annual): WomenHack Winnipeg next-event date scraper"
```

---

## Task 7: North Forge RampUp scraper

Site: `https://www.northforge.ca/rampup`. The RampUp Weekend page lists the
weekend dates (e.g. "April 10–12, 2026").

**Files:**
- Create: `internal/annual/testdata/northforge.html`
- Create: `internal/annual/northforge.go`
- Create: `internal/annual/northforge_test.go`

- [ ] **Step 1: Capture the fixture**

```bash
curl -sSL -A "Mozilla/5.0" https://www.northforge.ca/rampup \
    -o internal/annual/testdata/northforge.html
```

Inspect for date pattern. Likely "April 10-12, 2026" or "April 10, 2026".

- [ ] **Step 2: Write the failing test**

Create `internal/annual/northforge_test.go`:

```go
package annual

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestNorthForge_Scrape(t *testing.T) {
	html, err := os.ReadFile("testdata/northforge.html")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write(html)
	}))
	defer srv.Close()

	s := newNorthForgeWith(srv.URL)
	got, err := s.Scrape(context.Background())
	if err != nil {
		t.Fatalf("scrape: %v", err)
	}
	if got.Name != "North Forge RampUp Weekend" {
		t.Errorf("name = %q", got.Name)
	}
	if got.URL != "https://www.northforge.ca/rampup" {
		t.Errorf("url = %q", got.URL)
	}
	// Adjust if the captured fixture shows a different next date.
	want := time.Date(2026, 4, 10, 0, 0, 0, 0, time.UTC)
	if !got.Date.Equal(want) {
		t.Errorf("date = %v, want %v", got.Date, want)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/annual/ -run TestNorthForge_Scrape -v`
Expected: compile error.

- [ ] **Step 4: Implement the scraper**

Create `internal/annual/northforge.go`:

```go
package annual

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"time"
)

const northForgeURL = "https://www.northforge.ca/rampup"

// NorthForge scrapes the next RampUp Weekend date.
type NorthForge struct {
	client   *http.Client
	fetchURL string
}

func NewNorthForge() *NorthForge { return newNorthForgeWith(northForgeURL) }

func newNorthForgeWith(fetchURL string) *NorthForge {
	return &NorthForge{
		client:   &http.Client{Timeout: 30 * time.Second},
		fetchURL: fetchURL,
	}
}

func (n *NorthForge) Name() string { return "North Forge RampUp Weekend" }
func (n *NorthForge) URL() string  { return northForgeURL }

func (n *NorthForge) Scrape(ctx context.Context) (AnnualEvent, error) {
	html, err := fetch(ctx, n.client, n.fetchURL)
	if err != nil {
		return AnnualEvent{}, err
	}
	d, err := parseNorthForgeDate(html)
	if err != nil {
		return AnnualEvent{}, err
	}
	return AnnualEvent{Name: n.Name(), URL: northForgeURL, Date: d}, nil
}

// parseNorthForgeDate matches "Month D[-D2], YYYY" with optional en/em-dash.
var nfDateRe = regexp.MustCompile(
	`(?i)(January|February|March|April|May|June|July|August|September|October|November|December)\s+(\d{1,2})(?:\s*[-–—]\s*\d{1,2})?,\s+(20\d{2})`,
)

func parseNorthForgeDate(html string) (time.Time, error) {
	m := nfDateRe.FindStringSubmatch(html)
	if m == nil {
		return time.Time{}, fmt.Errorf("northforge: no date found")
	}
	month, ok := monthFromString(m[1])
	if !ok {
		return time.Time{}, fmt.Errorf("northforge: unknown month %q", m[1])
	}
	var day, year int
	fmt.Sscanf(m[2], "%d", &day)
	fmt.Sscanf(m[3], "%d", &year)
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC), nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/annual/...`
Expected: all green. Adjust `want` in the test if the fixture shows a different date.

- [ ] **Step 6: Commit**

```bash
git add internal/annual/northforge.go internal/annual/northforge_test.go internal/annual/testdata/northforge.html
git commit -m "feat(annual): North Forge RampUp Weekend date scraper"
```

---

## Task 8: MbTech Week scraper

Site: `https://mbtechweek.ca/`. Date format typically
"February 22-28, 2026" or "February 22, 2026" on the landing page.

**Files:**
- Create: `internal/annual/testdata/mbtechweek.html`
- Create: `internal/annual/mbtechweek.go`
- Create: `internal/annual/mbtechweek_test.go`

- [ ] **Step 1: Capture the fixture**

```bash
curl -sSL -A "Mozilla/5.0" https://mbtechweek.ca/ \
    -o internal/annual/testdata/mbtechweek.html
```

- [ ] **Step 2: Write the failing test**

Create `internal/annual/mbtechweek_test.go`:

```go
package annual

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"
)

func TestMbTechWeek_Scrape(t *testing.T) {
	html, err := os.ReadFile("testdata/mbtechweek.html")
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write(html)
	}))
	defer srv.Close()

	s := newMbTechWeekWith(srv.URL)
	got, err := s.Scrape(context.Background())
	if err != nil {
		t.Fatalf("scrape: %v", err)
	}
	if got.Name != "MbTech Week" {
		t.Errorf("name = %q", got.Name)
	}
	if got.URL != "https://mbtechweek.ca/" {
		t.Errorf("url = %q", got.URL)
	}
	// Adjust if the captured fixture shows a different date.
	want := time.Date(2026, 2, 22, 0, 0, 0, 0, time.UTC)
	if !got.Date.Equal(want) {
		t.Errorf("date = %v, want %v", got.Date, want)
	}
}
```

- [ ] **Step 3: Run test to verify it fails**

Run: `go test ./internal/annual/ -run TestMbTechWeek_Scrape -v`
Expected: compile error.

- [ ] **Step 4: Implement the scraper**

Create `internal/annual/mbtechweek.go`:

```go
package annual

import (
	"context"
	"fmt"
	"net/http"
	"regexp"
	"time"
)

const mbTechWeekURL = "https://mbtechweek.ca/"

// MbTechWeek scrapes the next MbTech Week date from mbtechweek.ca.
type MbTechWeek struct {
	client   *http.Client
	fetchURL string
}

func NewMbTechWeek() *MbTechWeek { return newMbTechWeekWith(mbTechWeekURL) }

func newMbTechWeekWith(fetchURL string) *MbTechWeek {
	return &MbTechWeek{
		client:   &http.Client{Timeout: 30 * time.Second},
		fetchURL: fetchURL,
	}
}

func (m *MbTechWeek) Name() string { return "MbTech Week" }
func (m *MbTechWeek) URL() string  { return mbTechWeekURL }

func (m *MbTechWeek) Scrape(ctx context.Context) (AnnualEvent, error) {
	html, err := fetch(ctx, m.client, m.fetchURL)
	if err != nil {
		return AnnualEvent{}, err
	}
	d, err := parseMbTechWeekDate(html)
	if err != nil {
		return AnnualEvent{}, err
	}
	return AnnualEvent{Name: m.Name(), URL: mbTechWeekURL, Date: d}, nil
}

// parseMbTechWeekDate matches "Month D[-D2], YYYY" with optional dash.
var mtwDateRe = regexp.MustCompile(
	`(?i)(January|February|March|April|May|June|July|August|September|October|November|December)\s+(\d{1,2})(?:\s*[-–—]\s*\d{1,2})?,\s+(20\d{2})`,
)

func parseMbTechWeekDate(html string) (time.Time, error) {
	m := mtwDateRe.FindStringSubmatch(html)
	if m == nil {
		return time.Time{}, fmt.Errorf("mbtechweek: no date found")
	}
	month, ok := monthFromString(m[1])
	if !ok {
		return time.Time{}, fmt.Errorf("mbtechweek: unknown month %q", m[1])
	}
	var day, year int
	fmt.Sscanf(m[2], "%d", &day)
	fmt.Sscanf(m[3], "%d", &year)
	return time.Date(year, month, day, 0, 0, 0, 0, time.UTC), nil
}
```

- [ ] **Step 5: Run tests to verify they pass**

Run: `go test ./internal/annual/...`
Expected: all green. Adjust the test `want` to match the captured fixture if needed.

- [ ] **Step 6: Commit**

```bash
git add internal/annual/mbtechweek.go internal/annual/mbtechweek_test.go internal/annual/testdata/mbtechweek.html
git commit -m "feat(annual): MbTech Week date scraper"
```

---

## Task 9: Add `AnnualSSMParam` to config + tests

**Files:**
- Modify: `internal/config/config.go`
- Modify: `internal/config/config_test.go`

- [ ] **Step 1: Update the failing test**

Open `internal/config/config_test.go`. Find the existing default-values test (line ~18 — `if cfg.City != "Winnipeg" || cfg.Categories != "tech" || cfg.PeriodDays != 30`) and extend it to assert the new default:

```go
if cfg.AnnualSSMParam != "/winnipeg-tech-events/annual-events" {
    t.Errorf("AnnualSSMParam default = %q", cfg.AnnualSSMParam)
}
```

Add a second test for the env override:

```go
func TestLoad_AnnualSSMParamOverride(t *testing.T) {
    t.Setenv("ANNUAL_SSM_PARAM", "/custom/path")
    cfg := Load()
    if cfg.AnnualSSMParam != "/custom/path" {
        t.Errorf("AnnualSSMParam = %q, want /custom/path", cfg.AnnualSSMParam)
    }
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/config/...`
Expected: failure — `cfg.AnnualSSMParam` undefined.

- [ ] **Step 3: Implement the config field**

In `internal/config/config.go`:

Add the field to the `Config` struct:
```go
type Config struct {
    BotToken        string
    ChatID          string
    PollBotToken    string
    PollChatID      string
    City            string
    Categories      string
    PeriodDays      int
    TestMode        bool
    AnnualSSMParam  string
}
```

Add the assignment inside `Load`:
```go
cfg := Config{
    // ...existing fields...
    AnnualSSMParam: envOr("ANNUAL_SSM_PARAM", "/winnipeg-tech-events/annual-events"),
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/config/...`
Expected: all green.

- [ ] **Step 5: Commit**

```bash
git add internal/config/
git commit -m "feat(config): add AnnualSSMParam env config"
```

---

## Task 10: `jobs.RunAnnualScrape` + tests

**Files:**
- Create: `internal/jobs/annual.go`
- Create: `internal/jobs/annual_test.go`

- [ ] **Step 1: Write the failing test**

Create `internal/jobs/annual_test.go`:

```go
package jobs

import (
	"context"
	"errors"
	"testing"
	"time"

	"event_calendar/internal/annual"
)

type fakeStore struct {
	events []annual.AnnualEvent
	putErr error
}

func (f *fakeStore) Get(ctx context.Context) ([]annual.AnnualEvent, error) {
	return f.events, nil
}
func (f *fakeStore) Put(ctx context.Context, e []annual.AnnualEvent) error {
	f.events = e
	return f.putErr
}

type fakeAnnualScraper struct {
	name string
	out  annual.AnnualEvent
	err  error
}

func (f *fakeAnnualScraper) Name() string { return f.name }
func (f *fakeAnnualScraper) URL() string  { return "https://x" }
func (f *fakeAnnualScraper) Scrape(ctx context.Context) (annual.AnnualEvent, error) {
	return f.out, f.err
}

func TestRunAnnualScrape_Success(t *testing.T) {
	now := time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC)
	store := &fakeStore{}
	deps := AnnualScrapeDeps{
		Store: store,
		Scrapers: []annual.SiteScraper{
			&fakeAnnualScraper{name: "A", out: annual.AnnualEvent{Name: "A", Date: now.AddDate(0, 2, 0), URL: "https://a"}},
		},
		Now: now,
	}
	res := RunAnnualScrape(context.Background(), deps)
	if !res.Success {
		t.Errorf("expected success, got %+v", res)
	}
	if res.EventsCount != 1 {
		t.Errorf("EventsCount = %d, want 1", res.EventsCount)
	}
	if len(store.events) != 1 {
		t.Errorf("store should have 1 event, has %d", len(store.events))
	}
}

func TestRunAnnualScrape_StoreError(t *testing.T) {
	now := time.Date(2026, 5, 23, 0, 0, 0, 0, time.UTC)
	store := &fakeStore{putErr: errors.New("ssm down")}
	deps := AnnualScrapeDeps{
		Store: store,
		Scrapers: []annual.SiteScraper{
			&fakeAnnualScraper{name: "A", out: annual.AnnualEvent{Name: "A", Date: now.AddDate(0, 2, 0), URL: "https://a"}},
		},
		Now: now,
	}
	res := RunAnnualScrape(context.Background(), deps)
	if res.Success {
		t.Errorf("expected failure on store error, got %+v", res)
	}
	if res.Error == "" {
		t.Errorf("expected non-empty Error message")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/jobs/ -run TestRunAnnualScrape -v`
Expected: compile error.

- [ ] **Step 3: Implement `internal/jobs/annual.go`**

```go
package jobs

import (
	"context"
	"fmt"
	"time"

	"event_calendar/internal/annual"
)

// AnnualScrapeDeps holds collaborators for RunAnnualScrape.
type AnnualScrapeDeps struct {
	Store    annual.Store
	Scrapers []annual.SiteScraper
	Now      time.Time
}

// RunAnnualScrape executes the monthly annual-events refresh: scrape each
// site, merge with last-known SSM state, persist back. Per-site failures
// are logged inside the orchestrator and do not fail the job; only a store
// read/write failure produces a non-success Result.
func RunAnnualScrape(ctx context.Context, deps AnnualScrapeDeps) Result {
	n, err := annual.Scrape(ctx, deps.Scrapers, deps.Store, deps.Now)
	if err != nil {
		return Result{Error: fmt.Sprintf("annual scrape failed: %v", err)}
	}
	return Result{Success: true, EventsCount: n}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/jobs/...`
Expected: all green.

- [ ] **Step 5: Commit**

```bash
git add internal/jobs/annual.go internal/jobs/annual_test.go
git commit -m "feat(jobs): RunAnnualScrape orchestrator job"
```

---

## Task 11: Add AWS SDK v2 deps

**Files:**
- Modify: `go.mod`, `go.sum`

- [ ] **Step 1: Add the modules**

Run:
```bash
go get github.com/aws/aws-sdk-go-v2
go get github.com/aws/aws-sdk-go-v2/config
go get github.com/aws/aws-sdk-go-v2/service/ssm
go mod tidy
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: no output, exit 0.

- [ ] **Step 3: Commit**

```bash
git add go.mod go.sum
git commit -m "build: add aws-sdk-go-v2 (config + ssm) for SSMStore"
```

---

## Task 12: Real SSM adapter (wraps SDK to satisfy `ssmAPI`)

The SDK's `ssm.Client` returns AWS-shaped responses; we need a tiny adapter
that implements our `ssmAPI` interface (string in, string out, sentinel error
on not-found). This keeps the unit tests SDK-free.

**Files:**
- Create: `internal/annual/ssm_aws.go`

- [ ] **Step 1: Implement the adapter**

```go
package annual

import (
	"context"
	"errors"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/ssm"
	ssmtypes "github.com/aws/aws-sdk-go-v2/service/ssm/types"
)

// AWSSSMClient is the subset of the SDK ssm.Client we use.
type AWSSSMClient interface {
	GetParameter(ctx context.Context, params *ssm.GetParameterInput, optFns ...func(*ssm.Options)) (*ssm.GetParameterOutput, error)
	PutParameter(ctx context.Context, params *ssm.PutParameterInput, optFns ...func(*ssm.Options)) (*ssm.PutParameterOutput, error)
}

// NewAWSSSMAdapter wraps an SDK client so it satisfies ssmAPI.
func NewAWSSSMAdapter(c AWSSSMClient) ssmAPI {
	return &awsSSMAdapter{c: c}
}

type awsSSMAdapter struct {
	c AWSSSMClient
}

func (a *awsSSMAdapter) GetParameter(ctx context.Context, name string) (string, error) {
	out, err := a.c.GetParameter(ctx, &ssm.GetParameterInput{Name: aws.String(name)})
	if err != nil {
		var nf *ssmtypes.ParameterNotFound
		if errors.As(err, &nf) {
			return "", ErrParameterNotFound
		}
		return "", err
	}
	if out.Parameter == nil || out.Parameter.Value == nil {
		return "", nil
	}
	return *out.Parameter.Value, nil
}

func (a *awsSSMAdapter) PutParameter(ctx context.Context, name, value string) error {
	_, err := a.c.PutParameter(ctx, &ssm.PutParameterInput{
		Name:      aws.String(name),
		Value:     aws.String(value),
		Type:      ssmtypes.ParameterTypeString,
		Tier:      ssmtypes.ParameterTierStandard,
		Overwrite: aws.Bool(true),
	})
	return err
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: no output, exit 0.

- [ ] **Step 3: Commit**

```bash
git add internal/annual/ssm_aws.go
git commit -m "feat(annual): AWS SDK adapter implementing ssmAPI"
```

---

## Task 13: Wire Lambda `scrape_annual` action + SSM-backed `events` read

**Files:**
- Modify: `cmd/lambda/main.go`

- [ ] **Step 1: Update `cmd/lambda/main.go`**

Replace the existing file with:

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
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"

	"event_calendar/internal/annual"
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
		return runEvents(ctx, cfg), nil
	case "poll":
		return runPoll(cfg), nil
	case "scrape_annual":
		return runAnnualScrape(ctx, cfg), nil
	default:
		return jobs.Result{Error: fmt.Sprintf("unknown action %q", evt.Action)}, nil
	}
}

func runEvents(ctx context.Context, cfg config.Config) jobs.Result {
	factory := scraping.NewScrapingServiceFactory()
	scraper := factory.CreateDefaultService()
	sender := telegram.NewService(cfg.BotToken)

	store, err := newAnnualStore(ctx, cfg)
	var annualEvts []annual.AnnualEvent
	if err != nil {
		log.Printf("events: SSM client init failed (footer will be empty): %v", err)
	} else {
		annualEvts, err = store.Get(ctx)
		if err != nil {
			log.Printf("events: SSM read failed (footer will be empty): %v", err)
			annualEvts = nil
		}
	}

	return jobs.RunEventsDigest(jobs.EventsDeps{
		Scraper: scraper,
		Sender:  sender,
		Now:     time.Now(),
	}, cfg, annualEvts)
}

func runPoll(cfg config.Config) jobs.Result {
	poller := telegram.NewService(cfg.PollBotToken)
	return jobs.RunMonthlyPoll(jobs.PollDeps{Poller: poller}, cfg)
}

func runAnnualScrape(ctx context.Context, cfg config.Config) jobs.Result {
	store, err := newAnnualStore(ctx, cfg)
	if err != nil {
		return jobs.Result{Error: fmt.Sprintf("ssm init: %v", err)}
	}
	return jobs.RunAnnualScrape(ctx, jobs.AnnualScrapeDeps{
		Store: store,
		Scrapers: []annual.SiteScraper{
			annual.NewPrairieDevCon(),
			annual.NewWomenHack(),
			annual.NewNorthForge(),
			annual.NewMbTechWeek(),
		},
		Now: time.Now(),
	})
}

// newAnnualStore builds an SSM-backed Store using default AWS credentials
// (the Lambda execution role).
func newAnnualStore(ctx context.Context, cfg config.Config) (annual.Store, error) {
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
	if err != nil {
		return nil, err
	}
	client := ssm.NewFromConfig(awsCfg)
	return annual.NewSSMStore(annual.NewAWSSSMAdapter(client), cfg.AnnualSSMParam), nil
}

func recoverPanic() {
	if r := recover(); r != nil {
		log.Printf("PANIC: %v", r)
	}
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: no output, exit 0.

- [ ] **Step 3: Run all tests**

Run: `go test ./...`
Expected: all green.

- [ ] **Step 4: Commit**

```bash
git add cmd/lambda/main.go
git commit -m "feat(lambda): scrape_annual action + SSM-backed annual footer read"
```

---

## Task 14: CLI `-action=scrape_annual` subcommand

**Files:**
- Modify: `cmd/cli/main.go`

- [ ] **Step 1: Update `cmd/cli/main.go`**

Replace the file with:

```go
// Command cli runs the same jobs the Lambda runs, from your terminal.
// Useful for local testing without invoking Lambda.
//
// Usage:
//
//	go run ./cmd/cli -action=events
//	go run ./cmd/cli -action=poll
//	go run ./cmd/cli -action=scrape_annual                # writes to SSM via default AWS creds
//	go run ./cmd/cli -action=scrape_annual -out=/tmp/x.json  # writes JSON to file, skips SSM
package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"os"
	"time"

	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/ssm"

	"event_calendar/internal/annual"
	"event_calendar/internal/config"
	"event_calendar/internal/jobs"
	"event_calendar/pkg/scraping"
	"event_calendar/pkg/telegram"
)

func main() {
	action := flag.String("action", "events", "job to run: events, poll, or scrape_annual")
	out := flag.String("out", "", "scrape_annual only: write JSON to this file instead of SSM")
	flag.Parse()

	ctx := context.Background()
	cfg := config.Load()
	var result jobs.Result
	switch *action {
	case "events":
		result = runEvents(ctx, cfg)
	case "poll":
		result = runPoll(cfg)
	case "scrape_annual":
		result = runAnnualScrape(ctx, cfg, *out)
	default:
		log.Fatalf("unknown action %q (want events, poll, or scrape_annual)", *action)
	}

	js, _ := json.MarshalIndent(result, "", "  ")
	log.Printf("result:\n%s", js)
	if !result.Success {
		os.Exit(1)
	}
}

func runEvents(ctx context.Context, cfg config.Config) jobs.Result {
	factory := scraping.NewScrapingServiceFactory()
	scraper := factory.CreateDefaultService()
	sender := telegram.NewService(cfg.BotToken)
	return jobs.RunEventsDigest(jobs.EventsDeps{Scraper: scraper, Sender: sender, Now: time.Now()}, cfg, nil)
}

func runPoll(cfg config.Config) jobs.Result {
	poller := telegram.NewService(cfg.PollBotToken)
	return jobs.RunMonthlyPoll(jobs.PollDeps{Poller: poller}, cfg)
}

func runAnnualScrape(ctx context.Context, cfg config.Config, outFile string) jobs.Result {
	scrapers := []annual.SiteScraper{
		annual.NewPrairieDevCon(),
		annual.NewWomenHack(),
		annual.NewNorthForge(),
		annual.NewMbTechWeek(),
	}

	var store annual.Store
	if outFile != "" {
		store = &fileStore{path: outFile}
	} else {
		awsCfg, err := awsconfig.LoadDefaultConfig(ctx)
		if err != nil {
			return jobs.Result{Error: "aws config: " + err.Error()}
		}
		store = annual.NewSSMStore(
			annual.NewAWSSSMAdapter(ssm.NewFromConfig(awsCfg)),
			cfg.AnnualSSMParam,
		)
	}

	return jobs.RunAnnualScrape(ctx, jobs.AnnualScrapeDeps{
		Store: store, Scrapers: scrapers, Now: time.Now(),
	})
}

// fileStore writes the JSON blob to a local file and reads it back.
// Used by the CLI's -out flag for local inspection.
type fileStore struct{ path string }

func (f *fileStore) Get(ctx context.Context) ([]annual.AnnualEvent, error) {
	b, err := os.ReadFile(f.path)
	if err != nil {
		if os.IsNotExist(err) {
			return []annual.AnnualEvent{}, nil
		}
		return nil, err
	}
	var out []annual.AnnualEvent
	if len(b) == 0 {
		return out, nil
	}
	return out, json.Unmarshal(b, &out)
}

func (f *fileStore) Put(ctx context.Context, events []annual.AnnualEvent) error {
	b, err := json.MarshalIndent(events, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(f.path, b, 0o644)
}
```

- [ ] **Step 2: Verify build**

Run: `go build ./...`
Expected: no output, exit 0.

- [ ] **Step 3: Commit**

```bash
git add cmd/cli/main.go
git commit -m "feat(cli): scrape_annual subcommand with -out file fallback"
```

---

## Task 15: Deploy script — IAM, SSM seed, EventBridge rule, env var

**Files:**
- Modify: `deploy/deploy.sh`

- [ ] **Step 1: Add the new constants**

In `deploy/deploy.sh`, near the top with the other `RULE_*` constants, add:

```bash
RULE_ANNUAL="winnipeg-annual-scrape-monthly"
SSM_PARAM="/winnipeg-tech-events/annual-events"
```

- [ ] **Step 2: Attach the SSM IAM policy**

After the `ensure IAM role` block (right before `Deploying Lambda function`), add:

```bash
echo "== Attaching SSM read/write policy to ${ROLE_NAME} =="
SSM_PARAM_ARN="arn:aws:ssm:${REGION}:${ACCOUNT_ID}:parameter${SSM_PARAM}"
aws iam put-role-policy \
    --role-name "$ROLE_NAME" \
    --policy-name "ssm-annual-events" \
    --policy-document "{
        \"Version\": \"2012-10-17\",
        \"Statement\": [{
            \"Effect\": \"Allow\",
            \"Action\": [\"ssm:GetParameter\", \"ssm:PutParameter\"],
            \"Resource\": \"${SSM_PARAM_ARN}\"
        }]
    }" >/dev/null
```

- [ ] **Step 3: Idempotently seed the SSM parameter**

After the IAM policy block, add:

```bash
echo "== Ensuring SSM parameter ${SSM_PARAM} =="
if ! aws ssm get-parameter --name "$SSM_PARAM" --region "$REGION" >/dev/null 2>&1; then
    aws ssm put-parameter \
        --name "$SSM_PARAM" \
        --type String \
        --value '[]' \
        --region "$REGION" >/dev/null
    echo "  -> created (empty list)"
else
    echo "  -> exists; leaving value untouched"
fi
```

- [ ] **Step 4: Add the EventBridge rule**

After the two existing `create_or_update_rule` calls (events weekly + poll monthly), add a third:

```bash
# 1st of each month, 13:00 UTC — refreshes annual events ahead of the
# weekly digest (Mondays 14:00 UTC).
create_or_update_rule "$RULE_ANNUAL" \
    "cron(0 13 1 * ? *)" \
    '{"action":"scrape_annual"}' \
    "Monthly refresh of curated annual events"
```

- [ ] **Step 5: Pass the env var to Lambda**

In the `Setting Lambda environment variables` block, add `ANNUAL_SSM_PARAM=${SSM_PARAM}` to the `--environment` JSON:

```bash
--environment "Variables={
    TELEGRAM_BOT_TOKEN=${TELEGRAM_BOT_TOKEN:-},
    TELEGRAM_CHAT_ID=${TELEGRAM_CHAT_ID:-},
    TELEGRAM_POLL_BOT_TOKEN=${TELEGRAM_POLL_BOT_TOKEN:-},
    TELEGRAM_POLL_CHAT_ID=${TELEGRAM_POLL_CHAT_ID:-},
    CITY=${CITY:-Winnipeg},
    CATEGORIES=${CATEGORIES:-tech},
    PERIOD_DAYS=${PERIOD_DAYS:-30},
    TEST_MODE=${TEST_MODE:-false},
    ANNUAL_SSM_PARAM=${SSM_PARAM}
}"
```

- [ ] **Step 6: Update the deploy.sh header docs**

In the `# Environment` comment block at the top, add:
```
#   ANNUAL_SSM_PARAM=/winnipeg-tech-events/annual-events (default; set by this script)
```

- [ ] **Step 7: Sanity check shell syntax**

Run: `bash -n deploy/deploy.sh`
Expected: no output, exit 0.

- [ ] **Step 8: Commit**

```bash
git add deploy/deploy.sh
git commit -m "deploy: SSM param + IAM policy + monthly EventBridge rule for annual scrape"
```

---

## Task 16: Manual end-to-end verification + CHANGELOG

**Files:**
- Modify: `CHANGELOG.md`

- [ ] **Step 1: Local end-to-end with file store**

Run:
```bash
go run ./cmd/cli -action=scrape_annual -out=/tmp/annual.json
cat /tmp/annual.json
```

Expected: JSON array with up to 4 entries. Each has `name`, `date` (future ISO timestamp), `url`. Entries whose actual conference date is in the past should be absent. If a site fails its parser, the orchestrator log line `annual: scrape <Name> failed (retaining last-known)` appears and that entry will be absent from the empty-baseline run.

If any site fails to parse, inspect the captured fixture in `internal/annual/testdata/` and tighten the corresponding regex in the parser. Re-run the per-site test, then re-run the CLI.

- [ ] **Step 2: Verify digest renders with the data**

```bash
TEST_MODE=true CITY=Winnipeg go run ./cmd/cli -action=events
```

This won't actually exercise SSM read (CLI passes `nil` for the annual slice — by design, since the CLI doesn't authenticate to AWS for the events path). To verify the footer end-to-end, run the deployed Lambda after Task 15 lands.

- [ ] **Step 3: Update CHANGELOG.md**

Prepend a new entry under the existing format (peek at the existing CHANGELOG to match style — the project uses commit-style summaries). Add:

```markdown
## Unreleased

- feat(digest): annual events footer now sourced from monthly SSM-backed scrape instead of a hardcoded slice. New Lambda action `scrape_annual` runs on the 1st of each month via EventBridge; the weekly digest reads the latest result from SSM Parameter Store. Per-site scrape failures retain last-known-good data so the digest never breaks.
```

- [ ] **Step 4: Commit**

```bash
git add CHANGELOG.md
git commit -m "docs(changelog): annual events scraped monthly via SSM"
```

- [ ] **Step 5: Final test sweep**

Run: `go test ./...`
Expected: all green.

Run: `go build ./...`
Expected: clean.

- [ ] **Step 6: Push & deploy**

```bash
git push origin main
```

GitHub Actions runs `deploy/deploy.sh`, which:
- Builds the new bootstrap binary with SDK deps
- Attaches the SSM IAM policy
- Creates the SSM parameter if absent
- Creates the monthly EventBridge rule
- Updates Lambda env with `ANNUAL_SSM_PARAM`

After deploy, manually trigger the first scrape:
```bash
aws lambda invoke --function-name winnipeg-tech-events \
    --payload '{"action":"scrape_annual"}' \
    --cli-binary-format raw-in-base64-out /tmp/out.json && cat /tmp/out.json
```

Inspect SSM:
```bash
aws ssm get-parameter --name /winnipeg-tech-events/annual-events \
    --query Parameter.Value --output text | jq .
```

Trigger the events digest in TEST_MODE first (if you want to dry-run) or let the next Monday's schedule fire it.

---

## Done

After Task 16, the hardcoded annual-events list is gone, the data is refreshed monthly from each site, and the digest footer is driven entirely by SSM. Future maintenance is per-parser only (no source edits to update dates).
