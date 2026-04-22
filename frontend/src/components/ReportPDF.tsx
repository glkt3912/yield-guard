"use client";

import React from "react";
import {
  Document,
  Page,
  Text,
  View,
  StyleSheet,
  Font,
} from "@react-pdf/renderer";
import type {
  InvestmentInput,
  InvestmentResult,
  YearlyResult,
  StressScenarioResult,
  AcquisitionCostBreakdown,
} from "@/types/investment";

// Register Noto Sans JP for Japanese text rendering
Font.register({
  family: "NotoSansJP",
  src: "https://fonts.gstatic.com/s/notosansjp/v53/nKKF-GM_FYFRJvXzVXaAPe97P1K9CtBL.otf",
});

const COLORS = {
  primary: "#1a56db",
  headerBg: "#1e3a5f",
  rowEven: "#f8fafc",
  rowOdd: "#ffffff",
  border: "#e2e8f0",
  danger: "#dc2626",
  safe: "#16a34a",
  text: "#1e293b",
  muted: "#64748b",
  white: "#ffffff",
};

const styles = StyleSheet.create({
  page: {
    fontFamily: "NotoSansJP",
    fontSize: 9,
    color: COLORS.text,
    paddingHorizontal: 36,
    paddingTop: 36,
    paddingBottom: 48,
  },
  // Cover page
  coverPage: {
    fontFamily: "NotoSansJP",
    color: COLORS.text,
    paddingHorizontal: 48,
    paddingTop: 80,
    paddingBottom: 48,
  },
  coverTitle: {
    fontSize: 28,
    fontWeight: "bold",
    color: COLORS.headerBg,
    marginBottom: 8,
  },
  coverSubtitle: {
    fontSize: 14,
    color: COLORS.muted,
    marginBottom: 48,
  },
  coverCard: {
    borderRadius: 6,
    border: `1 solid ${COLORS.border}`,
    padding: 20,
    marginBottom: 16,
  },
  coverCardTitle: {
    fontSize: 11,
    fontWeight: "bold",
    color: COLORS.muted,
    marginBottom: 12,
    textTransform: "uppercase",
  },
  coverRow: {
    flexDirection: "row",
    justifyContent: "space-between",
    marginBottom: 8,
  },
  coverLabel: {
    fontSize: 10,
    color: COLORS.muted,
    width: "40%",
  },
  coverValue: {
    fontSize: 10,
    fontWeight: "bold",
    color: COLORS.text,
    width: "60%",
    textAlign: "right",
  },
  // Section heading
  sectionTitle: {
    fontSize: 14,
    fontWeight: "bold",
    color: COLORS.headerBg,
    marginBottom: 12,
    paddingBottom: 6,
    borderBottom: `2 solid ${COLORS.primary}`,
  },
  // KPI grid
  kpiGrid: {
    flexDirection: "row",
    gap: 8,
    marginBottom: 16,
    flexWrap: "wrap",
  },
  kpiCard: {
    flex: 1,
    minWidth: 100,
    border: `1 solid ${COLORS.border}`,
    borderRadius: 6,
    padding: 10,
  },
  kpiLabel: {
    fontSize: 8,
    color: COLORS.muted,
    marginBottom: 4,
  },
  kpiValue: {
    fontSize: 18,
    fontWeight: "bold",
    color: COLORS.primary,
  },
  kpiUnit: {
    fontSize: 9,
    color: COLORS.muted,
  },
  // Table
  table: {
    marginBottom: 16,
  },
  tableHeader: {
    flexDirection: "row",
    backgroundColor: COLORS.headerBg,
    paddingVertical: 5,
    paddingHorizontal: 4,
  },
  tableHeaderCell: {
    fontSize: 8,
    fontWeight: "bold",
    color: COLORS.white,
    textAlign: "right",
    flex: 1,
  },
  tableHeaderCellFirst: {
    fontSize: 8,
    fontWeight: "bold",
    color: COLORS.white,
    textAlign: "left",
    width: 28,
  },
  tableRow: {
    flexDirection: "row",
    paddingVertical: 4,
    paddingHorizontal: 4,
    borderBottom: `0.5 solid ${COLORS.border}`,
  },
  tableCellFirst: {
    fontSize: 8,
    textAlign: "left",
    width: 28,
    color: COLORS.muted,
  },
  tableCell: {
    fontSize: 8,
    textAlign: "right",
    flex: 1,
    color: COLORS.text,
  },
  tableCellDanger: {
    fontSize: 8,
    textAlign: "right",
    flex: 1,
    color: COLORS.danger,
  },
  // Stress scenarios
  scenarioTable: {
    marginBottom: 16,
  },
  scenarioHeader: {
    flexDirection: "row",
    backgroundColor: COLORS.headerBg,
    paddingVertical: 5,
    paddingHorizontal: 4,
  },
  scenarioHeaderLabel: {
    fontSize: 8,
    fontWeight: "bold",
    color: COLORS.white,
    flex: 3,
  },
  scenarioHeaderCell: {
    fontSize: 8,
    fontWeight: "bold",
    color: COLORS.white,
    textAlign: "right",
    flex: 2,
  },
  scenarioHeaderStatus: {
    fontSize: 8,
    fontWeight: "bold",
    color: COLORS.white,
    textAlign: "center",
    width: 40,
  },
  scenarioRow: {
    flexDirection: "row",
    paddingVertical: 5,
    paddingHorizontal: 4,
    borderBottom: `0.5 solid ${COLORS.border}`,
    alignItems: "center",
  },
  scenarioLabel: {
    fontSize: 8,
    color: COLORS.text,
    flex: 3,
  },
  scenarioCell: {
    fontSize: 8,
    textAlign: "right",
    flex: 2,
    color: COLORS.text,
  },
  scenarioStatusSafe: {
    fontSize: 8,
    textAlign: "center",
    color: COLORS.safe,
    fontWeight: "bold",
    width: 40,
  },
  scenarioStatusDanger: {
    fontSize: 8,
    textAlign: "center",
    color: COLORS.danger,
    fontWeight: "bold",
    width: 40,
  },
  // Cost breakdown
  costRow: {
    flexDirection: "row",
    justifyContent: "space-between",
    paddingVertical: 5,
    borderBottom: `0.5 solid ${COLORS.border}`,
    paddingHorizontal: 4,
  },
  costLabel: {
    fontSize: 9,
    color: COLORS.text,
  },
  costValue: {
    fontSize: 9,
    color: COLORS.text,
    textAlign: "right",
  },
  costTotal: {
    flexDirection: "row",
    justifyContent: "space-between",
    paddingVertical: 6,
    paddingHorizontal: 4,
    backgroundColor: COLORS.rowEven,
  },
  costTotalLabel: {
    fontSize: 9,
    fontWeight: "bold",
    color: COLORS.text,
  },
  costTotalValue: {
    fontSize: 9,
    fontWeight: "bold",
    color: COLORS.primary,
    textAlign: "right",
  },
  // Footer
  footer: {
    position: "absolute",
    bottom: 24,
    left: 36,
    right: 36,
    fontSize: 7,
    color: COLORS.muted,
    borderTop: `0.5 solid ${COLORS.border}`,
    paddingTop: 6,
    textAlign: "center",
  },
});

// --- helpers ---
function formatJPY(value: number): string {
  if (Math.abs(value) >= 1_000_000) {
    return `${(value / 1_000_000).toFixed(1)}百万円`;
  }
  return `${value.toLocaleString("ja-JP")}円`;
}

function formatPct(value: number): string {
  return `${(value * 100).toFixed(2)}%`;
}

function todayStr(): string {
  const d = new Date();
  return `${d.getFullYear()}年${d.getMonth() + 1}月${d.getDate()}日`;
}

// --- Sub-components ---
const FooterNote = () => (
  <Text style={styles.footer}>
    本資料は yield-guard による試算です。実際の投資判断は専門家にご相談ください。
  </Text>
);

interface CoverPageProps {
  input: InvestmentInput;
  result: InvestmentResult;
  analysisDate: string;
}

const CoverPage = ({ input, result, analysisDate }: CoverPageProps) => (
  <Page size="A4" style={styles.coverPage}>
    <View>
      <Text style={styles.coverTitle}>不動産投資分析レポート</Text>
      <Text style={styles.coverSubtitle}>Real Estate Investment Analysis Report</Text>
    </View>

    <View style={styles.coverCard}>
      <Text style={styles.coverCardTitle}>物件概要</Text>
      <View style={styles.coverRow}>
        <Text style={styles.coverLabel}>物件価格（土地）</Text>
        <Text style={styles.coverValue}>{formatJPY(input.landPrice)}</Text>
      </View>
      <View style={styles.coverRow}>
        <Text style={styles.coverLabel}>建物費用</Text>
        <Text style={styles.coverValue}>{formatJPY(input.buildingCost)}</Text>
      </View>
      <View style={styles.coverRow}>
        <Text style={styles.coverLabel}>築年数</Text>
        <Text style={styles.coverValue}>
          {input.buildingAge === 0 ? "新築" : `${input.buildingAge}年`}
        </Text>
      </View>
      <View style={styles.coverRow}>
        <Text style={styles.coverLabel}>構造</Text>
        <Text style={styles.coverValue}>{input.buildingType}</Text>
      </View>
      {input.stationMinutes > 0 && (
        <View style={styles.coverRow}>
          <Text style={styles.coverLabel}>最寄り駅徒歩</Text>
          <Text style={styles.coverValue}>{input.stationMinutes}分</Text>
        </View>
      )}
      <View style={styles.coverRow}>
        <Text style={styles.coverLabel}>月額賃料</Text>
        <Text style={styles.coverValue}>{formatJPY(input.monthlyRent)}</Text>
      </View>
      <View style={styles.coverRow}>
        <Text style={styles.coverLabel}>想定空室率</Text>
        <Text style={styles.coverValue}>{formatPct(input.vacancyRate)}</Text>
      </View>
      <View style={styles.coverRow}>
        <Text style={styles.coverLabel}>ローン金額</Text>
        <Text style={styles.coverValue}>{formatJPY(input.loanAmount)}</Text>
      </View>
      <View style={styles.coverRow}>
        <Text style={styles.coverLabel}>ローン金利</Text>
        <Text style={styles.coverValue}>{formatPct(input.annualLoanRate)}</Text>
      </View>
      <View style={styles.coverRow}>
        <Text style={styles.coverLabel}>ローン期間</Text>
        <Text style={styles.coverValue}>{input.loanYears}年</Text>
      </View>
    </View>

    <View style={styles.coverCard}>
      <Text style={styles.coverCardTitle}>分析情報</Text>
      <View style={styles.coverRow}>
        <Text style={styles.coverLabel}>分析実施日</Text>
        <Text style={styles.coverValue}>{analysisDate}</Text>
      </View>
      <View style={styles.coverRow}>
        <Text style={styles.coverLabel}>総投資額</Text>
        <Text style={styles.coverValue}>{formatJPY(result.totalInvestment)}</Text>
      </View>
      <View style={styles.coverRow}>
        <Text style={styles.coverLabel}>表面利回り</Text>
        <Text style={styles.coverValue}>{formatPct(result.grossYield)}</Text>
      </View>
    </View>

    <FooterNote />
  </Page>
);

interface SummaryPageProps {
  result: InvestmentResult;
  input: InvestmentInput;
}

const SummaryPage = ({ result, input }: SummaryPageProps) => {
  // Calculate DSCR from year 1
  const year1 = result.yearlyResults[0];
  const dscr =
    year1 && year1.annualLoanPayment > 0
      ? year1.annualRent / year1.annualLoanPayment
      : 0;
  const ltv =
    result.totalInvestment > 0 ? input.loanAmount / result.totalInvestment : 0;

  return (
    <Page size="A4" style={styles.page}>
      <Text style={styles.sectionTitle}>P1 - 投資サマリー</Text>

      <View style={styles.kpiGrid}>
        <View style={styles.kpiCard}>
          <Text style={styles.kpiLabel}>表面利回り</Text>
          <Text style={styles.kpiValue}>{(result.grossYield * 100).toFixed(2)}</Text>
          <Text style={styles.kpiUnit}>%</Text>
        </View>
        <View style={styles.kpiCard}>
          <Text style={styles.kpiLabel}>実効利回り（税引後）</Text>
          <Text style={styles.kpiValue}>{(result.netYield * 100).toFixed(2)}</Text>
          <Text style={styles.kpiUnit}>%</Text>
        </View>
        <View style={styles.kpiCard}>
          <Text style={styles.kpiLabel}>DSCR（1年目）</Text>
          <Text style={[styles.kpiValue, { color: dscr >= 1.0 ? COLORS.safe : COLORS.danger }]}>
            {dscr.toFixed(2)}
          </Text>
          <Text style={styles.kpiUnit}>倍</Text>
        </View>
        <View style={styles.kpiCard}>
          <Text style={styles.kpiLabel}>LTV（ローン比率）</Text>
          <Text style={[styles.kpiValue, { color: ltv <= 0.8 ? COLORS.safe : COLORS.danger }]}>
            {(ltv * 100).toFixed(1)}
          </Text>
          <Text style={styles.kpiUnit}>%</Text>
        </View>
      </View>

      <View style={[styles.kpiGrid, { marginBottom: 8 }]}>
        <View style={styles.kpiCard}>
          <Text style={styles.kpiLabel}>総投資額</Text>
          <Text style={styles.kpiValue}>{(result.totalInvestment / 1_000_000).toFixed(1)}</Text>
          <Text style={styles.kpiUnit}>百万円</Text>
        </View>
        <View style={styles.kpiCard}>
          <Text style={styles.kpiLabel}>月額賃料</Text>
          <Text style={styles.kpiValue}>{(input.monthlyRent / 10_000).toFixed(0)}</Text>
          <Text style={styles.kpiUnit}>万円/月</Text>
        </View>
        <View style={styles.kpiCard}>
          <Text style={styles.kpiLabel}>デッドクロス年</Text>
          <Text style={[styles.kpiValue, { color: result.deadCrossYear > 0 ? COLORS.danger : COLORS.safe }]}>
            {result.deadCrossYear > 0 ? `${result.deadCrossYear}年目` : "なし"}
          </Text>
        </View>
        <View style={styles.kpiCard}>
          <Text style={styles.kpiLabel}>{(result.yieldTarget * 100).toFixed(0)}%基準</Text>
          <Text style={[styles.kpiValue, { color: result.isAboveYieldTarget ? COLORS.safe : COLORS.danger }]}>
            {result.isAboveYieldTarget ? "達成" : "未達"}
          </Text>
        </View>
      </View>

      {/* Exit strategy */}
      <Text style={[styles.sectionTitle, { fontSize: 11, marginTop: 8 }]}>出口戦略（{input.holdingYears}年後売却）</Text>
      <View style={{ marginBottom: 8 }}>
        <View style={styles.costRow}>
          <Text style={styles.costLabel}>想定売却価格</Text>
          <Text style={styles.costValue}>{formatJPY(result.exitSalePrice)}</Text>
        </View>
        <View style={styles.costRow}>
          <Text style={styles.costLabel}>譲渡所得税</Text>
          <Text style={styles.costValue}>{formatJPY(result.exitTransferTax)}</Text>
        </View>
        <View style={styles.costRow}>
          <Text style={styles.costLabel}>売却手取り</Text>
          <Text style={styles.costValue}>{formatJPY(result.exitNetProceeds)}</Text>
        </View>
        <View style={styles.costTotal}>
          <Text style={styles.costTotalLabel}>トータルエクイティ（CF累積＋売却益）</Text>
          <Text style={[styles.costTotalValue, { color: result.exitTotalEquity >= 0 ? COLORS.safe : COLORS.danger }]}>
            {formatJPY(result.exitTotalEquity)}
          </Text>
        </View>
      </View>

      <FooterNote />
    </Page>
  );
};

interface CashFlowPageProps {
  yearlyResults: YearlyResult[];
  holdingYears: number;
}

const CashFlowPage = ({ yearlyResults, holdingYears }: CashFlowPageProps) => {
  const rows = yearlyResults.slice(0, Math.min(holdingYears, 10));
  return (
    <Page size="A4" style={styles.page}>
      <Text style={styles.sectionTitle}>P2 - 10年間キャッシュフロー</Text>

      <View style={styles.table}>
        <View style={styles.tableHeader}>
          <Text style={styles.tableHeaderCellFirst}>年</Text>
          <Text style={styles.tableHeaderCell}>賃料収入</Text>
          <Text style={styles.tableHeaderCell}>ローン返済</Text>
          <Text style={styles.tableHeaderCell}>運営経費</Text>
          <Text style={styles.tableHeaderCell}>税引前CF</Text>
          <Text style={styles.tableHeaderCell}>税引後CF</Text>
          <Text style={styles.tableHeaderCell}>累積CF</Text>
          <Text style={styles.tableHeaderCell}>残債</Text>
        </View>
        {rows.map((r, idx) => {
          const isEven = idx % 2 === 0;
          return (
            <View
              key={r.year}
              style={[styles.tableRow, { backgroundColor: isEven ? COLORS.rowEven : COLORS.rowOdd }]}
            >
              <Text style={styles.tableCellFirst}>{r.year}年</Text>
              <Text style={styles.tableCell}>{formatJPY(r.annualRent)}</Text>
              <Text style={styles.tableCell}>{formatJPY(r.annualLoanPayment)}</Text>
              <Text style={styles.tableCell}>{formatJPY(r.annualExpenses)}</Text>
              <Text style={r.cashFlow < 0 ? styles.tableCellDanger : styles.tableCell}>
                {formatJPY(r.cashFlow)}
              </Text>
              <Text style={r.afterTaxCashFlow < 0 ? styles.tableCellDanger : styles.tableCell}>
                {formatJPY(r.afterTaxCashFlow)}
              </Text>
              <Text style={r.cumulativeCashFlow < 0 ? styles.tableCellDanger : styles.tableCell}>
                {formatJPY(r.cumulativeCashFlow)}
              </Text>
              <Text style={styles.tableCell}>{formatJPY(r.remainingLoanBalance)}</Text>
            </View>
          );
        })}
      </View>

      <FooterNote />
    </Page>
  );
};

interface StressPageProps {
  stressScenarios: StressScenarioResult[];
}

const StressPage = ({ stressScenarios }: StressPageProps) => (
  <Page size="A4" style={styles.page}>
    <Text style={styles.sectionTitle}>P3 - ストレステスト結果</Text>

    <View style={styles.scenarioTable}>
      <View style={styles.scenarioHeader}>
        <Text style={styles.scenarioHeaderLabel}>シナリオ</Text>
        <Text style={styles.scenarioHeaderCell}>総CF</Text>
        <Text style={styles.scenarioHeaderCell}>DSCR</Text>
        <Text style={styles.scenarioHeaderCell}>回収年</Text>
        <Text style={styles.scenarioHeaderStatus}>判定</Text>
      </View>
      {stressScenarios.map((s, idx) => {
        const isEven = idx % 2 === 0;
        return (
          <View
            key={s.label}
            style={[styles.scenarioRow, { backgroundColor: isEven ? COLORS.rowEven : COLORS.rowOdd }]}
          >
            <Text style={styles.scenarioLabel}>{s.label}</Text>
            <Text style={[styles.scenarioCell, { color: s.totalCashFlow < 0 ? COLORS.danger : COLORS.text }]}>
              {formatJPY(s.totalCashFlow)}
            </Text>
            <Text style={[styles.scenarioCell, { color: s.dscr < 1.0 ? COLORS.danger : COLORS.text }]}>
              {s.dscr.toFixed(2)}
            </Text>
            <Text style={styles.scenarioCell}>
              {s.breakEvenYear > 0 ? `${s.breakEvenYear}年` : "－"}
            </Text>
            <Text style={s.isSafe ? styles.scenarioStatusSafe : styles.scenarioStatusDanger}>
              {s.isSafe ? "安全" : "危険"}
            </Text>
          </View>
        );
      })}
    </View>

    <FooterNote />
  </Page>
);

interface CostPageProps {
  acquisitionCosts: AcquisitionCostBreakdown;
  input: InvestmentInput;
  result: InvestmentResult;
}

const CostPage = ({ acquisitionCosts, input, result }: CostPageProps) => (
  <Page size="A4" style={styles.page}>
    <Text style={styles.sectionTitle}>P4 - 取得コスト内訳</Text>

    {/* Initial investment */}
    <Text style={[styles.sectionTitle, { fontSize: 11, marginTop: 0 }]}>初期投資内訳</Text>
    <View style={{ marginBottom: 16 }}>
      <View style={styles.costRow}>
        <Text style={styles.costLabel}>土地価格</Text>
        <Text style={styles.costValue}>{formatJPY(input.landPrice)}</Text>
      </View>
      <View style={styles.costRow}>
        <Text style={styles.costLabel}>建物費用</Text>
        <Text style={styles.costValue}>{formatJPY(input.buildingCost)}</Text>
      </View>
      <View style={styles.costRow}>
        <Text style={styles.costLabel}>諸経費合計</Text>
        <Text style={styles.costValue}>{formatJPY(result.miscExpenses)}</Text>
      </View>
      <View style={styles.costTotal}>
        <Text style={styles.costTotalLabel}>総投資額</Text>
        <Text style={styles.costTotalValue}>{formatJPY(result.totalInvestment)}</Text>
      </View>
    </View>

    {/* Acquisition costs breakdown */}
    <Text style={[styles.sectionTitle, { fontSize: 11 }]}>取得時諸経費の明細</Text>
    <View style={{ marginBottom: 16 }}>
      <View style={styles.costRow}>
        <Text style={styles.costLabel}>仲介手数料（税込）</Text>
        <Text style={styles.costValue}>{formatJPY(acquisitionCosts.brokerageFee)}</Text>
      </View>
      <View style={styles.costRow}>
        <Text style={styles.costLabel}>印紙税</Text>
        <Text style={styles.costValue}>{formatJPY(acquisitionCosts.stampDuty)}</Text>
      </View>
      <View style={styles.costRow}>
        <Text style={styles.costLabel}>登録免許税</Text>
        <Text style={styles.costValue}>{formatJPY(acquisitionCosts.registrationTax)}</Text>
      </View>
      <View style={styles.costRow}>
        <Text style={styles.costLabel}>不動産取得税（概算）</Text>
        <Text style={styles.costValue}>{formatJPY(acquisitionCosts.realEstateAcquisitionTax)}</Text>
      </View>
      {acquisitionCosts.propertyTaxProration > 0 && (
        <View style={styles.costRow}>
          <Text style={styles.costLabel}>固定資産税日割り精算</Text>
          <Text style={styles.costValue}>{formatJPY(acquisitionCosts.propertyTaxProration)}</Text>
        </View>
      )}
      <View style={styles.costTotal}>
        <Text style={styles.costTotalLabel}>諸経費合計</Text>
        <Text style={styles.costTotalValue}>{formatJPY(acquisitionCosts.total)}</Text>
      </View>
    </View>

    {/* Running costs year 1 */}
    {result.yearlyResults.length > 0 && (
      <>
        <Text style={[styles.sectionTitle, { fontSize: 11 }]}>年間費用の内訳（1年目）</Text>
        <View>
          {result.yearlyResults[0].annualLoanPayment > 0 && (
            <View style={styles.costRow}>
              <Text style={styles.costLabel}>ローン返済</Text>
              <Text style={styles.costValue}>{formatJPY(result.yearlyResults[0].annualLoanPayment)}</Text>
            </View>
          )}
          <View style={styles.costRow}>
            <Text style={styles.costLabel}>運営経費</Text>
            <Text style={styles.costValue}>{formatJPY(result.yearlyResults[0].annualExpenses)}</Text>
          </View>
          {result.yearlyResults[0].incomeTax > 0 && (
            <View style={styles.costRow}>
              <Text style={styles.costLabel}>所得税</Text>
              <Text style={styles.costValue}>{formatJPY(result.yearlyResults[0].incomeTax)}</Text>
            </View>
          )}
          <View style={styles.costTotal}>
            <Text style={styles.costTotalLabel}>税引後キャッシュフロー（1年目）</Text>
            <Text style={[styles.costTotalValue, {
              color: result.yearlyResults[0].afterTaxCashFlow >= 0 ? COLORS.safe : COLORS.danger
            }]}>
              {formatJPY(result.yearlyResults[0].afterTaxCashFlow)}
            </Text>
          </View>
        </View>
      </>
    )}

    <FooterNote />
  </Page>
);

// --- Main document ---
export interface ReportPDFProps {
  input: InvestmentInput;
  result: InvestmentResult;
}

export const ReportPDF = ({ input, result }: ReportPDFProps) => {
  const analysisDate = todayStr();

  return (
    <Document
      title="不動産投資分析レポート - yield-guard"
      author="yield-guard"
      subject="不動産投資シミュレーション結果"
      creator="yield-guard"
    >
      <CoverPage input={input} result={result} analysisDate={analysisDate} />
      <SummaryPage result={result} input={input} />
      <CashFlowPage yearlyResults={result.yearlyResults} holdingYears={input.holdingYears} />
      {result.stressScenarios.length > 0 && (
        <StressPage stressScenarios={result.stressScenarios} />
      )}
      {result.acquisitionCosts && (
        <CostPage acquisitionCosts={result.acquisitionCosts} input={input} result={result} />
      )}
    </Document>
  );
};
