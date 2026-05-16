import { test, expect } from "@playwright/test";
import analyzeCriticalFixture from "./fixtures/analyze-response-critical.json";
import { setupApiMocks } from "./helpers/routes";
import { ONBOARDING_KEY } from "./helpers/constants";

// デフォルト fixture (analyze-response.json) には multiExitComparison が含まれるため
// beforeEach のセットアップで判定サマリー・出口比較テーブルの両セクションをカバーできる
test.describe("PDF出力 — デフォルト fixture (OK判定 / multiExitComparison あり)", () => {
  test.beforeEach(async ({ page }) => {
    await setupApiMocks(page);
    await page.addInitScript((key) => localStorage.setItem(key, "1"), ONBOARDING_KEY);
    await page.goto("/");
  });

  test("@p3 ダウンロードイベントが発生しファイル名が .pdf で終わる", async ({ page }) => {
    await page.getByLabel("物件価格（土地＋建物の総額）").fill("1700");
    await page.getByLabel("想定月額賃料").fill("15");
    await page.getByText("シミュレーション実行").click();
    await expect(page.getByText("9.89", { exact: true })).toBeVisible({ timeout: 10_000 });

    const downloadPromise = page.waitForEvent("download", { timeout: 30_000 });
    await page.getByText("PDFレポート出力").click();

    const download = await downloadPromise;
    expect(download.suggestedFilename()).toMatch(/\.pdf$/);
  });
});

// RISK判定（criticalErrors=REJECT）では StatusSummary が "リスク" になる
// マイナスエクイティの multiExitComparison でも PDF が例外なく生成されることを確認
test.describe("PDF出力 — RISK判定 fixture (criticalErrors=REJECT / マイナスエクイティ)", () => {
  test.beforeEach(async ({ page }) => {
    await setupApiMocks(page, {
      analyze: { status: 200, body: analyzeCriticalFixture },
    });
    await page.addInitScript((key) => localStorage.setItem(key, "1"), ONBOARDING_KEY);
    await page.goto("/");
  });

  test("@p3 RISK判定でPDFが生成される", async ({ page }) => {
    await page.getByLabel("物件価格（土地＋建物の総額）").fill("1700");
    await page.getByLabel("想定月額賃料").fill("15");
    await page.getByText("シミュレーション実行").click();
    // 表面利回りラベルが表示されれば分析完了（"3.30" は複数箇所に登場するため使わない）
    await expect(page.getByText("表面利回り（満室想定年収 / 総投資額）")).toBeVisible({ timeout: 10_000 });

    const downloadPromise = page.waitForEvent("download", { timeout: 30_000 });
    await page.getByText("PDFレポート出力").click();

    const download = await downloadPromise;
    expect(download.suggestedFilename()).toMatch(/\.pdf$/);
  });
});
