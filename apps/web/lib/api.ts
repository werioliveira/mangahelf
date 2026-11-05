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

export async function lookupChapter(slug: string, number: number): Promise<Chapter | null> {
  const res = await fetch(`${API_URL}/chapters/lookup?series_slug=${encodeURIComponent(slug)}&number=${number}`, {
    next: { tags: [`chapter:${slug}:${number}`] },
  });
  if (!res.ok) {
    if (res.status === 404) return null;
    throw new Error(`Falha ao buscar capítulo: ${res.status}`);
  }
  return res.json();
}

export async function getChapterPages(chapterId: string, variant = "full"): Promise<ChapterPagesResponse> {
  const res = await fetch(`${API_URL}/chapters/${chapterId}/pages?variant=${encodeURIComponent(variant)}`, {
    next: { tags: [`chapter:${chapterId}`] },
  });
  if (!res.ok) {
    throw new Error(`Falha ao buscar páginas: ${res.status}`);
  }
  return res.json();
}

export async function lookupChapterPages(slug: string, number: number, variant = "full"): Promise<{ chapter: Chapter; pages: ChapterPagesResponse } | null> {
  const chapter = await lookupChapter(slug, number);
  if (!chapter) return null;
  const pages = await getChapterPages(chapter.id, variant);
  return { chapter, pages };
}
