// Package t212 is a Go client for the Trading 212 public API.
//
// Trading 212's API is currently in beta. See https://docs.trading212.com for
// the upstream specification.
//
// Construct a Client with NewClient and use its methods to call endpoints.
// All methods take a context.Context and return either generated model types
// (see models.gen.go) or sentinel-wrapped errors that can be inspected with
// errors.Is — see errors.go for the available sentinels.
package t212

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// ClientOpts configures a Client.
type ClientOpts struct {
	// Env selects the Trading 212 environment. Accepts "demo" (default) or
	// "live" (case-insensitive). Empty defaults to demo.
	Env string
	// APIKeyID is the API key identifier issued by Trading 212. API keys
	// generated in a demo account only work against the demo environment, and
	// vice versa.
	APIKeyID string
	// APISecret is the API secret paired with APIKeyID.
	APISecret string
}

// Validate reports whether opts is usable. It returns one of ErrNilOpts,
// ErrInvalidEnv, ErrEmptyAPIKeyID, or ErrEmptyAPISecret on failure.
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

// Client is a Trading 212 API client. It is safe for concurrent use.
type Client struct {
	httpc      http.Client
	authHeader string
	baseURL    string
}

// NewClient validates opts and returns a configured Client. The returned
// Client uses HTTP Basic auth derived from APIKeyID and APISecret and an HTTP
// timeout of DefaultTimeout.
func NewClient(opts *ClientOpts) (*Client, error) {
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	baseURL := demoURL
	if strings.ToLower(opts.Env) == "live" {
		baseURL = liveURL
	}

	creds := base64.StdEncoding.EncodeToString([]byte(opts.APIKeyID + ":" + opts.APISecret))

	return &Client{
		httpc: http.Client{
			Timeout: DefaultTimeout,
		},
		authHeader: "Basic " + creds,
		baseURL:    baseURL,
	}, nil
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
// cursor-based pagination, ticker filtering, and per-page limit.
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

// Dividends fetches a page of dividend history.
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
// fees, internal transfers).
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
// download link.
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
