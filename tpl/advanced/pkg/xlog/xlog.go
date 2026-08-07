package xlog

import (
	"context"
	"fmt"
	"os"
	"strings"

	"advanced/pkg/xconfig"

	"google.golang.org/grpc"

	"go.opentelemetry.io/otel/trace"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

type (
	// Logger 封装 *zap.Logger,所有日志方法通过实例调用,不依赖全局状态。
	Logger struct {
		zl         *zap.Logger
		kafkaClose func() error // 可选,Kafka Writer 关闭句柄
		traceBound bool         // 是否已绑定 trace_id(避免 logAt 重复附加)
	}
	loggerKey struct{}

	ConfLog struct {
		Level    string
		FilePath string
		Stdout   bool
	}
)

// New 创建一个新的日志记录器,不再 ReplaceGlobals。
// Logger 通过 WithLogger(ctx, log) 存入 context,由 xlog.Info 等包级函数取出。
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
	var kafkaClose func() error

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
		if kw, ok := opt.kafkaHook.(*KafkaWriter); ok {
			kafkaClose = kw.Close
		}
	}

	zl := zap.New(
		zapcore.NewTee(cores...),
		zap.AddCaller(),                   // 显示打日志点的文件名和行数
		zap.AddStacktrace(zap.ErrorLevel), // 为错误添加堆栈跟踪
		zap.AddCallerSkip(1),              // 避免zap始终将包装器(wrapper)代码报告为调用方。
	)

	return &Logger{zl: zl, kafkaClose: kafkaClose}
}

// Sync 刷新 logger 缓冲
func (l *Logger) Sync() error {
	return l.zl.Sync()
}

// Close 先 Sync 刷新缓冲,再关闭 Logger 持有的底层资源(如 Kafka Writer),应在进程退出前调用。
func (l *Logger) Close() error {
	if err := l.zl.Sync(); err != nil {
		// Sync 失败多为 stderr/stdout 不可写(如容器无 tty),降级用 fmt 提示,不再递归写日志。
		fmt.Printf("xlog sync warn: %v\n", err)
	}
	if l.kafkaClose != nil {
		return l.kafkaClose()
	}
	return nil
}

// WithTraceID 返回派生 Logger,自动附加 trace_id/span_id。
// 若 ctx 无有效 span 则返回自身。
// 中间件入口调用一次即可,后续同 ctx 内日志复用同一派生 Logger,避免每条日志重复分配。
func (l *Logger) WithTraceID(ctx context.Context) *Logger {
	if l.traceBound {
		return l
	}
	spanContext := trace.SpanContextFromContext(ctx)
	if !spanContext.IsValid() {
		return l
	}
	return &Logger{
		zl: l.zl.With(
			zap.String("trace_id", spanContext.TraceID().String()),
			zap.String("span_id", spanContext.SpanID().String()),
		),
		kafkaClose: l.kafkaClose,
		traceBound: true,
	}
}

// WithTraceIDGRPC 返回派生 Logger,附加 trace_id/span_id/method。
func (l *Logger) WithTraceIDGRPC(ctx context.Context, info *grpc.UnaryServerInfo) *Logger {
	if l.traceBound {
		return l
	}
	spanContext := trace.SpanContextFromContext(ctx)
	if !spanContext.IsValid() {
		return l
	}
	return &Logger{
		zl: l.zl.With(
			zap.String("trace_id", spanContext.TraceID().String()),
			zap.String("span_id", spanContext.SpanID().String()),
			zap.String("method", info.FullMethod),
		),
		kafkaClose: l.kafkaClose,
		traceBound: true,
	}
}

// logAt 在 logger 之上按 ctx 一次性附加 trace_id 后输出日志。
// 若 logger 已通过 WithTraceID 绑定 trace 字段(中间件入口),则直接输出避免重复;
// 否则在此按 ctx 补上,保证未预绑定的场景(如 broker 消费)链路追踪不丢失。
func (l *Logger) logAt(ctx context.Context, lvl zapcore.Level, c logContent) {
	zl := l.zl
	if !l.traceBound {
		if sc := trace.SpanContextFromContext(ctx); sc.IsValid() {
			zl = zl.With(
				zap.String("trace_id", sc.TraceID().String()),
				zap.String("span_id", sc.SpanID().String()),
			)
		}
	}
	switch lvl {
	case zapcore.DebugLevel:
		zl.Debug("", zap.Any("content", c))
	case zapcore.InfoLevel:
		zl.Info("", zap.Any("content", c))
	case zapcore.WarnLevel:
		zl.Warn("", zap.Any("content", c))
	case zapcore.ErrorLevel:
		zl.Error("", zap.Any("content", c))
	}
}

// Info 输出 Info 级别日志,自动附加 trace_id,格式与 xlog.Info 一致。
func (l *Logger) Info(ctx context.Context, message string, params ...any) {
	l.logAt(ctx, zapcore.InfoLevel, newContent(message, nil, params))
}

// Error 输出 Error 级别日志,自动附加 trace_id,格式与 xlog.Error 一致。
func (l *Logger) Error(ctx context.Context, message string, err error, params ...any) {
	l.logAt(ctx, zapcore.ErrorLevel, newContent(message, err, params))
}

// Warn 输出 Warn 级别日志,自动附加 trace_id,格式与 xlog.Warn 一致。
func (l *Logger) Warn(ctx context.Context, message string, params ...any) {
	l.logAt(ctx, zapcore.WarnLevel, newContent(message, nil, params))
}

// Debug 输出 Debug 级别日志,自动附加 trace_id,格式与 xlog.Debug 一致。
func (l *Logger) Debug(ctx context.Context, message string, params ...any) {
	l.logAt(ctx, zapcore.DebugLevel, newContent(message, nil, params))
}

type logContent struct {
	Message string `json:"msg,omitempty"`    // 日志消息
	Error   string `json:"error,omitempty"`  // 错误信息
	Params  any    `json:"params,omitempty"` // 参数信息
}

// newContent 构造日志内容。params 非空时取首个元素作为 Params。
func newContent(message string, err error, params []any) logContent {
	c := logContent{Message: message}
	if err != nil {
		c.Error = err.Error()
	}
	if len(params) > 0 {
		c.Params = params[0]
	}
	return c
}

// Info 统一的 Info 级别日志记录,从 ctx 取 Logger 并自动附加 trace_id。
func Info(ctx context.Context, message string, params ...any) {
	fromContext(ctx).logAt(ctx, zapcore.InfoLevel, newContent(message, nil, params))
}
func Infof(ctx context.Context, message string, params ...any) {
	fromContext(ctx).logAt(ctx, zapcore.InfoLevel, logContent{Message: fmt.Sprintf(message, params...)})
}

// Error 统一的 Error 级别日志记录,从 ctx 取 Logger 并自动附加 trace_id。
func Error(ctx context.Context, message string, err error, params ...any) {
	fromContext(ctx).logAt(ctx, zapcore.ErrorLevel, newContent(message, err, params))
}
func Errorf(ctx context.Context, message string, params ...any) {
	fromContext(ctx).logAt(ctx, zapcore.ErrorLevel, logContent{Message: fmt.Sprintf(message, params...)})
}

// Warn 统一的 Warn 级别日志记录,从 ctx 取 Logger 并自动附加 trace_id。
func Warn(ctx context.Context, message string, params ...any) {
	fromContext(ctx).logAt(ctx, zapcore.WarnLevel, newContent(message, nil, params))
}
func Warnf(ctx context.Context, message string, params ...any) {
	fromContext(ctx).logAt(ctx, zapcore.WarnLevel, logContent{Message: fmt.Sprintf(message, params...)})
}

// Debug 统一的 Debug 级别日志记录,从 ctx 取 Logger 并自动附加 trace_id。
func Debug(ctx context.Context, message string, params ...any) {
	fromContext(ctx).logAt(ctx, zapcore.DebugLevel, newContent(message, nil, params))
}
func Debugf(ctx context.Context, message string, params ...any) {
	fromContext(ctx).logAt(ctx, zapcore.DebugLevel, logContent{Message: fmt.Sprintf(message, params...)})
}

// 将 logger 添加到 context 中
func WithLogger(ctx context.Context, logger *Logger) context.Context {
	return context.WithValue(ctx, loggerKey{}, logger)
}

// fromContext 从 context 中获取 Logger,不存在时返回 nop Logger。
// 不再回退到全局 zap.L()——日志实例必须通过注入或中间件存入 ctx。
func fromContext(ctx context.Context) *Logger {
	if logger, ok := ctx.Value(loggerKey{}).(*Logger); ok {
		return logger
	}
	// 无日志实例时返回 nop,避免 panic;同时让开发者发现未注入的问题。
	return &Logger{zl: zap.NewNop()}
}

// Ctx 从 context 中获取 Logger,不存在时返回 nop Logger。
// 公开的等价于 fromContext,供需要直接操作 Logger 实例的场景使用。
func Ctx(ctx context.Context) *Logger {
	return fromContext(ctx)
}

// WithTraceID 返回带 trace_id/span_id 的 Logger 副本。
// 保留此包级函数向后兼容,新代码建议用 log.WithTraceID(ctx) 方法。
func WithTraceID(ctx context.Context, logger *Logger) *Logger {
	return logger.WithTraceID(ctx)
}

// WithTraceIDGRPC 返回带 trace_id/span_id/method 的 Logger 副本。
func WithTraceIDGRPC(ctx context.Context, logger *Logger, info *grpc.UnaryServerInfo) *Logger {
	return logger.WithTraceIDGRPC(ctx, info)
}

// KVStr 构造字符串 Field
func KVStr(key string, val string) zap.Field {
	return zap.Field{Key: key, Type: zapcore.StringType, String: val}
}

// KVInt 构造整型 Field
func KVInt(key string, val int) zap.Field {
	return zap.Int(key, val)
}

// KVAny 构造任意类型 Field
func KVAny(key string, val any) zap.Field {
	return zap.Any(key, val)
}

// KVErr 构造错误 Field
func KVErr(err error) zap.Field {
	return zap.NamedError("error", err)
}
