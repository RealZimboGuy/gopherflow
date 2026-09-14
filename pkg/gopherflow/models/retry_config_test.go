package models

import (
	"testing"
	"time"
)

func TestSlidingIntervalGrowsWithRetries(t *testing.T) {
	rc := RetryConfig{
		MaxRetryCount:    10,
		RetryIntervalMin: time.Second * 10,
		RetryIntervalMax: time.Minute * 60,
	}

	first := rc.SlidingInterval(1)
	second := rc.SlidingInterval(2)

	if first < rc.RetryIntervalMin {
		t.Errorf("first interval %v is below the configured minimum %v", first, rc.RetryIntervalMin)
	}
	if second < first {
		t.Errorf("interval shrank between retries: retry 2 = %v, retry 1 = %v", second, first)
	}
}

func TestSlidingIntervalIsCappedAtMax(t *testing.T) {
	rc := RetryConfig{
		MaxRetryCount:    100,
		RetryIntervalMin: time.Second * 10,
		RetryIntervalMax: time.Minute * 5,
	}

	for _, retry := range []int{0, 1, 5, 20, 100, 1000} {
		if got := rc.SlidingInterval(retry); got > rc.RetryIntervalMax {
			t.Errorf("SlidingInterval(%d) = %v, exceeds max %v", retry, got, rc.RetryIntervalMax)
		}
	}
}

func TestSlidingIntervalIsNeverNegative(t *testing.T) {
	rc := RetryConfig{
		MaxRetryCount:    5,
		RetryIntervalMin: time.Second,
		RetryIntervalMax: time.Minute,
	}

	for _, retry := range []int{-5, -1, 0} {
		if got := rc.SlidingInterval(retry); got < 0 {
			t.Errorf("SlidingInterval(%d) = %v, want a non-negative duration", retry, got)
		}
	}
}
