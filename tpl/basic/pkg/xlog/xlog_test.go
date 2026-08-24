package xlog

import (
	"bytes"
	"context"
	"strings"
	"testing"

	"basic/pkg/xconfig"

	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func TestCallerSkip_ShowsActualCaller(t *testing.T) {
	var buf bytes.Buffer

	encoderConfig := zapcore.EncoderConfig{
		TimeKey:        "timestamp",
		LevelKey:       "level",
		NameKey:        "logger",
		CallerKey:      "caller",
		FunctionKey:    "func",
		MessageKey:     "content",
		LineEnding:     zapcore.DefaultLineEnding,
		EncodeLevel:    zapcore.LowercaseLevelEncoder,
		EncodeTime:     zapcore.ISO8601TimeEncoder,
		EncodeDuration: zapcore.SecondsDurationEncoder,
		EncodeCaller:   zapcore.ShortCallerEncoder,
	}

	core := zapcore.NewCore(
		zapcore.NewJSONEncoder(encoderConfig),
		zapcore.AddSync(&buf),
		zap.DebugLevel,
	)

	logger := zap.New(
		core,
		zap.AddCaller(),
		zap.AddCallerSkip(1),
	)
	zap.ReplaceGlobals(logger)

	// 调用 xlog.Info
	ctx := context.Background()
	Info(ctx, "test caller skip")

	output := buf.String()
	// 期望 caller 包含当前测试文件 xlog_test.go, 而不是 xlog.go
	if !strings.Contains(output, "xlog_test.go") {
		t.Fatalf("expected caller to contain xlog_test.go, got: %s", output)
	}
	if strings.Contains(output, "xlog.go") {
		t.Fatalf("caller incorrectly reported xlog.go: %s", output)
	}
}

func TestNew_Config(t *testing.T) {
	conf := &xconfig.Conf{
		Log: xconfig.LogConf{
			Level:  "info",
			Stdout: true,
		},
	}
	l := New(conf)
	if l == nil {
		t.Fatal("expected non-nil logger")
	}
}
