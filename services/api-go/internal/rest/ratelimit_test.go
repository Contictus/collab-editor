package rest

import (
	"testing"
)

func TestRateLimiterWindow(t *testing.T) {
	l := NewRateLimiter()
	if ok, _ := l.Allow("k", 2, 60_000); !ok {
		t.Fatal("first denied")
	}
	if ok, _ := l.Allow("k", 2, 60_000); !ok {
		t.Fatal("second denied")
	}
	ok, retry := l.Allow("k", 2, 60_000)
	if ok || retry <= 0 {
		t.Fatalf("third = %v %d", ok, retry)
	}
	if ok, _ := l.Allow("other", 2, 60_000); !ok {
		t.Fatal("other key denied")
	}
	l.Reset()
	if ok, _ := l.Allow("k", 2, 60_000); !ok {
		t.Fatal("post-reset denied")
	}
}
