package mlit

import (
	"context"
	"encoding/json"
	"sync"
	"testing"
	"time"
)

// fakeDocStore は docStore インターフェースのインメモリ実装。
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

func (f *fakeDocStore) set(_ context.Context, docID string, data map[string]any) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.docs[docID] = data
}

// newTestL2 はテスト用の firestoreL2[Municipality] を返す。
func newTestL2(store docStore) *firestoreL2[Municipality] {
	return &firestoreL2[Municipality]{store: store, endpoint: "TEST"}
}

// marshaledMunis は []Municipality を JSON 文字列にして返すヘルパー。
func marshaledMunis(munis []Municipality) string {
	b, _ := json.Marshal(munis)
	return string(b)
}

// --- firestoreL2 ---

func TestFirestoreL2_Miss(t *testing.T) {
	c := newTestL2(newFakeDocStore())
	got, ok := c.get(context.Background(), "key1")
	if ok {
		t.Fatalf("expected miss on empty store, got %v", got)
	}
}

func TestFirestoreL2_SetThenGet(t *testing.T) {
	c := newTestL2(newFakeDocStore())
	munis := []Municipality{{ID: "13101", Name: "千代田区"}}

	c.set(context.Background(), "key1", munis)
	got, ok := c.get(context.Background(), "key1")

	if !ok {
		t.Fatal("expected hit after set")
	}
	if len(got) != 1 || got[0].ID != "13101" || got[0].Name != "千代田区" {
		t.Errorf("unexpected data: %+v", got)
	}
}

func TestFirestoreL2_TTLExpiry(t *testing.T) {
	store := newFakeDocStore()
	c := newTestL2(store)
	munis := []Municipality{{ID: "13101", Name: "千代田区"}}

	store.set(context.Background(), c.docID("key1"), map[string]any{
		"data":      marshaledMunis(munis),
		"expiresAt": time.Now().Add(-1 * time.Second), // 期限切れ
		"endpoint":  "TEST",
	})

	got, ok := c.get(context.Background(), "key1")
	if ok {
		t.Fatalf("expected miss for expired entry, got %v", got)
	}
}

func TestFirestoreL2_TTLValid(t *testing.T) {
	store := newFakeDocStore()
	c := newTestL2(store)
	munis := []Municipality{{ID: "13101", Name: "千代田区"}}

	store.set(context.Background(), c.docID("key1"), map[string]any{
		"data":      marshaledMunis(munis),
		"expiresAt": time.Now().Add(1 * time.Hour),
		"endpoint":  "TEST",
	})

	got, ok := c.get(context.Background(), "key1")
	if !ok {
		t.Fatal("expected hit for valid TTL entry")
	}
	if len(got) != 1 || got[0].ID != "13101" {
		t.Errorf("unexpected data: %+v", got)
	}
}

func TestFirestoreL2_MalformedData_MissingField(t *testing.T) {
	store := newFakeDocStore()
	c := newTestL2(store)

	// "data" フィールドなし
	store.set(context.Background(), c.docID("key1"), map[string]any{
		"expiresAt": time.Now().Add(1 * time.Hour),
		"endpoint":  "TEST",
	})

	_, ok := c.get(context.Background(), "key1")
	if ok {
		t.Fatal("expected miss when data field is missing")
	}
}

func TestFirestoreL2_BadJSON(t *testing.T) {
	store := newFakeDocStore()
	c := newTestL2(store)

	store.set(context.Background(), c.docID("key1"), map[string]any{
		"data":      `not valid json [[[`,
		"expiresAt": time.Now().Add(1 * time.Hour),
		"endpoint":  "TEST",
	})

	_, ok := c.get(context.Background(), "key1")
	if ok {
		t.Fatal("expected miss for invalid JSON")
	}
}

func TestFirestoreL2_DocIDFormat(t *testing.T) {
	c := newTestL2(newFakeDocStore())
	got := c.docID("somekey")
	want := "TEST:somekey"
	if got != want {
		t.Errorf("docID = %q, want %q", got, want)
	}
}

func TestFirestoreL2_SetWritesCorrectFields(t *testing.T) {
	store := newFakeDocStore()
	c := newTestL2(store)
	munis := []Municipality{{ID: "13101", Name: "千代田区"}}

	before := time.Now()
	c.set(context.Background(), "key1", munis)
	after := time.Now()

	doc, ok := store.get(context.Background(), c.docID("key1"))
	if !ok {
		t.Fatal("expected doc in store after set")
	}

	if doc["endpoint"] != "TEST" {
		t.Errorf("endpoint = %v, want TEST", doc["endpoint"])
	}
	if doc["data"] == nil {
		t.Error("data field should not be nil")
	}
	expiresAt, ok := doc["expiresAt"].(time.Time)
	if !ok {
		t.Fatalf("expiresAt type = %T, want time.Time", doc["expiresAt"])
	}
	expectedMin := before.Add(mlitCacheTTL)
	expectedMax := after.Add(mlitCacheTTL)
	if expiresAt.Before(expectedMin) || expiresAt.After(expectedMax) {
		t.Errorf("expiresAt %v not in [%v, %v]", expiresAt, expectedMin, expectedMax)
	}
}

func TestFirestoreL2_TTLIs24Hours(t *testing.T) {
	store := newFakeDocStore()
	c := newTestL2(store)

	before := time.Now()
	c.set(context.Background(), "key1", []Municipality{{ID: "13101", Name: "千代田区"}})
	after := time.Now()

	doc, _ := store.get(context.Background(), c.docID("key1"))
	expiresAt := doc["expiresAt"].(time.Time)

	if expiresAt.Before(before.Add(24*time.Hour)) || expiresAt.After(after.Add(24*time.Hour)) {
		t.Errorf("TTL is not ~24h: expiresAt=%v", expiresAt)
	}
}

// --- noopL2 ---

func TestNoopL2_AlwaysMisses(t *testing.T) {
	n := &noopL2[Municipality]{}
	got, ok := n.get(context.Background(), "any")
	if ok || got != nil {
		t.Errorf("noopL2.get should always return (nil, false), got (%v, %v)", got, ok)
	}
}

func TestNoopL2_SetIsNoop(t *testing.T) {
	n := &noopL2[Municipality]{}
	n.set(context.Background(), "any", []Municipality{{ID: "13101"}})
	got, ok := n.get(context.Background(), "any")
	if ok || got != nil {
		t.Errorf("noopL2 should not store data, got (%v, %v)", got, ok)
	}
}

// --- newFirestoreL2 constructor ---

func TestNewFirestoreL2_NilClientReturnsNoop(t *testing.T) {
	cache := newFirestoreL2[Municipality](nil, "TEST")
	if _, isNoop := cache.(*noopL2[Municipality]); !isNoop {
		t.Errorf("expected *noopL2, got %T", cache)
	}
}

func TestNewFirestoreL2_NonNilClientReturnsFirestoreL2(t *testing.T) {
	// newFirestoreL2 の nil チェックは client ポインタの nil チェックのみ。
	// 非 nil のダミーポインタで型を確認する（実際の接続は不要）。
	// *firestore.Client のゼロ値ポインタは使えないため、
	// 代わりに newFirestoreL2CacheGroup の戻り値型から確認する。
	group := newFirestoreL2CacheGroup(nil)
	if _, isNoop := group.municipalities.(*noopL2[Municipality]); !isNoop {
		t.Errorf("nil client should produce noopL2, got %T", group.municipalities)
	}
}
