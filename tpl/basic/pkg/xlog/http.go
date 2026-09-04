package xlog

import (
	"net"
	"net/http"
	"strings"
)

// HTTPRequestInfo 请求信息结构
type HTTPRequestInfo struct {
	Host       string            `json:"host"`
	Method     string            `json:"method"`
	Path       string            `json:"path"`
	Query      string            `json:"query,omitempty"`
	Headers    map[string]string `json:"headers"`
	Body       string            `json:"body,omitempty"`
	IP         string            `json:"ip"`
	UserAgent  string            `json:"user_agent,omitempty"`
	ClientIP   string            `json:"client_ip"`
	RemoteAddr string            `json:"remote_addr"`
	Referer    string            `json:"referer,omitempty"`
}

// HTTPResponseInfo 响应信息结构
type HTTPResponseInfo struct {
	Method   string            `json:"method"`
	Path     string            `json:"path"`
	HttpCode int               `json:"code"`
	Headers  map[string]string `json:"headers,omitempty"`
	Body     string            `json:"body,omitempty"`
	Duration float64           `json:"duration"`
}

// GetClientIP 获取客户端真实IP
func GetClientIP(r *http.Request) string {
	headers := []string{
		"X-Real-IP",
		"X-Forwarded-For",
		"X-Client-IP",
		"CF-Connecting-IP",
		"True-Client-IP",
	}

	for _, header := range headers {
		if ip := r.Header.Get(header); ip != "" {
			if header == "X-Forwarded-For" {
				return strings.TrimSpace(strings.Split(ip, ",")[0])
			}
			return strings.TrimSpace(ip)
		}
	}

	if r.RemoteAddr != "" {
		if host, _, err := net.SplitHostPort(r.RemoteAddr); err == nil {
			return host
		}
		return strings.Split(r.RemoteAddr, ":")[0]
	}

	return ""
}
