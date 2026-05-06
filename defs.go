package t212

import "time"

// DefaultTimeout is the default HTTP timeout used when a Client is constructed
// via NewClient.
const DefaultTimeout = 15 * time.Second

const (
	demoURL = "https://demo.trading212.com/api/v0"
	liveURL = "https://live.trading212.com/api/v0"
)
