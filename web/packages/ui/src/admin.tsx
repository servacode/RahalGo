"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **عُدّةُ لوحة الإدارة — قالبُ الصفحة وما يُبنى به** (دفعةُ تصميم اللوحة ١، ٢٠٢٦-١٠-٣١)
 * ══════════════════════════════════════════════════════════════════════
 *
 * قرّر المالك أنّ اللوحة «مزحومة وما بتشبه بعضها». فكلُّ صفحةٍ في اللوحة تُبنى من هنا:
 *
 *   AdminPage     رأسٌ بسطرٍ واحد (عنوان · عدد · أفعال) وتحته التبويباتُ خطّاً — **بلا سطرِ شرح**
 *   FilterBar     بحثٌ + قائمتان على الأكثر + «فلاتر» تفتح ورقةً للباقي + «امسح»
 *   PeriodPicker  اليوم · أمس · آخر 7 أيام · هالشهر · تاريخ مخصّص (والتاريخان يظهران حين يُختار)
 *   SummaryStrip  أربعةُ أرقامٍ على الأكثر — أرقامٌ لا فلترٌ ثانٍ
 *   AdminCard · RowList · Row   لوحٌ واحدٌ بلا صناديقَ داخل صناديق
 *   AdminLoading · AdminReload · AdminEmpty   حالةٌ واحدةٌ بشكلٍ واحد
 *   MoreMenu · ConfirmDanger    النادرُ والخطرُ خلف «⋯» وتأكيد
 *
 * **ولا يمسّ `/shop` ولا `/store`**: أصنافُه `admin-*` من `theme.css` لا يكتبها غيرُه.
 */

import { isValidElement, useState, type ComponentType, type ReactNode } from "react";
import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";
import { Button, Input, Select } from "./components";
import { Confirm, SkeletonList } from "./feedback";
import { Pagination } from "./navigation";
import { Sheet } from "./overlay";
import { ActionMenu, type ActionMenuItem } from "./ActionMenu";
import { IconMore, IconOptions, IconSearch, IconWarning } from "./icons";

const m = getMessages(defaultLocale);
const K = m.admin.kit;

type IconT = ComponentType<{ size?: number; className?: string }>;

// ═════════════════════════════════════════════════════════════════════
//  قالبُ الصفحة
// ═════════════════════════════════════════════════════════════════════

export interface AdminTab<T extends string = string> {
  key: T;
  label: string;
  /** **عددٌ بجانب التبويب** — وفارغُه لا يُرسم. */
  count?: number | null;
  /** **عددٌ يحتاج انتباهاً** — يُلوَّن (والعددُ نفسُه يقول المعنى). */
  attention?: boolean;
}

/** **تبويباتٌ خطٌّ تحت الرأس** — لا بطاقةٌ ولا حبّات. */
export function AdminTabs<T extends string>({
  items,
  value,
  onChange,
}: {
  items: readonly AdminTab<T>[];
  value: T;
  onChange: (k: T) => void;
}) {
  return (
    <div role="tablist" className="admin-tabs">
      {items.map((t) => {
        const on = t.key === value;
        return (
          <button
            key={t.key}
            type="button"
            role="tab"
            aria-selected={on}
            onClick={() => onChange(t.key)}
            className="admin-tab"
          >
            {t.label}
            {t.count != null && (
              <span className="admin-tab-count" data-attention={t.attention && t.count > 0 ? "true" : "false"}>
                {fmtNum(t.count)}
              </span>
            )}
          </button>
        );
      })}
    </div>
  );
}

/**
 * **قالبُ صفحة اللوحة.**
 *
 * الترتيبُ ثابتٌ في كلّ صفحة: الرأسُ ← التنبيهات ← شريطُ الفلاتر ← الأرقام ← المحتوى ← الترقيم.
 * **ولا `subtitle`** — قرارُ المالك: لا سطرَ شرحٍ تحت العناوين.
 */
export function AdminPage<T extends string = string>({
  title,
  icon: Icon,
  count,
  badge,
  actions,
  tabs,
  tab,
  onTab,
  notices,
  toolbar,
  summary,
  pagination,
  children,
}: {
  title: string;
  icon?: IconT;
  /** **عددُ ما في الصفحة** — بجانب العنوان. */
  count?: number | null;
  /** **شارةٌ بجانب العنوان** — «مباشر» مثلاً. */
  badge?: ReactNode;
  /** **الأفعالُ في آخر السطر** — زرٌّ أساسيٌّ واحدٌ على الأكثر، والنادرُ في `MoreMenu`. */
  actions?: ReactNode;
  tabs?: readonly AdminTab<T>[];
  tab?: T;
  onTab?: (k: T) => void;
  /** **تنبيهاتٌ فوق الفلاتر** — خطأُ تحديثٍ، تقييدٌ من رابط… */
  notices?: ReactNode;
  toolbar?: ReactNode;
  summary?: ReactNode;
  pagination?: { page: number; total: number; perPage: number; onChange: (p: number) => void; busy?: boolean };
  children: ReactNode;
}) {
  const hasTabs = !!tabs && tabs.length > 0 && tab !== undefined && !!onTab;
  return (
    <div className="admin-page">
      <header className="admin-head" data-tabs={hasTabs ? "true" : "false"}>
        <div className="flex flex-wrap items-center justify-between gap-x-4 gap-y-3">
          <div className="flex min-w-0 items-center gap-3">
            {Icon && (
              <span className="admin-head-icon" aria-hidden>
                <Icon size={18} />
              </span>
            )}
            <h1 className="admin-title min-w-0 truncate">{title}</h1>
            {count != null && <span className="admin-count">{fmtNum(count)}</span>}
            {badge}
          </div>
          {actions && <div className="flex flex-wrap items-center gap-2">{actions}</div>}
        </div>
        {hasTabs && <AdminTabs items={tabs} value={tab} onChange={onTab} />}
      </header>
      {notices}
      {toolbar}
      {summary}
      {children}
      {pagination && pagination.total > pagination.perPage && (
        <div className="flex justify-center">
          <Pagination
            page={pagination.page}
            total={pagination.total}
            perPage={pagination.perPage}
            onChange={pagination.onChange}
            busy={pagination.busy}
          />
        </div>
      )}
    </div>
  );
}

// ═════════════════════════════════════════════════════════════════════
//  شريطُ الفلاتر
// ═════════════════════════════════════════════════════════════════════

/**
 * **شريطُ الفلاتر بسطرٍ واحد.**
 *
 * بحثٌ، ثمّ ما يُعطى في `children` (قائمتان على الأكثر)، ثمّ «فلاتر» تفتح ورقةً بما في `more`،
 * ثمّ «امسح» حين يكون شيءٌ مفلتراً. **و`end` في آخر السطر** — عددُ النتائج مثلاً.
 */
export function FilterBar({
  search,
  children,
  more,
  moreActive = 0,
  filtered = false,
  onClear,
  end,
}: {
  search?: { value: string; onChange: (v: string) => void; placeholder?: string; id?: string };
  children?: ReactNode;
  /** **ما زاد على قائمتين** — يُفتح في ورقة. */
  more?: ReactNode;
  /** **كم فلتراً مفعّلاً داخل الورقة** — يُرسم عدداً على الزرّ. */
  moreActive?: number;
  filtered?: boolean;
  onClear?: () => void;
  end?: ReactNode;
}) {
  const [open, setOpen] = useState(false);
  return (
    <div className="flex flex-wrap items-center gap-2">
      {search && (
        <div className="min-w-0 flex-1 basis-56 sm:max-w-sm">
          <Input
            id={search.id ?? "admin-search"}
            type="search"
            icon={<IconSearch size={15} />}
            aria-label={K.searchLabel}
            placeholder={search.placeholder}
            value={search.value}
            onChange={(e) => search.onChange(e.target.value)}
          />
        </div>
      )}
      {children}
      {more && (
        <Button variant="secondary" onClick={() => setOpen(true)} className="inline-flex items-center gap-2">
          <IconOptions size={15} />
          {K.filters}
          {moreActive > 0 && <span className="admin-tab-count" data-attention="false">{fmtNum(moreActive)}</span>}
        </Button>
      )}
      {filtered && onClear && (
        <Button variant="ghost" onClick={onClear}>
          {K.clear}
        </Button>
      )}
      {end && <div className="ms-auto text-sm text-ink-muted">{end}</div>}
      {more && (
        <Sheet
          open={open}
          onClose={() => setOpen(false)}
          title={K.filters}
          footer={
            <div className="flex gap-2">
              {filtered && onClear && (
                <Button variant="ghost" className="flex-1" onClick={onClear}>
                  {K.clear}
                </Button>
              )}
              <Button className="flex-1" onClick={() => setOpen(false)}>
                {K.done}
              </Button>
            </div>
          }
        >
          <div className="space-y-4">{more}</div>
        </Sheet>
      )}
    </div>
  );
}

/** **قائمةٌ منسدلةٌ للشريط** — بلا تسميةٍ ظاهرة (والتسميةُ للقارئ الصوتيّ). */
export function FilterSelect({
  id,
  label,
  value,
  onChange,
  children,
}: {
  id: string;
  label: string;
  value: string;
  onChange: (v: string) => void;
  children: ReactNode;
}) {
  return (
    <div className="w-auto min-w-0 max-w-full">
      <Select id={id} aria-label={label} value={value} onChange={(e) => onChange(e.target.value)}>
        {children}
      </Select>
    </div>
  );
}

/** **حقلٌ داخل ورقة الفلاتر** — بتسميته فوقه. */
export function FilterField({ id, label, children }: { id?: string; label: string; children: ReactNode }) {
  return (
    <div className="space-y-1.5">
      <label htmlFor={id} className="block text-sm font-medium">
        {label}
      </label>
      {children}
    </div>
  );
}

// ═════════════════════════════════════════════════════════════════════
//  الفترة
// ═════════════════════════════════════════════════════════════════════

export type PeriodKey = "all" | "today" | "yesterday" | "last7" | "month" | "custom";

/** **يومٌ ± أيّام** — على التاريخ وحدَه (غرينتش) فلا تُزحزحه منطقةُ الجهاز. */
export function shiftIsoDay(iso: string, days: number): string {
  const d = new Date(`${iso}T00:00:00Z`);
  d.setUTCDate(d.getUTCDate() + days);
  return d.toISOString().slice(0, 10);
}

/** **مدى الفترة الجاهزة** — و`today` يومُ الحساب من الخادم (يبدأ بساعة فتح الدوام). */
export function periodRange(p: Exclude<PeriodKey, "all" | "custom">, today: string): { from: string; to: string } {
  switch (p) {
    case "today":
      return { from: today, to: today };
    case "yesterday": {
      const y = shiftIsoDay(today, -1);
      return { from: y, to: y };
    }
    case "last7":
      return { from: shiftIsoDay(today, -6), to: today };
    case "month":
      return { from: `${today.slice(0, 8)}01`, to: today };
  }
}

const PRESETS = ["today", "yesterday", "last7", "month"] as const;

/** **أيُّ فترةٍ هذه** — جاهزةٌ إن طابقت، وإلّا مخصّصة. */
export function periodOf(from: string, to: string, today: string): PeriodKey {
  if (!from && !to) return "all";
  if (!today) return "custom";
  return PRESETS.find((p) => {
    const r = periodRange(p, today);
    return r.from === from && r.to === to;
  }) ?? "custom";
}

/**
 * **مختارُ الفترة الواحد.**
 *
 * قائمةٌ واحدةٌ في الشريط: (كل الأيام) · اليوم · أمس · آخر 7 أيام · هالشهر · تاريخ مخصّص.
 * **والتاريخان لا يظهران إلّا حين يُختار «تاريخ مخصّص».** و`today` يومُ الحساب من الخادم —
 * لا يُحسب هنا شيءٌ غيرُ ما يحسبه، فلا يتغيّر معنى «اليوم» في صفحة.
 */
export function PeriodPicker({
  id = "admin-period",
  from,
  to,
  today,
  onChange,
  allowAll = true,
}: {
  id?: string;
  from: string;
  to: string;
  today: string;
  onChange: (from: string, to: string) => void;
  allowAll?: boolean;
}) {
  const P = K.period;
  const [customOpen, setCustomOpen] = useState(false);
  const detected = periodOf(from, to, today);
  const value: PeriodKey = customOpen ? "custom" : detected;
  const pick = (v: string) => {
    const k = v as PeriodKey;
    if (k === "custom") {
      setCustomOpen(true);
      return;
    }
    setCustomOpen(false);
    if (k === "all") onChange("", "");
    else if (today) {
      const r = periodRange(k, today);
      onChange(r.from, r.to);
    }
  };
  return (
    <>
      <FilterSelect id={id} label={P.label} value={value} onChange={pick}>
        {allowAll && <option value="all">{P.all}</option>}
        {today && PRESETS.map((p) => <option key={p} value={p}>{P[p]}</option>)}
        <option value="custom">{P.custom}</option>
      </FilterSelect>
      {value === "custom" && (
        <div className="flex flex-wrap items-center gap-2">
          <Input
            id={`${id}-from`}
            type="date"
            dir="ltr"
            aria-label={P.from}
            value={from}
            onChange={(e) => onChange(e.target.value, to)}
          />
          <Input
            id={`${id}-to`}
            type="date"
            dir="ltr"
            aria-label={P.to}
            value={to}
            onChange={(e) => onChange(from, e.target.value)}
          />
        </div>
      )}
    </>
  );
}

// ═════════════════════════════════════════════════════════════════════
//  الأرقام
// ═════════════════════════════════════════════════════════════════════

export type StatTone = "neutral" | "attention" | "danger" | "good" | "muted";

export interface SummaryItem {
  key: string;
  label: string;
  value: ReactNode;
  icon?: IconT;
  tone?: StatTone;
  /** **سطرٌ صغيرٌ تحت الرقم** — عددٌ أو تفصيلٌ قصير، لا شرح. */
  meta?: ReactNode;
  /** **ينقل لصفحةٍ أخرى** — ولا يُستعمل فلتراً ثانياً لما في الصفحة. */
  onOpen?: () => void;
}

/** **رقمٌ واحد** — تسميةٌ فوقه وسطرٌ صغيرٌ تحته، ونبرتُه خطٌّ على طرفه (والتسميةُ تقول المعنى). */
export function Stat({ label, value, icon: Icon, tone = "neutral", meta, onOpen }: Omit<SummaryItem, "key">) {
  const body = (
    <>
      <span className="flex items-center justify-between gap-2 text-sm text-ink-muted">
        <span className="min-w-0 truncate">{label}</span>
        {Icon && <Icon size={16} className="shrink-0" />}
      </span>
      <span
        className={`admin-figure ${
          tone === "muted" ? "text-ink-muted" : tone === "danger" ? "text-danger" : tone === "attention" ? "text-warning" : ""
        }`}
      >
        {value}
      </span>
      {meta && <span className="truncate text-xs text-ink-muted">{meta}</span>}
    </>
  );
  const cls = "admin-panel admin-stat";
  if (!onOpen) {
    return (
      <div className={cls} data-tone={tone}>
        {body}
      </div>
    );
  }
  return (
    <button type="button" onClick={onOpen} className={`${cls} transition-colors hover:bg-row-hover`} data-tone={tone}>
      {body}
    </button>
  );
}

/** **شريطُ الأرقام** — أربعةٌ على الأكثر: عمودان على الجوّال وأربعةٌ على الشاشة. */
export function SummaryStrip({ items }: { items: readonly SummaryItem[] }) {
  return (
    <div className="grid grid-cols-2 gap-3 lg:grid-cols-4">
      {items.slice(0, 4).map(({ key, ...it }) => (
        <Stat key={key} {...it} />
      ))}
    </div>
  );
}

// ═════════════════════════════════════════════════════════════════════
//  اللوحُ والصفوف
// ═════════════════════════════════════════════════════════════════════

/** **اللوحُ الواحد** — لا لوحَ داخلَه. وعنوانُه وأفعالُه بسطرٍ واحد. */
export function AdminCard({
  title,
  actions,
  padded = true,
  as: Tag = "section",
  className = "",
  children,
}: {
  title?: string;
  actions?: ReactNode;
  /** **بلا حشوة** حين يكون ما فيه صفوفاً (`RowList`). */
  padded?: boolean;
  as?: "section" | "div" | "article" | "li";
  className?: string;
  children: ReactNode;
}) {
  return (
    <Tag className={`admin-panel ${padded ? "p-4 lg:p-5" : "overflow-hidden"} ${className}`}>
      {(title || actions) && (
        <div className={`flex flex-wrap items-center justify-between gap-2 ${padded ? "mb-3" : "px-4 pt-4 pb-3 lg:px-5"}`}>
          {title && <h2 className="admin-h2">{title}</h2>}
          {actions && <div className="flex items-center gap-2">{actions}</div>}
        </div>
      )}
      {children}
    </Tag>
  );
}

/** **صفوفٌ في لوحٍ واحد** — بدل خمسين بطاقةً منفصلة. */
export function RowList({ children, className = "" }: { children: ReactNode; className?: string }) {
  return <ul className={`admin-panel admin-rows ${className}`}>{children}</ul>;
}

/** **عنوانُ مجموعةٍ داخل الصفوف** — يومٌ مثلاً. */
export function RowGroupTitle({ icon: Icon, children }: { icon?: IconT; children: ReactNode }) {
  return (
    <li className="admin-group-title" role="presentation">
      {Icon && <Icon size={14} />}
      {children}
    </li>
  );
}

/**
 * **صفٌّ واحد** — بدايةٌ (أيقونة/صورة) · عنوانٌ وسطرٌ تحته · نهايةٌ (مبلغ/وقت/أفعال).
 * **ويُضغط كلُّه إن أُعطي `onOpen`** — والروابطُ داخله تُوقف الضغطة عنه.
 */
export function Row({
  leading,
  title,
  meta,
  trailing,
  onOpen,
  label,
}: {
  leading?: ReactNode;
  title: ReactNode;
  meta?: ReactNode;
  trailing?: ReactNode;
  onOpen?: () => void;
  /** **اسمُه للقارئ الصوتيّ** حين يُضغط. */
  label?: string;
}) {
  const click = onOpen
    ? {
        role: "button" as const,
        tabIndex: 0,
        "aria-label": label,
        onClick: onOpen,
        onKeyDown: (ev: React.KeyboardEvent) => {
          if (ev.key === "Enter" || ev.key === " ") {
            ev.preventDefault();
            onOpen();
          }
        },
      }
    : {};
  return (
    <li>
      <div className="admin-row" data-click={onOpen ? "true" : "false"} {...click}>
        {leading && <span className="shrink-0">{leading}</span>}
        <span className="min-w-0 flex-1">
          <span className="block font-bold">{title}</span>
          {meta && <span className="mt-0.5 block text-sm text-ink-muted">{meta}</span>}
        </span>
        {trailing && <span className="shrink-0 text-end">{trailing}</span>}
      </div>
    </li>
  );
}

/** **دائرةُ الأيقونة في بداية الصفّ** — هادئةٌ، والنبرةُ تُقال بالنصّ لا باللون وحدَه. */
export function RowIcon({ icon: Icon, tone = "neutral" }: { icon: IconT; tone?: "neutral" | "attention" | "danger" }) {
  const paint =
    tone === "danger" ? "bg-danger-tint text-danger" : tone === "attention" ? "bg-warning-tint text-warning" : "bg-field text-ink-muted";
  return (
    <span className={`inline-flex h-9 w-9 items-center justify-center rounded-full ${paint}`}>
      <Icon size={16} />
    </span>
  );
}

/** **خانةُ معلومةٍ صغيرة** — تسميةٌ فوق قيمة، بلا صندوق. */
export function Fact({ label, children, className = "" }: { label: string; children: ReactNode; className?: string }) {
  return (
    <div className={`min-w-0 ${className}`}>
      <dt className="text-xs text-ink-muted">{label}</dt>
      <dd className="mt-0.5 text-sm">{children}</dd>
    </div>
  );
}

// ═════════════════════════════════════════════════════════════════════
//  الحالات
// ═════════════════════════════════════════════════════════════════════

/** **جارٍ التحميل** — هيكلُ صفوفٍ في لوحٍ واحد. */
export function AdminLoading({ rows = 5 }: { rows?: number }) {
  return (
    <div role="status" aria-live="polite" aria-label={m.common.loading} className="admin-panel p-4">
      <SkeletonList rows={rows} />
    </div>
  );
}

/** **صار خطأ** — جملةٌ واحدةٌ وزرٌّ واحد. */
export function AdminReload({ onRetry, label }: { onRetry?: () => void; label?: string }) {
  return (
    <div className="admin-panel admin-state" role="alert">
      <span className="admin-state-icon">
        <IconWarning size={22} />
      </span>
      <p className="text-sm font-bold">{label ?? K.loadFailed}</p>
      {onRetry && (
        <Button variant="secondary" size="sm" onClick={onRetry}>
          {K.retry}
        </Button>
      )}
    </div>
  );
}

/** **لا شيء هنا** — أيقونةٌ وجملةٌ وفعلٌ اختياريّ. بلا سطرِ شرح. */
export function AdminEmpty({
  icon: Icon,
  title,
  action,
}: {
  icon?: IconT | ReactNode;
  title: string;
  action?: ReactNode;
}) {
  return (
    <div className="admin-panel admin-state">
      {Icon && (
        <span className="admin-state-icon">
          {isValidElement(Icon) ? Icon : (() => {
            const I = Icon as IconT;
            return <I size={22} />;
          })()}
        </span>
      )}
      <p className="text-sm font-bold">{title}</p>
      {action}
    </div>
  );
}

// ═════════════════════════════════════════════════════════════════════
//  النادرُ والخطِر
// ═════════════════════════════════════════════════════════════════════

/** **«⋯» — الأفعالُ النادرةُ والخطِرة** في قائمةٍ تُفتح، والخطِرُ بالأحمر في آخرها. */
export function MoreMenu({ items, label, size = "sm" }: { items: readonly ActionMenuItem[]; label?: string; size?: "sm" | "md" }) {
  return (
    <ActionMenu
      label={<IconMore size={18} />}
      ariaLabel={label ?? K.moreActions}
      items={items}
      variant="ghost"
      chevron={false}
      size={size}
    />
  );
}

/** **تأكيدُ ما لا يُستدرَك** — حذفٌ وحظرٌ وشطب: زرُّه يقول فعلَه، والتركيزُ على الإلغاء. */
export function ConfirmDanger(props: {
  open: boolean;
  title: string;
  body?: ReactNode;
  confirmLabel: string;
  onConfirm: () => void;
  onCancel: () => void;
  busy?: boolean;
}) {
  return <Confirm {...props} tone="danger" />;
}
