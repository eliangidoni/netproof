package netproof

import (
	"context"
	"sort"
	"sync"
	"testing"
	"time"

	"fmt"
	"runtime"

	"github.com/stretchr/testify/assert"
)

func TestHedge_MultipleErrors_Hedge_SendsRequestInOrder(t *testing.T) {
	t.Parallel()
	withContext(func(ctx context.Context) {
		timer := newFakeTimer()
		attempts := 3
		delay := time.Millisecond * 200
		hedged := newHedge(ctx, attempts, func() time.Duration { return delay }, timer)
		assert.NotNil(t, hedged)

		ch := make(chan time.Time, attempts)
		wg := &sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			for len(ch) < attempts {
				time.Sleep(time.Millisecond)
				timer.Advance(time.Millisecond)
			}
		}()
		newError := assert.AnError
		err := hedged.Hedge(func() error {
			timer.Sleep(time.Second)
			ch <- timer.Now()
			return newError
		})
		wg.Wait()
		assert.Equal(t, newError, err)
		var times []int64
		for i := 0; i < attempts; i++ {
			times = append(times, (<-ch).Unix())
		}
		assert.Zero(t, len(ch))
		timesCopy := make([]int64, len(times))
		copy(timesCopy, times)
		sort.Slice(timesCopy, func(i, j int) bool {
			return timesCopy[i] < timesCopy[j]
		})
		assert.Equal(t, timesCopy, times)
	})
}

func TestHedge_StopError_Hedge_ReturnsStopError(t *testing.T) {
	t.Parallel()
	withContext(func(ctx context.Context) {
		timer := newFakeTimer()
		attempts := 3
		delay := time.Millisecond * 200
		hedged := newHedge(ctx, attempts, func() time.Duration { return delay }, timer)
		assert.NotNil(t, hedged)

		chResult := make(chan error, attempts)
		wg := &sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			for len(chResult) < attempts {
				time.Sleep(time.Millisecond)
				timer.Advance(time.Millisecond)
			}
		}()
		chErr := make(chan error, attempts)
		newError := assert.AnError
		expected := []error{newError, ErrStopHedging, newError}
		for _, e := range expected {
			chErr <- e
		}
		err := hedged.Hedge(func() error {
			timer.Sleep(time.Second)
			e := <-chErr
			chResult <- e
			return e
		})
		wg.Wait()
		assert.ErrorIs(t, err, ErrStopHedging)
		var actual []error
		for i := 0; i < attempts; i++ {
			actual = append(actual, (<-chResult))
		}
		assert.Zero(t, len(chResult))
		assert.Equal(t, expected, actual)
	})
}

func TestHedge_Success_Hedge_ReturnsNil(t *testing.T) {
	t.Parallel()
	withContext(func(ctx context.Context) {
		timer := newFakeTimer()
		attempts := 3
		delay := time.Millisecond * 200
		hedged := newHedge(ctx, attempts, func() time.Duration { return delay }, timer)
		assert.NotNil(t, hedged)

		chResult := make(chan error, attempts)
		wg := &sync.WaitGroup{}
		wg.Add(1)
		go func() {
			defer wg.Done()
			for len(chResult) < attempts {
				time.Sleep(time.Millisecond)
				timer.Advance(time.Millisecond)
			}
		}()
		chErr := make(chan error, attempts)
		newError := assert.AnError
		expected := []error{newError, nil, newError}
		for _, e := range expected {
			chErr <- e
		}
		err := hedged.Hedge(func() error {
			timer.Sleep(time.Second)
			e := <-chErr
			chResult <- e
			return e
		})
		wg.Wait()
		assert.NoError(t, err)
		var actual []error
		for i := 0; i < attempts; i++ {
			actual = append(actual, (<-chResult))
		}
		assert.Zero(t, len(chResult))
		assert.Equal(t, expected, actual)
	})
}

func TestHedge_ContextCanceled_Hedge_ReturnsCanceled(t *testing.T) {
	t.Parallel()
	withContext(func(ctx context.Context) {
		ctx, cancel := context.WithCancel(ctx)
		attempts := 3
		delay := time.Millisecond * 200
		hedged := NewHedgeFixed(ctx, attempts, delay)
		assert.NotNil(t, hedged)
		assert.Equal(t, hedged.WaitingRequests(), 0)
		cancel()
		newError := assert.AnError
		err := hedged.Hedge(func() error {
			time.Sleep(time.Minute)
			return newError
		})
		assert.ErrorIs(t, err, context.Canceled)
	})
}

func Benchmark_Hedge(b *testing.B) {
	var goprocs = runtime.GOMAXPROCS(0)
	from := 1000
	to := 6000
	step := 1000
	attempts := 3
	hedged := NewHedgeFixed(context.Background(), attempts, time.Millisecond*500)
	for i := from; i < to; i += step {
		b.Run(fmt.Sprintf("goroutines-%d", i*goprocs), func(b *testing.B) {
			b.SetParallelism(i)
			b.RunParallel(func(pb *testing.PB) {
				for pb.Next() {
					hedged.Hedge(func() error {
						time.Sleep(time.Second)
						return assert.AnError
					})
				}
			})
		})
	}
}
