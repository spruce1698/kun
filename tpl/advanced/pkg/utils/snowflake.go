/**
 * @Author:
 * @Date: 2024-03-28 18:31
 * @Desc: 雪花算法 id
 */

package utils

import (
	"fmt"
	"strconv"
	"sync"
	"time"

	"github.com/sony/sonyflake"
)

var (
	sonyOnce    sync.Once
	sonyFlake   *sonyflake.Sonyflake
	sonyInitErr error
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

func (s *Snowflake) GenID() int64 {
	s.Lock()
	defer func() {
		s.Unlock()
	}()
	now := time.Now().UnixNano() / 1e6 // 转毫秒
	// 时钟回拨保护:当前时间小于上次记录的时间戳时,等待追平,避免生成重复/乱序 id。
	// 回拨幅度过大(超过 5ms)直接 panic,避免长时间阻塞消费协程。
	if now < s.timestamp {
		if s.timestamp-now > 5 {
			panic(fmt.Sprintf("snowflake clock moved backwards %dms, refusing to generate id", s.timestamp-now))
		}
		for now < s.timestamp {
			now = time.Now().UnixNano() / 1e6
		}
	}
	if s.timestamp == now {
		// 当同一时间戳（精度：毫秒）下多次生成id会增加序列号
		s.sequence = (s.sequence + 1) & sequenceMask
		if s.sequence == 0 {
			// 如果当前序列超出12bit长度，则需要等待下一毫秒
			// 下一毫秒将使用sequence:0
			for now <= s.timestamp {
				now = time.Now().UnixNano() / 1e6
			}
		}
	} else {
		// 不同时间戳（精度：毫秒）下直接使用序列号：0
		s.sequence = 0
	}
	t := now - epoch
	if t > timestampMax {
		panic(fmt.Sprintf("snowflake timestamp overflow: epoch must be between 0 and %d", timestampMax-1))
	}
	s.timestamp = now
	r := (t)<<timestampShift | (s.centerId << centerIdShift) | (s.workerId << workerIdShift) | (s.sequence)
	return r
}

func (s *Snowflake) GenIDString() string {
	return strconv.FormatInt(s.GenID(), 10)
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

// InitSonyFlake 初始化 SonySnowFlake 单例(仅需调用一次)
func InitSonyFlake(machineID uint16) {
	sonyOnce.Do(func() {
		st, err := time.ParseInLocation("2006-01-02", "2026-01-01", time.Local)
		if err != nil {
			sonyInitErr = err
			return
		}
		sonyFlake = sonyflake.NewSonyflake(sonyflake.Settings{
			StartTime: st,
			MachineID: func() (uint16, error) { return machineID, nil },
		})
		if sonyFlake == nil {
			sonyInitErr = fmt.Errorf("sonyflake: initialization failed")
		}
	})
	if sonyInitErr != nil {
		panic(sonyInitErr)
	}
}

// SonySnowFlake 生成一个唯一 ID(需先调用 InitSonyFlake 初始化)
func SonySnowFlake() uint64 {
	if sonyFlake == nil {
		panic("sonyflake not initialized, call InitSonyFlake first")
	}
	id, err := sonyFlake.NextID()
	if err != nil {
		panic(err)
	}
	return id
}
