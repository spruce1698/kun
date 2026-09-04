package token

import (
	"context"
	"testing"
	"time"

	"advanced/pkg/xconfig"
	"advanced/pkg/xredis"
)

func BenchmarkTokenGenAndParse(b *testing.B) {
	// mock redis is not strictly needed for Gen, but for Parse blacklist check:
	conf := &xconfig.Conf{
		Token: xconfig.TokenConf{
			Secret:        "123456789012345678901234567890123456789012345678",
			RefreshSecret: "123456789012345678901234567890123456789012345678",
		},
		Redis: xconfig.RedisConf{
			Source: []string{"127.0.0.1:6379"},
		},
	}
	rdb, _ := xredis.New(conf)
	j, _ := NewJwt(conf, rdb)

	jwtTok, err := j.Gen(123456, 1)
	if err != nil {
		b.Fatalf("Gen failed: %v", err)
	}

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		// Test Gen
		_, _ = j.Gen(123456, 1)
	}
	_ = jwtTok
	_ = context.Background()
	_ = time.Now()
}
