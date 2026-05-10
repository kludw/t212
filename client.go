// Package t212 is a Go client for the Trading 212 public API.
//
// Trading 212's API is currently in beta. See https://docs.trading212.com for
// the upstream specification.
package t212

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"
)

// PositionCallback is invoked by the position watcher when a position is
// detected as opened or closed.
type PositionCallback = func(p *Position)

// PollErrorCallback is invoked by the position watcher when a polling tick
// fails. The watcher continues running regardless.
type PollErrorCallback = func(err error)

// ClientOpts configures a Client. All fields except APIKeyID and APISecret
// are optional.
type ClientOpts struct {
	// Env selects which Trading 212 environment to talk to. Empty defaults to
	// "demo". "live" routes to the production API. Ignored when BaseURL is set.
	Env string

	// BaseURL overrides Env's default URL. Useful for sandboxes, proxies, and
	// tests. Should not include a trailing slash and should already include
	// the API version segment (e.g. http://localhost:8080/api/v0).
	BaseURL string

	// APIKeyID and APISecret authenticate via HTTP Basic.
	APIKeyID  string
	APISecret string

	// UserAgent overrides the default User-Agent. Empty uses defaultUserAgent.
	UserAgent string

	// HTTPClient is the underlying HTTP client used for all requests. If nil,
	// a new http.Client is constructed with timeout RequestTimeout (or
	// DefaultTimeout when RequestTimeout is zero).
	HTTPClient *http.Client

	// RequestTimeout overrides DefaultTimeout when HTTPClient is nil.
	// Ignored when HTTPClient is provided.
	RequestTimeout time.Duration

	// OnPositionOpen and OnPositionClose receive the position whenever the
	// watcher detects a transition. Both are optional — the watcher always
	// runs so Snapshot stays current.
	OnPositionOpen  PositionCallback
	OnPositionClose PositionCallback

	// OnPollError is invoked with any error returned by the watcher's
	// background polling loop. The watcher does not stop on errors.
	OnPollError PollErrorCallback

	// WatcherInterval overrides DefaultWatcherInterval. Must be >= 0; zero
	// uses the default.
	WatcherInterval time.Duration
}

// Validate checks that ClientOpts has the minimum required fields.
func (opts *ClientOpts) Validate() error {
	if opts == nil {
		return ErrNilOpts
	}

	if opts.Env != "" {
		env := strings.ToLower(opts.Env)
		if env != "demo" && env != "live" {
			return ErrInvalidEnv
		}
	}

	if opts.APIKeyID == "" {
		return ErrEmptyAPIKeyID
	}

	if opts.APISecret == "" {
		return ErrEmptyAPISecret
	}

	return nil
}

// Client is the Trading 212 API client. Construct one via NewClient. The
// client owns a background goroutine that polls /equity/positions and
// maintains a snapshot reachable via Snapshot. Call Close when done to stop
// the goroutine.
type Client struct {
	httpc      *http.Client
	authHeader string
	baseURL    string
	userAgent  string

	onPosOpen  PositionCallback
	onPosClose PositionCallback
	onPollErr  PollErrorCallback
	interval   time.Duration

	mu        sync.RWMutex
	positions map[string]Position

	cancel    context.CancelFunc
	done      chan struct{}
	closeOnce sync.Once
}

// NewClient constructs a Client and starts its background position watcher.
// It performs an initial synchronous Positions fetch so Snapshot returns
// fresh data on return; transient rate-limit errors during that initial
// fetch are tolerated, but other errors are returned.
func NewClient(opts *ClientOpts) (*Client, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	baseURL := opts.BaseURL
	if baseURL == "" {
		baseURL = demoURL
		if strings.ToLower(opts.Env) == "live" {
			baseURL = liveURL
		}
	}

	httpc := opts.HTTPClient
	if httpc == nil {
		timeout := opts.RequestTimeout
		if timeout == 0 {
			timeout = DefaultTimeout
		}
		httpc = &http.Client{Timeout: timeout}
	}

	ua := opts.UserAgent
	if ua == "" {
		ua = defaultUserAgent
	}

	interval := opts.WatcherInterval
	if interval == 0 {
		interval = DefaultWatcherInterval
	}

	creds := base64.StdEncoding.EncodeToString([]byte(opts.APIKeyID + ":" + opts.APISecret))

	ctx, cancel := context.WithCancel(context.Background())

	c := &Client{
		httpc:      httpc,
		authHeader: "Basic " + creds,
		baseURL:    baseURL,
		userAgent:  ua,
		onPosOpen:  opts.OnPositionOpen,
		onPosClose: opts.OnPositionClose,
		onPollErr:  opts.OnPollError,
		interval:   interval,
		positions:  make(map[string]Position),
		cancel:     cancel,
		done:       make(chan struct{}),
	}

	current, err := c.Positions(ctx, nil)
	if err != nil && !errors.Is(err, ErrRateLimited) {
		cancel()
		close(c.done)
		return nil, err
	}

	c.mu.Lock()
	for _, p := range current {
		if p.Instrument == nil || p.Instrument.Ticker == nil {
			continue
		}
		c.positions[*p.Instrument.Ticker] = p
	}
	c.mu.Unlock()

	if c.onPosOpen != nil {
		for i := range current {
			p := current[i]
			c.onPosOpen(&p)
		}
	}

	go c.runWatcher(ctx)

	return c, nil
}

// Close stops the background position watcher and waits for it to exit.
// Subsequent calls are no-ops. Close does not block in-flight API calls
// initiated by other callers.
func (c *Client) Close() {
	c.closeOnce.Do(func() {
		c.cancel()
		<-c.done
	})
}

// Snapshot returns the most recently observed open positions, keyed by
// ticker. The returned slice is a copy and safe to mutate.
func (c *Client) Snapshot() []Position {
	c.mu.RLock()
	defer c.mu.RUnlock()

	out := make([]Position, 0, len(c.positions))
	for _, p := range c.positions {
		out = append(out, p)
	}
	return out
}

func (c *Client) runWatcher(ctx context.Context) {
	defer close(c.done)

	ticker := time.NewTicker(c.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			if ctx.Err() != nil {
				return
			}
			c.pollOnce(ctx)
		}
	}
}

func (c *Client) pollOnce(ctx context.Context) {
	current, err := c.Positions(ctx, nil)
	if err != nil {
		if c.onPollErr != nil && ctx.Err() == nil {
			c.onPollErr(err)
		}
		return
	}

	next := make(map[string]Position, len(current))
	for _, p := range current {
		if p.Instrument == nil || p.Instrument.Ticker == nil {
			continue
		}
		next[*p.Instrument.Ticker] = p
	}

	c.mu.Lock()
	known := c.positions
	c.positions = next
	c.mu.Unlock()

	if c.onPosOpen != nil {
		for key := range next {
			if _, ok := known[key]; !ok {
				p := next[key]
				c.onPosOpen(&p)
			}
		}
	}

	if c.onPosClose != nil {
		for key := range known {
			if _, ok := next[key]; !ok {
				p := known[key]
				c.onPosClose(&p)
			}
		}
	}
}

func (c *Client) do(ctx context.Context, method, path string, params url.Values, body, out any) error {
	fullURL := c.baseURL + path
	if len(params) > 0 {
		fullURL += "?" + params.Encode()
	}

	var bodyReader io.Reader
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("%w: %v", ErrEncode, err)
		}
		bodyReader = bytes.NewReader(b)
	}

	req, err := http.NewRequestWithContext(ctx, method, fullURL, bodyReader)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrRequest, err)
	}
	req.Header.Set("Authorization", c.authHeader)
	req.Header.Set("User-Agent", c.userAgent)
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpc.Do(req)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrRequest, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBody, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("%w: status %d: %s", statusError(resp.StatusCode), resp.StatusCode, respBody)
	}

	if out == nil {
		return nil
	}
	if err := json.NewDecoder(resp.Body).Decode(out); err != nil {
		return fmt.Errorf("%w: %v", ErrDecode, err)
	}
	return nil
}

func statusError(status int) error {
	switch status {
	case http.StatusBadRequest:
		return ErrBadRequest
	case http.StatusUnauthorized:
		return ErrUnauthorized
	case http.StatusForbidden:
		return ErrForbidden
	case http.StatusNotFound:
		return ErrNotFound
	case http.StatusRequestTimeout:
		return ErrTimeout
	case http.StatusTooManyRequests:
		return ErrRateLimited
	default:
		return ErrUnexpectedStatus
	}
}

// AccountSummary fetches the equity account summary: account ID, primary
// currency, total value, and the cash and investments breakdown.
//
// GET /api/v0/equity/account/summary (rate limit: 1 req / 5s).
func (c *Client) AccountSummary(ctx context.Context) (*AccountSummary, error) {
	var out AccountSummary
	if err := c.do(ctx, http.MethodGet, "/equity/account/summary", nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Positions fetches all open positions, optionally filtered by ticker via
// params.Ticker. Pass nil for params to fetch all positions.
//
// GET /api/v0/equity/positions (rate limit: 1 req / 1s).
func (c *Client) Positions(ctx context.Context, params *GetPositionsParams) ([]Position, error) {
	q := url.Values{}
	if params != nil && params.Ticker != nil {
		q.Set("ticker", *params.Ticker)
	}

	var out []Position
	if err := c.do(ctx, http.MethodGet, "/equity/positions", q, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Orders fetches all currently active (pending) orders.
//
// GET /api/v0/equity/orders (rate limit: 1 req / 5s).
func (c *Client) Orders(ctx context.Context) ([]Order, error) {
	var out []Order
	if err := c.do(ctx, http.MethodGet, "/equity/orders", nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// OrderByID fetches a single pending order by its numeric ID.
//
// GET /api/v0/equity/orders/{id} (rate limit: 1 req / 1s).
func (c *Client) OrderByID(ctx context.Context, id int64) (*Order, error) {
	var out Order
	if err := c.do(ctx, http.MethodGet, fmt.Sprintf("/equity/orders/%d", id), nil, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// CancelOrder requests cancellation of an active order. Cancellation is not
// guaranteed if the order is already in the process of being filled.
//
// DELETE /api/v0/equity/orders/{id} (rate limit: 50 req / 1m).
func (c *Client) CancelOrder(ctx context.Context, id int64) error {
	return c.do(ctx, http.MethodDelete, fmt.Sprintf("/equity/orders/%d", id), nil, nil, nil)
}

// PlaceMarketOrder places a market order. A positive Quantity buys; a negative
// Quantity sells.
//
// Note: the Trading 212 API is not idempotent — sending the same request twice
// may create duplicate orders.
//
// POST /api/v0/equity/orders/market (rate limit: 50 req / 1m).
func (c *Client) PlaceMarketOrder(ctx context.Context, req *MarketRequest) (*Order, error) {
	var out Order
	if err := c.do(ctx, http.MethodPost, "/equity/orders/market", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// PlaceLimitOrder places a limit order.
//
// POST /api/v0/equity/orders/limit (rate limit: 1 req / 2s).
func (c *Client) PlaceLimitOrder(ctx context.Context, req *LimitRequest) (*Order, error) {
	var out Order
	if err := c.do(ctx, http.MethodPost, "/equity/orders/limit", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// PlaceStopOrder places a stop order, which becomes a market order once
// StopPrice is reached.
//
// POST /api/v0/equity/orders/stop (rate limit: 1 req / 2s).
func (c *Client) PlaceStopOrder(ctx context.Context, req *StopRequest) (*Order, error) {
	var out Order
	if err := c.do(ctx, http.MethodPost, "/equity/orders/stop", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// PlaceStopLimitOrder places a stop-limit order, which becomes a limit order
// once StopPrice is reached.
//
// POST /api/v0/equity/orders/stop_limit (rate limit: 1 req / 2s).
func (c *Client) PlaceStopLimitOrder(ctx context.Context, req *StopLimitRequest) (*Order, error) {
	var out Order
	if err := c.do(ctx, http.MethodPost, "/equity/orders/stop_limit", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Instruments fetches all instruments tradable on the account. The list is
// large (~5MB) and refreshed every 10 minutes server-side; consider caching.
//
// GET /api/v0/equity/metadata/instruments (rate limit: 1 req / 50s).
func (c *Client) Instruments(ctx context.Context) ([]TradableInstrument, error) {
	var out []TradableInstrument
	if err := c.do(ctx, http.MethodGet, "/equity/metadata/instruments", nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// Exchanges fetches exchange metadata, including each exchange's working
// schedule (market open/close events).
//
// GET /api/v0/equity/metadata/exchanges (rate limit: 1 req / 30s).
func (c *Client) Exchanges(ctx context.Context) ([]Exchange, error) {
	var out []Exchange
	if err := c.do(ctx, http.MethodGet, "/equity/metadata/exchanges", nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}

// HistoricalOrders fetches a page of historical orders. Use params for
// cursor-based pagination, ticker filtering, and per-page limit. For
// ergonomic full-history iteration, see HistoricalOrdersIter.
//
// GET /api/v0/equity/history/orders (rate limit: 50 req / 1m).
func (c *Client) HistoricalOrders(ctx context.Context, params *Orders1Params) (*PaginatedResponseHistoricalOrder, error) {
	q := url.Values{}
	if params != nil {
		if params.Cursor != nil {
			q.Set("cursor", strconv.FormatInt(*params.Cursor, 10))
		}
		if params.Ticker != nil {
			q.Set("ticker", *params.Ticker)
		}
		if params.Limit != nil {
			q.Set("limit", strconv.FormatInt(int64(*params.Limit), 10))
		}
	}

	var out PaginatedResponseHistoricalOrder
	if err := c.do(ctx, http.MethodGet, "/equity/history/orders", q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Dividends fetches a page of dividend history. For ergonomic full-history
// iteration, see DividendsIter.
//
// GET /api/v0/equity/history/dividends (rate limit: 50 req / 1m).
func (c *Client) Dividends(ctx context.Context, params *DividendsParams) (*PaginatedResponseHistoryDividendItem, error) {
	q := url.Values{}
	if params != nil {
		if params.Cursor != nil {
			q.Set("cursor", strconv.FormatInt(*params.Cursor, 10))
		}
		if params.Ticker != nil {
			q.Set("ticker", *params.Ticker)
		}
		if params.Limit != nil {
			q.Set("limit", strconv.FormatInt(int64(*params.Limit), 10))
		}
	}

	var out PaginatedResponseHistoryDividendItem
	if err := c.do(ctx, http.MethodGet, "/equity/history/dividends", q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Transactions fetches a page of cash transactions (deposits, withdrawals,
// fees, internal transfers). For ergonomic full-history iteration, see
// TransactionsIter.
//
// GET /api/v0/equity/history/transactions (rate limit: 50 req / 1m).
func (c *Client) Transactions(ctx context.Context, params *TransactionsParams) (*PaginatedResponseHistoryTransactionItem, error) {
	q := url.Values{}
	if params != nil {
		if params.Cursor != nil {
			q.Set("cursor", *params.Cursor)
		}
		if params.Time != nil {
			q.Set("time", params.Time.Format("2006-01-02T15:04:05Z"))
		}
		if params.Limit != nil {
			q.Set("limit", strconv.FormatInt(int64(*params.Limit), 10))
		}
	}

	var out PaginatedResponseHistoryTransactionItem
	if err := c.do(ctx, http.MethodGet, "/equity/history/transactions", q, nil, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// RequestReport asynchronously kicks off generation of a CSV report covering
// the requested data range. Use Reports to poll for completion and obtain the
// download link, or WaitForReport for a one-shot helper.
//
// POST /api/v0/equity/history/exports (rate limit: 1 req / 30s).
func (c *Client) RequestReport(ctx context.Context, req *PublicReportRequest) (*EnqueuedReportResponse, error) {
	var out EnqueuedReportResponse
	if err := c.do(ctx, http.MethodPost, "/equity/history/exports", nil, req, &out); err != nil {
		return nil, err
	}
	return &out, nil
}

// Reports lists all requested CSV reports and their current status. When a
// report's status is "Finished", its DownloadLink is populated.
//
// GET /api/v0/equity/history/exports (rate limit: 1 req / 1m).
func (c *Client) Reports(ctx context.Context) ([]ReportResponse, error) {
	var out []ReportResponse
	if err := c.do(ctx, http.MethodGet, "/equity/history/exports", nil, nil, &out); err != nil {
		return nil, err
	}
	return out, nil
}
