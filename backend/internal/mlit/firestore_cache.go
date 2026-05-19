package mlit

import (
	"context"
	"encoding/json"
	"time"

	"cloud.google.com/go/firestore"

	"github.com/yield-guard/backend/internal/domain"
)

const (
	mlitCacheCollection = "mlit_cache"
	mlitCacheTTL        = 24 * time.Hour
)

// docStore は firestoreL2 が必要とする最小限のドキュメント読み書き操作を表す。
// テストでは fakeDocStore で差し替えられる。
type docStore interface {
	get(ctx context.Context, docID string) (map[string]any, bool)
	set(ctx context.Context, docID string, data map[string]any)
}

// firestoreDocStore は実際の Firestore クライアントを docStore に適合させるアダプタ。
type firestoreDocStore struct {
	client     *firestore.Client
	collection string
}

func (s *firestoreDocStore) get(ctx context.Context, docID string) (map[string]any, bool) {
	doc, err := s.client.Collection(s.collection).Doc(docID).Get(ctx)
	if err != nil {
		return nil, false
	}
	return doc.Data(), true
}

func (s *firestoreDocStore) set(ctx context.Context, docID string, data map[string]any) {
	_, _ = s.client.Collection(s.collection).Doc(docID).Set(ctx, data)
}

// l2Cache はFirestore L2キャッシュの汎用インターフェース。
type l2Cache[E any] interface {
	get(ctx context.Context, key string) ([]E, bool)
	set(ctx context.Context, key string, data []E)
}

// firestoreL2 はFirestoreをL2キャッシュとして使用する汎用実装。
type firestoreL2[E any] struct {
	store    docStore
	endpoint string
}

// noopL2 はFirestore未設定時のダミー実装。
type noopL2[E any] struct{}

func (n *noopL2[E]) get(_ context.Context, _ string) ([]E, bool) { return nil, false }
func (n *noopL2[E]) set(_ context.Context, _ string, _ []E)      {}

// newFirestoreL2 は client が nil の場合 noopL2 を、それ以外は firestoreL2 を返す。
func newFirestoreL2[E any](client *firestore.Client, endpoint string) l2Cache[E] {
	if client == nil {
		return &noopL2[E]{}
	}
	return &firestoreL2[E]{
		store:    &firestoreDocStore{client: client, collection: mlitCacheCollection},
		endpoint: endpoint,
	}
}

func (f *firestoreL2[E]) docID(key string) string {
	return f.endpoint + ":" + key
}

func (f *firestoreL2[E]) get(ctx context.Context, key string) ([]E, bool) {
	data, ok := f.store.get(ctx, f.docID(key))
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

	var result []E
	if err := json.Unmarshal(raw, &result); err != nil {
		return nil, false
	}
	return result, true
}

func (f *firestoreL2[E]) set(ctx context.Context, key string, data []E) {
	raw, err := json.Marshal(data)
	if err != nil {
		return
	}
	f.store.set(ctx, f.docID(key), map[string]any{
		"data":      string(raw),
		"expiresAt": time.Now().Add(mlitCacheTTL),
		"endpoint":  f.endpoint,
	})
}

// l2CacheGroup は全エンドポイントの L2 キャッシュをまとめた構造体。
type l2CacheGroup struct {
	landPrices      l2Cache[domain.LandTransaction]
	municipalities  l2Cache[Municipality]
	ridership       l2Cache[StationRidership]
	population      l2Cache[domain.PopulationForecastItem]
	appraisals      l2Cache[domain.LandAppraisalItem]
	locationOpt     l2Cache[domain.LocationOptimizationItem]
	embankment      l2Cache[domain.EmbankmentItem]
	urbanRoad       l2Cache[domain.UrbanRoadItem]
	disaster        l2Cache[domain.DisasterHistoryItem]
	urbanZoning     l2Cache[domain.UrbanZoningItem]
	liquefaction    l2Cache[domain.LiquefactionRiskItem]
	floodHazard     l2Cache[domain.FloodHazardItem]
	stormHazard     l2Cache[domain.StormHazardItem]
	tsunamiHazard   l2Cache[domain.TsunamiHazardItem]
	landslideHazard l2Cache[domain.LandslideHazardItem]
	rentStats       l2Cache[domain.RentStatsResult]
}

// newNoopL2CacheGroup は全フィールドが noopL2 の l2CacheGroup を返す。
func newNoopL2CacheGroup() l2CacheGroup {
	return l2CacheGroup{
		landPrices:      &noopL2[domain.LandTransaction]{},
		municipalities:  &noopL2[Municipality]{},
		ridership:       &noopL2[StationRidership]{},
		population:      &noopL2[domain.PopulationForecastItem]{},
		appraisals:      &noopL2[domain.LandAppraisalItem]{},
		locationOpt:     &noopL2[domain.LocationOptimizationItem]{},
		embankment:      &noopL2[domain.EmbankmentItem]{},
		urbanRoad:       &noopL2[domain.UrbanRoadItem]{},
		disaster:        &noopL2[domain.DisasterHistoryItem]{},
		urbanZoning:     &noopL2[domain.UrbanZoningItem]{},
		liquefaction:    &noopL2[domain.LiquefactionRiskItem]{},
		floodHazard:     &noopL2[domain.FloodHazardItem]{},
		stormHazard:     &noopL2[domain.StormHazardItem]{},
		tsunamiHazard:   &noopL2[domain.TsunamiHazardItem]{},
		landslideHazard: &noopL2[domain.LandslideHazardItem]{},
		rentStats:       &noopL2[domain.RentStatsResult]{},
	}
}

// newFirestoreL2CacheGroup は Firestore クライアントから全フィールドが firestoreL2 の l2CacheGroup を返す。
func newFirestoreL2CacheGroup(fs *firestore.Client) l2CacheGroup {
	return l2CacheGroup{
		landPrices:      newFirestoreL2[domain.LandTransaction](fs, "XIT001"),
		municipalities:  newFirestoreL2[Municipality](fs, "XIT002"),
		ridership:       newFirestoreL2[StationRidership](fs, "XKT015"),
		population:      newFirestoreL2[domain.PopulationForecastItem](fs, "XKT013"),
		appraisals:      newFirestoreL2[domain.LandAppraisalItem](fs, "XCT001"),
		locationOpt:     newFirestoreL2[domain.LocationOptimizationItem](fs, "XKT003"),
		embankment:      newFirestoreL2[domain.EmbankmentItem](fs, "XKT020"),
		urbanRoad:       newFirestoreL2[domain.UrbanRoadItem](fs, "XKT030"),
		disaster:        newFirestoreL2[domain.DisasterHistoryItem](fs, "XST001"),
		urbanZoning:     newFirestoreL2[domain.UrbanZoningItem](fs, "XKT001"),
		liquefaction:    newFirestoreL2[domain.LiquefactionRiskItem](fs, "XKT025"),
		floodHazard:     newFirestoreL2[domain.FloodHazardItem](fs, "XKT026"),
		stormHazard:     newFirestoreL2[domain.StormHazardItem](fs, "XKT027"),
		tsunamiHazard:   newFirestoreL2[domain.TsunamiHazardItem](fs, "XKT028"),
		landslideHazard: newFirestoreL2[domain.LandslideHazardItem](fs, "XKT029"),
		rentStats:       newFirestoreL2[domain.RentStatsResult](fs, "XIT001_rent"),
	}
}
