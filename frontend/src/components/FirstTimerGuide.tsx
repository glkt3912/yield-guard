"use client";
import React from "react";
import { X, TrendingUp, AlertTriangle, MapPin, DoorOpen } from "lucide-react";
import { Button } from "@/components/ui/button";
import type { InvestmentInput } from "@/types/investment";

interface Props {
  onUseSample: (input: InvestmentInput, sampleName: string) => void;
  onDismiss: () => void;
  sampleProperty: InvestmentInput;
}

const STEPS = [
  {
    icon: <TrendingUp className="h-5 w-5 text-blue-600" />,
    title: "利回りチェック",
    body: "表面利回り8%を目安に。これを下回る場合は価格交渉の余地を探りましょう。",
  },
  {
    icon: <AlertTriangle className="h-5 w-5 text-orange-500" />,
    title: "デッドクロス確認",
    body: "元金返済が減価償却費を上回る年（デッドクロス）が10年以内なら要注意。税負担が突然重くなります。",
  },
  {
    icon: <MapPin className="h-5 w-5 text-green-600" />,
    title: "立地スコア",
    body: "人口動態・駅需要・ハザードリスクを統合した60点以上のエリアが目安です。",
  },
  {
    icon: <DoorOpen className="h-5 w-5 text-purple-600" />,
    title: "出口戦略",
    body: "5年超保有で譲渡税が約20%に下がります。売却時の手残りも必ず試算しましょう。",
  },
];

export function FirstTimerGuide({ onUseSample, onDismiss, sampleProperty }: Props) {
  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/50 p-4">
      <div className="relative w-full max-w-lg rounded-2xl bg-card shadow-2xl">
        <button
          onClick={onDismiss}
          className="absolute right-4 top-4 rounded-full p-1 text-muted-foreground hover:bg-muted"
          aria-label="閉じる"
        >
          <X className="h-4 w-4" />
        </button>

        <div className="p-6">
          <h2 className="text-lg font-bold">🏠 Yield-Guard の使い方</h2>
          <p className="mt-1 text-sm text-muted-foreground">不動産投資を4つの視点で判断しよう</p>

          <ol className="mt-4 space-y-3">
            {STEPS.map((step, i) => (
              <li key={i} className="flex gap-3 rounded-lg border bg-muted/30 p-3">
                <span className="mt-0.5 shrink-0">{step.icon}</span>
                <div>
                  <p className="text-sm font-semibold">
                    {i + 1}. {step.title}
                  </p>
                  <p className="mt-0.5 text-xs text-muted-foreground">{step.body}</p>
                </div>
              </li>
            ))}
          </ol>

          <div className="mt-5 flex flex-col gap-2 sm:flex-row">
            <Button
              className="flex-1"
              onClick={() => onUseSample(sampleProperty, "木造アパート（築15年・月25万円）")}
            >
              サンプル物件で試す
            </Button>
            <Button variant="outline" className="flex-1" onClick={onDismiss}>
              自分で入力する
            </Button>
          </div>
          <p className="mt-2 text-center text-xs text-muted-foreground">
            サンプル: 木造アパート 3,500万円・月額家賃25万円・借入2,400万円
          </p>
        </div>
      </div>
    </div>
  );
}
