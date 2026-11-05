import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Admin — MangaShelf",
  description: "Painel administrativo do MangaShelf.",
  robots: { index: false },
};

export default function AdminLayout({
  children,
}: {
  children: React.ReactNode;
}) {
  return (
    <div className="flex h-screen bg-slate-50 text-slate-900">
      {/* Sidebar */}
      <aside className="hidden lg:flex flex-col w-64 bg-slate-900 text-slate-200 shrink-0">
        <div className="h-16 flex items-center px-6 border-b border-slate-800">
          <span className="text-xl font-black tracking-tighter text-white">MangaShelf</span>
          <span className="ml-1 text-xs font-semibold text-slate-400 uppercase tracking-wider">admin</span>
        </div>
        <nav className="flex-1 px-4 space-y-1">
          <a
            href="/admin"
            className="flex items-center gap-3 px-4 py-3 rounded-lg bg-slate-800 text-white"
          >
            <span className="w-5 h-5">📊</span>
            <span className="font-medium">Dashboard</span>
          </a>
          <a
            href="/admin/series"
            className="flex items-center gap-3 px-4 py-3 rounded-lg text-slate-300 hover:bg-slate-800 hover:text-white transition-all"
          >
            <span className="w-5 h-5">📚</span>
            <span className="font-medium">Séries</span>
          </a>
          <a
            href="/admin/chapters"
            className="flex items-center gap-3 px-4 py-3 rounded-lg text-slate-300 hover:bg-slate-800 hover:text-white transition-all"
          >
            <span className="w-5 h-5">📖</span>
            <span className="font-medium">Capítulos</span>
          </a>
          <a
            href="/admin/users"
            className="flex items-center gap-3 px-4 py-3 rounded-lg text-slate-300 hover:bg-slate-800 hover:text-white transition-all"
          >
            <span className="w-5 h-5">👥</span>
            <span className="font-medium">Usuários</span>
          </a>
          <a
            href="/admin/vip"
            className="flex items-center gap-3 px-4 py-3 rounded-lg text-slate-300 hover:bg-slate-800 hover:text-white transition-all"
          >
            <span className="w-5 h-5">💎</span>
            <span className="font-medium">VIP</span>
          </a>
        </nav>
        <div className="p-4 border-t border-slate-800">
          <a
            href="/auth/logout"
            className="flex items-center gap-3 px-4 py-3 rounded-lg text-slate-400 hover:text-white transition-all"
          >
            <span className="w-5 h-5">🚪</span>
            <span className="font-medium">Sair</span>
          </a>
        </div>
      </aside>

      {/* Conteúdo principal */}
      <div className="flex-1 flex flex-col overflow-hidden">
        {/* Header */}
        <header className="h-16 border-b border-slate-200 bg-white/90 backdrop-blur flex items-center justify-between px-6 shrink-0 z-10">
          <div className="flex items-center gap-4">
            <button className="lg:hidden p-2 rounded-md text-slate-600 hover:bg-slate-100">
              <span className="w-5 h-5">☰</span>
            </button>
            <h1 className="text-sm font-semibold text-slate-700">Dashboard</h1>
          </div>
          <div className="flex items-center gap-4">
            <div className="hidden ml-4 flex items-center gap-2 px-3 py-1.5 rounded-full bg-slate-100 text-xs font-medium text-slate-600">
              <span className="w-2 h-2 rounded-full bg-green-500"></span>
              <span>Online</span>
            </div>
            <div className="w-8 h-8 rounded-full bg-slate-200 overflow-hidden border border-white">
              <img src="/covers/solo-leveling.png" alt="Admin" className="w-full h-full object-cover" />
            </div>
          </div>
        </header>

        <main className="flex-1 overflow-y-auto p-6">
          {children}
        </main>
      </div>
    </div>
  );
}
