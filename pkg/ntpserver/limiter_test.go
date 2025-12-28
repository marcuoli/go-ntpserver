package ntpserver

import (
	"testing"
	"time"
)

func TestTokenBucket_Allow(t *testing.T) {
	b := newTokenBucket(1, 2) // 1 token/sec, burst 2
	t0 := time.Unix(1000, 0)

	if !b.allow(t0) {
		t.Fatalf("expected first allow")
	}
	if !b.allow(t0) {
		t.Fatalf("expected second allow (burst)")
	}
	if b.allow(t0) {
		t.Fatalf("expected third to be denied")
	}

	// After one second, one token refills.
	if !b.allow(t0.Add(1 * time.Second)) {
		t.Fatalf("expected allow after refill")
	}
}

func TestLimiter_PerIP(t *testing.T) {
	l := newLimiter(1, 1)
	now := time.Unix(2000, 0)

	if !l.allow("1.1.1.1", now) {
		t.Fatalf("expected first allow")
	}
	if l.allow("1.1.1.1", now) {
		t.Fatalf("expected second deny for same IP")
	}
	// Different IP gets its own bucket.
	if !l.allow("2.2.2.2", now) {
		t.Fatalf("expected allow for other IP")
	}
	// Empty IP bypasses limiter.
	if !l.allow("", now) {
		t.Fatalf("expected allow for empty IP")
	}
}
