package netproof

import (
	"context"
	"sort"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestBulkhead_ExhaustedResources_Bulkhead_ReturnsExpectedErrors(t *testing.T) {
	t.Parallel()
	withContext(func(ctx context.Context) {
		timer := newFakeTimer()
		resources := []any{1, 2, 3}
		bhead := NewBulkhead(ctx, resources)
		assert.NotNil(t, bhead)
		assert.Equal(t, 0, bhead.TotalUsed())

		newError := assert.AnError
		expected := []error{ErrExhausted, newError, newError, newError}
		chErr := make(chan error, len(expected))
		chResources := make(chan any, len(expected))
		for i := 0; i < len(expected); i++ {
			go func() {
				err := bhead.Bulkhead(ctx, func(r any) error {
					timer.Sleep(time.Second)
					chResources <- r
					return newError
				})
				chErr <- err
			}()
		}
		time.Sleep(time.Second)
		timer.Advance(time.Second)
		actualErr := []error{}
		for i := 0; i < len(expected); i++ {
			actualErr = append(actualErr, <-chErr)
		}
		assert.Zero(t, len(chErr))
		actualResources := []any{}
		for i := 0; i < len(resources); i++ {
			actualResources = append(actualResources, <-chResources)
		}
		sort.Slice(actualResources, func(i, j int) bool {
			return actualResources[i].(int) < actualResources[j].(int)
		})
		assert.Equal(t, resources, actualResources)
		assert.Equal(t, expected, actualErr)
	})
}

func TestBulkhead_CanceledContext_Bulkhead_ReturnsCanceled(t *testing.T) {
	t.Parallel()
	withContext(func(ctx context.Context) {
		ctx, cancel := context.WithCancel(ctx)
		cancel()
		bhead := NewBulkhead(ctx, []any{})
		assert.NotNil(t, bhead)
		err := bhead.Bulkhead(ctx, func(_ any) error { return nil })
		assert.ErrorIs(t, err, context.Canceled)
	})
}
