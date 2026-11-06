"use client";

import Link from "next/link";
import { useCallback, useEffect, useMemo, useState } from "react";
import {
  API_URL, Badge, Btn, Empty, Icon, Modal, PageHeader, Panel, SearchBox, Segmented, Spinner, Th,
  errMsg, fmtDate, inputCls, useToast, type Tone,
} from "../_components/ui";
import { SeriesForm, type Series, type SeriesFormValues } from "../_components/SeriesForm";

const ACCESS: Record<string, { label: string; tone: Tone }> = {
  public: { label: "Público", tone: "green" },
  vip_only: { label: "VIP", tone: "violet" },
};

const STATUS: Record<string, { label: string; tone: Tone }> = {
  ongoing: { label: "Em curso", tone: "sky" },
  hiatus: { label: "Hiato", tone: "amber" },
  completed: { label: "Completo", tone: "green" },
  cancelled: { label: "Cancelado", tone: "red" },
};

const TYPE: Record<string, { label: string }> = {
  manga: { label: "Manga" },
  manhwa: { label: "Manhwa" },
  manhua: { label: "Manhua" },
};

async function request(path: string, init?: RequestInit) {
  const res = await fetch(`${API_URL}${path}`, { credentials: "include", ...init });
  if (!res.ok) throw new Error((await res.text()) || `Erro ${res.status}`);
  return res;
}

export default function SeriesPage() {
  const [items, setItems] = useState<Series[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [query, setQuery] = useState("");
  const [filterAccess, setFilterAccess] = useState("all");
  const [filterStatus, setFilterStatus] = useState("all");
  const [filterType, setFilterType] = useState("all");

  const [modal, setModal] = useState<{ id: string | null } | null>(null);
  const [toDelete, setToDelete] = useState<Series | null>(null);
  const [busy, setBusy] = useState(false);
  const toast = useToast();

  const load = useCallback(async () => {
    try {
      const res = await request("/admin/series");
      setItems((await res.json()) ?? []);
      setError("");
    } catch (e) {
      setError(errMsg(e, "Erro ao carregar séries"));
    } finally {
      setLoading(false);
    }
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  const openCreate = () => setModal({ id: null });
  const openEdit = (s: Series) => setModal({ id: s.id });

  const save = async (values: SeriesFormValues) => {
    setBusy(true);
    try {
      const headers = { "Content-Type": "application/json" };
      if (modal?.id) {
        await request(`/admin/series/${modal.id}`, { method: "PATCH", headers, body: JSON.stringify(values) });
        setItems((p) => p.map((s) => (s.id === modal.id ? { ...s, ...values } : s)));
        toast.show("Série atualizada.");
      } else {
        const res = await request("/admin/series", { method: "POST", headers, body: JSON.stringify(values) });
        const created = (await res.json()) as Series;
        setItems((p) => [created, ...p]);
        toast.show("Série criada.");
      }
      setModal(null);
    } catch (e) {
      toast.show(errMsg(e, "Erro ao salvar série"), false);
      throw e; // repassa para o form tratar (ex: erro de slug duplicado em campo)
    } finally {
      setBusy(false);
    }
  };

  const remove = async () => {
    if (!toDelete) return;
    setBusy(true);
    try {
      await request(`/admin/series/${toDelete.id}`, { method: "DELETE" });
      setItems((p) => p.filter((s) => s.id !== toDelete.id));
      toast.show("Série excluída.");
      setToDelete(null);
    } catch (e) {
      toast.show(errMsg(e, "Erro ao excluir série"), false);
    } finally {
      setBusy(false);
    }
  };

  const visible = useMemo(() => {
    const q = query.trim().toLowerCase();
    return items.filter((s) => {
      if (filterAccess !== "all" && s.access !== filterAccess) return false;
      if (filterStatus !== "all" && s.status !== filterStatus) return false;
      if (filterType !== "all" && s.type !== filterType) return false;
      if (!q) return true;
      return (
        s.title.toLowerCase().includes(q) ||
        s.slug.includes(q) ||
        (s.alt_titles ?? []).some((t) => t.toLowerCase().includes(q)) ||
        (s.authors ?? []).some((a) => a.name.toLowerCase().includes(q))
      );
    });
  }, [items, query, filterAccess, filterStatus, filterType]);

  const countAccess = (a: string) => items.filter((s) => s.access === a).length;
  const countStatus = (s: string) => items.filter((x) => x.status === s).length;
  const countType = (t: string) => items.filter((s) => s.type === t).length;

  if (loading) return <Spinner label="Carregando séries…" />;

  const newBtn = (
    <Btn onClick={openCreate}>
      <Icon name="plus" /> Nova série
    </Btn>
  );

  return (
    <div className="mx-auto max-w-7xl space-y-6">
      <PageHeader
        title="Séries"
        subtitle={`${items.length} no catálogo. Cadastre, edite e controle o acesso de cada obra.`}
        action={newBtn}
      />

      {error && (
        <div role="alert" className="rounded-xl border border-red-400/20 bg-red-400/10 px-4 py-3 text-sm text-red-300">
          {error}
        </div>
      )}

      <div className="flex flex-wrap items-center gap-3">
        <SearchBox value={query} onChange={setQuery} placeholder="Buscar por título, slug, autor ou título alternativo" />
        <Segmented
          value={filterType}
          onChange={setFilterType}
          options={[
            { value: "all", label: "Todos", count: items.length },
            { value: "manga", label: "Manga", count: countType("manga") },
            { value: "manhwa", label: "Manhwa", count: countType("manhwa") },
            { value: "manhua", label: "Manhua", count: countType("manhua") },
          ]}
        />
        <Segmented
          value={filterAccess}
          onChange={setFilterAccess}
          options={[
            { value: "all", label: "Acesso: todos" },
            { value: "public", label: "Públicos", count: countAccess("public") },
            { value: "vip_only", label: "VIP", count: countAccess("vip_only") },
          ]}
        />
        <Segmented
          value={filterStatus}
          onChange={setFilterStatus}
          options={[
            { value: "all", label: "Status: todos" },
            { value: "ongoing", label: "Em curso", count: countStatus("ongoing") },
            { value: "hiatus", label: "Hiato", count: countStatus("hiatus") },
            { value: "completed", label: "Completo", count: countStatus("completed") },
          ]}
        />
      </div>

      <Panel>
        {visible.length === 0 ? (
          <Empty
            icon="book"
            title={items.length === 0 ? "Nenhuma série cadastrada" : "Nada encontrado"}
            text={
              items.length === 0
                ? "Crie a primeira série para começar a publicar capítulos."
                : "Tente outro termo ou remova o filtro."
            }
            action={items.length === 0 ? newBtn : undefined}
          />
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead className="border-b border-white/5 bg-white/[0.02]">
                <tr>
                  <Th>Série</Th>
                  <Th>Tipo</Th>
                  <Th>Status</Th>
                  <Th>Capítulos</Th>
                  <Th>Acesso</Th>
                  <Th>Criada em</Th>
                  <Th right>Ações</Th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/5">
                {visible.map((s) => {
                  const a = ACCESS[s.access] ?? { label: s.access, tone: "gray" as Tone };
                  const st = STATUS[s.status] ?? { label: s.status, tone: "gray" as Tone };
                  const t = TYPE[s.type] ?? { label: s.type };
                  return (
                    <tr key={s.id} className="group transition-colors hover:bg-white/[0.03]">
                      <td className="max-w-md px-5 py-4">
                        <div className="flex items-center gap-3">
                          <div className="relative h-16 w-11 shrink-0 overflow-hidden rounded-md bg-gradient-to-br from-indigo-500/30 to-sky-400/20">
                            {s.cover_url ? (
                              // eslint-disable-next-line @next/next/no-img-element
                              <img
                                src={s.cover_url}
                                alt={`Capa de ${s.title}`}
                                className="h-full w-full object-cover"
                                loading="lazy"
                              />
                            ) : (
                              <div className="grid h-full place-items-center text-sm font-bold text-indigo-200">
                                {s.title.charAt(0).toUpperCase()}
                              </div>
                            )}
                          </div>
                          <div className="min-w-0">
                            <p className="truncate font-medium text-slate-100">{s.title}</p>
                            <p className="truncate font-mono text-xs text-slate-500">/{s.slug}</p>
                            {s.alt_titles && s.alt_titles.length > 0 && (
                              <p className="mt-0.5 line-clamp-1 text-xs text-slate-500">
                                {s.alt_titles.slice(0, 2).join(" · ")}
                                {s.alt_titles.length > 2 && ` +${s.alt_titles.length - 2}`}
                              </p>
                            )}
                          </div>
                        </div>
                      </td>
                      <td className="px-5 py-4 text-slate-400">{t.label}</td>
                      <td className="px-5 py-4">
                        <Badge tone={st.tone}>{st.label}</Badge>
                      </td>
                      <td className="px-5 py-4">
                        <Link
                          href={`/admin/chapters?series=${s.id}`}
                          className="inline-flex items-center gap-1.5 text-slate-300 hover:text-indigo-300"
                        >
                          <Icon name="layers" className="h-3.5 w-3.5" />
                          {s.chapter_count}
                        </Link>
                      </td>
                      <td className="px-5 py-4">
                        <Badge tone={a.tone}>{a.label}</Badge>
                      </td>
                      <td className="px-5 py-4 text-slate-500">{fmtDate(s.created_at)}</td>
                      <td className="px-5 py-4">
                        <div className="flex justify-end gap-1">
                          <button
                            onClick={() => openEdit(s)}
                            aria-label={`Editar ${s.title}`}
                            className="rounded-lg p-2 text-slate-400 hover:bg-white/5 hover:text-white"
                          >
                            <Icon name="edit" />
                          </button>
                          <button
                            onClick={() => setToDelete(s)}
                            aria-label={`Excluir ${s.title}`}
                            className="rounded-lg p-2 text-slate-400 hover:bg-red-500/10 hover:text-red-300"
                          >
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

      {modal && (
        <SeriesForm
          mode={modal.id ? "edit" : "create"}
          series={modal.id ? items.find((s) => s.id === modal.id) ?? null : null}
          onClose={() => setModal(null)}
          onSubmit={save}
          busy={busy}
        />
      )}

      {toDelete && (
        <Modal
          title="Excluir série"
          onClose={() => setToDelete(null)}
          footer={
            <>
              <Btn variant="ghost" onClick={() => setToDelete(null)}>
                Cancelar
              </Btn>
              <Btn variant="danger" onClick={remove} disabled={busy}>
                {busy ? "Excluindo…" : "Excluir"}
              </Btn>
            </>
          }
        >
          <p className="text-sm text-slate-300">
            Tem certeza que deseja excluir <strong className="text-white">{toDelete.title}</strong>?
            {toDelete.chapter_count > 0 && (
              <span className="mt-2 block text-amber-300">
                Esta série tem {toDelete.chapter_count} capítulo(s) que também podem ser afetados.
              </span>
            )}
          </p>
        </Modal>
      )}

      {toast.node}
    </div>
  );
}
