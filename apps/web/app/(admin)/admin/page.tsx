import Link from "next/link";
import {
  Badge, Btn, Empty, Field, Icon, Modal, PageHeader, Panel, SearchBox, Segmented, Spinner, Th,
  errMsg, fmtDate, inputCls, useToast, StatCard, type Tone,
} from "./_components/ui";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

type Series = {
  id: string;
  title: string;
  slug: string;
  synopsis: string;
  access: string;
  chapter_count: number;
  created_at: string;
};

type User = {
  id: string;
  email: string;
  is_active: boolean;
  roles: string;
  has_vip: boolean;
  vip_expires: string | null;
  created_at: string;
};

async function fetchJson(path: string) {
  const res = await fetch(`${API_URL}${path}`, { cache: "no-store", credentials: "include" });
  if (!res.ok) return [];
  return res.json();
}

const MONTHS = ["Jan", "Fev", "Mar", "Abr", "Mai", "Jun", "Jul", "Ago", "Set", "Out", "Nov", "Dez"];

function signupsByMonth(users: User[]) {
  const now = new Date();
  const buckets = Array.from({ length: 12 }, (_, i) => {
    const d = new Date(now.getFullYear(), now.getMonth() - (11 - i), 1);
    return { key: `${d.getFullYear()}-${d.getMonth()}`, label: MONTHS[d.getMonth()], count: 0 };
  });
  for (const u of users) {
    const d = new Date(u.created_at);
    const b = buckets.find((x) => x.key === `${d.getFullYear()}-${d.getMonth()}`);
    if (b) b.count++;
  }
  return buckets;
}

const AreaChart = ({ values }: { values: number[] }) => {
  const W = 600;
  const H = 200;
  const max = Math.max(...values, 1);
  const pts = values.map((v, i) => [(i / (values.length - 1)) * W, H - 12 - (v / max) * (H - 32)]);
  const line = pts.map(([x, y]) => `${x},${y}`).join(" ");
  return (
    <svg viewBox={`0 0 ${W} ${H}`} preserveAspectRatio="none" className="h-52 w-full">
      <defs>
        <linearGradient id="area" x1="0" y1="0" x2="0" y2="1">
          <stop offset="0%" stopColor="#818cf8" stopOpacity="0.35" />
          <stop offset="100%" stopColor="#818cf8" stopOpacity="0" />
        </linearGradient>
      </defs>
      {[0.25, 0.5, 0.75].map((f) => (
        <line key={f} x1="0" x2={W} y1={H * f} y2={H * f} stroke="white" strokeOpacity="0.05" />
      ))}
      <polygon points={`0,${H} ${line} ${W},${H}`} fill="url(#area)" />
      <polyline points={line} fill="none" stroke="#818cf8" strokeWidth="2.5" strokeLinejoin="round" strokeLinecap="round" vectorEffect="non-scaling-stroke" />
    </svg>
  );
};

export default async function DashboardPage() {
  const [series, users] = (await Promise.all([fetchJson("/admin/series"), fetchJson("/admin/users")])) as [
    Series[],
    User[],
  ];

  const totalChapters = series.reduce((acc, s) => acc + s.chapter_count, 0);
  const totalVip = users.filter((u) => u.has_vip).length;
  const activeUsers = users.filter((u) => u.is_active).length;
  const avgChapters = series.length ? Math.round(totalChapters / series.length) : 0;
  const vipPct = users.length ? Math.round((totalVip / users.length) * 100) : 0;

  const signups = signupsByMonth(users);
  const signupsTotal = signups.reduce((a, b) => a + b.count, 0);
  const thisMonth = signups[signups.length - 1].count;

  const topSeries = [...series].sort((a, b) => b.chapter_count - a.chapter_count).slice(0, 6);
  const topMax = Math.max(...topSeries.map((s) => s.chapter_count), 1);

  const byAccess = Object.keys(ACCESS_MAP).map((k) => ({
    key: k,
    ...ACCESS_MAP[k as keyof typeof ACCESS_MAP],
    count: series.filter((s) => s.access === k).length,
  }));

  const recentSeries = series.slice(0, 6);
  const recentUsers = users.slice(0, 6);

  return (
    <div className="mx-auto max-w-7xl space-y-6">
      {/* Stats */}
      <div className="grid grid-cols-1 gap-4 sm:grid-cols-2 xl:grid-cols-4">
        <StatCard label="Séries" value={series.length.toLocaleString("pt-BR")} iconName="book" tone="indigo" hint={`${byAccess[0].count} públicas`} />
        <StatCard label="Capítulos" value={totalChapters.toLocaleString("pt-BR")} iconName="layers" tone="sky" hint={`${avgChapters} por série em média`} />
        <StatCard label="Usuários" value={users.length.toLocaleString("pt-BR")} iconName="users" tone="emerald" hint={`${activeUsers} ativos`} />
        <StatCard label="Assinantes VIP" value={totalVip.toLocaleString("pt-BR")} iconName="crown" tone="amber" hint={`${vipPct}% dos usuários`} />
      </div>

      {/* Charts */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-5">
        <Panel className="p-6 lg:col-span-3">
          <div className="mb-6 flex items-start justify-between">
            <div>
              <h2 className="text-base font-semibold text-white">Novos usuários</h2>
              <p className="mt-1 text-xs text-slate-500">Últimos 12 meses</p>
            </div>
            <div className="text-right">
              <p className="text-2xl font-bold text-white">{signupsTotal}</p>
              <p className="text-xs text-slate-500">{thisMonth} neste mês</p>
            </div>
          </div>
          <AreaChart values={signups.map((s) => s.count)} />
          <div className="mt-2 flex justify-between text-[11px] text-slate-500">
            {signups.map((s, i) => (
              <span key={i}>{s.label}</span>
            ))}
          </div>
        </Panel>

        <Panel className="p-6 lg:col-span-2">
          <h2 className="text-base font-semibold text-white">Séries com mais capítulos</h2>
          <p className="mt-1 text-xs text-slate-500">Top {topSeries.length} do catálogo</p>
          <div className="mt-6 space-y-4">
            {topSeries.length === 0 && <p className="text-sm text-slate-500">Nenhuma série cadastrada ainda.</p>}
            {topSeries.map((s) => (
              <div key={s.id}>
                <div className="mb-1.5 flex justify-between text-sm">
                  <span className="truncate pr-3 text-slate-300">{s.title}</span>
                  <span className="font-medium text-white">{s.chapter_count}</span>
                </div>
                <div className="h-2 overflow-hidden rounded-full bg-white/5">
                  <div className="h-full rounded-full bg-gradient-to-r from-indigo-500 to-sky-400" style={{ width: `${(s.chapter_count / topMax) * 100}%` }} />
                </div>
              </div>
            ))}
          </div>

          {series.length > 0 && (
            <div className="mt-7 border-t border-white/5 pt-5">
              <p className="mb-3 text-xs text-slate-500">Acesso das séries</p>
              <div className="flex h-2 overflow-hidden rounded-full bg-white/5">
                {byAccess.map((a) => (
                  <div key={a.key} className={a.bar} style={{ width: `${(a.count / series.length) * 100}%` }} />
                ))}
              </div>
              <div className="mt-3 flex gap-5 text-xs text-slate-400">
                {byAccess.map((a) => (
                  <span key={a.key} className="flex items-center gap-1.5">
                    <span className={`h-2 w-2 rounded-full ${a.bar}`} />
                    {a.label} <span className="text-slate-500">{a.count}</span>
                  </span>
                ))}
              </div>
            </div>
          )}
        </Panel>
      </div>

      {/* Tables */}
      <div className="grid grid-cols-1 gap-4 xl:grid-cols-5">
        <Panel className="overflow-hidden xl:col-span-3">
          <div className="flex items-center justify-between px-6 py-5">
            <h2 className="text-base font-semibold text-white">Séries recentes</h2>
            <Link href="/admin/series" className="text-xs font-medium text-indigo-300 hover:text-indigo-200">
              Ver todas
            </Link>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-left text-sm">
              <thead className="border-b border-white/5 bg-white/[0.02]">
                <tr>
                  <Th>Título</Th>
                  <Th>Slug</Th>
                  <Th>Capítulos</Th>
                  <Th right>Acesso</Th>
                </tr>
              </thead>
              <tbody className="divide-y divide-white/5">
                {recentSeries.length === 0 && (
                  <tr>
                    <td colSpan={4} className="py-10 text-center text-slate-500">
                      Nenhuma série encontrada.
                    </td>
                  </tr>
                )}
                {recentSeries.map((s) => {
                  const a = ACCESS_MAP[s.access] ?? { label: s.access, tone: "gray" as Tone };
                  return (
                    <tr key={s.id} className="transition-colors hover:bg-white/[0.03]">
                      <td className="px-5 py-3.5 font-medium text-slate-100">{s.title}</td>
                      <td className="px-5 py-3.5 font-mono text-xs text-slate-500">{s.slug}</td>
                      <td className="px-5 py-3.5 text-slate-300">{s.chapter_count}</td>
                      <td className="px-5 py-3.5 text-right">
                        <Badge tone={a.tone}>{a.label}</Badge>
                      </td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </Panel>

        <Panel className="overflow-hidden xl:col-span-2">
          <div className="flex items-center justify-between px-6 py-5">
            <h2 className="text-base font-semibold text-white">Usuários recentes</h2>
            <Link href="/admin/users" className="text-xs font-medium text-indigo-300 hover:text-indigo-200">
              Ver todos
            </Link>
          </div>
          <table className="w-full text-left text-sm divide-y divide-white/5">
            <thead className="border-b border-white/5 bg-white/[0.02]">
              <tr>
                <Th>E-mail</Th>
                <Th>Status</Th>
                <Th right>Criado em</Th>
              </tr>
            </thead>
            <tbody className="divide-y divide-white/5">
              {recentUsers.length === 0 && (
                <tr>
                  <td colSpan={3} className="py-10 text-center text-slate-500">
                    Nenhum usuário encontrado.
                  </td>
                </tr>
              )}
              {recentUsers.map((u) => (
                <tr key={u.id} className="transition-colors hover:bg-white/[0.03]">
                  <td className="max-w-[180px] truncate px-5 py-3.5 font-medium text-slate-100">{u.email}</td>
                  <td className="px-5 py-3.5">
                    <div className="flex gap-1.5">
                      <Badge tone={u.is_active ? "green" : "red"}>{u.is_active ? "Ativo" : "Inativo"}</Badge>
                      {u.has_vip && <Badge tone="violet">VIP</Badge>}
                    </div>
                  </td>
                  <td className="px-5 py-3.5 text-right text-xs text-slate-500">
                    {new Date(u.created_at).toLocaleDateString("pt-BR")}
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        </Panel>
      </div>
    </div>
  );
}

const ACCESS_MAP: Record<string, { label: string; tone: Tone; bar: string }> = {
  public: { label: "Público", tone: "green", bar: "bg-emerald-400" },
  early: { label: "Early", tone: "sky", bar: "bg-sky-400" },
  vip_only: { label: "VIP", tone: "violet", bar: "bg-violet-400" },
};
