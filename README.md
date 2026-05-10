# t212

Unofficial Go client for the [Trading 212 public API](https://docs.trading212.com).

> **Disclaimer.** This project is not affiliated with, endorsed by, or otherwise connected to Trading 212. The Trading 212 API is in beta and may change without notice; use at your own risk.

## Features

- All non-deprecated endpoints in the spec wrapped: orders, account summary, positions, history (orders / dividends / transactions), metadata (instruments / exchanges), report exports.
- Background **position watcher** — `NewClient` starts polling `/equity/positions` automatically, fires open / close / error callbacks, and exposes a thread-safe `Snapshot()`.
- **Pagination iterators** for the cursor-based history endpoints (Go 1.23+ `range`-over-func).
- **Report poller** (`WaitForReport`) that blocks until a requested CSV export is `Finished`.
- Pluggable `http.Client` / `Transport`, configurable timeout, custom `User-Agent`.
- Generated request/response types come with hand-written **safe accessors** (`*AccountSummary.GetCurrency() string` etc.) and **constructors** (`NewMarketRequest("AAPL_US_EQ", 5)`).

## Install

```sh
go get github.com/kludw/t212
```

Requires Go 1.23+ (uses `iter.Seq2`).

## Getting started

```go
package main

import (
    "context"
    "fmt"
    "log"

    "github.com/kludw/t212"
)

func main() {
    c, err := t212.NewClient(&t212.ClientOpts{
        Env:       "demo", // or "live"
        APIKeyID:  "...",
        APISecret: "...",
    })
    if err != nil {
        log.Fatal(err)
    }
    defer c.Close()

    summary, err := c.AccountSummary(context.Background())
    if err != nil {
        log.Fatal(err)
    }
    fmt.Printf("account %d (%s) total: %.2f\n",
        summary.GetID(), summary.GetCurrency(), summary.GetTotalValue())
}
```

Demo and live API keys are issued separately from the Trading 212 dashboard.

## Placing orders

```go
order, err := c.PlaceMarketOrder(ctx, t212.NewMarketRequest("AAPL_US_EQ", 5))
// negative quantity sells:
sell, err := c.PlaceMarketOrder(ctx, t212.NewMarketRequest("AAPL_US_EQ", -5))

// limit / stop / stop-limit have similar constructors:
lim, err := c.PlaceLimitOrder(ctx, t212.NewLimitRequest("AAPL_US_EQ", 5, 150, "DAY"))
```

> The Trading 212 API is **not idempotent** — retrying a failed order request may create duplicates.

## Watching positions

The position watcher is owned by the `Client` and runs from the moment `NewClient` returns. Wire callbacks via `ClientOpts`, then read `c.Snapshot()` whenever you want the latest known set:

```go
c, err := t212.NewClient(&t212.ClientOpts{
    Env: "demo", APIKeyID: "...", APISecret: "...",

    OnPositionOpen:  func(p *t212.Position) { log.Printf("OPEN  %s", p.GetTicker()) },
    OnPositionClose: func(p *t212.Position) { log.Printf("CLOSE %s", p.GetTicker()) },
    OnPollError:     func(err error) { log.Printf("poll: %v", err) },

    WatcherInterval: 5 * time.Second, // optional; default 3s
})
defer c.Close()

for _, p := range c.Snapshot() {
    fmt.Println(p.GetTicker(), p.GetQuantity())
}
```

Polling errors are passed to `OnPollError` (if set) and silently retried — the watcher keeps running. `Close` stops the watcher and waits for its goroutine to exit.

## Walking history

The history endpoints are cursor-paginated. Use the `*Iter` helpers to walk every page:

```go
for order, err := range c.HistoricalOrdersIter(ctx, nil) {
    if err != nil {
        return err // page fetch failed
    }
    process(order)
}
```

`DividendsIter` and `TransactionsIter` work the same way. Filters in the params struct (`Ticker`, `Limit`, etc.) are preserved across pages; the `Cursor` field is overwritten as the iterator advances.

## Generating reports

```go
enq, _ := c.RequestReport(ctx, &t212.PublicReportRequest{ /* ... */ })

report, err := c.WaitForReport(ctx, *enq.ReportId, &t212.WaitForReportOpts{
    PollInterval: 30 * time.Second,
    MaxWait:      10 * time.Minute,
})
if err != nil { return err }
fmt.Println(report.GetDownloadLink())
```

`WaitForReport` swallows transient `ErrRateLimited` responses and retries; non-rate-limit errors are returned. `MaxWait` returns `ErrReportTimeout` if exceeded.

## Custom transports / timeouts

```go
c, err := t212.NewClient(&t212.ClientOpts{
    APIKeyID: "...", APISecret: "...",

    HTTPClient: &http.Client{
        Timeout:   60 * time.Second,
        Transport: myInstrumentedTransport,
    },
    UserAgent: "my-strategy/1.0",
})
```

If `HTTPClient` is nil, a fresh client is constructed with `RequestTimeout` (or `DefaultTimeout` when zero).

## Errors

Endpoint helpers wrap HTTP errors with a sentinel from `errors.go`. Match with `errors.Is`:

```go
_, err := c.AccountSummary(ctx)
switch {
case errors.Is(err, t212.ErrUnauthorized):
    // bad credentials
case errors.Is(err, t212.ErrRateLimited):
    // back off and retry
}
```

Retry / backoff is **not** built in — wrap calls in your preferred retry loop (or supply an instrumented `HTTPClient`).

## Examples

See [`example/`](example/):
- `example/main.go` — minimal smoke test.
- `example/watcher/` — long-running position watcher with callbacks.
- `example/pagination/` — historical-orders iterator.
- `example/report/` — request and wait for a CSV export.

Run any with `go run ./example/<name>`. Each loads `.env` (see `.env.example`).

## Pies

The `/equity/pies` endpoints are marked deprecated upstream and are intentionally not wrapped. File an issue if you need them.

## Contributing

See [CLAUDE.md](CLAUDE.md) for build/test/regen instructions.
