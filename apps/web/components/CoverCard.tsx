import Link from "next/link";
import Image from "next/image";
import type { Series } from "@/lib/data";

export function CoverCard({ series, rank }: { series: Series; rank?: number }) {
  return (
    <div className="group flex-shrink-0 w-[140px] md:w-[160px]">
      <Link className="block" href={`/manga/${series.slug}`}>
        <div className="relative">
          <div className="relative w-full aspect-[3/4] rounded-xl overflow-hidden shadow-lg shadow-black/30 ring-1 ring-white/5 group-hover:ring-primary/30 transition-all duration-300 group-hover:shadow-primary/15">
            <Image
              src={series.cover}
              alt={series.title}
              fill
              sizes="160px"
              className="object-cover group-hover:scale-105 transition-transform duration-500"
            />
            <div className="absolute inset-0 bg-gradient-to-t from-black/70 via-transparent to-transparent" />
            {rank != null && (
              <div className="absolute -bottom-2 -left-2 z-10">
                <span className="font-heading text-5xl md:text-6xl font-black leading-none rank-stroke">{rank}</span>
              </div>
            )}
            <div className="absolute top-2 right-2">
              <span className="px-2 py-0.5 text-[10px] font-bold text-white bg-primary/80 backdrop-blur-sm rounded-md">
                {series.chapters} CAPS
              </span>
            </div>
          </div>
          <h3 className="mt-3 text-sm font-semibold text-foreground leading-tight line-clamp-2 group-hover:text-primary transition-colors">
            {series.title}
          </h3>
        </div>
      </Link>
    </div>
  );
}
