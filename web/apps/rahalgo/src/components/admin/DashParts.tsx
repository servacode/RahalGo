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
  actions,
  className = "",
  children,
}: {
  title: string;
  note?: string;
  /** **زرٌّ صغيرٌ على طرف العنوان** — «عرض الكل» مثلاً. اختياريّ. */
  actions?: ReactNode;
  className?: string;
  children: ReactNode;
}) {
  return (
    <section className={`surface p-4 ${className}`}>
      <div className="mb-3 flex flex-wrap items-start justify-between gap-2">
        <h2 className="heading-card flex min-w-0 flex-wrap items-baseline gap-x-2">
          {title}
          {note && <span className="text-xs font-normal text-ink-muted">{note}</span>}
        </h2>
        {actions && <div className="flex shrink-0 items-center gap-2">{actions}</div>}
      </div>
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

/**
 * **بطاقةٌ صغيرةٌ تُضغط** — أيقونةٌ في دائرةٍ بلون النبرة، ورقمٌ ولفظ.
 *
 * **واللفظُ على سطرين لا يُقطع بنقاط** (تدقيقُ الرئيسيّة ٢٠٢٦-١٠-٢٧): على الجوّال بطاقتان في السطر،
 * **و«شكاوى متأخّرة…» تصير «شكاوى متأ…» فلا تُفهم.** **وبلا `onClick` تُرسم رقماً لا زرّاً** — بطاقةٌ
 * تبدو زرّاً ولا تفعل شيئاً أسوأُ من رقمٍ هادئ. و`extra` شاراتٌ صغيرةٌ تحت اللفظ (اختياريّة).
 */
export function Tile({
  icon: Icon,
  label,
  value,
  sub,
  tone = "default",
  onClick,
  extra,
}: {
  icon: ComponentType<{ size?: number; className?: string }>;
  label: string;
  value: string;
  sub?: string;
  tone?: TileTone;
  onClick?: () => void;
  extra?: ReactNode;
}) {
  const cls = `flex items-center gap-3 rounded-card border p-3 text-start ${
    onClick ? "transition-colors hover:bg-row-hover" : ""
  } ${tone === "danger" ? "border-danger" : tone === "warning" ? "border-warning" : "border-line-soft"}`;
  const body = (
    <>
      <span className={`inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full ${TILE_ICON[tone]}`}>
        <Icon size={18} />
      </span>
      <span className="min-w-0 flex-1">
        <span className={`figure block leading-none ${tone === "muted" ? "text-ink-muted" : ""}`}>{value}</span>
        <span className="mt-1 line-clamp-2 text-xs text-ink-muted">{label}</span>
        {sub && <span className="line-clamp-2 text-2xs text-ink-muted">{sub}</span>}
        {extra && <span className="mt-1 flex flex-wrap gap-1">{extra}</span>}
      </span>
    </>
  );
  if (!onClick) return <div className={cls}>{body}</div>;
  return (
    <button type="button" onClick={onClick} className={cls}>
      {body}
    </button>
  );
}

