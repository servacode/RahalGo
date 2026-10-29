"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **قطعُ شاشات أدوات الأصناف — مرآةُ `MenuTools.kt` و`ItemEditor.kt`**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (تصميمُ ٢٠٢٦-١٠-٢٩.) رأسٌ بسهم رجوعٍ واضح، ومربّعُ اختيارٍ بأيقونة، وزرّا
 * «نسبة / مبلغ»، ودائرتا العملة، ورقاقةٌ تُقرأ. **ولا سطرَ شرح** — «ما في داعي
 * للشرح».
 */

import Link from "next/link";
import type { ComponentType, ReactNode } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { IconCheck, IconPrev } from "@rahalgo/ui";

const m = getMessages(defaultLocale);

/** **رأسُ شاشة أداة** — سهمُ رجوعٍ (٤٨) ثمّ العنوان، وما يُضاف في الطرف. */
export function ToolHeader({
  title,
  href,
  onBack,
  end,
}: {
  title: string;
  href?: string;
  onBack?: () => void;
  end?: ReactNode;
}) {
  const cls =
    "flex h-12 w-12 shrink-0 items-center justify-center rounded-control text-ink hover:bg-row-hover";
  return (
    <header className="flex items-center gap-1">
      {href ? (
        <Link href={href} aria-label={m.common.back} title={m.common.back} className={cls} data-testid="tool-back">
          <IconPrev size={24} />
        </Link>
      ) : onBack ? (
        <button type="button" onClick={onBack} aria-label={m.common.back} title={m.common.back} className={cls} data-testid="tool-back">
          <IconPrev size={24} />
        </button>
      ) : null}
      <h2 className="heading-page min-w-0 flex-1 truncate">{title}</h2>
      {end}
    </header>
  );
}

/**
 * **شريطُ حفظٍ ثابتٌ أسفلَ الشاشة** — «حفظ / إلغاء» لا ينزل إليه صاحبُ المتجر
 * (مراجعةُ التصميم ٢٠٢٦-١٠-٢٩، كمعالج الإعداد). **وعلى الجوّال فوق الشريط
 * السفليّ** لا تحته — كسلّة الزبون.
 */
export function StickyBar({ children }: { children: ReactNode }) {
  return (
    <div className="surface sticky bottom-[calc(3.75rem+env(safe-area-inset-bottom))] z-30 flex flex-col gap-2 border-t border-line-soft p-3 md:bottom-0">
      {children}
    </div>
  );
}

type Icon = ComponentType<{ size?: number; className?: string }>;

/** **مربّعُ اختيار** — أيقونةٌ في دائرةٍ واسمٌ، والمختارُ بحدّ المنصّة وأرضِها. */
export function ChoiceBox({
  icon: Ico,
  label,
  tone,
  on,
  disabled,
  onPick,
  testId,
}: {
  icon: Icon;
  label: string;
  tone: "success" | "danger" | "accent" | "muted";
  on: boolean;
  disabled?: boolean;
  onPick: () => void;
  testId?: string;
}) {
  const circle = {
    success: "bg-success-tint text-success",
    danger: "bg-danger-tint text-danger",
    accent: "bg-accent-tint text-accent-text",
    muted: "bg-field text-ink-muted",
  }[tone];
  return (
    <button
      type="button"
      role="radio"
      aria-checked={on}
      disabled={disabled}
      onClick={onPick}
      data-testid={testId}
      className={`relative flex min-h-16 flex-1 items-center justify-center gap-2 rounded-card border px-3 py-2 transition-colors disabled:opacity-60 ${
        on ? "border-2 border-accent bg-accent-tint" : "border-line bg-surface hover:bg-row-hover"
      }`}
    >
      <span className={`flex h-9 w-9 shrink-0 items-center justify-center rounded-full ${circle}`} aria-hidden>
        <Ico size={20} />
      </span>
      <span className={`font-bold ${on ? "text-accent-text" : "text-ink"}`}>{label}</span>
      {on && <CheckBadge className="absolute end-2 top-2" />}
    </button>
  );
}

/** **علامةُ ✓ في زاوية المختار.** */
export function CheckBadge({ className = "" }: { className?: string }) {
  return (
    <span className={`flex h-5 w-5 items-center justify-center rounded-full bg-accent text-on-bright ${className}`} aria-hidden>
      <IconCheck size={13} />
    </span>
  );
}

/** **زرّان متساويان بحدٍّ واحد** — «نسبة / مبلغ» و«ليرة / دولار». */
export function Segmented<T extends string>({
  options,
  value,
  onChange,
  disabled,
  label,
}: {
  options: readonly (readonly [T, string])[];
  value: T;
  onChange: (v: T) => void;
  disabled?: boolean;
  label: string;
}) {
  return (
    <div role="radiogroup" aria-label={label} className="flex h-12 w-full overflow-hidden rounded-control border border-line">
      {options.map(([v, text]) => (
        <button
          key={v}
          type="button"
          role="radio"
          aria-checked={value === v}
          disabled={disabled}
          onClick={() => value !== v && onChange(v)}
          className={`flex-1 text-sm font-bold transition-colors disabled:opacity-60 ${
            value === v ? "bg-accent text-on-bright" : "bg-surface text-ink hover:bg-row-hover"
          }`}
        >
          {text}
        </button>
      ))}
    </div>
  );
}

/** **دائرتا العملة** — «ل.س» و«$»، والمختارةُ ممتلئةٌ بلون المنصّة. */
export function CurrencyCircles({
  value,
  onChange,
  disabled,
  label,
  syp,
  usd,
}: {
  value: "SYP" | "USD";
  onChange: (v: "SYP" | "USD") => void;
  disabled?: boolean;
  label: string;
  syp: string;
  usd: string;
}) {
  return (
    <div role="radiogroup" aria-label={label} className="flex shrink-0 gap-2">
      {(
        [
          ["SYP", syp],
          ["USD", usd],
        ] as const
      ).map(([code, text]) => (
        <button
          key={code}
          type="button"
          role="radio"
          aria-checked={value === code}
          disabled={disabled}
          onClick={() => value !== code && onChange(code)}
          data-testid={`currency-${code}`}
          className={`flex h-12 w-12 items-center justify-center rounded-full border text-sm font-bold transition-colors disabled:opacity-60 ${
            value === code ? "border-accent bg-accent text-on-bright" : "border-line bg-field text-ink hover:bg-row-hover"
          }`}
        >
          {text}
        </button>
      ))}
    </div>
  );
}

/** **رقاقةٌ تُقرأ** — طريقةُ السعر أو القاعدة أو سعرُ الوزن. */
export function TagChip({ text, tone }: { text: string; tone: "accent" | "warning" | "muted" }) {
  const cls = {
    accent: "bg-accent-tint text-accent-text",
    warning: "bg-warning-tint text-warning",
    muted: "bg-field text-ink-muted",
  }[tone];
  return <span className={`inline-block rounded-full px-2 py-0.5 text-2xs font-bold ${cls}`}>{text}</span>;
}

/** **حقلُ السطر** — ارتفاعٌ واحدٌ لحقول الخيارات والمشروبات، بلا تسميةٍ عائمة. */
export function RowField({
  value,
  onChange,
  placeholder,
  label,
  numeric,
  decimal,
  suffix,
  disabled,
  className = "",
  id,
}: {
  value: string;
  onChange: (v: string) => void;
  placeholder: string;
  label: string;
  numeric?: boolean;
  decimal?: boolean;
  suffix?: string;
  disabled?: boolean;
  className?: string;
  id?: string;
}) {
  return (
    <label className={`flex h-11 items-center gap-1 rounded-control border border-line bg-field px-2 focus-within:border-accent ${className}`}>
      <input
        id={id}
        value={value}
        onChange={(e) => onChange(e.target.value)}
        placeholder={placeholder}
        aria-label={label}
        disabled={disabled}
        inputMode={decimal ? "decimal" : numeric ? "numeric" : undefined}
        dir={numeric || decimal ? "ltr" : undefined}
        className="min-w-0 flex-1 bg-transparent text-sm text-ink outline-none placeholder:text-ink-muted"
      />
      {suffix && <span className="shrink-0 text-xs text-ink-muted">{suffix}</span>}
    </label>
  );
}

/** **عنوانُ عمودٍ فوق الأسطر** — مرّةً لا في كلّ حقل. */
export function ColumnLabel({ text, className = "" }: { text: string; className?: string }) {
  return <span className={`truncate text-2xs text-ink-muted ${className}`}>{text}</span>;
}
