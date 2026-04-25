package mlit

import (
	"fmt"
	"sync"
	"time"

	"github.com/yield-guard/backend/internal/domain"
)

const cacheTTL = 24 * time.Hour

type genericEntry[E any] struct {
	data      []E
	expiresAt time.Time
}

type genericCache[E any] struct {
	mu      sync.RWMutex
	entries map[string]genericEntry[E]
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
		delete(c.entries, key)
		c.mu.Unlock()
		return nil, false
	}

	copied := make([]E, len(entry.data))
	copy(copied, entry.data)
	return copied, true
}

// set はキャッシュに保存する。
func (c *genericCache[E]) set(key string, data []E) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.entries[key] = genericEntry[E]{data: data, expiresAt: time.Now().Add(cacheTTL)}
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
	}
}

// cacheKey は LandPriceQuery からキャッシュキーを生成する。
func cacheKey(q LandPriceQuery) string {
	return fmt.Sprintf("%s:%s:%d:%d:%d:%d", q.Area, q.City, q.Year, q.Quarter, q.ToYear, q.ToQuarter)
}
