"use client";

import type { ComponentType, ReactNode } from "react";

/*
 * **قطعُ لوحاتِ الأرقام المشتركة** (قرارُ المالك ٢٠٢٦-١٠-٢٦: «التصميم الجديد») — لوحةُ قسمٍ ببطاقةٍ واحدة،
 * وبطاقةٌ صغيرةٌ تُضغط بأيقونةٍ ملوّنةٍ ورقمٍ ولفظ. **للرئيسيّة والخزينة وما يأتي** — شكلٌ واحدٌ في اللوحة.
 */

/** **لوحةُ قسمٍ في الرئيسيّة** — عنوانٌ وملاحظةٌ صغيرةٌ ثمّ المحتوى في بطاقةٍ واحدة. */
export function Panel({
  title,
  note,
  className = "",
  children,
}: {
  title: string;
  note?: string;
  className?: string;
  children: ReactNode;
}) {
  return (
    <section className={`surface p-4 ${className}`}>
      <h2 className="heading-card mb-3 flex flex-wrap items-baseline gap-2">
        {title}
        {note && <span className="text-xs font-normal text-ink-muted">{note}</span>}
      </h2>
      {children}
    </section>
  );
}

export type TileTone = "default" | "muted" | "danger" | "warning" | "accent" | "success";

const TILE_ICON: Record<TileTone, string> = {
  default: "bg-field text-ink",
  muted: "bg-field text-ink-muted",
  danger: "bg-danger-solid text-on-solid",
  warning: "bg-warning-solid text-on-solid",
  accent: "bg-accent text-on-bright",
  success: "bg-success-solid text-on-solid",
};

/** **بطاقةٌ صغيرةٌ تُضغط** — أيقونةٌ في دائرةٍ بلون النبرة، ورقمٌ ولفظ. */
export function Tile({
  icon: Icon,
  label,
  value,
  sub,
  tone = "default",
  onClick,
}: {
  icon: ComponentType<{ size?: number; className?: string }>;
  label: string;
  value: string;
  sub?: string;
  tone?: TileTone;
  onClick?: () => void;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      className={`flex items-center gap-3 rounded-card border p-3 text-start transition-colors hover:bg-row-hover ${
        tone === "danger" ? "border-danger" : tone === "warning" ? "border-warning" : "border-line-soft"
      }`}
    >
      <span className={`inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full ${TILE_ICON[tone]}`}>
        <Icon size={18} />
      </span>
      <span className="min-w-0">
        <span className={`figure block leading-none ${tone === "muted" ? "text-ink-muted" : ""}`}>{value}</span>
        <span className="mt-1 block truncate text-xs text-ink-muted">{label}</span>
        {sub && <span className="block truncate text-2xs text-ink-muted">{sub}</span>}
      </span>
    </button>
  );
}

