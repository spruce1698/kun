/**
* @Author: spruce
 * @Date: 2024-07-03 17:00
 * @Desc: 支付
*/

package global

// Demo交易状态
const (
	PayStatusIng     = 1 // 进行中
	PayStatusSuccess = 2 // 成功
	PayStatusFail    = 3 // 失败
)

// Demo支付方式
const (
	PayTypeAlipay = 1 // 支付宝
	PayTypeWxpay  = 2 // 微信
)
