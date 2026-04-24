import type { Metadata, Viewport } from "next";
import { ServiceWorkerRegistrar } from "@/components/ServiceWorkerRegistrar";
import "./globals.css";
import { TooltipProvider } from "@/components/ui/tooltip";

export const viewport: Viewport = {
  width: "device-width",
  initialScale: 1,
  themeColor: "#0f172a",
};

export const metadata: Metadata = {
  title: "Yield-Guard — 不動産投資リスク可視化",
  description:
    "不動産投資のリスクを可視化するシミュレーター。利回り・デッドクロス・出口戦略を分析",
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang="ja">
      <body>
        <ServiceWorkerRegistrar />
        <TooltipProvider>{children}</TooltipProvider>
      </body>
    </html>
  );
}
