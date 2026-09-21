package api

import (
	"context"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"google.golang.org/api/idtoken"
)

// TokenValidator は Google 発行の OIDC ID トークンを検証する
type TokenValidator interface {
	Validate(ctx context.Context, token, audience string) (*idtoken.Payload, error)
}

type idtokenValidator struct{}

func (idtokenValidator) Validate(ctx context.Context, token, audience string) (*idtoken.Payload, error) {
	return idtoken.Validate(ctx, token, audience)
}

// WarmupAuth は /warm を呼び出せる Cloud Scheduler の OIDC 設定
// Audience・InvokerEmail が空の場合、ローカル開発では素通し、release モードでは全拒否する
type WarmupAuth struct {
	Validator    TokenValidator // nil の場合は idtoken.Validate を使う
	Audience     string
	InvokerEmail string
}

func (a WarmupAuth) configured() bool {
	return a.Audience != "" && a.InvokerEmail != ""
}

// schedulerOIDCMiddleware は Cloud Scheduler の OIDC トークンを検証する
// Cloud Run は allUsers に公開しているため、IAM ではなくアプリ側で検証する
func schedulerOIDCMiddleware(auth WarmupAuth) gin.HandlerFunc {
	if !auth.configured() {
		return func(c *gin.Context) {
			if gin.Mode() == gin.ReleaseMode {
				c.AbortWithStatus(http.StatusUnauthorized)
				return
			}
			c.Next()
		}
	}
	v := auth.Validator
	if v == nil {
		v = idtokenValidator{}
	}
	return func(c *gin.Context) {
		token, ok := strings.CutPrefix(c.GetHeader("Authorization"), "Bearer ")
		if !ok || token == "" {
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		payload, err := v.Validate(c.Request.Context(), token, auth.Audience)
		if err != nil {
			slog.Warn("warmup OIDC token rejected", "err", err)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		email, _ := payload.Claims["email"].(string)
		verified, _ := payload.Claims["email_verified"].(bool)
		if email != auth.InvokerEmail || !verified {
			slog.Warn("warmup OIDC token from unexpected principal", "email", email)
			c.AbortWithStatus(http.StatusUnauthorized)
			return
		}
		c.Next()
	}
}
