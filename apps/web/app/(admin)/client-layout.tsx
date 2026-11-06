"use client";

import Link from "next/link";
import { usePathname } from "next/navigation";
import { useState } from "react";

type NavItem = { label: string; href: string };
type NavGroup = { label: string; icon: string; items: NavItem[] };

const ICONS: Record<string, string> = {
  home: "M3 10.5 12 3l9 7.5V20a1 1 0 0 1-1 1h-5v-6H9v6H4a1 1 0 0 1-1-1z",
  book: "M4 5a2 2 0 0 1 2-2h13v16H6a2 2 0 0 0-2 2zM4 19a2 2 0 0 1 2-2h13",
  users: "M16 20v-1a4 4 0 0 0-4-4H7a4 4 0 0 0-4 4v1M9.5 11a3.5 3.5 0 1 0 0-7 3.5 3.5 0 0 0 0 7M21 20v-1a4 4 0 0 0-3-3.9M16 4.1a3.5 3.5 0 0 1 0 6.8",
  crown: "M3 8l4.5 4L12 5l4.5 7L21 8l-2 11H5z",
  cog: "M12 15a3 3 0 1 0 0-6 3 3 0 0 0 0 6M19.4 15a1.7 1.7 0 0 0 .3 1.9l.1.1a2 2 0 1 1-2.8 2.8l-.1-.1a1.7 1.7 0 0 0-1.9-.3 1.7 1.7 0 0 0-1 1.5V21a2 2 0 1 1-4 0v-.1a1.7 1.7 0 0 0-1-1.5 1.7 1.7 0 0 0-1.9.3l-.1.1a2 2 0 1 1-2.8-2.8l.1-.1a1.7 1.7 0 0 0 .3-1.9 1.7 1.7 0 0 0-1.5-1H3a2 2 0 1 1 0-4h.1a1.7 1.7 0 0 0 1.5-1 1.7 1.7 0 0 0-.3-1.9l-.1-.1a2 2 0 1 1 2.8-2.8l.1.1a1.7 1.7 0 0 0 1.9.3h0a1.7 1.7 0 0 0 1-1.5V3a2 2 0 1 1 4 0v.1a1.7 1.7 0 0 0 1 1.5h0a1.7 1.7 0 0 0 1.9-.3l.1-.1a2 2 0 1 1 2.8 2.8l-.1.1a1.7 1.7 0 0 0-.3 1.9v0a1.7 1.7 0 0 0 1.5 1H21a2 2 0 1 1 0 4h-.1a1.7 1.7 0 0 0-1.5 1z",
  menu: "M4 6h16M4 12h16M4 18h16",
  chevron: "m6 9 6 6 6-6",
  logout: "M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4M16 17l5-5-5-5M21 12H9",
  search: "m21 21-4.3-4.3M17 11a6 6 0 1 1-12 0 6 6 0 0 1 12 0",
};

const Icon = ({ name, className = "w-[18px] h-[18px]" }: { name: string; className?: string }) => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" className={className}>
    <path d={ICONS[name]} />
  </svg>
);

const GROUPS: NavGroup[] = [
  { label: "Geral", icon: "home", items: [{ label: "Dashboard", href: "/admin" }] },
  {
    label: "Catálogo",
    icon: "book",
    items: [
      { label: "Séries", href: "/admin/series" },
      { label: "Capítulos", href: "/admin/chapters" },
    ],
  },
  { label: "Usuários", icon: "users", items: [{ label: "Usuários", href: "/admin/users" }] },
  { label: "VIP", icon: "crown", items: [{ label: "Planos", href: "/admin/vip" }] },
  {
    label: "Configurações",
    icon: "cog",
    items: [
      { label: "Geral", href: "/admin/settings?group=site" },
      { label: "Busca", href: "/admin/settings?group=search" },
      { label: "Interface", href: "/admin/settings?group=ui" },
      { label: "Interno", href: "/admin/settings?group=internal" },
    ],
  },
];

const basePath = (href: string) => href.split("?")[0];

export default function ClientLayout({ children }: { children: React.ReactNode }) {
  const pathname = usePathname();
  const [collapsed, setCollapsed] = useState(false);
  const [open, setOpen] = useState<Set<string>>(new Set(["Geral", "Catálogo"]));

  const isActive = (href: string) =>
    !href.includes("?") && (href === "/admin" ? pathname === "/admin" : pathname.startsWith(href));

  const toggle = (label: string) =>
    setOpen((prev) => {
      const next = new Set(prev);
      next.has(label) ? next.delete(label) : next.add(label);
      return next;
    });

  const title =
    GROUPS.flatMap((g) => g.items).find((i) => basePath(i.href) === pathname)?.label ?? "Admin";

  return (
    <div className="flex h-screen bg-[#0b1020] text-slate-200">
      <aside
        className={`flex shrink-0 flex-col border-r border-white/5 bg-[#0e1426] transition-[width] duration-200 ${
          collapsed ? "w-[72px]" : "w-64"
        }`}
      >
        <div className="flex h-16 items-center gap-3 border-b border-white/5 px-5">
          <div className="grid h-8 w-8 shrink-0 place-items-center rounded-lg bg-gradient-to-br from-indigo-500 to-sky-400 text-sm font-black text-white">
            M
          </div>
          {!collapsed && (
            <div className="leading-none">
              <p className="text-[15px] font-bold tracking-tight text-white">MangaShelf</p>
              <p className="mt-1 text-[11px] text-slate-500">Painel admin</p>
            </div>
          )}
        </div>

        <nav className="flex-1 space-y-1 overflow-y-auto px-3 py-4">
          {GROUPS.map((group) => {
            const groupActive = group.items.some((i) => isActive(i.href));
            const expanded = open.has(group.label) && !collapsed;
            const single = group.items.length === 1;

            if (single) {
              const item = group.items[0];
              const active = isActive(item.href);
              return (
                <Link
                  key={group.label}
                  href={item.href}
                  title={group.label}
                  className={`flex items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors ${
                    active ? "bg-indigo-500/15 text-indigo-300" : "text-slate-400 hover:bg-white/5 hover:text-white"
                  } ${collapsed ? "justify-center" : ""}`}
                >
                  <Icon name={group.icon} />
                  {!collapsed && <span>{group.label}</span>}
                </Link>
              );
            }

            return (
              <div key={group.label}>
                <button
                  onClick={() => (collapsed ? setCollapsed(false) : toggle(group.label))}
                  title={group.label}
                  className={`flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium transition-colors ${
                    groupActive ? "text-white" : "text-slate-400 hover:bg-white/5 hover:text-white"
                  } ${collapsed ? "justify-center" : ""}`}
                >
                  <Icon name={group.icon} />
                  {!collapsed && (
                    <>
                      <span className="flex-1 text-left">{group.label}</span>
                      <Icon name="chevron" className={`h-4 w-4 transition-transform ${expanded ? "rotate-180" : ""}`} />
                    </>
                  )}
                </button>
                {expanded && (
                  <div className="ml-[22px] mt-1 space-y-0.5 border-l border-white/10 pl-3">
                    {group.items.map((item) => (
                      <Link
                        key={item.href}
                        href={item.href}
                        className={`block rounded-md px-3 py-2 text-[13px] transition-colors ${
                          isActive(item.href)
                            ? "bg-indigo-500/15 font-medium text-indigo-300"
                            : "text-slate-400 hover:bg-white/5 hover:text-white"
                        }`}
                      >
                        {item.label}
                      </Link>
                    ))}
                  </div>
                )}
              </div>
            );
          })}
        </nav>

        <div className="border-t border-white/5 p-3">
          <button
            title="Sair"
            className={`flex w-full items-center gap-3 rounded-lg px-3 py-2.5 text-sm font-medium text-slate-400 transition-colors hover:bg-red-500/10 hover:text-red-300 ${
              collapsed ? "justify-center" : ""
            }`}
          >
            <Icon name="logout" />
            {!collapsed && <span>Sair</span>}
          </button>
        </div>
      </aside>

      <div className="flex min-w-0 flex-1 flex-col">
        <header className="flex h-16 shrink-0 items-center gap-4 border-b border-white/5 bg-[#0b1020]/80 px-6 backdrop-blur">
          <button
            onClick={() => setCollapsed((p) => !p)}
            aria-label="Alternar sidebar"
            className="rounded-lg p-2 text-slate-400 transition-colors hover:bg-white/5 hover:text-white"
          >
            <Icon name="menu" className="h-5 w-5" />
          </button>
          <div>
            <h1 className="text-lg font-bold leading-tight text-white">{title}</h1>
            <p className="hidden text-xs text-slate-500 sm:block">Painel administrativo do MangaShelf</p>
          </div>

          <div className="ml-auto flex items-center gap-4">
            <label className="hidden w-64 items-center gap-2 rounded-full border border-white/10 bg-white/5 px-4 py-2 text-slate-500 focus-within:border-indigo-400/50 md:flex">
              <Icon name="search" className="h-4 w-4" />
              <input
                placeholder="Buscar"
                className="w-full bg-transparent text-sm text-slate-200 outline-none placeholder:text-slate-500"
              />
            </label>
            <div className="hidden items-center gap-2 rounded-full border border-emerald-400/20 bg-emerald-400/10 px-3 py-1.5 text-xs font-medium text-emerald-300 md:flex">
              <span className="h-1.5 w-1.5 rounded-full bg-emerald-400" />
              Sistema online
            </div>
            <div className="h-9 w-9 overflow-hidden rounded-full border border-white/10">
              {/* eslint-disable-next-line @next/next/no-img-element */}
              <img src="/covers/solo-leveling.png" alt="Admin" className="h-full w-full object-cover" />
            </div>
          </div>
        </header>

        <main className="flex-1 overflow-y-auto p-6 lg:p-8">{children}</main>
      </div>
    </div>
  );
}