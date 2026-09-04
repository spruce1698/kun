package utils

import "github.com/shopspring/decimal"

const (
	base62 = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"
	base58 = "123456789abcdefghijkmnopqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ"
)

// intToBase 通用的进制转换，将整数转换为指定进制的字符串
func intToBase(n int64, alphabet string, base int) string {
	if n == 0 {
		return string(alphabet[0])
	}
	neg := false
	// 转 uint64 取绝对值。注意 MinInt64 时 -n 仍为 MinInt64(int64 溢出),
	// 必须在无符号域做取反(^u+1 == 0-u),才能得到正确的绝对值。
	u := uint64(n)
	if n < 0 {
		neg = true
		u = ^u + 1
	}

	// 预分配:64bit 整数在最小进制(2)下最多 64 位
	result := make([]byte, 0, 64)
	for u > 0 {
		result = append(result, alphabet[u%uint64(base)])
		u /= uint64(base)
	}

	// 反转字符串
	for i, j := 0, len(result)-1; i < j; i, j = i+1, j-1 {
		result[i], result[j] = result[j], result[i]
	}

	if neg {
		return "-" + string(result)
	}
	return string(result)
}

func IntToBase62(n int64) string {
	return intToBase(n, base62, 62)
}

// IntToBase58 方便阅读的字符串,排除数字0,消息字母l,大小字母O,I
func IntToBase58(n int64) string {
	return intToBase(n, base58, 58)
}

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
	cent = decimal.NewFromFloat(yuan).Mul(decimal.NewFromInt(100)).IntPart()
	return
}

// 分转元,除100 把分转成元，转换后的结果包含整数和小数部分
func Cent2Yuan(cent int64) (yuan float64) {
	yuan, _ = decimal.NewFromInt(cent).DivRound(decimal.NewFromInt(100), 2).Float64()
	return
}
