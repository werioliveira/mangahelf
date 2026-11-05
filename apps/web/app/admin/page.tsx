import Link from "next/link";
import { BookOpen, Users, Shield, TrendingUp, Clock } from "lucide-react";

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

type Series = {
  id: string;
  title: string;
  slug: string;
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
  const res = await fetch(`${API_URL}${path}`, { cache: "no-store" });
  if (!res.ok) return [];
  return res.json();
}

function StatCard({
  title,
  value,
  Icon,
  color = "blue",
}: {
  title: string;
  value: string | number;
  Icon: React.FC<{ className?: string }>;
  color?: string;
}) {
  const colorMap: Record<string, string> = {
    blue: "bg-blue-50 text-blue-700 border-blue-200",
    green: "bg-green-50 text-green-700 border-green-200",
    amber: "bg-amber-50 text-amber-700 border-amber-200",
    slate: "bg-slate-50 text-slate-700 border-slate-200",
  };
  return (
    <div className="rounded-xl border p-4 flex items-center gap-3">
      <div className={`w-10 h-10 rounded-lg flex items-center justify-center ${colorMap[color]}`}>
        <Icon className="w-5 h-5" />
      </div>
      <div>
        <p className="text-xs text-slate-500">{title}</p>
        <p className="text-2xl font-black text-slate-900">{value}</p>
      </div>
    </div>
  );
}

function Badge({ children, variant = "gray" }: { children: React.ReactNode; variant?: string }) {
  const variants: Record<string, string> = {
    gray: "bg-slate-100 text-slate-600 text-xs px-2 py-0.5 rounded",
    green: "bg-green-100 text-green-700 text-xs px-2 py-0.5 rounded",
    amber: "bg-amber-100 text-amber-700 text-xs px-2 py-0.5 rounded",
    red: "bg-red-100 text-red-700 text-xs px-2 py-0.5 rounded",
    blue: "bg-blue-100 text-blue-700 text-xs px-2 py-0.5 rounded",
  };
  return <span className={variants[variant]}>{children}</span>;
}

export default async function DashboardPage() {
  const [series, users] = await Promise.all([
    fetchJson("/admin/series"),
    fetchJson("/admin/users"),
  ]) as [Series[], User[]];

  const totalSeries = series.length;
  const totalUsers = users.length;
  const totalVip = users.filter((u) => u.has_vip).length;
  const publishedChapters = series.reduce((acc, s) => acc + s.chapter_count, 0);

  const recentSeries = series.slice(0, 5);
  const recentUsers = users.slice(0, 5);

  return (
    <div className="space-y-6">
      {/* Header da página */}
      <div>
        <h2 className="text-2xl font-black text-slate-900">Visão geral</h2>
        <p className="text-slate-500">Métricas e atividades recentes da plataforma.</p>
      </div>

      {/* Cards de estatísticas */}
      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-4">
        <StatCard title="Séries" value={totalSeries} Icon={BookOpen} color="blue" />
        <StatCard title="Usuários" value={totalUsers} Icon={Users} color="green" />
        <StatCard title="VIP ativos" value={totalVip} Icon={Shield} color="amber" />
        <StatCard title="Capítulos publicados" value={publishedChapters} Icon={TrendingUp} color="slate" />
      </div>

      <div className="grid grid-cols-1 lg:grid-cols-3 gap-6">
        {/* Tabela de séries recentes */}
        <div className="lg:col-span-2 bg-white rounded-xl border border-slate-200 p-6">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-sm font-black text-slate-900">Séries recentes</h3>
            <Link href="/admin/series" className="text-sm font-medium text-blue-600 hover:text-blue-700">
              Ver todas →
            </Link>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-sm text-left">
              <thead>
                <tr className="border-b border-slate-200">
                  <th className="pb-3 text-xs font-semibold text-slate-500 uppercase">Título</th>
                  <th className="pb-3 text-xs font-semibold text-slate-500 uppercase">Slug</th>
                  <th className="pb-3 text-xs font-semibold text-slate-500 uppercase">Capítulos</th>
                  <th className="pb-3 text-xs font-semibold text-slate-500 uppercase">Acesso</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {recentSeries.map((s) => (
                  <tr key={s.id} className="hover:bg-slate-50/50">
                    <td className="py-3 text-sm text-slate-700">{s.title}</td>
                    <td className="py-3 text-xs text-slate-500">{s.slug}</td>
                    <td className="py-3 text-xs text-slate-600">{s.chapter_count}</td>
                    <td className="py-3">
                      <Badge variant={s.access === "public" ? "green" : "blue"}>
                        {s.access === "public" ? "Público" : s.access === "vip_only" ? "VIP" : "Early"}
                      </Badge>
                    </td>
                  </tr>
                ))}
                {recentSeries.length === 0 && (
                  <tr>
                    <td colSpan={4} className="py-8 text-center text-slate-400 text-sm">
                      Nenhuma série encontrada.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>

        {/* Tabela de usuários recentes */}
        <div className="bg-white rounded-xl border border-slate-200 p-6">
          <div className="flex items-center justify-between mb-4">
            <h3 className="text-sm font-black text-slate-900">Usuários recentes</h3>
            <Link href="/admin/users" className="text-sm font-medium text-blue-600 hover:text-blue-700">
              Ver todos →
            </Link>
          </div>
          <div className="overflow-x-auto">
            <table className="w-full text-sm text-left">
              <thead>
                <tr className="border-b border-slate-200">
                  <th className="pb-3 text-xs font-semibold text-slate-500 uppercase">E-mail</th>
                  <th className="pb-3 text-xs font-semibold text-slate-500 uppercase">VIP</th>
                  <th className="pb-3 text-xs font-semibold text-slate-500 uppercase">Criado em</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {recentUsers.map((u) => (
                  <tr key={u.id} className="hover:bg-slate-50/50">
                    <td className="py-3 text-sm text-slate-700">{u.email}</td>
                    <td className="py-3 text-xs">
                      {u.has_vip ? (
                        <Badge variant="green">VIP ativo</Badge>
                      ) : (
                        <Badge variant="gray">Sem VIP</Badge>
                      )}
                    </td>
                    <td className="py-3 text-xs text-slate-500">
                      {new Date(u.created_at).toLocaleDateString("pt-BR")}
                    </td>
                  </tr>
                ))}
                {recentUsers.length === 0 && (
                  <tr>
                    <td colSpan={3} className="py-8 text-center text-slate-400 text-sm">
                      Nenhum usuário encontrado.
                    </td>
                  </tr>
                )}
              </tbody>
            </table>
          </div>
        </div>
      </div>

      {/* Ações rápidas */}
      <div className="bg-white rounded-xl border border-slate-200 p-6">
        <h3 className="text-sm font-black text-slate-900 mb-4">Ações rápidas</h3>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-4">
          <Link
            href="/admin/series"
            className="flex items-center gap-3 p-4 rounded-lg border border-slate-200 hover:bg-slate-50 hover:border-blue-300 transition-all"
          >
            <div className="w-10 h-10 rounded-lg bg-blue-50 flex items-center justify-center text-blue-600">
              <BookOpen className="w-5 h-5" />
            </div>
            <div>
              <h4 className="text-sm font-semibold text-slate-900">Gerenciar Séries</h4>
              <p className="text-xs text-slate-500">Criar, editar e aprovar séries.</p>
            </div>
          </Link>
          <Link
            href="/admin/users"
            className="flex items-center gap-3 p-4 rounded-lg border border-slate-200 hover:bg-slate-50 hover:border-blue-300 transition-all"
          >
            <div className="w-10 h-10 rounded-lg bg-green-50 flex items-center justify-center text-green-600">
              <Users className="w-5 h-5" />
            </div>
            <div>
              <h4 className="text-sm font-semibold text-slate-900">Gerenciar Usuários</h4>
              <p className="text-xs text-slate-500">Visualizar contas e status VIP.</p>
            </div>
          </Link>
          <div className="flex items-center gap-3 p-4 rounded-lg border border-slate-200 hover:bg-slate-50 hover:border-amber-300 transition-all">
            <div className="w-10 h-10 rounded-lg bg-amber-50 flex items-center justify-center text-amber-600">
              <Shield className="w-5 h-5" />
            </div>
            <div>
              <h4 className="text-sm font-semibold text-slate-900">Conceder VIP</h4>
              <p className="text-xs text-slate-500">Ativar assinatura para usuários.</p>
            </div>
          </div>
        </div>
      </div>
    </div>
  );
}
