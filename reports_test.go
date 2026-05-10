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

func TestWaitForReport_ReturnsWhenFinished(t *testing.T) {
	var calls int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/equity/history/exports", r.URL.Path)
		switch atomic.AddInt32(&calls, 1) {
		case 1:
			fmt.Fprintln(w, `[{"reportId": 7, "status": "Processing"}]`)
		default:
			fmt.Fprintln(w, `[{"reportId": 7, "status": "Finished", "downloadLink": "https://example.com/r.csv"}]`)
		}
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	r, err := c.WaitForReport(context.Background(), 7, &WaitForReportOpts{PollInterval: 5 * time.Millisecond})
	require.NoError(t, err)
	assert.Equal(t, "https://example.com/r.csv", r.GetDownloadLink())
}

func TestWaitForReport_TolerantToRateLimit(t *testing.T) {
	var calls int32
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch atomic.AddInt32(&calls, 1) {
		case 1:
			w.WriteHeader(http.StatusTooManyRequests)
		default:
			fmt.Fprintln(w, `[{"reportId": 9, "status": "Finished", "downloadLink": "ok"}]`)
		}
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	r, err := c.WaitForReport(context.Background(), 9, &WaitForReportOpts{PollInterval: 5 * time.Millisecond})
	require.NoError(t, err)
	assert.Equal(t, "ok", r.GetDownloadLink())
}

func TestWaitForReport_HonoursMaxWait(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `[{"reportId": 1, "status": "Processing"}]`)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	_, err := c.WaitForReport(context.Background(), 1, &WaitForReportOpts{
		PollInterval: 10 * time.Millisecond,
		MaxWait:      30 * time.Millisecond,
	})
	assert.ErrorIs(t, err, ErrReportTimeout)
}

func TestWaitForReport_HonoursContextCancellation(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, `[{"reportId": 1, "status": "Processing"}]`)
	}))
	defer ts.Close()

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(20 * time.Millisecond)
		cancel()
	}()

	c := newTestClient(ts.URL)
	_, err := c.WaitForReport(ctx, 1, &WaitForReportOpts{PollInterval: 10 * time.Millisecond})
	assert.ErrorIs(t, err, context.Canceled)
}

func TestWaitForReport_PropagatesNonRateLimitError(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	_, err := c.WaitForReport(context.Background(), 1, nil)
	assert.ErrorIs(t, err, ErrUnauthorized)
}
