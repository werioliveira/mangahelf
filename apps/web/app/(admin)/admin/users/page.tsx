"use client";

import Link from "next/link";
import {
  API_URL, Badge, Btn, Empty, Field, Icon, Modal, PageHeader, Panel, SearchBox, Segmented, Spinner, Th,
  errMsg, fmtDate, inputCls, useToast, StatCard, type Tone,
} from "../_components/ui";

type User = {
  id: string;
  email: string;
  username: string | null;
  status: string;
  has_vip: boolean;
  vip_expires: string | null;
  created_at: string;
};

const STATUS: Record<string, { label: string; tone: Tone }> = {
  active: { label: "Ativo", tone: "green" },
  suspended: { label: "Suspenso", tone: "amber" },
  banned: { label: "Banido", tone: "red" },
  pending_deletion: { label: "Pendente de exclusão", tone: "red" },
};

async function fetchUsers() {
  const res = await fetch(`${API_URL}/admin/users`, { credentials: "include" });
  if (!res.ok) throw new Error("Erro ao carregar usuários");
  return res.json() as Promise<User[]>;
}

async function patchUser(id: string, body: Record<string, unknown>) {
  const res = await fetch(`${API_URL}/admin/users/${id}`, {
    method: "PATCH",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify(body),
  });
  if (!res.ok) throw new Error("Erro ao salvar");
  return res.json();
}

export default function UsersPage() {
  const [users, setUsers] = useState<User[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [query, setQuery] = useState("");
  const [statusFilter, setStatusFilter] = useState("all");

  const [editing, setEditing] = useState<User | null>(null);
  const [editStatus, setEditStatus] = useState("");
  const [saving, setSaving] = useState(false);
  const [toDelete, setToDelete] = useState<User | null>(null);
  const toast = useToast();

  useEffect(() => {
    fetchUsers()
      .then((data) => setUsers(data ?? []))
      .catch((e) => setError(errMsg(e, "Erro ao carregar usuários")))
      .finally(() => setLoading(false));
  }, []);

  const visible = useMemo(() => {
    const q = query.trim().toLowerCase();
    return users.filter(
      (u) =>
        (statusFilter === "all" || u.status === statusFilter) &&
        (!q || u.email.toLowerCase().includes(q) || (u.username ?? "").toLowerCase().includes(q)),
    );
  }, [users, query, statusFilter]);

  const count = (s: string) => users.filter((u) => u.status === s).length;

  const openEdit = (u: User) => {
    setEditing(u);
    setEditStatus(u.status);
  };

  const save = async () => {
    if (!editing) return;
    setSaving(true);
    try {
      await patchUser(editing.id, { status: editStatus });
      setUsers((p) => p.map((u) => (u.id === editing.id ? { ...u, status: editStatus } : u)));
      toast.show("Usuário atualizado.");
      setEditing(null);
    } catch (e) {
      toast.show(errMsg(e, "Erro ao salvar usuário"), false);
    } finally {
      setSaving(false);
    }
  };

  const remove = async () => {
    if (!toDelete) return;
    setSaving(true);
    try {
      const res = await fetch(`${API_URL}/admin/users/${toDelete.id}`, {
        method: "DELETE",
        credentials: "include",
      });
      if (!res.ok) throw new Error("Erro ao excluir");
      setUsers((p) => p.filter((u) => u.id !== toDelete.id));
      toast.show("Usuário excluído.");
      setToDelete(null);
    } catch (e) {
      toast.show(errMsg(e, "Erro ao excluir usuário"), false);
    } finally {
      setSaving(false);
    }
  };

  const newBtn = (
    <Btn onClick={() => toast.show("Cadastro de usuário não implementado.", false)}>
      <Icon name="plus" /> Novo usuário
    </Btn>
  );

  if (loading) return <Spinner label="Carregando usuários…" />;

  return (
    <div className="mx-auto max-w-7xl space-y-6">
      <PageHeader
        title="Usuários"
        subtitle={`${users.length} no catálogo. Controle status e acesso VIP de cada conta.`}
        action={newBtn}
      />

      {error && (
        <div className="rounded-xl border border-red-400/20 bg-red-400/10 px-4 py-3 text-sm text-red-300">
          {error}
        </div>
      )}

      <div className="flex flex-wrap items-center gap-3">
        <SearchBox value={query} onChange={setQuery} placeholder="Buscar por e-mail ou usuário" />
        <Segmented
          value={statusFilter}
          onChange={setStatusFilter}
          options={[
            { value: "all", label: "Todos", count: users.length },
            { value: "active", label: "Ativos", count: count("active") },
            { value: "suspended", label: "Suspensos", count: count("suspended") },
            { value: "banned", label: "Banidos", count: count("banned") },
          ]}
        />
      </div>

      <Panel>
        {visible.length === 0 ? (
          <Empty
            icon="users"
            title={users.length === 0 ? "Nenhum usuário cadastrado" : "Nada encontrado"}
            text={users.length === 0 ? "Cadastre o primeiro usuário para começar." : "Tente outro termo ou remova o filtro."}
            action={users.length === 0 ? newBtn : undefined}
          />
        ) : (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead className="border-b border-white/5 bg-white/[0.02]">
                <tr>
                  <Th>E-mail</Th>
                  <Th>Usuário</Th>
                  <Th>Status</Th>
                  <Th>VIP</Th>
                  <Th right>Criado em</Th>
                  <Th right>Ações</Th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/5">
                {visible.map((u) => (
                  <tr key={u.id} className="group transition-colors hover:bg-white/[0.03]">
                    <td className="max-w-[200px] truncate px-5 py-3.5 font-medium text-slate-100">{u.email}</td>
                    <td className="px-5 py-3.5 font-mono text-xs text-slate-500">{u.username ?? "—"}</td>
                    <td className="px-5 py-3.5">
                      <Badge tone={STATUS[u.status]?.tone ?? "gray"}>
                        {STATUS[u.status]?.label ?? u.status}
                      </Badge>
                    </td>
                    <td className="px-5 py-3.5">
                      <Badge tone={u.has_vip ? "violet" : "gray"}>
                        {u.has_vip ? "VIP" : "Sem VIP"}
                      </Badge>
                    </td>
                    <td className="px-5 py-3.5 text-right font-mono text-xs text-slate-500">
                      {fmtDate(u.created_at)}
                    </td>
                    <td className="px-5 py-3.5 text-right">
                      <div className="flex items-center justify-end gap-1">
                        <button
                          onClick={() => openEdit(u)}
                          aria-label={`Editar ${u.email}`}
                          className="rounded-lg p-2 text-slate-400 hover:bg-white/5 hover:text-white"
                        >
                          <Icon name="edit" />
                        </button>
                        <button
                          onClick={() => setToDelete(u)}
                          aria-label={`Excluir ${u.email}`}
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
          title="Editar usuário"
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
          <Field label="E-mail">
            <input value={editing.email} className={inputCls} readOnly />
          </Field>
          <Field label="Status">
            <select value={editStatus} onChange={(e) => setEditStatus(e.target.value)} className={inputCls}>
              <option value="active">Ativo</option>
              <option value="suspended">Suspenso</option>
              <option value="banned">Banido</option>
              <option value="pending_deletion">Pendente de exclusão</option>
            </select>
          </Field>
        </Modal>
      )}

      {toDelete && (
        <Modal
          title="Excluir usuário"
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
            Tem certeza que deseja excluir <strong className="text-white">{toDelete.email}</strong>?
          </p>
        </Modal>
      )}

      {toast.node}
    </div>
  );
}

import { useEffect, useMemo, useState } from "react";
