import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import DeadCrossChart from "@/components/DeadCrossChart";
import { makeResult } from "./helpers";

describe("DeadCrossChart", () => {
  it("デッドクロスがない場合は成功バッジを表示する", () => {
    const result = makeResult({ deadCrossYear: 0 });
    const { container } = render(<DeadCrossChart result={result} />);
    expect(container.textContent).toContain("デッドクロスなし（35年以内）");
  });

  it("デッドクロスがある場合は警告バッジを表示する", () => {
    const result = makeResult({ deadCrossYear: 10 });
    const { container } = render(<DeadCrossChart result={result} />);
    expect(container.textContent).toContain("10年目〜");
    expect(container.textContent).toContain("デッドクロスゾーン");
  });

  it("35年超のデッドクロス年はデッドクロスなしとして扱う", () => {
    const result = makeResult({ deadCrossYear: 40 });
    const { container } = render(<DeadCrossChart result={result} />);
    expect(container.textContent).toContain("デッドクロスなし（35年以内）");
  });

  it("デッドクロスがある場合は警告テキストを表示する", () => {
    const result = makeResult({ deadCrossYear: 15 });
    render(<DeadCrossChart result={result} />);
    expect(screen.getByText(/15年目以降、所得税の実質負担が増加します/)).toBeInTheDocument();
  });
});
