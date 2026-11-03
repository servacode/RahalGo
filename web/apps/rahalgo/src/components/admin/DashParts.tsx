"use client";

import type { ComponentType, ReactNode } from "react";
import { Stat, type StatTone } from "@rahalgo/ui";

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

/** **اسمُ النبرة القديم ← نبرةُ العُدّة** — أخضرُ تمّ · أصفرُ ينتظر · أحمرُ مشكلة · رماديٌّ هادئ. */
const STAT_TONE: Record<TileTone, StatTone> = {
  default: "neutral",
  muted: "muted",
  danger: "danger",
  warning: "attention",
  accent: "neutral",
  success: "good",
};

/**
 * **بطاقةٌ صغيرةٌ تُضغط** — أيقونةٌ في دائرةٍ بلون النبرة، ورقمٌ ولفظ.
 *
 * **واللفظُ على سطرين لا يُقطع بنقاط** (تدقيقُ الرئيسيّة ٢٠٢٦-١٠-٢٧): على الجوّال بطاقتان في السطر،
 * **و«شكاوى متأخّرة…» تصير «شكاوى متأ…» فلا تُفهم.** **وبلا `onClick` تُرسم رقماً لا زرّاً** — بطاقةٌ
 * تبدو زرّاً ولا تفعل شيئاً أسوأُ من رقمٍ هادئ. و`extra` شاراتٌ صغيرةٌ تحت اللفظ (اختياريّة).
 */
export function Tile({
  icon,
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
  // **مربّعٌ واحدٌ في اللوحة كلِّها** — `Stat` من العُدّة؛ هذا الاسمُ القديمُ يمرّ إليه فلا يبقى شكلان.
  return (
    <Stat
      icon={icon}
      label={label}
      value={value}
      tone={STAT_TONE[tone]}
      meta={
        sub || extra ? (
          <>
            {sub}
            {extra && <span className="ms-1 inline-flex flex-wrap gap-1">{extra}</span>}
          </>
        ) : undefined
      }
      onOpen={onClick}
    />
  );
}

