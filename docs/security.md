# セキュリティ設計

## 内部通信認証（APP_INTERNAL_API_KEY）

### 背景

Cloud Run バックエンドの URL を知っている者が Vercel を経由せずに `/api/*` を直接呼び出せる状態を防ぐため、Vercel-Cloud Run 間に共有シークレットによる内部認証を設けている。

### 仕組み

```
ブラウザ → Vercel (Next.js) → Cloud Run (Go/Gin)
                ↓ middleware.ts が付与
          X-Internal-Key: <APP_INTERNAL_API_KEY>
```

1. Vercel の `frontend/src/middleware.ts` が `/api/:path*` へのリクエストに `X-Internal-Key` ヘッダーを自動付与する
2. Cloud Run の `internalKeyMiddleware`（`backend/internal/api/router.go`）が `/api/*` グループでヘッダーを検証する
3. キーが不一致または未送信の場合は `401 Unauthorized` を返す

### 適用範囲

| エンドポイント | 認証 |
|---|---|
| `/api/*` | `APP_INTERNAL_API_KEY` 設定時に必須 |
| `/health` | スキップ（ヘルスチェックは認証不要） |

### ローカル開発

`APP_INTERNAL_API_KEY` を未設定（空）にするとミドルウェアは検証をスキップする。`docker-compose` / `make dev` での動作は変わらない。

### 本番環境でのキー設定

Vercel と Cloud Run の双方に **同一の値** を設定する。

```bash
# キー生成例
openssl rand -hex 32
```

- Vercel: 環境変数 `APP_INTERNAL_API_KEY` に設定
- Cloud Run: Secret Manager 経由で注入（`terraform apply` 時に `app_internal_api_key` 変数として設定）

Secret Manager への手動登録が必要な場合:
```bash
echo -n "your-key-value" | gcloud secrets versions add app-internal-api-key-prod --data-file=-
```

---

## Workload Identity Federation（OIDC パスワードレスデプロイ）

GitHub Actions から Cloud Run へのデプロイに静的な GCP 認証情報を使わない。GitHub の OIDC トークンを GCP の短命アクセストークンに交換する Workload Identity Federation（WIF）を使用している。

### 仕組み

```
GitHub Actions
  └─► GitHub OIDC トークン発行
        └─► WIF Pool/Provider でトークン検証
              └─► 短命 GCP アクセストークン発行
                    └─► SA impersonation で Artifact Registry / Cloud Run を操作
```

- `WIF_PROVIDER`・`SA_EMAIL` の 2 つの Secrets のみ GitHub に登録すれば良い
- 静的なサービスアカウントキー（JSON）は一切不要
- `attribute_condition` により、トークン交換は `glkt3912/yield-guard` リポジトリのみに制限

### GitHub Secrets

| Secret 名 | 値の取得方法 |
|---|---|
| `WIF_PROVIDER` | `terraform output -raw wif_provider` |
| `SA_EMAIL` | `terraform output -raw sa_email` |
| `GCP_PROJECT_ID` | GCP プロジェクト ID |

---

## SHA 固定 GitHub Actions

`.github/workflows/deploy-backend.yml` 内のすべてのサードパーティ Action はコミット SHA で固定している。バージョンタグはミュータブルであり、タグを悪意あるコミットに移動させるサプライチェーン攻撃を防ぐためである。

| Action | SHA | バージョン |
|---|---|---|
| `actions/checkout` | `11bd71901bbe5b1630ceea73d27597364c9af683` | v4.2.2 |
| `google-github-actions/auth` | `6fc4af4b145ae7821d527454aa9bd537d1f2dc5f` | v2.1.7 |
| `google-github-actions/setup-gcloud` | `6189d56e4096ee891640bb02ac264be376592d6a` | v2.1.4 |
| `google-github-actions/deploy-cloudrun` | `251330ba9a8a34bfbc1622895f42e1d53fd14522` | v2.7.6 |

---

## Secret Manager

`MLIT_API_KEY` と `APP_INTERNAL_API_KEY` は Google Secret Manager で管理し、Cloud Run 起動時に環境変数として注入される（`secretKeyRef` 参照）。平文の環境変数としてデプロイ設定に含まれることはない。

- Cloud Run のサービスアカウントには、プロジェクト全体ではなく **対象シークレットのみ** に `roles/secretmanager.secretAccessor` を付与（最小権限）
- Secret の値は `terraform apply` 時に変数として渡す、または `gcloud` CLI で手動登録する

```bash
echo -n "your-mlit-key" | gcloud secrets versions add mlit-api-key-prod --data-file=-
```

---

## Artifact Registry クリーンアップポリシー

Artifact Registry の無料枠（0.5 GB/月）を超えないよう、最新 5 件のイメージのみを保持するクリーンアップポリシーを設定している。Go イメージは 1 枚 ~20 MB 程度のため、5 枚で約 100 MB に収まる。

- `keep-5-most-recent`: 最新 5 件を KEEP
- `delete-old`: TAGGED イメージのうち上記に含まれないものを DELETE

設定は `terraform/artifact_registry.tf` で管理。

---

## 関連ファイル

- `backend/internal/api/router.go` — `internalKeyMiddleware()`
- `frontend/src/middleware.ts` — ヘッダー付与
- `backend/internal/api/handler_test.go` — ミドルウェアの単体テスト
- `terraform/iam.tf` — Workload Identity Federation + SA 最小権限
- `terraform/secret_manager.tf` — Secret Manager リソース
- `terraform/cloud_run.tf` — Cloud Run v2 サービス定義
- `.github/workflows/deploy-backend.yml` — デプロイワークフロー（SHA 固定）
