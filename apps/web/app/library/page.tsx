import type { Metadata } from "next";
import { LibraryBig } from "lucide-react";
import { SectionHeader } from "@/components/SectionHeader";
import { LibraryBrowser } from "@/components/LibraryBrowser";
import { series } from "@/lib/data";

export const metadata: Metadata = {
  title: "Biblioteca",
  description: "Todos os manhwas, manhuas e mangas do MangaShelf.",
};

export default function LibraryPage() {
  return (
    <section className="relative py-8 md:py-12">
      <div className="max-w-[1400px] mx-auto px-6">
        <SectionHeader icon={LibraryBig} tone="primary" title="BIBLIOTECA" subtitle={`${series.length} títulos na coleção`} />
        <LibraryBrowser series={series} />
      </div>
    </section>
  );
}
