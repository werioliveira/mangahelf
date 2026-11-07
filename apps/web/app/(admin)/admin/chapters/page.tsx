"use client";

import { useEffect, useMemo, useState } from "react";
import {
  API_URL, Badge, Btn, Empty, Field, Icon, Modal, PageHeader, Panel, SearchBox, Segmented, Spinner, Th,
  errMsg, fmtDate, inputCls, useToast, type Tone,
} from "../_components/ui";

type Chapter = {
  id: string;
  series_id: string;
  series_title?: string;
  number: number;
  title: string;
  state: string;
  access: string;
  created_at: string;
};

const STATE: Record<string, { label: string; tone: Tone }> = {
  published: { label: "Publicado", tone: "green" },
  processing: { label: "Processando", tone: "amber" },
  draft: { label: "Rascunho", tone: "gray" },
  removed: { label: "Removido", tone: "red" },
};

// A API pode devolver "vip" ou "vip_only"; tratamos os dois como VIP.
const normAccess = (a: string) => (a === "vip" ? "vip_only" : a);
const ACCESS: Record<string, { label: string; tone: Tone }> = {
  public: { label: "Público", tone: "sky" },
  early: { label: "Antecipado", tone: "amber" },
  vip_only: { label: "VIP", tone: "violet" },
};

export default function ChaptersPage() {
  const [chapters, setChapters] = useState<Chapter[]>([]);
  const [seriesMap, setSeriesMap] = useState<Map<string, string>>(new Map());
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [query, setQuery] = useState("");
  const [stateFilter, setStateFilter] = useState("all");
  const [seriesFilter, setSeriesFilter] = useState("all");

  const [editing, setEditing] = useState<Chapter | null>(null);
  const [editState, setEditState] = useState("draft");
  const [editAccess, setEditAccess] = useState("public");
  const [saving, setSaving] = useState(false);
  const [toDelete, setToDelete] = useState<Chapter | null>(null);
  const [deleting, setDeleting] = useState(false);
  const toast = useToast();

  useEffect(() => {
    const sid = new URLSearchParams(window.location.search).get("series");
    if (sid) setSeriesFilter(sid);

    const loadChapters = async () => {
      try {
        const res = await fetch(`${API_URL}/admin/chapters`, { credentials: "include" });
        if (!res.ok) throw new Error((await res.text()) || `Erro ${res.status}`);
        const data = (await res.json()) as Chapter[];
        setChapters(data ?? []);

        // Buscar títulos das séries para exibir na tabela
        const seriesRes = await fetch(`${API_URL}/admin/series`, { credentials: "include" });
        if (seriesRes.ok) {
          const seriesData = (await seriesRes.json()) as { id: string; title: string }[];
          const map = new Map<string, string>();
          (seriesData ?? []).forEach((s) => map.set(s.id, s.title));
          setSeriesMap(map);
        }
      } catch (e) {
        setError(errMsg(e, "Erro ao carregar capítulos"));
      } finally {
        setLoading(false);
      }
    };

    loadChapters();
  }, []);

  const seriesOptions = useMemo(() => {
    return [...seriesMap.entries()].sort((a, b) => a[1].localeCompare(b[1]));
  }, [seriesMap]);

  const visible = useMemo(() => {
    const q = query.trim().toLowerCase();
    return chapters.filter(
      (c) =>
        (stateFilter === "all" || c.state === stateFilter) &&
        (seriesFilter === "all" || c.series_id === seriesFilter) &&
        (!q || c.title.toLowerCase().includes(q) || String(c.number).includes(q) || (seriesMap.get(c.series_id) ?? "").toLowerCase().includes(q)),
    );
  }, [chapters, query, stateFilter, seriesFilter, seriesMap]);

  const count = (s: string) => chapters.filter((c) => c.state === s).length;

  const openEdit = (c: Chapter) => {
    setEditing(c);
    setEditState(c.state);
    setEditAccess(normAccess(c.access));
  };

  const save = async () => {
    if (!editing) return;
    setSaving(true);
    try {
      const res = await fetch(`${API_URL}/admin/chapters/${editing.id}`, {
        method: "PATCH",
        credentials: "include",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({ state: editState, access: editAccess }),
      });
      if (!res.ok) throw new Error((await res.text()) || "Erro ao salvar");
      setChapters((p) => p.map((c) => (c.id === editing.id ? { ...c, state: editState, access: editAccess } : c)));
      toast.show("Capítulo atualizado.");
      setEditing(null);
    } catch (e) {
      toast.show(errMsg(e, "Erro ao editar capítulo"), false);
    } finally {
      setSaving(false);
    }
  };

  const remove = async () => {
    if (!toDelete) return;
    setDeleting(true);
    try {
      const res = await fetch(`${API_URL}/admin/chapters/${toDelete.id}`, {
        method: "DELETE",
        credentials: "include",
      });
      if (!res.ok) throw new Error((await res.text()) || "Erro ao excluir");
      setChapters((p) => p.filter((c) => c.id !== toDelete.id));
      toast.show("Capítulo excluído.");
      setToDelete(null);
    } catch (e) {
      toast.show(errMsg(e, "Erro ao excluir capítulo"), false);
    } finally {
      setDeleting(false);
    }
  };

  return (
    <div className="mx-auto max-w-7xl space-y-6">
      <PageHeader title="Capítulos" subtitle={`${chapters.length} capítulos em ${seriesOptions.length} séries. Controle o estado de publicação e o acesso.`} />

      {error && (
        <div className="rounded-xl border border-red-400/20 bg-red-400/10 px-4 py-3 text-sm text-red-300">{error}</div>
      )}

      <div className="flex flex-wrap items-center gap-3">
        <SearchBox value={query} onChange={setQuery} placeholder="Buscar capítulo ou série" />
        <select
          value={seriesFilter}
          onChange={(e) => setSeriesFilter(e.target.value)}
          aria-label="Filtrar por série"
          className={`${inputCls} !w-auto !rounded-xl !bg-[#111830]`}
        >
          <option value="all">Todas as séries</option>
          {seriesOptions.map(([id, name]) => (
            <option key={id} value={id}>{name}</option>
          ))}
        </select>
        <Segmented
          value={stateFilter}
          onChange={setStateFilter}
          options={[
            { value: "all", label: "Todos", count: chapters.length },
            { value: "published", label: "Publicados", count: count("published") },
            { value: "processing", label: "Processando", count: count("processing") },
            { value: "draft", label: "Rascunhos", count: count("draft") },
          ]}
        />
      </div>

      <Panel>
        {visible.length === 0 ? (
          <Empty
            icon="layers"
            title={chapters.length === 0 ? "Nenhum capítulo ainda" : "Nenhum capítulo encontrado"}
            text={chapters.length === 0 ? "Os capítulos enviados para as séries aparecem aqui." : "Ajuste a busca ou os filtros."}
          />
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead className="border-b border-white/5 bg-white/[0.02]">
                <tr>
                  <Th>Série</Th>
                  <Th>Capítulo</Th>
                  <Th>Estado</Th>
                  <Th>Acesso</Th>
                  <Th>Criado em</Th>
                  <Th right>Ações</Th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/5">
                {visible.map((c) => {
                  const st = STATE[c.state] ?? { label: c.state, tone: "gray" as Tone };
                  const ac = ACCESS[normAccess(c.access)] ?? { label: c.access, tone: "gray" as Tone };
                  return (
                    <tr key={c.id} className="transition-colors hover:bg-white/[0.03]">
                      <td className="px-5 py-3.5 font-medium text-slate-200">{seriesMap.get(c.series_id) || c.series_id}</td>
                      <td className="px-5 py-3.5">
                        <span className="mr-2 font-mono text-xs text-indigo-300">#{c.number}</span>
                        <span className="text-slate-300">{c.title}</span>
                      </td>
                      <td className="px-5 py-3.5"><Badge tone={st.tone}>{st.label}</Badge></td>
                      <td className="px-5 py-3.5"><Badge tone={ac.tone}>{ac.label}</Badge></td>
                      <td className="px-5 py-3.5 text-slate-500">{fmtDate(c.created_at)}</td>
                       <td className="px-5 py-3.5">
                         <div className="flex justify-end gap-1">
                           <button onClick={() => openEdit(c)} aria-label={`Editar capítulo ${c.number}`} className="rounded-lg p-2 text-slate-400 hover:bg-white/5 hover:text-white">
                             <Icon name="edit" />
                           </button>
                           <button onClick={() => setToDelete(c)} aria-label={`Excluir capítulo ${c.number}`} className="rounded-lg p-2 text-slate-400 hover:bg-red-500/10 hover:text-red-300">
                             <Icon name="trash" />
                           </button>
                         </div>
                       </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        )}
      </Panel>

      {editing && (
        <Modal
          title="Editar capítulo"
          onClose={() => setEditing(null)}
          footer={
            <>
              <Btn variant="ghost" onClick={() => setEditing(null)}>Cancelar</Btn>
              <Btn onClick={save} disabled={saving}>{saving ? "Salvando…" : "Salvar alterações"}</Btn>
            </>
          }
        >
          <div className="rounded-lg bg-white/[0.03] px-4 py-3">
            <p className="text-xs text-slate-500">{editing.series_title || editing.series_id}</p>
            <p className="mt-0.5 text-sm font-medium text-slate-100">#{editing.number} · {editing.title}</p>
          </div>
          <Field label="Estado">
            <select value={editState} onChange={(e) => setEditState(e.target.value)} className={inputCls}>
              <option value="draft">Rascunho</option>
              <option value="processing">Processando</option>
              <option value="published">Publicado</option>
              <option value="removed">Removido</option>
            </select>
          </Field>
          <Field label="Acesso">
            <select value={editAccess} onChange={(e) => setEditAccess(e.target.value)} className={inputCls}>
              <option value="public">Público</option>
              <option value="early">Acesso antecipado</option>
              <option value="vip_only">Somente VIP</option>
            </select>
          </Field>
        </Modal>
      )}

      {toDelete && (
        <Modal
          title="Excluir capítulo"
          onClose={() => setToDelete(null)}
          footer={
            <>
              <Btn variant="ghost" onClick={() => setToDelete(null)}>Cancelar</Btn>
              <Btn variant="danger" onClick={remove} disabled={deleting}>{deleting ? "Excluindo…" : "Excluir"}</Btn>
            </>
          }
        >
          <p className="text-sm text-slate-300">
            Tem certeza que deseja excluir <strong className="text-white">#{toDelete.number} · {toDelete.title}</strong>?
          </p>
        </Modal>
      )}

      {toast.node}
    </div>
  );
}