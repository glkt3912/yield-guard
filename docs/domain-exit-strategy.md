# 出口戦略・譲渡所得税計算仕様

`backend/internal/domain/investment.go` の `calcExit` 関数（非公開）が担当。
`Analyze` の最後に呼び出され、結果は `InvestmentResult` の `Exit*` フィールドに格納される。

---

## 売却価格の算出: 収益還元法（Direct Capitalization）

```
NOI（純収益）= 実効賃料収入 - 運営経費
             = exitYear.AnnualRent - exitYear.AnnualExpenses

売却価格 = NOI / ExitYieldTarget
```

- `exitYear` は `yearly[HoldingYears - 1]` の年次結果
- **ローン利息は NOI に含まない**（NOI は負債に依存しない純収益）
- `ExitYieldTarget` が低いほど高く売れる（例: 5%→売却価格2000万、6%→1667万）
- `ExitYieldTarget <= 0` または `HoldingYears <= 0` の場合は計算をスキップ

---

## 売却費用（仲介手数料）

```
売却費用 = (売却価格 × 3% + 60,000円) × 1.10（消費税込み）
```

根拠: **宅地建物取引業法 第46条** — 媒介報酬の上限額

---

## 建物の税務上の簿価

```
建物簿価 = BuildingCost - accumulatedDepreciation
```

- `accumulatedDepreciation`: `Analyze` ループで積算した定額法累計
- 簿価がマイナスになった場合は **0 にクランプ**（税法上の下限）

---

## 取得費の算出

```
取得費 = 土地取得費 + 建物簿価 + 取得時諸経費
       = input.LandPrice + bookValueBuilding + miscExpenses
```

根拠: **所得税法 第38条** — 取得費に含まれる付随費用

- 土地は減価償却されないため、**土地取得費はそのまま取得費**
- 諸経費（`miscExpenses`）も取得費に算入可能

---

## 譲渡所得の算出

```
譲渡所得 = 売却価格 - 売却費用 - 取得費
```

- 譲渡所得がマイナスの場合は `transferTax = 0`（譲渡損失は税計算をスキップ）

---

## 譲渡所得税率の3段階ルール

根拠: **租税特別措置法 第31条・第32条**、**復興財源確保法 第33条**（2037年まで）

| 保有年数 | 区分 | 税率 | 内訳 |
|---------|------|------|------|
| 5年以内 | 短期譲渡所得 | **39.63%** | 所得税30% + 復興特別所得税0.63% + 住民税9% |
| 5年超 | 長期譲渡所得 | **20.315%** | 所得税15% + 復興特別所得税0.315% + 住民税5% |
| 10年超 | 長期譲渡所得（軽減） | **14.21%** | 所得税10% + 復興特別所得税0.21% + 住民税4% |

```go
const (
    shortTermTransferTaxRate    = 0.3963   // 短期
    longTermTransferTaxRate     = 0.20315  // 長期
    longTerm10YrTransferTaxRate = 0.14210  // 10年超軽減
)

switch {
case input.HoldingYears > 10:
    taxRate = longTerm10YrTransferTaxRate
case input.HoldingYears > 5:
    taxRate = longTermTransferTaxRate
default:
    taxRate = shortTermTransferTaxRate
}
```

**注意**: `HoldingYears > 5` の判定は**「保有開始から5年超」**であり、
1月1日を境に判断される実務上の扱いとは異なる（本ツールは簡略化）。

### 設計上の簡略化（既知の制限）

10年超の軽減税率（14.21%）は本来、**6000万円以下の部分のみ**適用（6000万円超は通常長期税率 20.315%）。
本ツールは売却価格全体に14.21%を適用する簡略化を採用している。

---

## 売却手取り（ExitNetProceeds）

```
売却手取り = 売却価格 - 売却費用 - 譲渡所得税 - ローン残債
           = salePrice - sellExpenses - transferTax - exitYear.RemainingLoanBalance
```

---

## 最終手残り（ExitTotalEquity）

```
最終手残り = 売却手取り + 累積CF（売却年まで）
           = netProceeds + exitYear.CumulativeCashFlow
```

これが投資の**総合的な成果指標**。
累積CFには毎年の税引後キャッシュフローが積み上がっているため、
「ホールド期間中の収益 + 売却時の収益」を統合した数値。

---

## 計算の制約と免責事項

| 項目 | 本ツールの扱い |
|------|-------------|
| 消費税（建物に課税） | 考慮しない |
| 3000万円特別控除（居住用） | 考慮しない（投資用物件のため） |
| 損益通算（不動産 × 給与） | 考慮しない（所得税率の単純適用） |
| 10年超の6000万円超部分 | 全額14.21%で簡略化（本来は20.315%） |
| 固定資産税の年割り精算 | 考慮しない |
| 仲介手数料の実費差異 | 法定上限額を使用 |
