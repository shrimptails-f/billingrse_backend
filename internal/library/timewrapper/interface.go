package timewrapper

import "time"

// ClockInterface abstracts time-related operations for testability.
type ClockInterface interface {
	// Now returns the current time.
	Now() time.Time

	// After waits for the duration to elapse and then sends the current time on the returned channel.
	After(d time.Duration) <-chan time.Time
}
