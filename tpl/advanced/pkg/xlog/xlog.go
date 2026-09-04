package xlog

import (
	"context"
	"fmt"
	"os"
	"strings"
	"sync"

	"advanced/pkg/xconfig"

	"google.golang.org/grpc"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type (
	Logger    = zap.Logger
	loggerKey struct{}

	ConfLog struct {
		Level    string
		FilePath string
		Stdout   bool
	}
)

// kafkaClosers 保存所有日志 Kafka Writer 的关闭函数,由 New 追加,Close 统一关闭。
// 用切片而非单个变量:broker 模板中父进程与 fork 出的子进程各自 New 一个 logger,
// 单变量会被后者覆盖导致前者的 Writer 永不关闭(缓冲日志丢失/连接泄漏)。
// 受 closersMu 保护,Close 并发安全且只执行一次。
var (
	kafkaClosers  []func() error
	closersMu     sync.Mutex
	closersClosed bool
)

// New 创建一个新的日志记录器
func New(conf *xconfig.Conf) *Logger {
	opSet := make([]Option, 0)
	if conf.Log.Stdout {
		opSet = append(opSet, withEnableConsole())
	}

	switch strings.ToLower(conf.Log.Level) {
	case "info":
		opSet = append(opSet, withInfoLevel())
	case "warn":
		opSet = append(opSet, withWarnLevel())
	case "error":
		opSet = append(opSet, withErrorLevel())
	default: // "debug"
		opSet = append(opSet, withDebugLevel())
	}

	if conf.Log.FilePath != "" {
		opSet = append(opSet, withFilePath(conf.Log.FilePath))
	}

	// 是否启用 kafkaHook
	if conf.Log.KafkaTopic != "" && len(conf.Log.KafkaBrokers) > 0 {
		opSet = append(opSet, withKafka(conf.Log.KafkaTopic, conf.Log.KafkaBrokers))
	}

	opt := newOption(opSet...)

	// similar to zap.NewProductionEncoderConfig()
	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    "func",
		MessageKey:     "content",
		StacktraceKey:  "stacktrace",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,  // 小写编码器
		EncodeTime:     zapcore.ISO8601TimeEncoder,     // 使用ISO8601格式
		EncodeDuration: zapcore.SecondsDurationEncoder, // 使用Float64格式的持续时间
		EncodeCaller:   zapcore.ShortCallerEncoder,     // 包名:文件名
		EncodeName:     zapcore.FullNameEncoder,
	}

	// 创建具体的 zapCore.Core
	var cores []zapcore.Core

	// 文件日志
	if opt.fileHook != nil {
		fileEncoder := zapcore.NewJSONEncoder(encoderConfig) // json格式
		cores = append(cores, zapcore.NewCore(fileEncoder, zapcore.AddSync(opt.fileHook), opt.level))
	}

	// 控制台日志
	if opt.enableConsole {
		consoleEncoderConfig := encoderConfig
		consoleEncoderConfig.EncodeLevel = zapcore.LowercaseColorLevelEncoder
		consoleEncoderConfig.EncodeCaller = zapcore.FullCallerEncoder // 全路径编码器
		consoleEncoder := zapcore.NewConsoleEncoder(consoleEncoderConfig)
		cores = append(cores, zapcore.NewCore(consoleEncoder, zapcore.AddSync(os.Stdout), opt.level))
	}

	// Kafka日志 - 使用专门的编码器配置
	if opt.kafkaHook != nil {
		kafkaEncoderConfig := encoderConfig
		kafkaEncoder := zapcore.NewJSONEncoder(kafkaEncoderConfig) // json格式
		cores = append(cores, zapcore.NewCore(kafkaEncoder, zapcore.AddSync(opt.kafkaHook), opt.level))
		// 保存关闭句柄,供进程退出时 Close 调用,避免 Async Writer 缓冲日志丢失。
		// 多实例(broker 父子进程)各自追加,Close 时统一关闭,不再互相覆盖。
		if kw, ok := opt.kafkaHook.(*KafkaWriter); ok {
			closersMu.Lock()
			kafkaClosers = append(kafkaClosers, kw.Close)
			closersMu.Unlock()
		}
	}

	logger := zap.New(
		zapcore.NewTee(cores...),
		zap.AddCaller(),                   // 显示打日志点的文件名和行数
		zap.AddStacktrace(zap.ErrorLevel), // 为错误添加堆栈跟踪
		zap.AddCallerSkip(1),              // 跳过 xlog 封装层(1层),准确显示业务层调用点文件名与行号
	)

	zap.ReplaceGlobals(logger) // 替换zap包中全局的logger实例，后续在其他包中只需使用zap.L()调用即可

	// 异步写入
	_ = logger.Sync()
	return logger
}

// Close 关闭日志组件持有的底层资源(如日志 Kafka Writer),应在进程退出前调用。
// 关闭所有由 New 注册的 Kafka Writer(多实例场景下父/子进程各一份),仅执行一次。
func Close() error {
	closersMu.Lock()
	if closersClosed {
		closersMu.Unlock()
		return nil
	}
	closersClosed = true
	closers := kafkaClosers
	kafkaClosers = nil
	closersMu.Unlock()

	var firstErr error
	for _, c := range closers {
		if err := c(); err != nil && firstErr == nil {
			firstErr = err
		}
	}
	return firstErr
}

func KVStr(key string, val string) zap.Field {
	return zap.Field{Key: key, Type: zapcore.StringType, String: val}
}

func KVInt(key string, val int) zap.Field {
	return zap.Int(key, val)
}

func KVAny(key string, val any) zap.Field {
	return zap.Any(key, val)
}

func KVErr(err error) zap.Field {
	return zap.NamedError("error", err)
}

// 统一的日志内容结构
type logContent struct {
	Message string `json:"msg,omitempty"`    // 日志消息
	Error   string `json:"error,omitempty"`  // 错误信息
	Params  any    `json:"params,omitempty"` // 参数信息
}

// splitFieldsAndParams 分离出 zap.Field(作为顶层结构化日志字段)与其它通用参数
func splitFieldsAndParams(params ...any) ([]zap.Field, []any) {
	var fields []zap.Field
	var rest []any
	for _, p := range params {
		if f, ok := p.(zap.Field); ok {
			fields = append(fields, f)
		} else {
			rest = append(rest, p)
		}
	}
	return fields, rest
}

// Info 统一的 Info 级别日志记录。支持传 zap.Field(如 KVStr/KVInt/KVErr)作为顶层结构化字段,
// 非 zap.Field 类型的参数存入 content.params 数组。
func Info(ctx context.Context, message string, params ...any) {
	logger := fromContext(ctx)
	fields, rest := splitFieldsAndParams(params...)
	c := logContent{
		Message: message,
	}
	if len(rest) > 0 {
		c.Params = rest
	}
	fields = append(fields, zap.Any("content", c))
	logger.Info("", fields...)
}
func Infof(ctx context.Context, message string, params ...any) {
	logger := fromContext(ctx)
	logger.Info("", zap.Any("content", logContent{
		Message: fmt.Sprintf(message, params...),
	}))
}

// Error 统一的 Error 级别日志记录。支持传 zap.Field 作为顶层结构化字段。
func Error(ctx context.Context, message string, err error, params ...any) {
	logger := fromContext(ctx)
	fields, rest := splitFieldsAndParams(params...)
	c := logContent{
		Message: message,
	}
	if err != nil {
		c.Error = err.Error()
		fields = append(fields, zap.NamedError("error", err))
	}
	if len(rest) > 0 {
		c.Params = rest
	}
	fields = append(fields, zap.Any("content", c))
	logger.Error("", fields...)
}
func Errorf(ctx context.Context, message string, params ...any) {
	logger := fromContext(ctx)
	logger.Error("", zap.Any("content", logContent{
		Message: fmt.Sprintf(message, params...),
	}))
}

// Warn 统一的 Warn 级别日志记录。支持传 zap.Field 作为顶层结构化字段。
func Warn(ctx context.Context, message string, params ...any) {
	logger := fromContext(ctx)
	fields, rest := splitFieldsAndParams(params...)
	c := logContent{
		Message: message,
	}
	if len(rest) > 0 {
		c.Params = rest
	}
	fields = append(fields, zap.Any("content", c))
	logger.Warn("", fields...)
}
func Warnf(ctx context.Context, message string, params ...any) {
	logger := fromContext(ctx)
	logger.Warn("", zap.Any("content", logContent{
		Message: fmt.Sprintf(message, params...),
	}))
}

// Debug 统一的 Debug 级别日志记录。支持传 zap.Field 作为顶层结构化字段。
func Debug(ctx context.Context, message string, params ...any) {
	logger := fromContext(ctx)
	fields, rest := splitFieldsAndParams(params...)
	c := logContent{
		Message: message,
	}
	if len(rest) > 0 {
		c.Params = rest
	}
	fields = append(fields, zap.Any("content", c))
	logger.Debug("", fields...)
}
func Debugf(ctx context.Context, message string, params ...any) {
	logger := fromContext(ctx)
	logger.Debug("", zap.Any("content", logContent{
		Message: fmt.Sprintf(message, params...),
	}))
}

// 将 logger 添加到 context 中
func WithLogger(ctx context.Context, logger *zap.Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

// 从 context 中获取 logger
func fromContext(ctx context.Context) *zap.Logger {
	if ctx == nil {
		return zap.L()
	}
	if logger, ok := ctx.Value(loggerKey{}).(*zap.Logger); ok {
		return logger
	}
	// 兜底：若 ctx 含有 trace 信息，自动附带 trace_id 与 span_id
	return WithTraceID(ctx, zap.L())
}

// 添加追踪信息到 logger
func WithTraceID(ctx context.Context, logger *zap.Logger) *zap.Logger {
	spanContext := trace.SpanContextFromContext(ctx)
	if !spanContext.IsValid() {
		return logger
	}

	return logger.With(
		zap.String("trace_id", spanContext.TraceID().String()),
		zap.String("span_id", spanContext.SpanID().String()),
	)
}

// GRPC 添加追踪信息到 logger
func WithTraceIDGRPC(ctx context.Context, logger *zap.Logger, info *grpc.UnaryServerInfo) *zap.Logger {
	spanContext := trace.SpanContextFromContext(ctx)
	if !spanContext.IsValid() {
		return logger
	}

	return logger.With(
		zap.String("trace_id", spanContext.TraceID().String()),
		zap.String("span_id", spanContext.SpanID().String()),
		zap.String("method", info.FullMethod),
	)
}
