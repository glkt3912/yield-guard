package api

import (
	"context"
	"sync"
	"testing"
	"time"
)

// fakeGeocodeDocStore はテスト用のインメモリ geocodeDocStore 実装。
type fakeGeocodeDocStore struct {
	mu   sync.RWMutex
	docs map[string]map[string]any
}

func newFakeGeocodeDocStore() *fakeGeocodeDocStore {
	return &fakeGeocodeDocStore{docs: make(map[string]map[string]any)}
}

func (f *fakeGeocodeDocStore) get(_ context.Context, docID string) (map[string]any, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	d, ok := f.docs[docID]
	return d, ok
}

func (f *fakeGeocodeDocStore) set(_ context.Context, docID string, data map[string]any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.docs[docID] = data
	return nil
}

func newTestGeocodeCache(store geocodeDocStore) *firestoreGeocodeCache {
	return &firestoreGeocodeCache{store: store}
}

func TestGeocodeCache_SetDoesNotStoreAddress(t *testing.T) {
	store := newFakeGeocodeDocStore()
	c := newTestGeocodeCache(store)
	result := &GeocodeResult{Lat: 35.6895, Lng: 139.6917}

	c.Set(context.Background(), "東京都千代田区丸の内1-1-1", result)

	key := c.cacheKey("東京都千代田区丸の内1-1-1")
	doc, ok := store.get(context.Background(), key)
	if !ok {
		t.Fatal("expected doc in store after Set")
	}
	if _, found := doc["address"]; found {
		t.Error("address field must not be stored in cache document (PII)")
	}
}

func TestGeocodeCache_SetThenGet(t *testing.T) {
	store := newFakeGeocodeDocStore()
	c := newTestGeocodeCache(store)
	want := &GeocodeResult{Lat: 35.6895, Lng: 139.6917}

	c.Set(context.Background(), "東京都千代田区", want)
	got, ok := c.Get(context.Background(), "東京都千代田区")

	if !ok {
		t.Fatal("expected cache hit after Set")
	}
	if got.Lat != want.Lat || got.Lng != want.Lng {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

func TestGeocodeCache_GetMissOnEmpty(t *testing.T) {
	c := newTestGeocodeCache(newFakeGeocodeDocStore())
	_, ok := c.Get(context.Background(), "存在しない住所")
	if ok {
		t.Fatal("expected miss on empty store")
	}
}

func TestGeocodeCache_GetMissOnExpiredTTL(t *testing.T) {
	store := newFakeGeocodeDocStore()
	c := newTestGeocodeCache(store)
	result := &GeocodeResult{Lat: 35.0, Lng: 135.0}

	c.Set(context.Background(), "address", result)

	key := c.cacheKey("address")
	doc, _ := store.get(context.Background(), key)
	doc["expiresAt"] = time.Now().Add(-1 * time.Second)
	store.set(context.Background(), key, doc) //nolint:errcheck

	_, ok := c.Get(context.Background(), "address")
	if ok {
		t.Fatal("expected miss for expired entry")
	}
}

func TestGeocodeCache_TTLIs30Days(t *testing.T) {
	store := newFakeGeocodeDocStore()
	c := newTestGeocodeCache(store)

	before := time.Now()
	c.Set(context.Background(), "address", &GeocodeResult{Lat: 35.0, Lng: 135.0})
	after := time.Now()

	key := c.cacheKey("address")
	doc, _ := store.get(context.Background(), key)
	expiresAt, ok := doc["expiresAt"].(time.Time)
	if !ok {
		t.Fatalf("expiresAt type = %T, want time.Time", doc["expiresAt"])
	}

	expectedMin := before.Add(30 * 24 * time.Hour)
	expectedMax := after.Add(30 * 24 * time.Hour)
	if expiresAt.Before(expectedMin) || expiresAt.After(expectedMax) {
		t.Errorf("TTL is not ~30 days: expiresAt=%v", expiresAt)
	}
}

func TestNewFirestoreGeocodeCache_NilClientReturnsNoop(t *testing.T) {
	cache := NewFirestoreGeocodeCache(nil)
	if _, isNoop := cache.(*noopGeocodeCache); !isNoop {
		t.Errorf("expected *noopGeocodeCache, got %T", cache)
	}
}
