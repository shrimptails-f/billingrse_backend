package redisclient

import "fmt"

// ErrRedisUnavailable indicates Redis backend is unavailable.
// This error is used to signal that rate limiting cannot proceed due to Redis issues.
type ErrRedisUnavailable struct {
	Err error
}

func (e *ErrRedisUnavailable) Error() string {
	return fmt.Sprintf("redis unavailable: %v", e.Err)
}

func (e *ErrRedisUnavailable) Unwrap() error {
	return e.Err
}
