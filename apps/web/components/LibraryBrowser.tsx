"use client";

import { useState } from "react";
import { Search } from "lucide-react";
import type { Series } from "@/lib/data";
import { CoverCard } from "./CoverCard";

export function LibraryBrowser({ series }: { series: Series[] }) {
  const [q, setQ] = useState("");
  const [genre, setGenre] = useState<string | null>(null);

  const genres = [...new Set(series.flatMap((s) => s.genres))].sort();
  // filtro derivado, calculado na renderização (sem useState espelhado)
  const filtered = series.filter(
    (s) =>
      (!genre || s.genres.includes(genre)) &&
      s.title.toLowerCase().includes(q.trim().toLowerCase())
  );

  return (
    <div>
      <div className="flex flex-col sm:flex-row gap-3 mb-6">
        <div className="relative flex-1">
          <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
          <input
            value={q}
            onChange={(e) => setQ(e.target.value)}
            placeholder="Buscar por título..."
            className="w-full h-10 pl-9 pr-3 bg-card border border-border rounded-xl text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-1 focus:ring-ring"
          />
        </div>
      </div>

      <div className="flex flex-wrap gap-2 mb-6">
        {genres.map((g) => (
          <button
            key={g}
            onClick={() => setGenre(genre === g ? null : g)}
            className={`px-3 py-1.5 text-xs font-semibold rounded-full transition-all border ${
              genre === g
                ? "bg-primary text-primary-foreground border-primary shadow-lg shadow-primary/25"
                : "bg-secondary text-muted-foreground hover:text-foreground border-transparent"
            }`}
          >
            {g}
          </button>
        ))}
      </div>

      {filtered.length === 0 ? (
        <div className="bg-card border border-border rounded-xl p-10 text-center">
          <p className="text-sm text-muted-foreground">Nenhum título encontrado para "{q}".</p>
        </div>
      ) : (
        <div className="flex flex-wrap gap-4">
          {filtered.map((s) => (
            <CoverCard key={s.slug} series={s} />
          ))}
        </div>
      )}
    </div>
  );
}
