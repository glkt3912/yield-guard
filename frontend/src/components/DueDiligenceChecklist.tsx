"use client";

import React, { useEffect, useState } from "react";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { CheckCircle2, Circle, ClipboardCheck, ShieldCheck } from "lucide-react";
import { getChecklist, saveChecklist, type DueDiligenceState } from "@/lib/dueDiligenceStorage";

// ---------------------------------------------------------------------------
// Item definitions
// ---------------------------------------------------------------------------

export interface DueDiligenceItem {
  id: string;
  label: string;
  /** true = can be auto-linked to risk data; currently informational only */
  autoLinked?: boolean;
}

export interface DueDiligenceCategory {
  id: string;
  label: string;
  items: DueDiligenceItem[];
}

export const DUE_DILIGENCE_ITEMS: DueDiligenceCategory[] = [
  {
    id: "risk",
    label: "リスク確認",
    items: [
      { id: "risk_hazard", label: "ハザードリスクの確認（洪水・津波・土砂）", autoLinked: true },
      { id: "risk_zoning", label: "用途地域の確認", autoLinked: true },
      { id: "risk_population", label: "人口減少エリアの注意確認", autoLinked: true },
    ],
  },
  {
    id: "property",
    label: "物件調査",
    items: [
      { id: "prop_rent_roll", label: "レントロールの確認" },
      { id: "prop_repair_history", label: "修繕履歴の確認" },
      { id: "prop_mgmt_interview", label: "管理会社との面談" },
      { id: "prop_inspection", label: "インスペクションの依頼" },
      { id: "prop_survey_map", label: "測量図の確認" },
    ],
  },
  {
    id: "legal",
    label: "法務・権利",
    items: [
      { id: "legal_registry", label: "登記簿謄本の確認" },
      { id: "legal_building_permit", label: "建築確認済証の確認" },
      { id: "legal_soil_survey", label: "土壌汚染調査の確認" },
      { id: "legal_easement", label: "借地権・地役権の確認" },
    ],
  },
  {
    id: "finance",
    label: "資金・融資",
    items: [
      { id: "fin_bank_hearing", label: "金融機関へのヒアリング" },
      { id: "fin_equity", label: "自己資金の確保確認" },
      { id: "fin_insurance", label: "火災・地震保険の見積もり" },
    ],
  },
];

const ALL_IDS: string[] = DUE_DILIGENCE_ITEMS.flatMap((cat) => cat.items.map((i) => i.id));

// ---------------------------------------------------------------------------
// Component
// ---------------------------------------------------------------------------

interface DueDiligenceChecklistProps {
  /** Unique key per property used as the localStorage identifier */
  propertyKey: string;
}

export default function DueDiligenceChecklist({ propertyKey }: DueDiligenceChecklistProps) {
  const [state, setState] = useState<DueDiligenceState>({});
  const [mounted, setMounted] = useState(false);

  // Hydrate from localStorage on mount
  useEffect(() => {
    setState(getChecklist(propertyKey));
    setMounted(true);
  }, [propertyKey]);

  const toggle = (id: string) => {
    const next = { ...state, [id]: !state[id] };
    setState(next);
    saveChecklist(propertyKey, next);
  };

  const checkedCount = ALL_IDS.filter((id) => state[id]).length;
  const totalCount = ALL_IDS.length;
  const allDone = checkedCount === totalCount;

  // Avoid SSR hydration mismatch for checkbox state
  if (!mounted) return null;

  return (
    <Card>
      <CardHeader className="pb-3">
        <div className="flex items-center justify-between gap-3 flex-wrap">
          <CardTitle className="flex items-center gap-2 text-base">
            <ClipboardCheck className="h-5 w-5 text-muted-foreground" aria-hidden="true" />
            デューデリジェンスチェックリスト
          </CardTitle>

          {/* Completion badge — icon + text so color is never the sole signal (WCAG 1.4.1) */}
          {allDone ? (
            <span
              className="inline-flex items-center gap-1.5 rounded-full bg-emerald-100 px-3 py-1 text-sm font-medium text-emerald-800"
              role="status"
              aria-live="polite"
            >
              <ShieldCheck className="h-4 w-4" aria-hidden="true" />
              デューデリ完了
            </span>
          ) : (
            <span className="text-sm text-muted-foreground" role="status" aria-live="polite">
              {checkedCount} / {totalCount} 完了
            </span>
          )}
        </div>
      </CardHeader>

      <CardContent className="space-y-5">
        {DUE_DILIGENCE_ITEMS.map((category) => (
          <section key={category.id} aria-labelledby={`dd-cat-${category.id}`}>
            <h3
              id={`dd-cat-${category.id}`}
              className="mb-2 text-xs font-semibold uppercase tracking-wide text-muted-foreground"
            >
              {category.label}
            </h3>
            <ul className="space-y-1" role="list">
              {category.items.map((item) => {
                const checked = !!state[item.id];
                return (
                  <li key={item.id}>
                    <button
                      type="button"
                      onClick={() => toggle(item.id)}
                      className={`flex w-full items-center gap-3 rounded-md px-3 py-2 text-left text-sm transition-colors hover:bg-muted/50 focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-ring ${
                        checked ? "text-foreground" : "text-muted-foreground"
                      }`}
                      aria-pressed={checked}
                      aria-label={`${item.label}${checked ? "（確認済み）" : "（未確認）"}`}
                    >
                      {checked ? (
                        <CheckCircle2
                          className="h-4 w-4 shrink-0 text-emerald-600"
                          aria-hidden="true"
                        />
                      ) : (
                        <Circle
                          className="h-4 w-4 shrink-0 text-muted-foreground/40"
                          aria-hidden="true"
                        />
                      )}
                      <span className={checked ? "line-through opacity-60" : ""}>{item.label}</span>
                      {item.autoLinked && (
                        <span className="ml-auto shrink-0 rounded bg-blue-50 px-1.5 py-0.5 text-[10px] font-medium text-blue-600">
                          自動連携
                        </span>
                      )}
                    </button>
                  </li>
                );
              })}
            </ul>
          </section>
        ))}
      </CardContent>
    </Card>
  );
}
