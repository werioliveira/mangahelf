"use client";

import { useEffect, useMemo, useRef, useState } from "react";
import { API_URL, Badge, Btn, Field, Icon, Modal, errMsg, inputCls, useToast, type Tone } from "./ui";
import { CoverUploader } from "./CoverUploader";
import { TagsInput } from "./TagsInput";

// ============================================================================
// Tipos
// ============================================================================

export type Author = {
  id: string;
  name: string;
  role: "writer" | "artist" | "writer_artist";
};

export type Series = {
  id: string;
  title: string;
  slug: string;
  synopsis: string;
  access: "public" | "vip_only";
  type: "manga" | "manhwa" | "manhua";
  status: "ongoing" | "hiatus" | "completed" | "cancelled";
  demography: "shounen" | "shoujo" | "seinen" | "josei" | "unspecified";
  original_language: "ja" | "ko" | "zh" | "en" | "other";
  release_year: number | null;
  cover_url: string | null;
  cover_storage_key: string | null;
  alt_titles: string[];
  authors: Author[];
  tags: string[];
  group_id: string | null;
  public_at: string | null;
  chapter_count: number;
  created_at: string;
};

export type SeriesFormValues = Omit<Series, "id" | "chapter_count" | "created_at">;

type Props = {
  mode: "create" | "edit";
  series: Series | null;
  onClose: () => void;
  onSubmit: (values: SeriesFormValues) => Promise<void>;
  busy: boolean;
};

// ============================================================================
// Constantes
// ============================================================================

const TYPE_OPTIONS = [
  { value: "manga", label: "Manga" },
  { value: "manhwa", label: "Manhwa" },
  { value: "manhua", label: "Manhua" },
];

const STATUS_OPTIONS: { value: Series["status"]; label: string; tone: Tone }[] = [
  { value: "ongoing", label: "Em curso", tone: "sky" },
  { value: "hiatus", label: "Hiato", tone: "amber" },
  { value: "completed", label: "Completo", tone: "green" },
  { value: "cancelled", label: "Cancelado", tone: "red" },
];

const DEMOGRAPHY_OPTIONS = [
  { value: "unspecified", label: "— não especificada —" },
  { value: "shounen", label: "Shounen" },
  { value: "shoujo", label: "Shoujo" },
  { value: "seinen", label: "Seinen" },
  { value: "josei", label: "Josei" },
];

const LANGUAGE_OPTIONS = [
  { value: "ja", label: "Japonês" },
  { value: "ko", label: "Coreano" },
  { value: "zh", label: "Chinês" },
  { value: "en", label: "Inglês" },
  { value: "other", label: "Outro" },
];

const AUTHOR_ROLE_OPTIONS = [
  { value: "writer", label: "Roteiro" },
  { value: "artist", label: "Arte" },
  { value: "writer_artist", label: "Roteiro + Arte" },
];

const EMPTY: SeriesFormValues = {
  title: "",
  slug: "",
  synopsis: "",
  access: "public",
  type: "manga",
  status: "ongoing",
  demography: "unspecified",
  original_language: "ja",
  release_year: null,
  cover_url: null,
  cover_storage_key: null,
  alt_titles: [],
  authors: [],
  tags: [],
  group_id: null,
  public_at: null,
};

const slugify = (s: string) =>
  s
    .normalize("NFD")
    .replace(/[\u0300-\u036f]/g, "")
    .toLowerCase()
    .replace(/[^a-z0-9]+/g, "-")
    .replace(/^-+|-+$/g, "");

// ============================================================================
// Componente principal
// ============================================================================

export function SeriesForm({ mode, series, onClose, onSubmit, busy }: Props) {
  const toast = useToast();
  const [form, setForm] = useState<SeriesFormValues>(EMPTY);
  const [errors, setErrors] = useState<Record<string, string>>({});
  const [slugTouched, setSlugTouched] = useState(false);
  const [activeSection, setActiveSection] = useState<string>("identification");
  const titleRef = useRef<HTMLInputElement>(null);

  // Carrega dados ao editar
  useEffect(() => {
    if (mode === "edit" && series) {
      const { id, chapter_count, created_at, ...rest } = series;
      setForm({
        ...rest,
        alt_titles: rest.alt_titles ?? [],
        authors: rest.authors ?? [],
        tags: rest.tags ?? [],
      });
      setSlugTouched(true);
    } else {
      setForm(EMPTY);
      setSlugTouched(false);
    }
    // Foco no primeiro campo ao abrir
    setTimeout(() => titleRef.current?.focus(), 50);
  }, [mode, series]);

  // Helpers
  const set = <K extends keyof SeriesFormValues>(key: K, value: SeriesFormValues[K]) => {
    setForm((f) => ({ ...f, [key]: value }));
    if (errors[key]) setErrors((e) => ({ ...e, [key]: "" }));
  };

  const addAltTitle = () => set("alt_titles", [...form.alt_titles, ""]);
  const updateAltTitle = (i: number, v: string) =>
    set("alt_titles", form.alt_titles?.map((t, idx) => (idx === i ? v : t)) ?? []);
  const removeAltTitle = (i: number) => set("alt_titles", form.alt_titles.filter((_, idx) => idx !== i));

  const addAuthor = () =>
    set("authors", [...form.authors, { id: crypto.randomUUID(), name: "", role: "writer_artist" }]);
  const updateAuthor = (i: number, patch: Partial<Author>) =>
    set(
      "authors",
      form.authors?.map((a, idx) => (idx === i ? { ...a, ...patch } : a)) ?? [],
    );
  const removeAuthor = (i: number) => set("authors", form.authors.filter((_, idx) => idx !== i));

  const handleTitleChange = (title: string) => {
    setForm((f) => ({
      ...f,
      title,
      slug: slugTouched ? f.slug : slugify(title),
    }));
    if (errors.title) setErrors((e) => ({ ...e, title: "" }));
  };

  const handleSlugChange = (v: string) => {
    setSlugTouched(true);
    set("slug", slugify(v));
  };

  const handleCoverUploaded = (cover_url: string, cover_storage_key: string) => {
    set("cover_url", cover_url);
    set("cover_storage_key", cover_storage_key);
  };

  const handleCoverRemoved = () => {
    set("cover_url", null);
    set("cover_storage_key", null);
  };

  // Validação
  const validate = (): boolean => {
    const e: Record<string, string> = {};
    if (!form.title.trim()) e.title = "Título é obrigatório";
    if (!form.slug.trim()) e.slug = "Slug é obrigatório";
    if (form.release_year !== null && (form.release_year < 1900 || form.release_year > new Date().getFullYear() + 1))
      e.release_year = "Ano inválido";
    if (form.public_at && isNaN(Date.parse(form.public_at))) e.public_at = "Data inválida";
    // Autores: nome não pode ser vazio se presente
    const badAuthor = form.authors.find((a) => !a.name.trim());
    if (badAuthor) e.authors = "Todo autor precisa de um nome";

    setErrors(e);
    return Object.keys(e).length === 0;
  };

  const submit = async () => {
    if (!validate()) {
      toast.show("Verifique os campos destacados.", false);
      // Foca primeiro campo com erro
      const firstError = Object.keys(errors)[0];
      if (firstError) {
        const el = document.querySelector(`[data-field="${firstError}"]`) as HTMLElement | null;
        el?.focus();
      }
      return;
    }

    // Limpa alt_titles vazias antes de enviar
    const cleanValues: SeriesFormValues = {
      ...form,
      alt_titles: form.alt_titles?.map((t) => t.trim()).filter(Boolean),
        authors: form.authors?.map((a) => ({ ...a, name: a.name.trim() })) ?? [],
      synopsis: form.synopsis.trim(),
      title: form.title.trim(),
    };

    try {
      await onSubmit(cleanValues);
    } catch {
      // toast já tratado pelo pai
    }
  };

  // Teclado: Cmd/Ctrl+Enter envia
  const onKeyDown = (e: React.KeyboardEvent) => {
    if ((e.metaKey || e.ctrlKey) && e.key === "Enter" && !busy) {
      e.preventDefault();
      submit();
    }
    if (e.key === "Escape" && !busy) {
      e.preventDefault();
      onClose();
    }
  };

  // Sections para navegação por teclado e âncoras
  const sections = [
    { id: "identification", label: "Identificação", icon: "book" },
    { id: "classification", label: "Classificação", icon: "tag" },
    { id: "people", label: "Autores e tags", icon: "users" },
    { id: "cover", label: "Capa", icon: "image" },
    { id: "access", label: "Acesso e publicação", icon: "lock" },
  ] as const;

  return (
    <Modal
      title={mode === "edit" ? "Editar série" : "Nova série"}
      onClose={onClose}
      footer={
        <>
          <Btn variant="ghost" onClick={onClose} disabled={busy}>
            Cancelar
          </Btn>
          <Btn onClick={submit} disabled={busy}>
            {busy ? "Salvando…" : mode === "edit" ? "Salvar alterações" : "Criar série"}
          </Btn>
        </>
      }
    >
      <div className="flex flex-col gap-6 md:flex-row" onKeyDown={onKeyDown}>
        {/* Sidebar de seções (desktop) */}
        <nav
          aria-label="Seções do formulário"
          className="hidden shrink-0 md:block md:w-48 md:border-r md:border-white/5 md:pr-4"
        >
          <ul className="space-y-1">
            {sections.map((s) => (
              <li key={s.id}>
                <button
                  type="button"
                  onClick={() => {
                    setActiveSection(s.id);
                    document.getElementById(`section-${s.id}`)?.scrollIntoView({ behavior: "smooth", block: "start" });
                  }}
                  aria-current={activeSection === s.id}
                  className={`flex w-full items-center gap-2.5 rounded-lg px-3 py-2 text-left text-[13px] transition-colors ${
                    activeSection === s.id
                      ? "bg-indigo-500/15 font-medium text-indigo-300"
                      : "text-slate-400 hover:bg-white/5 hover:text-white"
                  }`}
                >
                  <Icon name={s.icon} className="h-4 w-4" />
                  {s.label}
                </button>
              </li>
            ))}
          </ul>
        </nav>

        {/* Form */}
        <div className="min-w-0 flex-1 space-y-8">
          {/* ============ Identificação ============ */}
          <FormSection id="identification" title="Identificação" icon="book">
            <Field label="Título">
              <input
                ref={titleRef}
                data-field="title"
                value={form.title}
                placeholder="Ex.: Solo Leveling"
                className={inputCls}
                aria-invalid={!!errors.title}
                aria-describedby={errors.title ? "err-title" : undefined}
                onChange={(e) => handleTitleChange(e.target.value)}
              />
              {errors.title && (
                <p id="err-title" className="mt-1 text-xs text-red-400">
                  {errors.title}
                </p>
              )}
            </Field>

            <Field label="Slug" hint="Endereço da série no site. Gerado a partir do título.">
              <input
                data-field="slug"
                value={form.slug}
                className={`${inputCls} font-mono`}
                aria-invalid={!!errors.slug}
                onChange={(e) => handleSlugChange(e.target.value)}
              />
              {form.slug && (
                <p className="mt-1 font-mono text-[11px] text-slate-500">
                  URL: <span className="text-slate-400">/series/{form.slug}</span>
                </p>
              )}
            </Field>

            <Field label="Títulos alternativos" hint="Ex.: romaji, título em inglês, título oficial em PT-BR.">
              <div className="space-y-2">
                {form.alt_titles?.map((t, i) => (
                  <div key={i} className="flex items-center gap-2">
                    <input
                      value={t}
                      placeholder={`Título alternativo ${i + 1}`}
                      className={inputCls}
                      onChange={(e) => updateAltTitle(i, e.target.value)}
                    />
                    <button
                      type="button"
                      onClick={() => removeAltTitle(i)}
                      aria-label={`Remover título alternativo ${i + 1}`}
                      className="shrink-0 rounded-lg p-2 text-slate-400 hover:bg-red-500/10 hover:text-red-300"
                    >
                      <Icon name="trash" className="h-4 w-4" />
                    </button>
                  </div>
                ))}
                <button
                  type="button"
                  onClick={addAltTitle}
                  className="inline-flex items-center gap-1.5 text-xs font-medium text-indigo-300 hover:text-indigo-200"
                >
                  <Icon name="plus" className="h-3.5 w-3.5" /> Adicionar título alternativo
                </button>
              </div>
            </Field>

            <Field label="Sinopse" hint="Markdown suportado. Evite spoilers.">
              <textarea
                rows={5}
                value={form.synopsis}
                placeholder="Resumo exibido na página da série"
                className={`${inputCls} resize-y`}
                onChange={(e) => set("synopsis", e.target.value)}
              />
            </Field>
          </FormSection>

          {/* ============ Classificação ============ */}
          <FormSection id="classification" title="Classificação" icon="tag">
            <div className="grid grid-cols-1 gap-4 md:grid-cols-2">
              <Field label="Tipo">
                <select
                  value={form.type}
                  className={inputCls}
                  onChange={(e) => set("type", e.target.value as Series["type"])}
                >
                  {TYPE_OPTIONS.map((o) => (
                    <option key={o.value} value={o.value}>
                      {o.label}
                    </option>
                  ))}
                </select>
              </Field>

              <Field label="Status">
                <div className="flex flex-wrap gap-2">
                  {STATUS_OPTIONS.map((o) => (
                    <button
                      key={o.value}
                      type="button"
                      onClick={() => set("status", o.value)}
                      aria-pressed={form.status === o.value}
                      className={`rounded-lg border px-3 py-1.5 text-xs font-medium transition-colors ${
                        form.status === o.value
                          ? "border-indigo-400/50 bg-indigo-500/15 text-indigo-300"
                          : "border-white/10 bg-white/5 text-slate-400 hover:bg-white/10 hover:text-white"
                      }`}
                    >
                      <Badge tone={o.tone}>{o.label}</Badge>
                    </button>
                  ))}
                </div>
              </Field>

              <Field label="Demografia">
                <select
                  value={form.demography}
                  className={inputCls}
                  onChange={(e) => set("demography", e.target.value as Series["demography"])}
                >
                  {DEMOGRAPHY_OPTIONS.map((o) => (
                    <option key={o.value} value={o.value}>
                      {o.label}
                    </option>
                  ))}
                </select>
              </Field>

              <Field label="Idioma original">
                <select
                  value={form.original_language}
                  className={inputCls}
                  onChange={(e) => set("original_language", e.target.value as Series["original_language"])}
                >
                  {LANGUAGE_OPTIONS.map((o) => (
                    <option key={o.value} value={o.value}>
                      {o.label}
                    </option>
                  ))}
                </select>
              </Field>

              <Field label="Ano de lançamento">
                <input
                  type="number"
                  min={1900}
                  max={new Date().getFullYear() + 1}
                  value={form.release_year ?? ""}
                  placeholder="Ex.: 2018"
                  className={inputCls}
                  onChange={(e) =>
                    set("release_year", e.target.value ? parseInt(e.target.value, 10) : null)
                  }
                />
              </Field>
            </div>
          </FormSection>

          {/* ============ Autores e tags ============ */}
          <FormSection id="people" title="Autores e tags" icon="users">
            <Field label="Autores">
              <div className="space-y-2">
                {form.authors?.map((a, i) => (
                  <div key={a.id} className="flex flex-wrap items-center gap-2">
                    <input
                      value={a.name}
                      placeholder="Nome do autor(a)"
                      className={`${inputCls} min-w-[180px] flex-1`}
                      onChange={(e) => updateAuthor(i, { name: e.target.value })}
                    />
                    <select
                      value={a.role}
                      className={`${inputCls} w-40`}
                      onChange={(e) => updateAuthor(i, { role: e.target.value as Author["role"] })}
                    >
                      {AUTHOR_ROLE_OPTIONS.map((o) => (
                        <option key={o.value} value={o.value}>
                          {o.label}
                        </option>
                      ))}
                    </select>
                    <button
                      type="button"
                      onClick={() => removeAuthor(i)}
                      aria-label={`Remover ${a.name || "autor"}`}
                      className="shrink-0 rounded-lg p-2 text-slate-400 hover:bg-red-500/10 hover:text-red-300"
                    >
                      <Icon name="trash" className="h-4 w-4" />
                    </button>
                  </div>
                ))}
                <button
                  type="button"
                  onClick={addAuthor}
                  className="inline-flex items-center gap-1.5 text-xs font-medium text-indigo-300 hover:text-indigo-200"
                >
                  <Icon name="plus" className="h-3.5 w-3.5" /> Adicionar autor
                </button>
              </div>
            </Field>

            <Field label="Tags" hint="Gêneros, temas, elementos. Use Enter para adicionar.">
              <TagsInput
                values={form.tags}
                onChange={(tags) => set("tags", tags)}
                suggestionsPath="/admin/tags?q="
              />
            </Field>
          </FormSection>

          {/* ============ Capa ============ */}
          <FormSection id="cover" title="Capa" icon="image">
            <Field label="Capa da série" hint="JPG, PNG ou WebP. Idealmente 2:3 (ex.: 600×900). Máx. 5MB.">
              <CoverUploader
                coverUrl={form.cover_url}
                onUploaded={handleCoverUploaded}
                onRemoved={handleCoverRemoved}
              />
            </Field>
          </FormSection>

          {/* ============ Acesso e publicação ============ */}
          <FormSection id="access" title="Acesso e publicação" icon="lock">
            <Field label="Acesso">
              <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
                <AccessOption
                  checked={form.access === "public"}
                  onClick={() => set("access", "public")}
                  tone="green"
                  title="Público"
                  desc="Qualquer visitante pode ler. Capítulos podem ainda assim ser VIP/early individualmente."
                />
                <AccessOption
                  checked={form.access === "vip_only"}
                  onClick={() => set("access", "vip_only")}
                  tone="violet"
                  title="Somente VIP"
                  desc="Apenas assinantes VIP ativos leem esta série. Herdado por todos os capítulos."
                />
              </div>
            </Field>

            <Field
              label="Agendar publicação"
              hint="Se preenchido, a série fica como rascunho até esta data. Vazio = publica imediatamente ao salvar."
            >
              <input
                type="datetime-local"
                value={form.public_at ?? ""}
                className={inputCls}
                onChange={(e) => set("public_at", e.target.value || null)}
              />
            </Field>

            <div className="rounded-lg border border-amber-400/20 bg-amber-400/5 px-4 py-3 text-xs text-amber-200/80">
              <strong className="text-amber-200">Atenção:</strong> a permissão{" "}
              <code className="rounded bg-black/30 px-1 py-0.5">chapter.set_access</code> controla quem pode alterar o
              acesso. Cada grupo editorial controla apenas as próprias séries.
            </div>
          </FormSection>
        </div>
      </div>
    </Modal>
  );
}

// ============================================================================
// Sub-componentes
// ============================================================================

function FormSection({
  id,
  title,
  icon,
  children,
}: {
  id: string;
  title: string;
  icon: string;
  children: React.ReactNode;
}) {
  return (
    <section id={`section-${id}`} className="space-y-4 scroll-mt-4">
      <div className="flex items-center gap-2 border-b border-white/5 pb-2">
        <Icon name={icon} className="h-4 w-4 text-indigo-300" />
        <h3 className="text-sm font-semibold text-white">{title}</h3>
      </div>
      {children}
    </section>
  );
}

function AccessOption({
  checked,
  onClick,
  tone,
  title,
  desc,
}: {
  checked: boolean;
  onClick: () => void;
  tone: Tone;
  title: string;
  desc: string;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      aria-pressed={checked}
      className={`flex items-start gap-3 rounded-lg border p-3 text-left transition-colors ${
        checked ? "border-indigo-400/50 bg-indigo-500/10" : "border-white/10 bg-white/5 hover:bg-white/10"
      }`}
    >
      <div className="mt-0.5 grid h-5 w-5 shrink-0 place-items-center rounded-full border border-white/20">
        {checked && (
          <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="3" className="h-3 w-3 text-indigo-300">
            <path d="M5 13l4 4L19 7" />
          </svg>
        )}
      </div>
      <div className="min-w-0">
        <div className="flex items-center gap-2">
          <Badge tone={tone}>{title}</Badge>
        </div>
        <p className="mt-1 text-xs text-slate-400">{desc}</p>
      </div>
    </button>
  );
}
