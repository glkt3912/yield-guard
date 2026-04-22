"use client";
import { Suspense } from "react";
import { useSearchParams } from "next/navigation";
import { Dashboard } from "@/components/Dashboard";

function DashboardWithParams() {
  const searchParams = useSearchParams();
  return <Dashboard initialParams={searchParams} />;
}

export default function Home() {
  return (
    <Suspense fallback={<Dashboard />}>
      <DashboardWithParams />
    </Suspense>
  );
}
