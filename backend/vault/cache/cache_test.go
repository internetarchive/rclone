package cache

import "testing"

func TestCache(t *testing.T) {
	cache := New()
	var (
		v0 = "v0"
		v  any
	)
	cache.Set("key0", v0)
	v = cache.Get("key0")
	if v != v0 {
		t.Fatalf("cache: got %v, want %v", v, v0)
	}
	cache.Reset()
	v = cache.Get("key0")
	if v != nil {
		t.Fatalf("cache: reset failed")
	}
	cache.SetGroup("key0", "group0", v0)
	v = cache.Get("key0")
	if v != nil {
		t.Fatalf("cache: key leak")
	}
	v = cache.GetGroup("key0", "xxx")
	if v != nil {
		t.Fatalf("cache: key, group mismatch")
	}
	v = cache.GetGroup("key0", "group0")
	if v != v0 {
		t.Fatalf("cache: cannot get value out")
	}
}
