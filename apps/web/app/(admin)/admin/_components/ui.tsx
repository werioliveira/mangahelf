"use client";

import { useCallback, useEffect, useState } from "react";

export const API_URL = process.env.NEXT_PUBLIC_API_URL || "http://localhost:8080";

export const fmtDate = (v: string) => {
  const d = new Date(v);
  return isNaN(d.getTime()) ? "—" : d.toLocaleDateString("pt-BR", { day: "2-digit", month: "short", year: "numeric" });
};

const PATHS: Record<string, string> = {
  plus: "M12 5v14M5 12h14",
  search: "m21 21-4.3-4.3M17 11a6 6 0 1 1-12 0 6 6 0 0 1 12 0",
  edit: "M12 20h9M16.5 3.5a2.1 2.1 0 0 1 3 3L7 19l-4 1 1-4z",
  trash: "M3 6h18M8 6V4a1 1 0 0 1 1-1h6a1 1 0 0 1 1 1v2M19 6l-1 14a2 2 0 0 1-2 2H8a2 2 0 0 1-2-2L5 6M10 11v6M14 11v6",
  x: "M18 6 6 18M6 6l12 12",
  layers: "m12 3 9 5-9 5-9-5zM3 13l9 5 9-5",
  book: "M4 5a2 2 0 0 1 2-2h13v16H6a2 2 0 0 0-2 2zM4 19a2 2 0 0 1 2-2h13",
  check: "M20 6 9 17l-5-5",
  alert: "M12 9v4M12 17h.01M10.3 3.9 1.8 18a2 2 0 0 0 1.7 3h17a2 2 0 0 0 1.7-3L13.7 3.9a2 2 0 0 0-3.4 0",
};

export const Icon = ({ name, className = "h-4 w-4" }: { name: string; className?: string }) => (
  <svg viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.8" strokeLinecap="round" strokeLinejoin="round" className={className}>
    <path d={PATHS[name]} />
  </svg>
);

export type Tone = "green" | "violet" | "sky" | "amber" | "red" | "gray";
const TONES: Record<Tone, string> = {
  green: "bg-emerald-400/10 text-emerald-300 ring-emerald-400/20",
  violet: "bg-violet-400/10 text-violet-300 ring-violet-400/20",
  sky: "bg-sky-400/10 text-sky-300 ring-sky-400/20",
  amber: "bg-amber-400/10 text-amber-300 ring-amber-400/20",
  red: "bg-red-400/10 text-red-300 ring-red-400/20",
  gray: "bg-white/5 text-slate-400 ring-white/10",
};

export const Badge = ({ tone = "gray", children }: { tone?: Tone; children: React.ReactNode }) => (
  <span className={`inline-flex items-center rounded-full px-2.5 py-0.5 text-xs font-medium ring-1 ring-inset ${TONES[tone]}`}>
    {children}
  </span>
);

export const inputCls =
  "w-full rounded-lg border border-white/10 bg-[#0b1020] px-3 py-2.5 text-sm text-slate-100 outline-none transition-colors placeholder:text-slate-600 focus:border-indigo-400/60 focus:ring-2 focus:ring-indigo-400/20 [color-scheme:dark]";

const BTN = {
  primary: "bg-indigo-500 text-white hover:bg-indigo-400 disabled:bg-indigo-500/50",
  ghost: "border border-white/10 text-slate-300 hover:bg-white/5",
  danger: "bg-red-500 text-white hover:bg-red-400 disabled:bg-red-500/50",
};

export const Btn = ({
  variant = "primary",
  className = "",
  ...props
}: React.ButtonHTMLAttributes<HTMLButtonElement> & { variant?: keyof typeof BTN }) => (
  <button
    {...props}
    className={`inline-flex items-center justify-center gap-2 rounded-lg px-4 py-2.5 text-sm font-medium transition-colors disabled:cursor-not-allowed ${BTN[variant]} ${className}`}
  />
);

export const Field = ({ label, hint, children }: { label: string; hint?: string; children: React.ReactNode }) => (
  <label className="block">
    <span className="mb-1.5 block text-sm font-medium text-slate-300">{label}</span>
    {children}
    {hint && <span className="mt-1.5 block text-xs text-slate-500">{hint}</span>}
  </label>
);

export function Modal({
  title,
  onClose,
  children,
  footer,
}: {
  title: string;
  onClose: () => void;
  children: React.ReactNode;
  footer: React.ReactNode;
}) {
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => e.key === "Escape" && onClose();
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, [onClose]);

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center p-4">
      <div className="absolute inset-0 bg-black/60 backdrop-blur-sm" onClick={onClose} />
      <div role="dialog" aria-modal="true" className="relative w-full max-w-lg rounded-2xl border border-white/10 bg-[#111830] shadow-2xl">
        <div className="flex items-center justify-between border-b border-white/5 px-6 py-4">
          <h3 className="text-base font-semibold text-white">{title}</h3>
          <button onClick={onClose} aria-label="Fechar" className="rounded-md p-1 text-slate-500 hover:bg-white/5 hover:text-white">
            <Icon name="x" className="h-5 w-5" />
          </button>
        </div>
        <div className="space-y-4 px-6 py-5">{children}</div>
        <div className="flex justify-end gap-2 border-t border-white/5 px-6 py-4">{footer}</div>
      </div>
    </div>
  );
}

export function Segmented<T extends string>({
  options,
  value,
  onChange,
}: {
  options: { value: T; label: string; count?: number }[];
  value: T;
  onChange: (v: T) => void;
}) {
  return (
    <div className="flex flex-wrap gap-1 rounded-xl border border-white/[0.06] bg-[#111830] p-1">
      {options.map((o) => (
        <button
          key={o.value}
          onClick={() => onChange(o.value)}
          className={`rounded-lg px-3 py-1.5 text-sm transition-colors ${
            value === o.value ? "bg-indigo-500/20 font-medium text-indigo-200" : "text-slate-400 hover:text-white"
          }`}
        >
          {o.label}
          {o.count !== undefined && <span className="ml-1.5 text-xs text-slate-500">{o.count}</span>}
        </button>
      ))}
    </div>
  );
}

export const SearchBox = ({
  value,
  onChange,
  placeholder,
}: {
  value: string;
  onChange: (v: string) => void;
  placeholder: string;
}) => (
  <div className="relative min-w-[240px] flex-1 sm:max-w-xs">
    <Icon name="search" className="absolute left-3 top-1/2 h-4 w-4 -translate-y-1/2 text-slate-500" />
    <input
      value={value}
      onChange={(e) => onChange(e.target.value)}
      placeholder={placeholder}
      className={`${inputCls} !rounded-xl !bg-[#111830] pl-9`}
    />
  </div>
);

export const PageHeader = ({
  title,
  subtitle,
  action,
}: {
  title: string;
  subtitle: string;
  action?: React.ReactNode;
}) => (
  <div className="flex flex-wrap items-end justify-between gap-4">
    <div>
      <h2 className="text-2xl font-bold tracking-tight text-white">{title}</h2>
      <p className="mt-1 text-sm text-slate-500">{subtitle}</p>
    </div>
    {action}
  </div>
);

export const Spinner = ({ label }: { label: string }) => (
  <div className="flex items-center justify-center gap-3 py-24 text-sm text-slate-500">
    <div className="h-5 w-5 animate-spin rounded-full border-2 border-indigo-400 border-t-transparent" />
    {label}
  </div>
);

export const Empty = ({
  icon,
  title,
  text,
  action,
}: {
  icon: string;
  title: string;
  text: string;
  action?: React.ReactNode;
}) => (
  <div className="flex flex-col items-center px-6 py-16 text-center">
    <div className="mb-4 grid h-12 w-12 place-items-center rounded-full bg-white/5 text-slate-400">
      <Icon name={icon} className="h-6 w-6" />
    </div>
    <p className="font-medium text-slate-200">{title}</p>
    <p className="mt-1 max-w-sm text-sm text-slate-500">{text}</p>
    {action && <div className="mt-5">{action}</div>}
  </div>
);

export function useToast() {
  const [toast, setToast] = useState<{ msg: string; ok: boolean } | null>(null);
  const show = useCallback((msg: string, ok = true) => {
    setToast({ msg, ok });
    setTimeout(() => setToast(null), 3500);
  }, []);
  const node = toast ? (
    <div
      role="status"
      className={`fixed bottom-6 right-6 z-[60] flex items-center gap-2 rounded-xl border px-4 py-3 text-sm shadow-xl ${
        toast.ok
          ? "border-emerald-400/20 bg-[#0f2a22] text-emerald-200"
          : "border-red-400/20 bg-[#2a1118] text-red-200"
      }`}
    >
      <Icon name={toast.ok ? "check" : "alert"} />
      {toast.msg}
    </div>
  ) : null;
  return { node, show };
}

export const Panel = ({ children, className }: { children: React.ReactNode; className?: string }) => (
  <div className={`overflow-hidden rounded-2xl border border-white/[0.06] bg-[#111830] ${className ?? ""}`}>{children}</div>
);

export const StatCard = ({
  label,
  value,
  iconName,
  tone = "indigo",
  hint,
}: {
  label: string;
  value: string | number;
  iconName: string;
  tone?: "indigo" | "sky" | "emerald" | "amber";
  hint?: string;
}) => {
  const tones: Record<string, { bg: string; text: string }> = {
    indigo: { bg: "bg-indigo-500/10", text: "text-indigo-300" },
    sky: { bg: "bg-sky-500/10", text: "text-sky-300" },
    emerald: { bg: "bg-emerald-500/10", text: "text-emerald-300" },
    amber: { bg: "bg-amber-500/10", text: "text-amber-300" },
  };
  const t = tones[tone] || tones.indigo;
  return (
    <div className="rounded-2xl border border-white/[0.06] bg-[#111830] p-5">
      <div className="mb-5 grid h-11 w-11 place-items-center rounded-full ${t.bg} ${t.text}">
        <Icon name={iconName} className="w-5 h-5" />
      </div>
      <p className="text-[28px] font-bold leading-none tracking-tight text-white">{value}</p>
      <p className="mt-2 text-sm font-semibold text-slate-400">{label}</p>
      {hint && <p className="mt-1 text-xs text-slate-500">{hint}</p>}
    </div>
  );
};

export const Th = ({ children, right }: { children?: React.ReactNode; right?: boolean }) => (
  <th className={`px-5 py-3 text-xs font-medium text-slate-500 ${right ? "text-right" : ""}`}>{children}</th>
);

export const errMsg = (e: unknown, fallback: string) => (e instanceof Error && e.message ? e.message : fallback);