import { test, expect } from "@playwright/test";
import { setupApiMocks, setSwTestMode } from "./helpers/routes";
import { ONBOARDING_KEY } from "./helpers/constants";

test.beforeEach(async ({ page }) => {
  await setupApiMocks(page);
  await page.addInitScript((key) => localStorage.setItem(key, "1"), ONBOARDING_KEY);
  await page.goto("/");
  await setSwTestMode(page);
});

test("@p3 PDF出力：ダウンロードイベントが発生しファイル名が .pdf で終わる", async ({ page }) => {
  await page.getByLabel("物件価格（土地＋建物の総額）").fill("1700");
  await page.getByLabel("想定月額賃料").fill("15");
  await page.getByText("シミュレーション実行").click();
  await expect(page.getByText("9.89%").first()).toBeVisible({ timeout: 10_000 });

  const downloadPromise = page.waitForEvent("download", { timeout: 30_000 });
  await page.getByText("PDFレポート出力").click();

  const download = await downloadPromise;
  expect(download.suggestedFilename()).toMatch(/\.pdf$/);
});
