---
purpose: バックエンド開発（Go / API / swagger）の出発点
triggers: [go, handler, router, swagger, api endpoint, mlit client, domain type]
reads_next: [docs/api-reference.md, docs/mlit-api-integration.md]
last_updated: 2026-07-05
---

## 重要ファイル

| ファイル | 役割 |
|---------|-----|
| `backend/cmd/server/main.go` | エントリーポイント・swag メタアノテーション置き場 |
| `backend/internal/api/handler.go` | HTTP ハンドラ（大半のエンドポイント） |
| `backend/internal/api/rent_handler.go` | `/api/rent-stats` ハンドラ |
| `backend/internal/api/router.go` | Gin ルーター・CORS・認証ミドルウェア |
| `backend/internal/domain/types.go` | ドメイン型（Go struct = TypeScript 型の Single Source of Truth） |
| `backend/internal/domain/investment.go` | 利回り / デッドクロス / 出口計算ロジック |
| `backend/internal/mlit/client.go` | MLIT API クライアント（リトライ・キャッシュ） |

## 新規エンドポイント追加の手順

1. `domain/types.go` にリクエスト/レスポンス型を追加
2. `handler.go`（または新ファイル）にハンドラを実装
3. `router.go` にルートを登録
4. `make swagger` を実行 → `docs/openapi/swagger.json` を更新
5. swagger.json を**コミットに含める**（CI が型チェックに使う）

## swag アノテーション記法

```go
// @Summary     概要（日本語可）
// @Tags        タグ名
// @Produce     json
// @Param       name  query  string  true  "説明"
// @Success     200  {object}  domain.ResponseType
// @Failure     400  {object}  map[string]string
// @Router      /api/path [get]
func (h *Handler) HandlerFunc(c *gin.Context) {
```

> `@title` 等のメタ情報は `cmd/server/main.go` の `func main()` 直前に書く（`doc.go` では読まれない）。

## OpenAPI 型生成パイプライン

```
backend/internal/domain/*.go  (Go struct + swag アノテーション)
    ↓ make swagger（--requiredByDefault 付き）
docs/openapi/swagger.json     (コミット対象 — 編集禁止)
    ↓ npm run generate:types
frontend/src/types/api.generated.ts  (git 管理外 — CI で毎回生成)
```

- `swagger.json` は git 管理対象。変更したら必ずコミットに含める
- `api.generated.ts` は git 管理外。ローカルで使う場合は `npm run generate:types` を実行

## struct タグ = TypeScript 契約（Issue #811 以降）

フロントエンドの主要型は `api.generated.ts` の再エクスポートのため、
**Go の struct タグがそのまま TS の型を決める**。タグの付け外しは契約変更になる。

| Go 側の実態 | 付けるタグ | 生成される TS 型 |
|------------|-----------|----------------|
| 常に出力される（デフォルト） | なし | required |
| 省略され得る（`Defaults()` 補完・条件付き出力） | `json:"...,omitempty"` | optional (`?`) |
| null を出し得る（`*T` や omitempty なしスライスの nil） | `extensions:"x-nullable"` | `\| null` |
| `*T` + `omitempty`（null は出ない） | そのまま | optional・x-nullable 不要 |

- 型を変更すると `frontend/src/types/__tests__/api-contract.ts`（契約テスト）と
  フロントの使用箇所が `tsc --noEmit` で fail する。**frontend の修正を同一 PR に含めること**
- 新しいレスポンス型を追加したら api-contract.ts の `HandwrittenBySchema` にも登録する
  （登録漏れは `CoverageCheck` が tsc エラーで検出する）

## テスト実行

```bash
cd backend && go test -race ./... -timeout 120s
```
