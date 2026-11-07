"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **عُدّة «لوحة التشغيل» — رأس الصفحة · المؤشّر · الحبّة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (دفعةُ تصميم لوحة المتجر ١، ٢٠٢٦-١١-٠٧: «لازم يكون أقوى واحترافي أكثر».)
 *
 * **الشكلُ في `theme.css`** (`console-*` · `kpi` · `chip`)، وهنا قشرتُه البرمجيّة فقط:
 * صفحةٌ جديدةٌ في لوحة المتجر تستعمل هذه الثلاثة ولا تكتب عنواناً ولا بطاقةَ رقمٍ ولا
 * رقاقةَ ترشيحٍ بيدها. **ولا أثرَ لها على الإدارة ولا على الزبون** — لا يستوردانها.
 */

import type { ComponentType, ReactNode } from "react";
import { fmtNum } from "@rahalgo/i18n";

type IconType = ComponentType<{ size?: number; className?: string }>;

/** **رأسُ الصفحة** — العنوانُ وما تحته وأزرارُ الفعل، على الأرضيّة لا في لوح. */
export function PageHead({
  title,
  subtitle,
  actions,
  leading,
  titleAs: Title = "h1",
}: {
  title: ReactNode;
  subtitle?: ReactNode;
  actions?: ReactNode;
  /** سهمُ رجوعٍ أو شارةٌ قبل العنوان. */
  leading?: ReactNode;
  titleAs?: "h1" | "h2";
}) {
  return (
    <div className="console-head">
      <div className="flex min-w-0 items-center gap-2">
        {leading}
        <div className="min-w-0">
          <Title className="console-title">{title}</Title>
          {subtitle && <p className="console-meta mt-0.5">{subtitle}</p>}
        </div>
      </div>
      {actions && <div className="flex min-w-0 max-w-full flex-wrap items-center gap-2">{actions}</div>}
    </div>
  );
}

/**
 * **مؤشّرٌ** — رقمٌ كبيرٌ بعرضٍ ثابتٍ وتسميةٌ تحته، وأيقونةٌ صغيرةٌ في الطرف.
 * `tone` يلوّن شريطَ البداية (لا الرقمَ) فلا يعتمد المعنى على اللون وحده: التسميةُ تقول.
 * ومع `href` يصير رابطاً كاملَ المساحة (يُمرَّر `Link` من التطبيق للتنقّل بلا إعادة تحميل).
 */
export function Kpi({
  icon: Icon,
  label,
  value,
  sub,
  unit,
  tone,
  href,
  Link,
  testId,
}: {
  icon?: IconType;
  label: string;
  value: string | number;
  sub?: ReactNode;
  /** **وحدةٌ صغيرةٌ بعد الرقم** (ل.س) — فلا تأكل عرضَ الرقم الكبير. */
  unit?: string;
  tone?: "good" | "attention" | "danger" | "accent";
  href?: string;
  Link?: ComponentType<{ href: string; className?: string; "data-tone"?: string; "data-testid"?: string; children: ReactNode }>;
  testId?: string;
}) {
  const body = (
    <>
      <span className="flex items-center justify-between gap-2">
        <span className="console-meta truncate font-bold">{label}</span>
        {Icon && <Icon size={18} className="shrink-0 text-ink-muted" />}
      </span>
      <span className="flex items-baseline gap-1.5 truncate">
        <span className="kpi-figure">{typeof value === "number" ? fmtNum(value) : value}</span>
        {unit && <span className="console-meta">{unit}</span>}
      </span>
      {sub && <span className="console-meta truncate">{sub}</span>}
    </>
  );
  if (!href) {
    return (
      <div className="kpi" data-tone={tone} data-testid={testId}>
        {body}
      </div>
    );
  }
  const Tag = Link ?? "a";
  return (
    <Tag href={href} className="kpi" data-tone={tone} data-testid={testId}>
      {body}
    </Tag>
  );
}

/** **شبكةُ المؤشّرات** — عمودان على الجوّال وأربعةٌ على الحاسوب. */
export function KpiGrid({ children, cols = 4 }: { children: ReactNode; cols?: 3 | 4 }) {
  return (
    <div className={`grid grid-cols-2 gap-3 ${cols === 3 ? "lg:grid-cols-3" : "lg:grid-cols-4"}`}>{children}</div>
  );
}

/** **حبّةُ ترشيح** — اسمٌ وعدّاد؛ المختارةُ ممتلئة. */
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
