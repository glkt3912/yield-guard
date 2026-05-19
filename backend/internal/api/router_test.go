package api

import (
	"net/http"
	"net/http/httptest"
	"testing"
)

// --- CORS ヘッダー ---

func TestRouter_CORS_DefaultOrigin(t *testing.T) {
	// ALLOW_ORIGINS 未設定 → デフォルト http://localhost:3000 が許可される
	t.Setenv("ALLOW_ORIGINS", "")

	r := newTestRouter(&mockMLITClient{}, &mockGeocodeClient{})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	got := w.Header().Get("Access-Control-Allow-Origin")
	if got != "http://localhost:3000" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "http://localhost:3000")
	}
}

func TestRouter_CORS_CustomOrigin(t *testing.T) {
	// ALLOW_ORIGINS に設定したオリジンが許可される
	t.Setenv("ALLOW_ORIGINS", "http://app.example.com")

	r := newTestRouter(&mockMLITClient{}, &mockGeocodeClient{})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Origin", "http://app.example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	got := w.Header().Get("Access-Control-Allow-Origin")
	if got != "http://app.example.com" {
		t.Errorf("Access-Control-Allow-Origin = %q, want %q", got, "http://app.example.com")
	}
}

func TestRouter_CORS_UnknownOriginNotReflected(t *testing.T) {
	// 許可リストにないオリジンはヘッダーに反映されない
	t.Setenv("ALLOW_ORIGINS", "http://app.example.com")

	r := newTestRouter(&mockMLITClient{}, &mockGeocodeClient{})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	req.Header.Set("Origin", "http://evil.example.com")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	got := w.Header().Get("Access-Control-Allow-Origin")
	if got == "http://evil.example.com" {
		t.Errorf("unlisted origin was reflected in Access-Control-Allow-Origin")
	}
}

func TestRouter_CORS_Preflight(t *testing.T) {
	// OPTIONS プリフライトリクエストに対して許可メソッドが返る
	t.Setenv("ALLOW_ORIGINS", "http://localhost:3000")

	r := newTestRouter(&mockMLITClient{}, &mockGeocodeClient{})

	req := httptest.NewRequest(http.MethodOptions, "/health", nil)
	req.Header.Set("Origin", "http://localhost:3000")
	req.Header.Set("Access-Control-Request-Method", "GET")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	methods := w.Header().Get("Access-Control-Allow-Methods")
	if methods == "" {
		t.Error("Access-Control-Allow-Methods header is missing in preflight response")
	}
}

func TestRouter_CORS_MultipleOrigins(t *testing.T) {
	// カンマ区切りで複数オリジンを許可できる
	t.Setenv("ALLOW_ORIGINS", "http://a.example.com,http://b.example.com")

	r := newTestRouter(&mockMLITClient{}, &mockGeocodeClient{})

	for _, origin := range []string{"http://a.example.com", "http://b.example.com"} {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		req.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		got := w.Header().Get("Access-Control-Allow-Origin")
		if got != origin {
			t.Errorf("origin %q: Access-Control-Allow-Origin = %q, want %q", origin, got, origin)
		}
	}
}

func TestRouter_CORS_WildcardSuffix(t *testing.T) {
	// *.vercel.app 形式はサフィックス一致で許可される
	t.Setenv("ALLOW_ORIGINS", "https://prod.example.com,*.vercel.app")

	r := newTestRouter(&mockMLITClient{}, &mockGeocodeClient{})

	allowed := []string{
		"https://prod.example.com",
		"https://frontend-git-feat-xxx-team.vercel.app",
		"https://myapp.vercel.app",
	}
	for _, origin := range allowed {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		req.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		got := w.Header().Get("Access-Control-Allow-Origin")
		if got != origin {
			t.Errorf("origin %q should be allowed, got Access-Control-Allow-Origin = %q", origin, got)
		}
	}

	denied := []string{
		"https://evil.com",
		"https://notvercel.app",
		"https://evil.vercel.app.evil.com",
	}
	for _, origin := range denied {
		req := httptest.NewRequest(http.MethodGet, "/health", nil)
		req.Header.Set("Origin", origin)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		got := w.Header().Get("Access-Control-Allow-Origin")
		if got == origin {
			t.Errorf("origin %q should be denied, but was allowed", origin)
		}
	}
}

// --- ルート登録 ---

func TestRouter_HealthEndpoint(t *testing.T) {
	r := newTestRouter(&mockMLITClient{}, &mockGeocodeClient{})

	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("GET /health status = %d, want 200", w.Code)
	}
}

func TestRouter_UnknownRoute_Returns404(t *testing.T) {
	r := newTestRouter(&mockMLITClient{}, &mockGeocodeClient{})

	req := httptest.NewRequest(http.MethodGet, "/no-such-route", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("unknown route status = %d, want 404", w.Code)
	}
}

// --- 認証ミドルウェア ---

func TestRouter_InternalKey_Required_WhenSet(t *testing.T) {
	// APP_INTERNAL_API_KEY が設定されているとき、キーなしリクエストは 401
	t.Setenv("APP_INTERNAL_API_KEY", "secret-key")

	r := newTestRouter(&mockMLITClient{}, &mockGeocodeClient{})

	req := httptest.NewRequest(http.MethodGet, "/api/municipalities?area=13", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("status = %d, want 401 when key is missing", w.Code)
	}
}

func TestRouter_InternalKey_Accepted_WhenCorrect(t *testing.T) {
	// 正しいキーを付けると認証を通過して 200 が返る
	t.Setenv("APP_INTERNAL_API_KEY", "secret-key")

	r := newTestRouter(&mockMLITClient{}, &mockGeocodeClient{})

	req := httptest.NewRequest(http.MethodGet, "/api/municipalities?area=13", nil)
	req.Header.Set("X-Internal-Key", "secret-key")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("correct key: status = %d, want 200", w.Code)
	}
}

func TestRouter_InternalKey_NotRequired_WhenUnset(t *testing.T) {
	// APP_INTERNAL_API_KEY 未設定のときはキーなしでも 200 が返る
	t.Setenv("APP_INTERNAL_API_KEY", "")

	r := newTestRouter(&mockMLITClient{}, &mockGeocodeClient{})

	req := httptest.NewRequest(http.MethodGet, "/api/municipalities?area=13", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("no key set: status = %d, want 200", w.Code)
	}
}
