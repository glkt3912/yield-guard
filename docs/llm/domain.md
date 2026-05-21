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

| 計算内容 | 関数 / ファイル |
|---------|--------------|
| 粗利回り（表面・実質） | `investment.go` — `CalcGrossYield` |
| デッドクロス判定 | `investment.go` — `CalcDeadCross` |
| 出口戦略・譲渡所得税 | `investment.go` — `CalcExitStrategy` |
| 減価償却（法定耐用年数） | `investment.go` — 詳細は `docs/domain-depreciation-dead-cross.md` |

## 型変更時の注意

ドメイン型（`types.go`）を変更したら必ず:

1. `make swagger` を実行
2. `docs/openapi/swagger.json` を変更に含めてコミット
3. フロントエンド側で `npm run generate:types` を実行して型ズレを確認

## テスト追加パターン

```go
func TestCalcXxx(t *testing.T) {
    input := domain.InvestmentInput{ /* フィールド */ }
    result := CalcXxx(input)
    assert.Equal(t, expected, result.Field)
}
```

詳細な計算仕様: `docs/domain-investment-calculation.md`（フィールド定義・計算手順）
