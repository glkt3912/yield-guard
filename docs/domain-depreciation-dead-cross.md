# 減価償却・デッドクロス詳細仕様

`backend/internal/domain/types.go` に `BuildingType` / `UsefulLife` / `CalcResidualUsefulLife` がある。
デッドクロス判定は `backend/internal/domain/investment.go` の年次ループ内。

---

## 法定耐用年数一覧

根拠: **減価償却資産の耐用年数等に関する省令 別表第一**（住宅用建物）

| BuildingType 定数 | 文字列 | 法定耐用年数 |
|-----------------|--------|------------|
| `BuildingTypeWood` | `"木造"` | 22年 |
| `BuildingTypeLightSteelThin` | `"軽量鉄骨(3mm以下)"` | 19年（薄板・プレハブ系） |
| `BuildingTypeLightSteel` | `"軽量鉄骨(4mm以下)"` | 27年 |
| `BuildingTypeHeavySteel` | `"重量鉄骨"` | 34年 |
| `BuildingTypeRC` | `"RC造"` | 47年 |
| `BuildingTypeSRC` | `"SRC造"` | 47年（鉄骨鉄筋コンクリート） |

デフォルト（不明な場合）は `BuildingTypeWood` = 22年を使用。

---

## 中古物件の簡便法耐用年数（`CalcResidualUsefulLife`）

根拠: **耐用年数の適用等に関する取扱通達 1-5-3**

```go
func CalcResidualUsefulLife(buildingType BuildingType, buildingAge int) int {
    legal := buildingType.UsefulLife()
    if buildingAge <= 0 {
        return legal  // 新築
    }
    if buildingAge >= legal {
        // パターン1: 法定耐用年数を超過した中古
        residual = int(float64(legal) * 0.2)
    } else {
        // パターン2: 法定耐用年数内の中古
        residual = (legal - buildingAge) + int(float64(buildingAge) * 0.2)
    }
    if residual < 2 { return 2 }  // 最低2年
    return residual
}
```

### パターン1: 法定耐用年数超過の中古

```
簡便法耐用年数 = 法定耐用年数 × 20%（端数切捨て、最低2年）
```

例: 築30年の木造（法定22年） → `22 × 0.2 = 4.4 → 4年`

### パターン2: 法定耐用年数内の中古

```
簡便法耐用年数 = (法定 - 経過年数) + 経過年数 × 20%（端数切捨て）
```

例: 築10年のRC造（法定47年） → `(47 - 10) + 10 × 0.2 = 37 + 2 = 39年`

---

## 定額法による年間減価償却費

```
年間減価償却費 = BuildingCost / 簡便法耐用年数
```

- 土地は減価償却の対象外（土地は使用しても価値が減らない）
- **耐用年数を超過した年以降は `yearDepreciation = 0`**

> **根拠・出典**:
> - **所得税法 第49条**（個人の不動産投資家の場合）: 業務用資産の償却費は定額法・定率法等の方法で計算。個人の建物は2007年4月以降、**定額法のみ**（定率法は選択不可）。
> - **法人税法 第31条**（法人の場合）: 減価償却資産の償却限度額を定める。
> - **減価償却資産の耐用年数等に関する省令 第3条**: 計算方法の詳細規定。
> - **残存価額の廃止**: 2007年度税制改正（所得税法施行令等の改正）により、旧来の残存価額10%を廃止。現在は1円（備忘価額）まで償却可能。本ツールは `BuildingCost / usefulLife` の単純計算を採用（残存1円は実務的影響が軽微のため省略）。

```go
yearDepreciation := 0.0
if year <= usefulLife {
    yearDepreciation = annualDepreciation
}
accumulatedDepreciation += yearDepreciation
```

`accumulatedDepreciation` は出口戦略の建物簿価計算に使用される。

---

## デッドクロスの定義と判定ロジック

### 定義

**元金返済額（AnnualPrincipal）> 減価償却費（AnnualDepreciation）** となる状態。

### 発生メカニズム

元利均等返済では、返済が進むにつれて**利息比率が下がり元金返済比率が増加**する。
一方、減価償却費は定額法のため毎年一定額（耐用年数超過後はゼロ）。

この2本の線が交差するのがデッドクロスの発生時点。

### 判定コード

```go
inDeadCrossZone := annualPrincipal > 0 && annualPrincipal > yearDepreciation
isDeadCrossYear := false
if deadCrossYear == -1 && inDeadCrossZone {
    deadCrossYear = year
    isDeadCrossYear = true
}
```

- `annualPrincipal > 0` の条件: ローン完済後（元金返済ゼロ）はデッドクロスから「脱出」
- `deadCrossYear == -1` の条件: **初年度フラグは初回のみ立てる**

### IsDeadCrossYear vs IsInDeadCrossZone の違い

| フィールド | 意味 | true になる範囲 |
|-----------|------|----------------|
| `IsDeadCrossYear` | デッドクロス初年度 | 最初の1年のみ |
| `IsInDeadCrossZone` | デッドクロス継続中 | ゾーン全体（完済まで続く） |

**耐用年数超過後の特殊ケース**: `yearDepreciation = 0` となるため、元金返済が少しでも残っていれば `inDeadCrossZone = true`。

`DeadCrossYear = -1`: 35年以内にデッドクロスが発生しないことを示す。

---

## 黒字倒産リスクのメカニズム

デッドクロスゾーンでは以下のような状況が発生する。

```
税引前CF = 収入 - ローン返済 - 経費  → プラス（手元にお金はある）
課税所得 = 収入 - 利息 - 減価償却 - 経費
         ↑ここから税金が徴収される

元金返済は「費用」として税務上控除できない
減価償却費が減少（または消滅）すると課税所得が増加
→ 税引後CF が圧迫される
```

CFがプラスにもかかわらず、所得税の実質負担が増加することで
「帳簿上は黒字、手元現金が少ない」状態が続く。これが「黒字倒産リスク」。

### 対策

| 対策 | 説明 |
|------|------|
| 早期売却 | デッドクロス発生前に売却して譲渡益を確定 |
| 繰上返済 | 早期完済によってデッドクロスゾーンを短縮 |
| 新規物件購入 | 新しい物件の減価償却費を積み増して課税所得を圧縮 |

---

## フロントエンドでの可視化（DeadCrossChart）

`frontend/src/components/DeadCrossChart.tsx`

- `ReferenceArea` でデッドクロスゾーン全体をハイライト（`fill: "#fee2e2"` = 薄い赤）
- `deadCrossEndYear`: `isInDeadCrossZone` が `true` の最後の年（ローン完済後に脱出）
- 元金返済ライン: 赤（`#ef4444`）
- 減価償却費ライン: 青の破線（`#3b82f6`, `strokeDasharray="5 5"`）
