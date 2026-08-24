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
	// 需要过滤的敏感字段精确集合
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
		"cert":        {},
		"private":     {},
		"private_key": {},
		"secret_key":  {},
		"signing_key": {},
		// 支付信息
		"credit_card":  {},
		"card_number":  {},
		"cvv":          {},
		"cvc":          {},
		"pin":          {},
		"bank_account": {},
		// 个人隐私
		"id_card":   {},
		"idnumber":  {},
		"id_number": {},
		"ssn":       {},
	}

	// 敏感模糊匹配特征词(仅在精确匹配不中时执行)
	sensitiveFuzzyKeywords = []string{
		"password", "passwd", "pwd", "token", "secret", "auth", "credential",
		"credit_card", "card_number", "cvv", "cvc", "pin", "ssn", "id_card", "idnumber",
		"private_key", "secret_key", "signing_key", "api_key", "apikey",
	}

	// 字节级预检特征词(用于 Fast-Path 零分配预检)
	sensitiveByteKeywords = [][]byte{
		[]byte("pass"), []byte("pwd"), []byte("token"), []byte("secret"),
		[]byte("auth"), []byte("cookie"), []byte("session"),
		[]byte("card"), []byte("cvv"), []byte("cvc"), []byte("pin"),
		[]byte("ssn"), []byte("cert"), []byte("priv"), []byte("api_key"), []byte("apikey"),
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

// mayContainSensitiveBytes 快速检测字节流是否可能含有敏感字段(Fast Path)
func mayContainSensitiveBytes(b []byte) bool {
	lower := bytes.ToLower(b)
	for _, kw := range sensitiveByteKeywords {
		if bytes.Contains(lower, kw) {
			return true
		}
	}
	return false
}

func mayContainSensitiveString(s string) bool {
	lower := strings.ToLower(s)
	for _, kw := range sensitiveByteKeywords {
		if strings.Contains(lower, string(kw)) {
			return true
		}
	}
	return false
}

// JSONFilter JSON过滤器
type JSONFilter struct{}

func (f JSONFilter) Filter(content []byte) (string, error) {
	if len(content) == 0 {
		return "", nil
	}

	// Fast Path: 绝大多数无敏感信息的请求/响应无需进行反序列化与反射重建,直接零拷贝返回
	if !mayContainSensitiveBytes(content) {
		return string(content), nil
	}

	var data any
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
	if len(content) == 0 {
		return "", nil
	}
	if !mayContainSensitiveBytes(content) {
		return string(content), nil
	}

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

	result := make(url.Values, len(values))
	for key, vals := range values {
		if isSensitive(key) {
			result[key] = []string{maskValue}
		} else {
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

	// 快速截取并匹配 Content-Type (避免 strings.Split 分配)
	ct := contentType
	if idx := strings.IndexByte(ct, ';'); idx != -1 {
		ct = ct[:idx]
	}
	ct = strings.TrimSpace(strings.ToLower(ct))

	filter, ok := supportedContentTypes[ct]
	if !ok {
		if strings.Contains(ct, "json") {
			filter = JSONFilter{}
		} else {
			return "[unsupported content type]"
		}
	}

	filtered, err := filter.Filter(content)
	if err != nil {
		return string(content) // 如果过滤失败，返回原始内容
	}

	return filtered
}

// FilterHeaders 过滤请求头中的敏感信息
func FilterHeaders(headers map[string][]string) map[string]string {
	if len(headers) == 0 {
		return nil
	}

	filtered := make(map[string]string, len(headers))
	for key, values := range headers {
		if len(values) == 0 {
			continue
		}

		lowerKey := strings.ToLower(key)
		switch lowerKey {
		case "authorization", "cookie", "set-cookie", "x-api-key", "proxy-authorization":
			filtered[key] = maskValue
		default:
			if isSensitive(lowerKey) {
				filtered[key] = maskValue
			} else {
				filtered[key] = values[0]
			}
		}
	}
	return filtered
}

// FilterStruct 过滤结构体
func FilterStruct(v any) any {
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
			if field.Kind() == reflect.String {
				result.Field(i).SetString(maskValue)
			}
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
func filterValue(v any) any {
	switch val := v.(type) {
	case map[string]any:
		return filterMap(val)
	case []any:
		return filterArray(val)
	case map[any]any:
		return filterInterfaceMap(val)
	case string:
		return filterString(val)
	default:
		return v
	}
}

func filterMap(data map[string]any) map[string]any {
	if data == nil {
		return nil
	}

	result := make(map[string]any, len(data))
	for key, value := range data {
		if isSensitive(key) {
			result[key] = maskValue
			continue
		}
		result[key] = filterValue(value)
	}
	return result
}

func filterArray(arr []any) []any {
	if arr == nil {
		return nil
	}

	result := make([]any, len(arr))
	for i, value := range arr {
		result[i] = filterValue(value)
	}
	return result
}

func filterInterfaceMap(data map[any]any) map[any]any {
	if data == nil {
		return nil
	}

	result := make(map[any]any, len(data))
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
	if !mayContainSensitiveString(s) {
		return s
	}
	return sensitiveRegex.ReplaceAllString(s, "${1}="+maskValue)
}

func isSensitive(field string) bool {
	if len(field) == 0 {
		return false
	}
	field = strings.ToLower(field)
	if _, ok := sensitiveFields[field]; ok {
		return true
	}
	for _, kw := range sensitiveFuzzyKeywords {
		if strings.Contains(field, kw) {
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
