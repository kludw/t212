package t212

// NewMarketRequest builds a MarketRequest with the required fields set.
// A positive quantity buys; a negative quantity sells.
func NewMarketRequest(ticker string, quantity float32) *MarketRequest {
	return &MarketRequest{
		Ticker:   &ticker,
		Quantity: &quantity,
	}
}

// NewLimitRequest builds a LimitRequest with the required fields set.
func NewLimitRequest(ticker string, quantity, limitPrice float32, validity TimeValidity) *LimitRequest {
	return &LimitRequest{
		Ticker:       &ticker,
		Quantity:     &quantity,
		LimitPrice:   &limitPrice,
		TimeValidity: &validity,
	}
}

// NewStopRequest builds a StopRequest with the required fields set.
func NewStopRequest(ticker string, quantity, stopPrice float32, validity TimeValidity) *StopRequest {
	return &StopRequest{
		Ticker:       &ticker,
		Quantity:     &quantity,
		StopPrice:    &stopPrice,
		TimeValidity: &validity,
	}
}

// NewStopLimitRequest builds a StopLimitRequest with the required fields set.
func NewStopLimitRequest(ticker string, quantity, stopPrice, limitPrice float32, validity TimeValidity) *StopLimitRequest {
	return &StopLimitRequest{
		Ticker:       &ticker,
		Quantity:     &quantity,
		StopPrice:    &stopPrice,
		LimitPrice:   &limitPrice,
		TimeValidity: &validity,
	}
}
