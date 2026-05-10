package t212

func NewMarketRequest(ticker string, quantity float32) *MarketRequest {
	return &MarketRequest{
		Ticker:   &ticker,
		Quantity: &quantity,
	}
}

func NewLimitRequest(ticker string, quantity, limitPrice float32, validity TimeValidity) *LimitRequest {
	return &LimitRequest{
		Ticker:       &ticker,
		Quantity:     &quantity,
		LimitPrice:   &limitPrice,
		TimeValidity: &validity,
	}
}

func NewStopRequest(ticker string, quantity, stopPrice float32, validity TimeValidity) *StopRequest {
	return &StopRequest{
		Ticker:       &ticker,
		Quantity:     &quantity,
		StopPrice:    &stopPrice,
		TimeValidity: &validity,
	}
}

func NewStopLimitRequest(ticker string, quantity, stopPrice, limitPrice float32, validity TimeValidity) *StopLimitRequest {
	return &StopLimitRequest{
		Ticker:       &ticker,
		Quantity:     &quantity,
		StopPrice:    &stopPrice,
		LimitPrice:   &limitPrice,
		TimeValidity: &validity,
	}
}
