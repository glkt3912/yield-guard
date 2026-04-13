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

// get はキャッシュを取得する。TTL 切れの場合はエントリを削除して (nil, false) を返す。
// 呼び出し元がスライスを変更してもキャッシュが汚染されないようコピーを返す。
func (c *cache) get(key string) ([]domain.LandTransaction, bool) {
	// まず読み取りロックで存在・有効期限を確認
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		// TTL 切れ: 書き込みロックに昇格してエントリを削除
		c.mu.Lock()
		delete(c.entries, key)
		c.mu.Unlock()
		return nil, false
	}

	// キャッシュヒット: コピーを返して呼び出し元による変更からキャッシュを保護する
	copied := make([]domain.LandTransaction, len(entry.data))
	copy(copied, entry.data)
	return copied, true
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
