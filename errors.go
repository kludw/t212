package t212

import "errors"

// Sentinel errors returned by the package. Callers should compare with
// errors.Is to handle specific failure modes.
var (
	// ErrNilOpts is returned when nil is passed where a *ClientOpts is required.
	ErrNilOpts = errors.New("nil client opts")
	// ErrInvalidEnv indicates ClientOpts.Env is set to a value other than
	// "demo" or "live" (case-insensitive).
	ErrInvalidEnv = errors.New("invalid env")
	// ErrEmptyAPIKeyID indicates ClientOpts.APIKeyID is empty.
	ErrEmptyAPIKeyID = errors.New("empty api key id")
	// ErrEmptyAPISecret indicates ClientOpts.APISecret is empty.
	ErrEmptyAPISecret = errors.New("empty api secret")

	// ErrRequest wraps transport-level failures: request construction errors,
	// network errors, and context cancellation.
	ErrRequest = errors.New("request failed")
	// ErrEncode wraps JSON marshaling failures of request bodies.
	ErrEncode = errors.New("encode failed")
	// ErrDecode wraps JSON unmarshaling failures of response bodies.
	ErrDecode = errors.New("decode failed")

	// ErrBadRequest is returned for HTTP 400 responses.
	ErrBadRequest = errors.New("bad request")
	// ErrUnauthorized is returned for HTTP 401 responses (bad API key or
	// environment mismatch — keys are demo-only or live-only).
	ErrUnauthorized = errors.New("unauthorized")
	// ErrForbidden is returned for HTTP 403 responses (missing scope on the
	// API key).
	ErrForbidden = errors.New("forbidden")
	// ErrNotFound is returned for HTTP 404 responses.
	ErrNotFound = errors.New("not found")
	// ErrTimeout is returned for HTTP 408 responses.
	ErrTimeout = errors.New("timeout")
	// ErrRateLimited is returned for HTTP 429 responses. Inspect response
	// headers (x-ratelimit-reset) on the underlying request to back off.
	ErrRateLimited = errors.New("rate limited")
	// ErrUnexpectedStatus is returned for any other non-2xx response.
	ErrUnexpectedStatus = errors.New("unexpected status")
)
