import { describe, it, expect, vi } from "vitest";
import { render, screen } from "@testing-library/react";
import userEvent from "@testing-library/user-event";
import { SimulationModeToggle } from "@/components/SimulationModeToggle";

describe("SimulationModeToggle", () => {
  it("かんたん判定とくわしく分析のラジオボタンが表示される", () => {
    render(<SimulationModeToggle mode="quick" onChange={vi.fn()} />);
    expect(screen.getByRole("radio", { name: /かんたん判定/ })).toBeInTheDocument();
    expect(screen.getByRole("radio", { name: /くわしく分析/ })).toBeInTheDocument();
  });

  it("mode='quick' のときかんたん判定が選択状態になる", () => {
    render(<SimulationModeToggle mode="quick" onChange={vi.fn()} />);
    expect(screen.getByRole("radio", { name: /かんたん判定/ })).toHaveAttribute(
      "aria-checked",
      "true"
    );
    expect(screen.getByRole("radio", { name: /くわしく分析/ })).toHaveAttribute(
      "aria-checked",
      "false"
    );
  });

  it("mode='full' のときくわしく分析が選択状態になる", () => {
    render(<SimulationModeToggle mode="full" onChange={vi.fn()} />);
    expect(screen.getByRole("radio", { name: /くわしく分析/ })).toHaveAttribute(
      "aria-checked",
      "true"
    );
    expect(screen.getByRole("radio", { name: /かんたん判定/ })).toHaveAttribute(
      "aria-checked",
      "false"
    );
  });

  it("かんたん判定ボタンをクリックすると onChange('quick') が呼ばれる", async () => {
    const onChange = vi.fn();
    render(<SimulationModeToggle mode="full" onChange={onChange} />);
    await userEvent.click(screen.getByRole("radio", { name: /かんたん判定/ }));
    expect(onChange).toHaveBeenCalledWith("quick");
  });

  it("くわしく分析ボタンをクリックすると onChange('full') が呼ばれる", async () => {
    const onChange = vi.fn();
    render(<SimulationModeToggle mode="quick" onChange={onChange} />);
    await userEvent.click(screen.getByRole("radio", { name: /くわしく分析/ }));
    expect(onChange).toHaveBeenCalledWith("full");
  });

  it("group role と aria-label が設定されている", () => {
    render(<SimulationModeToggle mode="quick" onChange={vi.fn()} />);
    expect(screen.getByRole("group", { name: "シミュレーションモード" })).toBeInTheDocument();
  });

  it("「2項目で即判定」サブラベルが表示される", () => {
    render(<SimulationModeToggle mode="quick" onChange={vi.fn()} />);
    expect(screen.getByText("2項目で即判定")).toBeInTheDocument();
  });

  it("「17項目で精密分析」サブラベルが表示される", () => {
    render(<SimulationModeToggle mode="quick" onChange={vi.fn()} />);
    expect(screen.getByText("17項目で精密分析")).toBeInTheDocument();
  });

  it("className prop が渡されると div に反映される", () => {
    const { container } = render(
      <SimulationModeToggle mode="quick" onChange={vi.fn()} className="mt-4" />
    );
    expect(container.firstChild).toHaveClass("mt-4");
  });
});
