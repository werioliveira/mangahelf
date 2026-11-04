import type { LucideIcon } from "lucide-react";

const tones = {
  primary: "bg-primary/10 border-primary/20 text-primary",
  accent: "bg-accent/10 border-accent/20 text-accent",
  destructive: "bg-destructive/10 border-destructive/20 text-destructive",
} as const;

export function SectionHeader({
  icon: Icon,
  tone = "primary",
  title,
  subtitle,
  actions,
}: {
  icon: LucideIcon;
  tone?: keyof typeof tones;
  title: string;
  subtitle?: string;
  actions?: React.ReactNode;
}) {
  return (
    <div className="flex items-center justify-between mb-6">
      <div className="flex items-center gap-3">
        <div className={`flex items-center justify-center w-10 h-10 rounded-xl border ${tones[tone]}`}>
          <Icon className="w-5 h-5" />
        </div>
        <div>
          <h2 className="font-heading text-2xl md:text-3xl font-black text-foreground tracking-tight">{title}</h2>
          {subtitle && <p className="text-xs text-muted-foreground mt-0.5">{subtitle}</p>}
        </div>
      </div>
      {actions}
    </div>
  );
}

export function Divider() {
  return (
    <div className="max-w-[1400px] mx-auto px-6">
      <div className="h-px bg-gradient-to-r from-transparent via-border to-transparent" />
    </div>
  );
}
