# frontend/

Next.js 16（App Router）/ TypeScript / Tailwind CSS v4 / Shadcn/UI → Vercel。ルートの `CLAUDE.md` の運用規則がここでも適用される。

作業を始める前に `docs/llm/frontend.md`（重要ファイル・型の追加ルール・UI デザインルール）を読む。

## この階層で特に外せない点

- **API 型はバックエンドが Single Source of Truth**。バックエンドと厳密一致する型は
  `investment.ts` で `Schemas["domain.X"]` を再エクスポートし、手書きで二重定義しない。
  手書きが許される範囲と契約テストの登録手順は `docs/llm/frontend.md`
- **`api.generated.ts` は編集禁止・git 管理外**。`make swagger` 済みの前提で `npm run generate:types` で再生成する
- **UI デザインルールは厳守**（グラデーション背景禁止・色だけの状態表現禁止・チャートは ResponsiveContainer など）。
  一覧と参照実装は `docs/llm/frontend.md`

## 検証

`make` はリポジトリルート専用（`frontend/` に Makefile は無い）。vitest だけ `frontend/` で回す。

```bash
(cd frontend && npm test)     # vitest（サブシェルで cwd を戻す）
make lint                     # ルートで — eslint + tsc --noEmit（契約テストのズレもここで出る）
```
