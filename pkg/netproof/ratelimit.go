package netproof

import (
	"context"
	"time"
)

type RateLimit struct {
	ch  <-chan struct{}
	ctx context.Context
}

// NewRateLimit creates a new rate limit with the given maxRps.
// When ctx is canceled, it stops any ongoing requests.
func NewRateLimit(ctx context.Context, maxRps int) *RateLimit {
	return newRateLimit(ctx, maxRps, realTimer{})
}

func newRateLimit(ctx context.Context, maxRps int, t timer) *RateLimit {
	ch := make(chan struct{}, maxRps)
	go func() {
		nextTick := t.Now().Add(time.Second)
		for {
			if ctx.Err() != nil {
				return
			}
			t.Sleep(t.Until(nextTick))
			nextTick = t.Now().Add(time.Second)
			for i := 0; i < maxRps; i++ {
				select {
				case ch <- struct{}{}:
				default:
				}
			}
		}
	}()
	return &RateLimit{ch: ch, ctx: ctx}
}

// RateLimit limits the rate of the given function (blocks).
// Returns f() error or either ctx error (if any).
func (r *RateLimit) RateLimit(ctx context.Context, f func() error) error {
	select {
	case <-r.ctx.Done():
		return r.ctx.Err()
	case <-ctx.Done():
		return ctx.Err()
	case <-r.ch:
		return f()
	}
}
