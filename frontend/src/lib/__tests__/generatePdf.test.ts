import { describe, it, expect, vi, beforeEach } from "vitest";
import { downloadReportPDF } from "@/lib/generatePdf";
import { makeInput, makeResult } from "@/components/__tests__/helpers";

// Mock pdfmake – avoids loading the full browser build in jsdom
const mockDownload = vi.fn();
const mockCreatePdf = vi.fn(() => ({ download: mockDownload }));

vi.mock("pdfmake/build/pdfmake", () => ({
  default: {
    createPdf: mockCreatePdf,
    vfs: {},
    fonts: {},
  },
}));

// Mock fetch for the Noto Sans JP font request
function makeFontFetchMock() {
  const fakeBytes = new Uint8Array([1, 2, 3]);
  return vi.fn().mockResolvedValue({
    ok: true,
    status: 200,
    arrayBuffer: () => Promise.resolve(fakeBytes.buffer),
  });
}

beforeEach(() => {
  vi.clearAllMocks();
});

describe("downloadReportPDF", () => {
  it("resolves without throwing for valid input", async () => {
    global.fetch = makeFontFetchMock();
    const input = makeInput();
    const result = makeResult();

    await expect(downloadReportPDF(input, result)).resolves.toBeUndefined();
    expect(mockCreatePdf).toHaveBeenCalledOnce();
    expect(mockDownload).toHaveBeenCalledOnce();
  });

  it("resolves with empty stressScenarios", async () => {
    global.fetch = makeFontFetchMock();
    const input = makeInput();
    const result = makeResult({ stressScenarios: [] });

    await expect(downloadReportPDF(input, result)).resolves.toBeUndefined();
  });

  it("resolves when acquisitionCosts is undefined", async () => {
    global.fetch = makeFontFetchMock();
    const input = makeInput();
    const result = makeResult({ acquisitionCosts: undefined });

    await expect(downloadReportPDF(input, result)).resolves.toBeUndefined();
  });

  it("resolves for newly built property (buildingAge = 0)", async () => {
    global.fetch = makeFontFetchMock();
    const input = makeInput({ buildingAge: 0 });
    const result = makeResult();

    await expect(downloadReportPDF(input, result)).resolves.toBeUndefined();
  });

  it("sanitizes special characters in buildingType before embedding in PDF", async () => {
    global.fetch = makeFontFetchMock();
    const input = makeInput({ buildingType: '<script>alert(1)</script>' as never });
    const result = makeResult();

    await downloadReportPDF(input, result);

    const docDef = (mockCreatePdf.mock.calls as unknown[][][])[0]?.[0];
    const json = JSON.stringify(docDef);
    expect(json).not.toContain("<script>");
    expect(json).not.toContain("</script>");
  });

  it("sanitizes special characters in stress scenario labels", async () => {
    global.fetch = makeFontFetchMock();
    const input = makeInput();
    const result = makeResult({
      stressScenarios: [
        {
          label: '"><img src=x onerror=alert(1)>',
          interestRateDelta: 0,
          vacancyRateDelta: 0,
          totalCashFlow: 1_000_000,
          dscr: 1.2,
          breakEvenYear: 5,
          isSafe: true,
        },
      ],
    });

    await downloadReportPDF(input, result);

    const docDef = (mockCreatePdf.mock.calls as unknown[][][])[0]?.[0];
    const json = JSON.stringify(docDef);
    // '<' and '>' are stripped — HTML tag structure is broken, making it harmless in PDF text
    expect(json).not.toContain("<img");
    expect(json).not.toContain("</");
  });

  it("throws when font fetch fails", async () => {
    // Reset module registry to clear the in-memory fontCache, then re-import
    vi.resetModules();
    global.fetch = vi.fn().mockResolvedValue({ ok: false, status: 404 });
    const { downloadReportPDF: freshFn } = await import("@/lib/generatePdf");
    const input = makeInput();
    const result = makeResult();

    await expect(freshFn(input, result)).rejects.toThrow("Font fetch failed");
  });

  it("passes the correct filename to download", async () => {
    global.fetch = makeFontFetchMock();
    const input = makeInput();
    const result = makeResult();

    await downloadReportPDF(input, result);

    const fileName = ((mockDownload.mock.calls as unknown[][][])[0]?.[0] as unknown) as string;
    expect(fileName).toMatch(/^yield-guard-report-\d{8}\.pdf$/);
  });
});
