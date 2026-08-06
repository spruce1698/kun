package middleware

import (
	"bytes"
	"fmt"
	"io"
	"strings"
	"sync"
	"time"

	"advanced/pkg/xlog"

	"github.com/gin-gonic/gin"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/attribute"
)

var bufferPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 0, 10240))
	},
}

const maxLogBodySize = 4096      // 日志记录的请求/响应 body 最大字节数
const maxRequestBodySize = 65536 // 完整读取 body 上限 64KB（超过则截断记录前 4KB）

// responseWriter 包装 gin.ResponseWriter 以捕获响应内容
type responseWriter struct {
	gin.ResponseWriter
	body    *bytes.Buffer
	bodyLen int // 已写入 body 的总字节数
}

func (w *responseWriter) Write(b []byte) (int, error) {
	// 仅记录前 maxLogBodySize 字节到日志 buffer，避免大响应体占用过多内存
	remain := maxLogBodySize - w.bodyLen
	if remain > 0 {
		if remain > len(b) {
			remain = len(b)
		}
		w.body.Write(b[:remain])
		w.bodyLen += remain
	}
	return w.ResponseWriter.Write(b)
}

// TracingWithLogger 创建一个带日志的追踪中间件
func TracingWithLogger(log *xlog.Logger, serviceName string) gin.HandlerFunc {

	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		method := c.Request.Method
		contentType := c.GetHeader("Content-Type")
		var requestBody []byte

		// 获取或创建 trace
		ctx := c.Request.Context()
		spanName := fmt.Sprintf("%s %s", method, path)

		// 创建新的 span
		tracer := otel.Tracer(serviceName)
		ctx, span := tracer.Start(ctx, spanName)
		defer span.End()

		if method == "POST" || method == "PUT" {
			// 处理表单数据
			switch {
			case strings.Contains(contentType, "multipart/form-data"):
				if err := c.Request.ParseMultipartForm(32 << 20); err == nil { // 32MB max
					requestBody = []byte(xlog.FilterForm(c.Request.PostForm))
				}
			case strings.Contains(contentType, "application/x-www-form-urlencoded"):
				if err := c.Request.ParseForm(); err == nil {
					requestBody = []byte(xlog.FilterForm(c.Request.PostForm))
				}
			case c.Request.Body != nil:
				contentLen := c.Request.ContentLength
				switch {
				case contentLen > 0 && contentLen <= maxRequestBodySize:
					// 小 body：完整读取、记录、重建供 handler 使用
					requestBody = make([]byte, contentLen)
					if _, err := io.ReadFull(c.Request.Body, requestBody); err == nil {
						c.Request.Body = io.NopCloser(bytes.NewReader(requestBody))
					} else {
						requestBody = nil
					}
				case contentLen > maxRequestBodySize:
					// 大 body（文件上传等）：不读取，仅记录元数据用于追踪
					requestBody = []byte(fmt.Sprintf("[large body, %d bytes, skipped]", contentLen))
				default:
					// chunked / 未知长度：安全读取最多 64KB，重建供 handler 使用
					buf := bufferPool.Get().(*bytes.Buffer)
					buf.Reset()
					limited := io.LimitReader(c.Request.Body, maxRequestBodySize)
					if _, err := io.Copy(buf, limited); err == nil && buf.Len() > 0 {
						data := buf.Bytes()
						requestBody = make([]byte, len(data))
						copy(requestBody, data)
						// 重建 body 供 handler 使用
						c.Request.Body = io.NopCloser(bytes.NewReader(data))
					}
					bufferPool.Put(buf)
				}
			}
		}

		// 包装 ResponseWriter
		buf := bufferPool.Get().(*bytes.Buffer)
		buf.Reset()
		defer bufferPool.Put(buf)

		w := &responseWriter{
			ResponseWriter: c.Writer,
			body:           buf,
		}
		c.Writer = w

		// 设置带追踪的 logger
		loggerWithTrace := xlog.WithTraceID(ctx, log)
		ctx = xlog.WithLogger(ctx, loggerWithTrace)
		c.Request = c.Request.WithContext(ctx)

		// 记录请求信息
		requestInfo := xlog.HTTPRequestInfo{
			Host:       c.Request.Host,
			Method:     method,
			Path:       path,
			Query:      c.Request.URL.RawQuery,
			Headers:    xlog.FilterHeaders(c.Request.Header),
			Body:       xlog.FilterContent(contentType, requestBody),
			IP:         c.ClientIP(),
			UserAgent:  c.Request.UserAgent(),
			ClientIP:   xlog.GetClientIP(c.Request),
			RemoteAddr: c.Request.RemoteAddr,
			Referer:    c.Request.Referer(),
		}

		loggerWithTrace.Info("", xlog.KVAny("content", requestInfo))

		c.Next()

		// 记录响应信息
		duration := time.Since(start)

		// 记录响应状态和耗时
		responseInfo := xlog.HTTPResponseInfo{
			Method:   method,
			Path:     path,
			HttpCode: c.Writer.Status(),
			Headers:  xlog.FilterHeaders(w.Header()),
			Body:     xlog.FilterContent(w.Header().Get("Content-Type"), w.body.Bytes()),
			Duration: float64(duration.Microseconds()) / 1000.0, // 转换为毫秒
		}
		// 根据状态码选择日志级别
		if c.Writer.Status() >= 500 {
			loggerWithTrace.Error("", xlog.KVAny("content", responseInfo))
		} else if c.Writer.Status() >= 400 {
			loggerWithTrace.Warn("", xlog.KVAny("content", responseInfo))
		} else {
			loggerWithTrace.Info("", xlog.KVAny("content", responseInfo))
		}

		// 记录追踪信息
		// 设置 span 属性（response body 截断避免泄露敏感数据）
		spanBody := responseInfo.Body
		if len(spanBody) > 512 {
			spanBody = spanBody[:512] + "...(truncated)"
		}
		span.SetAttributes(
			attribute.String("http.method", method),
			attribute.String("http.path", path),
			attribute.String("http.user_agent", requestInfo.UserAgent),
			attribute.String("http.client_ip", requestInfo.ClientIP),
			attribute.Int("http.status_code", responseInfo.HttpCode),
			attribute.String("http.response_body", spanBody),
			attribute.Float64("duration", float64(duration.Microseconds())/1000.0),
		)
	}
}
