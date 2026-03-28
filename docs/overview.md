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
[フロントエンド] Next.js 14 (App Router)
    ↓ fetch to http://localhost:8080
[バックエンド] Go + Gin
    ↓ https://www.land.mlit.go.jp/...
[外部API] 国交省 不動産取引価格情報取得API
```

---

## 技術スタック

### バックエンド

| 技術 | 用途 |
|------|------|
| Go 1.25 | 言語 |
| Gin v1.9.1 | HTTPルーティング |
| gin-contrib/cors | CORS ミドルウェア |

### フロントエンド

| 技術 | 用途 |
|------|------|
| Next.js 14 (App Router) | フレームワーク |
| React 18 | UI |
| TypeScript | 言語 |
| Tailwind CSS 3.4 | スタイリング |
| Shadcn/UI | コンポーネントライブラリ |
| Recharts 2.12 | グラフ描画 |
| Lucide React | アイコン |

---

## ディレクトリ構成

```
yield-guard/
├── backend/
│   ├── cmd/server/main.go         # エントリポイント・グレースフルシャットダウン
│   └── internal/
│       ├── domain/
│       │   ├── types.go           # ドメインモデル・BuildingType・耐用年数
│       │   ├── investment.go      # 計算ロジック・Analyze・calcExit
│       │   └── investment_test.go # ユニットテスト
│       ├── mlit/
│       │   ├── client.go          # 国交省APIクライアント・リトライ
│       │   └── types.go           # APIレスポンス型・都道府県マップ
│       └── api/
│           ├── handler.go         # HTTPハンドラー・バリデーション
│           └── router.go          # Ginルーター・CORS設定
├── frontend/
│   └── src/
│       ├── app/
│       │   ├── layout.tsx
│       │   └── page.tsx           # Dashboard をレンダリング
│       ├── components/
│       │   ├── Dashboard.tsx      # メインUI・状態管理
│       │   ├── InvestmentForm.tsx # 投資条件入力フォーム
│       │   ├── YieldAnalysis.tsx  # 利回り分析・8%ゲージ
│       │   ├── CashFlowChart.tsx  # CF推移グラフ・出口サマリー
│       │   ├── DeadCrossChart.tsx # デッドクロス予測グラフ
│       │   ├── LandPriceAnalysis.tsx # 土地相場分析
│       │   └── ui/                # Shadcn/UIコンポーネント
│       ├── lib/
│       │   ├── api.ts             # fetchLandPrices / analyze / etc.
│       │   └── utils.ts           # formatMan / formatPct / formatYen
│       └── types/
│           └── investment.ts      # TypeScript型定義・DEFAULT_INPUT
├── docs/                          # ドキュメント（docs-mcp-server用）
│   ├── metadata.json
│   ├── overview.md
│   └── ...
└── .mcp.json                      # docs-mcp-server 設定
```

---

## 開発サーバー起動手順

### バックエンド

```bash
cd backend
go mod tidy
PORT=8080 go run cmd/server/main.go
```

環境変数:
- `PORT`: リッスンポート（デフォルト: `8080`）
- `ALLOW_ORIGINS`: CORS許可オリジン（デフォルト: `http://localhost:3000`）

### フロントエンド

```bash
cd frontend
npm install
npm run dev   # http://localhost:3000
```

---

## テスト実行

```bash
cd backend
go test ./internal/domain/...
```

`backend/internal/domain/investment_test.go` にユニットテストあり。

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
