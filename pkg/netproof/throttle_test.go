package netproof

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestThrottle_LimitTotalRps_Throttle_BlocksRequestsPerSecond(t *testing.T) {
	t.Parallel()
	withContext(func(ctx context.Context) {
		timer := newFakeTimer()
		now := timer.Now()
		rps := 10
		throttle := newThrottle(ctx, rps, timer)
		assert.NotNil(t, throttle)

		wg := &sync.WaitGroup{}
		ch := make(chan struct{}, rps)
		for i := 0; i < rps; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				throttle.Throttle(ctx, func() error { return nil })
				ch <- struct{}{}
			}()
		}
		for len(ch) < rps {
			time.Sleep(time.Millisecond)
			timer.Advance(time.Millisecond)
		}
		wg.Wait()
		actual := timer.Since(now)
		assert.True(t, actual >= time.Second)
		assert.InDelta(t, 1, actual.Seconds(), 0.2)
	})
}

func TestThrottle_CanceledContext_Throttle_ReturnsCanceled(t *testing.T) {
	t.Parallel()
	withContext(func(ctx context.Context) {
		ctx, cancel := context.WithCancel(ctx)
		cancel()
		throttle := NewThrottle(ctx, 1)
		assert.NotNil(t, throttle)
		err := throttle.Throttle(ctx, func() error { return nil })
		assert.ErrorIs(t, err, context.Canceled)
	})
}
