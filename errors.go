package t212

import "errors"

var (
	ErrNilOpts        = errors.New("nil client opts")
	ErrInvalidEnv     = errors.New("invalid env")
	ErrEmptyAPIKeyID  = errors.New("empty api key id")
	ErrEmptyAPISecret = errors.New("empty api secret")

	ErrRequest = errors.New("request failed")
	ErrEncode  = errors.New("encode failed")
	ErrDecode  = errors.New("decode failed")

	ErrBadRequest       = errors.New("bad request")
	ErrUnauthorized     = errors.New("unauthorized")
	ErrForbidden        = errors.New("forbidden")
	ErrNotFound         = errors.New("not found")
	ErrTimeout          = errors.New("timeout")
	ErrRateLimited      = errors.New("rate limited")
	ErrUnexpectedStatus = errors.New("unexpected status")

	ErrReportTimeout = errors.New("report did not finish before timeout")
)
