"use client";
import React, { useState } from "react";
import { Tooltip, TooltipContent, TooltipTrigger } from "@/components/ui/tooltip";
import { Info } from "lucide-react";
import { GLOSSARY } from "@/lib/glossary";

interface Props {
  term: keyof typeof GLOSSARY;
  children: React.ReactNode;
}

export function TermTooltip({ term, children }: Props) {
  const entry = GLOSSARY[term];
  const [open, setOpen] = useState(false);
  if (!entry) return <>{children}</>;

  return (
    <>
      <Tooltip open={open} onOpenChange={setOpen}>
        <TooltipTrigger asChild>
          <span
            className="inline-flex items-center gap-0.5 cursor-help border-b border-dashed border-muted-foreground/50"
            onClick={() => setOpen((v) => !v)}
          >
            {children}
            <Info className="h-3 w-3 text-muted-foreground" />
          </span>
        </TooltipTrigger>
        <TooltipContent className="max-w-xs">
          <p className="font-semibold text-sm">{entry.title}</p>
          <p className="text-xs mt-1">{entry.body}</p>
          <p className="text-xs mt-1 text-muted-foreground">💡 {entry.tip}</p>
        </TooltipContent>
      </Tooltip>
      {open && (
        <div className="mt-1 rounded-md border bg-muted/50 p-2 text-xs md:hidden">
          <p className="font-semibold">{entry.title}</p>
          <p className="mt-0.5">{entry.body}</p>
          <p className="mt-0.5 text-muted-foreground">💡 {entry.tip}</p>
        </div>
      )}
    </>
  );
}
