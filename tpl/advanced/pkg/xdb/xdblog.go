/**
 * @Author:
 * @Date: 2024-03-28 15:06
 * @Desc: 数据库日志
 */

package xdb

import (
	"context"
	"fmt"
	"time"

	"advanced/pkg/xlog"

	"errors"

	"gorm.io/gorm/logger"
	"gorm.io/gorm/utils"
)

type xdbLogger struct {
	logger.Writer
	logger.Config
	infoStr, warnStr, errStr, traceStr, traceErrStr, traceWarnStr string
}

func initLog(writer logger.Writer, config logger.Config) logger.Interface {
	var (
		infoStr      = "%s [info] "
		warnStr      = "%s [warn] "
		errStr       = "%s [error] "
		traceStr     = "%s [%.3fms] [rows:%v] %s"
		traceWarnStr = "%s %s [%.3fms] [rows:%v] %s"
		traceErrStr  = "%s %s [%.3fms] [rows:%v] %s"
	)

	return &xdbLogger{
		Writer:       writer,
		Config:       config,
		infoStr:      infoStr,
		warnStr:      warnStr,
		errStr:       errStr,
		traceStr:     traceStr,
		traceWarnStr: traceWarnStr,
		traceErrStr:  traceErrStr,
	}
}

func (l *xdbLogger) LogMode(level logger.LogLevel) logger.Interface {
	newLogger := *l
	newLogger.LogLevel = level
	return &newLogger
}

func (l *xdbLogger) Info(ctx context.Context, msg string, data ...any) {
	if l.LogLevel >= logger.Info {
		xlog.Info(ctx, fmt.Sprintf(l.infoStr+msg, append([]any{utils.FileWithLineNum()}, data...)...))
	}
}

func (l *xdbLogger) Warn(ctx context.Context, msg string, data ...any) {
	if l.LogLevel >= logger.Warn {
		xlog.Warn(ctx, fmt.Sprintf(l.warnStr+msg, append([]any{utils.FileWithLineNum()}, data...)...))
	}
}

func (l *xdbLogger) Error(ctx context.Context, msg string, data ...any) {
	if l.LogLevel >= logger.Error {
		xlog.Error(ctx, fmt.Sprintf(l.errStr+msg, append([]any{utils.FileWithLineNum()}, data...)...), nil)
	}
}

func (l *xdbLogger) Trace(ctx context.Context, begin time.Time, fc func() (string, int64), err error) {

	if l.LogLevel <= logger.Silent {
		return
	}

	elapsed := time.Since(begin)
	duration := float64(elapsed.Nanoseconds()) / 1e6
	// IgnoreRecordNotFoundError 在 xdb.New 中恒为 true,因此 gorm.ErrRecordNotFound
	// 不会被当作 Error 记录(避免查询无结果刷错误日志);其余 err 正常按 Error 记录。
	switch {
	case err != nil && l.LogLevel >= logger.Error && (!errors.Is(err, logger.ErrRecordNotFound) || !l.IgnoreRecordNotFoundError):
		sql, rows := fc()
		if rows == -1 {
			xlog.Error(ctx, fmt.Sprintf(l.traceErrStr, utils.FileWithLineNum(), err, duration, "-", sql), err, map[string]any{"duration": duration})
		} else {
			xlog.Error(ctx, fmt.Sprintf(l.traceErrStr, utils.FileWithLineNum(), err, duration, rows, sql), err, map[string]any{"duration": duration})
		}
	case l.SlowThreshold != 0 && elapsed > l.SlowThreshold && l.LogLevel >= logger.Warn:
		sql, rows := fc()
		slowLog := fmt.Sprintf("SLOW SQL >= %v", l.SlowThreshold)
		if rows == -1 {
			xlog.Warn(ctx, fmt.Sprintf(l.traceWarnStr, utils.FileWithLineNum(), slowLog, duration, "-", sql), map[string]any{"duration": duration})
		} else {
			xlog.Warn(ctx, fmt.Sprintf(l.traceWarnStr, utils.FileWithLineNum(), slowLog, duration, rows, sql), map[string]any{"duration": duration})
		}
	case l.LogLevel == logger.Info:
		sql, rows := fc()
		if rows == -1 {
			xlog.Info(ctx, fmt.Sprintf(l.traceStr, utils.FileWithLineNum(), duration, "-", sql), map[string]any{"duration": duration})
		} else {
			xlog.Info(ctx, fmt.Sprintf(l.traceStr, utils.FileWithLineNum(), duration, rows, sql), map[string]any{"duration": duration})
		}
	}

}
