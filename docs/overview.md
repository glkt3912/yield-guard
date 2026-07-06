---
purpose: プロジェクト全体像・システム構成の概要
triggers: [overview, project, system, architecture overview]
audience: all
token_weight: medium
---

# プロジェクト概要とアーキテクチャ

## ツールの目的

Yield-Guard は不動産投資判断を支援するシミュレーションツール。
以下の5つの判断を1ツールで完結させる。

| 機能 | 説明 |
|------|------|
| **相場判定** | 国交省実取引データと比較して土地価格が「割安/相場/割高」かを判定 |
| **8%境界線** | 表面利回り8%未達時に「いくら値引きが必要か / いくら賃料が必要か」を逆算 |
| **デッドクロス予測** | 元金返済 > 減価償却費になる年を特定し、黒字倒産リスクを可視化 |
| **出口戦略** | ホールディング後の収益還元法売却価格・譲渡所得税・最終手残りを試算 |
| **ストレステスト** | 空室率・金利のシナリオ変動による収支への影響を確認 |

---

## システム構成

```
[ブラウザ]
    ↓ http://localhost:3000
[フロントエンド] Next.js 16 (App Router)
    ↓ fetch to http://localhost:8080
[バックエンド] Go + Gin
    ↓ HTTPS + Ocp-Apim-Subscription-Key ヘッダー
[外部API] 国交省 不動産情報ライブラリ API (reinfolib.mlit.go.jp)
```

---

## 技術スタック

### バックエンド

| 技術 | 用途 |
|------|------|
| Go 1.25 | 言語 |
| Gin v1.9.1 | HTTPルーティング |
| gin-contrib/cors | CORS ミドルウェア |
| OpenTelemetry SDK | 分散トレーシング・メトリクス計装 |
| opentelemetry-operations-go | Cloud Trace / Cloud Monitoring へのエクスポート（本番） |

### フロントエンド

| 技術 | 用途 |
|------|------|
| Next.js 16 (App Router) | フレームワーク |
| React 19 | UI |
| TypeScript | 言語 |
| Tailwind CSS 4.2 | スタイリング |
| Shadcn/UI | コンポーネントライブラリ |
| Recharts 2.12 | グラフ描画 |
| Lucide React | アイコン |
| @ducanh2912/next-pwa | PWA（Service Worker / Web App Manifest） |

---

## ディレクトリ構成

パッケージ（ディレクトリ）単位で責務を示す。ファイル一覧は列挙しない（各パッケージ内は実コードを参照）。

```
yield-guard/
├── backend/
│   ├── Dockerfile                 # マルチステージビルド（BuildKitキャッシュ最適化）
│   ├── cmd/server/                # エントリポイント・swag メタアノテーション・グレースフルシャットダウン
│   └── internal/
│       ├── api/                   # HTTPハンドラ（パース→service委譲→ステータス変換）・ルーター・CORS・認証・レートリミット・ジオコーダー
│       ├── service/               # アプリケーションサービス層（投資スコア・エリア探索・都市リスク・土地価格・賃料などのユースケース）
│       ├── domain/                # ドメインモデルと計算ロジック（利回り・デッドクロス・出口・税・理論価格・投資スコア等）
│       ├── mlit/                  # 国交省APIクライアント（リトライ・インメモリキャッシュ・Firestore L2キャッシュ）
│       ├── ai/                    # Gemini による投資サマリー生成（GEMINI_API_KEY 未設定時は no-op）
│       ├── concurrent/            # 並行処理ヘルパー（FanOut / SafeCall）
│       ├── logger/                # slog 構造化ログ設定
│       └── telemetry/             # OpenTelemetry セットアップ・カスタムメトリクス
├── frontend/
│   ├── Dockerfile                 # マルチステージビルド
│   ├── e2e/                       # Playwright E2E（spec・fixtures・ページオブジェクト）
│   └── src/
│       ├── app/                   # App Router エントリ（layout / page / manifest）
│       ├── components/            # UI コンポーネント（Dashboard 起点。sections/・ui/=Shadcn を含む）
│       ├── hooks/                 # API 呼び出し・シミュレーション・ネットワーク状態等のカスタムフック
│       ├── lib/                   # API クライアント・キャッシュ・PDF 生成・ユーティリティ
│       └── types/                 # 手書き型（investment.ts）+ 生成型（api.generated.ts、git管理外・CI で毎回生成）
├── docs/                          # ドキュメント（docs-mcp-server 用）・openapi/swagger.json（コミット対象）
├── docker-compose.yml             # backend + frontend 一括起動
├── .env.example                   # 環境変数テンプレート
└── .mcp.json                      # docs-mcp-server 設定
```

---

## 開発サーバー起動手順

### Docker（推奨）

```bash
cp .env.example .env   # MLIT_API_KEY を設定する
make docker-up         # backend :8080 + frontend :3000 を一括起動
```

### ローカル開発

Makefile の便利ターゲットを使うと片方だけ起動できる:

```bash
make backend    # バックエンドのみ起動（:8080）
make frontend   # フロントエンドのみ起動（:3000）
make install    # フロントエンド依存関係インストール（npm install）
make logs       # Docker ログ表示（未起動時は案内メッセージ）
```

**バックエンド（手動）**

```bash
cd backend
go mod tidy
go run cmd/server/main.go
```

環境変数（プロジェクトルートの `.env` から読み込む）:

| 変数名 | 説明 | デフォルト |
|---|---|---|
| `PORT` | リッスンポート | `8080` |
| `ALLOW_ORIGINS` | CORS許可オリジン | `http://localhost:3000` |
| `MLIT_API_KEY` | 不動産情報ライブラリ APIキー（必須） | — |
| `APP_INTERNAL_API_KEY` | Vercel-Cloud Run 間の内部通信認証キー。設定時は `/api/*` に `X-Internal-Key` ヘッダーが必要 | 未設定（ローカル開発時はスキップ） |
| `GOOGLE_CLOUD_PROJECT` | GCP プロジェクト ID。設定時は Cloud Trace / Cloud Monitoring へ OTel データを送信。未設定時は stdout 出力 | 未設定（ローカル開発時は stdout） |

**フロントエンド**

```bash
cd frontend
npm install
npm run dev   # http://localhost:3000
```

フロントエンド環境変数（`frontend/.env.local`）:

| 変数名 | 説明 | デフォルト |
|---|---|---|
| `APP_INTERNAL_API_KEY` | バックエンドへの内部認証キー（バックエンドと同一値を設定） | 未設定（ローカル開発時はスキップ） |

---

## デプロイフロー

### フロントエンド（Vercel）

```
main ブランチへの push
    ↓
[GitHub Actions: frontend-ci.yml]
    ci ジョブ: Lint → 型チェック → テスト → ビルド（next build --webpack）
    ↓ 成功時のみ
    deploy ジョブ: vercel deploy --prod
    ↓
https://yield-guard-alpha.vercel.app
```

- CI が失敗した場合はデプロイしない（`needs: ci`）
- PR 時はデプロイしない（main push 時のみ）
- Vercel 環境変数: `BACKEND_URL`・`APP_INTERNAL_API_KEY`（Vercel ダッシュボードで設定済み）
- GitHub Secrets: `VERCEL_TOKEN`・`VERCEL_ORG_ID`・`VERCEL_PROJECT_ID`

### バックエンド（Cloud Run）

```
main ブランチへの push（backend/** 変更時）
    ↓
[GitHub Actions: deploy-backend.yml]
    OIDC 認証（Workload Identity Federation）
    ↓
    Docker ビルド → Artifact Registry push → Cloud Run デプロイ
```

詳細は `docs/security.md` を参照。

---

## 国交省API 開発用リクエスト（Makefile）

`.env` の `MLIT_API_KEY` を使って国交省APIへ直接リクエストできる開発者向けターゲット。`jq` が必要。

```bash
# 市区町村一覧（XIT002）
make mlit-municipalities area=13

# 土地取引価格（XIT001）
make mlit-land-prices area=13 year=2024 quarter=1 to_year=2024 to_quarter=4

# 市区町村コード指定あり
make mlit-land-prices area=13 year=2024 quarter=1 to_year=2024 to_quarter=4 city=13101
```

- APIキーは `.env`（git管理外）から自動で読み込む。コマンドに直接記載しない
- レスポンスは gzip 圧縮のため `--compressed` を付与している
- 必須パラメータが未指定の場合はエラーメッセージを表示して終了する

---

## テスト実行

```bash
# バックエンド（レースチェック付き・カバレッジ計測）
cd backend
go test -race -coverprofile=coverage.out ./...

# カバレッジ確認（テキスト形式）
go tool cover -func=coverage.out

# フロントエンド（Vitest ユニットテスト）
cd frontend
npm test

# フロントエンド（Playwright E2E テスト）
make e2e        # テスト実行（make dev 起動中なら既存サーバーを再利用）
make e2e-ui     # Playwright Inspector でステップ実行
make e2e-report # 最後のレポートをブラウザで表示
```

CI（GitHub Actions）では `go test -race -coverprofile=coverage.out` を実行し、カバレッジサマリーを GitHub Actions Job Summary に出力する。最新のカバレッジ率は README のバッジで確認できる。

### テスト構成

ディレクトリ単位で示す（個別ファイル・テスト数は列挙しない）。

| レイヤー | 場所 | ツール・方針 |
|---|---|---|
| ドメイン計算 | `backend/internal/domain/*_test.go` | go test（境界値テスト含む） |
| サービス層（ユースケース） | `backend/internal/service/*_test.go` | go test（MLIT クライアント・Summarizer をモック注入） |
| HTTP バインディング | `backend/internal/api/*_test.go` | go test / httptest（パラメータ検証・ステータス変換のみ） |
| MLIT クライアント | `backend/internal/mlit/*_test.go` | go test / httptest（`integration_test.go` のみ実API・要APIキー） |
| フロントエンド UI | `frontend/src/**/__tests__/` | Vitest + React Testing Library |
| API 型契約 | `frontend/src/types/__tests__/api-contract.ts` | tsc --noEmit（生成型と手書き型の整合を型レベルで検証） |
| E2E（ユーザーフロー） | `frontend/e2e/**/*.spec.ts` | Playwright + page.route() |

#### フロントエンドテストの方針

- **ツール**: [Vitest](https://vitest.dev/) v4.x + [React Testing Library](https://testing-library.com/react)
- **環境**: jsdom（ブラウザAPI をエミュレート）
- **JSX 変換**: vitest 内蔵の oxc で処理（v4 で esbuild から移行）
- **モック**: `ResizeObserver`（Recharts が要求）、APIコールは `vi.fn()` で差し替え
- **テスト対象コンポーネント**:
  - `Dashboard`: 状態管理・API呼び出しフロー・コンポーネント統合
  - `CostBreakdown`: 初期投資内訳・取得時諸経費・年間費用の表示検証
  - `YieldAnalysis`: 8%しきい値による分岐（バッジ・カード・色）
  - `DeadCrossChart`: デッドクロスゾーンのバッジ・警告テキスト
  - `CashFlowChart`: 自己資金回収年の表示、exitTotalEquity の色分け
  - `InvestmentForm`: コールバック呼び出し、ローディング中のボタン無効化、詳細設定トグル

#### E2E テストの方針（Playwright）

- **ツール**: [Playwright](https://playwright.dev/) v1.50+ / Chromium
- **モック戦略**: `page.route()` でブラウザネットワーク層の `/api/**` をインターセプト。バックエンド不要で実行可能
- **テストデータ**: `frontend/e2e/fixtures/` の JSON フィクスチャ（`InvestmentResult` 型準拠）
- **localStorage 分離**: `storageState: { cookies: [], origins: [] }` でテスト間の状態汚染を防止
- **優先度タグ**: `@p1`（PR必須）/ `@p2`（PR必須）/ `@p3`（main のみ）
- **カバーするユーザーフロー**: Quick mode / Full mode / 地価取得 / URL共有 / ウォッチリスト / モード切替 / エリア探索 / PDF出力 / エラーハンドリング

> **`@emnapi` の `devDependencies` 明示について**: `@emnapi/core` と `@emnapi/runtime` は `next` → `sharp` のトランジティブ依存。npm 11（macOS arm64）は `optionalDependencies` に記載しても Linux 向けバイナリを lock ファイルから除外するため、`devDependencies` に移動して常に lock ファイルへの収録を強制するワークアラウンドを適用している。

---

## 各計算の法令・出典根拠サマリー

ツールが採用している計算式・数値の根拠を一覧で示す。詳細は各ドキュメントを参照。

| 計算項目 | 採用値・方式 | 根拠 法令・出典 |
|---------|------------|---------------|
| 法定耐用年数 | 木造22年、RC47年 等 | 減価償却資産の耐用年数等に関する**省令 別表第一**（住宅用建物） |
| 中古簡便法耐用年数 | `(法定 - 経過) + 経過×20%` | **耐用年数の適用等に関する取扱通達 1-5-3**（国税庁） |
| 定額法減価償却 | `BuildingCost / 耐用年数` | **所得税法 第49条**（個人）/ **法人税法 第31条**（法人） |
| 仲介手数料上限 | `(売却価格×3%+6万)×1.1` | **宅地建物取引業法 第46条**（媒介報酬の上限） |
| 取得費への諸経費算入 | `LandPrice + 建物簿価 + (LandPrice+BuildingCost)×MiscExpenseRate`（融資諸費用 loanFee は除外、所得税法基本通達 38-8） | **所得税法 第38条**（資産の取得費） |
| 譲渡所得税率（短期） | 39.63% | **租税特別措置法 第32条** + **復興財源確保法 第33条** |
| 譲渡所得税率（長期） | 20.315% | **租税特別措置法 第31条** + **復興財源確保法 第33条**（投資用は5年超でこの税率のみ） |
| 長期10年超軽減（非採用） | 14.21% | **租税特別措置法 第31条の3** は居住用財産の特例。投資用物件には不適用のため本ツールでは使用しない |
| 収益還元法（売却価格） | `NOI / 還元利回り` | **不動産鑑定評価基準**（国交省、令和2年改正）直接還元法 |
| 坪単価換算 | `×3.30578 m²/坪` | 計量法附則 / 不動産業界慣習（1坪=6尺²=3.30578…m²） |
| 8%利回り基準 | `targetYield8pct = 0.08` | 日本不動産研究所「不動産投資家調査」の期待利回り水準を参考とした業界経験則 |
| 元利均等返済 | `P×r×(1+r)^n/((1+r)^n-1)` | 金融工学標準公式（年金現価の逆算） |

> **注意**: 上記は参照時点（2024年）の法令に基づく。税制改正により税率や特例が変更される可能性がある。投資判断を行う際は必ず最新の法令および税理士・不動産鑑定士に確認すること。

---

## 計算の制約と免責事項

本ツールは投資判断の参考情報を提供するものであり、以下の要素は計算に含まれていない。

| 項目 | 備考 |
|------|------|
| 消費税（建物購入・売却時） | 課税仕入れの控除等は考慮しない |
| 3000万円特別控除 | 居住用財産の特例（投資用物件は対象外のため） |
| 損益通算 | 不動産赤字 × 給与所得の通算は考慮しない |
| 青色申告特別控除 | 65万円控除は考慮しない |
| 10年超軽減税率の上限 | 6000万円超部分の通常長期税率（20.315%）は簡略化 |
| 固定資産税の年割り精算 | 売却時の日割り精算は考慮しない |
| 修繕積立金の変動 | ExpenseRate は一定とみなす |
| 家賃の経年変動 | MonthlyRent は全期間一定とみなす |
