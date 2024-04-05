package uniqpool

import (
	"sync"
	"sync/atomic"
	"time"

	"github.com/alitto/pond"
)

type task[T comparable] struct {
	// The unique identifier of the task.
	id T
	// The function that will be executed by the task.
	fn func()
}

// UniqPool is a pool of tasks. Each task has a unique identifier. If several tasks with the same identifier
// get into the pool and have not yet been executed, only one of them will be executed.
// At the same time, if a task with such an identifier has already been executed, a new task will be executed.
// You can set an interval during which tasks will accumulate so as not to create many identical tasks.
type UniqPool[T comparable] struct {
	// The pool of workers that will execute the tasks.
	pool *pond.WorkerPool
	// The interval during which tasks will accumulate so as not to create many identical tasks.
	interval time.Duration

	// Channel for Submit.
	inboundChan chan task[T]
	// Map for checking the uniqueness of the task identifier. [key]->[position in inboundQueue]
	uniqMap map[T]struct{}
	// Mutex for working with the inbound queue.
	inboundMutex sync.Mutex

	// Wait group for waiting for all tasks to be executed before stopping the pool.
	stopWaitGroup sync.WaitGroup
	// Channel for stopping the pool.
	stopChan chan struct{}
	stopped  int32
}

// NewUniqPool creates a new UniqPool.
func NewUniqPool[T comparable](inboundQueueCapacity, poolWorkersCount, poolCapacity int, interval time.Duration) *UniqPool[T] {
	p := &UniqPool[T]{
		pool:        pond.New(poolWorkersCount, poolCapacity),
		interval:    interval,
		inboundChan: make(chan task[T], inboundQueueCapacity),
		uniqMap:     make(map[T]struct{}, inboundQueueCapacity),
		stopChan:    make(chan struct{}),
	}

	p.stopWaitGroup.Add(1)
	go p.processTasks()

	return p
}

// Try submit adds a task to the pool.
func (p *UniqPool[T]) TrySubmit(id T, fn func()) bool {
	if p.Stopped() {
		panic("pool is stopped")
	}

	p.inboundMutex.Lock()
	defer p.inboundMutex.Unlock()

	// check the uniqueness of the task identifier
	if _, ok := p.uniqMap[id]; ok {
		return true
	}

	select {
	case p.inboundChan <- task[T]{id: id, fn: fn}:
		p.uniqMap[id] = struct{}{}
		return true
	default:
		return false
	}
}

// Submit adds a task to the pool. Will block if the inbound queue is full.
func (p *UniqPool[T]) Submit(id T, fn func()) {
	if p.Stopped() {
		panic("pool is stopped")
	}

	p.inboundMutex.Lock()
	defer p.inboundMutex.Unlock()

	// check the uniqueness of the task identifier
	if _, ok := p.uniqMap[id]; ok {
		return
	}

	p.inboundChan <- task[T]{id: id, fn: fn}
	p.uniqMap[id] = struct{}{}
}

// StopAndWait stops the pool and waits for all tasks to be executed.
func (p *UniqPool[T]) StopAndWait() {
	// first stop the processTasks goroutine
	close(p.stopChan)
	p.stopWaitGroup.Wait()
	// then stop the pool
	p.pool.StopAndWait()
}

// processTasks processes the tasks from the inbound queue.
func (p *UniqPool[T]) processTasks() {
	defer p.stopWaitGroup.Done()

	ticker := time.NewTicker(p.interval)
	defer ticker.Stop()

	for {
		select {
		case <-p.stopChan:
			atomic.StoreInt32(&p.stopped, 1)
		case <-ticker.C:
		}

		drain := true
		for drain {
			select {
			case t := <-p.inboundChan:
				p.pool.Submit(t.fn)
				p.inboundMutex.Lock()
				delete(p.uniqMap, t.id)
				p.inboundMutex.Unlock()
			default:
				drain = false
			}
		}

		if p.Stopped() {
			return
		}
	}
}

// Stopped returns true if the pool is stopped.
func (p *UniqPool[T]) Stopped() bool {
	return atomic.LoadInt32(&p.stopped) == 1
}
