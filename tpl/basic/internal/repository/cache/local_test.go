package cache

import (
	"fmt"
	"testing"
	"time"
)

// 本地缓存必须有容量上限,否则遍历 id 刷接口可远程打爆内存。
func TestLocalCache_BoundedSize(t *testing.T) {
	const limit = 100
	c := NewLocalCacheWithLimit(5*time.Minute, 10*time.Minute, limit)

	// 写入远超上限的条目
	for i := 0; i < limit*50; i++ {
		c.Set(fmt.Sprintf("k%d", i), i, 5*time.Minute)
	}

	got := c.ItemCount()
	if got > limit {
		t.Fatalf("cache grew past its limit: %d > %d", got, limit)
	}
	if got == 0 {
		t.Fatal("cache evicted everything; should retain recent entries")
	}
	t.Logf("bounded at %d entries after %d inserts (limit %d)", got, limit*50, limit)
}

// 覆盖写同一个 key 不应触发淘汰,也不应让条目数增长。
func TestLocalCache_OverwriteDoesNotGrow(t *testing.T) {
	c := NewLocalCacheWithLimit(time.Minute, time.Minute, 10)
	for i := 0; i < 100; i++ {
		c.Set("same", i, time.Minute)
	}
	if n := c.ItemCount(); n != 1 {
		t.Fatalf("expected 1 entry after repeated overwrite, got %d", n)
	}
	v, ok := c.Get("same")
	if !ok || v.(int) != 99 {
		t.Fatalf("expected latest value 99, got %v (ok=%v)", v, ok)
	}
}

func TestLocalCache_GetSetDelete(t *testing.T) {
	c := NewLocalCache(time.Minute, time.Minute)
	c.Set("a", "v", time.Minute)
	if v, ok := c.Get("a"); !ok || v.(string) != "v" {
		t.Fatalf("get failed: %v %v", v, ok)
	}
	c.Delete("a")
	if _, ok := c.Get("a"); ok {
		t.Fatal("delete failed")
	}
}
