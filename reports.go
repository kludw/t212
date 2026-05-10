package t212

import (
	"context"
	"errors"
	"strings"
	"time"
)

// WaitForReportOpts tunes the polling behaviour of WaitForReport. Zero
// values fall back to sensible defaults.
type WaitForReportOpts struct {
	// PollInterval between Reports calls. Defaults to 10s. The Reports
	// endpoint is rate-limited at 1 req / 1m, so very short intervals will
	// produce ErrRateLimited errors that the helper transparently retries.
	PollInterval time.Duration

	// MaxWait caps the total time spent waiting. Zero means no cap (the
	// helper still respects ctx). When exceeded, ErrReportTimeout is
	// returned.
	MaxWait time.Duration
}

// WaitForReport polls the Reports endpoint until the report identified by
// reportID reaches status "Finished" and returns the final ReportResponse
// (with DownloadLink populated). Cancellation via ctx is honoured;
// ErrRateLimited responses are swallowed and retried on the next interval.
func (c *Client) WaitForReport(ctx context.Context, reportID int64, opts *WaitForReportOpts) (*ReportResponse, error) {
	interval := 10 * time.Second
	var maxWait time.Duration
	if opts != nil {
		if opts.PollInterval > 0 {
			interval = opts.PollInterval
		}
		maxWait = opts.MaxWait
	}

	deadline := time.Time{}
	if maxWait > 0 {
		deadline = time.Now().Add(maxWait)
	}

	for {
		reports, err := c.Reports(ctx)
		if err != nil && !errors.Is(err, ErrRateLimited) {
			if ctxErr := ctx.Err(); ctxErr != nil {
				return nil, ctxErr
			}
			return nil, err
		}

		if err == nil {
			for i := range reports {
				r := reports[i]
				if r.ReportId == nil || *r.ReportId != reportID {
					continue
				}
				if r.Status != nil && strings.EqualFold(string(*r.Status), "Finished") {
					return &r, nil
				}
				break
			}
		}

		if !deadline.IsZero() && time.Now().After(deadline) {
			return nil, ErrReportTimeout
		}

		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return nil, ctx.Err()
		case <-timer.C:
		}
	}
}
