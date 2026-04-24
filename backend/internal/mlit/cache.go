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

// landPriceGroup は土地価格系・市区町村・地価公示 のキャッシュグループ
type landPriceGroup struct {
	mu              sync.RWMutex
	entries         map[string]cacheEntry
	muniEntries     map[string]muniCacheEntry
	appraisalEntries map[string]appraisalCacheEntry
}

// tileGroup は駅別乗降客数・将来推計人口・立地適正化・大規模盛土・都市計画道路・
// 災害履歴・都市計画区域・液状化 のキャッシュグループ
type tileGroup struct {
	mu                          sync.RWMutex
	ridershipEntries            map[string]ridershipCacheEntry
	populationEntries           map[string]populationCacheEntry
	locationOptimizationEntries map[string]locationOptimizationCacheEntry
	embankmentEntries           map[string]embankmentCacheEntry
	urbanRoadEntries            map[string]urbanRoadCacheEntry
	disasterEntries             map[string]disasterCacheEntry
	urbanZoningEntries          map[string]urbanZoningCacheEntry
	liquefactionEntries         map[string]liquefactionCacheEntry
}

// hazardGroup は洪水・高潮・津波・土砂災害 のキャッシュグループ
type hazardGroup struct {
	mu                     sync.RWMutex
	floodHazardEntries     map[string]floodHazardCacheEntry
	stormHazardEntries     map[string]stormHazardCacheEntry
	tsunamiHazardEntries   map[string]tsunamiHazardCacheEntry
	landslideHazardEntries map[string]landslideHazardCacheEntry
}

// cache は TTL 付きインメモリキャッシュ
type cache struct {
	land   landPriceGroup
	tile   tileGroup
	hazard hazardGroup
}

func newCache() *cache {
	c := &cache{}
	c.land.entries = make(map[string]cacheEntry)
	c.land.muniEntries = make(map[string]muniCacheEntry)
	c.land.appraisalEntries = make(map[string]appraisalCacheEntry)

	c.tile.ridershipEntries = make(map[string]ridershipCacheEntry)
	c.tile.populationEntries = make(map[string]populationCacheEntry)
	c.tile.locationOptimizationEntries = make(map[string]locationOptimizationCacheEntry)
	c.tile.embankmentEntries = make(map[string]embankmentCacheEntry)
	c.tile.urbanRoadEntries = make(map[string]urbanRoadCacheEntry)
	c.tile.disasterEntries = make(map[string]disasterCacheEntry)
	c.tile.urbanZoningEntries = make(map[string]urbanZoningCacheEntry)
	c.tile.liquefactionEntries = make(map[string]liquefactionCacheEntry)

	c.hazard.floodHazardEntries = make(map[string]floodHazardCacheEntry)
	c.hazard.stormHazardEntries = make(map[string]stormHazardCacheEntry)
	c.hazard.tsunamiHazardEntries = make(map[string]tsunamiHazardCacheEntry)
	c.hazard.landslideHazardEntries = make(map[string]landslideHazardCacheEntry)
	return c
}

// get はキャッシュを取得する。TTL 切れの場合はエントリを削除して (nil, false) を返す。
// 呼び出し元がスライスを変更してもキャッシュが汚染されないようコピーを返す。
func (c *cache) get(key string) ([]domain.LandTransaction, bool) {
	c.land.mu.RLock()
	entry, ok := c.land.entries[key]
	c.land.mu.RUnlock()

	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		c.land.mu.Lock()
		delete(c.land.entries, key)
		c.land.mu.Unlock()
		return nil, false
	}

	copied := make([]domain.LandTransaction, len(entry.data))
	copy(copied, entry.data)
	return copied, true
}

// set はキャッシュに保存する。
func (c *cache) set(key string, data []domain.LandTransaction) {
	c.land.mu.Lock()
	defer c.land.mu.Unlock()

	c.land.entries[key] = cacheEntry{
		data:      data,
		expiresAt: time.Now().Add(cacheTTL),
	}
}

// muniCacheEntry は市区町村キャッシュの1エントリ
type muniCacheEntry struct {
	data      []Municipality
	expiresAt time.Time
}

// getMuni は市区町村キャッシュを取得する。TTL 切れの場合はエントリを削除して (nil, false) を返す。
func (c *cache) getMuni(area string) ([]Municipality, bool) {
	c.land.mu.RLock()
	entry, ok := c.land.muniEntries[area]
	c.land.mu.RUnlock()

	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		c.land.mu.Lock()
		delete(c.land.muniEntries, area)
		c.land.mu.Unlock()
		return nil, false
	}

	copied := make([]Municipality, len(entry.data))
	copy(copied, entry.data)
	return copied, true
}

// setMuni は市区町村キャッシュに保存する。
func (c *cache) setMuni(area string, data []Municipality) {
	c.land.mu.Lock()
	defer c.land.mu.Unlock()

	c.land.muniEntries[area] = muniCacheEntry{
		data:      data,
		expiresAt: time.Now().Add(cacheTTL),
	}
}

// ridershipCacheEntry は駅別乗降客数キャッシュの1エントリ
type ridershipCacheEntry struct {
	data      []StationRidership
	expiresAt time.Time
}

// getRidership は駅別乗降客数キャッシュを取得する。
func (c *cache) getRidership(key string) ([]StationRidership, bool) {
	c.tile.mu.RLock()
	entry, ok := c.tile.ridershipEntries[key]
	c.tile.mu.RUnlock()

	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		c.tile.mu.Lock()
		delete(c.tile.ridershipEntries, key)
		c.tile.mu.Unlock()
		return nil, false
	}

	copied := make([]StationRidership, len(entry.data))
	copy(copied, entry.data)
	return copied, true
}

// setRidership は駅別乗降客数キャッシュに保存する。
func (c *cache) setRidership(key string, data []StationRidership) {
	c.tile.mu.Lock()
	defer c.tile.mu.Unlock()

	c.tile.ridershipEntries[key] = ridershipCacheEntry{
		data:      data,
		expiresAt: time.Now().Add(cacheTTL),
	}
}

// populationCacheEntry は将来推計人口キャッシュの1エントリ
type populationCacheEntry struct {
	data      []domain.PopulationForecastItem
	expiresAt time.Time
}

// getPopulation は将来推計人口キャッシュを取得する。
func (c *cache) getPopulation(key string) ([]domain.PopulationForecastItem, bool) {
	c.tile.mu.RLock()
	entry, ok := c.tile.populationEntries[key]
	c.tile.mu.RUnlock()

	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		c.tile.mu.Lock()
		delete(c.tile.populationEntries, key)
		c.tile.mu.Unlock()
		return nil, false
	}

	copied := make([]domain.PopulationForecastItem, len(entry.data))
	copy(copied, entry.data)
	return copied, true
}

// setPopulation は将来推計人口キャッシュに保存する。
func (c *cache) setPopulation(key string, data []domain.PopulationForecastItem) {
	c.tile.mu.Lock()
	defer c.tile.mu.Unlock()

	c.tile.populationEntries[key] = populationCacheEntry{
		data:      data,
		expiresAt: time.Now().Add(cacheTTL),
	}
}

// appraisalCacheEntry は地価公示キャッシュの1エントリ
type appraisalCacheEntry struct {
	data      []domain.LandAppraisalItem
	expiresAt time.Time
}

// getAppraisals は地価公示キャッシュを取得する。
func (c *cache) getAppraisals(key string) ([]domain.LandAppraisalItem, bool) {
	c.land.mu.RLock()
	entry, ok := c.land.appraisalEntries[key]
	c.land.mu.RUnlock()

	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		c.land.mu.Lock()
		delete(c.land.appraisalEntries, key)
		c.land.mu.Unlock()
		return nil, false
	}

	copied := make([]domain.LandAppraisalItem, len(entry.data))
	copy(copied, entry.data)
	return copied, true
}

// setAppraisals は地価公示キャッシュに保存する。
func (c *cache) setAppraisals(key string, data []domain.LandAppraisalItem) {
	c.land.mu.Lock()
	defer c.land.mu.Unlock()

	c.land.appraisalEntries[key] = appraisalCacheEntry{
		data:      data,
		expiresAt: time.Now().Add(cacheTTL),
	}
}

// locationOptimizationCacheEntry は立地適正化計画キャッシュの1エントリ
type locationOptimizationCacheEntry struct {
	data      []domain.LocationOptimizationItem
	expiresAt time.Time
}

func (c *cache) getLocationOptimization(key string) ([]domain.LocationOptimizationItem, bool) {
	c.tile.mu.RLock()
	entry, ok := c.tile.locationOptimizationEntries[key]
	c.tile.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		c.tile.mu.Lock()
		delete(c.tile.locationOptimizationEntries, key)
		c.tile.mu.Unlock()
		return nil, false
	}
	copied := make([]domain.LocationOptimizationItem, len(entry.data))
	copy(copied, entry.data)
	return copied, true
}

func (c *cache) setLocationOptimization(key string, data []domain.LocationOptimizationItem) {
	c.tile.mu.Lock()
	defer c.tile.mu.Unlock()
	c.tile.locationOptimizationEntries[key] = locationOptimizationCacheEntry{data: data, expiresAt: time.Now().Add(cacheTTL)}
}

// embankmentCacheEntry は大規模盛土造成地キャッシュの1エントリ
type embankmentCacheEntry struct {
	data      []domain.EmbankmentItem
	expiresAt time.Time
}

func (c *cache) getEmbankment(key string) ([]domain.EmbankmentItem, bool) {
	c.tile.mu.RLock()
	entry, ok := c.tile.embankmentEntries[key]
	c.tile.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		c.tile.mu.Lock()
		delete(c.tile.embankmentEntries, key)
		c.tile.mu.Unlock()
		return nil, false
	}
	copied := make([]domain.EmbankmentItem, len(entry.data))
	copy(copied, entry.data)
	return copied, true
}

func (c *cache) setEmbankment(key string, data []domain.EmbankmentItem) {
	c.tile.mu.Lock()
	defer c.tile.mu.Unlock()
	c.tile.embankmentEntries[key] = embankmentCacheEntry{data: data, expiresAt: time.Now().Add(cacheTTL)}
}

// urbanRoadCacheEntry は都市計画道路キャッシュの1エントリ
type urbanRoadCacheEntry struct {
	data      []domain.UrbanRoadItem
	expiresAt time.Time
}

func (c *cache) getUrbanRoad(key string) ([]domain.UrbanRoadItem, bool) {
	c.tile.mu.RLock()
	entry, ok := c.tile.urbanRoadEntries[key]
	c.tile.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		c.tile.mu.Lock()
		delete(c.tile.urbanRoadEntries, key)
		c.tile.mu.Unlock()
		return nil, false
	}
	copied := make([]domain.UrbanRoadItem, len(entry.data))
	copy(copied, entry.data)
	return copied, true
}

func (c *cache) setUrbanRoad(key string, data []domain.UrbanRoadItem) {
	c.tile.mu.Lock()
	defer c.tile.mu.Unlock()
	c.tile.urbanRoadEntries[key] = urbanRoadCacheEntry{data: data, expiresAt: time.Now().Add(cacheTTL)}
}

// disasterCacheEntry は災害履歴キャッシュの1エントリ
type disasterCacheEntry struct {
	data      []domain.DisasterHistoryItem
	expiresAt time.Time
}

func (c *cache) getDisaster(key string) ([]domain.DisasterHistoryItem, bool) {
	c.tile.mu.RLock()
	entry, ok := c.tile.disasterEntries[key]
	c.tile.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		c.tile.mu.Lock()
		delete(c.tile.disasterEntries, key)
		c.tile.mu.Unlock()
		return nil, false
	}
	copied := make([]domain.DisasterHistoryItem, len(entry.data))
	copy(copied, entry.data)
	return copied, true
}

func (c *cache) setDisaster(key string, data []domain.DisasterHistoryItem) {
	c.tile.mu.Lock()
	defer c.tile.mu.Unlock()
	c.tile.disasterEntries[key] = disasterCacheEntry{data: data, expiresAt: time.Now().Add(cacheTTL)}
}

// urbanZoningCacheEntry は都市計画区域/区域区分キャッシュの1エントリ
type urbanZoningCacheEntry struct {
	data      []domain.UrbanZoningItem
	expiresAt time.Time
}

func (c *cache) getUrbanZoning(key string) ([]domain.UrbanZoningItem, bool) {
	c.tile.mu.RLock()
	entry, ok := c.tile.urbanZoningEntries[key]
	c.tile.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		c.tile.mu.Lock()
		delete(c.tile.urbanZoningEntries, key)
		c.tile.mu.Unlock()
		return nil, false
	}
	copied := make([]domain.UrbanZoningItem, len(entry.data))
	copy(copied, entry.data)
	return copied, true
}

func (c *cache) setUrbanZoning(key string, data []domain.UrbanZoningItem) {
	c.tile.mu.Lock()
	defer c.tile.mu.Unlock()
	c.tile.urbanZoningEntries[key] = urbanZoningCacheEntry{data: data, expiresAt: time.Now().Add(cacheTTL)}
}

// liquefactionCacheEntry は液状化発生傾向図キャッシュの1エントリ
type liquefactionCacheEntry struct {
	data      []domain.LiquefactionRiskItem
	expiresAt time.Time
}

func (c *cache) getLiquefaction(key string) ([]domain.LiquefactionRiskItem, bool) {
	c.tile.mu.RLock()
	entry, ok := c.tile.liquefactionEntries[key]
	c.tile.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		c.tile.mu.Lock()
		delete(c.tile.liquefactionEntries, key)
		c.tile.mu.Unlock()
		return nil, false
	}
	copied := make([]domain.LiquefactionRiskItem, len(entry.data))
	copy(copied, entry.data)
	return copied, true
}

func (c *cache) setLiquefaction(key string, data []domain.LiquefactionRiskItem) {
	c.tile.mu.Lock()
	defer c.tile.mu.Unlock()
	c.tile.liquefactionEntries[key] = liquefactionCacheEntry{data: data, expiresAt: time.Now().Add(cacheTTL)}
}

// floodHazardCacheEntry は洪水浸水想定区域キャッシュの1エントリ
type floodHazardCacheEntry struct {
	data      []domain.FloodHazardItem
	expiresAt time.Time
}

func (c *cache) getFloodHazard(key string) ([]domain.FloodHazardItem, bool) {
	c.hazard.mu.RLock()
	entry, ok := c.hazard.floodHazardEntries[key]
	c.hazard.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		c.hazard.mu.Lock()
		delete(c.hazard.floodHazardEntries, key)
		c.hazard.mu.Unlock()
		return nil, false
	}
	copied := make([]domain.FloodHazardItem, len(entry.data))
	copy(copied, entry.data)
	return copied, true
}

func (c *cache) setFloodHazard(key string, data []domain.FloodHazardItem) {
	c.hazard.mu.Lock()
	defer c.hazard.mu.Unlock()
	c.hazard.floodHazardEntries[key] = floodHazardCacheEntry{data: data, expiresAt: time.Now().Add(cacheTTL)}
}

// stormHazardCacheEntry は高潮浸水想定区域キャッシュの1エントリ
type stormHazardCacheEntry struct {
	data      []domain.StormHazardItem
	expiresAt time.Time
}

func (c *cache) getStormHazard(key string) ([]domain.StormHazardItem, bool) {
	c.hazard.mu.RLock()
	entry, ok := c.hazard.stormHazardEntries[key]
	c.hazard.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		c.hazard.mu.Lock()
		delete(c.hazard.stormHazardEntries, key)
		c.hazard.mu.Unlock()
		return nil, false
	}
	copied := make([]domain.StormHazardItem, len(entry.data))
	copy(copied, entry.data)
	return copied, true
}

func (c *cache) setStormHazard(key string, data []domain.StormHazardItem) {
	c.hazard.mu.Lock()
	defer c.hazard.mu.Unlock()
	c.hazard.stormHazardEntries[key] = stormHazardCacheEntry{data: data, expiresAt: time.Now().Add(cacheTTL)}
}

// tsunamiHazardCacheEntry は津波浸水想定キャッシュの1エントリ
type tsunamiHazardCacheEntry struct {
	data      []domain.TsunamiHazardItem
	expiresAt time.Time
}

func (c *cache) getTsunamiHazard(key string) ([]domain.TsunamiHazardItem, bool) {
	c.hazard.mu.RLock()
	entry, ok := c.hazard.tsunamiHazardEntries[key]
	c.hazard.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		c.hazard.mu.Lock()
		delete(c.hazard.tsunamiHazardEntries, key)
		c.hazard.mu.Unlock()
		return nil, false
	}
	copied := make([]domain.TsunamiHazardItem, len(entry.data))
	copy(copied, entry.data)
	return copied, true
}

func (c *cache) setTsunamiHazard(key string, data []domain.TsunamiHazardItem) {
	c.hazard.mu.Lock()
	defer c.hazard.mu.Unlock()
	c.hazard.tsunamiHazardEntries[key] = tsunamiHazardCacheEntry{data: data, expiresAt: time.Now().Add(cacheTTL)}
}

// landslideHazardCacheEntry は土砂災害警戒区域キャッシュの1エントリ
type landslideHazardCacheEntry struct {
	data      []domain.LandslideHazardItem
	expiresAt time.Time
}

func (c *cache) getLandslideHazard(key string) ([]domain.LandslideHazardItem, bool) {
	c.hazard.mu.RLock()
	entry, ok := c.hazard.landslideHazardEntries[key]
	c.hazard.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		c.hazard.mu.Lock()
		delete(c.hazard.landslideHazardEntries, key)
		c.hazard.mu.Unlock()
		return nil, false
	}
	copied := make([]domain.LandslideHazardItem, len(entry.data))
	copy(copied, entry.data)
	return copied, true
}

func (c *cache) setLandslideHazard(key string, data []domain.LandslideHazardItem) {
	c.hazard.mu.Lock()
	defer c.hazard.mu.Unlock()
	c.hazard.landslideHazardEntries[key] = landslideHazardCacheEntry{data: data, expiresAt: time.Now().Add(cacheTTL)}
}

// cacheKey は LandPriceQuery からキャッシュキーを生成する。
func cacheKey(q LandPriceQuery) string {
	return fmt.Sprintf("%s:%s:%d:%d:%d:%d", q.Area, q.City, q.Year, q.Quarter, q.ToYear, q.ToQuarter)
}
