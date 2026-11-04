import type { Metadata } from "next";
import { Trophy } from "lucide-react";
import { SectionHeader } from "@/components/SectionHeader";
import { RankRow } from "@/components/RankRow";
import { series } from "@/lib/data";

export const metadata: Metadata = {
  title: "Ranking",
  description: "Os títulos mais lidos da semana no MangaShelf.",
};

const viewsNumber = (v: string) => parseFloat(v) * (v.includes("K") ? 1000 : 1);

export default function RankingPage() {
  const ranked = [...series].sort((a, b) => viewsNumber(b.views) - viewsNumber(a.views));

  return (
    <section className="relative py-8 md:py-12">
      <div className="max-w-[1400px] mx-auto px-6">
        <SectionHeader icon={Trophy} tone="destructive" title="RANKING 🏆" subtitle="Mais lidos de todos os tempos" />
        <div className="max-w-3xl mx-auto lg:mx-0">
          <div className="bg-card rounded-xl border border-border overflow-hidden">
            {ranked.map((s, i) => (
              <RankRow key={s.slug} series={s} rank={i + 1} />
            ))}
          </div>
        </div>
      </div>
    </section>
  );
}
