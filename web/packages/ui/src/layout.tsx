"use client";

/**
 * العناصر البصرية المركزية — الشكل الموحّد لكل صفحة في المشروع.
 * ممنوع أن ترتجل أي صفحة عنواناً أو كرتاً أو حالة فراغ خاصة بها: كل شيء من هنا،
 * فيبقى المشروع بروح واحدة ويكفي تعديل واحد ليطال كل اللوحات (GROUND-RULES §1.2).
 */

import type { ComponentType, ReactNode } from "react";
import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";
import { IconStar } from "./icons";

const m = getMessages(defaultLocale);

type IconType = ComponentType<{ size?: number; className?: string }>;

// ---------- حاوية الصفحة ----------

const widths = {
  narrow: "mx-auto w-full max-w-lg", // نماذج ضيقة (الحساب، الإعدادات)
  medium: "mx-auto w-full max-w-2xl", // محتوى متوسط
  wide: "mx-auto w-full max-w-6xl", // جداول وقوائم
  full: "w-full", // يملأ العرض
} as const;

/** حاوية الصفحة الموحّدة — عرض وإيقاع رأسي واحد بدل الارتجال في كل صفحة. */
export function PageContainer({
  width = "full",
  className = "",
  children,
}: {
  width?: keyof typeof widths;
  className?: string;
  children: ReactNode;
}) {
  return <div className={`${widths[width]} space-y-5 ${className}`}>{children}</div>;
}

// ---------- عنوان الصفحة ----------

/** عنوان الصفحة الموحّد — حجم وأيقونة وتباعد واحد في كل اللوحات. */
export function PageHeader({
  icon: Icon,
  title,
  subtitle,
  actions,
}: {
  icon?: IconType;
  title: string;
  subtitle?: string;
  actions?: ReactNode;
}) {
  return (
    <div className="flex flex-wrap items-center justify-between gap-3">
      <div className="min-w-0">
        <h1 className="flex items-center gap-2 text-xl font-bold">
          {Icon && <Icon size={20} className="text-primary" />}
          {title}
        </h1>
        {subtitle && <p className="mt-1 text-sm text-ink-muted">{subtitle}</p>}
      </div>
      {actions && <div className="flex flex-wrap items-center gap-2">{actions}</div>}
    </div>
  );
}

// ---------- الكرت ----------

const pads = { sm: "p-3", md: "p-4", lg: "p-5" } as const;

/** الكرت الموحّد — الحاوية البصرية الوحيدة المسموحة للمحتوى. */
export function Card({
  title,
  icon: Icon,
  padding = "md",
  className = "",
  actions,
  children,
}: {
  title?: string;
  icon?: IconType;
  padding?: keyof typeof pads;
  className?: string;
  actions?: ReactNode;
  children: ReactNode;
}) {
  return (
    <section className={`rounded-card border border-line bg-surface ${pads[padding]} ${className}`}>
      {(title || actions) && (
        <div className="mb-3 flex items-center justify-between gap-2">
          {title && (
            <h2 className="flex items-center gap-2 text-sm font-bold">
              {Icon && <Icon size={16} className="text-primary" />}
              {title}
            </h2>
          )}
          {actions}
        </div>
      )}
      {children}
    </section>
  );
}

// ---------- حالات الفراغ والتحميل ----------

/** حالة "لا بيانات" الموحّدة. */
export function EmptyState({
  icon: Icon,
  title,
  tone = "muted",
  action,
}: {
  icon?: IconType;
  title: string;
  tone?: "muted" | "success";
  action?: ReactNode;
}) {
  return (
    <div className="rounded-card border border-line bg-surface p-10 text-center">
      {Icon && <Icon size={28} className="mx-auto mb-2 text-ink-muted" />}
      <p className={`text-sm ${tone === "success" ? "text-success" : "text-ink-muted"}`}>{title}</p>
      {action && <div className="mt-3 flex justify-center">{action}</div>}
    </div>
  );
}

/** حالة التحميل الموحّدة. */
export function LoadingState({ label }: { label?: string }) {
  return <p className="py-10 text-center text-sm text-ink-muted">{label ?? m.common.loading}</p>;
}

// ---------- صف قائمة ----------

/** صف قائمة موحّد — البديل المركزي للكروت المرصوفة يدوياً. */
export function ListRow({
  leading,
  title,
  subtitle,
  trailing,
  onClick,
  className = "",
}: {
  leading?: ReactNode;
  title: ReactNode;
  subtitle?: ReactNode;
  trailing?: ReactNode;
  onClick?: () => void;
  className?: string;
}) {
  return (
    <li
      onClick={onClick}
      className={`flex items-center gap-3 rounded-card border border-line bg-surface p-3 ${
        onClick ? "cursor-pointer transition-shadow hover:border-primary/40 hover:shadow-sm" : ""
      } ${className}`}
    >
      {leading}
      <div className="min-w-0 flex-1">
        <div className="truncate font-medium">{title}</div>
        {subtitle && <div className="mt-0.5 text-xs text-ink-muted">{subtitle}</div>}
      </div>
      {trailing && <div className="flex shrink-0 items-center gap-3">{trailing}</div>}
    </li>
  );
}

// ---------- كروت الإحصاء ----------

/** شبكة كروت الإحصاء الموحّدة. */
export function StatGrid({ children }: { children: ReactNode }) {
  return <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 lg:grid-cols-4">{children}</div>;
}

/** كرت مؤشر موحّد — شكل واحد لكل الأرقام في المشروع. */
export function StatCard({
  icon: Icon,
  label,
  value,
  sub,
  tone = "default",
  onClick,
  selected = false,
}: {
  icon?: IconType;
  label: string;
  value: string | number;
  sub?: string;
  tone?: "default" | "success" | "danger" | "accent";
  onClick?: () => void;
  selected?: boolean;
}) {
  const toneCls = {
    default: "text-ink",
    success: "text-success",
    danger: "text-danger",
    accent: "text-accent-dark",
  }[tone];
  const Tag = onClick ? "button" : "div";
  return (
    <Tag
      onClick={onClick}
      className={`rounded-card border bg-surface p-4 text-start transition-colors ${
        selected ? "border-primary ring-1 ring-primary/30" : "border-line"
      } ${onClick ? "hover:border-primary/40" : ""}`}
    >
      {Icon && <Icon size={18} className="mb-1 text-ink-muted" />}
      <p className={`text-xl font-bold ${toneCls}`}>
        {typeof value === "number" ? fmtNum(value) : value}
      </p>
      <p className="text-xs text-ink-muted">{label}</p>
      {sub && <p className="mt-0.5 text-xs text-ink-muted">{sub}</p>}
    </Tag>
  );
}

// ---------- النجوم ----------

/** عرض التقييم بالنجوم — نسخة واحدة بلون التمييز المركزي. */
export function Stars({
  value,
  size = "md",
  onChange,
}: {
  value: number;
  size?: "sm" | "md" | "lg";
  /** عند تمريرها تصبح النجوم قابلة للنقر (وضع التقييم) */
  onChange?: (v: number) => void;
}) {
  const n = Math.max(0, Math.min(5, Math.round(value)));
  const px = { sm: 14, md: 18, lg: 30 }[size];
  // نجوم من مجموعة الأيقونات لا من محرف ★ — تتبع التوكنز وتظهر متطابقة في كل نظام
  return (
    <span
      dir="ltr"
      className="inline-flex items-center gap-0.5 align-middle"
      role={onChange ? "radiogroup" : undefined}
      aria-label={`${n}/5`}
    >
      {[1, 2, 3, 4, 5].map((i) => {
        const star = (
          <IconStar
            size={px}
            strokeWidth={1.8}
            className={i <= n ? "fill-accent text-accent" : "text-line"}
          />
        );
        return onChange ? (
          <button
            key={i}
            type="button"
            onClick={() => onChange(i)}
            aria-label={String(i)}
            aria-checked={i === n}
            role="radio"
            className="rounded-control p-0.5 transition-transform hover:scale-110"
          >
            {star}
          </button>
        ) : (
          <span key={i}>{star}</span>
        );
      })}
    </span>
  );
}



// ---------- تبويبات ----------

export interface TabItem {
  /** المفتاح المستعمل في المقارنة — لا يُعرض */
  key: string;
  label: string;
  /** عدّاد اختياري بجانب العنوان — يُخفى إن كان صفراً */
  count?: number;
}

/**
 * شريط تبويبات موحّد — بديلٌ عن قائمة طويلة يختلط فيها كل شيء.
 *
 * قابل للتمرير أفقياً على الهاتف بدل أن ينكسر إلى سطور: التبويبات صفٌّ واحد
 * ذهنياً، وكسرها إلى ثلاثة سطور يُفقدها معناها. والمؤشّر خطٌّ سفليّ لا خلفية
 * ممتلئة، كي لا ينافس التبويبُ المحتوى تحته على الانتباه.
 */
export function Tabs({
  items,
  active,
  onChange,
  className = "",
}: {
  items: TabItem[];
  active: string;
  onChange: (key: string) => void;
  className?: string;
}) {
  return (
    <div
      role="tablist"
      className={`-mb-px flex gap-1 overflow-x-auto border-b border-line ${className}`}
    >
      {items.map((t) => {
        const on = t.key === active;
        return (
          <button
            key={t.key}
            type="button"
            role="tab"
            aria-selected={on}
            onClick={() => onChange(t.key)}
            className={`flex shrink-0 items-center gap-1.5 border-b-2 px-3 py-2.5 text-sm font-medium transition-colors ${
              on
                ? "border-primary text-primary-dark"
                : "border-transparent text-ink-muted hover:border-line hover:text-ink"
            }`}
          >
            {t.label}
            {!!t.count && (
              <span
                className={`rounded-badge px-1.5 py-0.5 text-[11px] font-bold ${
                  on ? "bg-primary-light text-primary-dark" : "bg-page text-ink-muted"
                }`}
              >
                {fmtNum(t.count)}
              </span>
            )}
          </button>
        );
      })}
    </div>
  );
}
