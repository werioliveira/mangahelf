"use client";

import { useState, useEffect } from "react";
import Link from "next/link";
import { useSettings, updateSetting, Setting } from "@/lib/api";

type Group = "site" | "search" | "ui" | "internal";

const GROUP_LABELS: Record<Group, string> = {
  site: "Site",
  search: "Busca",
  ui: "Interface",
  internal: "Interno",
};

const GROUPS: { key: Group; label: string }[] = [
  { key: "site", label: "Site" },
  { key: "search", label: "Busca" },
  { key: "ui", label: "Interface" },
  { key: "internal", label: "Interno" },
];

export default function SettingsPage() {
  const [activeGroup, setActiveGroup] = useState<Group>("site");
  const [editingKey, setEditingKey] = useState<string | null>(null);
  const [editingValue, setEditingValue] = useState("");
  const { data, isLoading, error } = useSettings(activeGroup);

  const settings = data?.settings ?? [];
  const total = data?.total ?? 0;

  const handleSave = async (key: string) => {
    const setting = settings.find((s) => s.key === key);
    if (!setting) return;
    try {
      let parsed: unknown = editingValue;
      if (setting.type === "boolean") parsed = editingValue === "true";
      if (setting.type === "color") parsed = editingValue;
      if (setting.type === "json") parsed = JSON.parse(editingValue);
      await updateSetting(key, parsed, setting.type, setting.setting_group);
      setEditingKey(null);
      setEditingValue("");
    } catch (err) {
      alert(`Erro ao salvar: ${(err as Error).message}`);
    }
  };

  if (isLoading) {
    return (
      <div className="flex items-center justify-center py-20">
        <div className="w-8 h-8 rounded-full border-2 border-primary border-t-transparent animate-spin"></div>
        <p className="text-sm text-muted-foreground ml-3">Carregando configurações…</p>
      </div>
    );
  }

  if (error) {
    return (
      <div className="bg-destructive/10 border border-destructive/20 rounded-xl p-6 text-destructive text-sm">
        Erro ao carregar configurações. Tente novamente.
      </div>
    );
  }

  return (
    <section className="relative py-12 md:py-20">
      <div className="max-w-[1400px] mx-auto px-6">
        <div className="flex items-center justify-between mb-8">
          <div>
            <h2 className="text-2xl font-black text-slate-900">Configurações do Sistema</h2>
            <p className="text-slate-500">Gerencie configurações globais do MangaShelf sem redeploy.</p>
          </div>
          <Link href="/admin" className="inline-flex items-center gap-2 text-sm font-medium text-muted-foreground hover:text-foreground">
            ← Voltar ao Dashboard
          </Link>
        </div>

        {/* Tabs de grupos */}
        <div className="flex gap-1 mb-6">
          {GROUPS.map(({ key, label }) => (
            <button
              key={key}
              onClick={() => setActiveGroup(key)}
              className={`px-4 py-2 rounded-lg text-sm font-medium transition-all ${
                activeGroup === key
                  ? "bg-primary text-white"
                  : "bg-transparent text-muted-foreground hover:bg-secondary/50"
              }`}
            >
              {label}
            </button>
          ))}
        </div>

        {error && (
          <div className="mb-6 bg-destructive/10 border border-destructive/20 rounded-xl p-6 text-destructive text-sm">
            {error}
          </div>
        )}

        <div className="bg-white rounded-xl border border-slate-200 p-6">
          <div className="flex items-center justify-between mb-6">
            <h3 className="text-sm font-black text-slate-900 uppercase tracking-wider">
              {GROUP_LABELS[activeGroup]} ({total} configuração{total !== 1 ? "s" : ""})
            </h3>
          </div>

          {settings.length === 0 ? (
            <div className="py-12 text-center">
              <div className="w-16 h-16 rounded-full bg-muted/50 flex items-center justify-center mx-auto mb-4">
                <svg className="w-8 h-8 text-muted-foreground/50" fill="none" stroke="currentColor" viewBox="0 0 24 24">
                  <path strokeLinecap="round" strokeLinejoin="round" strokeWidth="1.5" d="M4 7v10c0 2.21 3.582 4 8 4s8-1.79 8-4V4M4 7c0 2.21 3.582 4 8 4s8-1.79 8-4V4m0 0c0-2.21-3.582-4-8-4S4 1.79 4 4zm0 0v10c0 2.21 3.582 4 8 4s8-1.79 8-4V4" />
                </svg>
              </div>
              <p className="text-sm text-muted-foreground">Nenhuma configuração neste grupo.</p>
            </div>
          ) : (
            <div className="overflow-x-auto">
              <table className="w-full text-sm text-left">
                <thead>
                  <tr className="border-b border-slate-200">
                    <th className="pb-3 text-xs font-semibold text-slate-500 uppercase w-1/3">Chave</th>
                    <th className="pb-3 text-xs font-semibold text-slate-500 uppercase w-1/3">Valor</th>
                    <th className="pb-3 text-xs font-semibold text-slate-500 uppercase w-1/6">Tipo</th>
                    <th className="pb-3 text-xs font-semibold text-slate-500 uppercase text-right">Ação</th>
                  </tr>
                </thead>
                <tbody className="divide-y divide-slate-100">
                  {settings.map((s) => {
                    const isEditing = editingKey === s.key;
                    return (
                      <tr key={s.key} className="hover:bg-slate-50/50 transition-all">
                        <td className="py-3 text-sm text-slate-700 font-mono">{s.key}</td>
                        <td className="py-3">
                          {isEditing ? (
                            <div className="flex items-center gap-2">
                              {s.type === "boolean" ? (
                                <select
                                  value={editingValue}
                                  onChange={(e) => setEditingValue(e.target.value)}
                                  className="px-2 py-1 text-sm bg-card border border-input rounded-lg focus:outline-none focus:ring-2 focus:ring-ring"
                                >
                                  <option value="true">true</option>
                                  <option value="false">false</option>
                                </select>
                              ) : (
                                <input
                                  type="text"
                                  value={editingValue}
                                  onChange={(e) => setEditingValue(e.target.value)}
                                  className="px-2 py-1 text-sm bg-card border border-input rounded-lg focus:outline-none focus:ring-2 focus:ring-ring"
                                />
                              )}
                              <button
                                onClick={() => handleSave(s.key)}
                                className="px-3 py-1 text-xs font-medium bg-primary text-white rounded-lg hover:bg-primary/90 transition-colors"
                              >
                                Salvar
                              </button>
                              <button
                                onClick={() => setEditingKey(null)}
                                className="px-3 py-1 text-xs font-medium text-slate-500 hover:text-slate-700"
                              >
                                Cancelar
                              </button>
                            </div>
                          ) : (
                            <>
                              <span className=                              {s.type === "color" ? "block w-8 h-8 rounded-full border border-slate-200" : "inline-block"}>
                                {s.type === "color" ? (
                                  <div className="w-7 h-7 rounded-full mx-auto" style={{ backgroundColor: s.value }} />
                                ) : (
                                  s.value.length > 40 ? `${s.value.slice(0, 37)}...` : s.value
                                )}
                              </span>
                              {s.type === "color" ? (
                                <span className="text-[10px] text-muted-foreground block mt-0.5">{s.value}</span>
                              ) : null}
                            </>
                          )}
                        </td>
                        <td className="py-3 text-xs font-mono text-muted-foreground">{s.type}</td>
                        <td className="py-3 text-right">
                          {!isEditing && (
                            <button
                              onClick={() => {
                                setEditingKey(s.key);
                                setEditingValue(typeof s.value === "string" ? s.value : JSON.stringify(s.value));
                              }}
                              className="text-xs font-medium text-primary hover:text-primary/90 underline underline-offset-4"
                            >
                              Editar
                            </button>
                          )}
                        </td>
                      </tr>
                    );
                  })}
                </tbody>
              </table>
            </div>
          )}
        </div>
      </div>
    </section>
  );
}
