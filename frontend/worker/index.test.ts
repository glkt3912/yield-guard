import { describe, it, expect } from "vitest";
import { shareDescription } from "./index";

describe("shareDescription", () => {
  it("クイックモードは totalPrice を使う", () => {
    expect(shareDescription(new URLSearchParams("totalPrice=3000&rent=80000"))).toBe(
      "物件価格 3,000万円 / 月額賃料 80,000円"
    );
  });
  it("フルモードは landPrice + buildingCost を合算する", () => {
    expect(shareDescription(new URLSearchParams("landPrice=1000&buildingCost=500"))).toBe(
      "物件価格 1,500万円"
    );
  });
  it("価格が無ければ null", () => {
    expect(shareDescription(new URLSearchParams("rent=80000"))).toBeNull();
    expect(shareDescription(new URLSearchParams("totalPrice=abc"))).toBeNull();
  });
});
