import Link from "next/link";
import Image from "next/image";
import { Eye } from "lucide-react";
import type { Series } from "@/lib/data";

const rankColors = ["text-accent", "text-primary", "text-destructive"];

export function RankRow({ series, rank }: { series: Series; rank: number }) {
  return (
    <Link
      className="flex items-center gap-3 p-3 hover:bg-secondary/60 transition-all duration-200 group"
      href={`/manga/${series.slug}`}
    >
      <span className={`font-heading text-lg font-black w-6 text-center ${rankColors[rank - 1] ?? "text-muted-foreground"}`}>
        {rank}
      </span>
      <div className="w-10 h-14 rounded-md overflow-hidden flex-shrink-0 ring-1 ring-white/5">
        <Image src={series.cover} alt={series.title} width={40} height={56} sizes="40px" className="object-cover" />
      </div>
      <div className="flex-1 min-w-0">
        <h4 className="text-sm font-semibold text-foreground line-clamp-1 group-hover:text-primary transition-colors">
          {series.title}
        </h4>
        <div className="flex items-center gap-2 mt-0.5">
          <span className="text-[10px] text-muted-foreground">{series.genres[0]}</span>
        </div>
      </div>
      <div className="flex items-center gap-1 text-xs text-muted-foreground">
        <Eye className="w-3.5 h-3.5" />
        {series.views}
      </div>
    </Link>
  );
}
