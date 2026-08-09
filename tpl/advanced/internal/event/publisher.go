/**
* @Author: spruce
 * @Date: 2024-03-28 17:27
 * @Desc: 消息推送器
*/

package event

import (
	"fmt"
	"time"

	"advanced/internal/global"
	"advanced/pkg/asynq"
	"advanced/pkg/kafka"
	"advanced/pkg/xconfig"
	"advanced/pkg/xlog"
)

type (
	Pub struct {
		kQueue *kafka.Publisher
		aQueue *asynq.Asynq
		logger *xlog.Logger
	}
)

func NewPub(conf *xconfig.Conf, log *xlog.Logger) *Pub {
	return &Pub{
		kQueue: kafka.NewPublisher(conf),
		aQueue: asynq.New(conf),
		logger: log,
	}
}

// Close 关闭底层 kafka Writer 等资源,应在进程退出前调用。
func (p *Pub) Close() {
	if err := p.kQueue.Close(); err != nil {
		p.logger.Warn(fmt.Sprintf("kafka publisher close err: %v", err))
	}
}

// Kafka 发布kafka异步消息
func (p *Pub) Kafka(topic global.EventTopic, message []byte) error {
	topicStr := string(topic)
	err := p.kQueue.Pub(topicStr, message)
	if err != nil {
		return fmt.Errorf("kafka topic:%s 发布消息:%s 失败, error:%w", topicStr, message, err)
	}
	return nil
}

// KafkaWithType 发布kafka异步有type的消息
func (p *Pub) KafkaWithType(topic global.EventTopic, msgType global.EventType, message []byte) error {
	topicStr := string(topic)
	msgTypeStr := string(msgType)
	err := p.kQueue.PubWithKey(topicStr, msgTypeStr, message)
	if err != nil {
		return fmt.Errorf("kafka topic:%s 发布类型:%s 消息:%s 失败, error:%w", topicStr, msgTypeStr, message, err)
	}
	return nil
}

// Delay 发布延时消息,单位s
func (p *Pub) Delay(topic global.EventTopic, message string, delay int64) error {
	topicStr := string(topic)
	_, err := p.aQueue.DelayPub(asynq.CriticalQueue, topicStr, message, time.Duration(delay)*time.Second)
	if err != nil {
		return fmt.Errorf("topic:%s  发布延时消息:%s 失败, error:%w", topicStr, message, err)
	}
	return nil
}

// Sync 发布异步消息
func (p *Pub) Sync(topic global.EventTopic, message string) error {
	topicStr := string(topic)
	_, err := p.aQueue.SyncPub(asynq.DefaultQueue, topicStr, message)
	if err != nil {
		return fmt.Errorf("topic:%s  发布异步消息:%s 失败, error:%w", topicStr, message, err)
	}
	return nil
}

// Cron 发布定时消息
func (p *Pub) Cron(topic global.EventTopic, message, cronSpec string) error {
	topicStr := string(topic)
	err := p.aQueue.SetCronPub(topicStr, message, cronSpec)
	if err != nil {
		return fmt.Errorf("topic:%s  发布周期/定时消息:%s 失败, error:%w", topicStr, message, err)
	}
	return nil
}

// DelCron 发布取消消息
func (p *Pub) DelCron(topic global.EventTopic) error {
	topicStr := string(topic)
	err := p.aQueue.DelCronPub(topicStr)
	if err != nil {
		return fmt.Errorf("topic:%s  取消周期/定时消息 失败, error:%w", topicStr, err)
	}
	return nil
}

// // 推送充值事件
// //
// //	_ = s.pubsub.PayRechargeKafka(func(recharge *pubsub.PayRecharge) {
// //		recharge.TradeNo = "1111111"
// //		recharge.PayType = enum.PayTypeAlipay
// //		recharge.Status = enum.PayStatusComplete
// //		recharge.UserId = 123
// //		recharge.Cash = 1.20
// //		recharge.CreateTime = utils.GetTimeStr(time.Now())
// //		recharge.Remark = "KQ充值异步消息"
// //	})
// //
// // 推送支付充值消息
// func (p *pubsub.Pub) PayRechargeKafka(fill func(*PayRecharge)) error {
// 	entity := &PayRecharge{}
// 	fill(entity)
// 	message, err := json.Marshal(entity)
// 	if err != nil {
// 		return err
// 	}
// 	err = p.kQueue.PubWithKey(string(enum.MsgTopicPay), string(enum.MsgTypePayRecharge), message)
// 	if err != nil {
// 		log.Printf("支付事件[充值]推送失败.%s, error:%w \n", message, err)
// 		return fmt.Errorf("支付事件[充值]推送失败.%s, error:%w", message, err)
// 	}
//
// 	return nil
// }
//
// // 推送支付充值5s后发生消息
// func (p *pubsub.Pub) PayRechargeDelay(fill func(*PayRecharge)) error {
// 	entity := &PayRecharge{}
// 	fill(entity)
// 	message, err := json.Marshal(entity)
// 	if err != nil {
// 		return err
// 	}
// 	// 5s 后
// 	_, err = p.aQueue.DelayPub(asynq.DefaultQueue, string(enum.MsgTopicPay), string(message), 5*time.Second)
// 	if err != nil {
// 		log.Printf("支付事件[充值赠送]推送失败.%s, error:%w \n", message, err)
// 		return fmt.Errorf("支付事件[充值赠送]推送失败.%s, error:%w", message, err)
// 	}
// 	return nil
// }
//
// // 推送支付充值异步
// func (p *pubsub.Pub) PayRechargeSync(fill func(*PayRecharge)) error {
// 	entity := &PayRecharge{}
// 	fill(entity)
// 	message, err := json.Marshal(entity)
// 	if err != nil {
// 		return err
// 	}
// 	_, err = p.aQueue.SyncPub(asynq.CriticalQueue, string(enum.MsgTopicPay), string(message))
// 	if err != nil {
// 		log.Printf("支付事件[充值异步]推送失败.%s, error:%w \n", message, err)
// 		return fmt.Errorf("支付事件[充值异步]推送失败.%s, error:%w", message, err)
// 	}
// 	return nil
// }
//
// // 推送支付充值异步
// func (p *pubsub.Pub) PayRechargeCron(fill func(*PayRecharge)) error {
// 	entity := &PayRecharge{}
// 	fill(entity)
// 	message, err := json.Marshal(entity)
// 	if err != nil {
// 		return err
// 	}
// 	err = p.aQueue.SetCronPub(string(enum.MsgTopicPay), string(message), "@every 3s")
// 	if err != nil {
// 		log.Printf("支付事件[充值定时3S]推送失败.%s, error:%w \n", message, err)
// 		return fmt.Errorf("支付事件[充值定时3S]推送失败.%s, error:%w", message, err)
// 	}
// 	return nil
// }
