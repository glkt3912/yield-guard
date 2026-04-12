# Yield-Guard

不動産投資の意思決定をデータで支援するMVPツール。国土交通省の公式APIから土地取引価格を取得し、表面利回り・デッドクロス・出口戦略をリアルタイムで可視化する。

## 概要

「Yield-Guard」は不動産投資の3大リスクを定量化するツールです。

| 機能 | 内容 |
|------|------|
| **相場判定** | 国交省APIから土地取引実績を取得し、坪単価の平均・中央値と比較して割高/割安を即判定 |
| **8%境界線** | 表面利回りが8%を下回る場合、達成に必要な土地値・建築費の削減幅を逆算表示 |
| **デッドクロス予測** | 元金返済額が減価償却費を超える年を特定し、所得税負担増による黒字倒産リスクをグラフ化 |
| **出口戦略** | 任意年数後に利回り6%で売却した際の譲渡所得税込み手残り額（Equity）を算出 |
| **ストレステスト** | 空室率+10%・金利+1.5%時のキャッシュフロー変化をスライダーでリアルタイム可視化 |

## アーキテクチャ

```
┌─────────────────────────────┐     HTTP      ┌─────────────────────────────┐
│  Frontend (Next.js)         │ ◄──────────► │  Backend (Go / Gin)         │
│  localhost:3000             │              │  localhost:8080              │
└─────────────────────────────┘              └──────────────┬──────────────┘
                                                            │  HTTPS
                                                            ▼
                                              ┌─────────────────────────────┐
                                              │  国交省 不動産取引価格API    │
                                              │  land.mlit.go.jp            │
                                              └─────────────────────────────┘
```

**技術スタック**

- Backend: Go 1.25 / Gin / Clean Architecture
- Frontend: Next.js 14 (App Router) / TypeScript / Tailwind CSS / Shadcn/UI / Recharts
- Data: 国土交通省 不動産取引価格情報取得API（認証不要・公式）

## セットアップ

```bash
# リポジトリのクローン
git clone <repository-url>
cd yield-guard
```

### バックエンド

```bash
cd backend

# 依存関係のインストール
go mod tidy

# 開発サーバー起動 (デフォルト: :8080)
go run cmd/server/main.go

# ポートを変更する場合
PORT=9000 go run cmd/server/main.go
```

### フロントエンド

```bash
cd frontend

# 依存関係のインストール
npm install

# 開発サーバー起動 (デフォルト: :3000)
npm run dev
```

## 使い方

1. バックエンドを起動 (`localhost:8080`)
2. フロントエンドを起動して `http://localhost:3000` にアクセス
3. 都道府県・市区町村を選択し「相場データ取得」をクリック
4. 土地価格・建築費・想定賃料・ローン条件を入力して「分析実行」

### APIエンドポイント

| メソッド | パス | 説明 |
|----------|------|------|
| `GET` | `/api/land-prices` | 土地取引価格一覧・統計 |
| `GET` | `/api/land-prices/compare` | 検討地と相場の比較 |
| `POST` | `/api/analyze` | 投資シミュレーション実行 |
| `GET` | `/api/prefectures` | 都道府県一覧 |
| `GET` | `/health` | ヘルスチェック |

**`POST /api/analyze` リクエスト例:**

```json
{
  "landPrice": 5000000,
  "buildingCost": 10000000,
  "miscExpenseRate": 0.07,
  "monthlyRent": 120000,
  "vacancyRate": 0.05,
  "loanAmount": 13000000,
  "annualLoanRate": 0.015,
  "loanYears": 35,
  "buildingType": "木造",
  "expenseRate": 0.20,
  "incomeTaxRate": 0.33,
  "holdingYears": 10,
  "exitYieldTarget": 0.06
}
```

## ディレクトリ構成

```
yield-guard/
├── .github/
│   └── workflows/
│       ├── backend-ci.yml          # Go vet / test -race / build
│       └── frontend-ci.yml         # lint / tsc / build
├── backend/
│   ├── cmd/server/main.go          # エントリポイント
│   └── internal/
│       ├── domain/
│       │   ├── types.go            # ドメインモデル
│       │   ├── investment.go       # 収支計算ロジック（元利均等・減価償却・税金）
│       │   └── investment_test.go  # ユニットテスト
│       ├── mlit/
│       │   ├── client.go           # 国交省APIクライアント（リトライ付き）
│       │   ├── client_test.go      # ユニットテスト（httptest モック）
│       │   └── types.go            # APIレスポンス型
│       └── api/
│           ├── handler.go          # HTTPハンドラー
│           └── router.go           # Ginルーター
├── docs/
│   ├── overview.md                 # 全体設計概要
│   ├── mlit-api-integration.md     # 国交省APIクライアント仕様
│   ├── domain-investment-calculation.md
│   └── ...
├── frontend/
│   └── src/
│       ├── app/                    # Next.js App Router
│       ├── components/             # UIコンポーネント
│       ├── lib/                    # APIクライアント・計算ユーティリティ
│       └── types/                  # TypeScript型定義
└── README.md
```

## 開発

### テスト実行

```bash
# バックエンド
cd backend
go test -race ./... -v

# フロントエンド
cd frontend
npm test           # Vitest（17テスト）
npm run lint
npx tsc --noEmit
```

### CI

PR・mainへのpush時に GitHub Actions が自動実行される（ワークフロー自身の変更でも再トリガーされる）。

| ワークフロー | トリガーパス | チェック内容 |
|---|---|---|
| Backend CI | `backend/**`, `backend-ci.yml` | `golangci-lint` / `go test -race` / `go build` |
| Frontend CI | `frontend/**`, `frontend-ci.yml` | `lint` / `tsc --noEmit` / `vitest run` / `build` |

Dependabot により Go modules・npm の依存パッケージが毎週月曜（JST）に自動更新される（エコシステムごとに1PR）。

### ビルド

```bash
# バックエンド
cd backend && go build -o yield-guard-server ./cmd/server

# フロントエンド
cd frontend && npm run build
```

### 計算ロジック仕様

| 計算 | 式 |
|------|----|
| 表面利回り | `(月額賃料 × 12) / 総投資額` |
| 元利均等返済 | `P × r(1+r)^n / ((1+r)^n - 1)` |
| 減価償却（定額法） | `建築費 / 法定耐用年数` |
| デッドクロス | 元金返済額 > 減価償却費 となる最初の年 |
| 長期譲渡税率 | 20.315%（保有5年超） / 39.363%（5年以下） |

**法定耐用年数:** 木造 22年 / 軽量鉄骨 27年 / 重量鉄骨 34年 / RC造 47年

## ライセンス

MIT
