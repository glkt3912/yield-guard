---
purpose: 投資計算・ドメインロジック変更の出発点
triggers: [yield, 利回り, dead cross, デッドクロス, exit, 出口戦略, depreciation, 減価償却, investment calc]
reads_next: [docs/domain-investment-calculation.md, docs/domain-depreciation-dead-cross.md, docs/domain-exit-strategy.md]
last_updated: 2026-05-21
---

## 重要ファイル

| ファイル | 役割 |
|---------|-----|
| `backend/internal/domain/types.go` | 全ドメイン型定義（`InvestmentInput`, `InvestmentResult` 等） |
| `backend/internal/domain/investment.go` | 粗利回り・デッドクロス・出口戦略の計算実装 |
| `backend/internal/domain/investment_test.go` | 計算ロジックのユニットテスト |

## 計算ロジックの所在

すべての計算は `Analyze` 関数（公開）がエントリーポイント。個別ロジックは内部で分岐する。

| 計算内容 | 関数 / フィールド |
|---------|----------------|
| 投資シミュレーション全体 | `investment.go` — `Analyze(ctx, InvestmentInput) InvestmentResult` |
| 粗利回り（結果フィールド） | `InvestmentResult.GrossYield` |
| デッドクロス発生年（結果フィールド） | `InvestmentResult.DeadCrossYear` |
| 出口戦略・譲渡所得税（非公開） | `investment.go` — `calcExit(...)` |
| DSCR 計算（公開ユーティリティ） | `investment.go` — `CalcDSCR(noi, annualDebtService)` |
| 減価償却（法定耐用年数） | `investment.go` 内部 — 詳細は `docs/domain-depreciation-dead-cross.md` |

## 型変更時の注意

ドメイン型（`types.go`）を変更したら必ず:

1. `make swagger` を実行
2. `docs/openapi/swagger.json` を変更に含めてコミット
3. フロントエンド側で `npm run generate:types` を実行して型ズレを確認

## テスト追加パターン

```go
func TestAnalyze_Xxx(t *testing.T) {
    input := domain.InvestmentInput{ /* フィールド */ }
    result := domain.Analyze(context.Background(), input)
    assert.Equal(t, expected, result.GrossYield)
}
```

詳細な計算仕様: `docs/domain-investment-calculation.md`（フィールド定義・計算手順）
