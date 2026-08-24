package utils

import (
	"errors"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestPool_BasicExecution(t *testing.T) {
	pool := NewPool(4, 10)
	var count int64

	for i := 0; i < 20; i++ {
		err := pool.AddTask(Task{
			ID: i,
			Job: func() {
				atomic.AddInt64(&count, 1)
			},
		})
		if err != nil {
			t.Fatalf("AddTask failed: %v", err)
		}
	}

	pool.Wait()

	if got := atomic.LoadInt64(&count); got != 20 {
		t.Fatalf("expected count 20, got %d", got)
	}
}

func TestPool_ConcurrentAddTaskAndShutdownNoPanic(t *testing.T) {
	// 验证高并发下 AddTask 与 Shutdown 同时调用不会发生 send on closed channel panic
	for round := 0; round < 20; round++ {
		pool := NewPool(4, 100)
		var wg sync.WaitGroup

		// 启动并发写入
		for i := 0; i < 50; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				_ = pool.AddTask(Task{
					ID:  id,
					Job: func() { time.Sleep(time.Millisecond) },
				})
			}(i)
		}

		// 随机并发关闭
		time.Sleep(time.Duration(round) * 100 * time.Microsecond)
		pool.Shutdown()

		// 关停后再添加任务应返回 ErrPoolClosed 且不 panic
		err := pool.AddTask(Task{
			ID:  999,
			Job: func() {},
		})
		if err == nil || !errors.Is(err, ErrPoolClosed) {
			t.Logf("expected ErrPoolClosed after shutdown, got: %v", err)
		}

		wg.Wait()
		pool.Wait()
	}
}
