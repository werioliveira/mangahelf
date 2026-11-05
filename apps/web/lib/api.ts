"use client";

import useSWR from "swr";

export const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export type Chapter = {
  id: string;
  series_id: string;
  number: number;
  title: string;
  state: string;
  access: string;
};

export type Page = {
  number: number;
  url: string;
  width: number;
  height: number;
  storage_key: string;
};

export type ChapterPagesResponse = Page[];

// ---------------------------------------------------------------------------
// Fetch cru (Server Components) — sem deduplicação automática
// ---------------------------------------------------------------------------

async function fetchJson<T>(path: string): Promise<T> {
  const res = await fetch(`${API_URL}${path}`, { cache: "no-store" });
  if (!res.ok) {
    throw new Error(`Falha ao buscar ${path}: ${res.status}`);
  }
  return res.json();
}

export async function lookupChapter(slug: string, number: number): Promise<Chapter | null> {
  const data = await fetchJson<Chapter | null>(
    `/chapters/lookup?series_slug=${encodeURIComponent(slug)}&number=${number}`,
  );
  if (!resOk(data)) return null;
  return data as Chapter;
}

function resOk(data: any): data is Chapter {
  return data != null && typeof data.id === "string";
}

export async function getChapterPages(chapterId: string, variant = "full"): Promise<ChapterPagesResponse> {
  return fetchJson<ChapterPagesResponse>(
    `/chapters/${chapterId}/pages?variant=${encodeURIComponent(variant)}`,
  );
}

export async function lookupChapterPages(
  slug: string,
  number: number,
  variant = "full",
): Promise<{ chapter: Chapter; pages: ChapterPagesResponse } | null> {
  const chapter = await lookupChapter(slug, number);
  if (!chapter) return null;
  const pages = await getChapterPages(chapter.id, variant);
  return { chapter, pages };
}

// ---------------------------------------------------------------------------
// SWR hooks (Client Components) — deduplicação automática
// ---------------------------------------------------------------------------

export function useChapterLookup(slug: string | null, number: number | null) {
  return useSWR<Chapter | null>(
    slug && number ? `/chapters/lookup?series_slug=${encodeURIComponent(slug)}&number=${number}` : null,
    async (url: string) => {
      const res = await fetch(`${API_URL}${url}`, { cache: "no-store" });
      if (!res.ok) return null;
      return res.json();
    },
    { revalidateOnFocus: false, dedupingInterval: 60_000 },
  );
}

export function useChapterPages(chapterId: string | null, variant = "full") {
  return useSWR<ChapterPagesResponse | null>(
    chapterId ? `/chapters/${chapterId}/pages?variant=${encodeURIComponent(variant)}` : null,
    async (url: string) => {
      const res = await fetch(`${API_URL}${url}`, { cache: "no-store" });
      if (!res.ok) return [];
      return res.json();
    },
    { revalidateOnFocus: false, dedupingInterval: 60_000 },
  );
}

export function useChapterLookupAndPages(slug: string | null, number: number | null, variant = "full") {
  const { data: chapter, error: chapterError, isLoading: chapterLoading } = useChapterLookup(
    slug,
    number,
  );
  const { data: pages, error: pagesError, isLoading: pagesLoading } = useChapterPages(
    chapter?.id ?? null,
    variant,
  );

  return {
    chapter,
    pages,
    error: chapterError ?? pagesError,
    isLoading: chapterLoading || pagesLoading,
    isReady: !!chapter && (!!pages || chapter == null),
  };
}
