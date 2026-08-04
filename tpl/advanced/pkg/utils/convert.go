package utils

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
	if n < 0 {
		neg = true
		// 安全取绝对值，避免 math.MinInt64 溢出
		n = -(n + 1)
		n = -n
	}

	var result []byte
	for n > 0 {
		result = append(result, alphabet[n%int64(base)])
		n /= int64(base)
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
