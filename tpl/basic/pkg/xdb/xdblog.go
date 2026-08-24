/**
 * @Author:
 * @Date: 2024-03-28 15:06
 * @Desc: 数据库日志适配器(集成 GORM 与统一 xlog)
 */

package xdb

import (
	"context"
	"errors"
	"fmt"
	"runtime"
	"strconv"
	"strings"
	"time"

	"basic/pkg/xlog"

	"gorm.io/gorm/logger"
)

type xdbLogger struct {
	logger.Config
}

// findCaller 向上回溯调用栈，跳过 GORM 内部框架、runtime 以及 pkg/xdb、pkg/xlog 适配层，
// 精准定位到触发数据库操作的真实业务代码文件与行号，并裁剪绝对路径前缀（保留如 /repository/db/demo.go:125）。
func findCaller() string {
	for i := 1; i < 25; i++ {
		_, file, line, ok := runtime.Caller(i)
		if !ok {
			break
		}
		// 过滤 gorm 内部源码、runtime 源码以及项目自身的 xdb/xlog 适配层
		if strings.Contains(file, "gorm.io") ||
			strings.HasSuffix(file, "xdblog.go") || strings.HasSuffix(file, "xdb.go") ||
			strings.Contains(file, "pkg/xlog") || strings.Contains(file, `pkg\xlog`) ||
			strings.Contains(file, "runtime/") || strings.Contains(file, `runtime\`) {
			continue
		}
		return trimCallerPath(file, line)
	}
	return ""
}

// trimCallerPath 裁剪绝对路径前缀，保留如 "/repository/db/demo.go:125"
func trimCallerPath(file string, line int) string {
	norm := strings.ReplaceAll(file, "\\", "/")
	// 查找典型的业务目录前缀
	for _, marker := range []string{"/repository/", "/service/", "/handler/", "/internal/", "/cmd/"} {
		if idx := strings.Index(norm, marker); idx != -1 {
			return fmt.Sprintf("%s:%d", norm[idx:], line)
		}
	}
	// 兜底保留最后 3 级路径
	parts := strings.Split(norm, "/")
	if len(parts) > 3 {
		return fmt.Sprintf("/%s:%d", strings.Join(parts[len(parts)-3:], "/"), line)
	}
	return fmt.Sprintf("%s:%d", norm, line)
}

// initLog 构建走 xlog 的 GORM logger。
func initLog(_ logger.Writer, config logger.Config) logger.Interface {
	return &xdbLogger{
		Config: config,
	}
}

func (l *xdbLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

func (l *xdbLogger) Info(ctx context.Context, msg string, data ...any) {
	if l.LogLevel >= logger.Info {
		content := fmt.Sprintf(msg, data...)
		if caller := findCaller(); caller != "" {
			content = caller + " [info] " + content
		} else {
			content = "[info] " + content
		}
		xlog.Info(ctx, content)
	}
}

func (l *xdbLogger) Warn(ctx context.Context, msg string, data ...any) {
	if l.LogLevel >= logger.Warn {
		content := fmt.Sprintf(msg, data...)
		if caller := findCaller(); caller != "" {
			content = caller + " [warn] " + content
		} else {
			content = "[warn] " + content
		}
		xlog.Warn(ctx, content)
	}
}

func (l *xdbLogger) Error(ctx context.Context, msg string, data ...any) {
	if l.LogLevel >= logger.Error {
		content := fmt.Sprintf(msg, data...)
		if caller := findCaller(); caller != "" {
			content = caller + " [error] " + content
		} else {
			content = "[error] " + content
		}
		xlog.Error(ctx, content, nil)
	}
}

func (l *xdbLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {
	if l.LogLevel <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	duration := float64(elapsed.Nanoseconds()) / 1e6
	isNotFound := errors.Is(err, logger.ErrRecordNotFound)

	// 判断是否需要记录日志
	isErr := err != nil && l.LogLevel >= logger.Error && !(isNotFound && l.IgnoreRecordNotFoundError)
	isSlow := l.SlowThreshold != 0 && elapsed > l.SlowThreshold && l.LogLevel >= logger.Warn
	isInfo := l.LogLevel == logger.Info

	if !isErr && !isSlow && !isInfo {
		return
	}

	caller := findCaller()
	if caller != "" {
		caller = caller + " "
	}

	sql, rows := fc()
	rowsStr := "-"
	if rows != -1 {
		rowsStr = strconv.FormatInt(rows, 10)
	}

	durationParam := map[string]any{"duration": duration}

	switch {
	case isErr:
		msg := fmt.Sprintf("%s%v [%.3fms] [rows:%s] %s", caller, err, duration, rowsStr, sql)
		xlog.Error(ctx, msg, err, durationParam)
	case isSlow:
		msg := fmt.Sprintf("%sSLOW SQL >= %v [%.3fms] [rows:%s] %s", caller, l.SlowThreshold, duration, rowsStr, sql)
		xlog.Warn(ctx, msg, durationParam)
	case isInfo:
		msg := fmt.Sprintf("%s[%.3fms] [rows:%s] %s", caller, duration, rowsStr, sql)
		xlog.Info(ctx, msg, durationParam)
	}
}
