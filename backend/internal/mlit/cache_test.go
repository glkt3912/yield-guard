package mlit

import (
	"sync"
	"testing"
	"time"

	"github.com/yield-guard/backend/internal/domain"
)

// --- cacheKey ---

func TestCacheKey_Deterministic(t *testing.T) {
	q := LandPriceQuery{Area: "13", City: "101", Year: 2024, Quarter: 1, ToYear: 2024, ToQuarter: 4}
	k1 := cacheKey(q)
	k2 := cacheKey(q)
	if k1 != k2 {
		t.Errorf("cacheKey is not deterministic: %q != %q", k1, k2)
	}
}

func TestCacheKey_UniquePerQuery(t *testing.T) {
	q1 := LandPriceQuery{Area: "13", Year: 2024, Quarter: 1}
	q2 := LandPriceQuery{Area: "14", Year: 2024, Quarter: 1}
	if cacheKey(q1) == cacheKey(q2) {
		t.Errorf("different queries produced the same cache key: %q", cacheKey(q1))
	}
}

// --- municipalities ---

func TestCacheMuni_HitMiss(t *testing.T) {
	c := newCache()

	_, ok := c.municipalities.get("13")
	if ok {
		t.Fatal("expected miss on empty cache")
	}

	data := []Municipality{{ID: "13101", Name: "千代田区"}}
	c.municipalities.set("13", data)

	got, ok := c.municipalities.get("13")
	if !ok {
		t.Fatal("expected hit after set")
	}
	if len(got) != 1 || got[0].ID != "13101" {
		t.Errorf("unexpected data: %+v", got)
	}
}

func TestCacheMuni_TTLExpiry(t *testing.T) {
	c := newCache()
	c.municipalities.mu.Lock()
	c.municipalities.entries["13"] = genericEntry[Municipality]{
		data:      []Municipality{{ID: "13101", Name: "千代田区"}},
		expiresAt: time.Now().Add(-1 * time.Second), // 期限切れ
	}
	c.municipalities.mu.Unlock()

	_, ok := c.municipalities.get("13")
	if ok {
		t.Error("expected miss after TTL expiry")
	}

	// 期限切れエントリが削除されていること
	c.municipalities.mu.RLock()
	_, stillExists := c.municipalities.entries["13"]
	c.municipalities.mu.RUnlock()
	if stillExists {
		t.Error("expired muni entry was not deleted from map")
	}
}

func TestCacheMuni_ReturnsCopy(t *testing.T) {
	c := newCache()
	original := []Municipality{{ID: "13101", Name: "千代田区"}}
	c.municipalities.set("13", original)

	got, _ := c.municipalities.get("13")
	got[0].Name = "MODIFIED"

	got2, _ := c.municipalities.get("13")
	if got2[0].Name != "千代田区" {
		t.Errorf("cache was mutated by caller: got %q, want '千代田区'", got2[0].Name)
	}
}

// --- ridership ---

func TestCacheRidership_HitMiss(t *testing.T) {
	c := newCache()
	key := "14/2/2"

	_, ok := c.ridership.get(key)
	if ok {
		t.Fatal("expected miss on empty cache")
	}

	data := []StationRidership{{StationName: "横浜", Passengers: 100_000}}
	c.ridership.set(key, data)

	got, ok := c.ridership.get(key)
	if !ok {
		t.Fatal("expected hit after set")
	}
	if len(got) != 1 || got[0].StationName != "横浜" {
		t.Errorf("unexpected data: %+v", got)
	}
}

func TestCacheRidership_TTLExpiry(t *testing.T) {
	c := newCache()
	key := "14/2/2"
	c.ridership.mu.Lock()
	c.ridership.entries[key] = genericEntry[StationRidership]{
		data:      []StationRidership{{StationName: "横浜", Passengers: 100_000}},
		expiresAt: time.Now().Add(-1 * time.Second),
	}
	c.ridership.mu.Unlock()

	_, ok := c.ridership.get(key)
	if ok {
		t.Error("expected miss after TTL expiry")
	}

	c.ridership.mu.RLock()
	_, stillExists := c.ridership.entries[key]
	c.ridership.mu.RUnlock()
	if stillExists {
		t.Error("expired ridership entry was not deleted from map")
	}
}

// --- population ---

func TestCachePopulation_HitMiss(t *testing.T) {
	c := newCache()
	key := "14/5/5"

	_, ok := c.population.get(key)
	if ok {
		t.Fatal("expected miss on empty cache")
	}

	data := []domain.PopulationForecastItem{{Year: 2020, Pop: 50000}}
	c.population.set(key, data)

	got, ok := c.population.get(key)
	if !ok {
		t.Fatal("expected hit after set")
	}
	if len(got) != 1 || got[0].Year != 2020 {
		t.Errorf("unexpected data: %+v", got)
	}
}

// --- floodHazard ---

func TestCacheFloodHazard_HitMiss(t *testing.T) {
	c := newCache()
	key := "15/3/3"

	_, ok := c.floodHazard.get(key)
	if ok {
		t.Fatal("expected miss on empty cache")
	}

	data := []domain.FloodHazardItem{{DepthRank: 3, RiverName: "多摩川"}}
	c.floodHazard.set(key, data)

	got, ok := c.floodHazard.get(key)
	if !ok {
		t.Fatal("expected hit after set")
	}
	if len(got) != 1 || got[0].RiverName != "多摩川" {
		t.Errorf("unexpected data: %+v", got)
	}
}

func TestCacheFloodHazard_TTLExpiry(t *testing.T) {
	c := newCache()
	key := "15/3/3"
	c.floodHazard.mu.Lock()
	c.floodHazard.entries[key] = genericEntry[domain.FloodHazardItem]{
		data:      []domain.FloodHazardItem{{DepthRank: 3}},
		expiresAt: time.Now().Add(-1 * time.Second),
	}
	c.floodHazard.mu.Unlock()

	_, ok := c.floodHazard.get(key)
	if ok {
		t.Error("expected miss after TTL expiry")
	}

	c.floodHazard.mu.RLock()
	_, stillExists := c.floodHazard.entries[key]
	c.floodHazard.mu.RUnlock()
	if stillExists {
		t.Error("expired flood entry was not deleted from map")
	}
}

// --- 並行アクセス ---

func TestCacheTileGroup_ConcurrentAccess(t *testing.T) {
	c := newCache()
	const goroutines = 50

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			key := "14/5/5"
			data := []domain.PopulationForecastItem{{Year: 2020, Pop: float64(idx * 1000)}}
			if idx%2 == 0 {
				c.population.set(key, data)
			} else {
				c.population.get(key)
			}
		}(i)
	}
	wg.Wait()
}

func TestCacheHazardGroup_ConcurrentAccess(t *testing.T) {
	c := newCache()
	const goroutines = 50

	var wg sync.WaitGroup
	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			key := "15/3/3"
			data := []domain.FloodHazardItem{{DepthRank: idx % 5}}
			if idx%2 == 0 {
				c.floodHazard.set(key, data)
			} else {
				c.floodHazard.get(key)
			}
		}(i)
	}
	wg.Wait()
}
