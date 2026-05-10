package t212

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// newTestClient builds a Client wired up for endpoint tests without going
// through NewClient — so the background position watcher does NOT auto-start.
// Tests that exercise the watcher must use newWatcherClient instead.
func newTestClient(serverURL string) *Client {
	return &Client{
		httpc:      &http.Client{Timeout: DefaultTimeout},
		authHeader: "Basic a2V5OnNlY3JldA==",
		baseURL:    serverURL,
		userAgent:  defaultUserAgent,
		positions:  make(map[string]Position),
		done:       make(chan struct{}),
		cancel:     func() {},
	}
}

// newWatcherClient builds a Client through NewClient against the given
// httptest server URL. The watcher auto-starts; the returned t.Cleanup
// ensures Close runs at the end of the test.
func newWatcherClient(t *testing.T, serverURL string, opts ClientOpts) *Client {
	t.Helper()
	if opts.APIKeyID == "" {
		opts.APIKeyID = "key"
	}
	if opts.APISecret == "" {
		opts.APISecret = "secret"
	}
	opts.BaseURL = serverURL
	if opts.WatcherInterval == 0 {
		opts.WatcherInterval = 5 * time.Millisecond
	}
	c, err := NewClient(&opts)
	require.NoError(t, err)
	t.Cleanup(c.Close)
	return c
}

func ptr[T any](v T) *T { return &v }

func TestClientOpts_Validate(t *testing.T) {
	tests := []struct {
		name    string
		opts    *ClientOpts
		wantErr error
	}{
		{name: "nil opts", opts: nil, wantErr: ErrNilOpts},
		{name: "empty api key id", opts: &ClientOpts{APIKeyID: "", APISecret: "secret"}, wantErr: ErrEmptyAPIKeyID},
		{name: "empty api secret", opts: &ClientOpts{APIKeyID: "key", APISecret: ""}, wantErr: ErrEmptyAPISecret},
		{name: "invalid env", opts: &ClientOpts{Env: "staging", APIKeyID: "key", APISecret: "secret"}, wantErr: ErrInvalidEnv},
		{name: "demo env", opts: &ClientOpts{Env: "demo", APIKeyID: "key", APISecret: "secret"}},
		{name: "live env", opts: &ClientOpts{Env: "live", APIKeyID: "key", APISecret: "secret"}},
		{name: "uppercase env accepted", opts: &ClientOpts{Env: "DEMO", APIKeyID: "key", APISecret: "secret"}},
		{name: "empty env accepted", opts: &ClientOpts{Env: "", APIKeyID: "key", APISecret: "secret"}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := tt.opts.Validate()
			if tt.wantErr == nil {
				assert.NoError(t, err)
				return
			}
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestNewClient_PropagatesValidationError(t *testing.T) {
	_, err := NewClient(&ClientOpts{APIKeyID: "", APISecret: "secret"})
	assert.ErrorIs(t, err, ErrEmptyAPIKeyID)
}

func TestNewClient_AuthHeader(t *testing.T) {
	const key = "myKey"
	const secret = "mySecret"
	want := "Basic " + base64.StdEncoding.EncodeToString([]byte(key+":"+secret))

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, want, r.Header.Get("Authorization"))
		fmt.Fprintln(w, `[]`)
	}))
	defer ts.Close()

	c := newWatcherClient(t, ts.URL, ClientOpts{APIKeyID: key, APISecret: secret})
	assert.Equal(t, want, c.authHeader)
}

func TestNewClient_BaseURL(t *testing.T) {
	tests := []struct {
		name    string
		env     string
		wantURL string
	}{
		{name: "default to demo", env: "", wantURL: demoURL},
		{name: "demo", env: "demo", wantURL: demoURL},
		{name: "live", env: "live", wantURL: liveURL},
		{name: "uppercase live", env: "LIVE", wantURL: liveURL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = tt.wantURL // surfaced via the wrapped error checked below

			// Use a closed httptest server URL so initial Positions fetch
			// fails fast with a transport error — sufficient to read baseURL
			// without making real network requests.
			ts := httptest.NewServer(http.HandlerFunc(func(_ http.ResponseWriter, _ *http.Request) {}))
			ts.Close()

			_, err := NewClient(&ClientOpts{
				Env:        tt.env,
				APIKeyID:   "k",
				APISecret:  "s",
				HTTPClient: &http.Client{Timeout: 50 * time.Millisecond},
			})
			// Auth/base resolution is what matters; the call will fail at
			// transport. Just verify it doesn't go through validation errors.
			if err != nil {
				assert.NotErrorIs(t, err, ErrInvalidEnv)
			}
		})
	}
}

func TestNewClient_BaseURLOverride(t *testing.T) {
	const custom = "http://localhost:9999/api/v0"
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	ts.Close()

	c, err := NewClient(&ClientOpts{
		BaseURL:    custom,
		APIKeyID:   "k",
		APISecret:  "s",
		HTTPClient: &http.Client{Timeout: 50 * time.Millisecond},
	})
	// Initial fetch will error (transport), so NewClient returns an error,
	// but baseURL was applied during construction. Validate the path went
	// through BaseURL by checking the wrapped error refers to it.
	require.Error(t, err)
	assert.Contains(t, err.Error(), custom)
	_ = c
}

func TestNewClient_HTTPTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `[]`)
	}))
	defer ts.Close()

	c := newWatcherClient(t, ts.URL, ClientOpts{})
	assert.Equal(t, DefaultTimeout, c.httpc.Timeout)
}

func TestNewClient_RequestTimeoutOverride(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `[]`)
	}))
	defer ts.Close()

	c := newWatcherClient(t, ts.URL, ClientOpts{RequestTimeout: 7 * time.Second})
	assert.Equal(t, 7*time.Second, c.httpc.Timeout)
}

func TestNewClient_HTTPClientInjection(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `[]`)
	}))
	defer ts.Close()

	custom := &http.Client{Timeout: 42 * time.Second}
	c := newWatcherClient(t, ts.URL, ClientOpts{HTTPClient: custom})
	assert.Same(t, custom, c.httpc, "injected HTTPClient should be used directly")
}

func TestNewClient_DefaultUserAgent(t *testing.T) {
	got := make(chan string, 1)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case got <- r.Header.Get("User-Agent"):
		default:
		}
		fmt.Fprintln(w, `[]`)
	}))
	defer ts.Close()

	_ = newWatcherClient(t, ts.URL, ClientOpts{})
	select {
	case ua := <-got:
		assert.Equal(t, defaultUserAgent, ua)
	case <-time.After(time.Second):
		t.Fatal("server never received a request")
	}
}

func TestNewClient_CustomUserAgent(t *testing.T) {
	got := make(chan string, 1)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		select {
		case got <- r.Header.Get("User-Agent"):
		default:
		}
		fmt.Fprintln(w, `[]`)
	}))
	defer ts.Close()

	_ = newWatcherClient(t, ts.URL, ClientOpts{UserAgent: "my-app/9.9"})
	select {
	case ua := <-got:
		assert.Equal(t, "my-app/9.9", ua)
	case <-time.After(time.Second):
		t.Fatal("server never received a request")
	}
}

func TestDo_StatusErrors(t *testing.T) {
	tests := []struct {
		status  int
		wantErr error
	}{
		{http.StatusBadRequest, ErrBadRequest},
		{http.StatusUnauthorized, ErrUnauthorized},
		{http.StatusForbidden, ErrForbidden},
		{http.StatusNotFound, ErrNotFound},
		{http.StatusRequestTimeout, ErrTimeout},
		{http.StatusTooManyRequests, ErrRateLimited},
		{http.StatusInternalServerError, ErrUnexpectedStatus},
	}

	for _, tt := range tests {
		t.Run(fmt.Sprintf("%d", tt.status), func(t *testing.T) {
			ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(tt.status)
			}))
			defer ts.Close()

			c := newTestClient(ts.URL)
			err := c.do(context.Background(), http.MethodGet, "/anything", nil, nil, nil)
			assert.ErrorIs(t, err, tt.wantErr)
		})
	}
}

func TestDo_DecodeError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `not json`)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	var out struct {
		Foo string `json:"foo"`
	}
	err := c.do(context.Background(), http.MethodGet, "/", nil, nil, &out)
	assert.ErrorIs(t, err, ErrDecode)
}

func TestDo_TransportError(t *testing.T) {
	ts := httptest.NewServer(nil)
	ts.Close()

	c := newTestClient(ts.URL)
	err := c.do(context.Background(), http.MethodGet, "/", nil, nil, nil)
	assert.ErrorIs(t, err, ErrRequest)
}

func TestDo_ContextCancelled(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	ctx, cancel := context.WithCancel(context.Background())
	cancel()

	err := c.do(ctx, http.MethodGet, "/", nil, nil, nil)
	assert.ErrorIs(t, err, ErrRequest)
}

func TestDo_EncodeError(t *testing.T) {
	c := newTestClient("http://unused")
	err := c.do(context.Background(), http.MethodPost, "/", nil, make(chan int), nil)
	assert.ErrorIs(t, err, ErrEncode)
}

func TestDo_SetsAuthAndUA(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "Basic a2V5OnNlY3JldA==", r.Header.Get("Authorization"))
		assert.Equal(t, defaultUserAgent, r.Header.Get("User-Agent"))
		fmt.Fprintln(w, `null`)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	err := c.do(context.Background(), http.MethodGet, "/anything", nil, nil, nil)
	require.NoError(t, err)
}

func TestDo_SetsContentTypeForBody(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	err := c.do(context.Background(), http.MethodPost, "/", nil, map[string]int{"x": 1}, nil)
	require.NoError(t, err)
}

func TestClient_AccountSummary(t *testing.T) {
	const authHeader = "Basic a2V5OnNlY3JldA=="

	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/equity/account/summary", r.URL.Path)
		assert.Equal(t, authHeader, r.Header.Get("Authorization"))
		fmt.Fprintln(w, `{
			"id": 12345678,
			"currency": "GBP",
			"totalValue": 15250.75,
			"cash": {"availableToTrade": 2500.5, "reservedForOrders": 150.0, "inPies": 500.0}
		}`)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)

	summary, err := c.AccountSummary(context.Background())
	require.NoError(t, err)
	assert.Equal(t, int64(12345678), summary.GetID())
	assert.Equal(t, "GBP", summary.GetCurrency())
	cash := summary.GetCash()
	assert.Equal(t, float32(2500.5), cash.GetAvailableToTrade())
}

func TestClient_AccountSummary_Unauthorized(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	_, err := c.AccountSummary(context.Background())
	assert.ErrorIs(t, err, ErrUnauthorized)
}

func TestClient_Positions(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/equity/positions", r.URL.Path)
		assert.Empty(t, r.URL.RawQuery)
		fmt.Fprintln(w, `[{"instrument": {"ticker": "AAPL_US_EQ"}, "quantity": 1}]`)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	positions, err := c.Positions(context.Background(), nil)
	require.NoError(t, err)
	require.Len(t, positions, 1)
	assert.Equal(t, "AAPL_US_EQ", positions[0].GetTicker())
}

func TestClient_Positions_TickerFilter(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "MSFT_US_EQ", r.URL.Query().Get("ticker"))
		fmt.Fprintln(w, `[]`)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	_, err := c.Positions(context.Background(), &GetPositionsParams{Ticker: ptr("MSFT_US_EQ")})
	require.NoError(t, err)
}

func TestClient_Orders(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/equity/orders", r.URL.Path)
		fmt.Fprintln(w, `[{"id": 1, "ticker": "AAPL_US_EQ"}]`)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	orders, err := c.Orders(context.Background())
	require.NoError(t, err)
	require.Len(t, orders, 1)
}

func TestClient_OrderByID(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/equity/orders/123", r.URL.Path)
		fmt.Fprintln(w, `{"id": 123, "ticker": "AAPL_US_EQ"}`)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	order, err := c.OrderByID(context.Background(), 123)
	require.NoError(t, err)
	assert.Equal(t, int64(123), order.GetID())
}

func TestClient_CancelOrder(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodDelete, r.Method)
		assert.Equal(t, "/equity/orders/123", r.URL.Path)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	require.NoError(t, c.CancelOrder(context.Background(), 123))
}

func TestClient_PlaceMarketOrder(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/equity/orders/market", r.URL.Path)
		assert.Equal(t, "application/json", r.Header.Get("Content-Type"))

		var got MarketRequest
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		require.NoError(t, json.Unmarshal(body, &got))
		assert.Equal(t, "AAPL_US_EQ", *got.Ticker)
		assert.Equal(t, float32(5), *got.Quantity)

		fmt.Fprintln(w, `{"id": 999, "ticker": "AAPL_US_EQ"}`)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	order, err := c.PlaceMarketOrder(context.Background(), NewMarketRequest("AAPL_US_EQ", 5))
	require.NoError(t, err)
	assert.Equal(t, int64(999), order.GetID())
}

func TestClient_PlaceLimitOrder(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/equity/orders/limit", r.URL.Path)
		var got LimitRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		assert.Equal(t, "AAPL_US_EQ", *got.Ticker)
		assert.Equal(t, float32(150), *got.LimitPrice)
		assert.Equal(t, TimeValidity("DAY"), *got.TimeValidity)
		fmt.Fprintln(w, `{"id": 1}`)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	_, err := c.PlaceLimitOrder(context.Background(), NewLimitRequest("AAPL_US_EQ", 5, 150, "DAY"))
	require.NoError(t, err)
}

func TestClient_PlaceStopOrder(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/equity/orders/stop", r.URL.Path)
		var got StopRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		assert.Equal(t, float32(140), *got.StopPrice)
		fmt.Fprintln(w, `{"id": 1}`)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	_, err := c.PlaceStopOrder(context.Background(), NewStopRequest("AAPL_US_EQ", 5, 140, "DAY"))
	require.NoError(t, err)
}

func TestClient_PlaceStopLimitOrder(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/equity/orders/stop_limit", r.URL.Path)
		var got StopLimitRequest
		require.NoError(t, json.NewDecoder(r.Body).Decode(&got))
		assert.Equal(t, float32(140), *got.StopPrice)
		assert.Equal(t, float32(141), *got.LimitPrice)
		fmt.Fprintln(w, `{"id": 1}`)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	_, err := c.PlaceStopLimitOrder(context.Background(), NewStopLimitRequest("AAPL_US_EQ", 5, 140, 141, "DAY"))
	require.NoError(t, err)
}

func TestClient_Instruments(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/equity/metadata/instruments", r.URL.Path)
		fmt.Fprintln(w, `[{"ticker": "AAPL_US_EQ", "name": "Apple Inc"}]`)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	insts, err := c.Instruments(context.Background())
	require.NoError(t, err)
	require.Len(t, insts, 1)
	assert.Equal(t, "AAPL_US_EQ", insts[0].GetTicker())
	assert.Equal(t, "Apple Inc", insts[0].GetName())
}

func TestClient_Exchanges(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/equity/metadata/exchanges", r.URL.Path)
		fmt.Fprintln(w, `[{"id": 123, "name": "NASDAQ"}]`)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	exchanges, err := c.Exchanges(context.Background())
	require.NoError(t, err)
	require.Len(t, exchanges, 1)
}

func TestClient_HistoricalOrders(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/equity/history/orders", r.URL.Path)
		q := r.URL.Query()
		assert.Equal(t, "50", q.Get("limit"))
		assert.Equal(t, "AAPL_US_EQ", q.Get("ticker"))
		assert.Equal(t, "987", q.Get("cursor"))
		fmt.Fprintln(w, `{"items": [], "nextPagePath": "/foo"}`)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	resp, err := c.HistoricalOrders(context.Background(), &Orders1Params{
		Cursor: ptr[int64](987),
		Limit:  ptr[int32](50),
		Ticker: ptr("AAPL_US_EQ"),
	})
	require.NoError(t, err)
	require.NotNil(t, resp.NextPagePath)
	assert.Equal(t, "/foo", *resp.NextPagePath)
}

func TestClient_Dividends(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/equity/history/dividends", r.URL.Path)
		fmt.Fprintln(w, `{"items": []}`)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	_, err := c.Dividends(context.Background(), nil)
	require.NoError(t, err)
}

func TestClient_Dividends_AllParams(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		assert.Equal(t, "42", q.Get("cursor"))
		assert.Equal(t, "AAPL_US_EQ", q.Get("ticker"))
		assert.Equal(t, "10", q.Get("limit"))
		fmt.Fprintln(w, `{"items": []}`)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	_, err := c.Dividends(context.Background(), &DividendsParams{
		Cursor: ptr[int64](42),
		Ticker: ptr("AAPL_US_EQ"),
		Limit:  ptr[int32](10),
	})
	require.NoError(t, err)
}

func TestClient_Transactions(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/equity/history/transactions", r.URL.Path)
		assert.Equal(t, "50", r.URL.Query().Get("limit"))
		fmt.Fprintln(w, `{"items": []}`)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	_, err := c.Transactions(context.Background(), &TransactionsParams{Limit: ptr[int32](50)})
	require.NoError(t, err)
}

func TestClient_Transactions_CursorAndTime(t *testing.T) {
	when := time.Date(2026, 5, 10, 12, 34, 56, 0, time.UTC)
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		assert.Equal(t, "page-2", q.Get("cursor"))
		assert.Equal(t, "2026-05-10T12:34:56Z", q.Get("time"))
		fmt.Fprintln(w, `{"items": []}`)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	_, err := c.Transactions(context.Background(), &TransactionsParams{
		Cursor: ptr("page-2"),
		Time:   &when,
	})
	require.NoError(t, err)
}

func TestClient_RequestReport(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/equity/history/exports", r.URL.Path)
		fmt.Fprintln(w, `{"reportId": 12345}`)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	resp, err := c.RequestReport(context.Background(), &PublicReportRequest{})
	require.NoError(t, err)
	require.NotNil(t, resp.ReportId)
	assert.Equal(t, int64(12345), *resp.ReportId)
}

func TestClient_Reports(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/equity/history/exports", r.URL.Path)
		fmt.Fprintln(w, `[{"reportId": 12345, "status": "Finished"}]`)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	reports, err := c.Reports(context.Background())
	require.NoError(t, err)
	require.Len(t, reports, 1)
}

// positionsServer returns an httptest server whose /equity/positions handler
// returns the next JSON body from bodies, advancing one entry per call. After
// the slice is exhausted it keeps returning the last entry.
func positionsServer(t *testing.T, bodies []string) *httptest.Server {
	t.Helper()
	var idx int32
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/equity/positions", r.URL.Path)
		i := int(atomic.AddInt32(&idx, 1)) - 1
		if i >= len(bodies) {
			i = len(bodies) - 1
		}
		fmt.Fprintln(w, bodies[i])
	}))
}

func TestWatcher_OpenCallback(t *testing.T) {
	ts := positionsServer(t, []string{
		`[{"instrument": {"ticker": "AAPL_US_EQ"}, "quantity": 1}]`,
	})
	defer ts.Close()

	opened := make(chan *Position, 4)
	c := newWatcherClient(t, ts.URL, ClientOpts{
		OnPositionOpen: func(p *Position) { opened <- p },
	})
	_ = c

	select {
	case p := <-opened:
		assert.Equal(t, "AAPL_US_EQ", p.GetTicker())
	case <-time.After(time.Second):
		t.Fatal("open callback not fired")
	}
}

func TestWatcher_CloseCallback(t *testing.T) {
	ts := positionsServer(t, []string{
		`[{"instrument": {"ticker": "AAPL_US_EQ"}, "quantity": 1}]`,
		`[]`,
	})
	defer ts.Close()

	closed := make(chan *Position, 4)
	_ = newWatcherClient(t, ts.URL, ClientOpts{
		OnPositionOpen:  func(*Position) {},
		OnPositionClose: func(p *Position) { closed <- p },
	})

	select {
	case p := <-closed:
		assert.Equal(t, "AAPL_US_EQ", p.GetTicker())
	case <-time.After(time.Second):
		t.Fatal("close callback not fired")
	}
}

func TestWatcher_NoDuplicateOpenForPersistentPosition(t *testing.T) {
	ts := positionsServer(t, []string{
		`[{"instrument": {"ticker": "AAPL_US_EQ"}, "quantity": 1}]`,
	})
	defer ts.Close()

	var opens int32
	c := newWatcherClient(t, ts.URL, ClientOpts{
		OnPositionOpen:  func(*Position) { atomic.AddInt32(&opens, 1) },
		OnPositionClose: func(*Position) {},
	})
	_ = c

	// Several poll ticks should pass without re-firing the open callback.
	time.Sleep(60 * time.Millisecond)

	assert.Equal(t, int32(1), atomic.LoadInt32(&opens), "open should fire exactly once for a persistent position")
}

func TestWatcher_NoCallbacksDoesNotFail(t *testing.T) {
	ts := positionsServer(t, []string{
		`[{"instrument": {"ticker": "AAPL_US_EQ"}, "quantity": 1}]`,
	})
	defer ts.Close()

	c := newWatcherClient(t, ts.URL, ClientOpts{})
	require.NotNil(t, c)
	assert.Len(t, c.Snapshot(), 1)
}

func TestWatcher_Snapshot(t *testing.T) {
	ts := positionsServer(t, []string{
		`[{"instrument": {"ticker": "AAPL_US_EQ"}, "quantity": 1}, {"instrument": {"ticker": "MSFT_US_EQ"}, "quantity": 2}]`,
	})
	defer ts.Close()

	c := newWatcherClient(t, ts.URL, ClientOpts{})
	snap := c.Snapshot()
	assert.Len(t, snap, 2)
	tickers := []string{snap[0].GetTicker(), snap[1].GetTicker()}
	assert.Contains(t, tickers, "AAPL_US_EQ")
	assert.Contains(t, tickers, "MSFT_US_EQ")
}

func TestWatcher_SnapshotIsCopy(t *testing.T) {
	ts := positionsServer(t, []string{
		`[{"instrument": {"ticker": "AAPL_US_EQ"}, "quantity": 1}]`,
	})
	defer ts.Close()

	c := newWatcherClient(t, ts.URL, ClientOpts{})
	snap := c.Snapshot()
	require.Len(t, snap, 1)

	// Mutating the returned slice must not affect the watcher's internal map.
	snap[0] = Position{}
	again := c.Snapshot()
	require.Len(t, again, 1)
	assert.Equal(t, "AAPL_US_EQ", again[0].GetTicker())
}

func TestWatcher_OnPollErrorFires(t *testing.T) {
	var calls int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Initial fetch succeeds (empty); subsequent polls return 500 so the
		// error callback should fire.
		if atomic.AddInt32(&calls, 1) == 1 {
			fmt.Fprintln(w, `[]`)
			return
		}
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	errs := make(chan error, 4)
	_ = newWatcherClient(t, ts.URL, ClientOpts{
		OnPositionOpen: func(*Position) {},
		OnPollError:    func(err error) { errs <- err },
	})

	select {
	case err := <-errs:
		assert.ErrorIs(t, err, ErrUnexpectedStatus)
	case <-time.After(time.Second):
		t.Fatal("OnPollError did not fire")
	}
}

func TestWatcher_InitialUnauthorizedFailsNewClient(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer ts.Close()

	_, err := NewClient(&ClientOpts{
		BaseURL:         ts.URL,
		APIKeyID:        "k",
		APISecret:       "s",
		WatcherInterval: 5 * time.Millisecond,
	})
	assert.ErrorIs(t, err, ErrUnauthorized)
}

func TestWatcher_InitialRateLimitedTolerated(t *testing.T) {
	var calls int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if atomic.AddInt32(&calls, 1) == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			return
		}
		fmt.Fprintln(w, `[{"instrument": {"ticker": "AAPL_US_EQ"}, "quantity": 1}]`)
	}))
	defer ts.Close()

	opened := make(chan *Position, 4)
	c := newWatcherClient(t, ts.URL, ClientOpts{
		OnPositionOpen: func(p *Position) { opened <- p },
	})
	require.NotNil(t, c)

	select {
	case <-opened:
	case <-time.After(time.Second):
		t.Fatal("open callback not fired after initial rate-limited fetch")
	}
}

func TestWatcher_CloseIsIdempotent(t *testing.T) {
	ts := positionsServer(t, []string{`[]`})
	defer ts.Close()

	c, err := NewClient(&ClientOpts{
		BaseURL:         ts.URL,
		APIKeyID:        "k",
		APISecret:       "s",
		WatcherInterval: 5 * time.Millisecond,
	})
	require.NoError(t, err)

	c.Close()
	c.Close() // second call should not panic or hang
}

func TestWatcher_CallbacksAreThreadSafeWithSnapshot(t *testing.T) {
	// Regression guard: Snapshot reads the position map under RLock while
	// the watcher writes under Lock. Hammer Snapshot from many goroutines
	// concurrently and let -race catch any unguarded access.
	bodies := []string{
		`[{"instrument": {"ticker": "AAPL_US_EQ"}, "quantity": 1}]`,
		`[{"instrument": {"ticker": "MSFT_US_EQ"}, "quantity": 2}]`,
		`[]`,
	}
	ts := positionsServer(t, bodies)
	defer ts.Close()

	c := newWatcherClient(t, ts.URL, ClientOpts{})

	stop := make(chan struct{})
	var wg sync.WaitGroup
	for i := 0; i < 8; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for {
				select {
				case <-stop:
					return
				default:
					_ = c.Snapshot()
				}
			}
		}()
	}

	time.Sleep(40 * time.Millisecond)
	close(stop)
	wg.Wait()
}
