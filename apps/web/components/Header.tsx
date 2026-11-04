import Link from "next/link";
import { Search, Bell, User, Shield, Crown } from "lucide-react";
import { MobileMenu } from "./MobileMenu";

const links = [
  { href: "/", label: "Home" },
  { href: "/library", label: "Biblioteca" },
  { href: "/favorites", label: "Favoritos" },
  { href: "/ranking", label: "Ranking 🏆" },
];

export function Logo({ size = "text-xl md:text-2xl" }: { size?: string }) {
  return (
    <div className="relative">
      <span className={`font-heading ${size} font-black tracking-tight text-foreground`}>MANGA</span>
      <span className={`font-heading ${size} font-black tracking-tight text-primary`}>SHELF</span>
      <div className="absolute -bottom-1 left-0 w-full h-0.5 bg-gradient-to-r from-primary via-accent to-transparent rounded-full" />
    </div>
  );
}

const iconBtn =
  "p-2 text-muted-foreground hover:text-foreground hover:bg-secondary rounded-xl transition-all";

export function Header() {
  return (
    <header className="sticky top-0 z-50 w-full">
      <div className="absolute inset-0 bg-gradient-to-b from-background via-background/95 to-background/80 backdrop-blur-xl" />
      <div className="relative max-w-[1400px] mx-auto px-4 md:px-6">
        <div className="flex items-center justify-between h-16 gap-3">
          <Link className="flex items-center gap-2 flex-shrink-0" href="/">
            <Logo />
          </Link>

          <nav className="hidden md:flex items-center gap-0.5 flex-1 justify-center">
            {links.map((l) => (
              <Link
                key={l.href}
                href={l.href}
                className="relative px-3 py-2 text-sm font-medium transition-all duration-200 rounded-lg group text-muted-foreground hover:text-foreground"
              >
                <span className="relative z-10 flex items-center gap-1.5">{l.label}</span>
                <div className="absolute inset-0 rounded-lg bg-secondary/0 group-hover:bg-secondary/80 transition-all duration-200" />
              </Link>
            ))}
            <Link
              href="/vip"
              className="relative px-3 py-2 text-sm font-medium transition-all duration-200 rounded-lg group text-accent hover:text-accent"
            >
              <span className="relative z-10 flex items-center gap-1.5">
                <Crown className="w-3.5 h-3.5" />
                VIP
              </span>
              <div className="absolute inset-0 rounded-lg bg-secondary/0 group-hover:bg-secondary/80 transition-all duration-200" />
            </Link>
          </nav>

          <div className="flex items-center gap-1 flex-shrink-0">
            <Link href="/library" aria-label="Buscar" className={iconBtn}>
              <Search className="w-4 h-4" />
            </Link>
            <button aria-label="Notificações" className={iconBtn}>
              <Bell className="w-4 h-4" />
            </button>
            <Link href="/profile" aria-label="Perfil" className={iconBtn}>
              <User className="w-4 h-4" />
            </Link>
            <Link
              href="/admin"
              className="hidden sm:flex items-center gap-1.5 px-3 py-1.5 bg-secondary hover:bg-secondary/80 rounded-xl transition-all"
            >
              <Shield className="w-3.5 h-3.5 text-primary" />
              <span className="text-xs font-semibold text-foreground">Admin</span>
            </Link>
            <MobileMenu />
          </div>
        </div>
      </div>
      <div className="absolute bottom-0 left-0 right-0 h-px bg-gradient-to-r from-transparent via-border to-transparent" />
    </header>
  );
}
