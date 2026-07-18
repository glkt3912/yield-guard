package geocode

import (
	"context"
	"sync"
	"testing"
	"time"
)

// fakeDocStore はテスト用のインメモリ docStore 実装。
type fakeDocStore struct {
	mu   sync.RWMutex
	docs map[string]map[string]any
}

func newFakeDocStore() *fakeDocStore {
	return &fakeDocStore{docs: make(map[string]map[string]any)}
}

func (f *fakeDocStore) get(_ context.Context, docID string) (map[string]any, bool) {
	f.mu.RLock()
	defer f.mu.RUnlock()
	d, ok := f.docs[docID]
	return d, ok
}

func (f *fakeDocStore) set(_ context.Context, docID string, data map[string]any) error {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.docs[docID] = data
	return nil
}

func newTestCache(store docStore) *FirestoreCache {
	return &FirestoreCache{store: store}
}

func TestGeocodeCache_SetDoesNotStoreAddress(t *testing.T) {
	store := newFakeDocStore()
	c := newTestCache(store)
	result := &Result{Lat: 35.6895, Lng: 139.6917}

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
	store := newFakeDocStore()
	c := newTestCache(store)
	want := &Result{Lat: 35.6895, Lng: 139.6917}

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
	c := newTestCache(newFakeDocStore())
	_, ok := c.Get(context.Background(), "存在しない住所")
	if ok {
		t.Fatal("expected miss on empty store")
	}
}

func TestGeocodeCache_GetMissOnExpiredTTL(t *testing.T) {
	store := newFakeDocStore()
	c := newTestCache(store)
	result := &Result{Lat: 35.0, Lng: 135.0}

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
	store := newFakeDocStore()
	c := newTestCache(store)

	before := time.Now()
	c.Set(context.Background(), "address", &Result{Lat: 35.0, Lng: 135.0})
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

func TestNewFirestoreCache_NilClientReturnsNoop(t *testing.T) {
	cache := NewFirestoreCache(nil)
	if _, isNoop := cache.(*noopCache); !isNoop {
		t.Errorf("expected *noopCache, got %T", cache)
	}
}
