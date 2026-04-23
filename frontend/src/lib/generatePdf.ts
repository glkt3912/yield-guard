import type {
  InvestmentInput,
  InvestmentResult,
  YearlyResult,
} from "@/types/investment";
import { fmtYen, fmtPct, fmtDate, sanitize } from "./pdf/format";
import { calcVerdict } from "./pdf/verdict";
import { buildCfBarChartSvg, buildDeadCrossLineSvg, buildCostDonutSvg } from "./pdf/charts";

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

// TTF subset served from our own domain — pdfmake's pdfkit does not support woff2
const FONTS = {
  regular: "/fonts/NotoSansJP-Regular.ttf",
  bold: "/fonts/NotoSansJP-Bold.ttf",
} as const;

type FontKey = keyof typeof FONTS;
const fontCache: Partial<Record<FontKey, Uint8Array>> = {};

async function loadFont(key: FontKey): Promise<Uint8Array> {
  if (fontCache[key]) return fontCache[key]!;
  const url = FONTS[key];
  const res = await fetch(url);
  if (!res.ok) throw new Error(`Font fetch failed: ${res.status}`);
  fontCache[key] = new Uint8Array(await res.arrayBuffer());
  return fontCache[key]!;
}

function infoRow(label: string, value: string) {
  return {
    columns: [
      { text: label, width: "40%", color: C.muted, fontSize: 9, noWrap: true },
      { text: sanitize(value), width: "60%", bold: true, fontSize: 9, alignment: "right" as const },
    ],
    marginBottom: 3,
  };
}

function kpiBlock(label: string, value: string, color?: string) {
  return {
    stack: [
      // lineHeight: 1 + noWrap prevent pdfmake from splitting CJK glyphs with spurious spaces
      { text: label, fontSize: 7, color: C.muted, marginBottom: 2, lineHeight: 1, noWrap: true },
      { text: value, fontSize: 15, bold: true, color: color ?? C.primary, noWrap: true },
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
    { text: label, fontSize: 8, bold, color: color ? undefined : C.text, noWrap: true },
    { text: sanitize(value), fontSize: 8, bold, color: color ?? C.text, alignment: "right" as const },
  ];
}

export async function downloadReportPDF(
  input: InvestmentInput,
  result: InvestmentResult
): Promise<void> {
  const [pdfMakeModule, regularBytes, boldBytes] = await Promise.all([
    import("pdfmake/build/pdfmake"),
    loadFont("regular"),
    loadFont("bold"),
  ]);

  const pdfMake = ((pdfMakeModule as Record<string, unknown>).default ?? pdfMakeModule) as {
    virtualfs: { writeFileSync: (name: string, data: Uint8Array) => void };
    fonts: Record<string, unknown>;
    createPdf: (def: unknown) => { download: (name: string) => void };
  };

  // Pass raw binary (Uint8Array) — base64 strings are stored as ASCII bytes and fail font parsing
  pdfMake.virtualfs.writeFileSync("NotoSansJP-Regular.ttf", regularBytes);
  pdfMake.virtualfs.writeFileSync("NotoSansJP-Bold.ttf", boldBytes);
  pdfMake.fonts = {
    NotoSansJP: {
      normal: "NotoSansJP-Regular.ttf",
      bold: "NotoSansJP-Bold.ttf",
      italics: "NotoSansJP-Regular.ttf",
      bolditalics: "NotoSansJP-Bold.ttf",
    },
  };

  const date = fmtDate();
  const year1 = result.yearlyResults[0];
  const dscr =
    year1?.annualLoanPayment > 0 ? year1.annualRent / year1.annualLoanPayment : 0;
  const ltv =
    result.totalInvestment > 0 ? input.loanAmount / result.totalInvestment : 0;
  const dscrStress = result.stressScenarios.find((sc) => sc.label === "複合ストレス")?.dscr ?? 0;
  const verdict = calcVerdict(input, result, dscr, dscrStress);
  const baselineScenario = result.stressScenarios.find((sc) => sc.label === "ベースライン");
  const stressScenario = result.stressScenarios.find((sc) => sc.label === "複合ストレス");
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
    pageMargins: [36, 52, 36, 48],
    defaultStyle: { font: "NotoSansJP", fontSize: 9, color: C.text },
    info: {
      title: "不動産投資分析レポート",
      author: "yield-guard",
      subject: `物件分析 ${date}`,
      creator: "yield-guard",
    },
    header: (currentPage: number) => {
      if (currentPage === 1) return null;
      return {
        columns: [
          { text: "yield-guard 不動産投資分析レポート", fontSize: 7, color: C.muted, margin: [36, 16, 0, 0] },
          { text: date, fontSize: 7, color: C.muted, alignment: "right" as const, margin: [0, 16, 36, 0] },
        ],
      };
    },
    footer: (currentPage: number, pageCount: number) => ({
      columns: [
        {
          text: "本資料は yield-guard による試算です。実際の投資判断は専門家にご相談ください。",
          fontSize: 7,
          color: C.muted,
          margin: [36, 4, 0, 0],
        },
        {
          text: `${currentPage} / ${pageCount}`,
          fontSize: 7,
          color: C.muted,
          alignment: "right" as const,
          margin: [0, 4, 36, 0],
        },
      ],
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
      infoRow("物件価格（土地）", fmtYen(input.landPrice)),
      infoRow("建物費用", fmtYen(input.buildingCost)),
      infoRow("築年数", input.buildingAge === 0 ? "新築" : `${input.buildingAge}年`),
      infoRow("構造", sanitize(input.buildingType)),
      ...(input.stationMinutes > 0 ? [infoRow("最寄り駅徒歩", `${input.stationMinutes}分`)] : []),
      infoRow("月額賃料", fmtYen(input.monthlyRent)),
      infoRow("想定空室率", fmtPct(input.vacancyRate)),
      infoRow("ローン金額", fmtYen(input.loanAmount)),
      infoRow("ローン金利", fmtPct(input.annualLoanRate)),
      infoRow("ローン期間", `${input.loanYears}年`),

      { text: "分析情報", fontSize: 9, bold: true, color: C.muted, marginBottom: 8, marginTop: 16 },
      infoRow("分析実施日", date),
      infoRow("総投資額", fmtYen(result.totalInvestment)),
      infoRow("表面利回り", fmtPct(result.grossYield)),

      // ── Page 2: Summary (Executive) ────────────────────────────────
      sectionTitle("P1 - 投資サマリー", true),

      // 総合判定バッジ
      {
        columns: [
          {
            stack: [
              {
                text: verdict.label,
                fontSize: 20,
                bold: true,
                color: verdict.color,
              },
              {
                text: verdict.label,
                fontSize: 8,
                color: verdict.color,
                marginTop: 2,
              },
            ],
            width: "30%",
            margin: [0, 0, 12, 0],
          },
          {
            ul: verdict.reasons,
            fontSize: 8,
            color: C.text,
          },
        ],
        marginBottom: 12,
      },

      // KPI 2行×3列
      {
        columns: [
          kpiBlock("表面利回り", fmtPct(result.grossYield)),
          kpiBlock("実質利回り", fmtPct(result.netYield)),
          kpiBlock("DSCR 基本", dscr.toFixed(2), dscr >= 1.0 ? C.safe : C.danger),
        ],
        marginBottom: 4,
      },
      {
        columns: [
          kpiBlock(
            "DSCR 複合ストレス",
            dscrStress > 0 ? dscrStress.toFixed(2) : "－",
            dscrStress === 0 ? C.muted : dscrStress >= 1.0 ? C.safe : C.danger
          ),
          kpiBlock("LTV", fmtPct(ltv), ltv <= 0.8 ? C.safe : C.danger),
          kpiBlock(
            "出口Equity",
            fmtYen(result.exitTotalEquity),
            result.exitTotalEquity >= 0 ? C.safe : C.danger
          ),
        ],
        marginBottom: 12,
      },

      // ストレステスト要約（ベースライン vs 複合）
      ...(baselineScenario && stressScenario
        ? [
            subTitle("ストレステスト要約"),
            {
              table: {
                widths: ["*", "auto", "auto", "auto"],
                body: [
                  [
                    thCell("シナリオ", "left"),
                    thCell("DSCR"),
                    thCell("総CF"),
                    thCell("判定", "center"),
                  ],
                  [
                    tdCell("ベースライン", 0, { align: "left" }),
                    tdCell(baselineScenario.dscr.toFixed(2), 0, {
                      color: baselineScenario.dscr >= 1.0 ? C.safe : C.danger,
                    }),
                    tdCell(fmtYen(baselineScenario.totalCashFlow), 0, {
                      color: baselineScenario.totalCashFlow < 0 ? C.danger : C.text,
                    }),
                    {
                      text: baselineScenario.isSafe ? "安全" : "危険",
                      fontSize: 8, bold: true, alignment: "center",
                      color: baselineScenario.isSafe ? C.safe : C.danger,
                      fillColor: C.rowEven,
                    },
                  ],
                  [
                    tdCell("複合ストレス", 1, { align: "left" }),
                    tdCell(stressScenario.dscr.toFixed(2), 1, {
                      color: stressScenario.dscr >= 1.0 ? C.safe : C.danger,
                    }),
                    tdCell(fmtYen(stressScenario.totalCashFlow), 1, {
                      color: stressScenario.totalCashFlow < 0 ? C.danger : C.text,
                    }),
                    {
                      text: stressScenario.isSafe ? "安全" : "危険",
                      fontSize: 8, bold: true, alignment: "center",
                      color: stressScenario.isSafe ? C.safe : C.danger,
                      fillColor: C.white,
                    },
                  ],
                ],
              },
              layout: tableLayout,
              marginBottom: 12,
            },
          ]
        : []),

      // 出口戦略
      subTitle(`出口戦略（${input.holdingYears}年後売却）`),
      hLineTable([
        twoCol("想定売却価格", fmtYen(result.exitSalePrice)),
        twoCol("譲渡所得税", fmtYen(result.exitTransferTax)),
        twoCol("売却手取り", fmtYen(result.exitNetProceeds)),
        twoCol(
          "トータルエクイティ",
          fmtYen(result.exitTotalEquity),
          true,
          result.exitTotalEquity >= 0 ? C.safe : C.danger
        ),
      ]),

      // 自動生成コメント
      {
        text: verdict.autoComment,
        fontSize: 8,
        color: C.muted,
        italics: true,
        marginBottom: 8,
        marginTop: 4,
      },

      // ── Page 3: Cash Flow ──────────────────────────────────────────
      sectionTitle("P2 - 10年間キャッシュフロー", true),
      { svg: buildCfBarChartSvg(result.yearlyResults), width: 480, marginBottom: 12 },
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
              tdCell(fmtYen(r.annualRent), idx),
              tdCell(fmtYen(r.annualLoanPayment), idx),
              tdCell(fmtYen(r.annualExpenses), idx),
              tdCell(fmtYen(r.cashFlow), idx, { color: r.cashFlow < 0 ? C.danger : C.text }),
              tdCell(fmtYen(r.afterTaxCashFlow), idx, { color: r.afterTaxCashFlow < 0 ? C.danger : C.text }),
              tdCell(fmtYen(r.cumulativeCashFlow), idx, { color: r.cumulativeCashFlow < 0 ? C.danger : C.text }),
              tdCell(fmtYen(r.remainingLoanBalance), idx),
            ]),
          ],
        },
        layout: tableLayout,
      },

      // ── Page 4: Stress Test ────────────────────────────────────────
      ...(result.stressScenarios.length > 0
        ? [
            sectionTitle("P3 - ストレステスト結果", true),
            { svg: buildDeadCrossLineSvg(result.yearlyResults, result.deadCrossYear), width: 480, marginBottom: 12 },
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
                    tdCell(sanitize(sc.label), idx, { align: "left" }),
                    tdCell(fmtYen(sc.totalCashFlow), idx, {
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
            {
              svg: buildCostDonutSvg(input.landPrice, input.buildingCost, result.acquisitionCosts),
              width: 200,
              alignment: "center" as const,
              marginBottom: 12,
            },
            subTitle("初期投資内訳"),
            hLineTable([
              twoCol("土地価格", fmtYen(input.landPrice)),
              twoCol("建物費用", fmtYen(input.buildingCost)),
              twoCol("諸経費合計", fmtYen(result.miscExpenses)),
              twoCol("総投資額", fmtYen(result.totalInvestment), true, C.primary),
            ]),
            subTitle("取得時諸経費の明細"),
            hLineTable([
              twoCol("仲介手数料（税込）", fmtYen(result.acquisitionCosts.brokerageFee)),
              twoCol("印紙税", fmtYen(result.acquisitionCosts.stampDuty)),
              twoCol("登録免許税", fmtYen(result.acquisitionCosts.registrationTax)),
              twoCol("不動産取得税（概算）", fmtYen(result.acquisitionCosts.realEstateAcquisitionTax)),
              ...(result.acquisitionCosts.propertyTaxProration > 0
                ? [twoCol("固定資産税日割り精算", fmtYen(result.acquisitionCosts.propertyTaxProration))]
                : []),
              twoCol("諸経費合計", fmtYen(result.acquisitionCosts.total), true, C.primary),
            ]),
            ...(result.yearlyResults.length > 0
              ? [
                  subTitle("年間費用の内訳（1年目）"),
                  hLineTable([
                    ...(result.yearlyResults[0].annualLoanPayment > 0
                      ? [twoCol("ローン返済", fmtYen(result.yearlyResults[0].annualLoanPayment))]
                      : []),
                    twoCol("運営経費", fmtYen(result.yearlyResults[0].annualExpenses)),
                    ...(result.yearlyResults[0].incomeTax > 0
                      ? [twoCol("所得税", fmtYen(result.yearlyResults[0].incomeTax))]
                      : []),
                    twoCol(
                      "税引後CF（1年目）",
                      fmtYen(result.yearlyResults[0].afterTaxCashFlow),
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
