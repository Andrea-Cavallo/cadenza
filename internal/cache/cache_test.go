package cache

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestNew(t *testing.T) {
	c := New(7, t.TempDir())
	if c.ttl != 7*24*time.Hour {
		t.Fatalf("expected 7d TTL, got %v", c.ttl)
	}
}

func TestNewDefaultTTL(t *testing.T) {
	c := New(0, t.TempDir())
	if c.ttl != defaultTTL {
		t.Fatalf("expected default TTL %v, got %v", defaultTTL, c.ttl)
	}
}

func TestSetAndGet(t *testing.T) {
	dir := t.TempDir()
	c := New(30, dir)

	data := []byte(`{"notes":["A","C","E"]}`)
	if err := c.Set(data, "provider", "bassline", "Am"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	got, ok := c.Get("provider", "bassline", "Am")
	if !ok {
		t.Fatal("expected cache hit, got miss")
	}
	if string(got) != string(data) {
		t.Fatalf("data mismatch: got %q", got)
	}
}

func TestGetMiss(t *testing.T) {
	dir := t.TempDir()
	c := New(30, dir)

	_, ok := c.Get("nonexistent", "key")
	if ok {
		t.Fatal("expected cache miss for non-existent key")
	}
}

func TestGetExpired(t *testing.T) {
	dir := t.TempDir()
	c := New(30, dir)

	data := []byte(`{"test":"expired"}`)
	if err := c.Set(data, "expire-test"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Manually set TTL to 0 to force expiration
	c.ttl = 0

	_, ok := c.Get("expire-test")
	if ok {
		t.Fatal("expected cache miss for expired entry")
	}

	// Verify the file was cleaned up
	path := filepath.Join(dir, c.key("expire-test")+".json")
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		t.Fatal("expired cache file should be removed")
	}
}

func TestGetCorruptedFile(t *testing.T) {
	dir := t.TempDir()
	c := New(30, dir)

	// Write a corrupted cache file
	path := filepath.Join(dir, c.key("corrupt")+".json")
	if err := os.WriteFile(path, []byte("not json"), 0o644); err != nil {
		t.Fatalf("write corrupted file: %v", err)
	}

	_, ok := c.Get("corrupt")
	if ok {
		t.Fatal("expected cache miss for corrupted entry")
	}
}

func TestStats(t *testing.T) {
	dir := t.TempDir()
	c := New(30, dir)

	_ = c.Set([]byte(`"hit"`), "key1")
	c.Get("key1")     // hit
	c.Get("nonexist") // miss

	stats := c.Stats()
	if stats["hits"].(int) != 1 {
		t.Fatalf("expected 1 hit, got %v", stats["hits"])
	}
	if stats["misses"].(int) != 1 {
		t.Fatalf("expected 1 miss, got %v", stats["misses"])
	}
	if stats["keys"].(int) != 1 {
		t.Fatalf("expected 1 key, got %v", stats["keys"])
	}
}

func TestKeyDeterministic(t *testing.T) {
	c := New(30, t.TempDir())
	k1 := c.key("a", "b", "c")
	k2 := c.key("a", "b", "c")
	if k1 != k2 {
		t.Fatalf("key should be deterministic: %q != %q", k1, k2)
	}

	k3 := c.key("a", "bc")
	if k1 == k3 {
		t.Fatal("different input parts should produce different keys")
	}
}
