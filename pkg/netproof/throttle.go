package netproof

import (
	"context"
	"time"
)

type Throttle struct {
	ch  <-chan struct{}
	ctx context.Context
}

// NewThrottle creates a new Throttle that allows rps requests
// per second. When ctx is canceled, it stops any ongoing requests.
func NewThrottle(ctx context.Context, rps int) *Throttle {
	return newThrottle(ctx, rps, realTimer{})
}

func newThrottle(ctx context.Context, rps int, t timer) *Throttle {
	ch := make(chan struct{})
	tickerResolution := time.Second / time.Duration(rps)
	go func() {
		nextTick := t.Now().Add(tickerResolution)
		for {
			if ctx.Err() != nil {
				return
			}
			t.Sleep(t.Until(nextTick))
			nextTick = t.Now().Add(tickerResolution)
			select {
			case ch <- struct{}{}:
			default:
			}
		}
	}()
	return &Throttle{ch: ch, ctx: ctx}
}

// Throttle blocks until the next request is allowed.
// Returns f() error or either ctx error (if any).
func (t *Throttle) Throttle(ctx context.Context, f func() error) error {
	select {
	case <-t.ctx.Done():
		return t.ctx.Err()
	case <-ctx.Done():
		return ctx.Err()
	case <-t.ch:
		return f()
	}
}
