"use client";
import React from "react";
import { Card, CardHeader, CardTitle, CardContent } from "@/components/ui/card";
import { Badge } from "@/components/ui/badge";
import type { InvestmentResult, LoanMethod } from "@/types/investment";
import { formatMan, formatPct } from "@/lib/utils";
import { AlertTriangle, CheckCircle } from "lucide-react";

interface Props {
  result: InvestmentResult;
  loanMethod: LoanMethod;
  onLoanMethodChange: (method: LoanMethod) => void;
}

export function LoanOptimizationPanel({ result, loanMethod, onLoanMethodChange }: Props) {
  const dscr = result.dscr ?? 0;
  const dscrSafe = dscr >= 1.0;

  return (
    <Card>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between flex-wrap gap-2">
          <CardTitle className="text-base">ローン最適化（DSCR・LTV感度分析）</CardTitle>
          <select
            value={loanMethod}
            onChange={(e) => onLoanMethodChange(e.target.value as LoanMethod)}
            className="text-sm border rounded-md px-2 py-1 bg-background focus:outline-none focus:ring-1 focus:ring-primary"
          >
            <option value="equal-payment">元利均等返済</option>
            <option value="equal-principal">元金均等返済</option>
          </select>
        </div>
      </CardHeader>
      <CardContent className="space-y-5">
        {/* DSCR */}
        <div className="flex items-center gap-3">
          <div>
            <p className="text-xs text-muted-foreground mb-0.5">DSCR（借入金償還余裕率）</p>
            <p className="text-2xl font-bold tabular-nums">{dscr.toFixed(2)}</p>
            <p className="text-xs text-muted-foreground mt-0.5">NOI ÷ 年間返済額（1年目）</p>
          </div>
          {dscrSafe ? (
            <Badge className="flex items-center gap-1 bg-green-100 text-green-700 border-green-200 ml-auto">
              <CheckCircle className="h-3.5 w-3.5" />
              安全（≥ 1.0）
            </Badge>
          ) : (
            <Badge className="flex items-center gap-1 bg-red-100 text-red-700 border-red-200 ml-auto">
              <AlertTriangle className="h-3.5 w-3.5" />
              危険（&lt; 1.0）
            </Badge>
          )}
        </div>

        {/* LTV 感度テーブル */}
        {result.ltvSensitivity && result.ltvSensitivity.length > 0 && (
          <div>
            <div className="flex items-center gap-2 mb-2">
              <p className="text-xs font-medium text-muted-foreground">LTV 感度分析</p>
              <p className="text-xs text-muted-foreground">
                （ベースケース基準
                {loanMethod === "equal-principal" ? "・元金均等は1年目返済額で試算" : ""}）
              </p>
            </div>
            <div className="overflow-x-auto">
              <table className="w-full text-xs border-collapse">
                <thead>
                  <tr className="border-b text-muted-foreground">
                    <th className="text-left py-1.5 pr-3 font-medium">LTV</th>
                    <th className="text-right py-1.5 pr-3 font-medium">自己資金</th>
                    <th className="text-right py-1.5 pr-3 font-medium">借入額</th>
                    <th className="text-right py-1.5 pr-3 font-medium">DSCR</th>
                    <th className="text-right py-1.5 pr-3 font-medium">年間CF</th>
                    <th className="text-right py-1.5 font-medium">CF利回り</th>
                  </tr>
                </thead>
                <tbody>
                  {result.ltvSensitivity.map((row) => {
                    const safe = row.dscr >= 1.0;
                    return (
                      <tr key={row.ltv} className="border-b last:border-0 hover:bg-muted/40 transition-colors">
                        <td className="py-1.5 pr-3 font-medium">{formatPct(row.ltv)}</td>
                        <td className="text-right py-1.5 pr-3 tabular-nums">{formatMan(row.equity)}</td>
                        <td className="text-right py-1.5 pr-3 tabular-nums">{formatMan(row.loanAmount)}</td>
                        <td className="text-right py-1.5 pr-3 tabular-nums">
                          <span className={safe ? "text-green-600 font-medium" : "text-red-600 font-medium"}>
                            {row.dscr.toFixed(2)}
                          </span>
                        </td>
                        <td className="text-right py-1.5 pr-3 tabular-nums">
                          <span className={row.annualCF >= 0 ? "" : "text-red-600"}>
                            {formatMan(row.annualCF)}
                          </span>
                        </td>
                        <td className="text-right py-1.5 tabular-nums">
                          <span className={row.cfYield >= 0 ? "" : "text-red-600"}>
                            {formatPct(row.cfYield)}
                          </span>
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          </div>
        )}
      </CardContent>
    </Card>
  );
}
