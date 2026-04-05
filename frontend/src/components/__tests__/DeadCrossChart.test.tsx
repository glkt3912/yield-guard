import { render, screen } from "@testing-library/react";
import { DeadCrossChart } from "@/components/DeadCrossChart";
import { makeResult } from "./helpers";

describe("DeadCrossChart", () => {
  it("デッドクロスがない場合は成功バッジを表示する", () => {
    const result = makeResult({ deadCrossYear: 0 });
    render(<DeadCrossChart result={result} />);
    expect(screen.getByText("デッドクロスなし（35年以内）")).toBeInTheDocument();
  });

  it("デッドクロスがある場合は警告バッジを表示する", () => {
    const result = makeResult({ deadCrossYear: 10 });
    render(<DeadCrossChart result={result} />);
    expect(screen.getByText("10年目〜デッドクロスゾーン")).toBeInTheDocument();
  });

  it("35年超のデッドクロス年はデッドクロスなしとして扱う", () => {
    const result = makeResult({ deadCrossYear: 40 });
    render(<DeadCrossChart result={result} />);
    expect(screen.getByText("デッドクロスなし（35年以内）")).toBeInTheDocument();
  });

  it("デッドクロスがある場合は警告テキストを表示する", () => {
    const result = makeResult({ deadCrossYear: 15 });
    render(<DeadCrossChart result={result} />);
    expect(screen.getByText(/15年目以降、所得税の実質負担が増加します/)).toBeInTheDocument();
  });
});
