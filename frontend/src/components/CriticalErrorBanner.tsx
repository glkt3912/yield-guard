import { XOctagon, AlertTriangle } from "lucide-react";
import type { CriticalError } from "@/types/investment";

interface Props {
  errors: CriticalError[];
}

export function CriticalErrorBanner({ errors }: Props) {
  if (errors.length === 0) return null;

  return (
    <div className="space-y-2">
      {errors.map((err) => (
        <div
          key={err.code}
          role="alert"
          className={`flex items-start gap-3 rounded-md border-2 p-4 ${
            err.status === "REJECT"
              ? "border-red-500 bg-red-50 text-red-900"
              : "border-yellow-400 bg-yellow-50 text-yellow-900"
          }`}
        >
          {err.status === "REJECT" ? (
            <XOctagon className="mt-0.5 h-5 w-5 shrink-0 text-red-600" />
          ) : (
            <AlertTriangle className="mt-0.5 h-5 w-5 shrink-0 text-yellow-600" />
          )}
          <div>
            <p className="text-sm font-bold">
              {err.status === "REJECT" ? "⛔ 一発退場" : "⚠ 警告"}: {err.code}
            </p>
            <p className="mt-0.5 text-sm">{err.message}</p>
          </div>
        </div>
      ))}
    </div>
  );
}
