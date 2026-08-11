/**
 * @Author: spruce
 * @Date: 2024-03-28 21:15
 * @Desc: 系统输入参数校验
 */

package validator

import (
	"reflect"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin/binding"
	"github.com/go-playground/locales/en"
	"github.com/go-playground/locales/zh"
	ut "github.com/go-playground/universal-translator"
	vali10 "github.com/go-playground/validator/v10"
	enT "github.com/go-playground/validator/v10/translations/en"
	zhT "github.com/go-playground/validator/v10/translations/zh"
)

var (
	once       sync.Once
	Trans      ut.Translator
	reMobileCN = regexp.MustCompile(`^(13[0-9]|14[01456879]|15[0-35-9]|16[2567]|17[0-8]|18[0-9]|19[0-35-9])\d{8}$`)
)

type (
	Errors     = vali10.ValidationErrors
	customFunc struct {
		name string
		fun  func(fl vali10.FieldLevel) bool
		tips string
	}
)

// New 初始化校验器与翻译器。
// 注意:once.Do 保证只执行一次,locale 仅首次调用生效;需要切换 locale 请另加显式重注册接口。
func New(locale string) {
	once.Do(func() {
		// 修改gin框架中的Validator引擎属性，实现自定制
		if validate, ok := binding.Validator.Engine().(*vali10.Validate); ok {
			uni := ut.New(zh.New(), en.New(), zh.New())
			// 也可以使用 uni.FindTranslator(...) 传入多个locale进行查找
			Trans, _ = uni.GetTranslator(locale)

			// 注册一个获取json tag的自定义方法
			validate.RegisterTagNameFunc(func(fld reflect.StructField) string {
				json := strings.SplitN(fld.Tag.Get("json"), ",", 2)[0]
				if json == "-" {
					return ""
				} else {
					form := strings.SplitN(fld.Tag.Get("form"), ",", 2)[0]
					if form == "-" {
						return ""
					} else if form != "" {
						return form
					}
					return json
				}
			})

			// 注册翻译器
			switch locale {
			case "en":
				_ = enT.RegisterDefaultTranslations(validate, Trans)
			default:
				_ = zhT.RegisterDefaultTranslations(validate, Trans)
			}
			// 注意！因为要使用到trans实例, 所以这一步注册要放到trans初始化的后面
			for _, v := range customFuncSet {
				// 在校验器注册自定义的校验方法
				_ = validate.RegisterValidation(v.name, v.fun)
				// 解决自定义的校验方法不翻译问题, 注意上面注册的自定义的校验方法 v.name 要和下面的保持一致
				_ = validate.RegisterTranslation(v.name, Trans, regTranslator(v.name, v.tips), translate)
			}
		}
	})
}

// 翻译错误
func Message(errs Errors) []string {
	if Trans != nil {
		return removeTopStruct(errs.Translate(Trans))
	} else {
		return []string{"参数错误"}
	}
}

// 去掉结构体名称
func removeTopStruct(fields map[string]string) []string {
	res := make([]string, 0, len(fields))
	for _, err := range fields {
		// res[field[strings.Index(field, ".")+1:]] = err
		res = append(res, err)
	}
	return res
}

// 为自定义校验函数添加翻译功能
func regTranslator(tag string, msg string) vali10.RegisterTranslationsFunc {
	return func(trans ut.Translator) error {
		if err := trans.Add(tag, msg, true); err != nil {
			return err
		}
		return nil
	}
}

// 为自定义校验函数添加的翻译
func translate(trans ut.Translator, fe vali10.FieldError) string {
	msg, err := trans.T(fe.Tag(), fe.Field())
	if err != nil {
		return fe.Field() + " " + fe.Tag()
	}
	return msg
}

// 以下是自定义校验函数  函数名,函数,错误信息
var customFuncSet = []customFunc{
	{"minNow", afterNow, "参数:{0} 不能大于当前时间"},
	{"maxNow", beforeNow, "参数:{0} 不能小于当前时间"},
	{"mobile", mobile, "参数:{0} 手机号号码不合法"},
}

// 自定义校验函数,参数日期不能小于今天
func beforeNow(fl vali10.FieldLevel) bool {
	field := fl.Field().String()
	date, err := time.Parse("2006-01-02", field)
	if err != nil {
		return false
	}
	return time.Now().Before(date)
}

// 自定义校验函数,参数日期不能大于今天
func afterNow(fl vali10.FieldLevel) bool {
	field := fl.Field().String()
	date, err := time.Parse("2006-01-02", field)
	if err != nil {
		return false
	}
	return time.Now().After(date)
}

// 校验手机号
func mobile(fl vali10.FieldLevel) bool {
	mobile := fl.Field().String()
	return reMobileCN.MatchString(mobile)
}
