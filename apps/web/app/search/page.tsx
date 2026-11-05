"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { Search, BookOpen, Clock } from "lucide-react";
import { useSearchSeries } from "@/lib/api";

export default function SearchPage() {
  const [query, setQuery] = useState("");
  const [publicOnly, setPublicOnly] = useState(false);
  const { data, isLoading, error } = useSearchSeries(query, 20, publicOnly);

  useEffect(() => {
    setQuery("");
  }, []);

  const results = data?.results ?? [];
  const total = data?.total ?? 0;

  return (
    <section className="relative py-12 md:py-20">
      <div className="max-w-[1400px] mx-auto px-6">
        <div className="flex flex-col md:flex-row items-center justify-between gap-4 mb-8">
          <div className="relative flex-1">
            <Search className="absolute left-3 top-1/2 -translate-y-1/2 w-4 h-4 text-muted-foreground" />
            <input
              type="text"
              value={query}
              onChange={(e) => setQuery(e.target.value)}
              onSubmit={(e) => e.preventDefault()}
              placeholder="Buscar por título ou sinopse..."
              className="w-full h-11 pl-10 pr-4 bg-card border border-border rounded-xl text-sm text-foreground placeholder:text-muted-foreground focus:outline-none focus:ring-2 focus:ring-ring"
            />
          </div>
          <div className="flex items-center gap-2">
            <button
              onClick={() => setPublicOnly(!publicOnly)}
              className={`flex items-center gap-2 px-3 py-2.5 rounded-xl text-sm font-medium border ${publicOnly ? "bg-primary text-primary-foreground border-primary" : "bg-transparent text-muted-foreground border-border hover:bg-secondary/50"}`}
            >
              <Clock className="w-4 h-4" />
              {publicOnly ? "Somente público" : "Incluir VIP"}
            </button>
            <button
              disabled={!query}
              className="px-6 py-2.5 rounded-xl font-semibold bg-primary text-primary-foreground hover:bg-primary/90 disabled:opacity-50 disabled:pointer-events-none"
            >
              Buscar
            </button>
          </div>
        </div>

        {query && isLoading && (
          <div className="flex items-center justify-center py-20">
            <div className="flex flex-col items-center gap-3">
              <div className="w-8 h-8 rounded-full border-2 border-primary border-t-transparent animate-spin"></div>
              <p className="text-sm text-muted-foreground">Buscando títulos…</p>
            </div>
          </div>
        )}

        {query && error && (
          <div className="bg-destructive/10 border border-destructive/20 rounded-xl p-6 text-destructive text-sm">
            Erro ao buscar títulos. Tente novamente.
          </div>
        )}

        {query && !isLoading && !error && (
          <>
            <div className="flex items-center justify-between mb-4">
              <h2 className="text-sm font-bold text-slate-500 uppercase tracking-wider">
                {total} resultado{total !== 1 ? "s" : ""} para "{query}"
              </h2>
              <Link
                href={`/search?q=${encodeURIComponent(query)}&public_only=${publicOnly}`}
                className="text-xs text-muted-foreground hover:text-foreground"
              >
                Ajustar filtros
              </Link>
            </div>

            {results.length === 0 ? (
              <div className="bg-card border border-border rounded-xl p-12 text-center">
                <div className="w-16 h-16 rounded-full bg-muted/50 flex items-center justify-center mx-auto mb-4">
                  <Search className="w-8 h-8 text-muted-foreground/50" />
                </div>
                <p className="text-sm text-muted-foreground">Nenhum título encontrado para "{query}".</p>
              </div>
            ) : (
              <div className="grid grid-cols-2 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 gap-4">
                {results.map((s) => (
                  <div key={s.id} className="group flex-shrink-0 w-full">
                    <Link
                      className="block"
                      href={`/manga/${s.slug}`}
                    >
                      <div className="relative w-full aspect-[3/4] rounded-xl overflow-hidden shadow-lg shadow-black/30 ring-1 ring-white/5 group-hover:ring-primary/30 transition-all duration-300 group-hover:shadow-primary/15">
                        <div className="absolute inset-0 bg-gradient-to-t from-black/70 via-transparent to-transparent" />
                        <span className="absolute top-2 right-2 px-2 py-0.5 text-[10px] font-bold text-white bg-primary/80 backdrop-blur-sm rounded-md">
                          {s.access === "vip_only" ? "VIP" : "Público"}
                        </span>
                        <div className="absolute inset-0">
                          <div className="w-full h-full bg-secondary flex items-center justify-center">
                            <BookOpen className="w-12 h-12 text-muted-foreground/50" />
                          </div>
                        </div>
                      </div>
                      <h3 className="mt-3 text-sm font-semibold text-foreground leading-tight line-clamp-2 group-hover:text-primary transition-colors">
                        {s.title}
                      </h3>
                      <p className="mt-1 text-xs text-muted-foreground">
                        {s.access === "vip_only" ? "VIP Only" : "Público"}
                      </p>
                    </Link>
                  </div>
                ))}
              </div>
            )}
          </>
        )}

        {!query && (
          <div className="bg-card border border-border rounded-xl p-12 text-center">
            <div className="w-16 h-16 rounded-full bg-muted/50 flex items-center justify-center mx-auto mb-4">
              <Search className="w-8 h-8 text-muted-foreground/50" />
            </div>
            <h3 className="text-lg font-semibold text-foreground mb-2">Pesquise títulos</h3>
            <p className="text-sm text-muted-foreground">
              Digite o nome de um mangá, manhwa ou manhua na busca acima.
            </p>
          </div>
        )}
      </div>
    </section>
  );
}
