package t212

import "time"

// DefaultTimeout is the default per-request HTTP timeout used when a Client
// is constructed via NewClient with no HTTPClient or RequestTimeout override.
const DefaultTimeout = 15 * time.Second

// DefaultWatcherInterval is the default polling interval for the position
// watcher.
const DefaultWatcherInterval = 3 * time.Second

const (
	demoURL = "https://demo.trading212.com/api/v0"
	liveURL = "https://live.trading212.com/api/v0"

	defaultUserAgent = "Mozilla/5.0 (Linux i654 x86_64) AppleWebKit/602.15 (KHTML, like Gecko) Chrome/52.0.2887.360 Safari/534"
)
