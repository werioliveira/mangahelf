"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import Image from "next/image";
import { Star, BookOpen, ChevronLeft, ChevronRight } from "lucide-react";
import type { Series } from "@/lib/data";
import { btnOutline, btnPrimary } from "@/lib/ui";

export function HeroCarousel({ items }: { items: Series[] }) {
  const [index, setIndex] = useState(0);
  const [paused, setPaused] = useState(false);

  useEffect(() => {
    if (paused || items.length < 2) return;
    const t = setInterval(() => setIndex((i) => (i + 1) % items.length), 7000);
    return () => clearInterval(t);
  }, [paused, items.length]);

  const current = items[index];
  const prev = items[(index - 1 + items.length) % items.length];
  const next = items[(index + 1) % items.length];

  const sideCover = (s: Series, pos: "left" | "right") => (
    <div
      className={`absolute ${
        pos === "left" ? "-left-8 md:-left-12" : "-right-8 md:-right-12"
      } z-0 opacity-40 hidden sm:block`}
    >
      <div className={`w-36 md:w-44 h-52 md:h-64 rounded-xl overflow-hidden shadow-2xl ${pos === "left" ? "-rotate-6" : "rotate-6"}`}>
        <Image src={s.cover} alt="" fill sizes="176px" className="object-cover" />
      </div>
    </div>
  );

  return (
    <section
      className="relative w-full overflow-hidden"
      onMouseEnter={() => setPaused(true)}
      onMouseLeave={() => setPaused(false)}
    >
      <div className="absolute inset-0 overflow-hidden">
        <div className="absolute inset-0">
          <Image
            key={current.slug}
            src={current.cover}
            alt=""
            fill
            priority
            sizes="100vw"
            className="object-cover blur-3xl scale-150 animate-fade-in"
          />
        </div>
        <div className="absolute inset-0 bg-gradient-to-t from-background via-background/60 to-background/40" />
      </div>

      <div className="relative max-w-[1400px] mx-auto px-6 py-10 md:py-16">
        <div className="flex flex-col md:flex-row items-center gap-8 md:gap-12">
          <div key={current.slug} className="flex-1 text-center md:text-left z-10 animate-fade-in">
            <span className="inline-flex items-center gap-1.5 px-3 py-1 text-xs font-semibold text-accent bg-accent/10 border border-accent/20 rounded-full mb-4">
              <Star className="w-3 h-3 fill-accent" />
              DESTAQUE
            </span>
            <h1 className="font-heading text-4xl md:text-5xl lg:text-6xl font-black text-foreground mb-3 leading-tight">
              {current.title}
            </h1>
            <p className="text-muted-foreground text-base md:text-lg mb-2 max-w-md">{current.tagline}</p>
            <div className="flex items-center gap-4 justify-center md:justify-start mb-6">
              <span className="text-sm text-muted-foreground">{current.genres.join(" / ")}</span>
              <span className="text-muted-foreground">•</span>
              <span className="flex items-center gap-1 text-sm text-accent">
                <Star className="w-3.5 h-3.5 fill-accent" />
                {current.rating}
              </span>
              <span className="text-muted-foreground">•</span>
              <span className="flex items-center gap-1 text-sm text-muted-foreground">
                <BookOpen className="w-3.5 h-3.5" />
                {current.chapters} Caps
              </span>
            </div>
            <div className="flex gap-3 justify-center md:justify-start">
              <Link href={`/manga/${current.slug}`} className={btnPrimary}>
                Ler Agora
              </Link>
              <Link href={`/manga/${current.slug}`} className={btnOutline}>
                Detalhes
              </Link>
            </div>
          </div>

          <div className="relative flex items-center justify-center w-full md:w-auto">
            {sideCover(prev, "left")}
            <div className="relative z-10">
              <div className="relative w-52 md:w-64 lg:w-72 h-72 md:h-96 rounded-2xl overflow-hidden shadow-2xl shadow-primary/20 ring-1 ring-white/10">
                <Image
                  key={current.slug}
                  src={current.cover}
                  alt={current.title}
                  fill
                  priority
                  sizes="(max-width: 768px) 208px, 288px"
                  className="object-cover animate-fade-in"
                />
                <div className="absolute inset-0 bg-gradient-to-t from-black/40 to-transparent" />
              </div>
              <div className="absolute -inset-4 bg-primary/10 rounded-3xl blur-2xl -z-10 animate-glow-pulse" />
            </div>
            {sideCover(next, "right")}
          </div>
        </div>

        <div className="flex items-center justify-center gap-4 mt-8">
          <button
            aria-label="Anterior"
            onClick={() => setIndex((i) => (i - 1 + items.length) % items.length)}
            className="p-2 text-muted-foreground hover:text-foreground hover:bg-secondary rounded-lg transition-all"
          >
            <ChevronLeft className="w-4 h-4" />
          </button>
          <div className="flex gap-2">
            {items.map((s, i) => (
              <button
                key={s.slug}
                aria-label={`Ir para ${s.title}`}
                onClick={() => setIndex(i)}
                className={`h-1.5 rounded-full transition-all duration-500 ${
                  i === index ? "w-8 bg-primary" : "w-1.5 bg-muted-foreground/30 hover:bg-muted-foreground/60"
                }`}
              />
            ))}
          </div>
          <button
            aria-label="Próximo"
            onClick={() => setIndex((i) => (i + 1) % items.length)}
            className="p-2 text-muted-foreground hover:text-foreground hover:bg-secondary rounded-lg transition-all"
          >
            <ChevronRight className="w-4 h-4" />
          </button>
        </div>
      </div>
    </section>
  );
}
