/**
* @Author: spruce
 * @Date: 2024-03-28 11:08
 * @Desc: 消息订阅器
*/

package event

import (
	"context"
	"fmt"
	"sync"

	"advanced/internal/global"
	"advanced/pkg/asynq"
	"advanced/pkg/kafka"
	"advanced/pkg/xconfig"
	"advanced/pkg/xlog"

	githubAsynq "github.com/hibiken/asynq"
)

type (
	Sub struct {
		wg     *sync.WaitGroup
		conf   *xconfig.Conf
		ctx    context.Context
		log    *xlog.Logger
		aQueue *asynq.Asynq
		task   *Task
		// asynq server / cron manager,关闭时需要 Shutdown
		aSrv *githubAsynq.Server
		aMgr *githubAsynq.PeriodicTaskManager
		kSub *kafka.Subscriber
	}
	Task struct {
		KafkaSet []*Kafka
		AsynqSet []*Asynq
	}
	Kafka struct {
		Topic   global.EventTopic
		Group   global.EventGroup
		Handler func(key, value string) error
	}
	Asynq struct {
		Topic   global.EventTopic
		Handler func(context.Context, *asynq.Task) error
	}
)

func NewSub(wg *sync.WaitGroup, conf *xconfig.Conf, log *xlog.Logger, task *Task, ctx context.Context) *Sub {
	return &Sub{
		wg:     wg,
		conf:   conf,
		ctx:    ctx,
		log:    log,
		aQueue: asynq.New(conf),
		task:   task,
	}
}

// Kafka 消费者
func (s *Sub) Kafka() {

	if s.task.KafkaSet == nil || len(s.task.KafkaSet) == 0 {
		return
	}

	s.kSub = kafka.NewSubscriber(s.conf, s.log)

	s.wg.Add(len(s.task.KafkaSet))
	for _, kq := range s.task.KafkaSet {
		go func(v *Kafka) {
			defer s.wg.Done()
			s.kSub.SubFetch(s.ctx, string(v.Topic), string(v.Group), v.Handler)
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
		mux.HandleFunc(string(aq.Topic), aq.Handler)
	}

	// 用 Start(非阻塞) + Shutdown 主动控制生命周期,
	// 避免 srv.Run 自带的信号处理与 broker 信号处理冲突。
	if err := srv.Start(mux); err != nil {
		s.log.Error(s.ctx, "asynq server start err", err)
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
		s.log.Error(s.ctx, "asynq cron new manager err", err)
		return fmt.Errorf("asynq cron new manager err: %w", err)
	}
	s.aMgr = mgr
	if err := mgr.Start(); err != nil {
		s.log.Error(s.ctx, "asynq cron start err", err)
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

// ==============github.com/robfig/cron 周期性任务=====================================

// "github.com/robfig/cron/v3"
//  CronFun func(corner *cron.Cron)

// // 周期性任务服务精确到秒
// func (s *Sub) Cron() {
// 	if s.task.CronFunSet == nil || len(s.task.CronFunSet) <= 0 {
// 		return
// 	}
//
// 	loc, _ := time.LoadLocation("Asia/Shanghai")
// 	cs := cron.New(cron.WithSeconds(), cron.WithLocation(loc))
// 	s.wg.Add(1)
// 	go func() {
// 		// 添加任务
// 		for _, fun := range s.task.CronFunSet {
// 			fun(cs)
// 		}
// 		cs.Start()
//
// 		defer s.wg.Done()
// 	}()
// 	defer cs.Stop() // 需要在协程结束时关闭
// }

// // 支付事件消费
// func (s *Service) PayDemo(key, value string) error {
// 	switch enum.MsgType(key) {
// 	case enum.MsgTypePayRecharge: // 充值
//
// 		msg := &producer.PayRecharge{}
// 		_ = json.Unmarshal([]byte(value), msg)
//
// 		fmt.Printf("KafkaSubscriber-Fetch:消费时间:%s:%s \n", utils.GetTimeStr(time.Now()), value)
// 		// 逻辑处理start...
// 		// TODO: do something
// 		// err := c.DemoCache.SetDemo(context.Background(), 111111, &cache.Demo{Id: 111111, Name: "asdfsadfasf"}, 0)
// 		// fmt.Println(err)
//
// 		return nil
// 	default:
// 		log.Fatalln("未知的支付事件类型", key)
// 	}
// 	return nil
// }
//
// // 支付事件消费
// func (s *Service) KQPayDemo(key, value string) error {
// 	msg := &producer.PayRecharge{}
// 	_ = json.Unmarshal([]byte(value), msg)
//
// 	fmt.Printf("KafkaSubscriber:key 消费时间:%s:%s \n", utils.GetTimeStr(time.Now()), value)
// 	// 逻辑处理start...
// 	// TODO: do something
//
// 	// err := c.DemoCache.SetDemo(context.Background(), 111111, &cache.Demo{Id: 111111, Name: "asdfsadfasf"}, 0)
// 	// fmt.Println(err)
//
// 	return nil
// }
//
// // aq消费者
// func (s *Service) AqSubscriberDemo(ctx context.Context, task *asynq.Task) error {
// 	fmt.Printf("AqSubscriber:消费时间:%s: %s %s  \n", utils.GetTimeStr(time.Now()), task.Type(), task.Payload())
// 	// 逻辑处理start...
// 	// TODO: do something
// 	return nil
// }
//
// // Cron 消费者
// func (s *Service) CronSubscriberDemo(corner *cron.Cron) {
// 	// 每个偶数秒执行
// 	spec := "*/2 * * * * *"
// 	var demoLock sync.Mutex
// 	entryID, err := corner.AddFunc(spec, func() {
// 		demoLock.Lock()
// 		defer demoLock.Unlock()
//
// 		fmt.Printf("CronDemo:消费时间:%s \n", utils.GetTimeStr(time.Now()))
//
// 		// 逻辑处理start...
// 		// TODO: do something
// 	})
// 	if err != nil {
// 		fmt.Printf("启动[demo]定时任务失败:%v %v \n", entryID, err)
// 	}
// }
