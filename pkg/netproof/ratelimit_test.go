package netproof

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRateLimit_LimitHalfRps_RateLimit_ReturnsTrueForLastHalf(t *testing.T) {
	t.Parallel()
	withContext(func(ctx context.Context) {
		newError := assert.AnError
		wg := sync.WaitGroup{}
		rps := 4000
		timer := newFakeTimer()
		ratelimit := newRateLimit(ctx, rps/2, timer)
		assert.NotNil(t, ratelimit)

		ch := make(chan struct{}, rps)
		now := timer.Now()
		for i := 0; i < rps; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				err := ratelimit.RateLimit(ctx, func() error {
					return newError
				})
				assert.ErrorIs(t, err, newError)
				ch <- struct{}{}
			}()
		}
		for len(ch) < rps {
			time.Sleep(time.Millisecond)
			timer.Advance(time.Millisecond)
		}
		wg.Wait()
		assert.True(t, timer.Since(now) >= time.Second)
	})
}

func TestRateLimit_CanceledContext_RateLimit_ReturnsCanceled(t *testing.T) {
	t.Parallel()
	withContext(func(ctx context.Context) {
		ctx, cancel := context.WithCancel(ctx)
		cancel()
		ratelimit := NewRateLimit(ctx, 1)
		assert.NotNil(t, ratelimit)
		err := ratelimit.RateLimit(ctx, func() error { return nil })
		assert.ErrorIs(t, err, context.Canceled)
	})
}
