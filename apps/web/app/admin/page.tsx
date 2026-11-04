import type { Metadata } from "next";
import Link from "next/link";
import { Shield } from "lucide-react";
import { btnOutline } from "@/lib/ui";

export const metadata: Metadata = {
  title: "Admin",
  description: "Painel administrativo do MangaShelf.",
  robots: { index: false },
};

export default function AdminPage() {
  // shortcut: stub de navegação; o painel real depende do backend admin (Go) e auth
  return (
    <section className="relative py-16 md:py-24">
      <div className="max-w-md mx-auto px-6 text-center">
        <div className="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-primary/10 border border-primary/20 mb-6">
          <Shield className="w-8 h-8 text-primary" />
        </div>
        <h1 className="font-heading text-2xl font-black text-foreground mb-2">Painel em construção</h1>
        <p className="text-sm text-muted-foreground mb-6">
          A área administrativa chega junto com a API de moderação e upload de capítulos.
        </p>
        <Link href="/" className={btnOutline}>
          Voltar ao início
        </Link>
      </div>
    </section>
  );
}
