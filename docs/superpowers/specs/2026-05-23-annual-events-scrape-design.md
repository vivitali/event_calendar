# Annual Events: Replace Hardcoded List with Monthly Scrape

**Status:** Approved (design)
**Date:** 2026-05-23
**Author:** vivitali (with Claude)

## Problem

`internal/digest/annual.go` carries a hardcoded slice of 4 Winnipeg annual tech
events (Prairie Dev Con, WomenHack, North Forge RampUp, MbTech Week) with
manually-curated dates. The list rots: dates become wrong after each year and
require source edits. The maintainer wants the list driven by data scraped from
each event's own website on a monthly cadence, with no hardcoded event data
remaining in the binary.

The annual footer's value is reminding the audience about events further out
than the weekly digest's `PERIOD_DAYS` window (default 30 days). Events inside
that window already surface through the normal Meetup/Eventbrite scrape.

## Goals

- Remove the hardcoded `annualEvents` slice from source.
- Scrape the 4 event sites once per month and store the result so the weekly
  digest can read it without re-scraping.
- Tolerate per-site scrape failure: keep last-known-good data for the failing
  event; never break the weekly digest.
- Use the cheapest viable AWS storage. SSM Parameter Store Standard tier is
  free for our volume (1 parameter, well under 4 KB).
- Add no new AWS services beyond what's necessary (no S3, DynamoDB, Secrets
  Manager).

## Non-goals

- Generalizing the scraper to arbitrary user-supplied URLs.
- Live/on-demand scrape from the digest path.
- Backfilling past events.
- Adding the 4 events into the main Meetup/Eventbrite category scrape — they
  appear there naturally when registration opens.

## Architecture

```
EventBridge (monthly cron)
  → Lambda(action="scrape_annual")
    → annual.Scrape(ctx, scrapers, store, now)
      → SSMStore.Get  (last-known baseline)
      → for each SiteScraper: fetch + parse (parallel; per-site error tolerated)
      → merge by Name(): replace on success, retain previous on error
      → drop events whose date is in the past
      → SSMStore.Put (write merged JSON blob)

EventBridge (weekly cron, existing)
  → Lambda(action="events")
    → SSMStore.Get  (empty slice on miss/error)
    → jobs.RunEventsDigest(deps, cfg, annualEvts)
      → digest.FormatEventsMessage(events, annualEvts, now)
        → annualFooter(annualEvts, now)
```

### AWS resources

| Resource | Detail |
|---|---|
| SSM Parameter | `/winnipeg-tech-events/annual-events`, Standard tier, type `String`, JSON array of `AnnualEvent` |
| EventBridge rule | `winnipeg-annual-scrape-monthly`, schedule `cron(0 13 1 * ? *)` (1st of month, 13:00 UTC), input `{"action":"scrape_annual"}` |
| IAM policy (inline on existing role) | `ssm:GetParameter`, `ssm:PutParameter` scoped to the parameter ARN |

## Code Layout

New package `internal/annual/`:

```
internal/annual/
├── annual.go          AnnualEvent struct (moved from digest/)
├── store.go           Store interface + SSMStore impl
├── store_test.go
├── scraper.go         SiteScraper interface + Scrape() orchestrator
├── scraper_test.go
├── prairiedevcon.go
├── prairiedevcon_test.go
├── womenhack.go
├── womenhack_test.go
├── northforge.go
├── northforge_test.go
├── mbtechweek.go
├── mbtechweek_test.go
└── testdata/          captured HTML fixtures, one per site
```

### Types

```go
type AnnualEvent struct {
    Name string    `json:"name"`
    Date time.Time `json:"date"`
    URL  string    `json:"url"`
}

type Store interface {
    Get(ctx context.Context) ([]AnnualEvent, error)
    Put(ctx context.Context, events []AnnualEvent) error
}

type SiteScraper interface {
    Name() string                                    // stable identifier, used as merge key
    URL() string                                     // canonical event URL
    Scrape(ctx context.Context) (AnnualEvent, error)
}
```

### Orchestrator behaviour (`annual.Scrape`)

1. `baseline, _ := store.Get(ctx)` — start from last-known. On read error, log and continue with empty baseline.
2. Index baseline by `Name()` into a `map[string]AnnualEvent`.
3. For each scraper, run `Scrape(ctx)` concurrently — goroutine per scraper with a `sync.WaitGroup`. On success, replace the entry (mutex-guarded map write). On error, log and leave the baseline entry in place.
4. Drop entries whose `Date.Before(now)`.
5. Sort by `Date` ascending.
6. `store.Put(ctx, merged)`. Return count + first error if any (non-fatal).

### Store: SSM impl

`SSMStore` wraps an `ssm.Client` and parameter name. `Get` calls `GetParameter`, returns `[]AnnualEvent{}` if the parameter is missing (`ParameterNotFound`); other errors propagate. `Put` calls `PutParameter` with `Overwrite=true`, `Type=String`, `Tier=Standard`.

The parameter holds a JSON array. Empty state is `[]`, set by the deploy script (idempotent seed via `aws ssm put-parameter --overwrite`).

## Lambda Wiring

`cmd/lambda/main.go` dispatch gains a `scrape_annual` case and the existing `events` case reads the store:

```go
switch evt.Action {
case "events", "":
    store := annual.NewSSMStore(ssmClient, cfg.AnnualSSMParam)
    annualEvts, _ := store.Get(ctx)  // err → empty slice → no footer
    return runEvents(cfg, annualEvts), nil
case "poll":
    return runPoll(cfg), nil
case "scrape_annual":
    return runAnnualScrape(ctx, cfg), nil
}
```

`runAnnualScrape` constructs the SSM-backed store and the 4 site scrapers, then calls `jobs.RunAnnualScrape`. The SSM client is built once per Lambda init using `aws-sdk-go-v2`.

`jobs.RunEventsDigest` signature changes:

```go
func RunEventsDigest(deps EventsDeps, cfg config.Config, annualEvts []annual.AnnualEvent) Result
```

Passes `annualEvts` into `digest.FormatEventsMessage`, which gains a slice parameter and forwards it to `annualFooter`.

`digest.FormatEventsMessage` signature:

```go
func FormatEventsMessage(events []models.Event, annualEvts []annual.AnnualEvent, now time.Time) string
```

`internal/digest/annual.go` is deleted; the `AnnualEvent` struct moves to `internal/annual/annual.go`. `annualFooter` becomes:

```go
func annualFooter(events []annual.AnnualEvent, now time.Time) string {
    // filter Date.After(now), render expandable blockquote (unchanged markup)
}
```

## Config

`internal/config/config.go` gains:

```go
AnnualSSMParam string  // default "/winnipeg-tech-events/annual-events"
```

read via `envOr("ANNUAL_SSM_PARAM", "/winnipeg-tech-events/annual-events")`.

## Deploy

`deploy/deploy.sh` changes:

1. New constant `RULE_ANNUAL="winnipeg-annual-scrape-monthly"` and `SSM_PARAM="/winnipeg-tech-events/annual-events"`.
2. Attach inline IAM policy to `$ROLE_NAME` granting `ssm:GetParameter` + `ssm:PutParameter` on `arn:aws:ssm:${REGION}:${ACCOUNT_ID}:parameter${SSM_PARAM}` (idempotent — `aws iam put-role-policy`).
3. Idempotent SSM seed:
   ```
   aws ssm put-parameter --name "$SSM_PARAM" --type String \
       --value '[]' --overwrite --region "$REGION" >/dev/null
   ```
   Run only when `aws ssm get-parameter` returns `ParameterNotFound`, so existing data is never clobbered.
4. New `create_or_update_rule` call for `$RULE_ANNUAL` with `cron(0 13 1 * ? *)` and input `{"action":"scrape_annual"}`.
5. Lambda environment: add `ANNUAL_SSM_PARAM=${SSM_PARAM}` to the `update-function-configuration` call.

`.github/workflows/deploy.yml`: pass `ANNUAL_SSM_PARAM` env through if overridden via repo vars (default is the parameter path above).

## Local CLI

`cmd/cli` gains a `scrape_annual` subcommand mirroring the Lambda action so the maintainer can:

- Run a one-off scrape against real sites and inspect the resulting JSON.
- Test the orchestrator end-to-end without invoking Lambda.

The CLI uses the same `annual.NewSSMStore` when AWS creds are present, or accepts `--out <file.json>` to write locally for inspection (skipping SSM).

## Dependencies

Add to `go.mod`:

- `github.com/aws/aws-sdk-go-v2`
- `github.com/aws/aws-sdk-go-v2/config`
- `github.com/aws/aws-sdk-go-v2/service/ssm`

Adds ~5 MB to the deployed Lambda zip; acceptable given current ~few-MB binary.

## Testing

| Layer | Approach |
|---|---|
| Per-site scraper | Fixture HTML in `testdata/<site>.html` captured via `curl`. Test asserts the parser extracts the expected `Date` and `URL`. |
| Orchestrator (`annual.Scrape`) | Table-driven test with fake `SiteScraper`s (success, error, past-dated) and fake `Store`. Asserts: errors retain baseline; past events dropped; result sorted by date; store written. |
| `SSMStore` | Unit tests against a fake `ssmAPI` interface (covering `Get`, `Put`, `ParameterNotFound` path). No live AWS call in CI. |
| Digest | Update `TestFormatEventsMessage_HasExpandableAnnualFooter` to pass an explicit `[]annual.AnnualEvent` instead of relying on the deleted hardcoded slice. |
| Lambda dispatch | Existing tests in `internal/jobs` cover the events path; add a minimal `RunAnnualScrape` test using fakes. |

Manual verification before merge:

1. Run `go run ./cmd/cli scrape_annual --out /tmp/annual.json`; inspect that all 4 events parsed with plausible future dates.
2. Run digest path locally with the JSON wired in; confirm footer renders.

## Failure & Edge Cases

- **All 4 scrapers fail on first run, SSM empty:** Store ends up with `[]`. Weekly digest reads empty slice → no annual footer. Identical to current behavior when all events have passed. Acceptable.
- **One scraper consistently fails for months:** Last-known-good entry stays in SSM. Once that event's date passes, the orchestrator drops it, and the footer simply loses that entry. Maintainer's responsibility to fix the parser when noticed.
- **SSM read fails in `events` action:** Treated as empty slice; digest still sends. Error logged for ops visibility.
- **Site changes URL:** Parser breaks → logged → last-known retained until manually updated.

## Out of Scope / Non-changes

- The hardcoded scraper source URLs in `pkg/{eventbrite,meetup,luma}` are the sources of truth for those scrapers and remain unchanged.
- Brand strings ("Winnipeg Tech Events", hashtags, alert title) are presentation, not data, and remain in source.
- Config defaults (`CITY=Winnipeg`, `CATEGORIES=tech`, `PERIOD_DAYS=30`) remain env-overridable as today.
- The Ukrainian poll question in `telegram.SendMonthlyMeetupPoll` remains in source.

## Open Questions

None at this point.
