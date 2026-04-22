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

## フロントエンド GitHub Actions デプロイ

`.github/workflows/frontend-ci.yml` の `deploy` ジョブは以下のセキュリティ設計を採用している。

| 項目 | 設計 |
|---|---|
| トークン渡し方 | `VERCEL_TOKEN` を `env:` 経由で渡す（`--token=` 引数は使用しない。プロセス一覧への露出を防ぐため） |
| Vercel CLI バージョン | `vercel@51.8.0` に固定（`latest` 不使用） |
| 実行条件 | `needs: ci` により CI 全通過後のみ実行 |
| 実行タイミング | `github.ref == 'refs/heads/main' && github.event_name == 'push'` により main マージ時のみ |
| 権限 | `permissions: contents: read`（最小権限） |

### GitHub Secrets

| Secret 名 | 用途 |
|---|---|
| `VERCEL_TOKEN` | Vercel API 認証トークン（Vercel ダッシュボードで発行） |
| `VERCEL_ORG_ID` | Vercel 組織 ID（`.vercel/project.json` の `orgId`） |
| `VERCEL_PROJECT_ID` | Vercel プロジェクト ID（`.vercel/project.json` の `projectId`） |

---

## OpenTelemetry 観測基盤のセキュリティ考慮

### OTEL_EXPORTER_OTLP_ENDPOINT

`OTEL_EXPORTER_OTLP_ENDPOINT` はシークレットではなく、通常の環境変数として Cloud Run に設定する（Secret Manager 管理不要）。ただし、エンドポイント URL 自体がインフラ構成情報を含む場合は適宜アクセス制御を検討すること。

| 環境 | 値 | 管理方法 |
|---|---|---|
| ローカル開発 | 未設定（stdout 出力） | 不要 |
| Cloud Run 本番 | OTLP gRPC エンドポイント | Cloud Run 環境変数（Terraform） |

### stdout エクスポーターのデータスコープ（ローカル開発）

`OTEL_EXPORTER_OTLP_ENDPOINT` が未設定の場合、トレース・メトリクスは **stdout（コンソール）** に出力される。出力される情報は以下の通り。

- スパン名・開始/終了時刻・ステータス
- スパン属性（下記「スパン属性ポリシー」参照）
- メトリクス計測値（カウンター・ヒストグラム）

ログイン資格情報・APIキー・個人情報はスパン属性に含まれない設計としている。

### スパン属性ポリシー（mlit.query.* 属性）

MLIT API クライアント（`backend/internal/mlit/client.go`）は以下の属性をスパンに付与する。

| 属性名 | 例 | 分類 |
|---|---|---|
| `mlit.endpoint` | `"XIT001"` | API エンドポイント識別子（非機密） |
| `mlit.query.area` | `"13"` | 都道府県コード（非機密） |
| `mlit.cache.hit` | `true` / `false` | キャッシュ状態（非機密） |
| `mlit.retry.count` | `1` | リトライ回数（非機密） |
| `server.address` | `"www.reinfolib.mlit.go.jp"` | OTel セマンティクス準拠 |

**注意:** クエリパラメータに含まれる値（都道府県コード・タイル座標等）は公開情報のみであり、個人を特定できる情報（PII）はスパン属性として記録しない。

### GOOGLE_CLOUD_PROJECT と Cloud Logging トレース相関

`GOOGLE_CLOUD_PROJECT` はシークレットではなく、GCP プロジェクト ID（公開情報）を設定する。この値は Cloud Logging のトレースリンク URL 生成（`projects/<id>/traces/<traceId>`）にのみ使用される。未設定の場合、トレースフィールドは JSON ログに含まれず、Cloud Logging との相関表示が無効になる（ローカル開発では意図的に未設定とする）。

---

## 関連ファイル

- `backend/internal/api/router.go` — `internalKeyMiddleware()`
- `frontend/src/middleware.ts` — ヘッダー付与
- `backend/internal/api/handler_test.go` — ミドルウェアの単体テスト
- `terraform/iam.tf` — Workload Identity Federation + SA 最小権限
- `terraform/secret_manager.tf` — Secret Manager リソース
- `terraform/cloud_run.tf` — Cloud Run v2 サービス定義
- `.github/workflows/deploy-backend.yml` — バックエンドデプロイワークフロー（SHA 固定）
- `.github/workflows/frontend-ci.yml` — フロントエンド CI + Vercel デプロイワークフロー
- `backend/internal/telemetry/setup.go` — OTel TracerProvider / MeterProvider 初期化
- `backend/internal/logger/logger.go` — Cloud Logging 準拠 slog ハンドラー
