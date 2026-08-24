package xerror

import (
	"testing"
)

func TestBase36EncodeDecode(t *testing.T) {
	testValues := []uint64{
		0,
		1,
		35,
		36,
		100,
		1296,
		999999,
		1234567890,
	}

	for _, v := range testValues {
		encoded := base36Encode(v)
		decoded := base36Decode(encoded)
		if decoded != v {
			t.Errorf("base36 encode/decode mismatch: input %d -> encoded %s -> decoded %d", v, encoded, decoded)
		}
	}
}

func TestBase36Decode_NonASCIINoPanic(t *testing.T) {
	// 验证包含非 ASCII 字符或中文时不会发生 index out of range panic
	nonASCII := []string{
		"测试中文",
		"Hello世界123",
		"éàçüö",
		"Special!@#$",
		"12345678901234567890", // 超长
	}

	for _, s := range nonASCII {
		// 应安全执行不 panic
		_ = base36Decode(s)
	}
}
