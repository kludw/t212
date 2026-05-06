package t212

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestClient(serverURL string) *Client {
	return &Client{
		httpc:      http.Client{Timeout: DefaultTimeout},
		authHeader: "Basic a2V5OnNlY3JldA==",
		baseURL:    serverURL,
	}
}

//go:fix inline
func ptr[T any](v T) *T { return new(v) }

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

	c, err := NewClient(&ClientOpts{APIKeyID: key, APISecret: secret})
	require.NoError(t, err)
	assert.Equal(t, want, c.authHeader)
}

func TestNewClient_BaseUrl(t *testing.T) {
	tests := []struct {
		name    string
		env     string
		wantUrl string
	}{
		{name: "default to demo", env: "", wantUrl: demoURL},
		{name: "demo", env: "demo", wantUrl: demoURL},
		{name: "live", env: "live", wantUrl: liveURL},
		{name: "uppercase live", env: "LIVE", wantUrl: liveURL},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			c, err := NewClient(&ClientOpts{Env: tt.env, APIKeyID: "k", APISecret: "s"})
			require.NoError(t, err)
			assert.Equal(t, tt.wantUrl, c.baseURL)
		})
	}
}

func TestNewClient_HTTPTimeout(t *testing.T) {
	c, err := NewClient(&ClientOpts{APIKeyID: "k", APISecret: "s"})
	require.NoError(t, err)
	assert.Equal(t, DefaultTimeout, c.httpc.Timeout)
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
	require.NotNil(t, summary.Id)
	assert.Equal(t, int64(12345678), *summary.Id)
	require.NotNil(t, summary.Currency)
	assert.Equal(t, "GBP", *summary.Currency)
	require.NotNil(t, summary.Cash)
	require.NotNil(t, summary.Cash.AvailableToTrade)
	assert.Equal(t, float32(2500.5), *summary.Cash.AvailableToTrade)
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
	require.NotNil(t, order.Id)
	assert.Equal(t, int64(123), *order.Id)
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
		require.NotNil(t, got.Ticker)
		assert.Equal(t, "AAPL_US_EQ", *got.Ticker)

		fmt.Fprintln(w, `{"id": 999, "ticker": "AAPL_US_EQ"}`)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	order, err := c.PlaceMarketOrder(context.Background(), &MarketRequest{
		Ticker:   ptr("AAPL_US_EQ"),
		Quantity: ptr[float32](5),
	})
	require.NoError(t, err)
	require.NotNil(t, order.Id)
	assert.Equal(t, int64(999), *order.Id)
}

func TestClient_PlaceLimitOrder(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/equity/orders/limit", r.URL.Path)
		fmt.Fprintln(w, `{"id": 1}`)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	_, err := c.PlaceLimitOrder(context.Background(), &LimitRequest{Ticker: ptr("AAPL_US_EQ")})
	require.NoError(t, err)
}

func TestClient_PlaceStopOrder(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/equity/orders/stop", r.URL.Path)
		fmt.Fprintln(w, `{"id": 1}`)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	_, err := c.PlaceStopOrder(context.Background(), &StopRequest{Ticker: ptr("AAPL_US_EQ")})
	require.NoError(t, err)
}

func TestClient_PlaceStopLimitOrder(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/equity/orders/stop_limit", r.URL.Path)
		fmt.Fprintln(w, `{"id": 1}`)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	_, err := c.PlaceStopLimitOrder(context.Background(), &StopLimitRequest{Ticker: ptr("AAPL_US_EQ")})
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
	require.NotNil(t, insts[0].Ticker)
	assert.Equal(t, "AAPL_US_EQ", *insts[0].Ticker)
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
		assert.Equal(t, "50", r.URL.Query().Get("limit"))
		assert.Equal(t, "AAPL_US_EQ", r.URL.Query().Get("ticker"))
		fmt.Fprintln(w, `{"items": [], "nextPagePath": "/foo"}`)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	resp, err := c.HistoricalOrders(context.Background(), &Orders1Params{
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
