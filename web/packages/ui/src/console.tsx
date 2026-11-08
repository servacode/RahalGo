"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **عُدّةُ «كونسول» لوحة المتجر — الأوليّاتُ المركزيّة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (التصميمُ المعتمد ٢٠٢٦-١١-٠٨ بعد ثلاث جولاتٍ رُفضت: لابتوب بمقام Stripe/Shopify، وآيفون بمقام Apple HIG.)
 *
 * **الشكلُ في `theme.css`** (قسمُ «لوحة المتجر — كونسول»)، وهنا قشرتُه البرمجيّة فقط. وصفحةٌ جديدةٌ في لوحة المتجر
 * تستعمل هذه الأوليّاتِ ولا تكتب عنواناً ولا بطاقةَ رقمٍ ولا مفتاحاً ولا خانةَ اختيارٍ بيدها. **ولا أثرَ لها على
 * الإدارة ولا على الزبون** — لا يستوردانها، والأنماطُ مقيّدةٌ بـ`.console-root`.
 *
 * **الهاتفُ والحاسوب**: العتبةُ `lg` (١٠٢٤). دونها شريطُ تبويبٍ وعنوانٌ كبيرٌ يطوي إلى شريطٍ شفّافٍ عند التمرير ومجموعاتٌ مُدرجة؛
 * وفوقها سايدبارٌ وشريطٌ علويٌّ وبطاقاتٌ وجداول.
 */

import {
  createContext,
  useCallback,
  useContext,
  useEffect,
  useId,
  useLayoutEffect,
  useRef,
  useState,
  useSyncExternalStore,
  type ComponentType,
  type ReactNode,
} from "react";
import { createPortal } from "react-dom";
import { fmtNum } from "@rahalgo/i18n";
import { IconArrowDownRight, IconArrowUpRight, IconCheck, IconChevronDown, IconImageOff, IconNext } from "./icons";
import { Sparkline } from "./charts";

type IconType = ComponentType<{ size?: number; className?: string; strokeWidth?: number }>;
type LinkType = ComponentType<{
  href: string;
  className?: string;
  "data-tone"?: string;
  "data-testid"?: string;
  "data-ico"?: string;
  "aria-current"?: "page";
  children: ReactNode;
}>;

// ══════════════════════════════════════════════════════════════════════
// الثيم (فاتحٌ افتراضاً · يُحفظ لكلّ متصفّح · يحترم prefers-color-scheme في الزيارة الأولى)
// ══════════════════════════════════════════════════════════════════════

export type ConsoleTheme = "light" | "dark";
const THEME_KEY = "rahalgo_console_theme";
const themeListeners = new Set<() => void>();

/** **سكربتٌ يُحقن في الـ`<head>`/أوّل الجسم** فيُرسم أوّلُ إطارٍ بالثيم الصحيح بلا وميض. */
export const CONSOLE_THEME_SCRIPT = `(function(){try{var t=localStorage.getItem("${THEME_KEY}");if(t!=="dark"&&t!=="light"){t=window.matchMedia&&matchMedia("(prefers-color-scheme: dark)").matches?"dark":"light"}document.documentElement.setAttribute("data-console-theme",t)}catch(e){}})();`;

function readTheme(): ConsoleTheme {
  return document.documentElement.getAttribute("data-console-theme") === "dark" ? "dark" : "light";
}
function subscribeTheme(cb: () => void) {
  themeListeners.add(cb);
  return () => {
    themeListeners.delete(cb);
  };
}
function applyTheme(t: ConsoleTheme) {
  document.documentElement.setAttribute("data-console-theme", t);
  try {
    localStorage.setItem(THEME_KEY, t);
  } catch {
    /* وضعٌ خاصٌّ أو تخزينٌ ممنوع — يبقى الاختيارُ لهذه الجلسة. */
  }
  themeListeners.forEach((f) => f());
}

/** **ثيمُ لوحة المتجر** — `toggle` يبدّل ويحفظ. */
export function useConsoleTheme(): { theme: ConsoleTheme; toggle: () => void } {
  const theme = useSyncExternalStore(subscribeTheme, readTheme, () => "light" as ConsoleTheme);
  useEffect(() => {
    // **دخولٌ من صفحةٍ أخرى بلا سكربت الـ`<head>`** — نحسبه هنا مرّةً.
    if (document.documentElement.hasAttribute("data-console-theme")) return;
    let t: ConsoleTheme = "light";
    try {
      const saved = localStorage.getItem(THEME_KEY);
      if (saved === "dark" || saved === "light") t = saved;
      else if (window.matchMedia?.("(prefers-color-scheme: dark)").matches) t = "dark";
    } catch {
      /* يبقى الفاتح. */
    }
    document.documentElement.setAttribute("data-console-theme", t);
    themeListeners.forEach((f) => f());
  }, []);
  const toggle = useCallback(() => applyTheme(readTheme() === "dark" ? "light" : "dark"), []);
  return { theme, toggle };
}

/** **هل الشاشةُ حاسوب؟** (`lg` فما فوق) — لما يُرسم بشكلين مختلفين (جدولٌ أو قائمةٌ مُدرجة). */
export function useIsDesktop(): boolean {
  return useSyncExternalStore(
    (cb) => {
      const mq = window.matchMedia("(min-width: 1024px)");
      mq.addEventListener("change", cb);
      return () => mq.removeEventListener("change", cb);
    },
    () => window.matchMedia("(min-width: 1024px)").matches,
    () => false,
  );
}

// ══════════════════════════════════════════════════════════════════════
// شريطُ التنقّل على الهاتف — الصفحةُ تُسجّل عنوانَها وأزرارَها فيه
// ══════════════════════════════════════════════════════════════════════

export interface ConsoleNavCtx {
  setTitle: (t: string) => void;
  startEl: HTMLElement | null;
  endEl: HTMLElement | null;
}
export const ConsoleNavContext = createContext<ConsoleNavCtx | null>(null);

/**
 * **رأسُ الصفحة** — على الحاسوب: عنوانٌ ٢٢ وسطرٌ وصفيٌّ وأزرارُ الفعل في الطرف.
 * وعلى الهاتف: **عنوانٌ كبيرٌ ٣٤ يطوي إلى الشريط الشفّاف عند التمرير**، وأزرارُ الأيقونات (`navActions`) في الشريط نفسِه.
 */
export function PageHead({
  title,
  subtitle,
  actions,
  leading,
  navStart,
  navEnd,
  titleAs: Title = "h1",
}: {
  title: ReactNode;
  subtitle?: ReactNode;
  /** أزرارُ الفعل (حاسوباً؛ وجوّالاً تحت العنوان إن لم تُعطَ `navEnd`). */
  actions?: ReactNode;
  /** سهمُ رجوعٍ أو شارةٌ قبل العنوان. */
  leading?: ReactNode;
  /** **جوّالاً**: نصٌّ في طرف البداية من الشريط الشفّاف («تحديد»). */
  navStart?: ReactNode;
  /** **جوّالاً**: أيقوناتٌ في الطرف الآخر من الشريط (الجرس · الحساب · «+»). */
  navEnd?: ReactNode;
  titleAs?: "h1" | "h2";
}) {
  const nav = useContext(ConsoleNavContext);
  const plain = typeof title === "string" ? title : "";
  useLayoutEffect(() => {
    nav?.setTitle(plain);
    return () => nav?.setTitle("");
  }, [nav, plain]);
  return (
    <div className="console-head">
      {/* **جوّالاً سهمُ الرجوع في الشريط الشفّاف** (أو نصُّ `navStart` إن أُعطي) — لا قبلَ العنوان الكبير. */}
      {nav?.startEl && (navStart ?? leading) && createPortal(navStart ?? leading, nav.startEl)}
      {nav?.endEl && navEnd && createPortal(navEnd, nav.endEl)}
      <div className="flex min-w-0 items-center gap-2">
        {leading && <span className="max-lg:hidden">{leading}</span>}
        <div className="min-w-0">
          <Title className="console-title">{title}</Title>
          {subtitle && <p className="console-meta mt-0.5">{subtitle}</p>}
        </div>
      </div>
      {actions && (
        <div className={`flex min-w-0 max-w-full flex-wrap items-center gap-2 ${navEnd ? "max-lg:hidden" : ""}`}>{actions}</div>
      )}
    </div>
  );
}

// ══════════════════════════════════════════════════════════════════════
// المؤشّرات
// ══════════════════════════════════════════════════════════════════════

/** **دلتا** — سهمٌ مائلٌ ونسبةٌ (والمعنى بالسهم والنصّ لا باللون وحدَه). */
export function Delta({ pct, label }: { pct: number; label?: ReactNode }) {
  const up = pct >= 0;
  const Ico = up ? IconArrowUpRight : IconArrowDownRight;
  return (
    <span className="inline-flex items-center gap-1.5">
      <b className="delta" data-dir={up ? "up" : "down"}>
        <Ico size={13} />
        <span dir="ltr">{fmtNum(Math.abs(pct))}%</span>
      </b>
      {label}
    </span>
  );
}

/**
 * **مؤشّرٌ** — تسميةٌ بأيقونةٍ رماديّة، رقمٌ ٢٧، وسطرٌ سفليٌّ (دلتا أو وصف)، وبإمكانه رسمٌ مصغّرٌ في الطرف.
 * `tone` يلوّن الرسمَ فقط، **والمعنى في التسمية والنصّ.**
 * ومع `href` يصير رابطاً كاملَ المساحة (يُمرَّر `Link` من التطبيق).
 */
export function Kpi({
  icon: Icon,
  label,
  value,
  sub,
  unit,
  tone,
  spark,
  href,
  Link,
  testId,
  className = "",
}: {
  icon?: IconType;
  label: string;
  value: string | number;
  sub?: ReactNode;
  /** وحدةٌ صغيرةٌ بعد الرقم (ل.س). */
  unit?: string;
  tone?: "good" | "attention" | "danger" | "accent" | "violet";
  /** قيمٌ من الأقدم إلى الأحدث (أسبوعٌ مثلاً) — تُرسم بخطٍّ ناعم. */
  spark?: readonly number[];
  href?: string;
  Link?: LinkType;
  testId?: string;
  /** صفوفُ شبكةٍ إضافيّة (مثلاً `col-span-2`). */
  className?: string;
}) {
  const body = (
    <>
      <span className="flex items-center gap-2 text-sm font-medium text-ink-muted">
        {Icon && <Icon size={15} className="text-ink-dim" />}
        <span className="min-w-0 flex-1 truncate">{label}</span>
      </span>
      <span className="mt-1 flex flex-wrap items-baseline gap-x-1.5">
        <span className="kpi-figure">{typeof value === "number" ? fmtNum(value) : value}</span>
        {unit && <span className="console-meta text-sm">{unit}</span>}
      </span>
      {sub && <span className="console-meta mt-auto flex items-center gap-1.5 truncate pt-1">{sub}</span>}
      {spark && spark.length > 1 && (
        <span className="kpi-spark max-lg:hidden">
          <Sparkline data={spark} className="" />
        </span>
      )}
    </>
  );
  if (!href) {
    return (
      <div className={`kpi ${className}`} data-tone={tone} data-testid={testId}>
        {body}
      </div>
    );
  }
  const Tag = Link ?? "a";
  return (
    <Tag href={href} className={`kpi ${className}`} data-tone={tone} data-testid={testId}>
      {body}
    </Tag>
  );
}

/** **شبكةُ المؤشّرات** — عمودان على الجوّال وأربعةٌ (أو ثلاثةٌ) على الحاسوب. */
export function KpiGrid({ children, cols = 4 }: { children: ReactNode; cols?: 3 | 4 }) {
  return (
    <div className={`grid grid-cols-2 gap-3 lg:gap-4 ${cols === 3 ? "lg:grid-cols-3" : "lg:grid-cols-4"}`}>{children}</div>
  );
}

// ══════════════════════════════════════════════════════════════════════
// ضوابطُ الإدخال
// ══════════════════════════════════════════════════════════════════════

/** **مفتاحٌ مقسَّم** — خياراتٌ متنافية (لوحة | قائمة · ٧ أيام | ٣٠ يوماً · مفتوح | مغلق). */
export function Segmented<K extends string>({
  items,
  value,
  onChange,
  ariaLabel,
  block = false,
  testId,
}: {
  items: readonly { key: K; label: ReactNode; icon?: ReactNode; title?: string }[];
  value: K;
  onChange: (key: K) => void;
  ariaLabel: string;
  /** يملأ العرضَ كلَّه (الهاتف). */
  block?: boolean;
  testId?: string;
}) {
  return (
    <div role="group" aria-label={ariaLabel} className="seg" data-block={block ? "" : undefined} data-testid={testId}>
      {items.map((it) => (
        <button
          key={it.key}
          type="button"
          title={it.title}
          aria-pressed={it.key === value}
          onClick={() => onChange(it.key)}
          className="seg-item"
        >
          {it.icon}
          {it.label}
        </button>
      ))}
    </div>
  );
}

/** **مفتاحُ التشغيل** (سطحُه فقط) — الحاضنُ زرٌّ بدورٍ `switch`. `size="sm"` لخليّة الجدول. */
export function SwitchTrack({ on, size }: { on: boolean; size?: "sm" }) {
  return <span aria-hidden className="sw-track" data-on={on} data-size={size} />;
}

/** **سطرٌ بمفتاح** — تسميةٌ ووصفٌ ومفتاحٌ؛ واجهتُه كـ`Switch` المركزيّ القديم لتنتقل الاستعمالاتُ بلا تغيير. */
export function Toggle({
  checked,
  onChange,
  label,
  hint,
  disabled = false,
  id,
}: {
  checked: boolean;
  onChange: (next: boolean) => void;
  label: string;
  hint?: string;
  disabled?: boolean;
  id?: string;
}) {
  const auto = useId();
  const btnID = id ?? auto;
  return (
    <div className="flex items-center justify-between gap-3 py-1">
      <label htmlFor={btnID} className="min-w-0 flex-1 cursor-pointer">
        <span className="block text-sm font-medium">{label}</span>
        {hint && <span className="console-meta mt-0.5 block">{hint}</span>}
      </label>
      <button
        id={btnID}
        type="button"
        role="switch"
        aria-checked={checked}
        disabled={disabled}
        onClick={() => onChange(!checked)}
        className="flex min-h-9 shrink-0 items-center disabled:opacity-50"
      >
        <SwitchTrack on={checked} />
      </button>
    </div>
  );
}

/** **خانةُ اختيارٍ** — كاملةٌ أو جزئيّة؛ زرٌّ بدور `checkbox`. */
export function CheckMark({
  state,
  onChange,
  label,
  testId,
}: {
  state: boolean | "mixed";
  onChange: () => void;
  label: string;
  testId?: string;
}) {
  const on = state === true;
  return (
    <button
      type="button"
      role="checkbox"
      aria-checked={state}
      aria-label={label}
      data-testid={testId}
      onClick={(e) => {
        e.stopPropagation();
        onChange();
      }}
      className="flex size-6 items-center justify-center"
    >
      <span className="cb" data-state={state === "mixed" ? "mixed" : on ? "on" : "off"}>
        {on && <IconCheck size={12} strokeWidth={3} />}
        {state === "mixed" && <span className="block h-0.5 w-2 rounded-badge bg-on-bright" />}
      </span>
    </button>
  );
}

/** **زرُّ أيقونة** ٣٢ شفّاف. */
export function IconButton({
  icon: Icon,
  label,
  onClick,
  href,
  Link,
  dot = false,
  testId,
  size = 18,
}: {
  icon: IconType;
  label: string;
  onClick?: () => void;
  href?: string;
  Link?: LinkType;
  /** نقطةٌ حمراء (إشعارٌ غيرُ مقروء). */
  dot?: boolean;
  testId?: string;
  size?: number;
}) {
  const inner = (
    <>
      <Icon size={size} />
      {dot && <span className="icon-btn-dot" aria-hidden />}
    </>
  );
  if (href) {
    const Tag = Link ?? "a";
    return (
      <Tag href={href} className="icon-btn" data-testid={testId} aria-label={label}>
        {inner}
      </Tag>
    );
  }
  return (
    <button type="button" className="icon-btn" aria-label={label} title={label} onClick={onClick} data-testid={testId}>
      {inner}
    </button>
  );
}

/** **حبّةُ ترشيح** — اسمٌ وعدّاد؛ المفعَّلةُ ممتلئةٌ بصبغة العلامة. */
export function Chip({
  selected,
  count,
  onClick,
  children,
  testId,
}: {
  selected: boolean;
  count?: number;
  onClick: () => void;
  children: ReactNode;
  testId?: string;
}) {
  return (
    <button type="button" className="chip" aria-pressed={selected} onClick={onClick} data-testid={testId}>
      {children}
      {count != null && <span className="chip-count">{fmtNum(count)}</span>}
    </button>
  );
}

/** **ترويسةُ قسمٍ** — عنوانٌ صغير. */
export function SectionTitle({ children, action }: { children: ReactNode; action?: ReactNode }) {
  return (
    <div className="flex items-center gap-3">
      <h2 className="console-section min-w-0 flex-1">{children}</h2>
      {action}
    </div>
  );
}

/** **شارةُ حالة** — نقطةٌ ونصّ (المعنى بالنصّ لا باللون وحدَه). */
export function Pill({
  tone,
  dot = true,
  children,
  testId,
}: {
  tone?: "good" | "attention" | "danger" | "accent" | "info" | "violet";
  dot?: boolean;
  children: ReactNode;
  testId?: string;
}) {
  return (
    <span className="pill" data-tone={tone} data-testid={testId}>
      {dot && <span className="size-1.5 shrink-0 rounded-full bg-current" aria-hidden />}
      {children}
    </span>
  );
}

/** **صورةٌ مصغّرةٌ** — الصورةُ إن وُجدت وإلّا مربّعٌ متقطّعٌ بأيقونة «بلا صورة». */
export function Thumb({ src, size = 36, className = "" }: { src?: string | null; size?: number; className?: string }) {
  return (
    <span className={`thumb ${className}`} data-empty={src ? undefined : ""} style={{ ["--s" as string]: `${size / 16}rem` }}>
      {src ? <img src={src} alt="" loading="lazy" className="size-full object-cover" /> : <IconImageOff size={Math.round(size * 0.42)} />}
    </span>
  );
}

// ══════════════════════════════════════════════════════════════════════
// المجموعاتُ المُدرجة (آيفون) وبطاقاتُ الصفوف (لابتوب) — صنفٌ واحد
// ══════════════════════════════════════════════════════════════════════

/** **مجموعةٌ مُدرجة** — خلايا ببياضٍ وفواصلَ مزاحة؛ وعلى اللابتوب بطاقةٌ بصفوفٍ مفصولة. */
export function IGroup({
  children,
  title,
  footer,
  className = "",
  testId,
}: {
  children: ReactNode;
  title?: ReactNode;
  footer?: ReactNode;
  className?: string;
  testId?: string;
}) {
  return (
    <section data-testid={testId} className={className}>
      {title && <div className="sec-head">{title}</div>}
      <div className="igroup">{children}</div>
      {footer && <div className="sec-foot">{footer}</div>}
    </section>
  );
}

/**
 * **صفٌّ** — مربّعٌ ملوّنٌ ثمّ عنوانٌ ووصفٌ ثمّ قيمةٌ وسهمٌ، وبإمكانه أن يكون رابطاً أو زرّاً.
 * `warn` يلوّن القيمةَ ويضيف نقطةً (والنصُّ يقول).
 */
export function IRow({
  icon: Icon,
  tone,
  title,
  sub,
  value,
  valueTone,
  trailing,
  href,
  Link,
  onClick,
  disabled,
  chevron = true,
  testId,
}: {
  icon?: IconType;
  tone?: "good" | "attention" | "danger" | "info" | "violet" | "gray";
  title: ReactNode;
  sub?: ReactNode;
  value?: ReactNode;
  valueTone?: "danger" | "good";
  trailing?: ReactNode;
  href?: string;
  Link?: LinkType;
  onClick?: () => void;
  disabled?: boolean;
  chevron?: boolean;
  testId?: string;
}) {
  const body = (
    <>
      {Icon && (
        <span className="tile" data-tone={tone}>
          <Icon size={18} />
        </span>
      )}
      <span className="min-w-0 flex-1 py-2.5">
        <span className="block text-base font-medium leading-snug lg:text-sm">{title}</span>
        {sub && <span className="console-meta block">{sub}</span>}
      </span>
      {value != null && (
        <span className={`flex min-w-0 max-w-[42%] items-center gap-1.5 text-end ${valueTone === "danger" ? "text-danger" : "text-ink-dim"}`}>
          <span className="min-w-0 truncate">{value}</span>
        </span>
      )}
      {trailing}
      {chevron && (href || onClick) && <IconNext size={14} className="shrink-0 text-ink-dim opacity-70" />}
    </>
  );
  if (href) {
    const Tag = Link ?? "a";
    return (
      <Tag href={href} className="irow" data-ico={Icon ? "" : undefined} data-testid={testId}>
        {body}
      </Tag>
    );
  }
  if (onClick) {
    return (
      <button type="button" onClick={onClick} disabled={disabled} className="irow disabled:opacity-60" data-ico={Icon ? "" : undefined} data-testid={testId}>
        {body}
      </button>
    );
  }
  return (
    <div className="irow" data-ico={Icon ? "" : undefined} data-testid={testId}>
      {body}
    </div>
  );
}

/**
 * **رسمٌ مصغَّرٌ بالأعمدة** (أسبوعُ الهاتف) — أعمدةٌ بارتفاعٍ نسبيٍّ وتسميةٌ تحت كلٍّ منها،
 * واليومُ الأخيرُ بلون العلامة؛ **والزمنُ يمضي من اليمين إلى اليسار كالعربيّة.**
 */
export function MiniBars({
  data,
  height = 54,
  ariaLabel,
}: {
  data: { label: string; value: number; title?: string }[];
  height?: number;
  ariaLabel: string;
}) {
  const max = Math.max(1, ...data.map((d) => d.value));
  return (
    <div role="img" aria-label={ariaLabel} className="flex w-full flex-col gap-1">
      <div className="flex items-end gap-[7px]" style={{ height }}>
        {data.map((d, i) => (
          <div
            key={`${d.label}-${i}`}
            className="mini-bar"
            data-now={i === data.length - 1}
            title={d.title}
            style={{ ["--v" as string]: d.value / max }}
          />
        ))}
      </div>
      <div className="flex gap-[7px]">
        {data.map((d, i) => (
          <span
            key={`${d.label}-${i}`}
            className={`flex-1 text-center text-2xs ${i === data.length - 1 ? "font-bold text-ink" : "text-ink-dim"}`}
            aria-hidden
          >
            {d.label}
          </span>
        ))}
      </div>
    </div>
  );
}

// ══════════════════════════════════════════════════════════════════════
// حبّةُ ترشيحٍ منسدلة (لابتوب) — قائمةُ اختيارٍ واحد
// ══════════════════════════════════════════════════════════════════════

/**
 * **حبّةُ ترشيحٍ بقائمة** — «القسم: ألبان وأجبان ▾». المفعَّلةُ (قيمةٌ غيرُ الافتراضيّة) بصبغة العلامة.
 * الخيارُ الأوّلُ هو الافتراضيّ («كلُّ الأقسام»). تُغلق بالنقر خارجَها أو Esc.
 */
export function FilterMenu<K extends string>({
  label,
  options,
  value,
  onChange,
  icon: Icon,
  testId,
}: {
  label: string;
  options: readonly { key: K; label: string; count?: number }[];
  value: K;
  onChange: (key: K) => void;
  icon?: IconType;
  testId?: string;
}) {
  const [open, setOpen] = useState(false);
  const box = useRef<HTMLDivElement>(null);
  useEffect(() => {
    if (!open) return;
    const away = (e: PointerEvent) => {
      if (!box.current?.contains(e.target as Node)) setOpen(false);
    };
    const esc = (e: KeyboardEvent) => e.key === "Escape" && setOpen(false);
    document.addEventListener("pointerdown", away);
    document.addEventListener("keydown", esc);
    return () => {
      document.removeEventListener("pointerdown", away);
      document.removeEventListener("keydown", esc);
    };
  }, [open]);
  const first = options[0];
  const active = first != null && value !== first.key;
  const current = options.find((o) => o.key === value);
  return (
    <div ref={box} className="relative" data-testid={testId}>
      <button type="button" className="chip" aria-pressed={active} aria-haspopup="listbox" aria-expanded={open} onClick={() => setOpen((v) => !v)}>
        {Icon && <Icon size={14} />}
        {label}
        {active && current && <em className="border-s border-primary-edge ps-2 font-bold not-italic text-primary">{current.label}</em>}
        <IconChevronDown size={14} />
      </button>
      {open && (
        <div role="listbox" className="filter-pop">
          {options.map((o) => (
            <button
              key={o.key}
              type="button"
              role="option"
              aria-selected={o.key === value}
              onClick={() => {
                onChange(o.key);
                setOpen(false);
              }}
              className="flex w-full items-center gap-2 rounded-control px-2.5 py-2 text-start text-sm hover:bg-row-hover aria-selected:font-bold aria-selected:text-primary"
            >
              <span className="min-w-0 flex-1 truncate">{o.label}</span>
              {o.count != null && <span className="console-meta tabular-nums">{fmtNum(o.count)}</span>}
              {o.key === value && <IconCheck size={14} />}
            </button>
          ))}
        </div>
      )}
    </div>
  );
}

// ══════════════════════════════════════════════════════════════════════
// صفٌّ قابلٌ للسحب (آيفون)
// ══════════════════════════════════════════════════════════════════════

export interface SwipeAction {
  key: string;
  label: string;
  icon: IconType;
  tone: "info" | "attention" | "danger";
  onClick: () => void;
}

/**
 * **صفٌّ يُسحب نحو اليمين فيكشف أزرارَ الفعل عن اليسار** (اتّجاهُ العربيّة). السحبُ باللمس وحدَه؛ ولمسةٌ عاديّةٌ
 * على الصفّ المفتوح تُغلقه ولا تفتح شيئاً. وغيرُ اللمس (فأرة) لا يرى الأزرار — فيها أيقوناتُ الصفّ نفسُه.
 */
export function SwipeRow({ actions, children }: { actions: readonly SwipeAction[]; children: ReactNode }) {
  const W = 78 * actions.length;
  const [dx, setDx] = useState(0);
  const [drag, setDrag] = useState(false);
  const [openRow, setOpenRow] = useState(false);
  const start = useRef<{ x: number; y: number; base: number; lock: boolean | null } | null>(null);
  const moved = useRef(false);
  const shown = drag ? dx : openRow ? W : 0;
  return (
    <div className="swipe-row">
      <div className="swipe-acts" aria-hidden={!openRow}>
        {actions.map((a) => {
          const Ico = a.icon;
          return (
            <button
              key={a.key}
              type="button"
              tabIndex={openRow ? 0 : -1}
              className="swipe-act"
              data-tone={a.tone}
              onClick={() => {
                setOpenRow(false);
                a.onClick();
              }}
            >
              <Ico size={22} />
              {a.label}
            </button>
          );
        })}
      </div>
      <div
        className="swipe-front"
        style={{ transform: `translateX(${shown}px)`, transition: drag ? "none" : undefined }}
        onTouchStart={(e) => {
          const t = e.touches[0];
          if (!t) return;
          start.current = { x: t.clientX, y: t.clientY, base: openRow ? W : 0, lock: null };
          moved.current = false;
        }}
        onTouchMove={(e) => {
          const st = start.current;
          const t = e.touches[0];
          if (!st || !t) return;
          const ddx = t.clientX - st.x;
          const ddy = t.clientY - st.y;
          if (st.lock === null && (Math.abs(ddx) > 8 || Math.abs(ddy) > 8)) st.lock = Math.abs(ddx) > Math.abs(ddy);
          if (!st.lock) return;
          moved.current = true;
          setDrag(true);
          setDx(Math.max(0, Math.min(W + 24, st.base + ddx)));
        }}
        onTouchEnd={() => {
          if (start.current?.lock) setOpenRow(dx > W / 2);
          setDrag(false);
          start.current = null;
        }}
        onClickCapture={(e) => {
          if (moved.current || openRow) {
            e.preventDefault();
            e.stopPropagation();
            moved.current = false;
            if (openRow) setOpenRow(false);
          }
        }}
      >
        {children}
      </div>
    </div>
  );
}

// ══════════════════════════════════════════════════════════════════════
// تبويباتٌ بخطٍّ سفليّ وعدّادات
// ══════════════════════════════════════════════════════════════════════

export interface LineTab<K extends string = string> {
  key: K;
  label: string;
  /** عدّادٌ كاملٌ (٥٬٣١١ لا «٩٩+»). */
  count?: number;
  /** نبرةُ التنبيه للعدّاد (بلا صورة · بلا قسم). */
  warn?: boolean;
}

/** **تبويباتُ الأقسام** — خطٌّ سفليٌّ بلون العلامة للنشِط، وعدّادٌ بزاوية ٦. */
export function LineTabs<K extends string>({
  items,
  value,
  onChange,
  ariaLabel,
}: {
  items: readonly LineTab<K>[];
  value: K;
  onChange: (key: K) => void;
  ariaLabel?: string;
}) {
  return (
    <div role="tablist" aria-label={ariaLabel} className="line-tabs">
      {items.map((t) => (
        <button key={t.key} type="button" role="tab" aria-selected={t.key === value} onClick={() => onChange(t.key)}>
          {t.label}
          {t.count != null && (
            <span className="line-tab-n" data-warn={t.warn ? "" : undefined}>
              {fmtNum(t.count)}
            </span>
          )}
        </button>
      ))}
    </div>
  );
}
