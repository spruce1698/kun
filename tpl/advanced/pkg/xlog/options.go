/**
 * @Author:
 * @Date: 2024-03-28 14:31
 * @Desc: 自定义日志选项
 */

package xlog

import (
	"context"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/segmentio/kafka-go"
	"go.uber.org/zap/zapcore"
	"gopkg.in/natefinch/lumberjack.v2"
)

// Option custom setup config
type Option func(*option)

type option struct {
	level         zapcore.Level
	fileHook      io.Writer
	kafkaHook     io.Writer // 添加 Kafka hook
	enableConsole bool
}

func newOption(opts ...Option) *option {
	opt := &option{
		level:         zapcore.DebugLevel,
		enableConsole: false,
	}
	for _, o := range opts {
		o(opt)
	}
	return opt
}

// Only greater than 'level' will output
func withDebugLevel() Option {
	return func(opt *option) {
		opt.level = zapcore.DebugLevel
	}
}

// Only greater than 'level' will output
func withInfoLevel() Option {
	return func(opt *option) {
		opt.level = zapcore.InfoLevel
	}
}

// Only greater than 'level' will output
func withWarnLevel() Option {
	return func(opt *option) {
		opt.level = zapcore.WarnLevel
	}
}

// Only greater than 'level' will output
func withErrorLevel() Option {
	return func(opt *option) {
		opt.level = zapcore.ErrorLevel
	}
}

// Write a log to some file
func withFilePath(filePath string) Option {
	dir := filepath.Dir(filePath)
	if err := os.MkdirAll(dir, 0755); err != nil {
		panic(err)
	}
	return func(opt *option) {
		opt.fileHook = &lumberjack.Logger{ // concurrent-safety
			Filename:   filePath, // 文件路径
			MaxSize:    128,      // 单个文件最大尺寸，默认单位 M，超过则切割
			MaxBackups: 100,      // 最多保留 100 个备份
			MaxAge:     30,       // 最大时间，默认单位 day
			LocalTime:  true,     // 使用本地时间
			Compress:   true,     // 是否压缩 disabled by default
		}
	}
}

// Write a log to os.Stdout or os.Stderr
func withEnableConsole() Option {
	return func(opt *option) {
		opt.enableConsole = true
	}
}

// Write a log to Kafka
func withKafka(topic string, brokers []string) Option {
	return func(opt *option) {
		if len(brokers) == 0 || topic == "" {
			return
		}
		writer := &KafkaWriter{
			writer: &kafka.Writer{
				Addr:         kafka.TCP(brokers...),
				Topic:        topic,
				Async:        true,
				BatchTimeout: time.Millisecond * 100,
				BatchSize:    100,
				RequiredAcks: kafka.RequireOne,
			},
		}
		opt.kafkaHook = writer
	}
}

// Kafka写入器
type KafkaWriter struct {
	writer *kafka.Writer
}

// 实现io.Writer接口
func (k *KafkaWriter) Write(p []byte) (n int, err error) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second*3)
	defer cancel()

	msg := kafka.Message{
		Value: append([]byte{}, p...), // 复制数据以避免并发问题
		Time:  time.Now(),
	}

	err = k.writer.WriteMessages(ctx, msg)
	if err != nil {
		return 0, fmt.Errorf("failed to write message to kafka: %w", err)
	}

	return len(p), nil
}

// 关闭写入器
func (k *KafkaWriter) Close() error {
	if k.writer != nil {
		return k.writer.Close()
	}
	return nil
}
