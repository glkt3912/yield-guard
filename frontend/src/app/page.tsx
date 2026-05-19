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

  // クイックモード: totalPrice、フルモード: landPrice + buildingCost を合算
  const quickPrice = Number(params.totalPrice);
  const fullPrice = (Number(params.landPrice) || 0) + (Number(params.buildingCost) || 0);
  const resolvedPrice = quickPrice > 0 ? quickPrice : fullPrice;

  const rentNum = Number(params.rent);
  const price = resolvedPrice > 0 ? `物件価格 ${resolvedPrice.toLocaleString()}万円` : null;
  const rent =
    params.rent && !isNaN(rentNum) && rentNum > 0 ? `月額賃料 ${rentNum.toLocaleString()}円` : null;

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
