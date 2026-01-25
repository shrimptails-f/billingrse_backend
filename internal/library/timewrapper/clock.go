package timewrapper

import "time"

// Clock is the default implementation of ClockInterface using the standard time package.
type Clock struct{}

// NewClock creates a new Clock instance that uses real time operations.
func NewClock() ClockInterface {
	return &Clock{}
}

// Now returns the current time.
func (c *Clock) Now() time.Time {
	return time.Now()
}

// After waits for the duration to elapse and then sends the current time on the returned channel.
func (c *Clock) After(d time.Duration) <-chan time.Time {
	return time.After(d)
}
