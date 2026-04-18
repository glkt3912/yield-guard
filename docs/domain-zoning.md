# 用途地域・都市計画区域 ドメイン型仕様

実装: `backend/internal/domain/zoning.go`
テスト: `backend/internal/domain/zoning_test.go`

---

## ZoningType（用途地域）

根拠: **都市計画法 第8条第1項第1号**

MLIT API が返す生文字列をそのまま型の値として使用する。
`ParseZoningType(string)` で変換し、認識できない文字列は `ZoningUnknown`（空文字列）を返す。

| 定数 | 文字列 | リスクレベル |
|------|--------|------------|
| `ZoningFirstLowRise` | `"第一種低層住居専用地域"` | None |
| `ZoningSecondLowRise` | `"第二種低層住居専用地域"` | None |
| `ZoningFirstMidHighRise` | `"第一種中高層住居専用地域"` | None |
| `ZoningSecondMidHighRise` | `"第二種中高層住居専用地域"` | None |
| `ZoningFirstResidential` | `"第一種住居地域"` | None |
| `ZoningSecondResidential` | `"第二種住居地域"` | None |
| `ZoningQuasiResidential` | `"準住居地域"` | None |
| `ZoningNeighborhoodCommercial` | `"近隣商業地域"` | Caution |
| `ZoningCommercial` | `"商業地域"` | Caution |
| `ZoningQuasiIndustrial` | `"準工業地域"` | Caution |
| `ZoningIndustrial` | `"工業地域"` | High |
| `ZoningExclusiveIndustrial` | `"工業専用地域"` | High |
| `ZoningGardenCity` | `"田園住居地域"` | Caution |
| `ZoningUnknown` | `""` | — |

---

## ZoningRiskLevel（リスクレベル）

賃貸住宅投資の観点でのリスク分類。

| 定数 | 値 | 意味 |
|------|----|------|
| `ZoningRiskNone` | `0` | 問題なし（住居系地域） |
| `ZoningRiskCaution` | `1` | 注意（用途制限あり・混在リスク） |
| `ZoningRiskHigh` | `2` | 高リスク（住居不適・建設禁止） |

---

## ZoningMeta（メタデータ）

各用途地域に紐づくメタデータ。`ZoningType.Meta()` で取得。

```go
type ZoningMeta struct {
    DefaultBuildingCoverage int             // デフォルト建ぺい率（%）
    DefaultFloorAreaRatio   int             // デフォルト容積率（%）
    RiskLevel               ZoningRiskLevel
    RiskMessage             string
}
```

> **注意**: 建ぺい率・容積率は都市計画法の標準値であり、実際の値は自治体の都市計画により異なる。
> あくまで参考値として扱うこと。

| 用途地域 | 建ぺい率 | 容積率 |
|---------|---------|--------|
| 第一種低層住居専用地域 | 60% | 100% |
| 第二種低層住居専用地域 | 60% | 150% |
| 第一種中高層住居専用地域 | 60% | 200% |
| 第二種中高層住居専用地域 | 60% | 200% |
| 第一種住居地域 | 60% | 200% |
| 第二種住居地域 | 60% | 200% |
| 準住居地域 | 60% | 200% |
| 近隣商業地域 | 80% | 300% |
| 商業地域 | 80% | 400% |
| 準工業地域 | 60% | 200% |
| 工業地域 | 60% | 200% |
| 工業専用地域 | 60% | 200% |
| 田園住居地域 | 50% | 100% |

---

## CityPlanningArea（都市計画区域）

| 定数 | 文字列 | `IsHighRisk()` |
|------|--------|---------------|
| `CityPlanningUrbanized` | `"市街化区域"` | false |
| `CityPlanningUrbanizationControlled` | `"市街化調整区域"` | **true** |
| `CityPlanningUnzoned` | `"非線引き区域"` | false |
| `CityPlanningOutside` | `"都市計画区域外"` | false |
| `CityPlanningAreaUnknown` | `""` | false |

`IsHighRisk()` は市街化調整区域のみ `true` を返す。
市街化調整区域は原則として開発行為が制限されており、建築・増改築に自治体の許可が必要なため投資リスクが高い。

---

## 変換関数

### `ParseZoningType(s string) ZoningType`

MLIT API の `CityPlanning` フィールドの生文字列を `ZoningType` に変換する。
`zoningMetaMap` に存在しない文字列は `ZoningUnknown` を返す。

### `ParseCityPlanningArea(s string) CityPlanningArea`

MLIT API の `CityPlanning` フィールドの生文字列を `CityPlanningArea` に変換する。
4種の既定値以外は `CityPlanningAreaUnknown` を返す。

---

## 後続機能との関係

| issue | 機能 | 使用する型 |
|-------|------|-----------|
| #72 | 用途地域リスク警告表示 | `ZoningType.Meta().RiskLevel` |
| #66 | 都市計画リスク警告 | `CityPlanningArea.IsHighRisk()` |
| #61 | 建ぺい率・容積率の自動入力 | `ZoningMeta.DefaultBuildingCoverage/FloorAreaRatio` |
