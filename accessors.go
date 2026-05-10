package t212

import "time"

// Safe accessors for the most-used generated types. The OpenAPI schema
// marks nearly every field as optional, so the generator emits *T pointers.
// These helpers eliminate the nil-deref dance for common reads. Each one
// returns the field's zero value when the receiver, or any pointer along
// the path, is nil.

func (a *AccountSummary) GetCurrency() string {
	if a == nil || a.Currency == nil {
		return ""
	}
	return *a.Currency
}

func (a *AccountSummary) GetID() int64 {
	if a == nil || a.Id == nil {
		return 0
	}
	return *a.Id
}

func (a *AccountSummary) GetTotalValue() float32 {
	if a == nil || a.TotalValue == nil {
		return 0
	}
	return *a.TotalValue
}

func (a *AccountSummary) GetCash() Cash {
	if a == nil || a.Cash == nil {
		return Cash{}
	}
	return *a.Cash
}

func (a *AccountSummary) GetInvestments() Investments {
	if a == nil || a.Investments == nil {
		return Investments{}
	}
	return *a.Investments
}

func (c *Cash) GetAvailableToTrade() float32 {
	if c == nil || c.AvailableToTrade == nil {
		return 0
	}
	return *c.AvailableToTrade
}

func (c *Cash) GetReservedForOrders() float32 {
	if c == nil || c.ReservedForOrders == nil {
		return 0
	}
	return *c.ReservedForOrders
}

func (c *Cash) GetInPies() float32 {
	if c == nil || c.InPies == nil {
		return 0
	}
	return *c.InPies
}

func (p *Position) GetTicker() string {
	if p == nil || p.Instrument == nil || p.Instrument.Ticker == nil {
		return ""
	}
	return *p.Instrument.Ticker
}

func (p *Position) GetQuantity() float32 {
	if p == nil || p.Quantity == nil {
		return 0
	}
	return *p.Quantity
}

func (p *Position) GetAveragePricePaid() float32 {
	if p == nil || p.AveragePricePaid == nil {
		return 0
	}
	return *p.AveragePricePaid
}

func (p *Position) GetCurrentPrice() float32 {
	if p == nil || p.CurrentPrice == nil {
		return 0
	}
	return *p.CurrentPrice
}

func (p *Position) GetCreatedAt() time.Time {
	if p == nil || p.CreatedAt == nil {
		return time.Time{}
	}
	return *p.CreatedAt
}

func (o *Order) GetID() int64 {
	if o == nil || o.Id == nil {
		return 0
	}
	return *o.Id
}

func (o *Order) GetTicker() string {
	if o == nil || o.Ticker == nil {
		return ""
	}
	return *o.Ticker
}

func (o *Order) GetQuantity() float32 {
	if o == nil || o.Quantity == nil {
		return 0
	}
	return *o.Quantity
}

func (o *Order) GetFilledQuantity() float32 {
	if o == nil || o.FilledQuantity == nil {
		return 0
	}
	return *o.FilledQuantity
}

func (o *Order) GetStatus() OrderStatus {
	if o == nil || o.Status == nil {
		return ""
	}
	return *o.Status
}

func (o *Order) GetSide() OrderSide {
	if o == nil || o.Side == nil {
		return ""
	}
	return *o.Side
}

func (o *Order) GetType() OrderType {
	if o == nil || o.Type == nil {
		return ""
	}
	return *o.Type
}

func (o *Order) GetCreatedAt() time.Time {
	if o == nil || o.CreatedAt == nil {
		return time.Time{}
	}
	return *o.CreatedAt
}

// ----- Instrument -----

func (i *Instrument) GetTicker() string {
	if i == nil || i.Ticker == nil {
		return ""
	}
	return *i.Ticker
}

func (i *Instrument) GetName() string {
	if i == nil || i.Name == nil {
		return ""
	}
	return *i.Name
}

func (i *Instrument) GetCurrency() string {
	if i == nil || i.Currency == nil {
		return ""
	}
	return *i.Currency
}

func (i *Instrument) GetIsin() string {
	if i == nil || i.Isin == nil {
		return ""
	}
	return *i.Isin
}

func (t *TradableInstrument) GetTicker() string {
	if t == nil || t.Ticker == nil {
		return ""
	}
	return *t.Ticker
}

func (t *TradableInstrument) GetName() string {
	if t == nil || t.Name == nil {
		return ""
	}
	return *t.Name
}

func (t *TradableInstrument) GetCurrencyCode() string {
	if t == nil || t.CurrencyCode == nil {
		return ""
	}
	return *t.CurrencyCode
}

func (r *ReportResponse) GetReportID() int64 {
	if r == nil || r.ReportId == nil {
		return 0
	}
	return *r.ReportId
}

func (r *ReportResponse) GetStatus() ReportResponseStatus {
	if r == nil || r.Status == nil {
		return ""
	}
	return *r.Status
}

func (r *ReportResponse) GetDownloadLink() string {
	if r == nil || r.DownloadLink == nil {
		return ""
	}
	return *r.DownloadLink
}
