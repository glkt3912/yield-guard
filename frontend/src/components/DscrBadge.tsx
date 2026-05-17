import React from "react";
import { Badge } from "@/components/ui/badge";
import { CheckCircle2, AlertTriangle, XCircle } from "lucide-react";

export function getDscrColorClass(dscr: number): string {
  if (dscr >= 1.2) return "text-green-600";
  if (dscr >= 1.0) return "text-yellow-600";
  return "text-red-600";
}

export function getDscrBadge(dscr: number): React.ReactElement {
  if (dscr >= 1.2)
    return (
      <Badge variant="success" className="flex items-center gap-1">
        <CheckCircle2 className="h-3 w-3" />
        安全
      </Badge>
    );
  if (dscr >= 1.0)
    return (
      <Badge variant="warning" className="flex items-center gap-1">
        <AlertTriangle className="h-3 w-3" />
        注意
      </Badge>
    );
  return (
    <Badge variant="danger" className="flex items-center gap-1">
      <XCircle className="h-3 w-3" />
      危険
    </Badge>
  );
}

export function getDscrIcon(dscr: number): React.ReactElement {
  if (dscr >= 1.2) return <CheckCircle2 className="h-3 w-3" />;
  if (dscr >= 1.0) return <AlertTriangle className="h-3 w-3" />;
  return <XCircle className="h-3 w-3" />;
}
