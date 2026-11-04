import type { Metadata } from "next";
import { Heart } from "lucide-react";
import { SectionHeader } from "@/components/SectionHeader";
import { CoverCard } from "@/components/CoverCard";
import { bySlug } from "@/lib/data";

export const metadata: Metadata = {
  title: "Favoritos",
  description: "Seus títulos salvos no MangaShelf.",
};

// shortcut: lista demo fixa; ligar no endpoint de favoritos quando houver auth + API
const favoriteSlugs = ["solo-leveling", "shadow-monarch", "the-beginning-after-the-end", "return-of-the-mount-hua-sect"];

export default function FavoritesPage() {
  const favs = favoriteSlugs.map((s) => bySlug(s)!).filter(Boolean);

  return (
    <section className="relative py-8 md:py-12">
      <div className="max-w-[1400px] mx-auto px-6">
        <SectionHeader icon={Heart} tone="destructive" title="FAVORITOS" subtitle="Sua coleção salva" />
        {favs.length === 0 ? (
          <div className="bg-card border border-border rounded-xl p-10 text-center">
            <p className="text-sm text-muted-foreground">Você ainda não favoritou nenhum título.</p>
          </div>
        ) : (
          <div className="flex flex-wrap gap-4">
            {favs.map((s) => (
              <CoverCard key={s.slug} series={s} />
            ))}
          </div>
        )}
      </div>
    </section>
  );
}
