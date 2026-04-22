import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { CriticalErrorBanner } from "@/components/CriticalErrorBanner";
import type { CriticalError } from "@/types/investment";

const rejectError: CriticalError = {
  code: "LAND_VALUE_GUARD",
  status: "REJECT",
  message: "積算評価額が購入総額の50%未満です",
};

const warningError: CriticalError = {
  code: "DEADCROSS_EARLY",
  status: "WARNING",
  message: "10年以内にデッドクロスが発生します",
};

describe("CriticalErrorBanner", () => {
  it("errors が空配列のとき何もレンダリングしない", () => {
    const { container } = render(<CriticalErrorBanner errors={[]} />);
    expect(container).toBeEmptyDOMElement();
  });

  it("errors が undefined のとき何もレンダリングしない", () => {
    const { container } = render(
      <CriticalErrorBanner errors={undefined as unknown as CriticalError[]} />
    );
    expect(container).toBeEmptyDOMElement();
  });

  it("REJECT エラーのとき「⛔ 一発退場」ラベルとエラーコードを表示する", () => {
    render(<CriticalErrorBanner errors={[rejectError]} />);
    expect(screen.getByText(/⛔ 一発退場.*LAND_VALUE_GUARD/)).toBeInTheDocument();
    expect(screen.getByText("積算評価額が購入総額の50%未満です")).toBeInTheDocument();
  });

  it("WARNING エラーのとき「⚠ 警告」ラベルとエラーコードを表示する", () => {
    render(<CriticalErrorBanner errors={[warningError]} />);
    expect(screen.getByText(/⚠ 警告.*DEADCROSS_EARLY/)).toBeInTheDocument();
    expect(screen.getByText("10年以内にデッドクロスが発生します")).toBeInTheDocument();
  });

  it("複数エラーがあるとき全件表示する", () => {
    render(<CriticalErrorBanner errors={[rejectError, warningError]} />);
    expect(screen.getByText(/LAND_VALUE_GUARD/)).toBeInTheDocument();
    expect(screen.getByText(/DEADCROSS_EARLY/)).toBeInTheDocument();
  });

  it("各バナーに role=alert が付与されている", () => {
    render(<CriticalErrorBanner errors={[rejectError, warningError]} />);
    const alerts = screen.getAllByRole("alert");
    expect(alerts).toHaveLength(2);
  });

  it("REJECT のとき赤系クラスが適用されている", () => {
    render(<CriticalErrorBanner errors={[rejectError]} />);
    const alert = screen.getByRole("alert");
    expect(alert.className).toContain("border-red-500");
    expect(alert.className).toContain("bg-red-50");
  });

  it("WARNING のとき黄系クラスが適用されている", () => {
    render(<CriticalErrorBanner errors={[warningError]} />);
    const alert = screen.getByRole("alert");
    expect(alert.className).toContain("border-yellow-400");
    expect(alert.className).toContain("bg-yellow-50");
  });
});
