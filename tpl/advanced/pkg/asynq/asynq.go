/**
* @Author: spruce
 * @Date: 2024-03-28 9:14
 * @Desc: asynq封装
*/

package asynq

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"time"

	"advanced/pkg/xconfig"

	"github.com/hibiken/asynq"
	goRedis "github.com/redis/go-redis/v9"
)

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
		Redis asynq.RedisClientOpt
	}
	Task = asynq.Task
)

func New(conf *xconfig.Conf) *Asynq {
	if len(conf.Redis.Source) == 0 {
		panic("redis 配置错误")
	}
	return &Asynq{
		Redis: asynq.RedisClientOpt{Addr: conf.Redis.Source[0], Password: conf.Redis.Password},
	}
}

// 异步消息推送
func (a *Asynq) SyncPub(queue, taskName, payload string) (entryID string, err error) {
	if queue == "" {
		queue = DefaultQueue
	}
	// 用局部变量持有 client,避免并发调用时共享字段 a.Client 互相覆盖/误关闭
	client := asynq.NewClient(a.Redis)
	defer func() {
		_ = client.Close()
	}()

	task := asynq.NewTask(taskName, []byte(payload))
	info, err := client.Enqueue(task, asynq.MaxRetry(RetryTime), asynq.Timeout(TimeOut), asynq.Queue(queue))
	if err != nil {
		return "", err
	}
	return info.ID, nil
}

// 延时消息推送
func (a *Asynq) DelayPub(queue, taskName, payload string, after time.Duration) (entryID string, err error) {
	if queue == "" {
		queue = DefaultQueue
	}
	// 用局部变量持有 client,避免并发调用时共享字段 a.Client 互相覆盖/误关闭
	client := asynq.NewClient(a.Redis)
	defer func() {
		_ = client.Close()
	}()

	task := asynq.NewTask(taskName, []byte(payload))
	info, err := client.Enqueue(task, asynq.MaxRetry(RetryTime), asynq.Timeout(TimeOut), asynq.Queue(queue), asynq.ProcessIn(after))
	if err != nil {
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
		return entryID, fmt.Errorf("load location Asia/Shanghai: %w", err)
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

// 动态设置 周期性任务 推送
// payload 为空时按删除语义处理(HDel);非空时写入(HSet 覆盖)。
func (a *Asynq) SetCronPub(taskName, payload, cronSpec string) error {
	ctx := context.Background()
	redisClient := a.Redis.MakeRedisClient().(goRedis.UniversalClient)
	defer func() { _ = redisClient.Close() }()

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
	redisClient := a.Redis.MakeRedisClient().(goRedis.UniversalClient)
	defer func() { _ = redisClient.Close() }()

	_, err := redisClient.HDel(ctx, RedisKey, taskName).Result()
	return err
}

// 动态定时任务配置源
type cfgProvider struct {
	redis asynq.RedisClientOpt
}

// 解析 redis set的配置并返回 PeriodicTaskConfig 列表
func (cp *cfgProvider) GetConfigs() ([]*asynq.PeriodicTaskConfig, error) {
	ctx := context.Background()

	redisClient := cp.redis.MakeRedisClient().(goRedis.UniversalClient)
	defer func() { _ = redisClient.Close() }()
	configs := redisClient.HGetAll(ctx, RedisKey).Val()

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
		return fmt.Errorf("load location Asia/Shanghai: %w", err)
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
		return fmt.Errorf("new periodic task manager: %w", err)
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
		return nil, fmt.Errorf("load location Asia/Shanghai: %w", err)
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
		return nil, fmt.Errorf("new periodic task manager: %w", err)
	}
	return manager, nil
}

func NewServeMux() *asynq.ServeMux {
	return asynq.NewServeMux()
}
