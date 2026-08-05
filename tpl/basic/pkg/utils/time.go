package utils

import (
	"fmt"
	"strings"
	"time"
)

// 获取 Unix 时间戳
func UtcTime(tm time.Time) int64 {
	return tm.Unix() // - 8*60*60
}

// 当前时间向上取整点
func Hour(timestamp int64) int {
	//	formTime := time.Format("2006-01-02 15:04:05")
	tm := time.Unix(timestamp, 0)
	return tm.Hour()
}

// 获取offset天的现在时间:注意时区
func LastDayCurrentTime(timestamp int64, offset int) time.Time {
	tm := time.Unix(timestamp, 0)
	yesDay := tm.AddDate(0, 0, 1*offset)
	return yesDay
}

// 获取给定时间的星期
func TimeWeek(timestamp int64) int {
	tm := time.Unix(timestamp, 0)
	return int(tm.Weekday())
}

// 获取向上整时时间
func Hour0(timestamp int64) time.Time {
	tm := time.Unix(timestamp, 0)
	return time.Date(tm.Year(), tm.Month(), tm.Day(), tm.Hour(), 0, 0, 0, tm.Location())
}

// 获取给定日期的零点时间
func Day0(timestamp int64) time.Time {
	tm := time.Unix(timestamp, 0)
	return time.Date(tm.Year(), tm.Month(), tm.Day(), 0, 0, 0, 0, tm.Location())
}

// 获取offset 0点时间
func UtcDay0(now time.Time, timeZone *time.Location) int64 {
	if timeZone == nil {
		timeZone = now.Location()
	}
	tm := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, timeZone)
	return tm.Unix()
}

// 获取最近上个星期天的零点日期
func Week0(timestamp int64) time.Time {
	weekday := TimeWeek(timestamp)
	tm0 := Day0(timestamp)
	return tm0.AddDate(0, 0, -1*weekday)
}

// 获取最近上个星期天的零点日期
func UtcWeek0(timestamp int64) int64 {
	weekday := TimeWeek(timestamp)
	tm0 := Day0(timestamp)
	tm0 = tm0.AddDate(0, 0, -1*weekday)
	return tm0.Unix()
}

// 获取给定时间的当月1号零点时间
func Month0(timestamp int64) time.Time {
	tm0 := Day0(timestamp)
	month0 := tm0.Day() - 1
	tm0 = tm0.AddDate(0, 0, -1*month0) // 这个月1号
	return tm0
}

// 字符串转时间
func Str2Time(tStr, format string, timeZone *time.Location) time.Time {
	if len(format) == 0 {
		format = "2006-01-02 15:04:05"
	}
	if timeZone == nil {
		// chinaLocal, _ := time.LoadLocation("Local")
		timeZone = time.Local
	}

	ti, _ := time.ParseInLocation(format, tStr, timeZone)
	return ti
}

// 给定字符串时间转换成本地时间戳
func StringTime2Unix(timeStr string) int64 {
	return Str2Time(timeStr, "2006-01-02 15:04:05", time.Local).Unix()
}

// 整点执行操作
func TimerByHour(f func()) {
	for {
		now := time.Now()
		// 计算下一个整点
		next := now.Add(time.Hour * 1)
		next = time.Date(next.Year(), next.Month(), next.Day(), next.Hour(), 0, 0, 0, next.Location())
		t := time.NewTimer(next.Sub(now))
		<-t.C
		t.Stop()
		// 以下为定时执行的操作
		f()
	}
}

// 时间戳转换为time
func Unix2Time(timestamp int64) time.Time {
	return time.Unix(timestamp, 0)
}

// 获取本地时间
func LocalTime(tm time.Time) time.Time {
	local, _ := time.LoadLocation("Local")
	return tm.In(local)
	// return tm.Add(8 * 60 * 60 * time.Second)
}

// 获取系统时间的格式
func SysTimeLayout() string {
	t := time.Date(2006, 1, 2, 15, 4, 5, 0, time.UTC)
	strLayout := strings.Replace(t.String(), "+0000 UTC", "", -1)
	return strings.TrimSpace(strLayout)
}

// 格式化时间
func FormatTime(tm time.Time, forStr string) string {
	return tm.Format(forStr)
}

// 获取日期字符串
func DayStr(tm time.Time) string {
	return FormatTime(tm, "2006-01-02")
}

// 获取时间字符串
func TimeStr(tm time.Time) string {
	return FormatTime(tm, "2006-01-02 15:04:05")
}

// 时间戳转年月日
func Tamp2Str(timestamp int64) string {
	return FormatTime(time.Unix(timestamp, 0), "2006-01-02 15:04:05")
}

// 将持续时间格式化为人类可读的字符串
func FormatDuration(duration time.Duration) string {
	days := int(duration.Hours() / 24)
	hours := int(duration.Hours()) % 24
	minutes := int(duration.Minutes()) % 60
	seconds := int(duration.Seconds()) % 60
	return fmt.Sprintf("%d天 %02d小时 %02d分 %02d秒", days, hours, minutes, seconds)
}

// 将工作日添加到日期
func AddBusinessDays(startDate time.Time, daysToAdd int) time.Time {
	currentDate := startDate
	for i := 0; i < daysToAdd; {
		currentDate = currentDate.AddDate(0, 0, 1)
		if currentDate.Weekday() != time.Saturday && currentDate.Weekday() != time.Sunday {
			i++
		}
	}
	return currentDate
}

// 获取月初
func StartOfMonth(date time.Time) time.Time {
	return time.Date(date.Year(), date.Month(), 1, 0, 0, 0, 0, date.Location())
}

// 获取月底
func EndOfMonth(date time.Time) time.Time {
	firstDayOfNextMonth := StartOfMonth(date).AddDate(0, 1, 0)
	return firstDayOfNextMonth.Add(-time.Second)
}

// 该日期所在周的第一天
func StartOfDayOfWeek(date time.Time) time.Time {
	daysSinceSunday := int(date.Weekday())
	return date.AddDate(0, 0, -daysSinceSunday+1)
}

// 该日期所在周的最后一天。
func EndOfDayOfWeek(date time.Time) time.Time {
	daysUntilSaturday := 7 - int(date.Weekday())
	return date.AddDate(0, 0, daysUntilSaturday)
}

// 获取给定月份每周的开始日和结束日
func StartAndEndOfWeeksOfMonth(year, month int) []struct{ Start, End time.Time } {
	startOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	weeks := make([]struct{ Start, End time.Time }, 0)

	for current := startOfMonth; current.Month() == time.Month(month); current = current.AddDate(0, 0, 7) {
		startOfWeek := StartOfDayOfWeek(current)
		endOfWeek := EndOfDayOfWeek(current)

		if endOfWeek.Month() != time.Month(month) {
			endOfWeek = EndOfMonth(current)
		}
		weeks = append(weeks, struct{ Start, End time.Time }{startOfWeek, endOfWeek})
	}

	return weeks
}

// 获取从日期开始的一个月的周数
func WeekNumberInMonth(date time.Time) int {
	startOfMonth := StartOfMonth(date)
	_, week := date.ISOWeek()
	_, startWeek := startOfMonth.ISOWeek()
	return week - startWeek + 1
}

// 获取新年伊始和年底
func StartOfYear(date time.Time) time.Time {
	return time.Date(date.Year(), time.January, 1, 0, 0, 0, 0, date.Location())
}

// 获取年底
func EndOfYear(date time.Time) time.Time {
	startOfNextYear := StartOfYear(date).AddDate(1, 0, 0)
	return startOfNextYear.Add(-time.Second)
}

// 获取季度初数据
func StartOfQuarter(date time.Time) time.Time {
	// you can directly use 0, 1, 2, 3 quarter
	quarter := (int(date.Month()) - 1) / 3
	startMonth := time.Month(quarter*3 + 1)
	return time.Date(date.Year(), startMonth, 1, 0, 0, 0, 0, date.Location())
}

// 获取季度末
func EndOfQuarter(date time.Time) time.Time {
	startOfNextQuarter := StartOfQuarter(date).AddDate(0, 3, 0)
	return startOfNextQuarter.Add(-time.Second)
}

// 获取当前周范围
func CurrentWeekRange(timeZone string) (startOfWeek, endOfWeek time.Time) {
	loc, err := time.LoadLocation(timeZone)
	if err != nil {
		loc = time.Local
	}

	now := time.Now().In(loc)
	startOfWeek = StartOfDayOfWeek(now)
	endOfWeek = EndOfDayOfWeek(now)

	return startOfWeek, endOfWeek
}

// 获取给定月份的星期几的日期
func DatesForDayOfWeek(year, month int, day time.Weekday) []time.Time {
	var dates []time.Time

	firstDayOfMonth := time.Date(year, time.Month(month), 1, 0, 0, 0, 0, time.UTC)
	diff := int(day) - int(firstDayOfMonth.Weekday())
	if diff < 0 {
		diff += 7
	}

	firstDay := firstDayOfMonth.AddDate(0, 0, diff)
	for current := firstDay; current.Month() == time.Month(month); current = current.AddDate(0, 0, 7) {
		dates = append(dates, current)
	}

	return dates
}

// json marsh 重写
type Time struct {
	time.Time
}

func (t *Time) UnmarshalJSON(data []byte) (err error) {
	tmp := string(data)
	if tmp == `""` {
		return nil
	}
	// 去掉前后引号后,按"日期时间 -> 日期"顺序尝试解析,避免用长度判断的脆弱性
	raw := strings.Trim(tmp, `"`)
	if now, err1 := time.ParseInLocation("2006-01-02 15:04:05", raw, time.Local); err1 == nil {
		*t = Time{now}
		return nil
	}
	if now, err1 := time.ParseInLocation("2006-01-02", raw, time.Local); err1 == nil {
		*t = Time{now}
		return nil
	}
	return fmt.Errorf("invalid time format: %s", tmp)
}

func (t *Time) MarshalJSON() ([]byte, error) {
	var stamp = fmt.Sprintf(`"%s"`, t.Format("2006-01-02 15:04:05"))
	return []byte(stamp), nil
}

func (t *Time) String() string {
	return t.Format("2006-01-02 15:04:05")
}
