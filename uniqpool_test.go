package uniqpool

import (
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

// TestUniq checks that tasks with the same identifier are executed only once.
func TestUniq(t *testing.T) {
	// Create a new UniqPool instance
	pool := New[string](10, 2, 10, time.Millisecond*100)

	var processed int32

	// Add tasks to the pool
	pool.Submit("task1", func() {
		atomic.AddInt32(&processed, 1)
	})

	require.True(t, pool.TrySubmit("task2", func() {
		atomic.AddInt32(&processed, 1)
	}))

	// Add the same task again (should be ignored)
	require.True(t, pool.TrySubmit("task2", func() {
		atomic.AddInt32(&processed, 1)
	}))

	// Add the same task again (should be ignored)
	pool.Submit("task1", func() {
		atomic.AddInt32(&processed, 1)
	})

	// Stop the pool
	pool.StopAndWait()

	// Assert that only two tasks were processed
	require.Equal(t, int32(2), processed)
	require.Empty(t, pool.uniqMap)

	// panic because pool is stopped
	require.Panics(t, func() { pool.Submit("task1", func() {}) })
	require.Panics(t, func() { pool.TrySubmit("task1", func() {}) })
}

// TestInboundQueueOverflow checks that the inbound queue overflow and waiting for available space works correctly.
func TestInboundQueueOverflow(t *testing.T) {
	// Create a new UniqPool instance
	pool := New[string](2, 2, 10, time.Millisecond*100)

	var (
		processed int32
		now       = time.Now()
	)

	// Add tasks to the pool
	pool.Submit("task1", func() {
		atomic.AddInt32(&processed, 1)
	})

	pool.Submit("task2", func() {
		atomic.AddInt32(&processed, 1)
	})

	// check that the tasks are processed immediately
	require.Less(t, time.Since(now).Milliseconds(), int64(10))

	// Shold be rejected because the inbound queue is full
	require.False(t, pool.TrySubmit("task3", func() {
		atomic.AddInt32(&processed, 1)
	}))

	pool.Submit("task4", func() {
		atomic.AddInt32(&processed, 1)
	})

	// check that the task is not processed immediately because the inbound queue is full
	require.Greater(t, time.Since(now).Milliseconds(), int64(50))

	// Stop the pool
	pool.StopAndWait()

	// Assert that only two tasks were processed
	require.Equal(t, int32(3), processed)
	require.Empty(t, pool.uniqMap)
}
