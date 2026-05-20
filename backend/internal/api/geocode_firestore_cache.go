package api

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"log/slog"
	"time"

	"cloud.google.com/go/firestore"
)

const (
	geocodeCacheCollection = "geocode_cache"
	geocodeCacheTTL        = 30 * 24 * time.Hour // 30日
	geocodeCacheKeyLen     = 32
)

// geocodeDocStore は firestoreGeocodeCache が必要とする最小限のドキュメント読み書き操作を表す。
// テストでは fakeGeocodeDocStore で差し替えられる。
type geocodeDocStore interface {
	get(ctx context.Context, docID string) (map[string]any, bool)
	set(ctx context.Context, docID string, data map[string]any) error
}

// firestoreGeocodeDocStore は実際の Firestore クライアントを geocodeDocStore に適合させるアダプタ。
type firestoreGeocodeDocStore struct {
	client *firestore.Client
}

func (s *firestoreGeocodeDocStore) get(ctx context.Context, docID string) (map[string]any, bool) {
	doc, err := s.client.Collection(geocodeCacheCollection).Doc(docID).Get(ctx)
	if err != nil {
		return nil, false
	}
	return doc.Data(), true
}

func (s *firestoreGeocodeDocStore) set(ctx context.Context, docID string, data map[string]any) error {
	_, err := s.client.Collection(geocodeCacheCollection).Doc(docID).Set(ctx, data)
	return err
}

// firestoreGeocodeCache は Firestore を使ったジオコードキャッシュ実装
type firestoreGeocodeCache struct {
	store geocodeDocStore
}

// NewFirestoreGeocodeCache は Firestore クライアントからキャッシュを返す。
// client が nil の場合は noopGeocodeCache を返す。
func NewFirestoreGeocodeCache(client *firestore.Client) GeocodeCache {
	if client == nil {
		return &noopGeocodeCache{}
	}
	return &firestoreGeocodeCache{store: &firestoreGeocodeDocStore{client: client}}
}

func (f *firestoreGeocodeCache) cacheKey(address string) string {
	sum := sha256.Sum256([]byte(address))
	return fmt.Sprintf("%x", sum)[:geocodeCacheKeyLen]
}

func (f *firestoreGeocodeCache) Get(ctx context.Context, address string) (*GeocodeResult, bool) {
	key := f.cacheKey(address)
	data, ok := f.store.get(ctx, key)
	if !ok {
		return nil, false
	}

	expiresAt, ok := data["expiresAt"].(time.Time)
	if !ok || time.Now().After(expiresAt) {
		return nil, false
	}

	var raw []byte
	switch v := data["data"].(type) {
	case []byte:
		raw = v
	case string:
		raw = []byte(v)
	default:
		return nil, false
	}

	var result GeocodeResult
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, false
	}
	return &result, true
}

func (f *firestoreGeocodeCache) Set(ctx context.Context, address string, result *GeocodeResult) {
	key := f.cacheKey(address)
	raw, err := json.Marshal(result)
	if err != nil {
		return
	}
	if err := f.store.set(ctx, key, map[string]any{
		"data":      string(raw),
		"expiresAt": time.Now().Add(geocodeCacheTTL),
	}); err != nil {
		slog.WarnContext(ctx, "geocode_cache: failed to write to Firestore", "error", err)
	}
}
