package geocode

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
	cacheCollection = "geocode_cache"
	cacheTTL        = 30 * 24 * time.Hour // 30日
	cacheKeyLen     = 32
)

// docStore は FirestoreCache が必要とする最小限のドキュメント読み書き操作を表す。
// テストでは fakeDocStore で差し替えられる。
type docStore interface {
	get(ctx context.Context, docID string) (map[string]any, bool)
	set(ctx context.Context, docID string, data map[string]any) error
}

// firestoreDocStore は実際の Firestore クライアントを docStore に適合させるアダプタ。
type firestoreDocStore struct {
	client *firestore.Client
}

func (s *firestoreDocStore) get(ctx context.Context, docID string) (map[string]any, bool) {
	doc, err := s.client.Collection(cacheCollection).Doc(docID).Get(ctx)
	if err != nil {
		return nil, false
	}
	return doc.Data(), true
}

func (s *firestoreDocStore) set(ctx context.Context, docID string, data map[string]any) error {
	_, err := s.client.Collection(cacheCollection).Doc(docID).Set(ctx, data)
	return err
}

// FirestoreCache は Firestore を使ったジオコードキャッシュ実装
type FirestoreCache struct {
	store docStore
}

// NewFirestoreCache は Firestore クライアントからキャッシュを返す。
// client が nil の場合は noopCache を返す。
func NewFirestoreCache(client *firestore.Client) Cache {
	if client == nil {
		return &noopCache{}
	}
	return &FirestoreCache{store: &firestoreDocStore{client: client}}
}

func (f *FirestoreCache) cacheKey(address string) string {
	sum := sha256.Sum256([]byte(address))
	return fmt.Sprintf("%x", sum)[:cacheKeyLen]
}

func (f *FirestoreCache) Get(ctx context.Context, address string) (*Result, bool) {
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

	var result Result
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, false
	}
	return &result, true
}

func (f *FirestoreCache) Set(ctx context.Context, address string, result *Result) {
	key := f.cacheKey(address)
	raw, err := json.Marshal(result)
	if err != nil {
		return
	}
	if err := f.store.set(ctx, key, map[string]any{
		"data":      string(raw),
		"expiresAt": time.Now().Add(cacheTTL),
	}); err != nil {
		slog.WarnContext(ctx, "geocode_cache: failed to write to Firestore", "error", err)
	}
}
