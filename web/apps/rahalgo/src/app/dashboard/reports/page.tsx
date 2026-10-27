"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale, fmtNum, errorText, fmtMoney, fmtDate, damascusDay } from "@rahalgo/i18n";
import {
  Alert,
  Input,
  Button,
  Select,
  Chips,
  PageContainer,
  IconOrder,
  IconWallet,
  IconStore,
  IconDriver,
  IconDate,
  IconUsers,
  IconCheck,
  IconBlock,
  IconSwap,
  IconHourglass,
  IconReceipt,
  IconBalance,
  IconMoto,
  IconTrendUp,
  IconTrendDown,
  IconSmartphone,
  IconView,
  IconCompare,
  IconWarning,
  Money,
  SkeletonStats,
  SkeletonList,
} from "@rahalgo/ui";
import { api, apiFile } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useCanOpen } from "@/lib/policy";
import { PageHero } from "@/components/admin/PageHero";
import { Panel, type TileTone } from "@/components/admin/DashParts";
import { OpenLink } from "@/components/admin/OpenLink";

const m = getMessages(defaultLocale);

/**
 * ══════════════════════════════════════════════════════════════════════
 * **التقارير — قرارُ المالك ٢٠٢٦-١٠-٠٤**
 * ══════════════════════════════════════════════════════════════════════
 *
 * - **اليومُ يومُ الحساب** — يبدأ بساعة فتح الدوام (`platform.day_start_hour`)، **والخادمُ
 *   يقول «اليوم» (`today`)** فتُبنى الأزرارُ عليه (تدقيقُ ٢٠٢٦-١٠-٢٧: كانت تُبنى على يوم التقويم،
 *   فبين منتصف الليل وساعة الفتح كان «اليوم» أصفاراً وطلباتُ الليل في «أمس»).
 *   **وأوّلُ تحميلٍ بلا تاريخ** — يأخذ الافتراضَ من الخادم.
 * - **أرقامُ المال يحذفها الخادمُ** عمّن ليس من الماليّة أو مدير المنصّة؛
 *   والصفحةُ تعرض ما وصلها فقط (`money_visible`).
 * - **«ربح المنصّة» رقمُ صفحة الأرباح نفسُه** — وأحمرُ باسم «خسارة» إن كان سالباً.
 * - **مبيعاتُ الأصناف والتوصيلُ بطاقتان** (المالك ٢٠٢٦-١٠-٢٧) — الأصنافُ ومخطّطُها اليوميُّ بحساب
 *   أفضل المتاجر نفسِه (`items_sales`)، فمجموعُ المتاجر يساوي البطاقة؛ كانت بطاقةً واحدةً فيها التوصيل.
 * - **المستردُّ بطاقةٌ لحالها**، **ومقارنةٌ بالفترة السابقة المساوية** — وتاريخاها مكتوبان.
 * - **الزوّارُ أجهزةُ الزبائن وحدَها** — وعطبُهم يُقال لا يُخفى.
 * - **المدى المقلوبُ أو الأطولُ من سنة لا يُرسَل** — رسالةٌ تحت التاريخ، والأرقامُ السابقةُ باقية.
 *
 * **تصميمٌ احترافيّ (المالك ٢٠٢٦-١٠-٢٦/٢٧)** — رأسٌ موحّد، والفلاترُ ببطاقةٍ واحدةٍ بحبوب، والأرقامُ
 * بطاقاتٌ تُضغط فتفتح سجلَّ الطلبات أو الأرباح أو النقد بالفترة نفسِها، والمخطّطُ يعمل باللمس.
 */

interface DailyPoint {
  day: string;
  orders: number;
  delivered: number;
  sales?: number;
  /** **مبيعاتُ أصناف اليوم** — بحساب بطاقة «مبيعات الأصناف» نفسِه، بلا توصيل. */
  items_sales?: number;
}

interface Summary {
  orders_total: number;
  delivered: number;
  cancelled: number;
  refunded: number;
  avg_delivery_min: number;
  active_customers: number;
  gross_sales?: number;
  /** **مبيعاتُ الأصناف وحدَها** — تساوي مجموعَ مبيعات المتاجر (المالك ٢٠٢٦-١٠-٢٧). */
  items_sales?: number;
  delivery_fees?: number;
  commissions?: number;
  platform_profit?: number;
  wallet_paid?: number;
  cash_collected?: number;
}

interface Visitors {
  opens_today: number;
  opens_7d: number;
  opens_30d: number;
  devices_today: number;
  devices_7d: number;
  devices_30d: number;
}

interface Report {
  from: string;
  to: string;
  /** **يومُ الحساب اليوم** — من الخادم (يبدأ بساعة فتح الدوام). */
  today?: string;
  day_start_hour?: number;
  previous_from?: string;
  previous_to?: string;
  money_visible: boolean;
  profit_unfiltered: boolean;
  summary: Summary;
  previous: Summary;
  daily: DailyPoint[];
  top_merchants: { id: string; name: string; delivered: number; sales?: number }[];
  top_drivers: { id: string; name: string; phone?: string; delivered: number; cash?: number }[];
  visitors: Visitors | null;
  visitors_failed: boolean;
}

interface City {
  id: string;
  name: string;
}

const HOUR = 3_600_000;
/** أطولُ مدًى يقبله الخادم (`reportMaxDays`). */
const MAX_DAYS = 366;

/** **يومٌ بعد/قبل يوم** — حسابُ تاريخٍ بحت `YYYY-MM-DD`. */
function shiftDay(iso: string, n: number): string {
  const y = Number(iso.slice(0, 4));
  const mo = Number(iso.slice(5, 7));
  const d = Number(iso.slice(8, 10));
  return new Date(Date.UTC(y, mo - 1, d + n)).toISOString().slice(0, 10);
}

/** عددُ أيّام المدى شاملاً طرفيه. */
function spanDays(from: string, to: string): number {
  return Math.round((Date.parse(`${to}T00:00:00Z`) - Date.parse(`${from}T00:00:00Z`)) / (24 * HOUR)) + 1;
}

/** **الشهرُ الماضي بيوم الحساب** — من أوّله إلى آخره. */
function lastMonth(today: string): { from: string; to: string } {
  const y = Number(today.slice(0, 4));
  const mo = Number(today.slice(5, 7));
  const py = mo === 1 ? y - 1 : y;
  const pm = mo === 1 ? 12 : mo - 1;
  const last = new Date(Date.UTC(py, pm, 0)).getUTCDate();
  const mm = String(pm).padStart(2, "0");
  return { from: `${py}-${mm}-01`, to: `${py}-${mm}-${String(last).padStart(2, "0")}` };
}

/** «27/10» — يومٌ قصيرٌ للمخطّط. */
function shortDay(iso: string): string {
  return `${fmtNum(Number(iso.slice(8, 10)))}/${fmtNum(Number(iso.slice(5, 7)))}`;
}

/** تاريخٌ للعرض — منتصفُ النهار فلا ينزلق اليومُ بفرق التوقيت. */
function dayText(iso: string): string {
  return fmtDate(`${iso}T12:00:00Z`);
}

/** **التغيّرُ عن الفترة السابقة** — نصٌّ قصيرٌ واتّجاهُه (للّون)، أو لا شيء. */
interface Delta {
  text: string;
  dir: "up" | "down" | "flat" | "none";
}

function change(cur: number | undefined, prev: number | undefined): Delta | undefined {
  const r = m.admin.reports;
  if (cur === undefined || prev === undefined) return undefined;
  if (prev === 0) return cur === 0 ? undefined : { text: r.noCompare, dir: "none" };
  const pct = Math.round(((cur - prev) / Math.abs(prev)) * 100);
  if (pct === 0) return { text: r.previousSame, dir: "flat" };
  const d = `${pct > 0 ? "+" : "−"}${r.pct.replace("{n}", fmtNum(Math.abs(pct)))}`;
  return { text: r.vsPrevious.replace("{d}", d), dir: pct > 0 ? "up" : "down" };
}

/** نسبةٌ من الطلبات. */
function share(part: number, total: number): string | undefined {
  if (total <= 0) return undefined;
  return m.admin.reports.summary.ofOrders.replace("{p}", fmtNum(Math.round((part / total) * 100)));
}

const TONE_ICON: Record<TileTone, string> = {
  default: "bg-field text-ink",
  muted: "bg-field text-ink-muted",
  danger: "bg-danger-solid text-on-solid",
  warning: "bg-warning-solid text-on-solid",
  accent: "bg-accent text-on-bright",
  success: "bg-success-solid text-on-solid",
};

/**
 * **بطاقةُ رقمٍ بمقارنة** — شكلُ `Tile` نفسُه (أيقونةٌ في دائرةٍ ورقمٌ ولفظ)، وتحتها سطرُ التغيّر
 * ملوّناً: أخضرُ للأحسن وأحمرُ للأسوأ — **والأحسنُ في الملغى والمستردّ والوقت النزول** (`lowerIsBetter`).
 * **وبـ`onClick` تصير زرّاً** يفتح ما وراء الرقم؛ وبلاه رقمٌ هادئ. و`wide` تأخذ سطراً كاملاً على الجوّال.
 */
function StatTile({
  icon: Icon,
  label,
  value,
  sub,
  delta,
  tone = "default",
  lowerIsBetter = false,
  onClick,
  wide = false,
}: {
  icon: React.ComponentType<{ size?: number; className?: string }>;
  label: string;
  value: React.ReactNode;
  sub?: string;
  delta?: Delta;
  tone?: TileTone;
  lowerIsBetter?: boolean;
  onClick?: () => void;
  wide?: boolean;
}) {
  const good = delta && ((delta.dir === "up" && !lowerIsBetter) || (delta.dir === "down" && lowerIsBetter));
  const bad = delta && ((delta.dir === "down" && !lowerIsBetter) || (delta.dir === "up" && lowerIsBetter));
  const TrendIcon = delta?.dir === "up" ? IconTrendUp : delta?.dir === "down" ? IconTrendDown : null;
  const cls = `flex items-center gap-3 rounded-card border p-3 text-start ${wide ? "col-span-2 sm:col-span-1" : ""} ${
    onClick ? "transition-colors hover:bg-row-hover" : ""
  } ${tone === "danger" ? "border-danger" : tone === "warning" ? "border-warning" : "border-line-soft"}`;
  const body = (
    <>
      <span className={`inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full ${TONE_ICON[tone]}`}>
        <Icon size={18} />
      </span>
      <span className="min-w-0 flex-1">
        <span
          className={`figure block leading-none ${tone === "muted" ? "text-ink-muted" : tone === "danger" ? "text-danger" : ""}`}
        >
          {value}
        </span>
        <span className="mt-1 line-clamp-2 text-xs text-ink-muted">{label}</span>
        {sub && <span className="line-clamp-2 text-2xs text-ink-muted">{sub}</span>}
        {delta && (
          <span
            className={`mt-0.5 flex items-center gap-1 text-2xs ${
              good ? "font-bold text-success" : bad ? "font-bold text-danger" : "text-ink-muted"
            }`}
          >
            {TrendIcon && <TrendIcon size={12} />}
            {delta.text}
          </span>
        )}
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

/** **فارغٌ بالنمط** — دائرةٌ ملوّنةٌ وعنوانٌ وجملةٌ تساعد. */
function Empty({
  icon: Icon,
  title,
  hint,
}: {
  icon: React.ComponentType<{ size?: number; className?: string }>;
  title: string;
  hint: string;
}) {
  return (
    <div className="flex flex-col items-center gap-2 py-6 text-center">
      <span className="inline-flex h-12 w-12 items-center justify-center rounded-full bg-field text-ink-muted">
        <Icon size={22} />
      </span>
      <p className="text-sm font-bold">{title}</p>
      <p className="text-xs text-ink-muted">{hint}</p>
    </div>
  );
}

/** عمودٌ في المخطّط — يومٌ أو أسبوع. */
interface Bucket {
  key: string;
  label: string;
  week: boolean;
  v: number;
}

/**
 * **مخطّطُ أعمدةٍ يوميّ** — سلسلةٌ واحدةٌ بلونٍ واحد (عنوانُ اللوحة يسمّيها).
 *
 * (تدقيقُ ٢٠٢٦-١٠-٢٧) **يعمل باللمس** — الضغطُ يثبّت رقمَ العمود، والفأرةُ تُريه مروراً. **وفوقه
 * ملخّص** (أعلى يومٍ ومتوسّطُ اليوم)، **وأرقامُ الأيّام لا تتراكب** — في الفترة الطويلة يُكتب
 * بعضُها، **وفوق ٦٠ يوماً كلُّ عمودٍ أسبوع.** والتلميحُ عند الطرفين لا يخرج من الشاشة.
 */
function DailyBars({
  data,
  value,
  format,
}: {
  data: DailyPoint[];
  value: (d: DailyPoint) => number;
  format: (v: number) => string;
}) {
  const r = m.admin.reports;
  const [hover, setHover] = useState<number | null>(null);
  const [picked, setPicked] = useState<number | null>(null);

  const weekly = data.length > 60;
  const buckets: Bucket[] = [];
  if (weekly) {
    for (let i = 0; i < data.length; i += 7) {
      const chunk = data.slice(i, i + 7);
      const first = chunk[0]?.day ?? "";
      buckets.push({
        key: first,
        label: shortDay(first),
        week: true,
        v: chunk.reduce((a, d) => a + value(d), 0),
      });
    }
  } else {
    for (const d of data) buckets.push({ key: d.day, label: shortDay(d.day), week: false, v: value(d) });
  }

  const total = data.reduce((a, d) => a + value(d), 0);
  const max = Math.max(0, ...buckets.map((b) => b.v));
  const top = buckets.find((b) => b.v === max && max > 0);
  const avg = data.length > 0 ? total / data.length : 0;
  const scale = Math.max(1, max);
  const n = buckets.length;
  const step = n > 14 ? Math.ceil(n / 8) : 1;
  const shown = hover ?? picked;

  return (
    <div className="space-y-2">
      <p className="flex flex-wrap gap-x-3 gap-y-0.5 text-xs text-ink-muted">
        {top && (
          <span>
            {(top.week ? r.chartTopWeek : r.chartTop)
              .replace("{v}", format(top.v))
              .replace("{d}", top.label)}
          </span>
        )}
        <span>{r.chartAvg.replace("{v}", format(Math.round(avg)))}</span>
        {weekly && <span>{r.chartWeekly}</span>}
      </p>

      <div className="relative">
        <div className="pointer-events-none absolute inset-x-0 top-0 h-40">
          <div className="absolute inset-x-0 top-0 border-t border-dashed border-line-soft" />
          <div className="absolute inset-x-0 top-1/2 border-t border-dashed border-line-soft" />
          {max > 0 && (
            <span className="absolute end-0 top-0 -translate-y-full text-2xs text-ink-muted">{format(max)}</span>
          )}
        </div>

        <div className="flex h-40 items-end gap-0.5" dir="ltr">
          {buckets.map((b, i) => {
            const h = Math.max(b.v > 0 ? 6 : 2, (b.v / scale) * 100);
            const on = shown === i;
            const place = i < n / 3 ? "left-0" : i >= (2 * n) / 3 ? "right-0" : "left-1/2 -translate-x-1/2";
            const when = b.week ? r.weekOf.replace("{d}", b.label) : b.label;
            return (
              <button
                type="button"
                key={b.key}
                aria-label={`${when} · ${format(b.v)}`}
                className="relative flex h-full flex-1 items-end justify-center"
                onMouseEnter={() => setHover(i)}
                onMouseLeave={() => setHover(null)}
                onClick={() => setPicked((p) => (p === i ? null : i))}
              >
                <span
                  className={`block w-full max-w-6 transition-colors ${
                    b.v > 0 ? (on ? "bg-primary-dark" : "bg-primary") : "bg-line"
                  }`}
                  style={{ height: `${h}%`, borderRadius: "4px 4px 0 0" }}
                />
                {on && (
                  <span
                    className={`pointer-events-none absolute bottom-full z-10 mb-1 whitespace-nowrap surface-inset px-2.5 py-1.5 text-xs elev-2 ${place}`}
                  >
                    <span className="font-bold">{format(b.v)}</span>
                    <span className="text-ink-muted"> · {when}</span>
                  </span>
                )}
              </button>
            );
          })}
        </div>
        <div className="mt-1 flex gap-0.5 border-t border-line-soft pt-1" dir="ltr">
          {buckets.map((b, i) => (
            <div key={b.key} className="flex-1 overflow-visible whitespace-nowrap text-center text-2xs text-ink-muted">
              {i % step === 0 ? b.label : ""}
            </div>
          ))}
        </div>
      </div>
      <p className="text-2xs text-ink-muted">{r.chartTapHint}</p>
    </div>
  );
}

/** **سطرُ ترتيبٍ في «الأفضل»** — رقمُ المرتبة بدائرة، والاسمُ وسطرُه، والرقمُ على الطرف؛ ويفتح الملفّ لمن يُفتح له. */
function RankRow({
  rank,
  name,
  sub,
  end,
  href,
  allowed,
}: {
  rank: number;
  name: React.ReactNode;
  sub?: React.ReactNode;
  end?: React.ReactNode;
  href: string;
  allowed: boolean;
}) {
  return (
    <li>
      <OpenLink
        allowed={allowed}
        href={href}
        className="flex items-center gap-3 rounded-control px-2 py-2 transition-colors hover:bg-row-hover"
      >
        <span
          className={`inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-full text-xs font-bold tabular-nums ${
            rank === 1 ? "bg-accent text-on-bright" : "bg-field text-ink-muted"
          }`}
        >
          {fmtNum(rank)}
        </span>
        <span className="min-w-0 flex-1">
          <span className="block truncate text-sm font-bold">{name}</span>
          {sub && <span className="block truncate text-2xs text-ink-muted">{sub}</span>}
        </span>
        {end && <span className="shrink-0 text-end text-sm">{end}</span>}
      </OpenLink>
    </li>
  );
}

export default function ReportsPage() {
  const { can } = useAuth();
  const open = useCanOpen();
  const router = useRouter();
  const canExport = can("finance.export");
  const canOrders = can("orders.read");
  const canFinance = can("finance.read");
  const r = m.admin.reports;

  /** **فارغٌ = افتراضُ الخادم** (آخرُ سبعة أيّامٍ بيوم الحساب) — فلا يُحسب «اليوم» في المتصفّح. */
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const [kind, setKind] = useState("");
  const [city, setCity] = useState("");
  /** **والفلترُ من الرابط** — يُقرأ بعد التركيب فلا تختلف الصفحةُ بين الخادم والمتصفّح. */
  const [ready, setReady] = useState(false);
  const [cities, setCities] = useState<City[]>([]);
  const [report, setReport] = useState<Report | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [exporting, setExporting] = useState<"orders" | "ledger" | null>(null);

  useEffect(() => {
    const q = new URLSearchParams(window.location.search);
    setFrom(q.get("from") ?? "");
    setTo(q.get("to") ?? "");
    setKind(q.get("kind") ?? "");
    setCity(q.get("city_id") ?? "");
    setReady(true);
  }, []);

  useEffect(() => {
    api<{ cities: City[] }>("/api/v1/public/cities")
      .then((res) => setCities(res.cities ?? []))
      // @empty-ok — فلترُ المدينة اختياريّ: إن تعذّرت المدنُ غاب الفلترُ وبقي التقرير.
      .catch(() => setCities([]));
  }, []);

  // **«اليوم» من الخادم** — وقبل أوّل ردٍّ تقديرٌ بساعة الفتح الافتراضيّة (٩).
  const today = report?.today ?? damascusDay(Date.now() - (report?.day_start_hour ?? 9) * HOUR);
  const yesterday = shiftDay(today, -1);
  const lm = lastMonth(today);
  const presets = [
    { id: "today", label: r.range.today, from: today, to: today },
    { id: "yesterday", label: r.range.yesterday, from: yesterday, to: yesterday },
    { id: "last7", label: r.range.last7, from: shiftDay(today, -6), to: today },
    { id: "month", label: r.range.month, from: `${today.slice(0, 8)}01`, to: today },
    { id: "lastMonth", label: r.range.lastMonth, from: lm.from, to: lm.to },
  ];

  /** **المدى الفعليّ** — ما اختاره المستخدم، وإلّا ما ردّ به الخادم. */
  const effFrom = from || report?.from || "";
  const effTo = to || report?.to || "";
  const activePreset = presets.find((x) => effFrom === x.from && effTo === x.to)?.id ?? "";

  // **مدًى لا يقبله الخادم لا يُرسَل** — رسالةٌ تحت التاريخ، والأرقامُ السابقةُ باقية.
  const rangeError =
    effFrom && effTo
      ? effFrom > effTo
        ? r.rangeInverted
        : spanDays(effFrom, effTo) > MAX_DAYS
          ? r.rangeTooLong
          : ""
      : "";

  const query = new URLSearchParams();
  if (from) query.set("from", from);
  if (to) query.set("to", to);
  if (kind) query.set("kind", kind);
  if (city) query.set("city_id", city);
  const qs = query.toString();

  /** **رابطُ سجلّ الطلبات بالفترة والنوع نفسِهما** — والحالةُ إن طُلبت. */
  function historyHref(status?: string): string {
    const q = new URLSearchParams();
    if (report) {
      q.set("from", report.from);
      q.set("to", report.to);
    }
    if (kind) q.set("kind", kind);
    if (status) q.set("status", status);
    return `/dashboard/history?${q}`;
  }
  const goHistory = (status?: string) => (canOrders ? () => router.push(historyHref(status)) : undefined);
  const goProfits =
    canFinance && report ? () => router.push(`/dashboard/profits?from=${report.from}&to=${report.to}`) : undefined;
  const goCash = canFinance ? () => router.push("/dashboard/cash") : undefined;

  /** تنزيلُ CSV — **بالمدّة المعروضة نفسِها** (والنوعُ لتصدير الطلبات)؛ والزرُّ يقول إنّه يُنزّل. */
  function download(which: "orders" | "ledger") {
    if (!report || exporting) return;
    const path = which === "orders" ? "orders/export" : "ledger/export";
    const q = new URLSearchParams({ from: report.from, to: report.to });
    if (which === "orders" && kind) q.set("kind", kind);
    setExporting(which);
    void (async () => {
      try {
        const res = await apiFile(`/api/v1/admin/${path}?${q.toString()}`);
        const blob = await res.blob();
        const url = URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = `${which}-${report.from}_${report.to}.csv`;
        a.click();
        URL.revokeObjectURL(url);
      } catch (err) {
        setError(errorText(err));
      } finally {
        setExporting(null);
      }
    })();
  }

  const load = useCallback(async () => {
    setLoading(true);
    try {
      setReport(await api<Report>(`/api/v1/admin/reports${qs ? `?${qs}` : ""}`));
      setError("");
    } catch (err) {
      setError(errorText(err));
    } finally {
      setLoading(false);
    }
  }, [qs]);

  useEffect(() => {
    if (!ready || rangeError) return;
    try {
      window.history.replaceState(null, "", qs ? `${window.location.pathname}?${qs}` : window.location.pathname);
    } catch {
      /* الرابطُ راحةٌ لا شرط */
    }
    void load();
  }, [load, qs, ready, rangeError]);

  const s = report?.summary;
  const p = report?.previous;
  const money = report?.money_visible ?? false;
  const profit = s?.platform_profit ?? 0;
  const profitTone: TileTone = profit < 0 ? "danger" : profit === 0 ? "muted" : "success";
  const kinds = [
    { id: "all", label: r.kindAll },
    { id: "standard", label: r.kindStandard },
    { id: "custom", label: r.kindCustom },
    { id: "merchant_delivery", label: r.kindDelivery },
  ];
  const exportOff = !report || !!rangeError || exporting !== null;

  return (
    <PageContainer>
      {/* **رأسٌ موحّد** — والتنزيلُ لمن يملك `finance.export` وحدَه، كما كان. */}
      <PageHero
        icon={IconTrendUp}
        title={r.title}
        subtitle={r.subtitle}
        actions={
          canExport ? (
            <div className="flex flex-wrap gap-2">
              <Button variant="secondary" onClick={() => download("orders")} disabled={exportOff}>
                {exporting === "orders" ? r.exporting : r.exportOrders}
              </Button>
              <Button variant="secondary" onClick={() => download("ledger")} disabled={exportOff}>
                {exporting === "ledger" ? r.exporting : r.exportLedger}
              </Button>
            </div>
          ) : undefined
        }
      />

      {/* **الفترةُ والفلاترُ ببطاقةٍ واحدة** — حبوبٌ تلتفّ على الجوال ولا تُزيح الصفحة. */}
      <section className="surface space-y-3 p-3">
        <div className="flex flex-wrap items-center gap-2">
          <span className="text-sm text-ink-muted">{r.periodLabel}:</span>
          <Chips
            wrap
            items={presets.map((x) => ({ id: x.id, label: x.label }))}
            value={activePreset}
            onChange={(id) => {
              const x = presets.find((y) => y.id === id);
              if (!x) return;
              setFrom(x.from);
              setTo(x.to);
            }}
          />
        </div>
        <div className="grid grid-cols-2 gap-2 sm:flex sm:flex-wrap sm:items-end">
          <div className="sm:w-44">
            <Input
              id="r-from"
              label={r.from}
              icon={<IconDate />}
              type="date"
              value={effFrom}
              onChange={(e) => {
                setFrom(e.target.value);
                if (!to && report) setTo(report.to);
              }}
            />
          </div>
          <div className="sm:w-44">
            <Input
              id="r-to"
              label={r.to}
              icon={<IconDate />}
              type="date"
              value={effTo}
              onChange={(e) => {
                setTo(e.target.value);
                if (!from && report) setFrom(report.from);
              }}
            />
          </div>
        </div>
        {rangeError && (
          <p role="alert" className="flex items-center gap-1.5 text-xs font-bold text-danger">
            <IconWarning size={14} />
            {rangeError}
          </p>
        )}
        <Chips wrap items={kinds} value={kind || "all"} onChange={(id) => setKind(id === "all" ? "" : id)} />
        {cities.length > 1 &&
          (cities.length <= 3 ? (
            <Chips
              wrap
              items={[{ id: "all", label: r.cityAll }, ...cities.map((c) => ({ id: c.id, label: c.name }))]}
              value={city || "all"}
              onChange={(id) => setCity(id === "all" ? "" : id)}
            />
          ) : (
            <div className="sm:w-56">
              <Select aria-label={r.cityAll} value={city} onChange={(e) => setCity(e.target.value)}>
                <option value="">{r.cityAll}</option>
                {cities.map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.name}
                  </option>
                ))}
              </Select>
            </div>
          ))}
        <div className="space-y-1 text-2xs text-ink-muted">
          <p className="flex items-center gap-1.5">
            <IconHourglass size={12} />
            {r.byOrderDay}
          </p>
          {report?.previous_from && report.previous_to && (
            <p className="flex items-center gap-1.5">
              <IconCompare size={12} />
              {r.comparingWith
                .replace("{from}", dayText(report.previous_from))
                .replace("{to}", dayText(report.previous_to))}
            </p>
          )}
          {canExport && city && <p>{r.exportAllCities}</p>}
        </div>
      </section>

      {error && (
        <Alert>
          <span className="flex flex-wrap items-center gap-2">
            {error}
            <Button variant="secondary" onClick={() => void load()}>
              {r.retry}
            </Button>
          </span>
        </Alert>
      )}

      {loading && !report && (
        <div className="space-y-4">
          <SkeletonStats count={6} />
          <SkeletonList rows={4} />
        </div>
      )}

      {s && p && report && (
        <div className={`space-y-5 ${loading ? "opacity-60 transition-opacity" : "transition-opacity"}`}>
          {/* ── أرقامُ الطلبات — لكلّ من يفتح الصفحة، وكلُّ بطاقةٍ تفتح السجلَّ بالفترة نفسِها ── */}
          <Panel title={r.secOrders}>
            <div className="grid grid-cols-2 gap-2 lg:grid-cols-3">
              <StatTile
                icon={IconOrder}
                tone={s.orders_total > 0 ? "accent" : "muted"}
                label={r.summary.ordersTotal}
                value={fmtNum(s.orders_total)}
                delta={change(s.orders_total, p.orders_total)}
                onClick={goHistory()}
              />
              <StatTile
                icon={IconCheck}
                tone={s.delivered > 0 ? "success" : "muted"}
                label={r.summary.delivered}
                value={fmtNum(s.delivered)}
                sub={share(s.delivered, s.orders_total)}
                delta={change(s.delivered, p.delivered)}
                onClick={goHistory("delivered")}
              />
              <StatTile
                icon={IconBlock}
                tone={s.cancelled > 0 ? "danger" : "muted"}
                label={r.summary.cancelled}
                value={fmtNum(s.cancelled)}
                sub={share(s.cancelled, s.orders_total)}
                delta={change(s.cancelled, p.cancelled)}
                lowerIsBetter
                onClick={goHistory("cancelled")}
              />
              <StatTile
                icon={IconSwap}
                tone={s.refunded > 0 ? "warning" : "muted"}
                label={r.summary.refunded}
                value={fmtNum(s.refunded)}
                sub={share(s.refunded, s.orders_total)}
                delta={change(s.refunded, p.refunded)}
                lowerIsBetter
                onClick={goHistory("refunded")}
              />
              <StatTile
                icon={IconHourglass}
                tone={s.delivered > 0 ? "default" : "muted"}
                label={r.summary.avgOrderToDelivery}
                value={s.delivered > 0 ? `${fmtNum(s.avg_delivery_min)} ${r.summary.minutes}` : r.summary.noTime}
                sub={s.delivered > 0 ? undefined : r.summary.noDelivered}
                delta={
                  s.delivered > 0 && p.delivered > 0 ? change(s.avg_delivery_min, p.avg_delivery_min) : undefined
                }
                lowerIsBetter
              />
              <StatTile
                icon={IconUsers}
                tone={s.active_customers > 0 ? "default" : "muted"}
                label={r.summary.activeCustomers}
                value={fmtNum(s.active_customers)}
                delta={change(s.active_customers, p.active_customers)}
              />
            </div>
          </Panel>

          {/* ── المال — يصل من الخادم لمن يقرؤه وحدَه ── */}
          {money ? (
            <Panel title={r.secMoney}>
              <div className="grid grid-cols-2 gap-2 lg:grid-cols-3">
                {/* **الأصنافُ والتوصيلُ بطاقتان متجاورتان** (المالك ٢٠٢٦-١٠-٢٧) — الأصنافُ تساوي مجموعَ
                    مبيعات المتاجر تحت، والتوصيلُ لحاله، فلا يخلطهما رقمٌ واحد. */}
                <StatTile
                  icon={IconReceipt}
                  tone={(s.items_sales ?? 0) > 0 ? "accent" : "muted"}
                  label={r.summary.itemsSales}
                  value={<Money value={s.items_sales ?? 0} />}
                  sub={r.summary.itemsSalesHint}
                  delta={change(s.items_sales, p.items_sales)}
                  onClick={goHistory("delivered")}
                />
                <StatTile
                  icon={IconMoto}
                  label={r.summary.deliveryFees}
                  value={<Money value={s.delivery_fees ?? 0} />}
                  delta={change(s.delivery_fees, p.delivery_fees)}
                  onClick={goHistory("delivered")}
                />
                <StatTile
                  icon={profit < 0 ? IconTrendDown : IconWallet}
                  tone={profitTone}
                  label={profit < 0 ? r.summary.platformLoss : r.summary.platformProfit}
                  value={<Money value={profit} />}
                  sub={report.profit_unfiltered ? r.profitAllPlatform : r.profitHint}
                  delta={change(s.platform_profit, p.platform_profit)}
                  onClick={goProfits}
                />
                <StatTile
                  icon={IconStore}
                  label={r.summary.storeCommission}
                  value={<Money value={s.commissions ?? 0} />}
                  delta={change(s.commissions, p.commissions)}
                  onClick={goProfits}
                />
                <StatTile
                  icon={IconWallet}
                  label={r.summary.walletPaid}
                  value={<Money value={s.wallet_paid ?? 0} />}
                  delta={change(s.wallet_paid, p.wallet_paid)}
                />
                <StatTile
                  icon={IconBalance}
                  label={r.summary.cashCollected}
                  value={<Money value={s.cash_collected ?? 0} />}
                  delta={change(s.cash_collected, p.cash_collected)}
                  onClick={goCash}
                />
              </div>
            </Panel>
          ) : (
            <Alert tone="info">{r.moneyHidden}</Alert>
          )}

          {/* ── المخطّطات — كلُّ واحدٍ بلوحته ── */}
          <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
            <Panel title={r.dailyOrders}>
              <DailyBars data={report.daily} value={(d) => d.orders} format={(v) => fmtNum(v)} />
            </Panel>
            {money && (
              <Panel title={r.dailySales}>
                <DailyBars data={report.daily} value={(d) => d.items_sales ?? 0} format={(v) => fmtMoney(v)} />
              </Panel>
            )}
          </div>

          {/* ── الأفضلُ بالفترة — قوائمُ مرتّبةٌ لا جداول، وكلُّ سطرٍ يفتح ملفَّ صاحبه ── */}
          <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
            <Panel title={r.topMerchants} note={money ? r.storeSalesHint : undefined}>
              {report.top_merchants.length === 0 ? (
                <Empty icon={IconStore} title={r.noStores} hint={r.noDataHint} />
              ) : (
                <ol className="space-y-0.5">
                  {report.top_merchants.map((t, i) => (
                    <RankRow
                      key={t.id}
                      rank={i + 1}
                      href={`/dashboard/merchants/${t.id}`}
                      allowed={open.merchant}
                      name={t.name}
                      sub={`${fmtNum(t.delivered)} ${r.deliveries}`}
                      end={
                        t.sales !== undefined ? (
                          <span className="font-bold text-primary-dark">
                            <Money value={t.sales} />
                          </span>
                        ) : undefined
                      }
                    />
                  ))}
                </ol>
              )}
            </Panel>

            <Panel title={r.topDrivers}>
              {report.top_drivers.length === 0 ? (
                <Empty icon={IconDriver} title={r.noDrivers} hint={r.noDataHint} />
              ) : (
                <ol className="space-y-0.5">
                  {report.top_drivers.map((t, i) => (
                    <RankRow
                      key={t.id}
                      rank={i + 1}
                      href={`/dashboard/users/${t.id}`}
                      allowed={open.user}
                      name={t.name || (t.phone ? <span dir="ltr">{t.phone}</span> : r.noName)}
                      sub={
                        t.cash !== undefined ? (
                          <>
                            {r.summary.cashCollected} <Money value={t.cash} small />
                          </>
                        ) : undefined
                      }
                      end={
                        <>
                          <span className="font-bold tabular-nums">{fmtNum(t.delivered)}</span>{" "}
                          <span className="text-xs text-ink-muted">{r.deliveries}</span>
                        </>
                      }
                    />
                  ))}
                </ol>
              )}
            </Panel>
          </div>

          {/* **الزوّار** — أجهزةُ تطبيق الزبون وحدَها، بيوم الحساب، ولا يتبعون المدّة. */}
          <Panel title={m.admin.dash.visitors} note={r.visitorsHint}>
            {report.visitors ? (
              <div className="grid grid-cols-2 gap-2 lg:grid-cols-3">
                <StatTile icon={IconSmartphone} label={m.admin.dash.devicesToday} value={fmtNum(report.visitors.devices_today)} />
                <StatTile icon={IconSmartphone} label={m.admin.dash.devices7} value={fmtNum(report.visitors.devices_7d)} />
                <StatTile icon={IconSmartphone} label={m.admin.dash.devices30} value={fmtNum(report.visitors.devices_30d)} />
                <StatTile icon={IconView} label={m.admin.dash.opensToday} value={fmtNum(report.visitors.opens_today)} />
                <StatTile icon={IconView} label={m.admin.dash.opens7} value={fmtNum(report.visitors.opens_7d)} />
                <StatTile icon={IconView} label={m.admin.dash.opens30} value={fmtNum(report.visitors.opens_30d)} />
              </div>
            ) : (
              <Alert tone="warning">{r.visitorsFailed}</Alert>
            )}
          </Panel>
        </div>
      )}
    </PageContainer>
  );
}
