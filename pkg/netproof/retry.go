package netproof

import (
	"errors"
	"math"
	"math/rand"
	"time"
)

// ErrStopRetry is a sentinel error to stop retrying.
var ErrStopRetry = errors.New("stop retrying")

type Retry struct {
	maxCount    int
	maxWait     time.Duration
	scaleFactor time.Duration
	t           timer
}

// NewRetry returns a new Retry, maxWait can be zero (disabled).
func NewRetry(maxCount int, maxWait time.Duration) *Retry {
	return newRetry(maxCount, maxWait, realTimer{})
}

func newRetry(maxCount int, maxWait time.Duration, t timer) *Retry {
	return &Retry{maxCount: maxCount + 1, maxWait: maxWait, t: t}
}

// WithBackoff returns a new Retry with exponential backoff + jitter.
// maxWait can be zero (disabled).
func WithBackoff(maxCount int, maxWait time.Duration) *Retry {
	return &Retry{maxCount: maxCount, maxWait: maxWait, scaleFactor: time.Second, t: realTimer{}}
}

// Retry retries the given function f() up to maxCount times.
// It stops after maxWait or when f() returns ErrStop.
func (r *Retry) Retry(f func() error) error {
	var err error
	startTime := r.t.Now()
	for i := 0; i < r.maxCount; i++ {
		if err = f(); errors.Is(err, ErrStopRetry) || err == nil {
			return err
		}
		if r.maxWait > 0 && r.t.Since(startTime) >= r.maxWait {
			return err
		}
		if r.scaleFactor > 0 {
			wait := r.scaleFactor * time.Duration(math.Pow(2, float64(i)))
			jitter := time.Duration(rand.Intn(1000)) * time.Millisecond
			r.t.Sleep(wait + jitter)
		}
	}
	return err
}
