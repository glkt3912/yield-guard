# E2E セレクタ ガイドライン

Playwright を使った E2E テストを追加・変更する際の指針。
PR #513 の実装で得た知見と Playwright 公式推奨をまとめたもの。

---

## セレクタ優先順位

**上から順に検討し、最初に使えるものを採用する。**

| 優先度 | メソッド | 使いどころ |
|---|---|---|
| 1 | `getByRole` | ボタン・リンク・見出し・ラジオ・アラートなど ARIA role を持つ要素 |
| 2 | `getByLabel` | `<label>` または `aria-label` が付いたフォーム要素 |
| 3 | `getByPlaceholder` | プレースホルダーテキストで特定できる入力欄 |
| 4 | `getByText` | 上記で特定できない静的テキスト。**必ず `{ exact: true }` を付ける** |
| 5 | `getByTestId` | 他の方法で一意に特定できない要素（リストアイテム・SVGグラフ等） |
| 6 | `locator()` | 上記がすべて使えない場合の最終手段 |

### 理由

`getByRole` / `getByLabel` はアクセシビリティ属性に基づくため、HTML の内部構造が変わっても壊れにくい。
`locator("div.flex > span:nth-child(2)")` のような CSS セレクタは DOM 構造変更で即壊れる。

---

## getByText の落とし穴

### 必ず `{ exact: true }` を付ける

`getByText("割高")` はデフォルトで部分一致のため、`+3.5万円/坪（割高）` のような要素にもマッチする。
Playwright は strict mode により複数マッチを即エラーにする。

```typescript
// ❌ 複数マッチ → strict mode violation
await expect(page.getByText("割高")).toBeVisible();

// ✓ 完全一致 → 該当要素だけを返す
await expect(page.getByText("割高", { exact: true })).toBeVisible();
```

### テキストが分割されている場合

このプロジェクトでは `YieldAnalysis` の利回り表示が分割されている:

```jsx
<span className="text-4xl">9.89</span>   // "9.89" のみ
<span>%</span>                           // "%" のみ
```

`getByText("9.89%")` は `MobileSummaryCard` の `<p>9.89%</p>`（lg:hidden で非表示）にマッチしてしまう。
分割されている側を狙う:

```typescript
// ✓ text-4xl span だけにマッチ（MobileSummaryCard の p 要素には一致しない）
await expect(page.getByText("9.89", { exact: true })).toBeVisible();
```

### 複数マッチを避けられない場合は親要素でスコープを絞る

```typescript
// ✓ watchlist パネル内に限定して検索
const panel = page.locator('[data-testid="watchlist-panel"]');
await expect(panel.getByText("渋谷区テスト物件", { exact: true })).toBeVisible();
```

---

## CSS 非表示要素（lg:hidden など）の扱い

Tailwind の `lg:hidden` / `sm:hidden` 等でレスポンシブに隠れた要素は、DOM には存在するが Playwright では "hidden" 扱いになる。

| 状態 | 正しいアサーション |
|---|---|
| 見えているか確認 | `toBeVisible()` |
| CSS で隠れているか確認 | `toBeHidden()` |
| DOM に存在するか確認（可視性不問） | `toBeAttached()` |
| 消えたことを確認（DOM から削除） | `not.toBeAttached()` または `toBeHidden()` |

```typescript
// ✓ PC viewport (1280px) で MobileSummaryCard は lg:hidden → hidden
await expect(page.locator('[class~="lg:hidden"]')).toBeHidden();

// ✓ モード切替後に結果が消えたことを確認
await expect(page.getByText("9.89", { exact: true })).toBeHidden();
```

**`not.toBeVisible()` と `toBeHidden()` の違い**:
- `not.toBeVisible()` は「見えない」（hidden または DOM にない）
- `toBeHidden()` は「CSS 非表示」または「DOM にない」を明示的に検証

---

## `<select>` の option 要素

`<option>` はブラウザのネイティブ dropdown として描画されるため、Playwright は常に hidden 扱いにする。
`toBeVisible()` でオプションの存在を確認することはできない。

```typescript
// ❌ option は常に hidden → 失敗する
await expect(page.locator('option[value="13113"]')).toBeVisible();

// ✓ DOM への追加を待つ
await page.locator("option", { hasText: "渋谷区" }).first().waitFor({ state: "attached", timeout: 5_000 });

// ✓ selectOption() で選択（option が存在しなければ自動でエラー）
await page.locator("select").selectOption({ label: "渋谷区" });

// ✓ 選択後の値を確認
await expect(page.locator("select")).toHaveValue("13113");
```

---

## Service Worker と page.route()

### 必ず `serviceWorkers: "block"` を設定する

Service Worker は `networkFirst` 等のストラテジーで自前 `fetch()` を発行する。
Playwright の `page.route()` はページコンテキストからのリクエストしか傍受しないため、
SW が発行したリクエストはモックを素通りしてサーバーに届いてしまう。

```typescript
// playwright.config.ts
export default defineConfig({
  use: {
    serviceWorkers: "block",  // SW を無効化して page.route() を確実に機能させる
  },
});
```

SW が有効な状態でのリクエストフロー（問題のある状態）:

```
ページ → GET /api/municipalities
         ↓
         SW が先に横取り
         ↓
         SW が自前で fetch() → Next.js サーバー → ECONNREFUSED
         （page.route() は見ていない）
```

SW をブロックした状態（正常）:

```
ページ → GET /api/municipalities
         ↓
         page.route() が傍受
         ↓
         フィクスチャを返す
```

### LIFO（後入れ先出し）ルール

`page.route()` は後に登録したルートが優先される。
キャッチオールを **最初に**、個別ルートを **後に** 登録する:

```typescript
// キャッチオールを最初に登録（= 最低優先度）
await page.route("**/api/**", async (route) => {
  console.warn(`[E2E] 想定外の API 呼び出し: ${route.request().url()}`);
  await route.fulfill({ status: 500, ... });
});

// 個別ルートを後から登録（= 高優先度）
await page.route("**/api/investment/analyze", async (route) => {
  await route.fulfill({ status: 200, body: JSON.stringify(fixture) });
});
```

---

## data-testid の付与基準

**「`getByRole` → `getByLabel` → `getByText(exact:true)` で一意に特定できない場合のみ付与する」**

### 付与が必要な典型ケース

- **リストアイテムの個別取得**: 同じ構造が複数並ぶ `<li>` や `<tr>`
- **SVG・グラフ要素**: `<svg>` には role がなく `getByRole` で特定できない
- **同名ボタンが複数ある**: このプロジェクトの「追加」ボタンが該当（3箇所）

### 付与が不要なケース

| 要素 | 代わりに使うもの |
|---|---|
| `<button>削除</button>` | `getByRole("button", { name: "削除" })` |
| `<input aria-label="物件名" />` | `getByLabel("物件名")` |
| `<h3>キャッシュフロー推移（35年）</h3>` | `getByRole("heading", { name: /キャッシュフロー推移/ })` |
| `<div>8%超え ✓</div>` | `getByText("8%超え ✓", { exact: true })` |

### 命名規則

```
<スコープ>-<コンポーネント>[-<修飾子>]
```

```html
<!-- ✓ 良い例 -->
<li data-testid="watchlist-item-0">...</li>
<button data-testid="watchlist-add-button">追加</button>
<svg data-testid="cashflow-chart">...</svg>

<!-- ❌ 悪い例（何を指すか不明） -->
<li data-testid="item">...</li>
<button data-testid="btn">追加</button>
```

---

## Web-first アサーション

`waitForSelector` ではなく `expect(locator).toBeVisible()` を使う。
後者は条件を満たすまで自動でリトライするため、タイミング起因の不安定なテストになりにくい。

```typescript
// ❌ 古い書き方（リトライなし・可視性チェックなし）
await page.waitForSelector("text=9.89");

// ✓ 推奨（自動リトライ + CSS 可視性チェック）
await expect(page.getByText("9.89", { exact: true })).toBeVisible({ timeout: 10_000 });
```

### タイムアウトの目安

| 状況 | timeout |
|---|---|
| API 呼び出し + 再レンダリング | `10_000` |
| ローカル state 変更のみ | `3_000` – `5_000` |
| DOM 削除の確認 | `3_000` |

タイムアウトを省略するとデフォルト（5秒）が使われる。
API 待ちがある場合は明示的に指定すること。

---

## このプロジェクトで実際に発生した問題と対処

| 問題 | 原因 | 対処 |
|---|---|---|
| `getByText("9.89%").first()` が hidden 要素を返す | `MobileSummaryCard`（`lg:hidden`）が DOM で先に来る | `getByText("9.89", { exact: true })` に変更 |
| `getByText("割高")` が strict mode violation | Badge と `+3.5万円/坪（割高）` span の2要素にマッチ | `exact: true` を追加 |
| `getByText("デッドクロス")` が4要素にマッチ | `TermTooltip` が複数のスパンを生成 | `getByRole("heading", { name: /デッドクロス/ })` に変更 |
| `getByText("キャッシュフロー推移")` が見つからない | 実テキストは `"キャッシュフロー推移（35年）"` | `getByRole("heading", { name: /キャッシュフロー推移/ })` に変更 |
| `getByText("渋谷区").toBeVisible()` が失敗 | `<option>` は Playwright では常に hidden | `waitFor({ state: "attached" })` に変更 |
| GET モックが効かない（ECONNREFUSED） | SW が `page.route()` より先にリクエストを横取り | `serviceWorkers: "block"` を設定 |
| watchlist「追加」ボタンが strict mode violation | 同名ボタンが3箇所存在 | `data-testid="watchlist-add-button"` を付与し `getByTestId` で取得 |

---

## 参考資料

- [Locators](https://playwright.dev/docs/locators) — セレクタ優先順位・各ロケータ API の解説
- [Best Practices](https://playwright.dev/docs/best-practices) — `data-testid` の使いどころ・安定したテストの書き方
- [Assertions](https://playwright.dev/docs/test-assertions) — `toBeVisible` / `toBeHidden` / `toBeAttached` 等の Web-first assertion 一覧
- [Network](https://playwright.dev/docs/network) — `page.route()` によるリクエスト傍受・`serviceWorkers` オプション
