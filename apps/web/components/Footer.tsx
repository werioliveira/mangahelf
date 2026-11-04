import Link from "next/link";
import { Logo } from "./Header";

const footerLinks = [
  { label: "Sobre", href: "#" },
  { label: "Contato", href: "#" },
  { label: "DMCA", href: "#" },
  { label: "Termos", href: "#" },
  { label: "Privacidade", href: "#" },
];

export function Footer() {
  return (
    <footer className="border-t border-border mt-16">
      <div className="max-w-[1400px] mx-auto px-6 py-10">
        <div className="flex flex-col md:flex-row items-center justify-between gap-6">
          <div>
            <Logo size="text-xl" />
            <p className="text-xs text-muted-foreground mt-2">
              A melhor plataforma de manhwa e manga em português.
            </p>
          </div>
          <div className="flex flex-wrap gap-6 text-sm text-muted-foreground">
            {footerLinks.map((l) => (
              <Link key={l.label} href={l.href} className="hover:text-foreground transition-colors">
                {l.label}
              </Link>
            ))}
          </div>
          <p className="text-xs text-muted-foreground/50">© 2026 MangaShelf. Todos os direitos reservados.</p>
        </div>
      </div>
    </footer>
  );
}
