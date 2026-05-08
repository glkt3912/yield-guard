import { describe, it, expect } from "vitest";
import { render, screen } from "@testing-library/react";
import { HazardAlertBanner } from "@/components/HazardAlertBanner";
import type { UrbanRisk } from "@/types/investment";

const errorRisk: UrbanRisk = {
  code: "flood_major",
  level: "ERROR",
  title: "大規模洪水浸水区域",
  description: "浸水深3m以上の区域です",
};

const warningRisk: UrbanRisk = {
  code: "tsunami_low",
  level: "WARNING",
  title: "津波浸水想定区域",
  description: "津波により浸水するおそれがある区域です",
};

const infoRisk: UrbanRisk = {
  code: "info_zone",
  level: "INFO",
  title: "情報のみ",
  description: "参考情報です",
};

describe("HazardAlertBanner", () => {
  it("ERROR レベルのリスクがあるとき赤いアラートバナーとリスクタイトルを表示する", () => {
    render(<HazardAlertBanner hazardRisks={[errorRisk]} />);
    expect(screen.getByRole("alert")).toBeInTheDocument();
    expect(screen.getByText(/大規模洪水浸水区域/)).toBeInTheDocument();
    const alertDiv = screen
      .getAllByText(/重大ハザード/)
      .find((el) => el.closest(".border-red-500"));
    expect(alertDiv).toBeTruthy();
  });

  it("WARNING レベルのリスクがあるとき黄色いアラートバナーを表示する", () => {
    render(<HazardAlertBanner hazardRisks={[warningRisk]} />);
    expect(screen.getByRole("alert")).toBeInTheDocument();
    expect(screen.getByText(/津波浸水想定区域/)).toBeInTheDocument();
    const alertDiv = screen
      .getAllByText(/ハザード注意/)
      .find((el) => el.closest(".border-yellow-400"));
    expect(alertDiv).toBeTruthy();
  });

  it("INFO レベルのリスクのみのとき何もレンダリングしない", () => {
    const { container } = render(<HazardAlertBanner hazardRisks={[infoRisk]} />);
    expect(container).toBeEmptyDOMElement();
  });

  it("hazardRisks と externalUrbanRisks に同じ code があるとき1件のみ表示する（dedup）", () => {
    const duplicate: UrbanRisk = { ...errorRisk };
    render(<HazardAlertBanner hazardRisks={[errorRisk]} externalUrbanRisks={[duplicate]} />);
    const titles = screen.getAllByText(/大規模洪水浸水区域/);
    expect(titles).toHaveLength(1);
  });

  it("flood_major コードのとき「洪水」ラベルを表示する", () => {
    render(<HazardAlertBanner hazardRisks={[errorRisk]} />);
    expect(screen.getByText(/（洪水）/)).toBeInTheDocument();
  });

  it("tsunami_xxx コードのとき「津波」ラベルを表示する", () => {
    render(<HazardAlertBanner hazardRisks={[warningRisk]} />);
    expect(screen.getByText(/（津波）/)).toBeInTheDocument();
  });

  it("マッチしないコードのとき hazard type ラベルを表示しない", () => {
    const unknownRisk: UrbanRisk = {
      code: "unknown_risk",
      level: "ERROR",
      title: "未知のリスク",
      description: "説明なし",
    };
    render(<HazardAlertBanner hazardRisks={[unknownRisk]} />);
    expect(screen.queryByText(/（.*）/)).not.toBeInTheDocument();
    expect(screen.getByText(/未知のリスク/)).toBeInTheDocument();
  });

  it("両配列が null / undefined のとき何もレンダリングしない", () => {
    const { container } = render(
      <HazardAlertBanner hazardRisks={null} externalUrbanRisks={null} />
    );
    expect(container).toBeEmptyDOMElement();
  });

  it("外側ラッパーに role=alert が付与され、内側の div に role=listitem がない", () => {
    render(<HazardAlertBanner hazardRisks={[errorRisk, warningRisk]} />);
    expect(screen.getByRole("alert")).toBeInTheDocument();
    expect(screen.queryAllByRole("listitem")).toHaveLength(0);
  });
});
