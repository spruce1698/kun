/**
* @Author: spruce
 * @Date: 2024-03-28 11:08
 * @Desc: 消息订阅器
*/

package event

import (
	"context"
	"fmt"
	"os"
	"sync"

	"advanced/pkg/asynq"
	"advanced/pkg/kafka"
	"advanced/pkg/xbroker"
	"advanced/pkg/xconfig"

	githubAsynq "github.com/hibiken/asynq"
)

type (
	Sub struct {
		wg     *sync.WaitGroup
		conf   *xconfig.Conf
		ctx    context.Context
		aQueue *asynq.Asynq
		task   *xbroker.Task
		// asynq server / cron manager,关闭时需要 Shutdown
		aSrv *githubAsynq.Server
		aMgr *githubAsynq.PeriodicTaskManager
		kSub *kafka.Subscriber
	}
)

func NewSub(wg *sync.WaitGroup, conf *xconfig.Conf, task *xbroker.Task, ctx context.Context) *Sub {
	return &Sub{
		wg:     wg,
		conf:   conf,
		ctx:    ctx,
		aQueue: asynq.New(conf),
		task:   task,
	}
}

// Kafka 消费者
func (s *Sub) Kafka() {

	if s.task.KafkaSet == nil || len(s.task.KafkaSet) == 0 {
		return
	}

	s.kSub = kafka.NewSubscriber(s.conf)

	s.wg.Add(len(s.task.KafkaSet))
	for _, kq := range s.task.KafkaSet {
		go func(v *xbroker.Kafka) {
			defer s.wg.Done()
			s.kSub.SubFetch(s.ctx, v.Topic, v.Group, v.Handler)
		}(kq)
	}
}

// Asynq 消费者
// 启动失败时返回 error,由调用方决定如何退出(而非在此处 os.Exit,绕过资源回收)。
func (s *Sub) Asynq() error {
	if s.task.AsynqSet == nil || len(s.task.AsynqSet) <= 0 {
		return nil
	}

	srv := s.aQueue.Server()
	s.aSrv = srv
	mux := asynq.NewServeMux()
	// 添加任务
	for _, aq := range s.task.AsynqSet {
		mux.HandleFunc(aq.Topic, aq.Handler)
	}
	// 包裹 tracing 中间件:为每个任务创建 consumer span,记录耗时/错误。
	handler := asynq.TracingMiddleware(mux)

	// 用 Start(非阻塞) + Shutdown 主动控制生命周期,
	// 避免 srv.Run 自带的信号处理与 broker 信号处理冲突。
	if err := srv.Start(handler); err != nil {
		fmt.Fprintf(os.Stderr, "!!!CronJobErr!!! run err:%+v\n", err)
		srv.Shutdown()
		return fmt.Errorf("asynq server start err: %w", err)
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		// 等待关闭信号
		<-s.ctx.Done()
		srv.Shutdown()
	}()
	return nil
}

// AsynqCron 周期性任务服务
// 与 Asynq 一致:先同步 Start(失败则返回 error),成功后再起 goroutine 等待关闭信号。
func (s *Sub) AsynqCron() error {
	mgr, err := s.aQueue.PeriodicTaskManager()
	if err != nil {
		fmt.Fprintf(os.Stderr, "!!!AsynqCronErr!!! new manager err:%+v\n", err)
		return fmt.Errorf("asynq cron new manager err: %w", err)
	}
	s.aMgr = mgr
	if err := mgr.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "!!!AsynqCronErr!!! start err:%+v\n", err)
		return fmt.Errorf("asynq cron start err: %w", err)
	}

	s.wg.Add(1)
	go func() {
		defer s.wg.Done()
		// 等待关闭信号
		<-s.ctx.Done()
		mgr.Shutdown()
	}()
	return nil
}

// Close 关闭订阅器持有的资源(kafka reader 等),应在 wg.Wait() 之后调用。
func (s *Sub) Close() {
	if s.kSub != nil {
		s.kSub.Close()
	}
}

