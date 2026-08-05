/**
* @Author: spruce
 * @Date: 2024-03-28 11:19
 * @Desc: 常用函数
*/

package utils

import (
	"math/rand/v2"
	"regexp"
	"sort"
	"strings"
	"unicode"
	"unicode/utf8"
)

var reChinesePunct = regexp.MustCompile("[\u3002\uff1b\uff0c\uff1a\u201c\u201d\uff08\uff09\u3001\uff1f\u300a\u300b]")

// 判断是否有中文
func HasChinese(str string) bool {
	for _, r := range str {
		if unicode.Is(unicode.Han, r) || reChinesePunct.MatchString(string(r)) {
			return true
		}
	}
	return false
}

// 计算中文及字母字符串长度，例如："你好hello" = 7
func TextLen(str string) int {
	return utf8.RuneCountInString(str)
}

// 截取含中文的字符串,start 0开始,截取 length 个字符(rune)。
// start/length 越界时做边界保护,不 panic。
func TextIntercept(str string, start, length int64) string {
	runes := []rune(str)
	strLen := int64(len(runes))
	if start < 0 {
		start = 0
	}
	if start >= strLen {
		return ""
	}
	end := start + length
	if end < start {
		end = start
	}
	if end > strLen {
		end = strLen
	}
	return string(runes[start:end])
}

// 按字典顺序排序(不修改原切片)
func DictSort(res []string) string {
	sorted := make([]string, len(res))
	copy(sorted, res)
	sort.Strings(sorted)
	var builder strings.Builder
	for _, v := range sorted {
		builder.WriteString(v)
	}
	return builder.String()
}

// 随机顺序排序
func ShuffleSort(res []string) []string {
	rand.Shuffle(len(res), func(i, j int) {
		res[i], res[j] = res[j], res[i]
	})
	return res
}

// 身份证号星号掩码
func IdCodeMask(idCode string) string {
	idCodeLen := len(idCode)
	if idCodeLen > 2 {
		return idCode[:1] + strings.Repeat("*", idCodeLen-2) + idCode[idCodeLen-1:]
	}
	return ""
}

// 手机号中间4位替换为*号(适用11位手机号:前3 + 4个* + 后4)
func MobileMask(mobile string) string {
	mobileLen := len(mobile)
	if mobileLen <= 7 {
		// 过短的手机号无法按 前3后4 掩码,整体掩码中间
		if mobileLen <= 2 {
			return strings.Repeat("*", mobileLen)
		}
		return mobile[:1] + strings.Repeat("*", mobileLen-2) + mobile[mobileLen-1:]
	}
	return mobile[:3] + strings.Repeat("*", 4) + mobile[mobileLen-4:]
}

// 实名最后一位为证实,前面为*
func RealNameMask(realName string) string {
	newRealName := []rune(realName)
	realNameLen := len(newRealName)
	if realNameLen >= 2 {
		return strings.Repeat("*", realNameLen-1) + string(newRealName[realNameLen-1:])
	}
	return ""
}
