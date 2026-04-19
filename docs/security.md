# セキュリティ設計

## 内部通信認証（APP_INTERNAL_API_KEY）

### 背景

Render バックエンドの URL を知っている者が Vercel を経由せずに `/api/*` を直接呼び出せる状態を防ぐため、Vercel-Render 間に共有シークレットによる内部認証を設けている。

### 仕組み

```
ブラウザ → Vercel (Next.js) → Render (Go/Gin)
                ↓ middleware.ts が付与
          X-Internal-Key: <APP_INTERNAL_API_KEY>
```

1. Vercel の `frontend/src/middleware.ts` が `/api/:path*` へのリクエストに `X-Internal-Key` ヘッダーを自動付与する
2. Render の `internalKeyMiddleware`（`backend/internal/api/router.go`）が `/api/*` グループでヘッダーを検証する
3. キーが不一致または未送信の場合は `401 Unauthorized` を返す

### 適用範囲

| エンドポイント | 認証 |
|---|---|
| `/api/*` | `APP_INTERNAL_API_KEY` 設定時に必須 |
| `/health` | スキップ（ヘルスチェックは認証不要） |

### ローカル開発

`APP_INTERNAL_API_KEY` を未設定（空）にするとミドルウェアは検証をスキップする。`docker-compose` / `make dev` での動作は変わらない。

### 本番環境でのキー設定

Vercel と Render の双方に **同一の値** を設定する。

```bash
# キー生成例
openssl rand -hex 32
```

- Vercel: 環境変数 `APP_INTERNAL_API_KEY` に設定
- Render: 環境変数 `APP_INTERNAL_API_KEY` に設定
- Terraform (#113) で管理する場合は `app_internal_api_key` 変数として注入する

### 関連ファイル

- `backend/internal/api/router.go` — `internalKeyMiddleware()`
- `frontend/src/middleware.ts` — ヘッダー付与
- `backend/internal/api/handler_test.go` — ミドルウェアの単体テスト
