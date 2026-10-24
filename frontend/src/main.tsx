import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import { ServiceWorkerRegistrar } from "@/components/ServiceWorkerRegistrar";
import { TooltipProvider } from "@/components/ui/tooltip";
import { ToastProvider } from "@/components/ui/toast";
import RootLayoutClient from "@/components/RootLayoutClient";
import { Dashboard } from "@/components/Dashboard";
import "./globals.css";

createRoot(document.getElementById("root")!).render(
  <StrictMode>
    <ServiceWorkerRegistrar />
    <RootLayoutClient>
      <ToastProvider>
        <TooltipProvider>
          <Dashboard initialParams={new URLSearchParams(location.search)} />
        </TooltipProvider>
      </ToastProvider>
    </RootLayoutClient>
  </StrictMode>
);
