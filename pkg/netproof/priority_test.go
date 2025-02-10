package netproof

import (
	"context"
	"slices"
	"sort"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPriority_ConcurrentPop_Pop_ReturnsAllElements(t *testing.T) {
	t.Parallel()
	withContext(func(ctx context.Context) {
		prio := NewPriority(ctx)
		assert.NotNil(t, prio)

		total := 1000
		elems := make(map[int]bool)
		ch := make(chan int)
		wg := sync.WaitGroup{}
		for i := 0; i < total; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				elem, err := prio.Pop()
				assert.NoError(t, err)
				ch <- elem.(int)
			}()
		}

		wg.Add(1)
		go func() {
			defer wg.Done()
			for i := 0; i < total; i++ {
				prio.Push(0, i)
				elems[i] = true
			}
			for i := 0; i < total; i++ {
				elem := <-ch
				assert.True(t, elems[elem])
				delete(elems, elem)
			}
		}()
		wg.Wait()
	})
}

func TestPriority_ConcurrentPush_Pop_ReturnsValidPriority(t *testing.T) {
	t.Parallel()
	withContext(func(ctx context.Context) {
		prio := NewPriority(ctx)
		assert.NotNil(t, prio)

		total := 1000
		wg := sync.WaitGroup{}
		for i := 0; i < total; i++ {
			wg.Add(1)
			go func(i int) {
				defer wg.Done()
				prio.Push(i, i)
			}(i)
		}
		wg.Wait()
		assert.Equal(t, total, prio.Len())

		actual := []int{}
		expected := []int{}
		for i := 0; i < total; i++ {
			elem, err := prio.Pop()
			assert.NoError(t, err)
			actual = append(actual, elem.(int))
			expected = append(expected, i)
		}
		sort.Ints(expected)
		slices.Reverse(expected)
		assert.Equal(t, expected, actual)
	})
}

func TestPriority_CanceledContext_Pop_ReturnsCanceled(t *testing.T) {
	t.Parallel()
	withContext(func(ctx context.Context) {
		ctx, cancel := context.WithCancel(ctx)
		prio := NewPriority(ctx)
		assert.NotNil(t, prio)

		total := 1000
		wg := sync.WaitGroup{}
		for i := 0; i < total; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				_, err := prio.Pop()
				assert.ErrorIs(t, err, context.Canceled)
			}()
		}
		cancel()
		wg.Wait()
	})
}
