package core

import "time"

// Clock is the engine's source of time. Workflows and the engine take a Clock
// rather than calling the time package directly so tests can advance time
// without waiting for it.
type Clock interface {
	// Now returns the current time.
	Now() time.Time
	// After waits for d to elapse and then sends the time on the returned channel.
	After(d time.Duration) <-chan time.Time
	// Sleep pauses the calling goroutine for at least d.
	Sleep(d time.Duration)
}

// RealClock is the Clock implementation backed by the time package. It is what
// the engine uses outside of tests.
type RealClock struct{}

// NewRealClock returns a Clock backed by the system clock.
func NewRealClock() Clock { return RealClock{} }

// Now returns the current system time.
func (RealClock) Now() time.Time { return time.Now() }

// After waits for d to elapse and then sends the time on the returned channel.
func (RealClock) After(d time.Duration) <-chan time.Time { return time.After(d) }

// Sleep pauses the calling goroutine for at least d.
func (RealClock) Sleep(d time.Duration) { time.Sleep(d) }
