package api

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"google.golang.org/api/idtoken"
)

const (
	testAudience = "yield-guard-warmup-test"
	testInvoker  = "sa-scheduler@example.iam.gserviceaccount.com"
)

type fakeValidator struct {
	claims map[string]any
	err    error
	// 検証時に渡された値
	gotToken    string
	gotAudience string
}

func (f *fakeValidator) Validate(_ context.Context, token, audience string) (*idtoken.Payload, error) {
	f.gotToken = token
	f.gotAudience = audience
	if f.err != nil {
		return nil, f.err
	}
	return &idtoken.Payload{Audience: audience, Claims: f.claims}, nil
}

func serveWarm(t *testing.T, auth WarmupAuth, authHeader string) int {
	t.Helper()
	r := gin.New()
	r.POST("/warm", schedulerOIDCMiddleware(auth), func(c *gin.Context) { c.Status(http.StatusOK) })
	req := httptest.NewRequest(http.MethodPost, "/warm", nil)
	if authHeader != "" {
		req.Header.Set("Authorization", authHeader)
	}
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)
	return w.Code
}

func TestSchedulerOIDCMiddleware(t *testing.T) {
	validClaims := map[string]any{"email": testInvoker, "email_verified": true}

	tests := []struct {
		name       string
		validator  *fakeValidator
		authHeader string
		want       int
	}{
		{"Authorization ヘッダーなし", &fakeValidator{claims: validClaims}, "", http.StatusUnauthorized},
		{"Bearer 以外のスキーム", &fakeValidator{claims: validClaims}, "Basic abc", http.StatusUnauthorized},
		{"トークン検証失敗", &fakeValidator{err: errors.New("invalid")}, "Bearer tok", http.StatusUnauthorized},
		{"email 不一致", &fakeValidator{claims: map[string]any{"email": "other@example.com", "email_verified": true}}, "Bearer tok", http.StatusUnauthorized},
		{"email_verified=false", &fakeValidator{claims: map[string]any{"email": testInvoker, "email_verified": false}}, "Bearer tok", http.StatusUnauthorized},
		{"正常", &fakeValidator{claims: validClaims}, "Bearer tok", http.StatusOK},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			auth := WarmupAuth{Validator: tt.validator, Audience: testAudience, InvokerEmail: testInvoker}
			if got := serveWarm(t, auth, tt.authHeader); got != tt.want {
				t.Errorf("status = %d, want %d", got, tt.want)
			}
		})
	}
}

func TestSchedulerOIDCMiddleware_PassesTokenAndAudience(t *testing.T) {
	v := &fakeValidator{claims: map[string]any{"email": testInvoker, "email_verified": true}}
	serveWarm(t, WarmupAuth{Validator: v, Audience: testAudience, InvokerEmail: testInvoker}, "Bearer tok123")

	if v.gotToken != "tok123" {
		t.Errorf("token = %q, want %q", v.gotToken, "tok123")
	}
	if v.gotAudience != testAudience {
		t.Errorf("audience = %q, want %q", v.gotAudience, testAudience)
	}
}

func TestSchedulerOIDCMiddleware_Unconfigured(t *testing.T) {
	prev := gin.Mode()
	t.Cleanup(func() { gin.SetMode(prev) })

	// ローカル開発（debug モード）では素通し
	gin.SetMode(gin.DebugMode)
	if got := serveWarm(t, WarmupAuth{}, ""); got != http.StatusOK {
		t.Errorf("debug mode: status = %d, want %d", got, http.StatusOK)
	}

	// release モードでは設定漏れを fail-closed で拒否
	gin.SetMode(gin.ReleaseMode)
	if got := serveWarm(t, WarmupAuth{}, "Bearer tok"); got != http.StatusUnauthorized {
		t.Errorf("release mode: status = %d, want %d", got, http.StatusUnauthorized)
	}
}
