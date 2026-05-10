package t212

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAccessors_NilReceiverReturnsZero(t *testing.T) {
	var (
		a *AccountSummary
		c *Cash
		p *Position
		o *Order
		i *Instrument
		s *TradableInstrument
		r *ReportResponse
	)

	assert.Equal(t, "", a.GetCurrency())
	assert.Equal(t, int64(0), a.GetID())
	assert.Equal(t, float32(0), a.GetTotalValue())
	assert.Equal(t, Cash{}, a.GetCash())
	assert.Equal(t, Investments{}, a.GetInvestments())

	assert.Equal(t, float32(0), c.GetAvailableToTrade())
	assert.Equal(t, float32(0), c.GetReservedForOrders())
	assert.Equal(t, float32(0), c.GetInPies())

	assert.Equal(t, "", p.GetTicker())
	assert.Equal(t, float32(0), p.GetQuantity())
	assert.Equal(t, float32(0), p.GetAveragePricePaid())
	assert.Equal(t, float32(0), p.GetCurrentPrice())
	assert.Equal(t, time.Time{}, p.GetCreatedAt())

	assert.Equal(t, int64(0), o.GetID())
	assert.Equal(t, "", o.GetTicker())
	assert.Equal(t, float32(0), o.GetQuantity())
	assert.Equal(t, float32(0), o.GetFilledQuantity())
	assert.Equal(t, OrderStatus(""), o.GetStatus())
	assert.Equal(t, OrderSide(""), o.GetSide())
	assert.Equal(t, OrderType(""), o.GetType())
	assert.Equal(t, time.Time{}, o.GetCreatedAt())

	assert.Equal(t, "", i.GetTicker())
	assert.Equal(t, "", i.GetName())
	assert.Equal(t, "", i.GetCurrency())
	assert.Equal(t, "", i.GetIsin())

	assert.Equal(t, "", s.GetTicker())
	assert.Equal(t, "", s.GetName())
	assert.Equal(t, "", s.GetCurrencyCode())

	assert.Equal(t, int64(0), r.GetReportID())
	assert.Equal(t, ReportResponseStatus(""), r.GetStatus())
	assert.Equal(t, "", r.GetDownloadLink())
}

func TestAccessors_PartialNilFields(t *testing.T) {
	// Position with nil instrument returns "" ticker.
	p := &Position{}
	assert.Equal(t, "", p.GetTicker())

	// Position with non-nil instrument but nil ticker returns "".
	p.Instrument = &Instrument{}
	assert.Equal(t, "", p.GetTicker())
}

func TestAccessors_PopulatedReturnsValues(t *testing.T) {
	now := time.Date(2026, 5, 10, 12, 0, 0, 0, time.UTC)
	a := &AccountSummary{
		Id:         ptr[int64](42),
		Currency:   ptr("USD"),
		TotalValue: ptr[float32](1000),
		Cash:       &Cash{AvailableToTrade: ptr[float32](250)},
	}
	assert.Equal(t, "USD", a.GetCurrency())
	assert.Equal(t, int64(42), a.GetID())
	assert.Equal(t, float32(1000), a.GetTotalValue())
	cash := a.GetCash()
	assert.Equal(t, float32(250), cash.GetAvailableToTrade())

	o := &Order{
		Id:        ptr[int64](7),
		Ticker:    ptr("AAPL_US_EQ"),
		Quantity:  ptr[float32](3),
		Status:    ptr(OrderStatus("FILLED")),
		Side:      ptr(OrderSide("BUY")),
		Type:      ptr(OrderType("MARKET")),
		CreatedAt: &now,
	}
	assert.Equal(t, int64(7), o.GetID())
	assert.Equal(t, "AAPL_US_EQ", o.GetTicker())
	assert.Equal(t, float32(3), o.GetQuantity())
	assert.Equal(t, OrderStatus("FILLED"), o.GetStatus())
	assert.Equal(t, OrderSide("BUY"), o.GetSide())
	assert.Equal(t, OrderType("MARKET"), o.GetType())
	assert.Equal(t, now, o.GetCreatedAt())
}
