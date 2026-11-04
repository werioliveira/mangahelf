import type { Metadata } from "next";
import { User, BookOpen, Heart, History } from "lucide-react";
import { SectionHeader } from "@/components/SectionHeader";
import { RankRow } from "@/components/RankRow";
import { bySlug } from "@/lib/data";

export const metadata: Metadata = { title: "Perfil", description: "Sua conta no MangaShelf." };

const stats = [
  { icon: BookOpen, label: "Lidos", value: "24" },
  { icon: Heart, label: "Favoritos", value: "4" },
  { icon: History, label: "Capítulos", value: "312" },
];

export default function ProfilePage() {
  const recent = ["solo-leveling", "omniscient-reader"].map((s) => bySlug(s)!);

  return (
    <section className="relative py-8 md:py-12">
      <div className="max-w-[1400px] mx-auto px-6">
        <div className="flex flex-col sm:flex-row items-center gap-5 mb-8">
          <div className="flex items-center justify-center w-20 h-20 rounded-2xl bg-primary/10 border border-primary/20">
            <User className="w-10 h-10 text-primary" />
          </div>
          <div className="text-center sm:text-left">
            <h1 className="font-heading text-2xl md:text-3xl font-black text-foreground">Leitor</h1>
            <p className="text-sm text-muted-foreground mt-1">Membro desde 2026 · plano gratuito</p>
          </div>
        </div>

        <div className="grid grid-cols-1 sm:grid-cols-3 gap-3 mb-10">
          {stats.map(({ icon: Icon, label, value }) => (
            <div key={label} className="bg-card border border-border rounded-xl p-4 flex items-center gap-3">
              <div className="flex items-center justify-center w-10 h-10 rounded-xl bg-secondary">
                <Icon className="w-4 h-4 text-accent" />
              </div>
              <div>
                <p className="font-heading text-xl font-black text-foreground">{value}</p>
                <p className="text-xs text-muted-foreground">{label}</p>
              </div>
            </div>
          ))}
        </div>

        <SectionHeader icon={History} tone="accent" title="CONTINUAR LENDO" subtitle="Seu histórico recente" />
        <div className="max-w-3xl bg-card rounded-xl border border-border overflow-hidden">
          {recent.map((s, i) => (
            <RankRow key={s.slug} series={s} rank={i + 1} />
          ))}
        </div>
      </div>
    </section>
  );
}
