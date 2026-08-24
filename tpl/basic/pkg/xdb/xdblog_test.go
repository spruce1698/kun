package xdb

import (
	"context"
	"strings"
	"testing"
	"time"

	"gorm.io/gorm/logger"
)

func TestXdbLogger_TraceCapturesCaller(t *testing.T) {
	l := initLog(nil, logger.Config{
		LogLevel: logger.Info,
	})

	// 模拟 GORM 触发 Trace
	ctx := context.Background()
	begin := time.Now()

	callerFound := findCaller()
	if !strings.Contains(callerFound, "xdblog_test.go") {
		t.Fatalf("expected caller to contain xdblog_test.go, got: %s", callerFound)
	}

	l.Trace(ctx, begin, func() (string, int64) {
		return "SELECT 1", 1
	}, nil)
}

func TestTrimCallerPath(t *testing.T) {
	cases := []struct {
		input string
		line  int
		want  string
	}{
		{"E:/workspace/prajna/kun/tpl/advanced/internal/repository/db/demo.go", 125, "/repository/db/demo.go:125"},
		{"E:\\workspace\\prajna\\kun\\tpl\\advanced\\internal\\service\\svc\\demo.go", 80, "/service/svc/demo.go:80"},
		{"/app/internal/handler/demo/demo.go", 42, "/handler/demo/demo.go:42"},
	}

	for _, c := range cases {
		got := trimCallerPath(c.input, c.line)
		if got != c.want {
			t.Fatalf("trimCallerPath(%q, %d) = %q, want %q", c.input, c.line, got, c.want)
		}
	}
}
