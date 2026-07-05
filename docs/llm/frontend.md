---
purpose: フロントエンド開発（Next.js / TSX / UI）の出発点
triggers: [tsx, react, component, tailwind, shadcn, recharts, frontend, ui]
reads_next: [docs/frontend-components.md, docs/usecases.md]
last_updated: 2026-07-05
---

## 重要ファイル

| ファイル | 役割 |
|---------|-----|
| `frontend/src/app/` | Next.js App Router ページ |
| `frontend/src/components/Dashboard.tsx` | トップレベルレイアウト |
| `frontend/src/components/InvestmentForm.tsx` | 入力フォーム |
| `frontend/src/components/YieldAnalysis.tsx` | 粗利回り + 8%閾値（免責事項 Alert の参照実装） |
| `frontend/src/components/DeadCrossChart.tsx` | デッドクロス可視化（ResponsiveContainer の参照実装） |
| `frontend/src/components/WatchlistPanel.tsx` | ウォッチリスト（Badge + アイコンの参照実装） |
| `frontend/src/lib/` | API クライアント・計算ユーティリティ |
| `frontend/src/types/investment.ts` | API 型定義（バックエンド一致型は生成型の再エクスポート + フロント固有の手書き型） |
| `frontend/src/types/api.generated.ts` | 自動生成型（編集禁止・git 管理外） |
| `frontend/src/types/__tests__/api-contract.ts` | 契約テスト（手書き型と生成型の一致を tsc で検証） |

## TypeScript 型の再生成

```bash
# make swagger が実行済みであること（swagger.json が最新であること）が前提
cd frontend && npm run generate:types
```

## API 型の追加・変更ルール（Issue #811 以降）

- バックエンドと厳密一致する型は `investment.ts` で `Schemas["domain.X"]` の再エクスポート。
  **手書きで二重定義しない**（フィールドコメントは Go 側コメントが JSDoc として引き継がれる）
- 手書きが許されるのは: 生成型の string をユニオンに絞る型（`LoanMethod` / `GeocodeResult` 等）、
  フロント固有フィールドを持つ型（`InvestmentInput.stationMinutes`）、フロント固有型（`Watchlist*` 等）。
  手書き型は `api-contract.ts` の検証対象になる
- 新しい API 型を追加したら `api-contract.ts` の `HandwrittenBySchema` に1行登録する
  （登録漏れは `CoverageCheck` が tsc エラーで知らせる）
- 契約テストが fail したら: Go 側の実態（omitempty 有無・nil 可能性）を確認し、
  タグ修正か手書き型修正かを判断する（`docs/llm/backend.md` のタグ対応表参照）

## UI デザインルール

**禁止**
- グラデーション背景（`bg-gradient-*`）— ブランドカラーのみ
- Lucide 以外のアイコンライブラリ
- 色だけで状態（エラー/警告/成功）を表現 — Badge + アイコンを必ず併記（WCAG 1.4.1）
- ネスト3段以上の可変幅コンテナ
- ラベルなしアイコンボタン（`aria-label` か `Tooltip` を必ず付ける）

**必須**
- 投資シミュレーション結果には `<Alert>` で免責事項を表示（`YieldAnalysis.tsx` 参照）
- ステータス表示は `<Badge>` + アイコン（`WatchlistPanel.tsx` 参照）
- チャートは `<ResponsiveContainer>` でラップ（`DeadCrossChart.tsx` 参照）
- 削除・リセット操作は `AlertDialog` で確認
- ローディングは Skeleton UI（`<Spinner>` 不使用）

## テスト実行

```bash
cd frontend && npm test
```
