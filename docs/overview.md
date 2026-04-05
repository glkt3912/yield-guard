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
│       │   ├── client_test.go     # ユニットテスト（httptest モック）
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
# バックエンド（レースチェック付き・全パッケージ）
cd backend
go test -race ./... -v

# フロントエンド（Vitest）
cd frontend
npm test
```

### テスト構成

| レイヤー | ファイル | ツール | テスト数 |
|---|---|---|---|
| ドメイン計算 | `backend/internal/domain/investment_test.go` | go test | 複数 |
| MLIT クライアント | `backend/internal/mlit/client_test.go` | go test / httptest | 11 |
| フロントエンド UI | `frontend/src/components/__tests__/*.test.tsx` | Vitest + RTL | 17 |

#### フロントエンドテストの方針

- **ツール**: [Vitest](https://vitest.dev/) + [React Testing Library](https://testing-library.com/react)
- **環境**: jsdom（ブラウザAPI をエミュレート）
- **モック**: `ResizeObserver`（Recharts が要求）、APIコールは `vi.fn()` で差し替え
- **テスト対象コンポーネント**:
  - `YieldAnalysis`: 8%しきい値による分岐（バッジ・カード・色）
  - `DeadCrossChart`: デッドクロスゾーンのバッジ・警告テキスト
  - `CashFlowChart`: 自己資金回収年の表示、exitTotalEquity の色分け
  - `InvestmentForm`: コールバック呼び出し、ローディング中のボタン無効化、詳細設定トグル

---

## 各計算の法令・出典根拠サマリー

ツールが採用している計算式・数値の根拠を一覧で示す。詳細は各ドキュメントを参照。

| 計算項目 | 採用値・方式 | 根拠 法令・出典 |
|---------|------------|---------------|
| 法定耐用年数 | 木造22年、RC47年 等 | 減価償却資産の耐用年数等に関する**省令 別表第一**（住宅用建物） |
| 中古簡便法耐用年数 | `(法定 - 経過) + 経過×20%` | **耐用年数の適用等に関する取扱通達 1-5-3**（国税庁） |
| 定額法減価償却 | `BuildingCost / 耐用年数` | **所得税法 第49条**（個人）/ **法人税法 第31条**（法人） |
| 仲介手数料上限 | `(売却価格×3%+6万)×1.1` | **宅地建物取引業法 第46条**（媒介報酬の上限） |
| 取得費への諸経費算入 | `LandPrice + 建物簿価 + miscExpenses` | **所得税法 第38条**（資産の取得費） |
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
