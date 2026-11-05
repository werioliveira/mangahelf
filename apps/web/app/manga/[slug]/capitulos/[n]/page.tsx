import type { Metadata } from "next";
import Link from "next/link";
import Image from "next/image";
import { notFound } from "next/navigation";
import { ChevronLeft, ChevronRight } from "lucide-react";
import { bySlug } from "@/lib/data";
import { lookupChapterPages } from "@/lib/api";

type Props = { params: Promise<{ slug: string; n: string }> };

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { slug, n } = await params;
  const s = bySlug(slug);
  return { title: s ? `Capítulo ${n} · ${s.title}` : "Capítulo", robots: { index: false } };
}

const navBtn =
  "p-2 rounded-lg bg-secondary hover:bg-secondary/70 text-muted-foreground hover:text-foreground transition-all disabled:opacity-30 disabled:pointer-events-none";

export default async function ChapterPage({ params }: Props) {
  const { slug, n } = await params;
  const s = bySlug(slug);
  const num = Number(n);

  if (!s || !Number.isInteger(num) || num < 1 || num > s.chapters) notFound();

  // Busca capítulo e páginas reais da API Go (upload direto para storage via presigned URL)
  const data = await lookupChapterPages(slug, num, "full");
  if (!data) notFound();

  const { chapter, pages } = data;

  return (
    <div className="max-w-3xl mx-auto px-4 py-6">
      <div className="sticky top-16 z-40 flex items-center justify-between gap-3 bg-background/90 backdrop-blur-xl border border-border rounded-xl px-3 py-2 mb-6">
        <Link href={`/manga/${s.slug}`} className="flex items-center gap-1.5 text-sm font-medium text-muted-foreground hover:text-foreground transition-all">
          <ChevronLeft className="w-4 h-4" />
          <span className="line-clamp-1">{s.title}</span>
        </Link>
        <span className="text-sm font-semibold text-foreground whitespace-nowrap">Capítulo {chapter.number}</span>
        <div className="flex gap-1.5">
          {num > 1 ? (
            <Link aria-label="Capítulo anterior" href={`${num - 1}`} className={navBtn}>
              <ChevronLeft className="w-4 h-4" />
            </Link>
          ) : (
            <span aria-disabled className={`${navBtn} opacity-30 pointer-events-none`}>
              <ChevronLeft className="w-4 h-4" />
            </span>
          )}
          {num < s.chapters ? (
            <Link aria-label="Próximo capítulo" href={`${num + 1}`} className={navBtn}>
              <ChevronRight className="w-4 h-4" />
            </Link>
          ) : (
            <span aria-disabled className={`${navBtn} opacity-30 pointer-events-none`}>
              <ChevronRight className="w-4 h-4" />
            </span>
          )}
        </div>
      </div>

      <div className="space-y-4">
        {pages.map((p: { number: number; url: string; width: number; height: number; storage_key?: string }) => (
          <div key={p.number} className="relative w-full aspect-[2/3] rounded-xl overflow-hidden ring-1 ring-white/5 bg-card">
            <Image src={p.url} alt={`Página ${p.number}`} fill sizes="(max-width: 768px) 100vw, 768px" className="object-cover" />
            <span className="absolute bottom-2 right-2 px-2 py-0.5 text-[10px] font-bold text-white bg-black/60 backdrop-blur-sm rounded-md">
              Página {p.number}
            </span>
          </div>
        ))}
      </div>

      <div className="flex items-center justify-between mt-8">
        {num > 1 ? (
          <Link href={`${num - 1}`} className="flex items-center gap-1.5 text-sm font-medium text-muted-foreground hover:text-foreground">
            <ChevronLeft className="w-4 h-4" />
            Capítulo {num - 1}
          </Link>
        ) : (
          <span />
        )}
        {num < s.chapters ? (
          <Link href={`${num + 1}`} className="flex items-center gap-1.5 text-sm font-semibold text-primary hover:text-primary/80">
            Capítulo {num + 1}
            <ChevronRight className="w-4 h-4" />
          </Link>
        ) : (
          <span className="text-xs text-muted-foreground">Você está no capítulo mais recente</span>
        )}
      </div>
    </div>
  );
}
