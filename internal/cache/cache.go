package cache

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"time"
)

const defaultTTL = 30 * 24 * time.Hour

type Cache struct {
	dir string
	ttl time.Duration
}

type entry struct {
	Data      []byte    `json:"data"`
	CreatedAt time.Time `json:"created_at"`
}

func New(dir string) *Cache {
	return &Cache{dir: dir, ttl: defaultTTL}
}

func (c *Cache) key(parts ...string) string {
	h := sha256.New()
	for _, p := range parts {
		h.Write([]byte(p))
		h.Write([]byte{0})
	}
	return hex.EncodeToString(h.Sum(nil))
}

func (c *Cache) Get(keys ...string) ([]byte, bool) {
	path := filepath.Join(c.dir, c.key(keys...)+".json")
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, false
	}

	var e entry
	if err := json.Unmarshal(data, &e); err != nil {
		return nil, false
	}

	if time.Since(e.CreatedAt) > c.ttl {
		_ = os.Remove(path)
		return nil, false
	}

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
