"use client";

/**
 * **مصروفاتُ التشغيل — ما ينفقه المكتبُ لا ما يخسره العمل.**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٦: «يوجد مكتبٌ للشركة وموظّفون وعمّال وما إلى ذلك من
 *  المصاريف — يجب أن تُوثَّق بشكلٍ صحيح».)
 *
 * **والإيجارُ ليس خسارة** — كلفةُ تشغيلٍ مخطَّطة، وتخرج من محفظة المنصّة.
 *
 * # قراراتُ المالك ٢٠٢٦-١٠-٠٤
 *
 * ١ · فوق سقف الموافقة (من الإعدادات) يصير المصروفُ اقتراحاً يوافق عليه موظّفٌ آخر.
 * ٢ · الشهرُ بتاريخ الصرف — هنا وفي صفحة الأرباح.
 * ٣ · الصفحةُ بمسارها، ويربطها قسمُ الخزينة تبويباً.
 * ٤ · صورةُ الإيصال اختياريّة، وإلزاميّةٌ فوق السقف.
 * ٥ · الإلغاءُ بسببٍ ومن موظّفٍ غيرِ من سجّل، والملغى يبقى ظاهراً مشطوباً.
 *
 * # قرارُ المالك ٢٠٢٦-١٠-١٥ — «منين دفعت؟»
 *
 * **كاشٌ من الدرج** (الافتراض) يخرج من محفظة المنصّة ومن درج المكتب حين يُقيَّد، وإلغاؤه يردّه
 * للاثنين. **ومن برّا الدرج** يمسّ المحفظةَ وحدَها. والطريقةُ تُحفظ على الصفّ (`paid_from`).
 *
 * # قراراتُ المالك ٢٠٢٦-١٠-١٥
 *
 * - **اسمُ الموظّف** على المصروف — إلزاميٌّ في باب الرواتب (`is_salary`)، ويُبحث به،
 *   ومجموعُ كلّ موظّفٍ في المدى حين يُختار بابُ الرواتب.
 * - **«شهري متكرر»** علَمٌ على الباب — تذكيرٌ أوّلَ كلّ شهر.
 * - **الموافقةُ والرفضُ بتأكيد**، والرفضُ بسببٍ يُعرض على صفّه.
 * - **الأزرارُ من الخادم** (`can_void` · `can_approve`) — لا تُعرض ما يردّه.
 * - **البطاقاتُ تتبع البابَ والبحث** · والترتيبُ نداءٌ واحد.
 *
 * # قرارُ المالك ٢٠٢٦-١٠-٢٧
 *
 * **مصروفُ المالك أو المدير فوق السقف ينفُذ فوراً** ويُعلَّم «وافق على حاله» على كرته.
 *
 * # التصميمُ الاحترافيّ (٢٠٢٦-١٠-٢٧)
 *
 * أربعةُ أرقامٍ تُضغط فترشّح · المدّةُ والحالةُ حبّات · لونٌ ثابتٌ لكلّ باب · كرتٌ لكلّ مصروف
 * بلونٍ هادئ (الأحمرُ للخطر وحدَه) · رسائلُ نجاحٍ تمّحي وحدَها · ونافذةُ أبوابٍ بكروتٍ ومفاتيح.
 */

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import {
  getMessages,
  defaultLocale,
  fmtNum,
  fmtMoney,
  fmtDate,
  fmtLongDate,
  errorText,
  westernDigits,
  damascusDay,
} from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  Chips,
  Input,
  Select,
  Modal,
  Textarea,
  PageContainer,
  LoadingState,
  ReloadState,
  Pagination,
  Switch,
  ImageUpload,
  IconWallet,
  IconAdd,
  IconDate,
  IconUser,
  IconNote,
  IconReceipt,
  IconSearch,
  IconTrendUp,
  IconTrendDown,
  IconTrendFlat,
  IconEdit,
  IconHourglass,
  IconArrowIn,
  IconArrowOut,
  IconClose,
  FormActions,
} from "@rahalgo/ui";
import { api, apiFile, mediaUrl } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { PageHero } from "@/components/admin/PageHero";
import { Panel, Tile } from "@/components/admin/DashParts";

const m = getMessages(defaultLocale);
const X = m.admin.expenses;
const CUR = m.common.currency;
/** **الفاصلةُ العربيّة «،» وعلامةُ الناقص** — برمزها لا بحرفها، فلا نصَّ عربيّاً في الشيفرة. */
const ARABIC_COMMA = String.fromCharCode(0x060c);
const MINUS = String.fromCharCode(0x2212);

type Status = "posted" | "voided" | "pending" | "rejected";

interface Row {
  id: string;
  source: "expense" | "request";
  status: Status;
  category: string;
  amount: number;
  note: string;
  spent_at: string;
  by: string;
  by_id: string;
  /** **من ألغى أو وافق أو رفض** — بحسب الحالة (للمقيَّد: من وافق فوق السقف). */
  decider: string;
  decided_at: string | null;
  reason: string;
  /** **وافق على حاله** — مصروفُ المالك أو المدير فوق السقف (قرارُ ٢٠٢٦-١٠-٢٧). */
  self_approved: boolean;
  receipt_url: string | null;
  /** **منين دفعت؟** — كاشٌ من الدرج أو من برّا الدرج. */
  paid_from?: PaidFrom;
  /** **اسمُ الموظّف** — إلزاميٌّ في باب الرواتب. */
  employee_name?: string;
  /** **أعلامُ الأزرار من الخادم** — بحكم الموافقة نفسِه، لا بمقارنة معرّفات. */
  can_void?: boolean;
  can_approve?: boolean;
}
type PaidFrom = "cash" | "outside";
interface Cat {
  id: string;
  name: string;
  active: boolean;
  sort_order: number;
  /** **بابُ الرواتب** — اسمُ الموظّف فيه إلزاميّ. */
  is_salary?: boolean;
  /** **مصروفٌ شهريٌّ متكرّر** — يُذكَّر به أوّلَ الشهر. */
  monthly_recurring?: boolean;
  used: number;
  spent: number;
}
interface Data {
  expenses: Row[];
  total: number;
  total_amount: number;
  pending_amount: number;
  pending_count: number;
  page: number;
  per_page: number;
  by_category: { id: string; name: string; count: number; total: number }[];
  /** **مجموعُ كلّ موظّفٍ في المدى** — المقيَّدُ وحدَه وبالمرشّحات نفسِها. */
  by_employee?: { name: string; count: number; total: number }[];
  summary: {
    /** **الشهرُ شهرُ نهاية المدى** — يقوله عنوانُ البطاقة باسمه. */
    month_from?: string;
    month_to?: string;
    month_total: number;
    prev_month_total: number;
    top_category: string;
    top_category_total: number;
  };
  threshold: number;
  treasury_balance: number;
  today: string;
  account_day?: string;
}

type Preset = "month" | "prev" | "three" | "custom";
const PRESETS: Preset[] = ["month", "prev", "three", "custom"];
type Range = { from: string; to: string };

const pad = (n: number) => String(n).padStart(2, "0");

/** **الشهرُ الذي فيه اليوم `day` مزاحاً بـ`offset`** — أوّلُه وآخرُه بصيغة `YYYY-MM-DD`. */
function monthAt(day: string, offset = 0): Range {
  const [y, mo] = day.split("-").map(Number);
  const d = new Date(Date.UTC(y!, mo! - 1 + offset, 1));
  const yy = d.getUTCFullYear();
  const mm = d.getUTCMonth() + 1;
  const last = new Date(Date.UTC(yy, mm, 0)).getUTCDate();
  return { from: `${yy}-${pad(mm)}-01`, to: `${yy}-${pad(mm)}-${pad(last)}` };
}

/** **أزرارُ المدّة** — هالشهر · الماضي · آخر ٣ شهور، بيوم دمشق لا بيوم المتصفّح. */
function presetRange(p: Exclude<Preset, "custom">, today: string): Range {
  if (p === "prev") return monthAt(today, -1);
  if (p === "three") return { from: monthAt(today, -2).from, to: monthAt(today, 0).to };
  return monthAt(today, 0);
}

function matchPreset(r: Range, today: string): Preset {
  for (const p of ["month", "prev", "three"] as const) {
    const x = presetRange(p, today);
    if (x.from === r.from && x.to === r.to) return p;
  }
  return "custom";
}

/** **اسمُ الشهر** — «تشرين الأول 2026» من أوّل يومٍ فيه. */
function monthName(day: string): string {
  return fmtLongDate(`${day.slice(0, 7)}-15`).replace(/^\d+\s+/, "");
}

/**
 * **قراءةُ المبلغ بالأرقام العربيّة والفواصل** — «١٥٠٬٠٠٠» و«150,000» و«150 000» سواء.
 * ويردّ `null` مع سببٍ يُقال — وكان الحفظُ يسكت فلا يحدث شيء.
 */
function parseAmount(raw: string): { value: number | null; error: string } {
  const s = westernDigits(raw).replace(/[\s,_]/g, "").split(ARABIC_COMMA).join("");
  if (s === "") return { value: null, error: X.amountZero };
  if (!/^\d+$/.test(s)) return { value: null, error: X.amountInvalid };
  const n = Number(s);
  if (!Number.isSafeInteger(n)) return { value: null, error: X.amountInvalid };
  if (n <= 0) return { value: null, error: X.amountZero };
  return { value: n, error: "" };
}

const statusVariant: Record<Status, "success" | "neutral" | "warning" | "danger"> = {
  posted: "success",
  voided: "neutral",
  pending: "warning",
  rejected: "danger",
};

/** **لونٌ ثابتٌ لكلّ باب** — بموضعه في قائمة الأبواب، فلا يتبدّل لونُ بابٍ بتبدّل ترتيب المبالغ. */
const CAT_COLORS = [
  "bg-primary",
  "bg-accent",
  "bg-info",
  "bg-violet",
  "bg-warning",
  "bg-success",
  "bg-sky-solid",
  "bg-ink-muted",
];

/** **مصروفُ المالك أو المدير ينفُذ فوراً** — يطابق `approval.AutoApplies` في المحرّك. */
const AUTO_ROLES = ["owner_super_admin", "admin"];

export default function ExpensesPage() {
  const [today] = useState(() => damascusDay());
  const [range, setRange] = useState<Range>(() => monthAt(today, 0));
  const [customOpen, setCustomOpen] = useState(false);
  const [category, setCategory] = useState("");
  const [status, setStatus] = useState<"" | Status>("");
  // **البحثُ بتأخيرٍ صغير** — كان كلُّ حرفٍ نداءً، **وجوابٌ قديمٌ يصل بعد الجديد يُعرض.**
  const [qInput, setQInput] = useState("");
  const [q, setQ] = useState("");
  const [page, setPage] = useState(1);
  const [data, setData] = useState<Data | null | "failed">(null);
  const [refreshing, setRefreshing] = useState(false);
  const [loadErr, setLoadErr] = useState("");
  const [cats, setCats] = useState<Cat[]>([]);
  const [adding, setAdding] = useState(false);
  const [catsOpen, setCatsOpen] = useState(false);
  const [error, setError] = useState("");
  const [notice, setNotice] = useState("");
  const { can, user } = useAuth();
  const canManage = can("finance.manage");
  const canExport = can("finance.export");
  const meId = user?.id ?? "";
  const autoApplies = (user?.roles ?? []).some((r) => AUTO_ROLES.includes(r));
  const [voiding, setVoiding] = useState<Row | null>(null);
  // **الموافقةُ والرفضُ بتأكيد** — والرفضُ بسببٍ إلزاميّ (قرارُ المالك ٢٠٢٦-١٠-١٥).
  // **ورفضُ المقترِح اقتراحَه سحبٌ لا رفض** — يُسمّى باسمه.
  const [deciding, setDeciding] = useState<{ row: Row; mode: DecideMode } | null>(null);
  const seq = useRef(0);

  useEffect(() => {
    const t = setTimeout(() => {
      setQ(qInput);
      setPage(1);
    }, 300);
    return () => clearTimeout(t);
  }, [qInput]);

  // **ورسالةُ النجاح تمّحي وحدَها** — كانت تبقى للأبد فوق الصفحة.
  useEffect(() => {
    if (!notice) return;
    const t = setTimeout(() => setNotice(""), 5000);
    return () => clearTimeout(t);
  }, [notice]);

  // **ومدًى مقلوبٌ يُقال** — وكان يعطي قائمةً فارغةً بلا تنبيه.
  const inverted = range.from !== "" && range.to !== "" && range.from > range.to;

  const qs = useMemo(() => {
    const p = new URLSearchParams({ from: range.from, to: range.to, page: String(page) });
    if (category) p.set("category_id", category);
    if (status) p.set("status", status);
    if (q.trim()) p.set("q", q.trim());
    return p.toString();
  }, [range, page, category, status, q]);

  const load = useCallback(() => {
    if (inverted) return;
    // **والجوابُ القديمُ يُهمل** — لا يُعرض إلّا جوابُ آخرِ نداء.
    const my = ++seq.current;
    setRefreshing(true);
    api<Data>(`/api/v1/admin/expenses?${qs}`)
      .then((d) => {
        if (my !== seq.current) return;
        setLoadErr("");
        setData(d);
      })
      .catch((e) => {
        if (my !== seq.current) return;
        setLoadErr(errorText(e));
        // **وما عُرض قبلاً يبقى** مع رسالةٍ فوقه — لا تُمحى الصفحةُ لسقوط تحديثٍ واحد.
        setData((prev) => (prev && prev !== "failed" ? prev : "failed"));
        setError(errorText(e));
      })
      .finally(() => {
        if (my === seq.current) setRefreshing(false);
      });
  }, [qs, inverted]);

  const loadCats = useCallback(() => {
    api<{ categories: Cat[] }>("/api/v1/admin/expenses/categories")
      .then((r) => setCats(r.categories ?? []))
      .catch((err) => setError(errorText(err)));
  }, []);

  useEffect(load, [load]);
  useEffect(loadCats, [loadCats]);

  function exportCsv() {
    setError("");
    void (async () => {
      try {
        const p = new URLSearchParams(qs);
        p.delete("page");
        const res = await apiFile(`/api/v1/admin/expenses/export?${p.toString()}`);
        const blob = await res.blob();
        const url = URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = `expenses-${range.from}_${range.to}.csv`;
        a.click();
        URL.revokeObjectURL(url);
      } catch (err) {
        setError(errorText(err));
      }
    })();
  }

  function pickRange(r: Range) {
    setRange(r);
    setCustomOpen(false);
    setPage(1);
  }
  function pickStatus(v: "" | Status) {
    setStatus(v);
    setPage(1);
  }
  function pickCategory(v: string) {
    setCategory(v);
    setPage(1);
  }
  function clearFilters() {
    setCategory("");
    setStatus("");
    setQInput("");
    setQ("");
    pickRange(monthAt(today, 0));
  }
  /** **كلُّ فعلٍ جديدٍ يمسح ما قبله** — كان الخطأُ القديمُ يبقى فوق الصفحة. */
  function begin(fn: () => void) {
    setError("");
    setNotice("");
    fn();
  }

  if (data === "failed") return <ReloadState onRetry={load} label={loadErr || undefined} />;
  if (!data) return <LoadingState />;

  const s = data.summary;
  const diff = s.month_total - s.prev_month_total;
  const monthFrom = s.month_from || monthAt(range.to || today, 0).from;
  const monthLabel = monthName(monthFrom);
  const salaryCat = cats.find((c) => c.id === category)?.is_salary === true;
  const byEmployee = data.by_employee ?? [];
  const activePreset: Preset = customOpen ? "custom" : matchPreset(range, today);
  const postedCount = data.by_category.reduce((n, c) => n + c.count, 0);
  const catColor = (id: string, i: number) => {
    const at = cats.findIndex((c) => c.id === id);
    return CAT_COLORS[(at >= 0 ? at : i) % CAT_COLORS.length]!;
  };

  const filtered = category !== "" || status !== "" || q.trim() !== "";
  const anyFilter = filtered || activePreset !== "month";
  const rows = inverted ? [] : data.expenses;

  const statusChips: { id: "all" | Status; label: string; count?: number }[] = [
    { id: "all", label: X.statusAll },
    { id: "posted", label: X.status.posted },
    { id: "pending", label: X.status.pending, count: data.pending_count > 0 ? data.pending_count : undefined },
    { id: "voided", label: X.status.voided },
    { id: "rejected", label: X.status.rejected },
  ];

  return (
    <PageContainer>
      <PageHero
        icon={IconWallet}
        title={X.title}
        subtitle={X.hint}
        actions={
          <div className="flex flex-wrap gap-2">
            {canExport && (
              <Button variant="secondary" onClick={exportCsv} disabled={inverted}>
                {X.export}
              </Button>
            )}
            {canManage && (
              <>
                <Button variant="secondary" onClick={() => begin(() => setCatsOpen(true))}>
                  {X.manageCats}
                </Button>
                <Button onClick={() => begin(() => setAdding(true))} className="flex items-center gap-1.5">
                  <IconAdd size={16} />
                  {X.add}
                </Button>
              </>
            )}
          </div>
        }
      />

      {error && <Alert className="mb-3">{error}</Alert>}
      {notice && (
        <Alert tone="success" className="mb-3">
          {notice}
        </Alert>
      )}

      {/* ── أربعةُ أرقامٍ تُضغط فترشّح ── */}
      <div className="mb-4 grid grid-cols-2 gap-2 lg:grid-cols-4">
        <Tile
          icon={IconWallet}
          label={X.periodTotal}
          value={fmtMoney(data.total_amount)}
          sub={X.usedCount.replace("{n}", fmtNum(postedCount))}
          tone={data.total_amount > 0 ? "default" : "muted"}
          onClick={() => pickStatus("posted")}
        />
        <Tile
          icon={IconDate}
          label={X.monthTotalOf.replace("{month}", monthLabel)}
          value={fmtMoney(s.month_total)}
          sub={
            s.top_category
              ? X.topCategoryOf.replace("{name}", s.top_category).replace("{amount}", fmtMoney(s.top_category_total))
              : undefined
          }
          tone={s.month_total > 0 ? "accent" : "muted"}
          onClick={() => {
            pickRange(monthAt(monthFrom, 0));
            pickStatus("posted");
          }}
        />
        <Tile
          icon={diff > 0 ? IconTrendUp : diff < 0 ? IconTrendDown : IconTrendFlat}
          label={X.vsLastMonth}
          value={`${diff > 0 ? "+" : ""}${fmtMoney(diff)}`}
          sub={X.lastMonthWas.replace("{n}", fmtMoney(s.prev_month_total))}
          tone={diff > 0 ? "warning" : diff < 0 ? "success" : "muted"}
          onClick={() => {
            pickRange(monthAt(monthFrom, -1));
            pickStatus("posted");
          }}
        />
        <Tile
          icon={IconHourglass}
          label={X.pendingTile}
          value={data.pending_count > 0 ? fmtMoney(data.pending_amount) : fmtNum(0)}
          sub={data.pending_count > 0 ? X.usedCount.replace("{n}", fmtNum(data.pending_count)) : X.noneWaiting}
          tone={data.pending_count > 0 ? "warning" : "muted"}
          onClick={() => pickStatus("pending")}
        />
      </div>

      {/* ── الفلاتر في كرتٍ واحد: المدّةُ والحالةُ حبّات، والبابُ قائمةٌ لأنّ الأبوابَ كثيرة ── */}
      <section className="surface mb-4 flex flex-col gap-3 p-3 sm:p-4">
        <div className="flex flex-col gap-1.5">
          <span className="text-xs text-ink-muted">{X.periodLabel}</span>
          <Chips<Preset>
            items={PRESETS.map((p) => ({ id: p, label: X.period[p] }))}
            value={activePreset}
            onChange={(p) => {
              if (p === "custom") {
                setCustomOpen(true);
                return;
              }
              pickRange(presetRange(p, today));
            }}
            wrap
          />
          {activePreset === "custom" && (
            <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
              <Input
                id="exp-from"
                type="date"
                icon={<IconDate />}
                label={m.shared.statement.from}
                value={range.from}
                onChange={(e) => {
                  setRange({ ...range, from: e.target.value });
                  setPage(1);
                }}
              />
              <Input
                id="exp-to"
                type="date"
                icon={<IconDate />}
                label={m.shared.statement.to}
                value={range.to}
                onChange={(e) => {
                  setRange({ ...range, to: e.target.value });
                  setPage(1);
                }}
              />
            </div>
          )}
        </div>
        <div className="flex flex-col gap-1.5 border-t border-line-soft pt-3">
          <span className="text-xs text-ink-muted">{X.statusLabel}</span>
          <Chips<"all" | Status>
            items={statusChips}
            value={status || "all"}
            onChange={(k) => pickStatus(k === "all" ? "" : k)}
            wrap
          />
        </div>
        <div className="grid grid-cols-1 gap-2 border-t border-line-soft pt-3 sm:grid-cols-2">
          <Select
            id="exp-filter-cat"
            label={X.colCategory}
            value={category}
            onChange={(e) => pickCategory(e.target.value)}
          >
            <option value="">{X.allCategories}</option>
            {cats.map((c) => (
              <option key={c.id} value={c.id}>
                {c.active ? c.name : `${c.name} ${X.catOffSuffix}`}
              </option>
            ))}
          </Select>
          <Input
            id="exp-search"
            icon={<IconSearch size={15} />}
            label={X.search}
            value={qInput}
            onChange={(e) => setQInput(e.target.value)}
          />
        </div>
        {(anyFilter || refreshing) && (
          <div className="flex flex-wrap items-center justify-between gap-2">
            <span className="text-xs text-ink-muted" aria-live="polite">
              {refreshing ? X.refreshing : ""}
            </span>
            {anyFilter && (
              <Button variant="ghost" className="flex items-center gap-1 !px-2 text-xs" onClick={clearFilters}>
                <IconClose size={13} />
                {X.clearFilters}
              </Button>
            )}
          </div>
        )}
      </section>

      {inverted && <Alert className="mb-3">{X.invertedRange}</Alert>}

      <div className={`transition-opacity ${refreshing ? "opacity-60" : ""}`} aria-busy={refreshing}>
        {/* **ومجموعُ كلّ موظّفٍ حين يُختار بابُ الرواتب** (قرارُ المالك ٢٠٢٦-١٠-١٥). */}
        {salaryCat && byEmployee.length > 0 && (
          <Panel title={X.byEmployeeTitle} className="mb-4">
            <ul className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
              {byEmployee.map((e) => (
                <li key={e.name}>
                  <button
                    type="button"
                    onClick={() => setQInput(e.name)}
                    className="flex w-full items-center gap-3 rounded-control bg-field px-3 py-2 text-start transition-colors hover:bg-row-hover"
                  >
                    <span className="inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full bg-surface text-ink">
                      <IconUser size={16} />
                    </span>
                    <span className="min-w-0 flex-1">
                      <span className="block truncate font-medium">{e.name}</span>
                      <span className="block text-2xs text-ink-muted">{X.usedCount.replace("{n}", fmtNum(e.count))}</span>
                    </span>
                    <span dir="ltr" className="whitespace-nowrap font-bold tabular-nums">
                      {fmtMoney(e.total)}
                    </span>
                  </button>
                </li>
              ))}
            </ul>
          </Panel>
        )}

        {/* **والتوزيعُ على الأبواب بالمبلغ** — لونٌ ثابتٌ لكلّ باب، وكلُّ بابٍ يُضغط فيرشّح. */}
        {data.by_category.length > 0 && data.total_amount > 0 && (
          <Panel title={X.byCategoryTitle} note={fmtMoney(data.total_amount)} className="mb-4">
            <div className="mb-3 flex h-3 w-full overflow-hidden rounded-full bg-line">
              {data.by_category.map((c, i) => (
                <span
                  key={c.id}
                  title={`${c.name} · ${fmtMoney(c.total)}`}
                  className={catColor(c.id, i)}
                  style={{ width: `${(c.total / data.total_amount) * 100}%` }}
                />
              ))}
            </div>
            <ul className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
              {data.by_category.map((c, i) => {
                const on = category === c.id;
                return (
                  <li key={c.id}>
                    <button
                      type="button"
                      aria-pressed={on}
                      onClick={() => pickCategory(on ? "" : c.id)}
                      className={`flex w-full items-center gap-2 rounded-control border px-3 py-2 text-start text-sm transition-colors ${
                        on ? "border-primary bg-primary-tint" : "border-transparent bg-field hover:bg-row-hover"
                      }`}
                    >
                      <span aria-hidden className={`h-2.5 w-2.5 shrink-0 rounded-full ${catColor(c.id, i)}`} />
                      <span className="min-w-0 flex-1">
                        <span className="block truncate font-medium">{c.name}</span>
                        <span className="block text-2xs text-ink-muted">
                          {X.usedCount.replace("{n}", fmtNum(c.count))} ·{" "}
                          {fmtNum(Math.round((c.total / data.total_amount) * 100))}%
                        </span>
                      </span>
                      <span dir="ltr" className="whitespace-nowrap font-bold tabular-nums">
                        {fmtMoney(c.total)}
                      </span>
                    </button>
                  </li>
                );
              })}
            </ul>
            <p className="mt-2 text-2xs text-ink-muted">{X.byCategoryTap}</p>
          </Panel>
        )}

        {rows.length === 0 ? (
          <section className="surface flex flex-col items-center gap-3 p-8 text-center">
            <span
              className={`inline-flex h-16 w-16 items-center justify-center rounded-full ${
                anyFilter ? "bg-info-solid" : "bg-success-solid"
              } text-on-solid`}
            >
              <IconReceipt size={28} />
            </span>
            <h2 className="heading-section">{filtered ? X.emptyFiltered : X.empty}</h2>
            <p className="max-w-md text-sm text-ink-muted">
              {anyFilter ? X.emptyFilteredHint : canManage ? X.emptyHint : X.emptyHintRead}
            </p>
            {anyFilter && (
              <Button variant="secondary" onClick={clearFilters}>
                {X.clearFilters}
              </Button>
            )}
          </section>
        ) : (
          <ul className="grid grid-cols-1 gap-3 lg:grid-cols-2 2xl:grid-cols-3">
            {rows.map((e) => (
              <ExpenseCard
                key={e.id}
                e={e}
                canManage={canManage}
                own={meId !== "" && e.by_id === meId}
                onVoid={() => begin(() => setVoiding(e))}
                onDecide={(mode) => begin(() => setDeciding({ row: e, mode }))}
              />
            ))}
          </ul>
        )}
      </div>

      {data.total > data.per_page && (
        <div className="mt-4 flex justify-center">
          <Pagination page={page} total={data.total} perPage={data.per_page} onChange={setPage} />
        </div>
      )}

      {adding && (
        <AddModal
          cats={cats.filter((c) => c.active)}
          threshold={data.threshold}
          balance={data.treasury_balance}
          today={data.today}
          accountDay={data.account_day}
          autoApplies={autoApplies}
          onClose={() => setAdding(false)}
          onDone={(st) => {
            setAdding(false);
            setNotice(st === "pending" ? X.sentForApproval : X.donePosted);
            load();
            loadCats();
          }}
        />
      )}
      {catsOpen && (
        <CatsModal
          cats={cats}
          onClose={() => setCatsOpen(false)}
          onChanged={() => {
            loadCats();
            load();
          }}
        />
      )}
      {deciding && (
        <DecideModal
          row={deciding.row}
          mode={deciding.mode}
          onClose={() => setDeciding(null)}
          onDone={() => {
            const mode = deciding.mode;
            setDeciding(null);
            setNotice(mode === "approve" ? X.doneApproved : mode === "withdraw" ? X.doneWithdrawn : X.doneRejected);
            load();
            loadCats();
          }}
        />
      )}
      {voiding && (
        <VoidModal
          row={voiding}
          onClose={() => setVoiding(null)}
          onDone={() => {
            setVoiding(null);
            setNotice(X.doneVoided);
            load();
            loadCats();
          }}
        />
      )}
    </PageContainer>
  );
}

/** **لونُ الدائرة بالحالة** — المقيَّدُ شيءٌ طبيعيّ فلونُه هادئ، والأصفرُ لما ينتظر. */
const STATUS_TINT: Record<Status, string> = {
  posted: "bg-field text-ink",
  pending: "bg-warning-solid text-on-solid",
  voided: "bg-field text-ink-muted",
  rejected: "bg-field text-ink-muted",
};

type DecideMode = "approve" | "reject" | "withdraw";

/**
 * **كرتُ المصروف**: الباب ومنين انصرف وحالته فوق، المبلغ كبير بخلفيّة، ثمّ التاريخ ومين سجّله
 * ومين وافق والموظّف والإيصال، والملاحظة، والأفعال بعرض الكرت.
 * **والأزرارُ بأعلام الخادم نفسِها** (`can_void` · `can_approve`) كما كانت.
 */
function ExpenseCard({
  e,
  canManage,
  own,
  onVoid,
  onDecide,
}: {
  e: Row;
  canManage: boolean;
  own: boolean;
  onVoid: () => void;
  onDecide: (mode: DecideMode) => void;
}) {
  const dead = e.status === "voided" || e.status === "rejected";
  const showVoid = canManage && e.status === "posted" && e.can_void === true;
  const showDecide = canManage && e.status === "pending";
  const withdrawn = e.status === "rejected" && e.decider !== "" && e.decider === e.by;
  const when = e.decided_at ? ` · ${X.decidedOn.replace("{date}", fmtDate(e.decided_at))}` : "";
  return (
    <li className="surface flex flex-col gap-3 p-4">
      <div className="flex items-start gap-3">
        <span className={`inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-full ${STATUS_TINT[e.status]}`}>
          <IconReceipt size={20} />
        </span>
        <span className="min-w-0 flex-1">
          <span className="block truncate font-bold">{e.category}</span>
          {e.employee_name && (
            <span className="flex items-center gap-1 truncate text-sm text-ink-muted">
              <IconUser size={13} />
              {e.employee_name}
            </span>
          )}
        </span>
        <span className="flex shrink-0 flex-col items-end gap-1">
          <Badge variant={statusVariant[e.status]}>{X.status[e.status]}</Badge>
          {e.paid_from && (
            <Badge variant={e.paid_from === "cash" ? "warning" : "neutral"}>{X.paidFromOptions[e.paid_from]}</Badge>
          )}
          {e.status === "posted" && e.self_approved && <Badge variant="warning">{X.selfApproved}</Badge>}
        </span>
      </div>

      <div className="flex items-end justify-between gap-2 rounded-control bg-field px-3 py-2.5">
        <span>
          <span className="block text-2xs text-ink-muted">{X.colAmount}</span>
          <span dir="ltr" className={`figure block ${dead ? "text-ink-muted line-through" : ""}`}>
            {`${MINUS}${fmtMoney(e.amount)}`}
          </span>
        </span>
        <span className="text-end text-sm">
          <span className="block text-2xs text-ink-muted">{X.colDate}</span>
          <b dir="ltr" className="whitespace-nowrap tabular-nums">
            {fmtDate(e.spent_at)}
          </b>
        </span>
      </div>

      {(e.note || (dead && e.reason)) && (
        <div className="space-y-1 text-sm">
          {e.note && (
            <p className={`flex items-start gap-1.5 ${e.status === "voided" ? "text-ink-muted line-through" : ""}`}>
              <IconNote size={14} className="mt-0.5 shrink-0 text-ink-muted" />
              <span className="min-w-0">{e.note}</span>
            </p>
          )}
          {e.status === "voided" && e.reason && (
            <p className="text-xs text-ink-muted">
              {X.voidedBy.replace("{name}", e.decider).replace("{reason}", e.reason)}
              {when}
            </p>
          )}
          {e.status === "rejected" && e.reason && (
            <p className={`text-xs ${withdrawn ? "text-ink-muted" : "text-danger"}`}>
              {(withdrawn ? X.withdrawnBy : X.rejectedBy).replace("{name}", e.decider).replace("{reason}", e.reason)}
              {when}
            </p>
          )}
        </div>
      )}

      <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-2xs text-ink-muted">
        <span className="flex items-center gap-1">
          <IconUser size={12} />
          {X.colBy}: {e.by}
        </span>
        {e.status === "posted" && e.decider && <span>{X.approvedBy.replace("{name}", e.decider)}</span>}
        {e.receipt_url && (
          <a
            href={mediaUrl(e.receipt_url) ?? "#"}
            target="_blank"
            rel="noreferrer"
            className="flex items-center gap-1 text-primary underline"
          >
            <IconReceipt size={12} />
            {X.colReceipt}: {X.viewReceipt}
          </a>
        )}
      </div>

      {(showVoid || showDecide) && (
        <div className="mt-auto flex gap-2 border-t border-line-soft pt-3">
          {showVoid && (
            <Button className="flex-1" variant="secondary" onClick={onVoid}>
              {X.void}
            </Button>
          )}
          {showDecide && e.can_approve && (
            <Button className="flex-1" onClick={() => onDecide("approve")}>
              {X.approve}
            </Button>
          )}
          {showDecide && (
            <Button
              className="flex-1"
              variant={own ? "secondary" : "danger"}
              onClick={() => onDecide(own ? "withdraw" : "reject")}
            >
              {own ? X.withdraw : X.reject}
            </Button>
          )}
        </div>
      )}
    </li>
  );
}

/**
 * **الموافقةُ أو الرفضُ بتأكيد** (قرارُ المالك ٢٠٢٦-١٠-١٥) — كان الزرُّ يقرّر بضغطةٍ
 * واحدة. **والرفضُ بسببٍ إلزاميّ** يُحفظ ويُعرض على الصفّ المرفوض.
 * **وسحبُ المقترِح اقتراحَه** النداءُ نفسُه بلفظٍ يقول ما هو.
 */
function DecideModal({
  row,
  mode,
  onClose,
  onDone,
}: {
  row: Row;
  mode: DecideMode;
  onClose: () => void;
  onDone: () => void;
}) {
  const approve = mode === "approve";
  const withdraw = mode === "withdraw";
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  async function submit() {
    if (busy || (!approve && !reason.trim())) return;
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/admin/expense-requests/${row.id}/${approve ? "approve" : "reject"}`, {
        method: "POST",
        body: JSON.stringify(approve ? {} : { note: reason.trim() }),
      });
      onDone();
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  }
  return (
    <Modal
      open
      onClose={onClose}
      title={approve ? X.approveConfirmTitle : withdraw ? X.withdrawConfirmTitle : X.rejectConfirmTitle}
    >
      <div className="space-y-3">
        <p className="text-sm text-ink-muted">
          {approve ? X.approveConfirmBody : withdraw ? X.withdrawConfirmBody : X.rejectConfirmBody}
        </p>
        <p className="text-sm">
          {row.category}
          {row.employee_name ? ` · ${row.employee_name}` : ""} ·{" "}
          <span dir="ltr" className="font-bold">
            {fmtMoney(row.amount)}
          </span>
        </p>
        {!approve && (
          <Textarea
            id="exp-reject-reason"
            label={withdraw ? X.withdrawReason : X.rejectReason}
            required
            rows={2}
            value={reason}
            onChange={(e) => setReason(e.target.value)}
          />
        )}
        {error && <Alert>{error}</Alert>}
        <FormActions
          onSave={() => void submit()}
          onCancel={onClose}
          busy={busy}
          saveLabel={approve ? X.approveConfirm : withdraw ? X.withdrawConfirm : X.rejectConfirm}
          cancelLabel={X.back}
        />
      </div>
    </Modal>
  );
}

/** **الإلغاءُ بسببٍ إلزاميّ** — ومن موظّفٍ غيرِ من سجّل ولا من وافق (يحكم الخادم ويقول السبب). */
function VoidModal({ row, onClose, onDone }: { row: Row; onClose: () => void; onDone: () => void }) {
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  async function submit() {
    if (!reason.trim() || busy) return;
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/admin/expenses/${row.id}/void`, {
        method: "POST",
        body: JSON.stringify({ reason: reason.trim() }),
      });
      onDone();
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  }
  return (
    <Modal open onClose={onClose} title={X.voidConfirmTitle}>
      <div className="space-y-3">
        <p className="text-sm text-ink-muted">{X.voidConfirmBody}</p>
        <p className="text-sm">
          {row.category} ·{" "}
          <span dir="ltr" className="font-bold">
            {fmtMoney(row.amount)}
          </span>
        </p>
        <Textarea
          id="exp-void-reason"
          label={X.voidReason}
          required
          rows={2}
          value={reason}
          onChange={(e) => setReason(e.target.value)}
        />
        {error && <Alert>{error}</Alert>}
        <FormActions
          onSave={() => void submit()}
          onCancel={onClose}
          busy={busy}
          saveLabel={X.voidConfirm}
          cancelLabel={X.back}
        />
      </div>
    </Modal>
  );
}

/**
 * **تسجيلٌ يدويٌّ على خطوتين** — النموذجُ ثمّ مراجعةُ المبلغ ورصيدِ المحفظة والدرج بعده.
 *
 * **والنموذجُ يبقى مركَّباً وقتَ المراجعة** (مخفيّاً) — كان يُزال فتضيع معاينةُ الإيصال
 * عند «رجوع» والإيصالُ ما زال مربوطاً، فيظنّ المستخدمُ أنّه ضاع.
 */
function AddModal({
  cats,
  threshold,
  balance,
  today,
  accountDay,
  autoApplies,
  onClose,
  onDone,
}: {
  cats: Cat[];
  threshold: number;
  balance: number;
  today: string;
  accountDay?: string;
  autoApplies: boolean;
  onClose: () => void;
  onDone: (status: string) => void;
}) {
  const [categoryID, setCategoryID] = useState(cats[0]?.id ?? "");
  const [amount, setAmount] = useState("");
  const [note, setNote] = useState("");
  // **ويومُ الصرف الافتراضيُّ يومُ الحساب** (قرارُ المالك ٢٠٢٦-١٠-٢٧) — من المحرّك
  // (`account_day`): قبل ساعة فتح الدوام ما زال أمس. و`today` يبقى حدَّ «لا صرفَ في الغد».
  const [spentAt, setSpentAt] = useState(accountDay || today || damascusDay());
  const [receipt, setReceipt] = useState("");
  // **«منين دفعت؟» — كاشٌ من الدرج افتراضاً** (قرارُ المالك ٢٠٢٦-١٠-١٥).
  const [paidFrom, setPaidFrom] = useState<PaidFrom>("cash");
  // **اسمُ الموظّف** — إلزاميٌّ في باب الرواتب (قرارُ المالك ٢٠٢٦-١٠-١٥).
  const [employee, setEmployee] = useState("");
  const [review, setReview] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  // **ما في درج المكتب الآن** — عرضٌ لا حساب، ومن لا يقرؤه لا يرى السطر.
  const [drawer, setDrawer] = useState<number | null>(null);

  useEffect(() => {
    api<{ position?: { expected?: number } }>("/api/v1/admin/cashbox?per_page=1")
      .then((r) => setDrawer(typeof r.position?.expected === "number" ? r.position.expected : null))
      .catch(() => setDrawer(null));
  }, []);

  const parsed = parseAmount(amount);
  const over = parsed.value !== null && parsed.value > threshold;
  // **فوق السقف: الموظّفُ ينتظر، والمالكُ أو المديرُ ينفُذ فوراً** (قرارُ ٢٠٢٦-١٠-٢٧).
  const instant = !over || autoApplies;
  const salary = cats.find((c) => c.id === categoryID)?.is_salary === true;
  const cap = fmtMoney(threshold);

  function next() {
    // **وكلُّ خطأٍ يُقال** — كان الحفظُ يسكت مع «١٠٠٠» أو «1,000» أو بلا باب.
    if (!categoryID) return setError(X.noCategory);
    if (parsed.value === null) return setError(parsed.error);
    if (salary && !employee.trim()) return setError(X.employeeMissing);
    if (spentAt > (today || damascusDay())) return setError(X.futureDate);
    if (over && !receipt) return setError(m.errors.expense_receipt_required);
    setError("");
    setReview(true);
  }

  async function submit() {
    if (busy || parsed.value === null) return;
    setBusy(true);
    setError("");
    try {
      const r = await api<{ id: string; status: string }>("/api/v1/admin/expenses", {
        method: "POST",
        body: JSON.stringify({
          category_id: categoryID,
          amount: parsed.value,
          note: note.trim(),
          spent_at: spentAt,
          receipt_media_id: receipt || undefined,
          paid_from: paidFrom,
          employee_name: employee.trim() || undefined,
        }),
      });
      onDone(r?.status ?? "posted");
    } catch (err) {
      setError(errorText(err));
      setReview(false);
    } finally {
      setBusy(false);
    }
  }

  const showReview = review && parsed.value !== null;
  const cat = cats.find((c) => c.id === categoryID);

  return (
    <Modal open onClose={onClose} title={showReview ? X.reviewTitle : X.add}>
      {showReview && parsed.value !== null && (
        <div className="space-y-3">
          <dl className="space-y-2 text-sm">
            <div className="flex justify-between gap-3">
              <dt className="text-ink-muted">{X.category}</dt>
              <dd className="font-medium">{cat?.name}</dd>
            </div>
            {employee.trim() && (
              <div className="flex justify-between gap-3">
                <dt className="text-ink-muted">{X.employeeName}</dt>
                <dd className="font-medium">{employee.trim()}</dd>
              </div>
            )}
            <div className="flex justify-between gap-3">
              <dt className="text-ink-muted">{X.reviewAmount}</dt>
              <dd dir="ltr" className="figure">
                {fmtMoney(parsed.value)}
              </dd>
            </div>
            <div className="flex justify-between gap-3">
              <dt className="text-ink-muted">{X.spentAt}</dt>
              <dd dir="ltr">{fmtDate(spentAt)}</dd>
            </div>
            <div className="flex justify-between gap-3">
              <dt className="text-ink-muted">{X.paidFrom}</dt>
              <dd className="font-medium">{X.paidFromOptions[paidFrom]}</dd>
            </div>
            {instant && (
              <div className="flex justify-between gap-3">
                <dt className="text-ink-muted">{X.reviewBalanceAfter}</dt>
                <dd dir="ltr" className={`font-bold ${balance - parsed.value < 0 ? "text-danger" : ""}`}>
                  {fmtMoney(balance - parsed.value)}
                </dd>
              </div>
            )}
            {instant && paidFrom === "cash" && drawer !== null && (
              <div className="flex justify-between gap-3">
                <dt className="text-ink-muted">{X.reviewDrawerAfter}</dt>
                <dd dir="ltr" className={`font-bold ${drawer - parsed.value < 0 ? "text-danger" : ""}`}>
                  {fmtMoney(drawer - parsed.value)}
                </dd>
              </div>
            )}
          </dl>
          <Alert tone={over && !autoApplies ? "warning" : "info"}>
            {over
              ? (autoApplies ? X.reviewInstant : X.reviewNeedsApproval).replace("{cap}", cap)
              : paidFrom === "cash"
                ? X.reviewDirectCash
                : X.reviewDirect}
          </Alert>
          {error && <Alert>{error}</Alert>}
          <FormActions
            onSave={() => void submit()}
            onCancel={() => setReview(false)}
            busy={busy}
            saveLabel={X.confirmSave}
            cancelLabel={X.back}
          />
        </div>
      )}
      <div className={showReview ? "hidden" : "space-y-3"}>
        <p className="text-sm text-ink-muted">{X.addHint.replace("{cap}", cap)}</p>
        {cats.length === 0 && <Alert>{X.noCategory}</Alert>}
        <Select
          id="exp-cat"
          label={X.category}
          required
          value={categoryID}
          onChange={(e) => setCategoryID(e.target.value)}
        >
          {cats.map((c) => (
            <option key={c.id} value={c.id}>
              {c.name}
            </option>
          ))}
        </Select>
        <Input
          id="exp-employee"
          label={salary ? X.employeeRequired : X.employeeOptional}
          required={salary}
          maxLength={80}
          value={employee}
          onChange={(e) => setEmployee(e.target.value)}
        />
        <Input
          id="exp-amount"
          label={`${X.amount} (${CUR})`}
          required
          inputMode="numeric"
          dir="ltr"
          value={amount}
          error={amount !== "" && parsed.value === null ? parsed.error : undefined}
          onChange={(e) => setAmount(e.target.value)}
        />
        <div>
          <Input
            id="exp-date"
            type="date"
            label={X.spentAt}
            max={today || damascusDay()}
            value={spentAt}
            onChange={(e) => setSpentAt(e.target.value)}
          />
          <p className="mt-1 text-2xs text-ink-muted">{X.spentAtHint}</p>
        </div>
        {/* **«منين دفعت؟» زرّان كبيران** — وتحت كلٍّ سطرٌ يقول ما يقع بالمال. */}
        <fieldset className="space-y-1.5">
          <legend className="mb-1 text-sm font-medium">{X.paidFrom}</legend>
          <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
            {(["cash", "outside"] as const).map((v) => (
              <button
                key={v}
                type="button"
                aria-pressed={paidFrom === v}
                onClick={() => setPaidFrom(v)}
                className={`rounded-control border px-3 py-2.5 text-start transition-colors ${
                  paidFrom === v ? "border-primary bg-primary-tint" : "border-line bg-surface hover:bg-row-hover"
                }`}
              >
                <span className="block text-sm font-bold">{X.paidFromOptions[v]}</span>
                <span className="block text-2xs text-ink-muted">
                  {v === "cash" ? X.paidFromCashHint : X.paidFromOutsideHint}
                </span>
              </button>
            ))}
          </div>
        </fieldset>
        <Textarea
          id="exp-note"
          label={X.note}
          rows={2}
          value={note}
          onChange={(e) => setNote(e.target.value)}
        />
        <ImageUpload
          kind="expense_receipt"
          label={over ? X.receiptRequired : X.receiptOptional}
          path="/api/v1/admin/expenses/receipt"
          onChange={setReceipt}
          api={api}
          mediaUrl={mediaUrl}
          errorText={errorText}
        />
        {error && !showReview && <Alert>{error}</Alert>}
        <FormActions onSave={next} onCancel={onClose} />
      </div>
    </Modal>
  );
}

/**
 * **أبوابُ المصروف — إضافةٌ وتعديلُ اسمٍ وترتيبٌ وتفعيلٌ وإطفاء، ومع كلّ بابٍ مبلغُه.**
 *
 * **ولا يُحذف بابٌ صُرف عليه** — يُطفأ، وحذفُه يمحو تبويبَ ما مضى.
 * **وكلُّ بابٍ كرتٌ صغير**: الاسمُ ومبلغُه وشاراتُه فوق، ومفتاحان تحته، وأسهمٌ بأيقونات —
 * كان سطراً بستّة أشياء يلتفّ على الجوّال فيتلخبط.
 */
function CatsModal({
  cats,
  onClose,
  onChanged,
}: {
  cats: Cat[];
  onClose: () => void;
  onChanged: () => void;
}) {
  const [name, setName] = useState("");
  const [editing, setEditing] = useState<{ id: string; name: string } | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  async function save(body: Record<string, unknown>) {
    if (busy) return;
    setBusy(true);
    setError("");
    try {
      await api("/api/v1/admin/expenses/categories", {
        method: "POST",
        body: JSON.stringify(body),
      });
      setName("");
      setEditing(null);
      onChanged();
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  /**
   * **الترتيبُ نداءٌ واحد** (قرارُ المالك ٢٠٢٦-١٠-١٥) — القائمةُ كاملةً بترتيبها الجديد.
   * كان نداءين يبدّلان مكانَي بابين، **فإن سقط الثاني بقي البابان على رقمٍ واحد.**
   */
  async function move(i: number, dir: -1 | 1) {
    const a = cats[i];
    const b = cats[i + dir];
    if (!a || !b || busy) return;
    const ids = cats.map((c) => c.id);
    ids[i] = b.id;
    ids[i + dir] = a.id;
    setBusy(true);
    setError("");
    try {
      await api("/api/v1/admin/expenses/categories/reorder", {
        method: "POST",
        body: JSON.stringify({ ids }),
      });
      onChanged();
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open onClose={onClose} title={X.manageCats}>
      <div className="space-y-3">
        <div className="flex gap-2">
          <div className="min-w-0 flex-1">
            <Input id="cat-name" label={X.catName} value={name} onChange={(e) => setName(e.target.value)} />
          </div>
          <Button
            className="self-end"
            disabled={busy || !name.trim()}
            onClick={() => void save({ name: name.trim() })}
          >
            {m.common.save}
          </Button>
        </div>
        <p className="text-xs text-ink-muted">{X.catRecurringHint}</p>
        {error && <Alert>{error}</Alert>}
        <ul className="space-y-2">
          {cats.map((c, i) => (
            <li
              key={c.id}
              className={`rounded-control border border-line-soft p-3 ${c.active ? "bg-surface" : "bg-field"}`}
            >
              <div className="flex items-start gap-2">
                {editing?.id === c.id ? (
                  <div className="flex min-w-0 flex-1 gap-2">
                    <div className="min-w-0 flex-1">
                      <Input
                        id={`cat-edit-${c.id}`}
                        label={X.catRename}
                        value={editing.name}
                        onChange={(e) => setEditing({ id: c.id, name: e.target.value })}
                      />
                    </div>
                    <Button
                      className="self-end"
                      disabled={busy || !editing.name.trim()}
                      onClick={() => void save({ id: c.id, name: editing.name.trim() })}
                    >
                      {m.common.save}
                    </Button>
                  </div>
                ) : (
                  <span className="flex min-w-0 flex-1 flex-col gap-1">
                    <span className="flex flex-wrap items-center gap-1.5">
                      <span className={`truncate font-medium ${c.active ? "" : "text-ink-muted"}`}>{c.name}</span>
                      {c.is_salary && <Badge variant="neutral">{X.catSalary}</Badge>}
                      {!c.active && <Badge variant="neutral">{X.catInactive}</Badge>}
                    </span>
                    {c.used > 0 && (
                      <span className="text-xs text-ink-muted">
                        {X.usedCount.replace("{n}", fmtNum(c.used))} ·{" "}
                        {X.catSpent.replace("{amount}", fmtMoney(c.spent))}
                      </span>
                    )}
                  </span>
                )}
                <span className="flex shrink-0 items-center">
                  {editing?.id !== c.id && (
                    <Button
                      variant="ghost"
                      className="!px-2"
                      aria-label={X.catRename}
                      onClick={() => setEditing({ id: c.id, name: c.name })}
                    >
                      <IconEdit size={15} />
                    </Button>
                  )}
                  <Button
                    variant="ghost"
                    className="!px-2"
                    aria-label={X.catUp}
                    disabled={busy || i === 0}
                    onClick={() => void move(i, -1)}
                  >
                    <IconArrowOut size={15} />
                  </Button>
                  <Button
                    variant="ghost"
                    className="!px-2"
                    aria-label={X.catDown}
                    disabled={busy || i === cats.length - 1}
                    onClick={() => void move(i, 1)}
                  >
                    <IconArrowIn size={15} />
                  </Button>
                </span>
              </div>
              <div className="mt-2 grid grid-cols-1 gap-2 border-t border-line-soft pt-2 sm:grid-cols-2">
                <Switch
                  id={`cat-${c.id}`}
                  label={X.catActive}
                  checked={c.active}
                  disabled={busy}
                  onChange={(next) => void save({ id: c.id, active: next })}
                />
                <Switch
                  id={`cat-rec-${c.id}`}
                  label={X.catRecurring}
                  checked={c.monthly_recurring === true}
                  disabled={busy}
                  onChange={(next) => void save({ id: c.id, monthly_recurring: next })}
                />
              </div>
            </li>
          ))}
        </ul>
      </div>
    </Modal>
  );
}
