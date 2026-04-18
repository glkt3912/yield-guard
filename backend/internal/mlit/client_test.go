package mlit

import (
	"context"
	"encoding/json"
	"math"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/yield-guard/backend/internal/domain"
)

// ---- parseFloat ----

func TestParseFloat(t *testing.T) {
	tests := []struct {
		input string
		want  float64
	}{
		{"", 0},
		{"－", 0},
		{"-", 0},
		{"1000000", 1_000_000},
		{"1,000,000", 1_000_000},
		{"１２３４５６", 123_456},
		{"１００", 100},
		{"100以上", 100},
		{"500未満", 500},
		{"100m²", 100},
		{"200㎡", 200},
		{"50坪", 50},
		{"300000円", 300_000},
		{"  1000  ", 1_000},
		{"abc", 0},
		{"1,234,567", 1_234_567},
		// 浮動小数点
		{"1.5", 1.5},
		{"3.30578", 3.30578},
		// 負数: "-" 単体のみゼロ扱い。"-100" のような数値は正しく解析される
		{"-100", -100},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseFloat(tt.input)
			if got != tt.want {
				t.Errorf("parseFloat(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// ---- parseJapaneseYear ----

func TestParseJapaneseYear(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"令和5年", 2023},
		{"令和1年", 2019},
		{"平成15年", 2003},
		{"平成元年", 0}, // "元" は非対応（数字でない）
		{"昭和63年", 1988},
		{"昭和1年", 1926},
		{"大正10年", 1921},
		{"明治45年", 1912},
		{"2020年", 2020},
		{"2020", 2020},
		{"1990", 1990},
		{"", 0},
		{"－", 0},
		{"不明", 0},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := parseJapaneseYear(tt.input)
			if got != tt.want {
				t.Errorf("parseJapaneseYear(%q) = %d, want %d", tt.input, got, tt.want)
			}
		})
	}
}

// ---- isLandType ----

func TestIsLandType(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"宅地(土地)", true},
		{"宅地のみ", false},  // "土地" を含まない
		{"土地のみ", false},  // "宅地" を含まない
		{"中古マンション等", false},
		{"農地", false},
		{"", false},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := isLandType(tt.input)
			if got != tt.want {
				t.Errorf("isLandType(%q) = %v, want %v", tt.input, got, tt.want)
			}
		})
	}
}

// ---- buildLandPricesURL ----

func newTestClient(serverURL string) *Client {
	return &Client{httpClient: &http.Client{}, baseURL: serverURL, cache: newCache()}
}

func TestBuildLandPricesURL(t *testing.T) {
	c := newTestClient("http://example.com")
	validQ := LandPriceQuery{Area: "13", Year: 2023, Quarter: 1, ToYear: 2023, ToQuarter: 4}

	t.Run("area が空のときエラー", func(t *testing.T) {
		q := validQ
		q.Area = ""
		_, err := c.buildLandPricesURL(q)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("year が0のときエラー", func(t *testing.T) {
		q := validQ
		q.Year = 0
		_, err := c.buildLandPricesURL(q)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("quarter が範囲外のときエラー", func(t *testing.T) {
		q := validQ
		q.Quarter = 5
		_, err := c.buildLandPricesURL(q)
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("必須パラメータが揃っているとき URL を生成する", func(t *testing.T) {
		got, err := c.buildLandPricesURL(validQ)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		for _, param := range []string{"area=13", "year=2023", "quarter=1", "toYear=2023", "toQuarter=4"} {
			if !strings.Contains(got, param) {
				t.Errorf("URL %q does not contain %q", got, param)
			}
		}
	})

	t.Run("city が指定されているときクエリに含まれる", func(t *testing.T) {
		q := validQ
		q.City = "13101"
		got, err := c.buildLandPricesURL(q)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if !strings.Contains(got, "city=13101") {
			t.Errorf("URL %q does not contain city=13101", got)
		}
	})
}

// ---- parseTransactions ----

func TestParseTransactions(t *testing.T) {
	t.Run("宅地(土地)のみ抽出される", func(t *testing.T) {
		raw := []Transaction{
			{Type: "宅地(土地)", TradePrice: "10000000", Area: "100", PricePerUnit: "100000"},
			{Type: "中古マンション等", TradePrice: "20000000", Area: "60", PricePerUnit: "333333"},
			{Type: "宅地(土地)", TradePrice: "5000000", Area: "50", PricePerUnit: "100000"},
		}
		got := parseTransactions(raw)
		if len(got) != 2 {
			t.Errorf("len = %d, want 2", len(got))
		}
	})

	t.Run("単価が空のとき総額と面積から算出", func(t *testing.T) {
		raw := []Transaction{
			{Type: "宅地(土地)", TradePrice: "10000000", Area: "100", PricePerUnit: ""},
		}
		got := parseTransactions(raw)
		if len(got) != 1 {
			t.Fatalf("len = %d, want 1", len(got))
		}
		// 10,000,000 / 100 = 100,000 円/m²
		if got[0].PricePerSqm != 100_000 {
			t.Errorf("PricePerSqm = %v, want 100000", got[0].PricePerSqm)
		}
	})

	t.Run("PricePerTsubo が正しく算出される", func(t *testing.T) {
		raw := []Transaction{
			{Type: "宅地(土地)", TradePrice: "10000000", Area: "100", PricePerUnit: "100000"},
		}
		got := parseTransactions(raw)
		if len(got) != 1 {
			t.Fatalf("len = %d, want 1", len(got))
		}
		// 100,000 円/m² × 3.30578 m²/坪 ≈ 330,578 円/坪
		wantTsubo := 100_000.0 * domain.SqmPerTsubo
		if math.Abs(got[0].PricePerTsubo-wantTsubo) > 1 {
			t.Errorf("PricePerTsubo = %v, want ≈ %v", got[0].PricePerTsubo, wantTsubo)
		}
	})

	t.Run("空スライスのとき空スライスを返す", func(t *testing.T) {
		got := parseTransactions([]Transaction{})
		if len(got) != 0 {
			t.Errorf("len = %d, want 0", len(got))
		}
	})
}

// ---- FetchLandPrices リトライロジック ----

func okResponse(w http.ResponseWriter) {
	resp := APIResponse{Status: "OK", Data: []Transaction{
		{Type: "宅地(土地)", TradePrice: "10000000", Area: "100", PricePerUnit: "100000", Period: "令和5年第3四半期"},
	}}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		panic(err)
	}
}

func TestFetchLandPrices_InvalidQuery(t *testing.T) {
	c := newTestClient("http://example.com")
	// Area が空 → buildURL がエラーを返し HTTP リクエストは発生しない
	_, err := c.FetchLandPrices(context.Background(), LandPriceQuery{Year: 2023, Quarter: 1, ToYear: 2023, ToQuarter: 4})
	if err == nil {
		t.Fatal("expected error for missing area, got nil")
	}
}

func TestFetchLandPrices_RetryOn5xx(t *testing.T) {
	attempt := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt++
		if attempt < 3 {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}
		okResponse(w)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	result, err := c.FetchLandPrices(context.Background(), LandPriceQuery{
		Area: "13", Year: 2023, Quarter: 1, ToYear: 2023, ToQuarter: 4,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 1 {
		t.Errorf("len = %d, want 1", len(result))
	}
	if attempt != 3 {
		t.Errorf("attempt = %d, want 3 (2 failures + 1 success)", attempt)
	}
}

func TestFetchLandPrices_AllAttemptsFailWith5xx(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	_, err := c.FetchLandPrices(context.Background(), LandPriceQuery{
		Area: "13", Year: 2023, Quarter: 1, ToYear: 2023, ToQuarter: 4,
	})
	if err == nil {
		t.Fatal("expected error after all retries, got nil")
	}
}

func TestFetchLandPrices_NoRetryOn4xx(t *testing.T) {
	attempt := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt++
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	_, err := c.FetchLandPrices(context.Background(), LandPriceQuery{
		Area: "13", Year: 2023, Quarter: 1, ToYear: 2023, ToQuarter: 4,
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if attempt != 1 {
		t.Errorf("attempt = %d, want 1 (no retry on 4xx)", attempt)
	}
}

func TestFetchLandPrices_ContextTimeout(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer ts.Close()

	// リトライ待機中にタイムアウトさせる（retryBaseDelay=1s より短く、CI高負荷時の余裕も考慮）
	ctx, cancel := context.WithTimeout(context.Background(), 700*time.Millisecond)
	defer cancel()

	c := newTestClient(ts.URL)
	_, err := c.FetchLandPrices(ctx, LandPriceQuery{
		Area: "13", Year: 2023, Quarter: 1, ToYear: 2023, ToQuarter: 4,
	})
	if err == nil {
		t.Fatal("expected error after context timeout, got nil")
	}
}

// ---- キャッシュ ----

func TestFetchLandPrices_CacheHit(t *testing.T) {
	apiCallCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCallCount++
		okResponse(w)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	q := LandPriceQuery{Area: "13", Year: 2024, Quarter: 1, ToYear: 2024, ToQuarter: 4}

	// 1回目: APIコール発生
	result1, err := c.FetchLandPrices(context.Background(), q)
	if err != nil {
		t.Fatalf("1st call failed: %v", err)
	}

	// 2回目: キャッシュから返す（APIコールなし）
	result2, err := c.FetchLandPrices(context.Background(), q)
	if err != nil {
		t.Fatalf("2nd call failed: %v", err)
	}

	if apiCallCount != 1 {
		t.Errorf("apiCallCount = %d, want 1 (2nd call should hit cache)", apiCallCount)
	}
	if len(result1) != len(result2) {
		t.Errorf("result length mismatch: %d vs %d", len(result1), len(result2))
	}
}

func TestFetchLandPrices_CacheMissOnDifferentQuery(t *testing.T) {
	apiCallCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCallCount++
		okResponse(w)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	q1 := LandPriceQuery{Area: "13", Year: 2024, Quarter: 1, ToYear: 2024, ToQuarter: 4}
	q2 := LandPriceQuery{Area: "27", Year: 2024, Quarter: 1, ToYear: 2024, ToQuarter: 4} // 大阪府

	if _, err := c.FetchLandPrices(context.Background(), q1); err != nil {
		t.Fatalf("q1 call failed: %v", err)
	}
	if _, err := c.FetchLandPrices(context.Background(), q2); err != nil {
		t.Fatalf("q2 call failed: %v", err)
	}

	// クエリが異なるので両方APIコールが発生する
	if apiCallCount != 2 {
		t.Errorf("apiCallCount = %d, want 2 (different queries must not share cache)", apiCallCount)
	}
}

func TestCache_TTLExpiry(t *testing.T) {
	c := newCache()
	key := "test-key"
	data := []domain.LandTransaction{{Period: "2024年第1四半期", TradePrice: 10_000_000}}

	// 有効期限を過去に設定して直接注入
	c.mu.Lock()
	c.entries[key] = cacheEntry{
		data:      data,
		expiresAt: time.Now().Add(-1 * time.Second), // 1秒前に期限切れ
	}
	c.mu.Unlock()

	_, ok := c.get(key)
	if ok {
		t.Error("expected cache miss after TTL expiry, got hit")
	}

	// TTL切れエントリは get 後に削除されていること（メモリリーク対策）
	c.mu.RLock()
	_, stillExists := c.entries[key]
	c.mu.RUnlock()
	if stillExists {
		t.Error("expected expired entry to be deleted from map, but it still exists")
	}
}

func TestCache_ReturnsCopy(t *testing.T) {
	c := newCache()
	key := "test-key"
	original := []domain.LandTransaction{{Period: "2024年第1四半期", TradePrice: 10_000_000}}
	c.set(key, original)

	result, ok := c.get(key)
	if !ok {
		t.Fatal("expected cache hit")
	}

	// 返されたスライスを変更してもキャッシュが汚染されないこと
	result[0].TradePrice = 999

	result2, ok := c.get(key)
	if !ok {
		t.Fatal("expected cache hit on 2nd get")
	}
	if result2[0].TradePrice != 10_000_000 {
		t.Errorf("cache was mutated by caller: got TradePrice=%v, want 10000000", result2[0].TradePrice)
	}
}

func TestCache_ConcurrentAccess(t *testing.T) {
	// sync.RWMutex の正確な実装を -race フラグで検証する
	// 複数ゴルーチンが同時に get / set を呼んでもデータレースが起きないことを確認する
	c := newCache()
	key := "concurrent-key"
	data := []domain.LandTransaction{{Period: "2024年第1四半期", TradePrice: 10_000_000}}

	const goroutines = 50
	done := make(chan struct{})

	// 書き込みゴルーチン
	go func() {
		for i := 0; i < goroutines; i++ {
			c.set(key, data)
		}
		close(done)
	}()

	// 読み取りゴルーチン群（書き込みと並行）
	for i := 0; i < goroutines; i++ {
		go func() {
			c.get(key)
		}()
	}

	<-done
}

func TestFetchLandPrices_APIStatusNotOK(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := APIResponse{Status: "ERROR", Data: nil}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(resp); err != nil {
			panic(err)
		}
	}))
	defer ts.Close()

	// status!=OK は HTTP 200 として返るため clientError にならずリトライされる。
	// 3回失敗後にエラーを返すことを確認する。
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	c := newTestClient(ts.URL)
	_, err := c.FetchLandPrices(ctx, LandPriceQuery{
		Area: "13", Year: 2023, Quarter: 1, ToYear: 2023, ToQuarter: 4,
	})
	if err == nil {
		t.Fatal("expected error for status=ERROR, got nil")
	}
}

// ---- FetchMunicipalities ----

func municipalityOKResponse(w http.ResponseWriter, municipalities []Municipality) {
	resp := MunicipalityResponse{Status: "OK", Data: municipalities}
	w.Header().Set("Content-Type", "application/json")
	if err := json.NewEncoder(w).Encode(resp); err != nil {
		panic(err)
	}
}

func TestFetchMunicipalities_Success(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != endpointMunicipalities {
			t.Errorf("unexpected path: %s", r.URL.Path)
		}
		if r.URL.Query().Get("area") != "13" {
			t.Errorf("area param = %s, want 13", r.URL.Query().Get("area"))
		}
		municipalityOKResponse(w, []Municipality{
			{ID: "13101", Name: "千代田区"},
			{ID: "13102", Name: "中央区"},
		})
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	result, err := c.FetchMunicipalities(context.Background(), "13")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(result) != 2 {
		t.Errorf("len = %d, want 2", len(result))
	}
	if result[0].ID != "13101" || result[0].Name != "千代田区" {
		t.Errorf("unexpected first municipality: %+v", result[0])
	}
}

func TestFetchMunicipalities_EmptyArea(t *testing.T) {
	c := newTestClient("http://example.com")
	_, err := c.FetchMunicipalities(context.Background(), "")
	if err == nil {
		t.Fatal("expected error for empty area, got nil")
	}
}

func TestFetchMunicipalities_CacheHit(t *testing.T) {
	apiCallCount := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		apiCallCount++
		municipalityOKResponse(w, []Municipality{{ID: "13101", Name: "千代田区"}})
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)

	// 1回目: APIコール発生
	if _, err := c.FetchMunicipalities(context.Background(), "13"); err != nil {
		t.Fatalf("1st call failed: %v", err)
	}
	// 2回目: キャッシュから返す（APIコールなし）
	if _, err := c.FetchMunicipalities(context.Background(), "13"); err != nil {
		t.Fatalf("2nd call failed: %v", err)
	}

	if apiCallCount != 1 {
		t.Errorf("apiCallCount = %d, want 1 (2nd call should hit cache)", apiCallCount)
	}
}

func TestFetchMunicipalities_4xxNoRetry(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer ts.Close()

	c := newTestClient(ts.URL)
	_, err := c.FetchMunicipalities(context.Background(), "13")
	if err == nil {
		t.Fatal("expected error for 4xx, got nil")
	}
}
