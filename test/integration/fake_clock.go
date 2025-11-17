package integration

import (
	"sync"
	"time"
)

type timer struct {
	deadline time.Time
	ch       chan time.Time
}

type FakeClock struct {
	mu     sync.Mutex
	now    time.Time
	timers []*timer
}

func NewFakeClock(start time.Time) *FakeClock {
	c := &FakeClock{
		now: start,
	}

	// Auto-start ticking every 100ms
	c.Tick(10 * time.Millisecond)

	return c
}

// Now just returns the current fake time
func (c *FakeClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.now
}

// After creates a timer that fires when fake time reaches now + d
func (c *FakeClock) After(d time.Duration) <-chan time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()

	t := &timer{
		deadline: c.now.Add(d),
		ch:       make(chan time.Time, 1),
	}

	c.timers = append(c.timers, t)
	return t.ch
}

// Sleep simply waits on After(d)
func (c *FakeClock) Sleep(d time.Duration) {
	<-c.After(d)
}

// Add advances fake time and fires timers whose deadlines have passed
func (c *FakeClock) Add(d time.Duration) {
	c.mu.Lock()
	c.now = c.now.Add(d)

	now := c.now

	var remaining []*timer

	for _, t := range c.timers {
		if !t.deadline.After(now) {
			// timer has expired → fire it
			t.ch <- now
		} else {
			// still pending
			remaining = append(remaining, t)
		}
	}

	c.timers = remaining
	c.mu.Unlock()
}

// Tick automatically advances the fake clock every interval by calling Add(d).
func (c *FakeClock) Tick(d time.Duration) func() {
	stop := make(chan struct{})

	go func() {
		ticker := time.NewTicker(d)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				// Advance fake time by d
				c.Add(d)
			case <-stop:
				return
			}
		}
	}()

	// return a function to stop ticking
	return func() { close(stop) }
}
