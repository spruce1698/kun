/**
 * @Author: spruce
 * @Date: 2024-03-28 21:06
 * @Desc: 元转分,分转元
 */

package utils

import (
	"math"

	"github.com/shopspring/decimal"
)

// 元(字符串)转分,乘100把元转为分，转换后整数部分即为分，小数部分为分之后的单位可忽略
func YuanStr2Cent(yuan string) (cent int64, err error) {
	d, err := decimal.NewFromString(yuan)
	if err != nil {
		return
	}
	return d.Mul(decimal.NewFromInt(100)).IntPart(), nil
}

// 元转分,乘100把元转为分，转换后整数部分即为分，小数部分为分之后的单位可忽略
func Yuan2Cent(yuan float64) (cent int64) {
	// NaN/Inf 会让 decimal.NewFromFloat panic,这里直接返回 0 兜底
	if math.IsNaN(yuan) || math.IsInf(yuan, 0) {
		return 0
	}
	cent = decimal.NewFromFloat(yuan).Mul(decimal.NewFromInt(100)).IntPart()
	return
}

// 分转元,除100 把分转成元，转换后的结果包含整数和小数部分
func Cent2Yuan(cent int64) (yuan float64) {
	yuan, _ = decimal.NewFromInt(cent).DivRound(decimal.NewFromInt(100), 2).Float64()
	return
}
