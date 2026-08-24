package utils

import (
	"encoding/json"
	"testing"
	"time"
)

func TestStartAndEndOfDayOfWeek(t *testing.T) {
	// 测试包含周日 (2026-08-23 为周日, 2026-08-17 为周一)
	loc := time.Local
	sunday := time.Date(2026, 8, 23, 15, 30, 45, 0, loc)

	start := StartOfDayOfWeek(sunday)
	if start.Weekday() != time.Monday {
		t.Fatalf("expected StartOfDayOfWeek on Sunday to be Monday, got %v", start.Weekday())
	}
	if start.Day() != 17 || start.Hour() != 0 || start.Minute() != 0 || start.Second() != 0 {
		t.Fatalf("expected Monday 2026-08-17 00:00:00, got %v", start)
	}

	end := EndOfDayOfWeek(sunday)
	if end.Weekday() != time.Sunday {
		t.Fatalf("expected EndOfDayOfWeek on Sunday to be Sunday, got %v", end.Weekday())
	}
	if end.Day() != 23 || end.Hour() != 23 || end.Minute() != 59 || end.Second() != 59 {
		t.Fatalf("expected Sunday 2026-08-23 23:59:59, got %v", end)
	}

	// 测试周三 (2026-08-19)
	wednesday := time.Date(2026, 8, 19, 10, 0, 0, 0, loc)
	startWed := StartOfDayOfWeek(wednesday)
	if startWed.Day() != 17 || startWed.Weekday() != time.Monday {
		t.Fatalf("expected Monday 2026-08-17, got %v", startWed)
	}
	endWed := EndOfDayOfWeek(wednesday)
	if endWed.Day() != 23 || endWed.Weekday() != time.Sunday {
		t.Fatalf("expected Sunday 2026-08-23, got %v", endWed)
	}
}

func TestTime_JSONAndSQL(t *testing.T) {
	type Model struct {
		CreatedAt Time `json:"created_at"`
	}

	now := time.Date(2026, 8, 21, 15, 30, 0, 0, time.Local)
	m := Model{CreatedAt: Time{now}}

	data, err := json.Marshal(m)
	if err != nil {
		t.Fatalf("json.Marshal failed: %v", err)
	}
	expectedJSON := `{"created_at":"2026-08-21 15:30:00"}`
	if string(data) != expectedJSON {
		t.Fatalf("expected %s, got %s", expectedJSON, string(data))
	}

	var m2 Model
	if err := json.Unmarshal(data, &m2); err != nil {
		t.Fatalf("json.Unmarshal failed: %v", err)
	}
	if m2.CreatedAt.Format("2006-01-02 15:04:05") != "2026-08-21 15:30:00" {
		t.Fatalf("unmarshal mismatch: %v", m2.CreatedAt)
	}

	// 测试 Value & Scan
	val, err := m.CreatedAt.Value()
	if err != nil || val == nil {
		t.Fatalf("Value failed: %v, val: %v", err, val)
	}

	var scanned Time
	if err := scanned.Scan("2026-08-21 15:30:00"); err != nil {
		t.Fatalf("Scan string failed: %v", err)
	}
	if scanned.Format("2006-01-02 15:04:05") != "2026-08-21 15:30:00" {
		t.Fatalf("Scan mismatch: %v", scanned)
	}
}
