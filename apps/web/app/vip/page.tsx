import type { Metadata } from "next";
import { Crown, Check, Sparkles } from "lucide-react";
import { btnPrimary } from "@/lib/ui";

export const metadata: Metadata = {
  title: "VIP",
  description: "Apoie o MangaShelf e desbloqueie vantagens.",
};

const benefits = [
  "Leitura sem anúncios",
  "Acesso antecipado a capítulos",
  "Qualidade de imagem em alta definição",
  "Apoio direto aos tradutores",
];

export default function VipPage() {
  return (
    <section className="relative py-12 md:py-20">
      <div className="max-w-[1400px] mx-auto px-6">
        <div className="relative max-w-lg mx-auto bg-card rounded-2xl border border-primary/20 overflow-hidden shadow-2xl shadow-primary/10">
          <div className="absolute -inset-16 bg-primary/5 blur-3xl -z-0" />
          <div className="relative p-8 md:p-10 text-center">
            <div className="inline-flex items-center justify-center w-16 h-16 rounded-2xl bg-accent/10 border border-accent/20 mb-6">
              <Crown className="w-8 h-8 text-accent" />
            </div>
            <h1 className="font-heading text-3xl md:text-4xl font-black text-foreground mb-2">MangaShelf VIP</h1>
            <p className="text-sm text-muted-foreground mb-6">
              Apoie a plataforma e a comunidade de tradução.
            </p>
            <div className="space-y-2.5 mb-8 text-left">
              {benefits.map((b) => (
                <div key={b} className="flex items-center gap-2.5 text-sm">
                  <span className="flex items-center justify-center w-5 h-5 rounded-full bg-primary/15 border border-primary/25">
                    <Check className="w-3 h-3 text-primary" />
                  </span>
                  <span className="text-foreground">{b}</span>
                </div>
              ))}
            </div>
            <div className="flex items-end justify-center gap-1 mb-6">
              <span className="font-heading text-5xl font-black text-primary">R$ 12</span>
              <span className="text-sm text-muted-foreground pb-1.5">/mês</span>
            </div>
            {/* shortcut: checkout real entra com o módulo de billing da API */}
            <button className={`${btnPrimary} w-full !h-11`}>
              <Sparkles className="w-4 h-4" />
              Assinar VIP
            </button>
          </div>
        </div>
      </div>
    </section>
  );
}
