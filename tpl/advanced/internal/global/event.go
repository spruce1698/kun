/**
 * @Author: spruce
 * @Date: 2022/4/2 15:19
 * @Desc: topic的枚举
 */

package global

type (
	EventType  string
	EventTopic string
	EventGroup string
)

// 消息类型
const (
	// 支付充值
	EventTypePayRecharge EventType = "pay_recharge"
)

// 消息Topic
const (
	// 支付消息topic
	EventTopicPay  EventTopic = "topic_pay"
	EventTopicPay1 EventTopic = "topic_pay11"
)

// 消息group
const (
	// 支付消息topic
	EventGroupPay  EventGroup = "group_pay"
	EventGroupPay1 EventGroup = "group_pay1"
)

// 支付充值消息
type PayRecharge struct {
	PayType    int64  `json:"pay_type"`    // 支付类型 参考 enum.PayTypeAlipay
	Status     int64  `json:"status"`      // 状态 1:发起中 2:成功 3:失败
	UserId     int64  `json:"user_id"`     // 充值用户id
	TradeNo    string `json:"trans_no"`    // 内部交易单号
	Cash       int64  `json:"cash"`        // 交易金额(单位:分)
	CreateTime string `json:"create_time"` // 发生时间
	Remark     string `json:"remark"`      // 备注信息
}
