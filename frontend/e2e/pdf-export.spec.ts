import { test, expect } from "@playwright/test";
import { analyzeCriticalFixture } from "./fixtures/analyze-response-critical";
import { SimulationPage } from "./pages/simulation.page";

// デフォルト fixture (analyze-response.json) には multiExitComparison が含まれるため
// beforeEach のセットアップで判定サマリー・出口比較テーブルの両セクションをカバーできる
// @p2 でPRスモーク（@p1|@p2）に含める。PDF生成を @p3 フルスイート（main限定）でしか
// 実行しないと、PDFを壊す変更がPRで緑のままマージされ main で初めて顕在化する（Issue #783）。
test.describe("PDF出力 — デフォルト fixture (OK判定 / multiExitComparison あり)", () => {
  test("@p2 ダウンロードイベントが発生しファイル名が .pdf で終わる", async ({ page }) => {
    const sim = new SimulationPage(page);
    await sim.setup();
    await sim.runQuickSimulation("1700", "15");
    await expect(sim.grossYield()).toHaveText("10.58");

    const downloadPromise = page.waitForEvent("download", { timeout: 30_000 });
    await page.getByText("PDFレポート出力").click();

    const download = await downloadPromise;
    expect(download.suggestedFilename()).toMatch(/\.pdf$/);
  });
});

// RISK判定（criticalErrors=REJECT）では StatusSummary が "リスク" になる
// マイナスエクイティの multiExitComparison でも PDF が例外なく生成されることを確認
test.describe("PDF出力 — RISK判定 fixture (criticalErrors=REJECT / マイナスエクイティ)", () => {
  test("@p2 RISK判定でPDFが生成される", async ({ page }) => {
    const sim = new SimulationPage(page);
    await sim.setup({
      analyze: { status: 200, body: analyzeCriticalFixture },
    });
    await page.getByLabel("物件価格（土地＋建物の総額）").fill("1700");
    await page.getByLabel("想定月額賃料").fill("15");
    await page.getByText("シミュレーション実行").click();
    // criticalFixture では RISK判定となり重大なリスクバナーが表示される
    await expect(page.getByRole("alert", { name: "重大なリスク" })).toBeVisible({ timeout: 10_000 });

    const downloadPromise = page.waitForEvent("download", { timeout: 30_000 });
    await page.getByText("PDFレポート出力").click();

    const download = await downloadPromise;
    expect(download.suggestedFilename()).toMatch(/\.pdf$/);
  });
});
