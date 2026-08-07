package xlog

import (
	"bytes"
	"encoding/json"
	"encoding/xml"
	"fmt"
	"io"
	"net/url"
	"reflect"
	"regexp"
	"strings"
)

const (
	maskValue = "******"
)

var (
	// 需要过滤的敏感字段
	sensitiveFields = map[string]struct{}{
		// 认证凭证
		"password":      {},
		"passwd":        {},
		"pwd":           {},
		"token":         {},
		"secret":        {},
		"authorization": {},
		"api_key":       {},
		"apikey":        {},
		"access_token":  {},
		"refresh_token": {},
		"bearer":        {},
		"credential":    {},
		"session_id":    {},
		"session":       {},
		"cookie":        {},
		// 加密密钥
		"key":         {},
		"cert":        {},
		"private":     {},
		"private_key": {},
		"signing_key": {},
		// 支付信息
		"credit_card":  {},
		"card_number":  {},
		"cvv":          {},
		"cvc":          {},
		"pin":          {},
		"bank_account": {},
		// 个人隐私
		"id_card":  {},
		"idnumber": {},
		"ssn":      {},
	}

	// 支持的内容类型
	supportedContentTypes = map[string]ContentFilter{
		"application/json":                  JSONFilter{},
		"application/xml":                   XMLFilter{},
		"application/x-www-form-urlencoded": FormFilter{},
		"multipart/form-data":               FormFilter{},
		"text/plain":                        PlainTextFilter{},
	}

	// 敏感字段正则表达式
	sensitiveRegex = regexp.MustCompile(fmt.Sprintf(`(?i)(%s)["\']?\s*[:=]\s*["\']?([^"\'}\s&]+)`,
		strings.Join(mapKeysToSlice(sensitiveFields), "|")))
)

// ContentFilter 内容过滤器接口
type ContentFilter interface {
	Filter(content []byte) (string, error)
}

// JSONFilter JSON过滤器
type JSONFilter struct{}

func (f JSONFilter) Filter(content []byte) (string, error) {
	var data interface{}
	if err := json.Unmarshal(content, &data); err != nil {
		return "", err
	}
	filtered := filterValue(data)
	result, err := json.Marshal(filtered)
	if err != nil {
		return "", err
	}
	return string(result), nil
}

// XMLFilter XML过滤器
type XMLFilter struct{}

func (f XMLFilter) Filter(content []byte) (string, error) {
	dec := xml.NewDecoder(bytes.NewReader(content))
	var buf strings.Builder
	var elemStack []string

	for {
		token, err := dec.Token()
		if err == io.EOF {
			break
		}
		if err != nil {
			return "", err
		}

		switch t := token.(type) {
		case xml.StartElement:
			elemStack = append(elemStack, t.Name.Local)
			buf.WriteString("<" + t.Name.Local)
			for _, attr := range t.Attr {
				attrName := attr.Name.Local
				attrValue := attr.Value
				if isSensitive(attrName) {
					attrValue = maskValue
				}
				buf.WriteString(fmt.Sprintf(` %s="%s"`, attrName, attrValue))
			}
			buf.WriteString(">")
		case xml.EndElement:
			if len(elemStack) > 0 {
				elemStack = elemStack[:len(elemStack)-1]
			}
			buf.WriteString("</" + t.Name.Local + ">")
		case xml.CharData:
			current := ""
			if len(elemStack) > 0 {
				current = elemStack[len(elemStack)-1]
			}
			text := string(t)
			if isSensitive(current) {
				buf.WriteString(maskValue)
			} else {
				buf.WriteString(filterString(text))
			}
		case xml.Comment:
			buf.WriteString("<!--")
			buf.WriteString(filterString(string(t)))
			buf.WriteString("-->")
		case xml.ProcInst:
			buf.WriteString("<?" + t.Target + " " + string(t.Inst) + "?>")
		case xml.Directive:
			buf.WriteString("<!" + string(t) + ">")
		}
	}
	return buf.String(), nil
}

// FormFilter 表单过滤器
type FormFilter struct{}

func (f FormFilter) Filter(content []byte) (string, error) {
	values, err := url.ParseQuery(string(content))
	if err != nil {
		return "", err
	}
	return filterFormValues(values), nil
}

// FilterForm 过滤表单数据
func FilterForm(form url.Values) string {
	return filterFormValues(form)
}

// filterFormValues 过滤表单值
func filterFormValues(values url.Values) string {
	if len(values) == 0 {
		return ""
	}

	result := make(url.Values)
	for key, vals := range values {
		if isSensitive(key) {
			result[key] = []string{maskValue}
		} else {
			// 检查值是否包含敏感信息
			filteredVals := make([]string, len(vals))
			for i, val := range vals {
				filteredVals[i] = filterString(val)
			}
			result[key] = filteredVals
		}
	}
	return result.Encode()
}

// PlainTextFilter 纯文本过滤器
type PlainTextFilter struct{}

func (f PlainTextFilter) Filter(content []byte) (string, error) {
	return filterString(string(content)), nil
}

// FilterContent 过滤内容
func FilterContent(contentType string, content []byte) string {
	if len(content) == 0 {
		return ""
	}

	contentType = strings.ToLower(strings.Split(contentType, ";")[0])
	filter, ok := supportedContentTypes[contentType]
	if !ok {
		return "[unsupported content type]"
	}

	filtered, err := filter.Filter(content)
	if err != nil {
		return string(content) // 如果过滤失败，返回原始内容
	}

	return filtered
}

// FilterHeaders 过滤请求头中的敏感信息
// 多值 header 用 ", " 拼接(如 Set-Cookie),敏感 header 整体脱敏。
func FilterHeaders(headers map[string][]string) map[string]string {
	if len(headers) == 0 {
		return nil
	}

	filtered := make(map[string]string, len(headers))
	for key, values := range headers {
		if len(values) == 0 {
			continue
		}

		if isSensitive(key) {
			filtered[key] = maskValue
		} else {
			filtered[key] = strings.Join(values, ", ")
		}
	}
	return filtered
}

// FilterStruct 过滤结构体
// 递归遍历结构体字段,对敏感字段进行脱敏。
// 对非 string 类型(如 []byte, int, float 等)统一替换为零值,避免信息泄漏。
func FilterStruct(v interface{}) interface{} {
	val := reflect.ValueOf(v)
	if val.Kind() == reflect.Ptr {
		val = val.Elem()
	}

	if val.Kind() != reflect.Struct {
		return v
	}

	result := reflect.New(val.Type()).Elem()
	for i := 0; i < val.NumField(); i++ {
		field := val.Field(i)
		fieldType := val.Type().Field(i)

		if isSensitive(fieldType.Name) || isSensitive(fieldType.Tag.Get("json")) {
			// 所有敏感字段统一清零,不再区分 string/非 string
			result.Field(i).Set(reflect.Zero(field.Type()))
			continue
		}

		switch field.Kind() {
		case reflect.Struct:
			result.Field(i).Set(reflect.ValueOf(FilterStruct(field.Interface())))
		case reflect.Slice, reflect.Array:
			newSlice := reflect.MakeSlice(field.Type(), field.Len(), field.Cap())
			for j := 0; j < field.Len(); j++ {
				elem := field.Index(j)
				if elem.Kind() == reflect.Struct {
					newSlice.Index(j).Set(reflect.ValueOf(FilterStruct(elem.Interface())))
				} else {
					newSlice.Index(j).Set(elem)
				}
			}
			result.Field(i).Set(newSlice)
		default:
			result.Field(i).Set(field)
		}
	}

	return result.Interface()
}

// 辅助函数
func filterValue(v interface{}) interface{} {
	switch val := v.(type) {
	case map[string]interface{}:
		return filterMap(val)
	case []interface{}:
		return filterArray(val)
	case map[interface{}]interface{}:
		return filterInterfaceMap(val)
	case string:
		return filterString(val)
	default:
		return v
	}
}

func filterMap(data map[string]interface{}) map[string]interface{} {
	if data == nil {
		return nil
	}

	result := make(map[string]interface{}, len(data))
	for key, value := range data {
		if isSensitive(key) {
			result[key] = maskValue
			continue
		}
		result[key] = filterValue(value)
	}
	return result
}

func filterArray(arr []interface{}) []interface{} {
	if arr == nil {
		return nil
	}

	result := make([]interface{}, len(arr))
	for i, value := range arr {
		result[i] = filterValue(value)
	}
	return result
}

func filterInterfaceMap(data map[interface{}]interface{}) map[interface{}]interface{} {
	if data == nil {
		return nil
	}

	result := make(map[interface{}]interface{}, len(data))
	for key, value := range data {
		keyStr := fmt.Sprint(key)
		if isSensitive(keyStr) {
			result[key] = maskValue
			continue
		}
		result[key] = filterValue(value)
	}
	return result
}

func filterString(s string) string {
	return sensitiveRegex.ReplaceAllString(s, "${1}="+maskValue)
}

func isSensitive(field string) bool {
	field = strings.ToLower(field)
	for sensitive := range sensitiveFields {
		if strings.Contains(field, sensitive) {
			return true
		}
	}
	return false
}

func mapKeysToSlice(m map[string]struct{}) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	return keys
}
