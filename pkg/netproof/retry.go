package netproof

import (
	"errors"
	"math"
	"math/rand"
	"sync"
	"time"
)

// ErrStopRetry is a sentinel error to stop retrying.
var ErrStopRetry = errors.New("stop retrying")

type Retry struct {
	maxCount    int
	maxWait     time.Duration
	scaleFactor time.Duration
	t           timer
	maxTokens   float64
	tokens      float64
	refillRate  float64
	lock        sync.Mutex
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

// WithBucket returns a new Retry with token bucket + exponential backoff + jitter.
// maxWait can be zero (disabled).
func WithBucket(maxTokens float64, refillRate float64, maxCount int, maxWait time.Duration) *Retry {
	return &Retry{maxCount: maxCount, maxWait: maxWait, scaleFactor: time.Second, t: realTimer{},
		maxTokens: maxTokens, refillRate: refillRate, tokens: maxTokens,
	}
}

func (r *Retry) consumeToken() bool {
	if r.maxTokens == 0 {
		return true
	}

	r.lock.Lock()
	defer r.lock.Unlock()
	if r.tokens >= 1 {
		r.tokens--
		return true
	}
	return false
}

func (r *Retry) produceToken() {
	if r.maxTokens == 0 {
		return
	}

	r.lock.Lock()
	defer r.lock.Unlock()
	r.tokens = math.Max(r.maxTokens, r.tokens+r.refillRate)
}

// Retry retries the given function f() up to maxCount times.
// It stops after maxWait or when f() returns ErrStop.
func (r *Retry) Retry(f func() error) error {
	var err error
	startTime := r.t.Now()
	for i := 0; i < r.maxCount; i++ {
		if err = f(); err == nil {
			r.produceToken()
			return nil
		}
		if !r.consumeToken() {
			return err
		}
		if errors.Is(err, ErrStopRetry) {
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
