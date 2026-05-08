"use client";
import React from "react";
import { ShieldAlert, AlertTriangle } from "lucide-react";
import type { UrbanRisk } from "@/types/investment";

const HAZARD_LABEL_MAP: Record<string, string> = {
  flood: "洪水",
  tsunami: "津波",
  landslide: "土砂",
  liquefaction: "液状化",
};

function getHazardTypeLabel(code: string): string {
  const lower = code.toLowerCase();
  for (const [key, label] of Object.entries(HAZARD_LABEL_MAP)) {
    if (lower.includes(key)) return label;
  }
  return "";
}

interface HazardAlertBannerProps {
  hazardRisks: UrbanRisk[] | null | undefined;
  externalUrbanRisks?: UrbanRisk[] | null;
}

export function HazardAlertBanner({ hazardRisks, externalUrbanRisks }: HazardAlertBannerProps) {
  const allRisks: UrbanRisk[] = [
    ...(hazardRisks ?? []),
    ...(externalUrbanRisks ?? []).filter(
      (r) => !(hazardRisks ?? []).some((h) => h.code === r.code)
    ),
  ];

  const errorRisks = allRisks.filter((r) => r.level === "ERROR");
  const warningRisks = allRisks.filter((r) => r.level === "WARNING");

  if (errorRisks.length === 0 && warningRisks.length === 0) return null;

  return (
    <div role="alert" aria-label="ハザード警告" className="space-y-2">
      {errorRisks.map((risk) => {
        const hazardType = getHazardTypeLabel(risk.code);
        return (
          <div
            key={risk.code}
            role="listitem"
            className="flex items-start gap-3 rounded-md border-2 border-red-500 bg-red-50 p-4 text-red-900"
          >
            <ShieldAlert className="mt-0.5 h-5 w-5 shrink-0 text-red-600" />
            <div>
              <p className="text-sm font-bold">
                ⚠ 重大ハザード{hazardType ? `（${hazardType}）` : ""}: {risk.title}
              </p>
              <p className="mt-0.5 text-sm">{risk.description}</p>
              <p className="mt-1 text-xs text-red-700">
                投資スコアに関わらず、このリスクは必ず確認してください。
              </p>
            </div>
          </div>
        );
      })}
      {warningRisks.map((risk) => {
        const hazardType = getHazardTypeLabel(risk.code);
        return (
          <div
            key={risk.code}
            role="listitem"
            className="flex items-start gap-3 rounded-md border-2 border-yellow-400 bg-yellow-50 p-4 text-yellow-900"
          >
            <AlertTriangle className="mt-0.5 h-5 w-5 shrink-0 text-yellow-600" />
            <div>
              <p className="text-sm font-bold">
                ⚠ ハザード注意{hazardType ? `（${hazardType}）` : ""}: {risk.title}
              </p>
              <p className="mt-0.5 text-sm">{risk.description}</p>
            </div>
          </div>
        );
      })}
    </div>
  );
}
