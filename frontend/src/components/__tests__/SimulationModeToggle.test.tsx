import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { SimulationModeToggle } from "@/components/SimulationModeToggle";

describe("SimulationModeToggle", () => {
  it("クイックと詳細のラジオボタンが表示される", () => {
    render(<SimulationModeToggle mode="quick" onChange={vi.fn()} />);
    expect(screen.getByRole("radio", { name: /クイック/ })).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: /詳細/ })).toBeInTheDocument();
  });

  it("mode='quick' のときクイックが選択状態になる", () => {
    render(<SimulationModeToggle mode="quick" onChange={vi.fn()} />);
    expect(screen.getByRole("radio", { name: /クイック/ })).toHaveAttribute("aria-checked", "true");
    expect(screen.getByRole("radio", { name: /詳細/ })).toHaveAttribute("aria-checked", "false");
  });

  it("mode='full' のとき詳細が選択状態になる", () => {
    render(<SimulationModeToggle mode="full" onChange={vi.fn()} />);
    expect(screen.getByRole("radio", { name: /詳細/ })).toHaveAttribute("aria-checked", "true");
    expect(screen.getByRole("radio", { name: /クイック/ })).toHaveAttribute(
      "aria-checked",
      "false"
    );
  });

  it("クイックボタンをクリックすると onChange('quick') が呼ばれる", async () => {
    const onChange = vi.fn();
    render(<SimulationModeToggle mode="full" onChange={onChange} />);
    await userEvent.click(screen.getByRole("radio", { name: /クイック/ }));
    expect(onChange).toHaveBeenCalledWith("quick");
  });

  it("詳細ボタンをクリックすると onChange('full') が呼ばれる", async () => {
    const onChange = vi.fn();
    render(<SimulationModeToggle mode="quick" onChange={onChange} />);
    await userEvent.click(screen.getByRole("radio", { name: /詳細/ }));
    expect(onChange).toHaveBeenCalledWith("full");
  });

  it("group role と aria-label が設定されている", () => {
    render(<SimulationModeToggle mode="quick" onChange={vi.fn()} />);
    expect(screen.getByRole("group", { name: "シミュレーションモード" })).toBeInTheDocument();
  });

  it("「内覧でサッと試す」サブラベルが表示される", () => {
    render(<SimulationModeToggle mode="quick" onChange={vi.fn()} />);
    expect(screen.getByText("内覧でサッと試す")).toBeInTheDocument();
  });

  it("「じっくり分析する」サブラベルが表示される", () => {
    render(<SimulationModeToggle mode="quick" onChange={vi.fn()} />);
    expect(screen.getByText("じっくり分析する")).toBeInTheDocument();
  });

  it("className prop が渡されると div に反映される", () => {
    const { container } = render(
      <SimulationModeToggle mode="quick" onChange={vi.fn()} className="mt-4" />
    );
    expect(container.firstChild).toHaveClass("mt-4");
  });
});
