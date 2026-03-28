import type { Metadata } from "next";
import "./globals.css";

export const metadata: Metadata = {
  title: "Yield-Guard — 不動産投資リスク可視化",
  description:
    "国交省APIの土地取引実績と表面利回り・デッドクロス・出口戦略でリスクを可視化するMVP",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="ja">
      <body>{children}</body>
    </html>
  );
}
