package api

import (
	"context"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"time"

	"cloud.google.com/go/firestore"
)

const (
	geocodeCacheCollection = "geocode_cache"
	geocodeCacheTTL        = 30 * 24 * time.Hour // 30日
	geocodeCacheKeyLen     = 32
)

// firestoreGeocodeCache は Firestore を使ったジオコードキャッシュ実装
type firestoreGeocodeCache struct {
	client *firestore.Client
}

// NewFirestoreGeocodeCache は Firestore クライアントからキャッシュを返す。
// client が nil の場合は noopGeocodeCache を返す。
func NewFirestoreGeocodeCache(client *firestore.Client) GeocodeCache {
	if client == nil {
		return &noopGeocodeCache{}
	}
	return &firestoreGeocodeCache{client: client}
}

func (f *firestoreGeocodeCache) cacheKey(address string) string {
	sum := sha256.Sum256([]byte(address))
	return fmt.Sprintf("%x", sum)[:geocodeCacheKeyLen]
}

func (f *firestoreGeocodeCache) Get(ctx context.Context, address string) (*GeocodeResult, bool) {
	key := f.cacheKey(address)
	doc, err := f.client.Collection(geocodeCacheCollection).Doc(key).Get(ctx)
	if err != nil {
		return nil, false
	}

	data := doc.Data()
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
	_, _ = f.client.Collection(geocodeCacheCollection).Doc(key).Set(ctx, map[string]any{
		"data":      string(raw),
		"expiresAt": time.Now().Add(geocodeCacheTTL),
		"address":   address,
	})
}
