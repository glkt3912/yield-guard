# backend/

Go 1.25 / Gin / Clean Architecture → Cloud Run。ルートの `CLAUDE.md` の運用規則がここでも適用される。

作業を始める前に `docs/llm/backend.md`（重要ファイル・エンドポイント追加手順・swag 記法）を読む。

## この階層で特に外せない点

- **Go struct タグ = TypeScript 契約**（Issue #811 以降）。フロントの主要型は生成型の再エクスポートのため、
  `domain/*.go` のタグ変更はそのまま TS の型を変える。詳細とタグ対応表は `docs/llm/backend.md`
- **Go 型を変更したら `make swagger`** → `docs/openapi/swagger.json` をコミットに含める（CI が型チェックに使う）。
  `api.generated.ts` は git 管理外
- 型を変えたら frontend 側（契約テスト・使用箇所）も同一 PR で直す。`make lint` の `tsc --noEmit` で検出される

## 検証

```bash
cd backend && go test -race ./... -timeout 120s   # 単体
make test && make lint                            # 全体（swagger 変更時は make swagger-check も）
```
