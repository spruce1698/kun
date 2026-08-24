/**
* @Author: spruce
 * @Date: 2024-03-28 15:33
 * @Desc: 协程池
*/

package utils

import (
	"errors"
	"fmt"
	"sync"
)

// ErrPoolClosed 协程池已关闭(Shutdown/Wait 后再 AddTask)。
var ErrPoolClosed = errors.New("pool is closed")

// 任务结构体
type Task struct {
	ID  int
	Job func()
}

// 协程池结构体
type Pool struct {
	taskQueue chan Task
	wg        sync.WaitGroup
	once      sync.Once
	closed    bool
	mu        sync.Mutex // 保护 closed
}

// 创建协程池
// queueSize 为任务队列缓冲大小,0 表示无缓冲(AddTask 会阻塞直到有空闲 worker);
// 建议设置一个合理的缓冲,避免调用方在无空闲 worker 时卡死。
func NewPool(numWorkers, queueSize int) *Pool {
	p := &Pool{
		taskQueue: make(chan Task, queueSize),
	}

	p.wg.Add(numWorkers)
	for i := 0; i < numWorkers; i++ {
		go p.worker()
	}

	return p
}

// AddTask 添加任务到协程池。
// 队列满时会阻塞,直到有 worker 取走任务。
// 若池已关闭(Shutdown/Wait 后),返回 ErrPoolClosed,不会 panic。
func (p *Pool) AddTask(task Task) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	if p.closed {
		return ErrPoolClosed
	}
	p.taskQueue <- task
	return nil
}

// 工作协程
func (p *Pool) worker() {
	defer p.wg.Done()
	for task := range p.taskQueue {
		func() {
			defer func() {
				if r := recover(); r != nil {
					fmt.Printf("Worker task %d panic: %v\n", task.ID, r)
				}
			}()
			task.Job()
		}()
	}
}

// Shutdown 关闭任务队列,通知所有 worker 不再接收新任务并处理完剩余任务后退出。
// 可安全多次调用(用 sync.Once 保证只关闭一次)。
func (p *Pool) Shutdown() {
	p.once.Do(func() {
		p.mu.Lock()
		p.closed = true
		close(p.taskQueue)
		p.mu.Unlock()
	})
}

// 等待所有任务完成(worker 退出)。
// 调用前应先 Shutdown 关闭队列,否则会一直阻塞等待 worker 退出。
func (p *Pool) Wait() {
	p.Shutdown()
	p.wg.Wait()
}
