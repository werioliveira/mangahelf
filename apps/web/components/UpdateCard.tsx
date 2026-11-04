import Link from "next/link";
import Image from "next/image";
import { Star } from "lucide-react";
import type { Series, Chapter } from "@/lib/data";

export function UpdateCard({ series, chapters }: { series: Series; chapters: Chapter[] }) {
  return (
    <div className="group">
      <Link
        className="block bg-card hover:bg-secondary/50 border border-transparent hover:border-border/60 rounded-xl overflow-hidden transition-all duration-300"
        href={`/manga/${series.slug}`}
      >
        <div className="relative w-full aspect-[2/3] overflow-hidden">
          <Image
            src={series.cover}
            alt={series.title}
            fill
            sizes="(max-width: 640px) 50vw, (max-width: 1024px) 33vw, (max-width: 1280px) 25vw, 20vw"
            className="object-cover group-hover:scale-105 transition-transform duration-500"
          />
          <div className="absolute inset-0 bg-gradient-to-t from-black/70 via-black/10 to-transparent" />
          <span className="absolute top-2 left-2 px-2 py-0.5 text-[9px] font-black tracking-widest uppercase bg-primary/90 text-white rounded-md backdrop-blur-sm">
            {series.type}
          </span>
          <div className="absolute bottom-2 left-2 flex items-center gap-1 bg-black/60 backdrop-blur-sm rounded-md px-1.5 py-0.5">
            <Star className="w-3 h-3 fill-accent text-accent" />
            <span className="text-[10px] font-bold text-white">{series.rating}</span>
          </div>
        </div>
        <div className="p-3 space-y-2.5">
          <h3 className="text-sm font-bold text-foreground leading-tight line-clamp-2 group-hover:text-primary transition-colors">
            {series.title}
          </h3>
          <div className="flex flex-col gap-1.5">
            {chapters.map((c, i) => (
              <div key={c.number} className="flex items-center justify-between gap-2">
                <div className="flex items-center gap-1.5 min-w-0">
                  <span
                    className={`w-1.5 h-1.5 rounded-full flex-shrink-0 ${
                      i === 0 ? "bg-accent" : "bg-muted-foreground/30"
                    }`}
                  />
                  <span
                    className={`text-[11px] font-medium truncate ${
                      i === 0 ? "text-foreground" : "text-muted-foreground"
                    }`}
                  >
                    Capítulo {c.number}
                  </span>
                  {c.isNew && (
                    <span className="flex-shrink-0 px-1.5 py-0.5 text-[9px] font-black bg-accent/20 text-accent border border-accent/30 rounded">
                      NOVO
                    </span>
                  )}
                </div>
                <span className="text-[10px] text-muted-foreground/50 flex-shrink-0 whitespace-nowrap">{c.ago}</span>
              </div>
            ))}
          </div>
        </div>
      </Link>
    </div>
  );
}
