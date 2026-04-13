package mlit

import (
	"fmt"
	"sync"
	"time"

	"github.com/yield-guard/backend/internal/domain"
)

const cacheTTL = 24 * time.Hour

// cacheEntry はキャッシュの1エントリ
type cacheEntry struct {
	data      []domain.LandTransaction
	expiresAt time.Time
}

// cache は TTL 付きインメモリキャッシュ
type cache struct {
	mu      sync.RWMutex
	entries map[string]cacheEntry
}

func newCache() *cache {
	return &cache{
		entries: make(map[string]cacheEntry),
	}
}

// get はキャッシュを取得する。TTL 切れの場合は (nil, false) を返す。
func (c *cache) get(key string) ([]domain.LandTransaction, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, ok := c.entries[key]
	if !ok || time.Now().After(entry.expiresAt) {
		return nil, false
	}
	return entry.data, true
}

// set はキャッシュに保存する。
func (c *cache) set(key string, data []domain.LandTransaction) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = cacheEntry{
		data:      data,
		expiresAt: time.Now().Add(cacheTTL),
	}
}

// cacheKey は LandPriceQuery からキャッシュキーを生成する。
func cacheKey(q LandPriceQuery) string {
	return fmt.Sprintf("%s:%s:%d:%d:%d:%d", q.Area, q.City, q.Year, q.Quarter, q.ToYear, q.ToQuarter)
}
