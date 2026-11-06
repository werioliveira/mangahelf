"use client";

import React, { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { useSearchParams } from "next/navigation";
import {
  API_URL, Badge, Btn, Empty, Field, Icon, Modal, PageHeader, Panel, SearchBox, Spinner, Th,
  errMsg, fmtDate, inputCls, useToast, type Tone,
} from "../_components/ui";

type Group = "site" | "search" | "ui" | "internal";

const GROUP_LABELS: Record<Group, string> = {
  site: "Site",
  search: "Busca",
  ui: "Interface",
  internal: "Interno",
};

const GROUP_ICONS: Record<Group, React.ReactNode> = {
  site: (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
      <path d="M3 7v16a2 2 0 002 2h14a2 2 0 002-2V7a2 2 0 00-2-2H5a2 2 0 00-2-2z" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
      <path d="M17 17h5M17 12h5" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  ),
  search: (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
      <circle cx="11" cy="11" r="8" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
      <path d="M21 21l-4.3-4.3" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  ),
  ui: (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
      <rect x="2" y="3" width="20" height="14" rx="2" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
      <line x1="8" y1="21" x2="16" y2="21" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
      <line x1="12" y1="17" x2="12" y2="21" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  ),
  internal: (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" xmlns="http://www.w3.org/2000/svg">
      <rect x="2" y="2" width="20" height="20" rx="5" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
      <path d="M16 11.37A4 4 0 1112.63 8 4 4 0 018.75 3.25" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" />
    </svg>
  ),
};

const GROUPS: { key: Group; label: string }[] = [
  { key: "site", label: "Site" },
  { key: "search", label: "Busca" },
  { key: "ui", label: "Interface" },
  { key: "internal", label: "Interno" },
];

const unquote = (v: unknown): string => {
  if (typeof v !== "string") return String(v);
  if ((v.startsWith('"') && v.endsWith('"')) || (v.startsWith("'") && v.endsWith("'"))) {
    return v.slice(1, -1);
  }
  return v;
};

const displayValue = (v: unknown) => {
  if (v === null || v === undefined) return "—";
  if (typeof v === "boolean") return v ? "true" : "false";
  if (typeof v === "number") return String(v);
  const s = unquote(v);
  return s.length > 40 ? `${s.slice(0, 37)}...` : s;
};

const TYPE_COLORS: Record<string, string> = {
  string: "bg-emerald-400/10 text-emerald-300 ring-emerald-400/20",
  boolean: "bg-sky-400/10 text-sky-300 ring-sky-400/20",
  color: "bg-violet-400/10 text-violet-300 ring-violet-400/20",
  json: "bg-amber-400/10 text-amber-300 ring-amber-400/20",
};

const TYPE_LABELS: Record<string, string> = {
  string: "Texto",
  boolean: "Booleano",
  color: "Cor",
  json: "JSON",
};

async function fetchSettings(group: Group) {
  const res = await fetch(`${API_URL}/admin/settings?group=${group}`, { credentials: "include" });
  if (!res.ok) throw new Error("Erro ao carregar configurações");
  return res.json() as Promise<{ settings: Setting[]; total: number }>;
}

async function updateSetting(key: string, value: unknown, type: string, group: Group) {
  const res = await fetch(`${API_URL}/admin/settings/${key}`, {
    method: "PATCH",
    credentials: "include",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ value, type, setting_group: group }),
  });
  if (!res.ok) throw new Error("Erro ao salvar");
  return res.json();
}

export default function SettingsPage() {
  const searchParams = useSearchParams();
  const urlGroup = searchParams.get("group") as Group | null;
  const [activeGroup, setActiveGroup] = useState<Group>(urlGroup ?? "site");
  const [editingKey, setEditingKey] = useState<string | null>(null);
  const [editingValue, setEditingValue] = useState("");
  const [savingKeys, setSavingKeys] = useState<Set<string>>(new Set());
  const { data, isLoading, error } = useSettings(activeGroup);
  const toast = useToast();

  // Re-fetch quando o grupo muda na URL (?group=site, ?group=search, etc.)
  useEffect(() => {
    if (urlGroup && urlGroup !== activeGroup) {
      setActiveGroup(urlGroup);
    }
  }, [urlGroup, activeGroup]);

  const settings: Setting[] = data?.settings ?? [];
  const total = data?.total ?? 0;

  const handleSave = async (key: string) => {
    const setting = settings.find((s) => s.key === key);
    if (!setting) return;
    try {
      setSavingKeys((prev) => new Set([...prev, key]));
      let parsed: unknown = editingValue;
      if (setting.type === "boolean") parsed = editingValue === "true";
      if (setting.type === "color") parsed = editingValue;
      if (setting.type === "json") parsed = JSON.parse(editingValue);
      await updateSetting(key, parsed, setting.type, setting.setting_group as Group);
      toast.show("Configuração salva.");
      setEditingKey(null);
      setEditingValue("");
    } catch (err) {
      toast.show(errMsg(err, "Erro ao salvar configuração"), false);
    } finally {
      setSavingKeys((prev) => {
        const next = new Set(prev);
        next.delete(key);
        return next;
      });
    }
  };

  if (isLoading) return <Spinner label="Carregando configurações…" />;

  if (error) {
    return (
      <div className="rounded-xl border border-red-400/20 bg-red-400/10 px-4 py-3 text-sm text-red-300">
        {error}
      </div>
    );
  }

  return (
    <div className="mx-auto max-w-7xl space-y-6">
      <PageHeader
        title="Configurações do Sistema"
        subtitle={`Gerencie configurações globais do MangaShelf sem redeploy. Grupo: ${GROUP_LABELS[activeGroup]}.`}
      />

      <Panel>
        {settings.length > 0 ? (
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead className="border-b border-white/5 bg-white/[0.02]">
                <tr>
                  <Th>Chave</Th>
                  <Th>Valor</Th>
                  <Th>Tipo</Th>
                  <Th right>Ações</Th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/5">
                {settings.map((s) => (
                  <SettingsRow key={s.key} s={s} isEditing={editingKey === s.key} isSaving={savingKeys.has(s.key)} onEdit={(key: string) => setEditingKey(key)} onSave={handleSave} onCancel={() => setEditingKey(null)} />
                ))}
              </tbody>
            </table>
          </div>
        ) : (
          <Empty
            icon="cog"
            title="Nenhuma configuração neste grupo."
            text={`O grupo ${GROUP_LABELS[activeGroup]} não possui configurações cadastradas.`}
          />
        )}
      </Panel>

      {toast.node}
    </div>
  );
}

function SettingsRow({ s, isEditing, isSaving, onEdit, onSave, onCancel }: { s: Setting; isEditing: boolean; isSaving: boolean; onEdit: (key: string) => void; onSave: (key: string) => void; onCancel: () => void }) {
  const [localValue, setLocalValue] = useState(typeof s.value === "string" ? s.value : JSON.stringify(s.value));

  const handleSave = () => {
    onSave(s.key);
  };

  return (
    <tr key={s.key} className="transition-colors hover:bg-white/[0.03]">
      <td className="px-5 py-3.5 font-medium text-slate-100 font-mono text-xs">{s.key}</td>
      <td className="px-5 py-3.5">
        {isEditing ? (
          <div className="flex gap-2">
            {s.type === "boolean" ? (
              <select
                value={localValue}
                onChange={(e) => setLocalValue(e.target.value)}
                className={`${inputCls} !w-auto`}
              >
                <option value="true">true</option>
                <option value="false">false</option>
              </select>
            ) : (
              <input
                type="text"
                value={localValue}
                onChange={(e) => setLocalValue(e.target.value)}
                placeholder={displayValue(s.value)}
                className={`${inputCls} !w-auto`}
              />
            )}
            <Btn onClick={handleSave} disabled={isSaving}>
              {isSaving ? "Salvando…" : "Salvar"}
            </Btn>
            <Btn variant="ghost" onClick={onCancel}>Cancelar</Btn>
          </div>
        ) : (
          <div className="flex items-center gap-3">
            {s.type === "color" && (
              <div className="w-8 h-8 rounded-full border border-white/10 flex items-center justify-center">
                <div className="w-6 h-6 rounded-full" style={{ backgroundColor: unquote(s.value as string) }} />
              </div>
            )}
            <span className={`font-mono text-sm ${s.type === "color" ? "hidden" : "block"} truncate max-w-[200px]`}>
              {displayValue(s.value)}
            </span>
            <Btn variant="ghost" onClick={() => onEdit(s.key)} title="Editar">
              <Icon name="edit" />
            </Btn>
          </div>
        )}
      </td>
      <td className="px-5 py-3.5">
        <Badge tone={s.type as Tone}>{TYPE_LABELS[s.type]}</Badge>
      </td>
      <td className="px-5 py-3.5 text-right text-xs text-slate-500 font-mono">
        {fmtDate(s.updated_at)}
      </td>
    </tr>
  );
}

type Setting = {
  key: string;
  value: string;
  type: string;
  setting_group: string;
  updated_at: string;
};

function useSettings(group: Group) {
  const [data, setData] = useState<{ settings: Setting[]; total: number } | null>(null);
  const [isLoading, setIsLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    fetchSettings(group)
      .then((d) => setData(d))
      .catch((e) => setError(errMsg(e, "Erro ao carregar configurações")))
      .finally(() => setIsLoading(false));
  }, [group]);

  return { data, isLoading, error };
}

function GroupIcon({ groupKey }: { groupKey: Group }) {
  return GROUP_ICONS[groupKey];
}
