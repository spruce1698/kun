/**
* @Author: spruce
 * @Date: 2024-03-28 13:40
 * @Desc: 常用验证函数
*/

package utils

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
	"unicode"
)

// 预编译正则表达式，避免每次调用重复编译
var (
	rePassword     = regexp.MustCompile(`[a-zA-Z]`)
	rePasswordNum  = regexp.MustCompile(`\d`)
	rePasswordSym  = regexp.MustCompile(`[^a-zA-Z\d]`)
	reEmail        = regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	reQQ           = regexp.MustCompile(`^[1-9][0-9]{4,12}$`)
	reWeChat       = regexp.MustCompile(`^[a-zA-Z][-_a-zA-Z0-9]{6,20}$`)
	reWeibo        = regexp.MustCompile(`^[a-zA-Z][\w-]*$`)
	rePostalCode   = regexp.MustCompile(`^[1-9]\d{5}$`)
	reTime         = regexp.MustCompile(`^(?:[01]\d|2[0-3]):[0-5]\d:[0-5]\d$`)
	reDate         = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)
	reDateTime     = regexp.MustCompile(`^\d{4}-\d{2}-\d{2} \d{2}:\d{2}:\d{2}$`)
	reNumber       = regexp.MustCompile(`^[0-9]+$`)
	reDecimal      = regexp.MustCompile(`^[0-9]+(\.[0-9]{1,2})?$`)
	reChineseName  = regexp.MustCompile("^[\u4E00-\u9FA5]{2,6}$")
	reEnglishName  = regexp.MustCompile(`^([a-zA-Z]+\s)*[a-zA-Z]+$`)
	reMobile       = regexp.MustCompile(`^1[3456789]\d{9}$`)
	reTelephone    = regexp.MustCompile(`^0\d{2,3}-?\d{7,8}$`)
	reURL          = regexp.MustCompile(`^(http|https)://[a-zA-Z0-9_-]+(\.[a-zA-Z0-9_-]+)+([\w.,@?^=%&:/~+#-]*[\w@?^=%&/~+#-])?$`)
	reIDCard18     = regexp.MustCompile(`^[1-9]\d{5}(19|20)\d{2}((0[1-9])|(1[0-2]))(([0-2][1-9])|10|20|30|31)\d{3}[0-9Xx]$`)
	reIDCard15     = regexp.MustCompile(`^\d{15}$`)
	reDateBirthday = regexp.MustCompile(`^(19|20)\d{2}(0[1-9]|1[0-2])(0[1-9]|[12]\d|3[01])$`)
)

// 验证密码是否合法,密码长度在6-20个字符之间，必须包含数字、字母和特殊符号
func IsPassword(password string) bool {
	if len(password) < 6 || len(password) > 20 {
		return false
	}

	if !rePassword.MatchString(password) {
		return false
	}

	if !rePasswordNum.MatchString(password) {
		return false
	}

	if !rePasswordSym.MatchString(password) {
		return false
	}

	return true
}

// 密码强度等级，D为最低
const (
	levelD = iota
	LevelC
	LevelB
	LevelA
	LevelS
)

// 预编译密码强度校验正则，避免循环中重复编译
var (
	reDigit  = regexp.MustCompile(`[0-9]+`)
	reLower  = regexp.MustCompile(`[a-z]+`)
	reUpper  = regexp.MustCompile(`[A-Z]+`)
	reSymbol = regexp.MustCompile(`[~!@#$%^&*?_-]+`)
)

func VerifyPasswordLever(minLength, maxLength, minLevel int, pwd string) error {
	// 首先校验密码长度是否在范围内
	if len(pwd) < minLength {
		return fmt.Errorf("BAD PASSWORD: The password is shorter than %d characters", minLength)
	}
	if len(pwd) > maxLength {
		return fmt.Errorf("BAD PASSWORD: The password is logner than %d characters", maxLength)
	}

	// 初始化密码强度等级为D，利用正则校验密码强度，若匹配成功则强度自增1
	var level = levelD
	for _, re := range []*regexp.Regexp{reDigit, reLower, reUpper, reSymbol} {
		if re.MatchString(pwd) {
			level++
		}
	}

	// 如果最终密码强度低于要求的最低强度，返回并报错
	if level < minLevel {
		return fmt.Errorf("The password does not satisfy the current policy requirements. ")
	}
	return nil

	// // 判断是否重复数字,连续数字
	// oldLen := len(pwd)
	// newSice := make(map[uint8]struct{}, oldLen)
	// for k := range pwd {
	//     newSice[pwd[k]] = struct{}{}
	//     // ascii 编码 (47 /)(48 0)(49 1)...(55 7)(56 8)(57 9)(58 :)
	//     if k+2 < oldLen && pwd[k+1] >= 49 && pwd[k+1] <= 56 {
	//         if (pwd[k] == pwd[k+1]+1 && pwd[k+1] == pwd[k+2]+1) || (pwd[k]+1 == pwd[k+1] && pwd[k+1]+1 == pwd[k+2]) {
	//             fmt.Printf("连续数字")
	//         }
	//     }
	// }
	// if len(newSice) == 1 {
	//     fmt.Printf("字符串重复")
	// }
}

func PasswordComplexityVerify(s string) bool {
	var (
		hasMinLen  = false
		hasUpper   = false
		hasLower   = false
		hasNumber  = false
		hasSpecial = false
	)
	if len(s) >= 8 {
		hasMinLen = true
	}
	for _, char := range s {
		switch {
		case unicode.IsUpper(char):
			hasUpper = true
		case unicode.IsLower(char):
			hasLower = true
		case unicode.IsNumber(char):
			hasNumber = true
		case unicode.IsPunct(char) || unicode.IsSymbol(char):
			hasSpecial = true
		}
	}
	return hasMinLen && hasUpper && hasLower && hasNumber && hasSpecial
}

// 是否为email
func IsEmail(input string) bool {
	return reEmail.MatchString(input)
}

// 是否为Json
func IsJSON(input string) bool {
	var js json.RawMessage
	return json.Unmarshal([]byte(input), &js) == nil
}

// 验证是否为QQ号
func IsQQ(qq string) bool {
	return reQQ.MatchString(qq)
}

// 验证是否为微信号
func IsWeChat(wechat string) bool {
	return reWeChat.MatchString(wechat)
}

// 验证是否为微博ID
func IsWeibo(weibo string) bool {
	if len(weibo) < 6 || len(weibo) > 20 {
		return false
	}
	return reWeibo.MatchString(weibo)
}

// 验证是否为邮编号码
func IsPostalCode(str string) bool {
	return rePostalCode.MatchString(str)
}

// 验证是否为大陆银行卡号
func IsBankCardNo(cardNumber string) bool {
	if len(cardNumber) != 16 && len(cardNumber) != 19 {
		return false
	}
	var cardArr []int
	for _, c := range cardNumber {
		if c < '0' || c > '9' {
			return false
		}
		cardArr = append(cardArr, int(c-'0'))
	}
	if len(cardArr) == 16 {
		sum := 0
		for i := len(cardArr) - 1; i >= 0; i-- {
			if i%2 == 0 {
				cardArr[i] *= 2
				if cardArr[i] > 9 {
					cardArr[i] -= 9
				}
			}
			sum += cardArr[i]
		}
		return sum%10 == 0
	} else {
		sum := 0
		for i := len(cardArr) - 1; i >= 0; i-- {
			if (len(cardArr)-i)%2 == 0 {
				cardArr[i] *= 2
				if cardArr[i] > 9 {
					cardArr[i] -= 9
				}
			}
			sum += cardArr[i]
		}
		return sum%10 == 0
	}
}

// 验证身份证号(18或15位)
func IsIDCard(str string) bool {
	if len(str) != 15 && len(str) != 18 {
		return false
	}
	if len(str) == 18 {
		return IsIDCard18(str)
	} else {
		return IsIDCard15(str)
	}
}

// 验证18位身份证号
func IsIDCard18(id string) bool {
	// 利用正则表达式匹配身份证号码
	if !reIDCard18.MatchString(id) {
		return false
	}
	// 解析身份证号码中的年、月、日
	year, _ := strconv.Atoi(id[6:10])
	month, _ := strconv.Atoi(id[10:12])
	day, _ := strconv.Atoi(id[12:14])
	// 判断年份是否合法
	if year < 1900 || year > time.Now().Year() {
		return false
	}
	// 判断月份是否合法
	if month < 1 || month > 12 {
		return false
	}
	// 判断日期是否合法
	if day < 1 || day > 31 {
		return false
	}
	// 对身份证号码的前17位进行加权和校验
	// 加权系数，根据规则固定
	weights := []int{7, 9, 10, 5, 8, 4, 2, 1, 6, 3, 7, 9, 10, 5, 8, 4, 2}
	// 计算加权和
	sum := 0
	for i := 0; i < len(weights); i++ {
		num, _ := strconv.Atoi(string(id[i]))
		sum += num * weights[i]
	}

	// 计算校验码
	checkCode := sum % 11
	// 校验码对照表:余数 0-10 对应字符 '1','0','X','9','8','7','6','5','4','3','2'
	// (其中 X 表示 10),用字符串切片替代 map,直接字符比较,更清晰。
	checkCodes := []byte{'1', '0', 'X', '9', '8', '7', '6', '5', '4', '3', '2'}
	expected := checkCodes[checkCode]
	// 实际最后一位:数字 0-9 取字符,X/x 统一当 X 比对
	lastChar := id[len(id)-1]
	if lastChar == 'x' {
		lastChar = 'X'
	}
	return lastChar == expected
}

// 验证15位身份证号
func IsIDCard15(idCard string) bool {
	// 验证是否为15位数字
	if !reIDCard15.MatchString(idCard) {
		return false
	}

	// 将身份证号前两位转换成省份代码
	provinceCode, err := strconv.Atoi(idCard[:2])
	if err != nil || provinceCode < 11 || provinceCode > 91 {
		return false
	}

	// 验证生日是否正确
	year := strconv.Itoa(1900 + int(idCard[6]-'0')*10 + int(idCard[7]-'0'))
	month := string(idCard[8:10])
	day := string(idCard[10:12])
	if !reDateBirthday.MatchString(year + month + day) {
		return false
	}

	return true
}

// 验证是否为时间格式（HH:mm:ss）
func IsTime(str string) bool {
	return reTime.MatchString(str)
}

// 验证是否为日期格式（yyyy-MM-dd）
func IsDate(str string) bool {
	if !reDate.MatchString(str) {
		return false
	}
	parts := strings.Split(str, "-")
	year, _ := strconv.Atoi(parts[0])
	month, _ := strconv.Atoi(parts[1])
	day, _ := strconv.Atoi(parts[2])
	return isValidMonth(month, year) && isValidDay(day, month, year)
}

func isValidMonth(month, year int) bool {
	switch month {
	case 1, 3, 5, 7, 8, 10, 12:
		return true
	case 4, 6, 9, 11:
		return true
	case 2:
		return true
	}
	return false
}

func isValidDay(day, month, year int) bool {
	switch month {
	case 1, 3, 5, 7, 8, 10, 12:
		if day >= 1 && day <= 31 {
			return true
		}
	case 4, 6, 9, 11:
		if day >= 1 && day <= 30 {
			return true
		}
	case 2:
		if year%4 == 0 && year%100 != 0 || year%400 == 0 {
			if day >= 1 && day <= 29 {
				return true
			}
		} else {
			if day >= 1 && day <= 28 {
				return true
			}
		}
	}
	return false
}

// 验证是否为日期时间格式（yyyy-MM-dd HH:mm:ss）
func IsDateTime(str string) bool {
	if !reDateTime.MatchString(str) {
		return false
	}
	if !IsDate(str[0:10]) || !IsTime(str[11:]) {
		return false
	}
	return true
}

// 验证是否全部为数字
func IsNumber(input string) bool {
	return reNumber.MatchString(input)
}

// 验证给定的字符串小数点后是否最多两位
func IsDecimal(input string) bool {
	return reDecimal.MatchString(input)
}

// 验证给定的字符串全部为中文
func IsAllChinese(input string) bool {
	for _, r := range input {
		if !unicode.Is(unicode.Scripts["Han"], r) {
			return false
		}
	}
	return true
}

// 验证给定的字符串包含中文
func IsContainChinese(input string) bool {
	for _, r := range input {
		if unicode.Is(unicode.Scripts["Han"], r) {
			return true
		}
	}
	return false
}

// 验证是否为中文名
func IsChineseName(name string) bool {
	return reChineseName.MatchString(name)
}

// 验证是否为英文名
func IsEnglishName(name string) bool {
	return reEnglishName.MatchString(name)
}

// 验证是否为手机号码
func IsMobile(phone string) bool {
	return reMobile.MatchString(phone)
}

// 验证是否为座机号码
func IsTelephone(telephone string) bool {
	return reTelephone.MatchString(telephone)
}

// 是否为URL地址
func IsURL(url string) bool {
	return reURL.MatchString(url)
}
