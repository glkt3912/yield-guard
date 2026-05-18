import * as React from "react";
import { cn } from "@/lib/utils";

interface BadgeProps extends React.HTMLAttributes<HTMLDivElement> {
  variant?: "default" | "success" | "warning" | "danger" | "outline";
}

function Badge({ className, variant = "default", ...props }: BadgeProps) {
  return (
    <div
      className={cn(
        "inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-semibold transition-colors",
        {
          "bg-primary text-white": variant === "default",
          "bg-success-bg text-success-fg": variant === "success",
          "bg-warning-bg text-warning-fg": variant === "warning",
          "bg-danger-bg text-danger-fg": variant === "danger",
          "border border-border text-foreground": variant === "outline",
        },
        className
      )}
      {...props}
    />
  );
}

export { Badge };
