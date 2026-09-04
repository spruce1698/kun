/**
* @Author: spruce
 * @Date: 2024-03-28 9:14
 * @Desc: asynq封装
*/

package asynq

import (
	"context"
	"encoding/json"
	"log"
	"time"

	"advanced/pkg/xconfig"

	"github.com/hibiken/asynq"
	goRedis "github.com/redis/go-redis/v9"
)

// tracerName asynq 包 span 使用的 tracer 名称。
const tracerName = "advanced/pkg/asynq"

const (
	DefaultQueue  = "default"
	CriticalQueue = "critical"
	LowQueue      = "low"
	RetryTime     = 10
	TimeOut       = 12 * time.Hour
	Concurrency   = 10                                 // 最大并发进程作业任务数
	RedisKey      = "running:pubsub:asynq:cronDynamic" // 动态周期任务 redis key
)

type (
	Asynq struct {
		Redis  asynq.RedisConnOpt
		client *asynq.Client
	}
	Task = asynq.Task
)

func New(conf *xconfig.Conf) *Asynq {
	if len(conf.Redis.Source) == 0 {
		return &Asynq{}
	}
	var opt asynq.RedisConnOpt
	if conf.Redis.Cluster {
		opt = asynq.RedisClusterClientOpt{
			Addrs:    conf.Redis.Source,
			Password: conf.Redis.Password,
		}
	} else {
		opt = asynq.RedisClientOpt{
			Addr:     conf.Redis.Source[0],
			Password: conf.Redis.Password,
			DB:       conf.Redis.DB,
		}
	}
	return &Asynq{
		Redis:  opt,
		client: asynq.NewClient(opt),
	}
}

// Close 关闭底层 client 连接池
func (a *Asynq) Close() error {
	if a.client != nil {
		return a.client.Close()
	}
	return nil
}

// 异步消息推送
func (a *Asynq) SyncPub(queue, taskName, payload string) (entryID string, err error) {
	return a.SyncPubCtx(context.Background(), queue, taskName, payload)
}

// SyncPubCtx 发布异步任务,并在 ctx 中记录 producer span(便于在 trace 中观察投递动作)。
func (a *Asynq) SyncPubCtx(ctx context.Context, queue, taskName, payload string) (entryID string, err error) {
	if queue == "" {
		queue = DefaultQueue
	}

	ctx, span := startEnqueueSpan(ctx, taskName)
	defer span.End()

	task := asynq.NewTask(taskName, []byte(payload))
	info, err := a.client.EnqueueContext(ctx, task, asynq.MaxRetry(RetryTime), asynq.Timeout(TimeOut), asynq.Queue(queue))
	if err != nil {
		span.RecordError(err)
		return "", err
	}
	return info.ID, nil
}

// 延时消息推送
func (a *Asynq) DelayPub(queue, taskName, payload string, after time.Duration) (entryID string, err error) {
	return a.DelayPubCtx(context.Background(), queue, taskName, payload, after)
}

// DelayPubCtx 同 DelayPub,带 ctx 以记录 producer span。
func (a *Asynq) DelayPubCtx(ctx context.Context, queue, taskName, payload string, after time.Duration) (entryID string, err error) {
	if queue == "" {
		queue = DefaultQueue
	}

	ctx, span := startEnqueueSpan(ctx, taskName)
	defer span.End()

	task := asynq.NewTask(taskName, []byte(payload))
	info, err := a.client.EnqueueContext(ctx, task, asynq.MaxRetry(RetryTime), asynq.Timeout(TimeOut), asynq.Queue(queue), asynq.ProcessIn(after))
	if err != nil {
		span.RecordError(err)
		return "", err
	}
	return info.ID, nil
}

// 周期性任务推送 （静态） 阻塞
/*
crontab格式 如下所示：
  ┌────────────── 分钟 (0 - 59)
  │ ┌────────────── 小时 (0 - 23)
  │ │ ┌────────────── 一个月中的第几天 (1 - 31)
  │ │ │ ┌────────────── 月（1 - 12）
  │ │ │ │ ┌────────────── 星期几（0 - 6）（周日至周六；7 在某些系统上也是星期日）
  │ │ │ │ │
  │ │ │ │ │
  * * * * * <要执行的命令>
(5) @daily            每天一次
(6) @midnight         同上
(7) @every 1m30s      定时1分30秒执行
*/
func (a *Asynq) CronStaticPub(queue, taskName, payload, cronSpec string) (entryID string, err error) {
	if queue == "" {
		queue = DefaultQueue
	}

	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.Local
	}
	// 创建调度器
	scheduler := asynq.NewScheduler(
		a.Redis,
		&asynq.SchedulerOpts{
			Location: loc,
		},
	)
	task := asynq.NewTask(taskName, []byte(payload))

	entryID, err = scheduler.Register(cronSpec, task, asynq.Queue(queue))
	if err != nil {
		return entryID, err
	}
	// 运行调度器 并阻塞
	err = scheduler.Run()

	return entryID, err
}

// 动态定时任务内容
type Provider struct {
	CronSpec string `json:"cronSpec"`
	Payload  string `json:"payload"`
}

// newRedisClient 创建一个底层 go-redis 客户端用于直接操作 Hash,调用方必须 Close。
// asynq.RedisConnOpt.MakeRedisClient() 每次都 NewClient 一个全新连接,
// 不 Close 会泄漏 fd/协程。原实现三处调用(SetCronPub/DelCronPub/GetConfigs)均未关闭,
// 而 GetConfigs 被周期任务管理器按 SyncInterval(10s) 反复调用 -> 持续泄漏。
func newRedisClient(opt asynq.RedisConnOpt) (goRedis.UniversalClient, func()) {
	c := opt.MakeRedisClient().(goRedis.UniversalClient)
	return c, func() { _ = c.Close() }
}

// 动态设置 周期性任务 推送
// payload 为空时按删除语义处理(HDel);非空时写入(HSet 覆盖)。
func (a *Asynq) SetCronPub(taskName, payload, cronSpec string) error {
	ctx := context.Background()
	redisClient, closeFn := newRedisClient(a.Redis)
	defer closeFn()

	// payload 为空 => 删除该周期任务
	if payload == "" {
		if _, err := redisClient.HDel(ctx, RedisKey, taskName).Result(); err != nil {
			return err
		}
		return nil
	}

	p, err := json.Marshal(Provider{
		CronSpec: cronSpec,
		Payload:  payload,
	})
	if err != nil {
		return err
	}
	// HSet 覆盖式写入,等价于先 HDel 再 HSetNX,但只一次往返
	if _, err := redisClient.HSet(ctx, RedisKey, taskName, string(p)).Result(); err != nil {
		return err
	}
	return nil
}

// 动态设置 周期性任务 删除
func (a *Asynq) DelCronPub(taskName string) error {
	ctx := context.Background()
	redisClient, closeFn := newRedisClient(a.Redis)
	defer closeFn()

	_, err := redisClient.HDel(ctx, RedisKey, taskName).Result()
	return err
}

// 动态定时任务配置源
type cfgProvider struct {
	redis asynq.RedisConnOpt
}

// 解析 redis set的配置并返回 PeriodicTaskConfig 列表
func (cp *cfgProvider) GetConfigs() ([]*asynq.PeriodicTaskConfig, error) {
	ctx := context.Background()

	redisClient, closeFn := newRedisClient(cp.redis)
	defer closeFn()
	configs, err := redisClient.HGetAll(ctx, RedisKey).Result()
	if err != nil {
		return nil, err
	}

	var tasks []*asynq.PeriodicTaskConfig
	for k, v := range configs {
		var c Provider
		if err := json.Unmarshal([]byte(v), &c); err != nil {
			log.Println(k, "Payload 不是json字符串")
			return nil, err
		}
		tasks = append(tasks, &asynq.PeriodicTaskConfig{Cronspec: c.CronSpec, Task: asynq.NewTask(k, []byte(c.Payload))})
	}
	return tasks, nil
}

// 周期性任务推送 （动态）
/*
crontab格式 如下所示：
  ┌────────────── 分钟 (0 - 59)
  │ ┌────────────── 小时 (0 - 23)
  │ │ ┌────────────── 一个月中的第几天 (1 - 31)
  │ │ │ ┌────────────── 月（1 - 12）
  │ │ │ │ ┌────────────── 星期几（0 - 6）（周日至周六；7 在某些系统上也是星期日）
  │ │ │ │ │
  │ │ │ │ │
  * * * * * <要执行的命令>
(5) @daily            每天一次
(6) @midnight         同上
(7) @every 1m30s      定时1分30秒执行
*/
func (a *Asynq) CronPub() error {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.Local
	}
	manager, err := asynq.NewPeriodicTaskManager(
		asynq.PeriodicTaskManagerOpts{
			RedisConnOpt:               a.Redis,
			PeriodicTaskConfigProvider: &cfgProvider{redis: a.Redis},
			SyncInterval:               10 * time.Second, // 和配置文件同步的时间，配置文件如果修改，可以及时运用到程序
			SchedulerOpts:              &asynq.SchedulerOpts{Location: loc},
		},
	)
	if err != nil {
		return err
	}
	return manager.Run()
}

// 消费者
func (a *Asynq) Server() *asynq.Server {
	return asynq.NewServer(
		a.Redis,
		asynq.Config{
			// IsFailure 判断 handler 返回的 error 是否算失败(决定是否重试)。
			// 所有非 nil error 都视为失败 -> 触发重试(受 MaxRetry 限制)。
			// asynq 自身已有失败任务的日志记录,这里无需额外打印。
			IsFailure: func(err error) bool {
				return true
			},
			Concurrency: Concurrency, // 最大并发进程作业任务数
			Queues: map[string]int{
				CriticalQueue: 5, // 关键队列中的任务将有50%的时间得到处理
				DefaultQueue:  4, // 默认队列中的任务将有40%的时间被处理
				LowQueue:      1, // 低队列中的任务将有10%的时间被处理
			},
		},
	)
}

// PeriodicTaskManager 周期性任务管理器(动态),非阻塞启动,需调用方持有并 Shutdown。
func (a *Asynq) PeriodicTaskManager() (*asynq.PeriodicTaskManager, error) {
	loc, err := time.LoadLocation("Asia/Shanghai")
	if err != nil {
		loc = time.Local
	}
	manager, err := asynq.NewPeriodicTaskManager(
		asynq.PeriodicTaskManagerOpts{
			RedisConnOpt:               a.Redis,
			PeriodicTaskConfigProvider: &cfgProvider{redis: a.Redis},
			SyncInterval:               10 * time.Second, // 和配置文件同步的时间，配置文件如果修改，可以及时运用到程序
			SchedulerOpts:              &asynq.SchedulerOpts{Location: loc},
		},
	)
	if err != nil {
		return nil, err
	}
	return manager, nil
}

func NewServeMux() *asynq.ServeMux {
	return asynq.NewServeMux()
}
