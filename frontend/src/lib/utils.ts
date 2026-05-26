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

/** 円 → 万円文字列 (入力フィールド向け、0のとき空文字) */
export function toMan(yen: number): string {
  if (yen === 0) return "";
  return String(Math.round(yen / 10_000));
}

/** 円 → 万円文字列・小数対応 (月額賃料など小数が生じるフィールド向け、0のとき空文字) */
export function toManFloat(yen: number): string {
  if (yen === 0) return "";
  const v = yen / 10_000;
  return v % 1 === 0 ? String(v) : v.toFixed(1);
}

/** 万円文字列 → 円 */
export function fromMan(s: string): number {
  return (parseFloat(s) || 0) * 10_000;
}

/** 小数レート → %文字列 (入力フィールド向け) */
export function toPct(rate: number, digits = 2): string {
  return (rate * 100).toFixed(digits);
}

/** %文字列 → 小数レート */
export function fromPct(s: string): number {
  return (parseFloat(s) || 0) / 100;
}
