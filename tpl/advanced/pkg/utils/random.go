package utils

import (
	"math/rand/v2"
	"strings"
)

const (
	AlphaNoZeroDigital      = iota + 1 // 字母和无零数字
	AlphaNoLowerZeroDigital            // 小写字母和无零数字,适合做验证码和短信
	Alpha                              // 大小写字母
	AlphaLower                         // 小写字母
	AlphaUpper                         // 大写字母
	Numeric                            // 数字
	NoZeroDigital                      // 无0数字
)

// 生成指定长度的字符串
func GenStr(mode, length int) string {
	var (
		pos     int
		seedStr string
	)
	switch mode {
	case AlphaNoZeroDigital:
		seedStr = "123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	case AlphaNoLowerZeroDigital:
		seedStr = "123456789abcdefghijklmnpqrstuvwxyz"
	case Alpha:
		seedStr = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	case AlphaLower:
		seedStr = "abcdefghijklmnopqrstuvwxyz"
	case AlphaUpper:
		seedStr = "ABCDEFGHIJKLMNOPQRSTUVWXYZ"
	case Numeric:
		seedStr = "0123456789"
	case NoZeroDigital:
		seedStr = "123456789"
	default:
		seedStr = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	}

	seedLen := len(seedStr)
	var b strings.Builder
	b.Grow(length)
	for i := 0; i < length; i++ {
		pos = rand.IntN(seedLen)
		b.WriteString(seedStr[pos : pos+1])
	}
	return b.String()
}

// 生成指定范围的数字
func GenNumeric(min int, max int) int {
	if min < max {
		return rand.IntN(max-min) + min
	} else if min > max {
		return rand.IntN(min-max) + max
	} else {
		return max
	}
}
