"use client";

import {
  API_URL, Badge, Btn, Empty, Field, Icon, Modal, PageHeader, Panel, SearchBox, Spinner, Th,
  errMsg, fmtDate, inputCls, useToast, StatCard,
} from "../_components/ui";

type Plan = {
  id: string;
  name: string;
  price: string;
  duration: number;
  perks: string[];
  is_active: boolean;
};

async function fetchPlans() {
  const res = await fetch(`${API_URL}/admin/vip/plans`, { credentials: "include" });
  if (!res.ok) throw new Error("Erro ao carregar planos VIP");
  return res.json() as Promise<Plan[]>;
}

async function patchPlan(id: string, body: Record<string, unknown>) {
  const res = await fetch(`${API_URL}/admin/vip/plans/${id}`, {
    method: "PATCH",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) throw new Error("Erro ao salvar");
  return res.json();
}

export default function VipPage() {
  const [plans, setPlans] = useState<Plan[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [query, setQuery] = useState("");

  const [editing, setEditing] = useState<Plan | null>(null);
  const [editName, setEditName] = useState("");
  const [editDuration, setEditDuration] = useState("");
  const [saving, setSaving] = useState(false);
  const [toDelete, setToDelete] = useState<Plan | null>(null);
  const toast = useToast();

  useEffect(() => {
    fetchPlans()
      .then((data) => setPlans(data ?? []))
      .catch((e) => setError(errMsg(e, "Erro ao carregar planos VIP")))
      .finally(() => setLoading(false));
  }, []);

  const visible = useMemo(() => {
    const q = query.trim().toLowerCase();
    return plans.filter(
      (p) => !q || p.name.toLowerCase().includes(q),
    );
  }, [plans, query]);

  const openEdit = (p: Plan) => {
    setEditing(p);
    setEditName(p.name);
    setEditDuration(String(p.duration));
  };

  const save = async () => {
    if (!editing) return;
    setSaving(true);
    try {
      await patchPlan(editing.id, { name: editName, duration: parseInt(editDuration) || 0 });
      setPlans((p) => p.map((p) => (p.id === editing.id ? { ...p, name: editName, duration: parseInt(editDuration) || 0 } : p)));
      toast.show("Plano atualizado.");
      setEditing(null);
    } catch (e) {
      toast.show(errMsg(e, "Erro ao editar plano"), false);
    } finally {
      setSaving(false);
    }
  };

  const remove = async () => {
    if (!toDelete) return;
    setSaving(true);
    try {
      const res = await fetch(`${API_URL}/admin/vip/plans/${toDelete.id}`, {
        method: "DELETE",
        credentials: "include",
      });
      if (!res.ok) throw new Error("Erro ao excluir");
      setPlans((p) => p.filter((p) => p.id !== toDelete.id));
      toast.show("Plano excluído.");
      setToDelete(null);
    } catch (e) {
      toast.show(errMsg(e, "Erro ao excluir plano"), false);
    } finally {
      setSaving(false);
    }
  };

  const newBtn = (
    <Btn onClick={() => toast.show("Criação de plano não implementada.", false)}>
      <Icon name="plus" /> Novo plano
    </Btn>
  );

  if (loading) return <Spinner label="Carregando planos VIP…" />;

  return (
    <div className="mx-auto max-w-7xl space-y-6">
      <PageHeader
        title="Planos VIP"
        subtitle={`${plans.length} plano(s) cadastrado(s). Gerencie assinaturas e benefícios.`}
        action={newBtn}
      />

      {error && (
        <div className="rounded-xl border border-red-400/20 bg-red-400/10 px-4 py-3 text-sm text-red-300">
          {error}
        </div>
      )}

      <div className="flex flex-wrap items-center gap-3">
        <SearchBox value={query} onChange={setQuery} placeholder="Buscar por nome do plano" />
      </div>

      <Panel>
        {visible.length === 0 ? (
          <Empty
            icon="crown"
            title={plans.length === 0 ? "Nenhum plano VIP cadastrado" : "Nada encontrado"}
            text={plans.length === 0 ? "Cadastre o primeiro plano para começar." : "Tente outro termo de busca."}
            action={plans.length === 0 ? newBtn : undefined}
          />
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead className="border-b border-white/5 bg-white/[0.02]">
                <tr>
                  <Th>Nome</Th>
                  <Th>Preço</Th>
                  <Th>Duração</Th>
                  <Th>Benefícios</Th>
                  <Th right>Status</Th>
                  <Th right>Ações</Th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/5">
                {visible.map((p) => (
                  <tr key={p.id} className="transition-colors hover:bg-white/[0.03]">
                    <td className="px-5 py-3.5 font-medium text-slate-100">{p.name}</td>
                    <td className="px-5 py-3.5 text-slate-300">{p.price}</td>
                    <td className="px-5 py-3.5 font-mono text-xs text-slate-300">{p.duration} dias</td>
                    <td className="px-5 py-3.5 text-slate-300">{Array.isArray(p.perks) ? p.perks.join(", ") : "—"}</td>
                    <td className="px-5 py-3.5 text-right">
                      <Badge tone={p.is_active ? "green" : "gray"}>
                        {p.is_active ? "Ativo" : "Inativo"}
                      </Badge>
                    </td>
                    <td className="px-5 py-3.5">
                      <div className="flex justify-end gap-1">
                        <button
                          onClick={() => openEdit(p)}
                          aria-label={`Editar ${p.name}`}
                          className="rounded-lg p-2 text-slate-400 hover:bg-white/5 hover:text-white"
                        >
                          <Icon name="edit" />
                        </button>
                        <button
                          onClick={() => setToDelete(p)}
                          aria-label={`Excluir ${p.name}`}
                          className="rounded-lg p-2 text-slate-400 hover:bg-red-500/10 hover:text-red-300"
                        >
                          <Icon name="trash" />
                        </button>
                      </div>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
      </Panel>

      {editing && (
        <Modal
          title="Editar plano VIP"
          onClose={() => setEditing(null)}
          footer={
            <>
              <Btn variant="ghost" onClick={() => setEditing(null)}>Cancelar</Btn>
              <Btn onClick={save} disabled={saving}>
                {saving ? "Salvando…" : "Salvar alterações"}
              </Btn>
            </>
          }
        >
          <Field label="Nome">
            <input value={editName} onChange={(e) => setEditName(e.target.value)} className={inputCls} />
          </Field>
          <Field label="Duração (dias)">
            <input type="number" value={editDuration} onChange={(e) => setEditDuration(e.target.value)} className={inputCls} />
          </Field>
        </Modal>
      )}

      {toDelete && (
        <Modal
          title="Excluir plano VIP"
          onClose={() => setToDelete(null)}
          footer={
            <>
              <Btn variant="ghost" onClick={() => setToDelete(null)}>Cancelar</Btn>
              <Btn variant="danger" onClick={remove} disabled={saving}>
                {saving ? "Excluindo…" : "Excluir"}
              </Btn>
            </>
          }
        >
          <p className="text-sm text-slate-300">
            Tem certeza que deseja excluir <strong className="text-white">{toDelete.name}</strong>?
          </p>
        </Modal>
      )}

      {toast.node}
    </div>
  );
}

import { useEffect, useMemo, useState } from "react";
