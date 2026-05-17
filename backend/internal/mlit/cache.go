package mlit

import (
	"fmt"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/yield-guard/backend/internal/domain"
)

const cacheTTL = 24 * time.Hour

// genericEntry はキャッシュの1エントリ。E は格納する要素の型。
type genericEntry[E any] struct {
	data      []E
	expiresAt time.Time
}

// genericCache は TTL・スレッドセーフ・コピーオンリードを提供する汎用インメモリキャッシュ。
type genericCache[E any] struct {
	mu      sync.RWMutex
	entries map[string]genericEntry[E]
	keys    []string           // insertion-order for LRU eviction
	group   singleflight.Group // deduplicates concurrent fetches
}

func newGenericCache[E any]() *genericCache[E] {
	return &genericCache[E]{entries: make(map[string]genericEntry[E])}
}

// get はキャッシュを取得する。TTL 切れの場合はエントリを削除して (nil, false) を返す。
// 呼び出し元がスライスを変更してもキャッシュが汚染されないようコピーを返す。
func (c *genericCache[E]) get(key string) ([]E, bool) {
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		c.mu.Lock()
		// RUnlock から Lock 取得までの間に set() で新しいエントリが書き込まれた場合は削除しない
		if current, exists := c.entries[key]; exists && time.Now().After(current.expiresAt) {
			delete(c.entries, key)
		}
		c.mu.Unlock()
		return nil, false
	}

	copied := make([]E, len(entry.data))
	copy(copied, entry.data)
	return copied, true
}

// set はキャッシュに保存する。最大1000エントリを超えた場合は最も古いエントリを削除する（LRU eviction）。
func (c *genericCache[E]) set(key string, data []E) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if _, exists := c.entries[key]; !exists {
		if len(c.keys) >= 1000 {
			oldest := c.keys[0]
			c.keys = c.keys[1:]
			delete(c.entries, oldest)
		}
		c.keys = append(c.keys, key)
	}
	c.entries[key] = genericEntry[E]{data: data, expiresAt: time.Now().Add(cacheTTL)}
}

// getOrFetch はキャッシュを確認し、ミスの場合は singleflight でフェッチを deduplicate する。
// 同一キーへの並行リクエストは1回だけ fetch を呼び出し、結果を共有する。
func (c *genericCache[E]) getOrFetch(key string, fetch func() ([]E, error)) ([]E, error) {
	if data, ok := c.get(key); ok {
		return data, nil
	}
	v, err, _ := c.group.Do(key, func() (any, error) {
		if data, ok := c.get(key); ok {
			return data, nil
		}
		data, err := fetch()
		if err != nil {
			return nil, err
		}
		c.set(key, data)
		return data, nil
	})
	if err != nil {
		return nil, err
	}
	return v.([]E), nil
}

// cache は TTL 付きインメモリキャッシュ。各フィールドが独立した mutex を持つ。
type cache struct {
	landPrices           *genericCache[domain.LandTransaction]
	municipalities       *genericCache[Municipality]
	appraisals           *genericCache[domain.LandAppraisalItem]
	ridership            *genericCache[StationRidership]
	population           *genericCache[domain.PopulationForecastItem]
	locationOptimization *genericCache[domain.LocationOptimizationItem]
	embankment           *genericCache[domain.EmbankmentItem]
	urbanRoad            *genericCache[domain.UrbanRoadItem]
	disaster             *genericCache[domain.DisasterHistoryItem]
	urbanZoning          *genericCache[domain.UrbanZoningItem]
	liquefaction         *genericCache[domain.LiquefactionRiskItem]
	floodHazard          *genericCache[domain.FloodHazardItem]
	stormHazard          *genericCache[domain.StormHazardItem]
	tsunamiHazard        *genericCache[domain.TsunamiHazardItem]
	landslideHazard      *genericCache[domain.LandslideHazardItem]
	rentStats            *genericCache[domain.RentStatsResult]
}

func newCache() *cache {
	return &cache{
		landPrices:           newGenericCache[domain.LandTransaction](),
		municipalities:       newGenericCache[Municipality](),
		appraisals:           newGenericCache[domain.LandAppraisalItem](),
		ridership:            newGenericCache[StationRidership](),
		population:           newGenericCache[domain.PopulationForecastItem](),
		locationOptimization: newGenericCache[domain.LocationOptimizationItem](),
		embankment:           newGenericCache[domain.EmbankmentItem](),
		urbanRoad:            newGenericCache[domain.UrbanRoadItem](),
		disaster:             newGenericCache[domain.DisasterHistoryItem](),
		urbanZoning:          newGenericCache[domain.UrbanZoningItem](),
		liquefaction:         newGenericCache[domain.LiquefactionRiskItem](),
		floodHazard:          newGenericCache[domain.FloodHazardItem](),
		stormHazard:          newGenericCache[domain.StormHazardItem](),
		tsunamiHazard:        newGenericCache[domain.TsunamiHazardItem](),
		landslideHazard:      newGenericCache[domain.LandslideHazardItem](),
		rentStats:            newGenericCache[domain.RentStatsResult](),
	}
}

// cacheKey は LandPriceQuery からキャッシュキーを生成する。
func cacheKey(q LandPriceQuery) string {
	return fmt.Sprintf("%s:%s:%d:%d:%d:%d", q.Area, q.City, q.Year, q.Quarter, q.ToYear, q.ToQuarter)
}
