package t212

import "time"

// Safe accessors for the most-used generated types. The OpenAPI schema
// marks nearly every field as optional, so the generator emits *T pointers.
// These helpers eliminate the nil-deref dance for common reads. Each one
// returns the field's zero value when the receiver, or any pointer along
// the path, is nil.

// GetCurrency returns the account's primary currency, or "" if unknown.
func (a *AccountSummary) GetCurrency() string {
	if a == nil || a.Currency == nil {
		return ""
	}
	return *a.Currency
}

// GetID returns the account ID, or 0 if unknown.
func (a *AccountSummary) GetID() int64 {
	if a == nil || a.Id == nil {
		return 0
	}
	return *a.Id
}

// GetTotalValue returns the account's total value, or 0 if unknown.
func (a *AccountSummary) GetTotalValue() float32 {
	if a == nil || a.TotalValue == nil {
		return 0
	}
	return *a.TotalValue
}

// GetCash returns a copy of the cash breakdown, or a zero Cash if unknown.
func (a *AccountSummary) GetCash() Cash {
	if a == nil || a.Cash == nil {
		return Cash{}
	}
	return *a.Cash
}

// GetInvestments returns a copy of the investments breakdown, or a zero
// Investments if unknown.
func (a *AccountSummary) GetInvestments() Investments {
	if a == nil || a.Investments == nil {
		return Investments{}
	}
	return *a.Investments
}

// GetAvailableToTrade returns the cash available to trade, or 0.
func (c *Cash) GetAvailableToTrade() float32 {
	if c == nil || c.AvailableToTrade == nil {
		return 0
	}
	return *c.AvailableToTrade
}

// GetReservedForOrders returns the cash reserved for pending orders, or 0.
func (c *Cash) GetReservedForOrders() float32 {
	if c == nil || c.ReservedForOrders == nil {
		return 0
	}
	return *c.ReservedForOrders
}

// GetInPies returns the uninvested cash held inside pies, or 0.
func (c *Cash) GetInPies() float32 {
	if c == nil || c.InPies == nil {
		return 0
	}
	return *c.InPies
}

// GetTicker returns the position's instrument ticker, or "" if unknown.
func (p *Position) GetTicker() string {
	if p == nil || p.Instrument == nil || p.Instrument.Ticker == nil {
		return ""
	}
	return *p.Instrument.Ticker
}

// GetQuantity returns the position's share quantity, or 0.
func (p *Position) GetQuantity() float32 {
	if p == nil || p.Quantity == nil {
		return 0
	}
	return *p.Quantity
}

// GetAveragePricePaid returns the position's average price paid, or 0.
func (p *Position) GetAveragePricePaid() float32 {
	if p == nil || p.AveragePricePaid == nil {
		return 0
	}
	return *p.AveragePricePaid
}

// GetCurrentPrice returns the position's current market price, or 0.
func (p *Position) GetCurrentPrice() float32 {
	if p == nil || p.CurrentPrice == nil {
		return 0
	}
	return *p.CurrentPrice
}

// GetCreatedAt returns when the position was opened, or the zero time.
func (p *Position) GetCreatedAt() time.Time {
	if p == nil || p.CreatedAt == nil {
		return time.Time{}
	}
	return *p.CreatedAt
}

// GetID returns the order ID, or 0 if unknown.
func (o *Order) GetID() int64 {
	if o == nil || o.Id == nil {
		return 0
	}
	return *o.Id
}

// GetTicker returns the order's instrument ticker, or "" if unknown.
func (o *Order) GetTicker() string {
	if o == nil || o.Ticker == nil {
		return ""
	}
	return *o.Ticker
}

// GetQuantity returns the order's requested share quantity, or 0.
func (o *Order) GetQuantity() float32 {
	if o == nil || o.Quantity == nil {
		return 0
	}
	return *o.Quantity
}

// GetFilledQuantity returns how many shares of the order have filled, or 0.
func (o *Order) GetFilledQuantity() float32 {
	if o == nil || o.FilledQuantity == nil {
		return 0
	}
	return *o.FilledQuantity
}

// GetStatus returns the order status, or "" if unknown.
func (o *Order) GetStatus() OrderStatus {
	if o == nil || o.Status == nil {
		return ""
	}
	return *o.Status
}

// GetSide returns the order side (BUY / SELL), or "" if unknown.
func (o *Order) GetSide() OrderSide {
	if o == nil || o.Side == nil {
		return ""
	}
	return *o.Side
}

// GetType returns the order type, or "" if unknown.
func (o *Order) GetType() OrderType {
	if o == nil || o.Type == nil {
		return ""
	}
	return *o.Type
}

// GetCreatedAt returns when the order was created, or the zero time.
func (o *Order) GetCreatedAt() time.Time {
	if o == nil || o.CreatedAt == nil {
		return time.Time{}
	}
	return *o.CreatedAt
}

// GetTicker returns the instrument ticker, or "" if unknown.
func (i *Instrument) GetTicker() string {
	if i == nil || i.Ticker == nil {
		return ""
	}
	return *i.Ticker
}

// GetName returns the instrument name, or "" if unknown.
func (i *Instrument) GetName() string {
	if i == nil || i.Name == nil {
		return ""
	}
	return *i.Name
}

// GetCurrency returns the instrument currency, or "" if unknown.
func (i *Instrument) GetCurrency() string {
	if i == nil || i.Currency == nil {
		return ""
	}
	return *i.Currency
}

// GetIsin returns the instrument ISIN, or "" if unknown.
func (i *Instrument) GetIsin() string {
	if i == nil || i.Isin == nil {
		return ""
	}
	return *i.Isin
}

// GetTicker returns the tradable instrument's ticker, or "" if unknown.
func (t *TradableInstrument) GetTicker() string {
	if t == nil || t.Ticker == nil {
		return ""
	}
	return *t.Ticker
}

// GetName returns the tradable instrument's name, or "" if unknown.
func (t *TradableInstrument) GetName() string {
	if t == nil || t.Name == nil {
		return ""
	}
	return *t.Name
}

// GetCurrencyCode returns the tradable instrument's currency code, or "".
func (t *TradableInstrument) GetCurrencyCode() string {
	if t == nil || t.CurrencyCode == nil {
		return ""
	}
	return *t.CurrencyCode
}

// GetReportID returns the report ID, or 0 if unknown.
func (r *ReportResponse) GetReportID() int64 {
	if r == nil || r.ReportId == nil {
		return 0
	}
	return *r.ReportId
}

// GetStatus returns the report's lifecycle status, or "" if unknown.
func (r *ReportResponse) GetStatus() ReportResponseStatus {
	if r == nil || r.Status == nil {
		return ""
	}
	return *r.Status
}

// GetDownloadLink returns the report download link (populated once the
// status is "Finished"), or "" if not yet available.
func (r *ReportResponse) GetDownloadLink() string {
	if r == nil || r.DownloadLink == nil {
		return ""
	}
	return *r.DownloadLink
}
