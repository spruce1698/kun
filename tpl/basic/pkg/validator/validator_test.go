package validator

import (
	"testing"
	"time"

	"github.com/gin-gonic/gin/binding"
	vali10 "github.com/go-playground/validator/v10"
)

type TestReq struct {
	FutureDate string `json:"future_date" binding:"afterNow"`
	PastDate   string `json:"past_date" binding:"beforeNow"`
	Phone      string `json:"phone" binding:"mobile"`
}

func TestValidator_BeforeNowAndAfterNow(t *testing.T) {
	New("zh")
	validate := binding.Validator.Engine().(*vali10.Validate)

	today := time.Now().Format("2006-01-02")
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	// 1. 验证有效用例
	valid := TestReq{
		FutureDate: tomorrow,
		PastDate:   yesterday,
		Phone:      "13800138000",
	}
	if err := validate.Struct(valid); err != nil {
		t.Fatalf("expected valid, got: %v", err)
	}

	// 2. 验证 afterNow 失败 (传昨天)
	invalidAfter := TestReq{
		FutureDate: yesterday,
		PastDate:   today,
		Phone:      "13800138000",
	}
	if err := validate.Struct(invalidAfter); err == nil {
		t.Fatal("expected afterNow to fail for yesterday")
	} else if valErrs, ok := err.(Errors); ok {
		msgs := Message(valErrs)
		t.Logf("translated error message for afterNow: %v", msgs)
		if len(msgs) == 0 {
			t.Fatal("expected translated error message")
		}
	}

	// 3. 验证 beforeNow 失败 (传明天)
	invalidBefore := TestReq{
		FutureDate: today,
		PastDate:   tomorrow,
		Phone:      "13800138000",
	}
	if err := validate.Struct(invalidBefore); err == nil {
		t.Fatal("expected beforeNow to fail for tomorrow")
	}
}
