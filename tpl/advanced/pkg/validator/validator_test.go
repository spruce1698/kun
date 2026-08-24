package validator

import (
	"testing"
	"time"

	"github.com/gin-gonic/gin/binding"
	vali10 "github.com/go-playground/validator/v10"
)

type TestDateReq struct {
	Min string `form:"min" json:"min" binding:"required,afterNow"`
	Max string `form:"max" json:"max" binding:"required,beforeNow"`
}

func TestValidator(t *testing.T) {
	New("zh")

	validate, ok := binding.Validator.Engine().(*vali10.Validate)
	if !ok {
		t.Fatal("failed to get validate engine")
	}

	today := time.Now().Format("2006-01-02")
	tomorrow := time.Now().AddDate(0, 0, 1).Format("2006-01-02")
	yesterday := time.Now().AddDate(0, 0, -1).Format("2006-01-02")

	// 1. 测试合法数据
	validReq := TestDateReq{
		Min: tomorrow,
		Max: yesterday,
	}
	if err := validate.Struct(validReq); err != nil {
		t.Fatalf("expected valid, got err: %v", err)
	}

	// 2. 测试今天
	todayReq := TestDateReq{
		Min: today,
		Max: today,
	}
	if err := validate.Struct(todayReq); err != nil {
		t.Fatalf("expected today valid, got err: %v", err)
	}

	// 3. 测试非法数据 (Min 过去日期，Max 未来日期)
	invalidReq := TestDateReq{
		Min: yesterday,
		Max: tomorrow,
	}
	err := validate.Struct(invalidReq)
	if err == nil {
		t.Fatal("expected invalid, got nil")
	}

	if valErrs, ok := err.(Errors); ok {
		msgs := Message(valErrs)
		if len(msgs) == 0 {
			t.Fatal("expected translated messages, got empty")
		}
		t.Logf("translated errors: %v", msgs)
	} else {
		t.Fatalf("expected Errors type, got: %T", err)
	}
}
