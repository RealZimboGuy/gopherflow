package models

import "time"

// RetryConfig controls how often a failing state is retried and how long to wait
// between attempts. The delay slides from RetryIntervalMin towards
// RetryIntervalMax as the retry count grows.
type RetryConfig struct {
	MaxRetryCount    int
	RetryIntervalMin time.Duration
	RetryIntervalMax time.Duration
}

// SlidingInterval returns a retry interval between min and max based on the current retry attempt.
func (rc *RetryConfig) SlidingInterval(retryNum int) time.Duration {
	if retryNum <= 0 {
		return rc.RetryIntervalMin
	}
	if retryNum >= rc.MaxRetryCount {
		return rc.RetryIntervalMax
	}
	scale := float64(retryNum) / float64(rc.MaxRetryCount)
	return rc.RetryIntervalMin + time.Duration(scale*float64(rc.RetryIntervalMax-rc.RetryIntervalMin))
}
