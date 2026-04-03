package mlit

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
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

// ---- isLandType ----

func TestIsLandType(t *testing.T) {
	tests := []struct {
		input string
		want  bool
	}{
		{"宅地(土地)", true},
		{"宅地のみ", false},   // "土地" を含まない
		{"土地のみ", false},   // "宅地" を含まない
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

// ---- buildURL ----

func TestBuildURL(t *testing.T) {
	t.Run("area が空のときエラー", func(t *testing.T) {
		_, err := buildURL(LandPriceQuery{From: "20231", To: "20234"})
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("from が空のときエラー", func(t *testing.T) {
		_, err := buildURL(LandPriceQuery{Area: "13", To: "20234"})
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("to が空のときエラー", func(t *testing.T) {
		_, err := buildURL(LandPriceQuery{Area: "13", From: "20231"})
		if err == nil {
			t.Error("expected error, got nil")
		}
	})

	t.Run("必須パラメータが揃っているとき URL を生成する", func(t *testing.T) {
		got, err := buildURL(LandPriceQuery{Area: "13", From: "20231", To: "20234"})
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if got == "" {
			t.Error("expected non-empty URL")
		}
		// area / from / to がクエリに含まれること
		for _, param := range []string{"area=13", "from=20231", "to=20234"} {
			if !strings.Contains(got, param) {
				t.Errorf("URL %q does not contain %q", got, param)
			}
		}
	})

	t.Run("city が指定されているときクエリに含まれる", func(t *testing.T) {
		got, err := buildURL(LandPriceQuery{Area: "13", From: "20231", To: "20234", City: "13101"})
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

func TestFetchLandPrices_RetryOn5xx(t *testing.T) {
	attempt := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt++
		if attempt < 3 {
			w.WriteHeader(http.StatusServiceUnavailable) // 5xx
			return
		}
		okResponse(w)
	}))
	defer ts.Close()

	origBaseURL := baseURL
	baseURL = ts.URL
	defer func() { baseURL = origBaseURL }()

	c := &Client{httpClient: &http.Client{}}
	// retryBaseDelay=1s のため遅延が発生するが許容範囲内
	result, err := c.FetchLandPrices(context.Background(), LandPriceQuery{
		Area: "13", From: "20231", To: "20234",
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

	origBaseURL := baseURL
	baseURL = ts.URL
	defer func() { baseURL = origBaseURL }()

	c := &Client{httpClient: &http.Client{}}
	_, err := c.FetchLandPrices(context.Background(), LandPriceQuery{
		Area: "13", From: "20231", To: "20234",
	})
	if err == nil {
		t.Fatal("expected error after all retries, got nil")
	}
}

func TestFetchLandPrices_NoRetryOn4xx(t *testing.T) {
	attempt := 0
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attempt++
		w.WriteHeader(http.StatusBadRequest) // 4xx
	}))
	defer ts.Close()

	origBaseURL := baseURL
	baseURL = ts.URL
	defer func() { baseURL = origBaseURL }()

	c := &Client{httpClient: &http.Client{}}
	_, err := c.FetchLandPrices(context.Background(), LandPriceQuery{
		Area: "13", From: "20231", To: "20234",
	})
	if err == nil {
		t.Fatal("expected error, got nil")
	}
	if attempt != 1 {
		t.Errorf("attempt = %d, want 1 (no retry on 4xx)", attempt)
	}
}

func TestFetchLandPrices_ContextCancel(t *testing.T) {
	ts := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
	}))
	defer ts.Close()

	origBaseURL := baseURL
	baseURL = ts.URL
	defer func() { baseURL = origBaseURL }()

	ctx, cancel := context.WithCancel(context.Background())
	// 1回目が終わったらすぐキャンセル
	go func() { cancel() }()

	c := &Client{httpClient: &http.Client{}}
	_, err := c.FetchLandPrices(ctx, LandPriceQuery{
		Area: "13", From: "20231", To: "20234",
	})
	if err == nil {
		t.Fatal("expected error after context cancel, got nil")
	}
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

	origBaseURL := baseURL
	baseURL = ts.URL
	defer func() { baseURL = origBaseURL }()

	c := &Client{httpClient: &http.Client{}}
	_, err := c.FetchLandPrices(context.Background(), LandPriceQuery{
		Area: "13", From: "20231", To: "20234",
	})
	if err == nil {
		t.Fatal("expected error for status=ERROR, got nil")
	}
}

