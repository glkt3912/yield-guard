import { test, expect } from "@playwright/test";
import analyzeFixture from "./fixtures/analyze-response.json";
import analyzeCriticalFixture from "./fixtures/analyze-response-critical.json";
import { setupApiMocks } from "./helpers/routes";
import { ONBOARDING_KEY } from "./helpers/constants";

test.beforeEach(async ({ page }) => {
  await setupApiMocks(page);
  await page.addInitScript((key) => localStorage.setItem(key, "1"), ONBOARDING_KEY);
  await page.goto("/");
});

test("@p3 PDF出力：ダウンロードイベントが発生しファイル名が .pdf で終わる", async ({ page }) => {
  await page.getByLabel("物件価格（土地＋建物の総額）").fill("1700");
  await page.getByLabel("想定月額賃料").fill("15");
  await page.getByText("シミュレーション実行").click();
  await expect(page.getByText("9.89", { exact: true })).toBeVisible({ timeout: 10_000 });

  const downloadPromise = page.waitForEvent("download", { timeout: 30_000 });
  await page.getByText("PDFレポート出力").click();

  const download = await downloadPromise;
  expect(download.suggestedFilename()).toMatch(/\.pdf$/);
});

test("@p3 PDF出力：multiExitComparison ありでPDFが生成される", async ({ page }) => {
  // analyze-response.json に multiExitComparison が含まれていることを前提とする
  // PDF生成コード内の出口比較テーブルセクションが例外なく実行されることを確認
  await setupApiMocks(page, {
    analyze: { status: 200, body: analyzeFixture },
  });
  await page.addInitScript((key) => localStorage.setItem(key, "1"), ONBOARDING_KEY);
  await page.goto("/");

  await page.getByLabel("物件価格（土地＋建物の総額）").fill("1700");
  await page.getByLabel("想定月額賃料").fill("15");
  await page.getByText("シミュレーション実行").click();
  await expect(page.getByText("9.89", { exact: true })).toBeVisible({ timeout: 10_000 });

  const downloadPromise = page.waitForEvent("download", { timeout: 30_000 });
  await page.getByText("PDFレポート出力").click();

  const download = await downloadPromise;
  expect(download.suggestedFilename()).toMatch(/\.pdf$/);
});

test("@p3 PDF出力：RISK判定（criticalErrors=REJECT）でPDFが生成される", async ({ page }) => {
  // analyze-response-critical.json: DSCR=0.72, criticalErrors=[REJECT]
  // StatusSummary が "リスク" になる状態でPDF生成が例外なく完了することを確認
  await setupApiMocks(page, {
    analyze: { status: 200, body: analyzeCriticalFixture },
  });
  await page.addInitScript((key) => localStorage.setItem(key, "1"), ONBOARDING_KEY);
  await page.goto("/");

  await page.getByLabel("物件価格（土地＋建物の総額）").fill("1700");
  await page.getByLabel("想定月額賃料").fill("15");
  await page.getByText("シミュレーション実行").click();
  // criticalFixture では grossYield=0.033 → "3.30%" が表示される
  await expect(page.getByText("3.30", { exact: false })).toBeVisible({ timeout: 10_000 });

  const downloadPromise = page.waitForEvent("download", { timeout: 30_000 });
  await page.getByText("PDFレポート出力").click();

  const download = await downloadPromise;
  expect(download.suggestedFilename()).toMatch(/\.pdf$/);
});
