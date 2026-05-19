import { Suspense } from "react";
import type { Metadata } from "next";
import { Dashboard } from "@/components/Dashboard";
import { DashboardWithParams } from "./_dashboard-client";

export async function generateMetadata({
  searchParams,
}: {
  searchParams: Promise<Record<string, string>>;
}): Promise<Metadata> {
  const params = await searchParams;
  const price = params.price
    ? `物件価格 ${Number(params.price).toLocaleString()}万円`
    : null;
  const rent = params.rent
    ? `月額賃料 ${Number(params.rent).toLocaleString()}円`
    : null;

  if (!price) return {};

  const description = [price, rent].filter(Boolean).join(" / ");
  return {
    title: `${description} — Yield-Guard シミュレーション結果`,
    description: `${description} のシミュレーション結果を確認する`,
    openGraph: {
      title: `${description} — Yield-Guard`,
      description: `利回り・デッドクロス・出口戦略を分析した結果です`,
    },
  };
}

export default function Home() {
  return (
    <Suspense fallback={<Dashboard />}>
      <DashboardWithParams />
    </Suspense>
  );
}
