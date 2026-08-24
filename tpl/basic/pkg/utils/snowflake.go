/**
 * @Author:
 * @Date: 2024-03-28 18:31
 * @Desc: 雪花算法 id
 */

package utils

import (
	"errors"
	"fmt"
	"strconv"
	"sync"
	"time"
)

const (
	epoch          = int64(1735660800000)                       // 设置起始时间(时间戳/毫秒)：2025-01-01 00:00:00.000，有效期69年
	timestampBits  = uint(41)                                   // 时间戳占用位数
	centerIdBits   = uint(5)                                    // 数据中心id所占位数
	workerIdBits   = uint(5)                                    // 机器id所占位数
	sequenceBits   = uint(12)                                   // 序列所占的位数
	timestampMax   = int64(-1 ^ (-1 << timestampBits))          // 时间戳最大值
	centerIdMax    = int64(-1 ^ (-1 << centerIdBits))           // 支持的最大数据中心id数量
	workerIdMax    = int64(-1 ^ (-1 << workerIdBits))           // 支持的最大机器id数量
	sequenceMask   = int64(-1 ^ (-1 << sequenceBits))           // 支持的最大序列id数量
	workerIdShift  = sequenceBits                               // 机器id左移位数
	centerIdShift  = sequenceBits + workerIdBits                // 数据中心id左移位数
	timestampShift = sequenceBits + workerIdBits + centerIdBits // 时间戳左移位数
)

// 时钟回拨/时间戳溢出错误,调用方可据此降级(等待/用旧时间戳)。
var (
	ErrClockBackwards    = errors.New("snowflake clock moved backwards beyond threshold")
	ErrTimestampOverflow = errors.New("snowflake timestamp overflow")
)

type Snowflake struct {
	sync.Mutex
	timestamp int64
	workerId  int64 // 节点ID
	centerId  int64 // 数据中心ID
	sequence  int64
}

func NewSnowflake(centerID, workerID int64) (*Snowflake, error) {
	if centerID < 0 || centerID > centerIdMax {
		return nil, fmt.Errorf("centerID %d out of range [0, %d]", centerID, centerIdMax)
	}
	if workerID < 0 || workerID > workerIdMax {
		return nil, fmt.Errorf("workerID %d out of range [0, %d]", workerID, workerIdMax)
	}
	return &Snowflake{
		timestamp: 0,
		centerId:  centerID,
		workerId:  workerID,
		sequence:  0,
	}, nil
}

// GenID 生成唯一 ID。时钟回拨超过 5ms 或时间戳溢出时返回 error,由调用方降级,
// 不再 panic 以免在业务协程里崩整个进程。
func (s *Snowflake) GenID() (int64, error) {
	s.Lock()
	defer s.Unlock()
	now := time.Now().UnixMilli()
	if now < epoch {
		return 0, fmt.Errorf("system clock is before epoch %d", epoch)
	}
	if now < s.timestamp {
		if s.timestamp-now > 5 {
			return 0, fmt.Errorf("%w: %dms", ErrClockBackwards, s.timestamp-now)
		}
		for now < s.timestamp {
			time.Sleep(time.Duration(s.timestamp-now) * time.Millisecond)
			now = time.Now().UnixMilli()
		}
	}
	if s.timestamp == now {
		s.sequence = (s.sequence + 1) & sequenceMask
		if s.sequence == 0 {
			for now <= s.timestamp {
				time.Sleep(100 * time.Microsecond)
				now = time.Now().UnixMilli()
			}
		}
	} else {
		s.sequence = 0
	}
	t := now - epoch
	if t > timestampMax {
		return 0, fmt.Errorf("%w: epoch must be between 0 and %d", ErrTimestampOverflow, timestampMax-1)
	}
	s.timestamp = now
	r := (t)<<timestampShift | (s.centerId << centerIdShift) | (s.workerId << workerIdShift) | (s.sequence)
	return r, nil
}

func (s *Snowflake) GenIDString() (string, error) {
	id, err := s.GenID()
	if err != nil {
		return "", err
	}
	return strconv.FormatInt(id, 10), nil
}

// 获取数据中心ID和机器ID
func GetDeviceID(sid int64) (centerID, workerID int64) {
	centerID = (sid >> centerIdShift) & centerIdMax
	workerID = (sid >> workerIdShift) & workerIdMax
	return
}

// 获取创建ID时的时间戳(毫秒)
func GetTimestamp(sid int64) (timestamp int64) {
	timestamp = ((sid >> timestampShift) & timestampMax) + epoch
	return
}

// 获取创建ID时的时间字符串(精度：秒)
func GetDateTime(sid int64) (t string) {
	t = time.UnixMilli(GetTimestamp(sid)).Format("2006-01-02 15:04:05")
	return
}

// 获取时间戳已使用的占比：范围（0.0 - 1.0）
func GetTimeStatus() (status float64) {
	status = float64(time.Now().UnixMilli()-epoch) / float64(timestampMax)
	return
}
