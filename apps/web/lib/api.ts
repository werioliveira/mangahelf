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

export type SearchSeriesResult = {
  id: string;
  title: string;
  slug: string;
  synopsis: string;
  access: string;
  group_id: string | null;
  rank: number;
  chapters: number;
};

export type SearchSeriesResponse = {
  query: string;
  limit: number;
  offset: number;
  total: number;
  results: SearchSeriesResult[];
};

export type Setting = {
  key: string;
  value: string;
  type: string;
  setting_group: string;
  updated_at: string;
};

export type ThemeSettings = {
  primary: string;
  secondary: string;
  accent: string;
};

export type SettingsResponse = {
  settings: Setting[];
  total: number;
};

// ---------------------------------------------------------------------------
// Fetch cru (Server Components) — sem deduplicação automática
// ---------------------------------------------------------------------------

export async function searchSeries(
  query: string,
  limit = 20,
  offset = 0,
  publicOnly = false,
): Promise<SearchSeriesResponse> {
  const params = new URLSearchParams({ q: query, limit: String(limit), offset: String(offset) });
  if (publicOnly) params.set("public_only", "true");
  return fetchJson<SearchSeriesResponse>(`/search?${params}`);
}

export async function fetchSettings(group?: string): Promise<SettingsResponse> {
  const params = new URLSearchParams();
  if (group) params.set("group", group);
  return fetchJson<SettingsResponse>(`/admin/settings?${params}`);
}

export async function updateSetting(
  key: string,
  value: unknown,
  type?: string,
  settingGroup?: string,
): Promise<void> {
  const body: Record<string, unknown> = { value };
  if (type) body.type = type;
  if (settingGroup) body.setting_group = settingGroup;
  const res = await fetch(`${API_URL}/admin/settings/${key}`, {
    method: "PATCH",
    headers: { "Content-Type": "application/json" }, credentials: "include",
    body: JSON.stringify(body),
  });
  if (!res.ok) {
    const text = await res.text();
    throw new Error(`Falha ao atualizar ${key}: ${res.status} ${text}`);
  }
}

// ---------------------------------------------------------------------------
// SWR hooks (Client Components) — deduplicação automática
// ---------------------------------------------------------------------------

export function useSearchSeries(query: string, limit = 20, publicOnly = false) {
  return useSWR<SearchSeriesResponse | null>(
    query ? `/search?q=${encodeURIComponent(query)}&limit=${limit}&public_only=${publicOnly}` : null,
    async (url: string) => {
      const res = await fetch(`${API_URL}${url}`, { cache: "no-store", credentials: "include" });
      if (!res.ok) return { query, limit, offset: 0, total: 0, results: [] };
      return res.json();
    },
    { revalidateOnFocus: false, dedupingInterval: 60_000 },
  );
}

export function useSettings(group?: string) {
  return useSWR<SettingsResponse | null>(
    group ? `/admin/settings?group=${encodeURIComponent(group)}` : "/admin/settings",
    async (url: string) => {
      const res = await fetch(`${API_URL}${url}`, { cache: "no-store", credentials: "include" });
      if (!res.ok) return { settings: [], total: 0 };
      return res.json();
    },
    { revalidateOnFocus: false, dedupingInterval: 60_000 },
  );
}

export function useThemeSettings() {
  return useSWR<ThemeSettings | null>(
    "/admin/settings?group=ui",
    async (url: string) => {
      const res = await fetch(`${API_URL}${url}`, { cache: "no-store", credentials: "include" });
      if (!res.ok) return null;
      const data = await res.json();
      const settings = data.settings ?? [];
      const get = (key: string) => settings.find((s: Setting) => s.key === key)?.value;
      return {
        primary: get("ui_primary_color") ?? "#3b82f6",
        secondary: get("ui_secondary_color") ?? "#1e293b",
        accent: get("ui_accent_color") ?? "#f59e0b",
      };
    },
    { revalidateOnFocus: false, dedupingInterval: 60_000 },
  );
}

async function fetchJson<T>(path: string): Promise<T> {
  const res = await fetch(`${API_URL}${path}`, { cache: "no-store", credentials: "include" });
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
      const res = await fetch(`${API_URL}${url}`, { cache: "no-store", credentials: "include" });
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
      const res = await fetch(`${API_URL}${url}`, { cache: "no-store", credentials: "include" });
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
