"use client";

import { useRef } from "react";
import { ChevronLeft, ChevronRight, Flame } from "lucide-react";
import type { Series } from "@/lib/data";
import { CoverCard } from "./CoverCard";
import { SectionHeader } from "./SectionHeader";

export function Top10({ items }: { items: { rank: number; series: Series }[] }) {
  const scroller = useRef<HTMLDivElement>(null);
  const scroll = (dir: number) => scroller.current?.scrollBy({ left: dir * 360, behavior: "smooth" });

  const arrow =
    "p-2 text-muted-foreground hover:text-foreground bg-secondary hover:bg-secondary/80 rounded-lg transition-all";

  return (
    <section className="relative py-8 md:py-12">
      <div className="max-w-[1400px] mx-auto px-6">
        <SectionHeader
          icon={Flame}
          tone="primary"
          title="TOP 10"
          subtitle="Os mais populares da semana"
          actions={
            <div className="flex gap-2">
              <button aria-label="Rolar para trás" onClick={() => scroll(-1)} className={arrow}>
                <ChevronLeft className="w-4 h-4" />
              </button>
              <button aria-label="Rolar para frente" onClick={() => scroll(1)} className={arrow}>
                <ChevronRight className="w-4 h-4" />
              </button>
            </div>
          }
        />
        <div ref={scroller} className="flex gap-4 overflow-x-auto pb-4 scrollbar-hide">
          {items.map(({ rank, series }) => (
            <CoverCard key={series.slug} series={series} rank={rank} />
          ))}
        </div>
      </div>
    </section>
  );
}
