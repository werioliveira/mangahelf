import type { Metadata } from "next";
import { notFound } from "next/navigation";
import Link from "next/link";
import Image from "next/image";
import { Star, BookOpen, Eye, ChevronRight, Heart } from "lucide-react";
import { SectionHeader, Divider } from "@/components/SectionHeader";
import { btnPrimary, btnOutline } from "@/lib/ui";
import { bySlug } from "@/lib/data";

type Props = { params: Promise<{ slug: string }> };

export async function generateMetadata({ params }: Props): Promise<Metadata> {
  const { slug } = await params;
  const s = bySlug(slug);
  if (!s) return { title: "Título não encontrado" };
  return {
    title: s.title,
    description: s.tagline,
    openGraph: { title: s.title, description: s.synopsis, images: [s.cover] },
  };
}

export default async function SeriesPage({ params }: Props) {
  const { slug } = await params;
  const s = bySlug(slug);
  if (!s) notFound();

  const latest = s.chapters;
  const list = Array.from({ length: Math.min(20, s.chapters) }, (_, i) => latest - i);

  return (
    <>
      <section className="relative w-full overflow-hidden">
        <div className="absolute inset-0 overflow-hidden">
          <Image src={s.cover} alt="" fill priority sizes="100vw" className="object-cover blur-3xl scale-150 opacity-30" />
          <div className="absolute inset-0 bg-gradient-to-t from-background via-background/60 to-background/40" />
        </div>
        <div className="relative max-w-[1400px] mx-auto px-6 py-10 md:py-14">
          <div className="flex flex-col md:flex-row items-center md:items-start gap-8 md:gap-12">
            <div className="relative flex-shrink-0">
              <div className="relative w-52 md:w-64 h-72 md:h-96 rounded-2xl overflow-hidden shadow-2xl shadow-primary/20 ring-1 ring-white/10">
                <Image src={s.cover} alt={s.title} fill priority sizes="(max-width: 768px) 208px, 256px" className="object-cover" />
                <div className="absolute inset-0 bg-gradient-to-t from-black/40 to-transparent" />
              </div>
              <div className="absolute -inset-4 bg-primary/10 rounded-3xl blur-2xl -z-10 animate-glow-pulse" />
            </div>

            <div className="flex-1 min-w-0 text-center md:text-left">
              <div className="flex flex-wrap items-center justify-center md:justify-start gap-2 mb-4">
                <span className="px-2 py-0.5 text-[9px] font-black tracking-widest uppercase bg-primary/90 text-white rounded-md">
                  {s.type}
                </span>
                {s.genres.map((g) => (
                  <Link
                    key={g}
                    href="/library"
                    className="px-3 py-1 text-xs font-semibold bg-secondary hover:bg-secondary/70 text-muted-foreground hover:text-foreground rounded-full transition-all"
                  >
                    {g}
                  </Link>
                ))}
              </div>
              <h1 className="font-heading text-3xl md:text-5xl font-black text-foreground mb-3 leading-tight">{s.title}</h1>
              <div className="flex items-center gap-4 justify-center md:justify-start mb-5 text-sm">
                <span className="flex items-center gap-1 text-accent">
                  <Star className="w-4 h-4 fill-accent" />
                  <span className="font-bold">{s.rating}</span>
                </span>
                <span className="text-muted-foreground">•</span>
                <span className="flex items-center gap-1 text-muted-foreground">
                  <BookOpen className="w-4 h-4" />
                  {s.chapters} capítulos
                </span>
                <span className="text-muted-foreground">•</span>
                <span className="flex items-center gap-1 text-muted-foreground">
                  <Eye className="w-4 h-4" />
                  {s.views}
                </span>
              </div>
              <p className="text-sm md:text-base text-muted-foreground mb-6 max-w-2xl leading-relaxed">{s.synopsis}</p>
              <div className="flex gap-3 justify-center md:justify-start">
                <Link href={`/manga/${s.slug}/capitulos/1`} className={btnPrimary}>
                  Começar a ler
                </Link>
                <Link href={`/manga/${s.slug}/capitulos/${latest}`} className={btnOutline}>
                  <Heart className="w-4 h-4" />
                  Capítulo {latest}
                </Link>
              </div>
            </div>
          </div>
        </div>
      </section>

      <Divider />

      <section className="relative py-8 md:py-12">
        <div className="max-w-[1400px] mx-auto px-6">
          <SectionHeader icon={BookOpen} tone="accent" title="CAPÍTULOS" subtitle={`Últimos ${list.length} de ${s.chapters}`} />
          <div className="max-w-3xl bg-card rounded-xl border border-border overflow-hidden">
            {list.map((n, i) => (
              <Link
                key={n}
                href={`/manga/${s.slug}/capitulos/${n}`}
                className="flex items-center gap-3 px-4 py-3 hover:bg-secondary/60 transition-all duration-200 group border-b border-border/40 last:border-b-0"
              >
                <span className={`w-1.5 h-1.5 rounded-full flex-shrink-0 ${i === 0 ? "bg-accent" : "bg-muted-foreground/30"}`} />
                <span className={`flex-1 text-sm font-medium ${i === 0 ? "text-foreground" : "text-muted-foreground"}`}>
                  Capítulo {n}
                </span>
                {i === 0 && (
                  <span className="px-1.5 py-0.5 text-[9px] font-black bg-accent/20 text-accent border border-accent/30 rounded">
                    NOVO
                  </span>
                )}
                <span className="text-[10px] text-muted-foreground/50 whitespace-nowrap">
                  {i === 0 ? "1 hora atrás" : `${i + 1} dias atrás`}
                </span>
                <ChevronRight className="w-4 h-4 text-muted-foreground/40 group-hover:text-primary transition-colors" />
              </Link>
            ))}
          </div>
        </div>
      </section>
    </>
  );
}
