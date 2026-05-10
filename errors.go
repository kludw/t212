package t212

import "errors"

// Sentinel errors returned by the package. Match with errors.Is.
var (
	// ErrNilOpts is returned when ClientOpts is nil.
	ErrNilOpts = errors.New("nil client opts")
	// ErrInvalidEnv is returned when ClientOpts.Env is set to a value
	// other than "demo" or "live" (case-insensitive).
	ErrInvalidEnv = errors.New("invalid env")
	// ErrEmptyAPIKeyID is returned when ClientOpts.APIKeyID is empty.
	ErrEmptyAPIKeyID = errors.New("empty api key id")
	// ErrEmptyAPISecret is returned when ClientOpts.APISecret is empty.
	ErrEmptyAPISecret = errors.New("empty api secret")

	// ErrRequest wraps any failure to build or execute the underlying
	// HTTP request (e.g. transport errors, context cancellation).
	ErrRequest = errors.New("request failed")
	// ErrEncode wraps a JSON marshal failure on the request body.
	ErrEncode = errors.New("encode failed")
	// ErrDecode wraps a JSON unmarshal failure on the response body.
	ErrDecode = errors.New("decode failed")

	// ErrBadRequest is returned for HTTP 400 responses.
	ErrBadRequest = errors.New("bad request")
	// ErrUnauthorized is returned for HTTP 401 responses.
	ErrUnauthorized = errors.New("unauthorized")
	// ErrForbidden is returned for HTTP 403 responses.
	ErrForbidden = errors.New("forbidden")
	// ErrNotFound is returned for HTTP 404 responses.
	ErrNotFound = errors.New("not found")
	// ErrTimeout is returned for HTTP 408 responses.
	ErrTimeout = errors.New("timeout")
	// ErrRateLimited is returned for HTTP 429 responses.
	ErrRateLimited = errors.New("rate limited")
	// ErrUnexpectedStatus is returned for any other non-2xx status.
	ErrUnexpectedStatus = errors.New("unexpected status")

	// ErrReportTimeout is returned by WaitForReport when the configured
	// MaxWait is exceeded before the report reaches "Finished".
	ErrReportTimeout = errors.New("report did not finish before timeout")
)
