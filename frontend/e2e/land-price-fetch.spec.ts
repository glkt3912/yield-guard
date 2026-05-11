import { test, expect } from "@playwright/test";
import { setupApiMocks } from "./helpers/routes";
import { ONBOARDING_KEY } from "./helpers/constants";

test.beforeEach(async ({ page }) => {
  await setupApiMocks(page);
  await page.addInitScript((key) => localStorage.setItem(key, "1"), ONBOARDING_KEY);
  await page.goto("/");
  await page.getByRole("radio", { name: "詳細" }).click();
});

test("@p2 地価データ取得：市区町村を選択して相場取得ボタンを押すと比較結果が表示される", async ({
  page,
}) => {
  await page.getByLabel("都道府県").selectOption("13");
  // <option> elements are not "visible"; wait for DOM attachment after mock returns
  await page.locator("option", { hasText: "渋谷区" }).first().waitFor({ state: "attached", timeout: 5_000 });
  // Select 渋谷区 via the select that contains the option
  await page
    .locator("select", { has: page.locator("option", { hasText: "渋谷区" }) })
    .selectOption({ label: "渋谷区" });

  await page.getByText("相場データを取得").click();

  // land-price-compare フィクスチャの assessment が表示される
  await expect(page.getByText("割高", { exact: true })).toBeVisible({ timeout: 10_000 });
});

test("@p2 地価データ取得：市区町村ドロップダウンに選択肢が表示される", async ({ page }) => {
  await page.getByLabel("都道府県").selectOption("13");

  await page.locator("option", { hasText: "千代田区" }).first().waitFor({ state: "attached", timeout: 5_000 });
  await page.locator("option", { hasText: "渋谷区" }).first().waitFor({ state: "attached" });
});
