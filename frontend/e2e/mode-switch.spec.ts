import { test, expect } from "@playwright/test";
import { setupApiMocks } from "./helpers/routes";

const ONBOARDING_KEY = "yield-guard:onboarded";

test.beforeEach(async ({ page }) => {
  await setupApiMocks(page);
  await page.addInitScript((key) => localStorage.setItem(key, "1"), ONBOARDING_KEY);
  await page.goto("/");
});

test("@p2 モード切替：Quick分析後に詳細へ切り替えると結果がクリアされバナーが表示される", async ({
  page,
}) => {
  await page.getByLabel("物件価格（土地＋建物の総額）").fill("1700");
  await page.getByLabel("想定月額賃料").fill("15");
  await page.getByText("シミュレーション実行").click();
  await expect(page.getByText("9.89%")).toBeVisible({ timeout: 10_000 });

  // 詳細モードに切り替え
  await page.getByRole("radio", { name: "詳細" }).click();

  await expect(
    page.getByText(
      "モードを切り替えたため、結果をクリアしました。再度シミュレーションを実行してください。"
    )
  ).toBeVisible({ timeout: 3_000 });
  // 結果値が消えている
  await expect(page.getByText("9.89%")).not.toBeVisible();
});
