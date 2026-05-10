import { test, expect } from "@playwright/test";
import { setupApiMocks } from "./helpers/routes";
import { ONBOARDING_KEY } from "./helpers/constants";

test("@p2 URL共有：分析後のURLを開くとフォームが復元され結果が表示される", async ({
  page,
  context,
}) => {
  await setupApiMocks(page);
  await page.addInitScript((key) => localStorage.setItem(key, "1"), ONBOARDING_KEY);
  await page.goto("/");

  await page.getByLabel("物件価格（土地＋建物の総額）").fill("1700");
  await page.getByLabel("想定月額賃料").fill("15");
  await page.getByText("シミュレーション実行").click();
  await expect(page.getByText("9.89%")).toBeVisible({ timeout: 10_000 });

  // Clipboard 権限を付与してシェアボタンをクリック
  await context.grantPermissions(["clipboard-read", "clipboard-write"]);
  await page.getByRole("button", { name: "この条件をクリップボードにコピー" }).click();
  await expect(page.getByText("コピーしました")).toBeVisible({ timeout: 5_000 });

  // コピーされた URL を取得
  const sharedUrl = await page.evaluate(() => navigator.clipboard.readText());
  expect(sharedUrl).toContain("localhost:3000");

  // 新しいページで URL を開く
  const newPage = await context.newPage();
  await setupApiMocks(newPage);
  await newPage.addInitScript((key) => localStorage.setItem(key, "1"), ONBOARDING_KEY);
  await newPage.goto(sharedUrl);

  // フォームが復元されている（quickTotalPriceMan パラメータ経由）
  await expect(newPage.getByLabel("物件価格（土地＋建物の総額）")).toHaveValue("1700", {
    timeout: 5_000,
  });
  // 結果も自動的に表示される
  await expect(newPage.getByText("9.89%")).toBeVisible({ timeout: 10_000 });
});
