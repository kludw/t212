package t212

import (
	"context"
	"fmt"
	"iter"
	"net/url"
	"strconv"
	"strings"
)

// nextPageCursor extracts the cursor query parameter from a Trading 212
// nextPagePath. The path is expected to be of the form
// "/api/v0/equity/history/...?cursor=...". Returns ("", false) when no
// cursor is present (terminal page).
func nextPageCursor(path string) (string, bool) {
	if path == "" {
		return "", false
	}
	q := path
	if i := strings.Index(path, "?"); i >= 0 {
		q = path[i+1:]
	} else {
		return "", false
	}
	values, err := url.ParseQuery(q)
	if err != nil {
		return "", false
	}
	cursor := values.Get("cursor")
	if cursor == "" {
		return "", false
	}
	return cursor, true
}

// HistoricalOrdersIter returns a Go 1.23+ iterator that walks every page of
// historical orders, yielding each order one at a time. Iteration stops on
// the first error or when the caller breaks. The starting page respects the
// fields in params (cursor, ticker, limit); subsequent pages reuse ticker
// and limit while advancing cursor.
//
// Example:
//
//	for o, err := range client.HistoricalOrdersIter(ctx, nil) {
//	    if err != nil { return err }
//	    process(o)
//	}
func (c *Client) HistoricalOrdersIter(ctx context.Context, params *Orders1Params) iter.Seq2[HistoricalOrder, error] {
	return func(yield func(HistoricalOrder, error) bool) {
		current := cloneOrders1Params(params)
		for {
			page, err := c.HistoricalOrders(ctx, current)
			if err != nil {
				yield(HistoricalOrder{}, err)
				return
			}
			if page.Items != nil {
				for _, item := range *page.Items {
					if !yield(item, nil) {
						return
					}
				}
			}
			cursor, ok := nextCursorFromHistoricalOrders(page)
			if !ok {
				return
			}
			parsed, err := strconv.ParseInt(cursor, 10, 64)
			if err != nil {
				yield(HistoricalOrder{}, fmt.Errorf("%w: invalid cursor %q in nextPagePath", ErrDecode, cursor))
				return
			}
			if current == nil {
				current = &Orders1Params{}
			}
			current.Cursor = &parsed
		}
	}
}

// DividendsIter returns a Go 1.23+ iterator over every dividend item in the
// account's history. See HistoricalOrdersIter for shape and semantics.
func (c *Client) DividendsIter(ctx context.Context, params *DividendsParams) iter.Seq2[HistoryDividendItem, error] {
	return func(yield func(HistoryDividendItem, error) bool) {
		current := cloneDividendsParams(params)
		for {
			page, err := c.Dividends(ctx, current)
			if err != nil {
				yield(HistoryDividendItem{}, err)
				return
			}
			if page.Items != nil {
				for _, item := range *page.Items {
					if !yield(item, nil) {
						return
					}
				}
			}
			cursor, ok := nextCursorFromDividends(page)
			if !ok {
				return
			}
			parsed, err := strconv.ParseInt(cursor, 10, 64)
			if err != nil {
				yield(HistoryDividendItem{}, fmt.Errorf("%w: invalid cursor %q in nextPagePath", ErrDecode, cursor))
				return
			}
			if current == nil {
				current = &DividendsParams{}
			}
			current.Cursor = &parsed
		}
	}
}

// TransactionsIter returns a Go 1.23+ iterator over every cash transaction
// in the account's history. See HistoricalOrdersIter for shape and semantics.
func (c *Client) TransactionsIter(ctx context.Context, params *TransactionsParams) iter.Seq2[HistoryTransactionItem, error] {
	return func(yield func(HistoryTransactionItem, error) bool) {
		current := cloneTransactionsParams(params)
		for {
			page, err := c.Transactions(ctx, current)
			if err != nil {
				yield(HistoryTransactionItem{}, err)
				return
			}
			if page.Items != nil {
				for _, item := range *page.Items {
					if !yield(item, nil) {
						return
					}
				}
			}
			cursor, ok := nextCursorFromTransactions(page)
			if !ok {
				return
			}
			if current == nil {
				current = &TransactionsParams{}
			}
			cursorCopy := cursor
			current.Cursor = &cursorCopy
			// Time is a "starting from" filter; once the first page is
			// fetched, subsequent pages should not re-filter by time —
			// the cursor already encodes position.
			current.Time = nil
		}
	}
}

func nextCursorFromHistoricalOrders(p *PaginatedResponseHistoricalOrder) (string, bool) {
	if p == nil || p.NextPagePath == nil {
		return "", false
	}
	return nextPageCursor(*p.NextPagePath)
}

func nextCursorFromDividends(p *PaginatedResponseHistoryDividendItem) (string, bool) {
	if p == nil || p.NextPagePath == nil {
		return "", false
	}
	return nextPageCursor(*p.NextPagePath)
}

func nextCursorFromTransactions(p *PaginatedResponseHistoryTransactionItem) (string, bool) {
	if p == nil || p.NextPagePath == nil {
		return "", false
	}
	return nextPageCursor(*p.NextPagePath)
}

func cloneOrders1Params(p *Orders1Params) *Orders1Params {
	if p == nil {
		return nil
	}
	out := *p
	return &out
}

func cloneDividendsParams(p *DividendsParams) *DividendsParams {
	if p == nil {
		return nil
	}
	out := *p
	return &out
}

func cloneTransactionsParams(p *TransactionsParams) *TransactionsParams {
	if p == nil {
		return nil
	}
	out := *p
	return &out
}
