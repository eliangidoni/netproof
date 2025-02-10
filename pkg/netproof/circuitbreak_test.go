package netproof

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestCircuitBreak_HalfOpen_CircuitBreak_ReturnsError(t *testing.T) {
	t.Parallel()
	withContext(func(ctx context.Context) {
		timer := newFakeTimer()
		resetTimeout := time.Millisecond * 100
		circuitbreak := newCircuitBreak(2, resetTimeout, timer)
		assert.NotNil(t, circuitbreak)

		newError := assert.AnError
		expected := []error{
			newError,
			newError, // trigger circuit break
			ErrCircuitBreak,
			newError, // trigger half open state
			ErrCircuitBreak,
			nil, // reset circuit break
			nil,
		}
		actual := []error{}
		for i := 0; i < 3; i++ {
			err := circuitbreak.CircuitBreak(func() error {
				return newError
			})
			actual = append(actual, err)
		}
		timer.Advance(resetTimeout)
		for i := 0; i < 2; i++ {
			err := circuitbreak.CircuitBreak(func() error {
				return newError
			})
			actual = append(actual, err)
		}
		timer.Advance(resetTimeout)
		for i := 0; i < 2; i++ {
			err := circuitbreak.CircuitBreak(func() error {
				return nil
			})
			actual = append(actual, err)
		}
		assert.Equal(t, expected, actual)
	})
}
