"use client";
import { useSearchParams } from "next/navigation";
import { Dashboard } from "@/components/Dashboard";

export function DashboardWithParams() {
  const searchParams = useSearchParams();
  return <Dashboard initialParams={searchParams} />;
}
