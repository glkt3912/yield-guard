import type {
  InvestmentInput,
  InvestmentResult,
  YearlyResult,
} from "@/types/investment";

const C = {
  primary: "#1a56db",
  headerBg: "#1e3a5f",
  rowEven: "#f0f4f8",
  border: "#e2e8f0",
  danger: "#dc2626",
  safe: "#16a34a",
  text: "#1e293b",
  muted: "#64748b",
  white: "#ffffff",
} as const;

// Sanitize string values before embedding in PDF content to prevent injection
function s(v: string | number | undefined): string {
  if (v === undefined || v === null) return "";
  return String(v).replace(/[<>&"'\\]/g, "");
}

function fmt(v: number): string {
  if (Math.abs(v) >= 1_000_000) return `${(v / 1_000_000).toFixed(1)}百万円`;
  return `${v.toLocaleString("ja-JP")}円`;
}

function pct(v: number): string {
  return `${(v * 100).toFixed(2)}%`;
}

function today(): string {
  const d = new Date();
  return `${d.getFullYear()}年${d.getMonth() + 1}月${d.getDate()}日`;
}

// Fonts are served from our own domain (public/fonts/) — no CDN dependency
const FONTS = {
  regular: "/fonts/NotoSansJP-Regular.woff2",
  bold: "/fonts/NotoSansJP-Bold.woff2",
} as const;

type FontKey = keyof typeof FONTS;
const fontCache: Partial<Record<FontKey, string>> = {};

async function loadFont(key: FontKey): Promise<string> {
  if (fontCache[key]) return fontCache[key]!;
  const url = FONTS[key];
  const res = await fetch(url);
  if (!res.ok) throw new Error(`Font fetch failed: ${res.status}`);
  const buf = await res.arrayBuffer();
  const bytes = new Uint8Array(buf);
  // Chunk-based btoa to avoid call-stack overflow on large font files
  const CHUNK = 8192;
  let binary = "";
  for (let i = 0; i < bytes.length; i += CHUNK) {
    binary += String.fromCharCode(...bytes.subarray(i, i + CHUNK));
  }
  fontCache[key] = btoa(binary);
  return fontCache[key]!;
}

function infoRow(label: string, value: string) {
  return {
    columns: [
      { text: label, width: "40%", color: C.muted, fontSize: 9 },
      { text: s(value), width: "60%", bold: true, fontSize: 9, alignment: "right" as const },
    ],
    marginBottom: 3,
  };
}

function kpiBlock(label: string, value: string, color?: string) {
  return {
    stack: [
      { text: label, fontSize: 7, color: C.muted, marginBottom: 2 },
      { text: value, fontSize: 15, bold: true, color: color ?? C.primary },
    ],
    margin: [0, 0, 6, 6],
  };
}

function sectionTitle(text: string, pageBreak = false) {
  return {
    text,
    fontSize: 13,
    bold: true,
    color: C.headerBg,
    marginBottom: 10,
    marginTop: 8,
    ...(pageBreak ? { pageBreak: "before" as const } : {}),
  };
}

function subTitle(text: string) {
  return { text, fontSize: 11, bold: true, color: C.headerBg, marginBottom: 8, marginTop: 6 };
}

function hLineTable(body: unknown[][]) {
  return {
    table: { widths: ["*", "auto"], body },
    layout: "lightHorizontalLines" as const,
    marginBottom: 12,
  };
}

function twoCol(label: string, value: string, bold = false, color?: string) {
  return [
    { text: label, fontSize: 8, bold, color: color ? undefined : C.text },
    { text: s(value), fontSize: 8, bold, color: color ?? C.text, alignment: "right" as const },
  ];
}

export async function downloadReportPDF(
  input: InvestmentInput,
  result: InvestmentResult
): Promise<void> {
  const [pdfMakeModule, regularB64, boldB64] = await Promise.all([
    import("pdfmake/build/pdfmake"),
    loadFont("regular"),
    loadFont("bold"),
  ]);

  const pdfMake = ((pdfMakeModule as Record<string, unknown>).default ?? pdfMakeModule) as {
    virtualfs: { writeFileSync: (name: string, data: string) => void };
    fonts: Record<string, unknown>;
    createPdf: (def: unknown) => { download: (name: string) => void };
  };

  // pdfmake 0.3.x uses virtualfs.writeFileSync (not pdfMake.vfs)
  pdfMake.virtualfs.writeFileSync("NotoSansJP-Regular.woff2", regularB64);
  pdfMake.virtualfs.writeFileSync("NotoSansJP-Bold.woff2", boldB64);
  pdfMake.fonts = {
    NotoSansJP: {
      normal: "NotoSansJP-Regular.woff2",
      bold: "NotoSansJP-Bold.woff2",
      italics: "NotoSansJP-Regular.woff2",
      bolditalics: "NotoSansJP-Bold.woff2",
    },
  };

  const date = today();
  const year1 = result.yearlyResults[0];
  const dscr =
    year1?.annualLoanPayment > 0 ? year1.annualRent / year1.annualLoanPayment : 0;
  const ltv =
    result.totalInvestment > 0 ? input.loanAmount / result.totalInvestment : 0;
  // Limit rows to prevent unbounded table generation
  const cfRows: YearlyResult[] = result.yearlyResults.slice(0, Math.min(input.holdingYears, 10));

  const thCell = (text: string, align: "left" | "right" | "center" = "right") => ({
    text,
    fontSize: 8,
    bold: true,
    color: C.white,
    fillColor: C.headerBg,
    alignment: align,
  });

  const tdCell = (
    text: string,
    idx: number,
    options: { align?: "left" | "right"; color?: string } = {}
  ) => ({
    text,
    fontSize: 8,
    alignment: options.align ?? "right",
    color: options.color ?? C.text,
    fillColor: idx % 2 === 0 ? C.rowEven : C.white,
  });

  const tableLayout = {
    hLineWidth: () => 0.5,
    vLineWidth: () => 0,
    hLineColor: () => C.border,
  };

  const docDef = {
    pageSize: "A4",
    pageMargins: [36, 36, 36, 48],
    defaultStyle: { font: "NotoSansJP", fontSize: 9, color: C.text },
    footer: () => ({
      text: "本資料は yield-guard による試算です。実際の投資判断は専門家にご相談ください。",
      fontSize: 7,
      color: C.muted,
      alignment: "center",
      margin: [36, 4],
    }),
    content: [
      // ── Page 1: Cover ──────────────────────────────────────────────
      {
        text: "不動産投資分析レポート",
        fontSize: 22,
        bold: true,
        color: C.headerBg,
        marginBottom: 6,
      },
      { text: "Real Estate Investment Analysis Report", fontSize: 12, color: C.muted, marginBottom: 24 },

      { text: "物件概要", fontSize: 9, bold: true, color: C.muted, marginBottom: 8 },
      infoRow("物件価格（土地）", fmt(input.landPrice)),
      infoRow("建物費用", fmt(input.buildingCost)),
      infoRow("築年数", input.buildingAge === 0 ? "新築" : `${input.buildingAge}年`),
      infoRow("構造", s(input.buildingType)),
      ...(input.stationMinutes > 0 ? [infoRow("最寄り駅徒歩", `${input.stationMinutes}分`)] : []),
      infoRow("月額賃料", fmt(input.monthlyRent)),
      infoRow("想定空室率", pct(input.vacancyRate)),
      infoRow("ローン金額", fmt(input.loanAmount)),
      infoRow("ローン金利", pct(input.annualLoanRate)),
      infoRow("ローン期間", `${input.loanYears}年`),

      { text: "分析情報", fontSize: 9, bold: true, color: C.muted, marginBottom: 8, marginTop: 16 },
      infoRow("分析実施日", date),
      infoRow("総投資額", fmt(result.totalInvestment)),
      infoRow("表面利回り", pct(result.grossYield)),

      // ── Page 2: Summary ────────────────────────────────────────────
      sectionTitle("P1 - 投資サマリー", true),
      {
        columns: [
          kpiBlock("表面利回り", `${(result.grossYield * 100).toFixed(2)}%`),
          kpiBlock("実効利回り（税引後）", `${(result.netYield * 100).toFixed(2)}%`),
          kpiBlock("DSCR（1年目）", dscr.toFixed(2), dscr >= 1.0 ? C.safe : C.danger),
          kpiBlock("LTV", `${(ltv * 100).toFixed(1)}%`, ltv <= 0.8 ? C.safe : C.danger),
        ],
        marginBottom: 6,
      },
      {
        columns: [
          kpiBlock("総投資額", `${(result.totalInvestment / 1_000_000).toFixed(1)}百万円`),
          kpiBlock("月額賃料", `${(input.monthlyRent / 10_000).toFixed(0)}万円/月`),
          kpiBlock(
            "デッドクロス年",
            result.deadCrossYear > 0 ? `${result.deadCrossYear}年目` : "なし",
            result.deadCrossYear > 0 ? C.danger : C.safe
          ),
          kpiBlock(
            `${(result.yieldTarget * 100).toFixed(0)}%基準`,
            result.isAboveYieldTarget ? "達成" : "未達",
            result.isAboveYieldTarget ? C.safe : C.danger
          ),
        ],
        marginBottom: 16,
      },

      subTitle(`出口戦略（${input.holdingYears}年後売却）`),
      hLineTable([
        twoCol("想定売却価格", fmt(result.exitSalePrice)),
        twoCol("譲渡所得税", fmt(result.exitTransferTax)),
        twoCol("売却手取り", fmt(result.exitNetProceeds)),
        twoCol(
          "トータルエクイティ（CF累積＋売却益）",
          fmt(result.exitTotalEquity),
          true,
          result.exitTotalEquity >= 0 ? C.safe : C.danger
        ),
      ]),

      // ── Page 3: Cash Flow ──────────────────────────────────────────
      sectionTitle("P2 - 10年間キャッシュフロー", true),
      {
        table: {
          headerRows: 1,
          widths: [20, "*", "*", "*", "*", "*", "*", "*"],
          body: [
            [
              thCell("年", "left"),
              thCell("賃料収入"),
              thCell("ローン返済"),
              thCell("運営経費"),
              thCell("税引前CF"),
              thCell("税引後CF"),
              thCell("累積CF"),
              thCell("残債"),
            ],
            ...cfRows.map((r, idx) => [
              tdCell(`${r.year}年`, idx, { align: "left" }),
              tdCell(fmt(r.annualRent), idx),
              tdCell(fmt(r.annualLoanPayment), idx),
              tdCell(fmt(r.annualExpenses), idx),
              tdCell(fmt(r.cashFlow), idx, { color: r.cashFlow < 0 ? C.danger : C.text }),
              tdCell(fmt(r.afterTaxCashFlow), idx, { color: r.afterTaxCashFlow < 0 ? C.danger : C.text }),
              tdCell(fmt(r.cumulativeCashFlow), idx, { color: r.cumulativeCashFlow < 0 ? C.danger : C.text }),
              tdCell(fmt(r.remainingLoanBalance), idx),
            ]),
          ],
        },
        layout: tableLayout,
      },

      // ── Page 4: Stress Test ────────────────────────────────────────
      ...(result.stressScenarios.length > 0
        ? [
            sectionTitle("P3 - ストレステスト結果", true),
            {
              table: {
                headerRows: 1,
                widths: ["*", "auto", "auto", "auto", 36],
                body: [
                  [
                    thCell("シナリオ", "left"),
                    thCell("総CF"),
                    thCell("DSCR"),
                    thCell("回収年"),
                    thCell("判定", "center"),
                  ],
                  ...result.stressScenarios.map((sc, idx) => [
                    tdCell(s(sc.label), idx, { align: "left" }),
                    tdCell(fmt(sc.totalCashFlow), idx, {
                      color: sc.totalCashFlow < 0 ? C.danger : C.text,
                    }),
                    tdCell(sc.dscr.toFixed(2), idx, {
                      color: sc.dscr < 1.0 ? C.danger : C.text,
                    }),
                    tdCell(sc.breakEvenYear > 0 ? `${sc.breakEvenYear}年` : "－", idx),
                    {
                      text: sc.isSafe ? "安全" : "危険",
                      fontSize: 8,
                      bold: true,
                      alignment: "center",
                      color: sc.isSafe ? C.safe : C.danger,
                      fillColor: idx % 2 === 0 ? C.rowEven : C.white,
                    },
                  ]),
                ],
              },
              layout: tableLayout,
            },
          ]
        : []),

      // ── Page 5: Cost Breakdown ─────────────────────────────────────
      ...(result.acquisitionCosts
        ? [
            sectionTitle("P4 - 取得コスト内訳", true),
            subTitle("初期投資内訳"),
            hLineTable([
              twoCol("土地価格", fmt(input.landPrice)),
              twoCol("建物費用", fmt(input.buildingCost)),
              twoCol("諸経費合計", fmt(result.miscExpenses)),
              twoCol("総投資額", fmt(result.totalInvestment), true, C.primary),
            ]),
            subTitle("取得時諸経費の明細"),
            hLineTable([
              twoCol("仲介手数料（税込）", fmt(result.acquisitionCosts.brokerageFee)),
              twoCol("印紙税", fmt(result.acquisitionCosts.stampDuty)),
              twoCol("登録免許税", fmt(result.acquisitionCosts.registrationTax)),
              twoCol("不動産取得税（概算）", fmt(result.acquisitionCosts.realEstateAcquisitionTax)),
              ...(result.acquisitionCosts.propertyTaxProration > 0
                ? [twoCol("固定資産税日割り精算", fmt(result.acquisitionCosts.propertyTaxProration))]
                : []),
              twoCol("諸経費合計", fmt(result.acquisitionCosts.total), true, C.primary),
            ]),
            ...(result.yearlyResults.length > 0
              ? [
                  subTitle("年間費用の内訳（1年目）"),
                  hLineTable([
                    ...(result.yearlyResults[0].annualLoanPayment > 0
                      ? [twoCol("ローン返済", fmt(result.yearlyResults[0].annualLoanPayment))]
                      : []),
                    twoCol("運営経費", fmt(result.yearlyResults[0].annualExpenses)),
                    ...(result.yearlyResults[0].incomeTax > 0
                      ? [twoCol("所得税", fmt(result.yearlyResults[0].incomeTax))]
                      : []),
                    twoCol(
                      "税引後キャッシュフロー（1年目）",
                      fmt(result.yearlyResults[0].afterTaxCashFlow),
                      true,
                      result.yearlyResults[0].afterTaxCashFlow >= 0 ? C.safe : C.danger
                    ),
                  ]),
                ]
              : []),
          ]
        : []),
    ],
  };

  const fileName = `yield-guard-report-${new Date().toISOString().slice(0, 10).replace(/-/g, "")}.pdf`;
  pdfMake.createPdf(docDef).download(fileName);
}
