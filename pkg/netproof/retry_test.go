package netproof

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestRetry_LimitRetries_Retry_RetriesMaxCount(t *testing.T) {
	t.Parallel()
	withContext(func(ctx context.Context) {
		cnt := 2
		timer := newFakeTimer()
		retry := newRetry(cnt-1, time.Minute, timer)
		assert.NotNil(t, retry)
		err := retry.Retry(func() error {
			cnt--
			return nil
		})
		assert.NoError(t, err)
		assert.Equal(t, 1, cnt)
	})
}

func TestRetry_LimitRetriesWithError_Retry_RetriesMaxCountWithError(t *testing.T) {
	t.Parallel()
	withContext(func(ctx context.Context) {
		cnt := 2
		timer := newFakeTimer()
		retry := newRetry(cnt-1, time.Minute, timer)
		assert.NotNil(t, retry)
		expected := assert.AnError
		err := retry.Retry(func() error {
			cnt--
			return expected
		})
		assert.Equal(t, expected, err)
		assert.Zero(t, cnt)
	})
}

func TestRetry_LimitRetriesWithStop_Retry_StopImmediately(t *testing.T) {
	t.Parallel()
	withContext(func(ctx context.Context) {
		cnt := 2
		timer := newFakeTimer()
		retry := newRetry(cnt-1, time.Minute, timer)
		assert.NotNil(t, retry)
		expected := assert.AnError
		err := retry.Retry(func() error {
			cnt--
			return errors.Join(expected, ErrStopRetry)
		})
		assert.ErrorIs(t, err, ErrStopRetry)
		assert.ErrorIs(t, err, expected)
		assert.Equal(t, 1, cnt)
	})
}
