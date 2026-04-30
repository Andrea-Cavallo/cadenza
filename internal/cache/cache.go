package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/Andrea-Cavallo/cadenza/internal/metrics"
)

const defaultTTL = 30 * 24 * time.Hour

// Cache is a SHA256-keyed disk cache with configurable TTL for LLM responses.
type Cache struct {
	dir   string
	ttl   time.Duration
	stats struct {
		sync.Mutex
		hits   int
		misses int
	}
}

type entry struct {
	Data      []byte    `json:"data"`
	CreatedAt time.Time `json:"created_at"`
}

func New(ttlDays int, dir string) *Cache {
	ttl := time.Duration(ttlDays) * 24 * time.Hour
	if ttlDays <= 0 {
		ttl = defaultTTL
	}
	return &Cache{dir: dir, ttl: ttl}
}

func (c *Cache) key(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func (c *Cache) recordMiss() {
	c.stats.Lock()
	c.stats.misses++
	c.stats.Unlock()
	metrics.CacheMisses.Add(1)
}

func (c *Cache) Get(keys ...string) ([]byte, bool) {
	path := filepath.Join(c.dir, c.key(keys...)+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		c.recordMiss()
		return nil, false
	}

	var e entry
	if err := json.Unmarshal(data, &e); err != nil {
		c.recordMiss()
		return nil, false
	}

	if time.Since(e.CreatedAt) > c.ttl {
		if err := os.Remove(path); err != nil {
			slog.Warn("cache: failed to remove expired entry", "path", path, "error", err)
		}
		c.recordMiss()
		return nil, false
	}

	c.stats.Lock()
	c.stats.hits++
	c.stats.Unlock()
	metrics.CacheHits.Add(1)
	return e.Data, true
}

func (c *Cache) Set(data []byte, keys ...string) error {
	if err := os.MkdirAll(c.dir, 0o755); err != nil {
		return fmt.Errorf("create cache dir: %w", err)
	}

	e := entry{Data: data, CreatedAt: time.Now()}
	b, err := json.Marshal(e)
	if err != nil {
		return fmt.Errorf("marshal cache entry: %w", err)
	}

	path := filepath.Join(c.dir, c.key(keys...)+".json")
	return os.WriteFile(path, b, 0o644)
}

// Stats returns cache statistics for dev mode inspection.
// REFACTOR.md point 20
func (c *Cache) Stats() map[string]interface{} {
	c.stats.Lock()
	defer c.stats.Unlock()

	// Count cache files
	var keys int
	if entries, err := os.ReadDir(c.dir); err == nil {
		keys = len(entries)
	}

	return map[string]interface{}{
		"hits":   c.stats.hits,
		"misses": c.stats.misses,
		"keys":   keys,
	}
}
