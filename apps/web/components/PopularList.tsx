"use client";

import { useState } from "react";
import { TrendingUp } from "lucide-react";
import type { Period, Series } from "@/lib/data";
import { RankRow } from "./RankRow";

const tabs: { id: Period; label: string }[] = [
  { id: "daily", label: "Diário" },
  { id: "weekly", label: "Semanal" },
  { id: "monthly", label: "Mensal" },
];

export function PopularList({ data }: { data: Record<Period, Series[]> }) {
  const [period, setPeriod] = useState<Period>("daily");

  return (
    <div>
      <div className="flex items-center gap-3 mb-4">
        <div className="flex items-center justify-center w-10 h-10 rounded-xl bg-destructive/10 border border-destructive/20 text-destructive">
          <TrendingUp className="w-5 h-5" />
        </div>
        <h2 className="font-heading text-xl font-black text-foreground tracking-tight">POPULARES</h2>
      </div>

      <div className="flex gap-1 p-1 bg-secondary rounded-xl mb-4">
        {tabs.map((t) => {
          const active = t.id === period;
          return (
            <button
              key={t.id}
              onClick={() => setPeriod(t.id)}
              className={`relative flex-1 py-2 text-xs font-semibold rounded-lg transition-all duration-300 ${
                active ? "text-primary-foreground" : "text-muted-foreground hover:text-foreground"
              }`}
            >
              {active && <span className="absolute inset-0 bg-primary rounded-lg" />}
              <span className="relative z-10">{t.label}</span>
            </button>
          );
        })}
      </div>

      <div className="bg-card rounded-xl border border-border overflow-hidden">
        {data[period].map((s, i) => (
          <RankRow key={s.slug} series={s} rank={i + 1} />
        ))}
      </div>
    </div>
  );
}
