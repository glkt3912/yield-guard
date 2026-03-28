import { type ClassValue, clsx } from "clsx";
import { twMerge } from "tailwind-merge";

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs));
}

/** 万円単位でフォーマット (例: 1234.5万円) */
export function formatMan(value: number, digits = 1): string {
  return `${(value / 10_000).toFixed(digits)}万円`;
}

/** パーセント表示 (例: 8.25%) */
export function formatPct(value: number, digits = 2): string {
  return `${(value * 100).toFixed(digits)}%`;
}

/** 整数カンマ区切り */
export function formatYen(value: number): string {
  return `¥${Math.round(value).toLocaleString("ja-JP")}`;
}

/** 坪単価フォーマット */
export function formatTsubo(value: number): string {
  if (value >= 10_000) return `${(value / 10_000).toFixed(1)}万円/坪`;
  return `${Math.round(value).toLocaleString("ja-JP")}円/坪`;
}
