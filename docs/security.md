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

2026年3月の Trivy サプライチェーン攻撃（CVE-2026-33634）では `trivy-action` のバージョンタグがマルウェアを指すように書き換えられた。この設計によりその手法を無効化できる。

| Action | SHA | バージョン |
|---|---|---|
| `actions/checkout` | `de0fac2e4500dabe0009e67214ff5f5447ce83dd` | v6.0.2 |
| `google-github-actions/auth` | `7c6bc770dae815cd3e89ee6cdf493a5fab2cc093` | v3.0.0 |
| `google-github-actions/setup-gcloud` | `aa5489c8933f4cc7a4f7d45035b3b1440c9c10db` | v3.0.1 |
| `docker/setup-buildx-action` | `4d04d5d9486b7bd6fa91e7baf45bbb4f8b9deedd` | v4.0.0 |
| `actions/cache` | `27d5ce7f107fe9357f9df03efb73ab90386fccae` | v5.0.5 |
| `aquasecurity/trivy-action` | `ed142fd0673e97e23eac54620cfb913e5ce36c25` | v0.36.0 |
| `github/codeql-action/upload-sarif` | `e46ed2cbd01164d986452f91f178727624ae40d7` | v4.35.3 |
| `google-github-actions/deploy-cloudrun` | `2028e2d7d30a78c6910e0632e48dd561b064884d` | v3.0.1 |

> SHA は Dependabot（`github-actions` エコシステム）が週次で更新 PR を自動作成する（`.github/dependabot.yml`）。

---

## Trivy コンテナ脆弱性スキャン

### 背景

`deploy-backend.yml` は Docker イメージをビルドして Cloud Run へデプロイするが、OS パッケージや Go モジュールに既知の脆弱性（CVE）が含まれていても lint・test では検知できない。Trivy スキャンをビルドと push の間に挿入することで、脆弱なイメージがリリースされる前に CI で検知・停止する。

### フロー

```
docker buildx build（--load でローカルに展開）
    ↓
Trivy スキャン（CRITICAL のみブロック）
    ↓ 常時
SARIF → GitHub Security タブ
    ↓ スキャン通過時のみ
docker push → Cloud Run deploy
```

`--load` を使いローカル Docker daemon にイメージを展開してからスキャンする。`--push` と組み合わせないのは、脆弱なイメージを Artifact Registry に残さないためである。

### スキャン設定

| 設定 | 値 | 理由 |
|------|-----|------|
| `severity` | `CRITICAL` | CVSS 9.0+ のみブロック。HIGH は SARIF に記録するが通過させる |
| `ignore-unfixed` | `true` | パッチ未提供の CVE は対応不能なため CI ブロックから除外 |
| `scanners` | `vuln` | 脆弱性スキャンのみ（OS パッケージ・言語ライブラリ対象） |
| `format` | `sarif` | GitHub Security タブへのアップロードに使用 |

### Trivy サプライチェーン攻撃（CVE-2026-33634）

2026年3月19日、脅威アクター TeamPCP が Aqua Security のリリースインフラを侵害し、`trivy-action` の 76/77 バージョンタグをクレデンシャル窃取マルウェアに書き換えた。GitHub Actions runner の `/proc/<pid>/mem` を直接読み取りログマスキングをバイパスして secrets を盗取する手法が使われた。

yield-guard はインシデント発生時点で Trivy を未使用だったため影響はなかった。現在は commit SHA 固定により同手法の再発を防いでいる。

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

## Terraform CI（terraform-ci.yml）

`.github/workflows/terraform-ci.yml` は PR 時に `terraform fmt` / `validate` / `tflint` / `plan` を実行し、`workflow_dispatch` の `apply: true` 指定時のみ `terraform apply` を行う。

### 参照する Secrets / Variables

| 名前 | 種別 | 用途 |
|------|------|------|
| `GCP_PROJECT_ID` | Secret | `TF_VAR_project_id`（GCP プロジェクト ID） |
| `MLIT_API_KEY` | Secret | `TF_VAR_mlit_api_key` |
| `APP_INTERNAL_API_KEY` | Secret | `TF_VAR_app_internal_api_key` |
| `WIF_PROVIDER` | Secret | OIDC 認証用 WIF プロバイダー |
| `SA_EMAIL` | Secret | OIDC 認証用 deployer SA |
| `VERCEL_FRONTEND_URL` | Variable | `TF_VAR_vercel_frontend_url`（CORS 許可オリジン） |
| `NOTIFICATION_EMAIL` | Variable | `TF_VAR_notification_email`（Cloud Monitoring アラート通知先） |

> `TF_VAR_env`（`"prod"`）・`TF_VAR_region`（`"asia-northeast1"`）はワークフロー内にハードコードされており、Secrets / Variables の設定は不要。

Variable（非機密）は `gh variable set`、Secret（機密）は `gh secret set` または GitHub UI で設定する。

```bash
# Variable の設定例
gh variable set NOTIFICATION_EMAIL --body "your@email.com"
gh variable set VERCEL_FRONTEND_URL --body "https://your-app.vercel.app"

# Secret の設定例
gh secret set GCP_PROJECT_ID --body "your-gcp-project-id"
gh secret set MLIT_API_KEY --body "your-mlit-api-key"
```

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
- `.github/workflows/deploy-backend.yml` — バックエンドデプロイワークフロー（SHA 固定・Trivy スキャン）
- `.github/workflows/frontend-ci.yml` — フロントエンド CI + Vercel デプロイワークフロー
- `.github/dependabot.yml` — Dependabot 自動更新設定（gomod / npm / github-actions）
- `backend/internal/telemetry/setup.go` — OTel TracerProvider / MeterProvider 初期化
- `backend/internal/logger/logger.go` — Cloud Logging 準拠 slog ハンドラー
