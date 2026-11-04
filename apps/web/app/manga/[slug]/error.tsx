"use client";

import { AlertTriangle } from "lucide-react";
import { btnPrimary } from "@/lib/ui";

export default function SeriesError({ error, reset }: { error: Error; reset: () => void }) {
  return (
    <div className="max-w-md mx-auto px-6 py-24 text-center">
      <div className="inline-flex items-center justify-center w-14 h-14 rounded-2xl bg-destructive/10 border border-destructive/20 mb-5">
        <AlertTriangle className="w-6 h-6 text-destructive" />
      </div>
      <h2 className="font-heading text-xl font-black text-foreground mb-2">Algo deu errado</h2>
      <p className="text-sm text-muted-foreground mb-6">{error.message || "Falha ao carregar o título."}</p>
      <button onClick={reset} className={btnPrimary}>
        Tentar novamente
      </button>
    </div>
  );
}
