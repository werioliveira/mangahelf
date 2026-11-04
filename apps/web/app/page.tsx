import { History } from "lucide-react";
import { HeroCarousel } from "@/components/HeroCarousel";
import { Top10 } from "@/components/Top10";
import { SectionHeader, Divider } from "@/components/SectionHeader";
import { UpdateCard } from "@/components/UpdateCard";
import { PopularList } from "@/components/PopularList";
import { bySlug, featured, top10, updates, popular } from "@/lib/data";

export default function HomePage() {
  return (
    <>
      <HeroCarousel items={featured} />
      <Divider />
      <Top10 items={top10} />

      <section className="relative py-8 md:py-12">
        <div className="max-w-[1400px] mx-auto px-6">
          <div className="flex flex-col lg:flex-row gap-8">
            <div className="flex-1 min-w-0">
              <SectionHeader
                icon={History}
                tone="accent"
                title="RECENTEMENTE ATUALIZADAS"
                subtitle="Últimos capítulos lançados"
              />
              <div className="grid grid-cols-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5 gap-3">
                {updates.map((u) => (
                  <UpdateCard key={u.slug} series={bySlug(u.slug)!} chapters={u.chapters} />
                ))}
              </div>
            </div>
            <div className="w-full lg:w-80 flex-shrink-0">
              <PopularList data={popular} />
            </div>
          </div>
        </div>
      </section>
    </>
  );
}
