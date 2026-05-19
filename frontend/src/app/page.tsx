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
  const priceNum = Number(params.price);
  const rentNum = Number(params.rent);
  const price =
    params.price && !isNaN(priceNum) && priceNum > 0
      ? `物件価格 ${priceNum.toLocaleString()}万円`
      : null;
  const rent =
    params.rent && !isNaN(rentNum) && rentNum > 0
      ? `月額賃料 ${rentNum.toLocaleString()}円`
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
