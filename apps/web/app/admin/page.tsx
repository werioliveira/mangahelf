import type { Metadata } from "next";
import Link from "next/link";
import { Shield, Users, List, BookOpen } from "lucide-react";
import { btnOutline, btnPrimary } from "@/lib/ui";

export const metadata: Metadata = {
  title: "Admin",
  description: "Painel administrativo do MangaShelf.",
  robots: { index: false },
};

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

const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

async function fetchJson(path: string, init?: RequestInit) {
  const res = await fetch(`${API_URL}${path}`, {
    ...init,
    headers: { ...init?.headers, "Content-Type": "application/json" },
    cache: "no-store",
  });
  if (!res.ok) throw new Error(`API ${path}: ${res.status}`);
  return res.json();
}

export default function AdminPage() {
  return (
    <section className="relative py-16 md:py-24">
      <div className="max-w-7xl mx-auto px-6">
        <div className="flex items-center justify-between mb-8">
          <div className="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-primary/10 border border-primary/20 mb-4">
            <Shield className="w-8 h-8 text-primary" />
          </div>
          <div>
            <h1 className="font-heading text-2xl font-black text-foreground mb-2">Painel Administrativo</h1>
            <p className="text-sm text-muted-foreground">Gerencie séries, capítulos, usuários e VIP.</p>
          </div>
          <Link href="/" className={btnOutline}>
            Voltar ao início
          </Link>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
          {/* Séries */}
          <div className="bg-card rounded-xl border border-border p-6">
            <div className="flex items-center justify-between mb-4">
              <h2 className="font-semibold text-foreground">Séries</h2>
              <Link href="/library" className={btnOutline}><List className="w-4 h-4 mr-2" /> Ver todas</Link>
            </div>
            <div id="series-list" className="space-y-3">
              <div className="text-center text-muted-foreground py-4">Carregando...</div>
            </div>
          </div>

          {/* Usuários */}
          <div className="bg-card rounded-xl border border-border p-6">
            <div className="flex items-center justify-between mb-4">
              <h2 className="font-semibold text-foreground">Usuários</h2>
              <Link href="/" className={btnOutline}><Users className="w-4 h-4 mr-2" /> Ver todos</Link>
            </div>
            <div id="users-list" className="space-y-3">
              <div className="text-center text-muted-foreground py-4">Carregando...</div>
            </div>
          </div>

          {/* Ações rápidas */}
          <div className="bg-card rounded-xl border border-border p-6">
            <div className="flex items-center justify-between mb-4">
              <h2 className="font-semibold text-foreground">Ações rápidas</h2>
            </div>
            <div className="space-y-3">
              <div className="flex items-center justify-between p-3 bg-secondary/50 rounded-lg border border-border/30">
                <div>
                  <h3 className="text-sm font-semibold text-foreground">Publicar capítulo</h3>
                  <p className="text-xs text-muted-foreground">Aprove capítulos pendentes para leitura pública.</p>
                </div>
                <button className={btnPrimary}><BookOpen className="w-4 h-4 mr-2" /> Publicar</button>
              </div>
              <div className="flex items-center justify-between p-3 bg-secondary/50 rounded-lg border border-border/30">
                <div>
                  <h3 className="text-sm font-semibold text-foreground">Conceder VIP</h3>
                  <p className="text-xs text-muted-foreground">Ative assinatura VIP para usuários.</p>
                </div>
                <button className={btnPrimary}><Shield className="w-4 h-4 mr-2" /> Conceder</button>
              </div>
            </div>
          </div>
        </div>
      </div>
    </section>
  );
}
