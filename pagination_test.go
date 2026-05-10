package t212

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNextPageCursor(t *testing.T) {
	tests := []struct {
		name string
		path string
		want string
		ok   bool
	}{
		{"empty", "", "", false},
		{"no query", "/api/v0/equity/history/orders", "", false},
		{"no cursor", "/api/v0/equity/history/orders?limit=50", "", false},
		{"with cursor", "/api/v0/equity/history/orders?cursor=42&limit=50", "42", true},
		{"only cursor", "/api/v0/equity/history/orders?cursor=abc", "abc", true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, ok := nextPageCursor(tt.path)
			assert.Equal(t, tt.ok, ok)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestHistoricalOrdersIter_WalksPages(t *testing.T) {
	var calls int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/equity/history/orders", r.URL.Path)
		switch atomic.AddInt32(&calls, 1) {
		case 1:
			assert.Empty(t, r.URL.Query().Get("cursor"))
			fmt.Fprintln(w, `{"items":[{},{}],"nextPagePath":"/api/v0/equity/history/orders?cursor=10&limit=50"}`)
		case 2:
			assert.Equal(t, "10", r.URL.Query().Get("cursor"))
			fmt.Fprintln(w, `{"items":[{}],"nextPagePath":"/api/v0/equity/history/orders?cursor=20"}`)
		case 3:
			assert.Equal(t, "20", r.URL.Query().Get("cursor"))
			fmt.Fprintln(w, `{"items":[{}]}`)
		default:
			t.Fatalf("unexpected extra call %d", calls)
		}
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	var seen int
	for _, err := range c.HistoricalOrdersIter(context.Background(), nil) {
		require.NoError(t, err)
		seen++
	}
	assert.Equal(t, 4, seen)
	assert.Equal(t, int32(3), atomic.LoadInt32(&calls))
}

func TestHistoricalOrdersIter_StopsOnEarlyBreak(t *testing.T) {
	var calls int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		atomic.AddInt32(&calls, 1)
		fmt.Fprintln(w, `{"items":[{},{},{}],"nextPagePath":"/api/v0/equity/history/orders?cursor=99"}`)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	var seen int
	for _, err := range c.HistoricalOrdersIter(context.Background(), nil) {
		require.NoError(t, err)
		seen++
		if seen == 2 {
			break
		}
	}
	assert.Equal(t, 2, seen)
	assert.Equal(t, int32(1), atomic.LoadInt32(&calls), "iterator should not fetch the next page after caller breaks")
}

func TestHistoricalOrdersIter_PropagatesError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	var gotErr error
	for _, err := range c.HistoricalOrdersIter(context.Background(), nil) {
		if err != nil {
			gotErr = err
			break
		}
	}
	assert.ErrorIs(t, gotErr, ErrUnexpectedStatus)
}

func TestHistoricalOrdersIter_PreservesInitialFilters(t *testing.T) {
	var calls int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		assert.Equal(t, "AAPL_US_EQ", q.Get("ticker"))
		assert.Equal(t, "5", q.Get("limit"))
		switch atomic.AddInt32(&calls, 1) {
		case 1:
			assert.Empty(t, q.Get("cursor"))
			fmt.Fprintln(w, `{"items":[{}],"nextPagePath":"/api/v0/equity/history/orders?cursor=7&ticker=AAPL_US_EQ&limit=5"}`)
		default:
			assert.Equal(t, "7", q.Get("cursor"))
			fmt.Fprintln(w, `{"items":[{}]}`)
		}
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	var seen int
	for _, err := range c.HistoricalOrdersIter(context.Background(), &Orders1Params{
		Ticker: ptr("AAPL_US_EQ"),
		Limit:  ptr[int32](5),
	}) {
		require.NoError(t, err)
		seen++
	}
	assert.Equal(t, 2, seen)
}

func TestHistoricalOrdersIter_DoesNotMutateInputParams(t *testing.T) {
	var calls int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch atomic.AddInt32(&calls, 1) {
		case 1:
			fmt.Fprintln(w, `{"items":[{}],"nextPagePath":"/api/v0/equity/history/orders?cursor=99"}`)
		default:
			fmt.Fprintln(w, `{"items":[{}]}`)
		}
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	params := &Orders1Params{Ticker: ptr("AAPL_US_EQ")}
	for _, err := range c.HistoricalOrdersIter(context.Background(), params) {
		require.NoError(t, err)
	}
	assert.Nil(t, params.Cursor, "iterator must not mutate caller's params")
	assert.Equal(t, "AAPL_US_EQ", *params.Ticker)
}

func TestDividendsIter_WalksPages(t *testing.T) {
	var calls int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/equity/history/dividends", r.URL.Path)
		switch atomic.AddInt32(&calls, 1) {
		case 1:
			fmt.Fprintln(w, `{"items":[{}],"nextPagePath":"/api/v0/equity/history/dividends?cursor=2"}`)
		default:
			fmt.Fprintln(w, `{"items":[{}]}`)
		}
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	var seen int
	for _, err := range c.DividendsIter(context.Background(), nil) {
		require.NoError(t, err)
		seen++
	}
	assert.Equal(t, 2, seen)
}

func TestTransactionsIter_WalksPagesAndDropsTimeAfterFirstPage(t *testing.T) {
	var calls int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/equity/history/transactions", r.URL.Path)
		q := r.URL.Query()
		switch atomic.AddInt32(&calls, 1) {
		case 1:
			assert.Equal(t, "2026-05-10T00:00:00Z", q.Get("time"))
			fmt.Fprintln(w, `{"items":[{}],"nextPagePath":"/api/v0/equity/history/transactions?cursor=abc"}`)
		case 2:
			assert.Equal(t, "abc", q.Get("cursor"))
			assert.Empty(t, q.Get("time"), "time filter must not be re-sent on subsequent pages")
			fmt.Fprintln(w, `{"items":[{}]}`)
		}
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	when := time.Date(2026, 5, 10, 0, 0, 0, 0, time.UTC)
	var seen int
	for _, err := range c.TransactionsIter(context.Background(), &TransactionsParams{Time: &when}) {
		require.NoError(t, err)
		seen++
	}
	assert.Equal(t, 2, seen)
}
