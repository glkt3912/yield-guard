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
	mu                         sync.RWMutex
	entries                    map[string]cacheEntry
	muniEntries                map[string]muniCacheEntry
	ridershipEntries           map[string]ridershipCacheEntry
	populationEntries          map[string]populationCacheEntry
	appraisalEntries           map[string]appraisalCacheEntry
	locationOptimizationEntries map[string]locationOptimizationCacheEntry
	embankmentEntries          map[string]embankmentCacheEntry
	urbanRoadEntries           map[string]urbanRoadCacheEntry
	disasterEntries            map[string]disasterCacheEntry
	urbanZoningEntries         map[string]urbanZoningCacheEntry
	liquefactionEntries        map[string]liquefactionCacheEntry
	floodHazardEntries         map[string]floodHazardCacheEntry
	stormHazardEntries         map[string]stormHazardCacheEntry
	tsunamiHazardEntries       map[string]tsunamiHazardCacheEntry
	landslideHazardEntries     map[string]landslideHazardCacheEntry
}

func newCache() *cache {
	return &cache{
		entries:                    make(map[string]cacheEntry),
		muniEntries:                make(map[string]muniCacheEntry),
		ridershipEntries:           make(map[string]ridershipCacheEntry),
		populationEntries:          make(map[string]populationCacheEntry),
		appraisalEntries:           make(map[string]appraisalCacheEntry),
		locationOptimizationEntries: make(map[string]locationOptimizationCacheEntry),
		embankmentEntries:          make(map[string]embankmentCacheEntry),
		urbanRoadEntries:           make(map[string]urbanRoadCacheEntry),
		disasterEntries:            make(map[string]disasterCacheEntry),
		urbanZoningEntries:         make(map[string]urbanZoningCacheEntry),
		liquefactionEntries:        make(map[string]liquefactionCacheEntry),
		floodHazardEntries:         make(map[string]floodHazardCacheEntry),
		stormHazardEntries:         make(map[string]stormHazardCacheEntry),
		tsunamiHazardEntries:       make(map[string]tsunamiHazardCacheEntry),
		landslideHazardEntries:     make(map[string]landslideHazardCacheEntry),
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

// muniCacheEntry は市区町村キャッシュの1エントリ
type muniCacheEntry struct {
	data      []Municipality
	expiresAt time.Time
}

// getMuni は市区町村キャッシュを取得する。TTL 切れの場合はエントリを削除して (nil, false) を返す。
func (c *cache) getMuni(area string) ([]Municipality, bool) {
	c.mu.RLock()
	entry, ok := c.muniEntries[area]
	c.mu.RUnlock()

	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		c.mu.Lock()
		delete(c.muniEntries, area)
		c.mu.Unlock()
		return nil, false
	}

	copied := make([]Municipality, len(entry.data))
	copy(copied, entry.data)
	return copied, true
}

// setMuni は市区町村キャッシュに保存する。
func (c *cache) setMuni(area string, data []Municipality) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.muniEntries[area] = muniCacheEntry{
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
	c.mu.RLock()
	entry, ok := c.ridershipEntries[key]
	c.mu.RUnlock()

	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		c.mu.Lock()
		delete(c.ridershipEntries, key)
		c.mu.Unlock()
		return nil, false
	}

	copied := make([]StationRidership, len(entry.data))
	copy(copied, entry.data)
	return copied, true
}

// setRidership は駅別乗降客数キャッシュに保存する。
func (c *cache) setRidership(key string, data []StationRidership) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.ridershipEntries[key] = ridershipCacheEntry{
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
	c.mu.RLock()
	entry, ok := c.populationEntries[key]
	c.mu.RUnlock()

	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		c.mu.Lock()
		delete(c.populationEntries, key)
		c.mu.Unlock()
		return nil, false
	}

	copied := make([]domain.PopulationForecastItem, len(entry.data))
	copy(copied, entry.data)
	return copied, true
}

// setPopulation は将来推計人口キャッシュに保存する。
func (c *cache) setPopulation(key string, data []domain.PopulationForecastItem) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.populationEntries[key] = populationCacheEntry{
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
	c.mu.RLock()
	entry, ok := c.appraisalEntries[key]
	c.mu.RUnlock()

	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		c.mu.Lock()
		delete(c.appraisalEntries, key)
		c.mu.Unlock()
		return nil, false
	}

	copied := make([]domain.LandAppraisalItem, len(entry.data))
	copy(copied, entry.data)
	return copied, true
}

// setAppraisals は地価公示キャッシュに保存する。
func (c *cache) setAppraisals(key string, data []domain.LandAppraisalItem) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.appraisalEntries[key] = appraisalCacheEntry{
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
	c.mu.RLock()
	entry, ok := c.locationOptimizationEntries[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		c.mu.Lock()
		delete(c.locationOptimizationEntries, key)
		c.mu.Unlock()
		return nil, false
	}
	copied := make([]domain.LocationOptimizationItem, len(entry.data))
	copy(copied, entry.data)
	return copied, true
}

func (c *cache) setLocationOptimization(key string, data []domain.LocationOptimizationItem) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.locationOptimizationEntries[key] = locationOptimizationCacheEntry{data: data, expiresAt: time.Now().Add(cacheTTL)}
}

// embankmentCacheEntry は大規模盛土造成地キャッシュの1エントリ
type embankmentCacheEntry struct {
	data      []domain.EmbankmentItem
	expiresAt time.Time
}

func (c *cache) getEmbankment(key string) ([]domain.EmbankmentItem, bool) {
	c.mu.RLock()
	entry, ok := c.embankmentEntries[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		c.mu.Lock()
		delete(c.embankmentEntries, key)
		c.mu.Unlock()
		return nil, false
	}
	copied := make([]domain.EmbankmentItem, len(entry.data))
	copy(copied, entry.data)
	return copied, true
}

func (c *cache) setEmbankment(key string, data []domain.EmbankmentItem) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.embankmentEntries[key] = embankmentCacheEntry{data: data, expiresAt: time.Now().Add(cacheTTL)}
}

// urbanRoadCacheEntry は都市計画道路キャッシュの1エントリ
type urbanRoadCacheEntry struct {
	data      []domain.UrbanRoadItem
	expiresAt time.Time
}

func (c *cache) getUrbanRoad(key string) ([]domain.UrbanRoadItem, bool) {
	c.mu.RLock()
	entry, ok := c.urbanRoadEntries[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		c.mu.Lock()
		delete(c.urbanRoadEntries, key)
		c.mu.Unlock()
		return nil, false
	}
	copied := make([]domain.UrbanRoadItem, len(entry.data))
	copy(copied, entry.data)
	return copied, true
}

func (c *cache) setUrbanRoad(key string, data []domain.UrbanRoadItem) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.urbanRoadEntries[key] = urbanRoadCacheEntry{data: data, expiresAt: time.Now().Add(cacheTTL)}
}

// disasterCacheEntry は災害履歴キャッシュの1エントリ
type disasterCacheEntry struct {
	data      []domain.DisasterHistoryItem
	expiresAt time.Time
}

func (c *cache) getDisaster(key string) ([]domain.DisasterHistoryItem, bool) {
	c.mu.RLock()
	entry, ok := c.disasterEntries[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		c.mu.Lock()
		delete(c.disasterEntries, key)
		c.mu.Unlock()
		return nil, false
	}
	copied := make([]domain.DisasterHistoryItem, len(entry.data))
	copy(copied, entry.data)
	return copied, true
}

func (c *cache) setDisaster(key string, data []domain.DisasterHistoryItem) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.disasterEntries[key] = disasterCacheEntry{data: data, expiresAt: time.Now().Add(cacheTTL)}
}

// urbanZoningCacheEntry は都市計画区域/区域区分キャッシュの1エントリ
type urbanZoningCacheEntry struct {
	data      []domain.UrbanZoningItem
	expiresAt time.Time
}

func (c *cache) getUrbanZoning(key string) ([]domain.UrbanZoningItem, bool) {
	c.mu.RLock()
	entry, ok := c.urbanZoningEntries[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		c.mu.Lock()
		delete(c.urbanZoningEntries, key)
		c.mu.Unlock()
		return nil, false
	}
	copied := make([]domain.UrbanZoningItem, len(entry.data))
	copy(copied, entry.data)
	return copied, true
}

func (c *cache) setUrbanZoning(key string, data []domain.UrbanZoningItem) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.urbanZoningEntries[key] = urbanZoningCacheEntry{data: data, expiresAt: time.Now().Add(cacheTTL)}
}

// liquefactionCacheEntry は液状化発生傾向図キャッシュの1エントリ
type liquefactionCacheEntry struct {
	data      []domain.LiquefactionRiskItem
	expiresAt time.Time
}

func (c *cache) getLiquefaction(key string) ([]domain.LiquefactionRiskItem, bool) {
	c.mu.RLock()
	entry, ok := c.liquefactionEntries[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		c.mu.Lock()
		delete(c.liquefactionEntries, key)
		c.mu.Unlock()
		return nil, false
	}
	copied := make([]domain.LiquefactionRiskItem, len(entry.data))
	copy(copied, entry.data)
	return copied, true
}

func (c *cache) setLiquefaction(key string, data []domain.LiquefactionRiskItem) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.liquefactionEntries[key] = liquefactionCacheEntry{data: data, expiresAt: time.Now().Add(cacheTTL)}
}

// floodHazardCacheEntry は洪水浸水想定区域キャッシュの1エントリ
type floodHazardCacheEntry struct {
	data      []domain.FloodHazardItem
	expiresAt time.Time
}

func (c *cache) getFloodHazard(key string) ([]domain.FloodHazardItem, bool) {
	c.mu.RLock()
	entry, ok := c.floodHazardEntries[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		c.mu.Lock()
		delete(c.floodHazardEntries, key)
		c.mu.Unlock()
		return nil, false
	}
	copied := make([]domain.FloodHazardItem, len(entry.data))
	copy(copied, entry.data)
	return copied, true
}

func (c *cache) setFloodHazard(key string, data []domain.FloodHazardItem) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.floodHazardEntries[key] = floodHazardCacheEntry{data: data, expiresAt: time.Now().Add(cacheTTL)}
}

// stormHazardCacheEntry は高潮浸水想定区域キャッシュの1エントリ
type stormHazardCacheEntry struct {
	data      []domain.StormHazardItem
	expiresAt time.Time
}

func (c *cache) getStormHazard(key string) ([]domain.StormHazardItem, bool) {
	c.mu.RLock()
	entry, ok := c.stormHazardEntries[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		c.mu.Lock()
		delete(c.stormHazardEntries, key)
		c.mu.Unlock()
		return nil, false
	}
	copied := make([]domain.StormHazardItem, len(entry.data))
	copy(copied, entry.data)
	return copied, true
}

func (c *cache) setStormHazard(key string, data []domain.StormHazardItem) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.stormHazardEntries[key] = stormHazardCacheEntry{data: data, expiresAt: time.Now().Add(cacheTTL)}
}

// tsunamiHazardCacheEntry は津波浸水想定キャッシュの1エントリ
type tsunamiHazardCacheEntry struct {
	data      []domain.TsunamiHazardItem
	expiresAt time.Time
}

func (c *cache) getTsunamiHazard(key string) ([]domain.TsunamiHazardItem, bool) {
	c.mu.RLock()
	entry, ok := c.tsunamiHazardEntries[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		c.mu.Lock()
		delete(c.tsunamiHazardEntries, key)
		c.mu.Unlock()
		return nil, false
	}
	copied := make([]domain.TsunamiHazardItem, len(entry.data))
	copy(copied, entry.data)
	return copied, true
}

func (c *cache) setTsunamiHazard(key string, data []domain.TsunamiHazardItem) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.tsunamiHazardEntries[key] = tsunamiHazardCacheEntry{data: data, expiresAt: time.Now().Add(cacheTTL)}
}

// landslideHazardCacheEntry は土砂災害警戒区域キャッシュの1エントリ
type landslideHazardCacheEntry struct {
	data      []domain.LandslideHazardItem
	expiresAt time.Time
}

func (c *cache) getLandslideHazard(key string) ([]domain.LandslideHazardItem, bool) {
	c.mu.RLock()
	entry, ok := c.landslideHazardEntries[key]
	c.mu.RUnlock()
	if !ok {
		return nil, false
	}
	if time.Now().After(entry.expiresAt) {
		c.mu.Lock()
		delete(c.landslideHazardEntries, key)
		c.mu.Unlock()
		return nil, false
	}
	copied := make([]domain.LandslideHazardItem, len(entry.data))
	copy(copied, entry.data)
	return copied, true
}

func (c *cache) setLandslideHazard(key string, data []domain.LandslideHazardItem) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.landslideHazardEntries[key] = landslideHazardCacheEntry{data: data, expiresAt: time.Now().Add(cacheTTL)}
}

// cacheKey は LandPriceQuery からキャッシュキーを生成する。
func cacheKey(q LandPriceQuery) string {
	return fmt.Sprintf("%s:%s:%d:%d:%d:%d", q.Area, q.City, q.Year, q.Quarter, q.ToYear, q.ToQuarter)
}
