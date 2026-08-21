package internal

import (
	"crypto/sha256"
	"testing"
	"time"
)

func newTestCache(ttl time.Duration) *dedupCache {
	return &dedupCache{
		entries: make(map[[sha256.Size]byte]time.Time),
		ttl:     ttl,
	}
}

func TestDedupCache_FirstSeenReturnsFalse(t *testing.T) {
	cache := newTestCache(50 * time.Millisecond)
	if cache.seen([]byte("hello")) {
		t.Fatal("expected first occurrence to return false")
	}
}

func TestDedupCache_SecondSeenReturnsTrue(t *testing.T) {
	cache := newTestCache(50 * time.Millisecond)
	cache.seen([]byte("hello"))
	if !cache.seen([]byte("hello")) {
		t.Fatal("expected duplicate within TTL to return true")
	}
}

func TestDedupCache_ExpiredEntryReturnsFalse(t *testing.T) {
	cache := newTestCache(20 * time.Millisecond)
	cache.seen([]byte("hello"))
	time.Sleep(30 * time.Millisecond)
	if cache.seen([]byte("hello")) {
		t.Fatal("expected entry to be expired and return false")
	}
}

func TestDedupCache_DistinctPacketsIndependent(t *testing.T) {
	cache := newTestCache(50 * time.Millisecond)
	cache.seen([]byte("packet-a"))
	if cache.seen([]byte("packet-b")) {
		t.Fatal("expected distinct packet to return false")
	}
}

func TestDedupCache_ExpiryDoesNotAffectOtherEntries(t *testing.T) {
	cache := newTestCache(30 * time.Millisecond)
	cache.seen([]byte("packet-a"))
	time.Sleep(40 * time.Millisecond)
	cache.seen([]byte("packet-b"))
	if cache.seen([]byte("packet-b")) != true {
		t.Fatal("expected packet-b (within TTL) to return true")
	}
	if cache.seen([]byte("packet-a")) != false {
		t.Fatal("expected packet-a (expired) to return false")
	}
}
