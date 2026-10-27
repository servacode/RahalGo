"use client";

/**
 * الأرباح والخسائر — بابٌ لحاله في القائمة الجانبيّة (قرارات المالك ٢٠٢٦-١٠-٠٤).
 *
 * - **«اليوم» يومُ الحساب** — يبدأ بساعة فتح الدوام (`platform.day_start_hour`)، والخادمُ يردّ
 *   `today` و`day_start_hour` فتُبنى أزرارُ المدّة عليهما (تدقيقُ ٢٠٢٦-١٠-٢٧: كان «اليوم» يومَ التقويم
 *   فيُصفّر قبل التاسعة صباحاً).
 * - **المدّةُ والتبويبُ في الرابط** — يُقرآن عند الفتح ويُكتبان عند التغيير، فرابطُ «صافي اليوم»
 *   من الرئيسيّة يفتح على يومه لا على الشهر.
 * - كشف يتجمّع بالضبط، وتحته سطر التحقّق «مجموع البنود = الصافي».
 * - الربح المعلّق والحركات خارج الربح والديون المشطوبة أرقامٌ خارج الصافي.
 * - تبويبات الأشخاص: كلّ عمود نوع قيد واحد، والأعمدة يرسلها المحرّك، **والرقمُ الكبيرُ
 *   هو الأساسيُّ للتبويب** (`primary`) لا أوّلُ عمود.
 */

import { useCallback, useEffect, useRef, useState, type ReactNode } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale, fmtNum, errorText, damascusDay } from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  ButtonLink,
  Chips,
  Input,
  PageContainer,
  LoadingState,
  ReloadState,
  Pagination,
  Money,
  IconWallet,
  IconUser,
  IconStore,
  IconDriver,
  IconStatus,
  IconDate,
  IconTrendUp,
  IconTrendDown,
  IconOrder,
  IconCheck,
  IconHourglass,
  IconSwap,
  IconBalance,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useCanOpen } from "@/lib/policy";
import { PageHero, type HeroTab } from "@/components/admin/PageHero";
import { Panel, Tile, type TileTone } from "@/components/admin/DashParts";

const m = getMessages(defaultLocale);
const P = m.admin.profits;
const CUR = m.common.currency;
const HOUR = 3_600_000;
const DAY = 24 * HOUR;
/** **ساعةُ فتح الدوام قبل أوّل ردّ** — تقديرٌ يُصحَّح من الخادم (`day_start_hour`) فور وصوله. */
const GUESS_HOUR = 9;

type Tab = "platform" | "customers" | "reps" | "drivers" | "merchants";
const TABS: readonly Tab[] = ["platform", "customers", "reps", "drivers", "merchants"];
type Preset = "today" | "yesterday" | "week" | "month" | "lastMonth" | "all";
const PRESETS: readonly Preset[] = ["today", "yesterday", "week", "month", "lastMonth", "all"];
const PRESET_LABEL: Record<Preset, string> = {
  today: P.presetToday,
  yesterday: P.presetYesterday,
  week: P.presetWeek,
  month: P.presetMonth,
  lastMonth: P.presetLastMonth,
  all: P.presetAll,
};

interface Line {
  key: string;
  amount: number;
}
interface DayInfo {
  /** **يومُ الحساب الآن** بحسب الخادم — YYYY-MM-DD. */
  today?: string;
  day_start_hour?: number;
}
interface Platform extends DayInfo {
  basis: string;
  orders: number;
  sales: number;
  income: number;
  lines: Line[];
  lines_sum: number;
  net: number;
  check_ok: boolean;
  check_diff: number;
  pending: number;
  outside: number;
  written_off?: number;
}
interface Row {
  user_id: string;
  name: string;
  /** **يغيب لمن لا يملك `users.contact.read`** — الخادمُ يحذفه (`XG-42`). */
  phone?: string;
  values: number[];
}
interface Party extends DayInfo {
  columns: string[];
  rows: Row[];
  total: number;
  per_page: number;
  /** **العمودُ الأساسيّ** — رقمُ الكرت الكبير وما يُرتَّب عليه. */
  primary?: string;
}

/** اسم كلّ سطر في الكشف. */
const LINE_LABEL: Record<string, string> = {
  margin: P.lineMargin,
  commission: P.lineCommission,
  delivery_share: P.lineDeliveryShare,
  discount: P.lineDiscount,
  free_delivery: P.lineFreeDelivery,
  item_discounts: P.lineItemDiscounts,
  settle_diff: P.lineSettleDiff,
  rep_share: P.lineRepShare,
  lost_failed: P.lineLostFailed,
  lost_cancelled: P.lineLostCancelled,
  lost_refunded: P.lineLostRefunded,
  compensations: P.lineCompensations,
  // **ونقصُ الصندوق والتسويةُ اليدويّةُ إيداعاً سطورٌ لحالها** (٢٠٢٦-١٠-١٥).
  shortfalls: P.lineShortfalls,
  adjustment_credits: P.lineAdjustmentCredits,
  written_off: P.lineWrittenOff,
  recovered: P.lineRecovered,
  // **والمكافآتُ أربعةُ سطورٍ بمصدرها** (قرارُ المالك ٢٠٢٦-١٠-١٥).
  targets: P.lineTargets,
  manual_rewards: P.lineManualRewards,
  referrals: P.lineReferrals,
  signup_bonuses: P.lineSignupBonuses,
  opex: P.lineOpex,
  penalties: P.linePenalties,
};
/** سطور الدخل — والباقي «ناقص» أو «زائد» بإشارته. */
const INCOME_KEYS = new Set([
  "margin",
  "commission",
  "delivery_share",
  "discount",
  "free_delivery",
  "item_discounts",
  "settle_diff",
]);
/** سطور تظهر حتى لو صفر — لأنّها أبواب الكشف الأساسيّة. */
const ALWAYS = new Set(["margin", "commission", "delivery_share", "rep_share", "opex"]);

/**
 * **الخصوماتُ مجموعاتٌ بعناوين** (تدقيقُ ٢٠٢٦-١٠-٢٧) — كانت قائمةً واحدةً طويلةً مخلوطة.
 * وسطرٌ جديدٌ لا تعرفه الخريطةُ يقع في «مصاريف» فلا يغيب.
 */
const GROUPS: readonly { key: string; label: string; lines: readonly string[] }[] = [
  {
    key: "people",
    label: P.groupPeople,
    lines: ["rep_share", "targets", "manual_rewards", "referrals", "signup_bonuses"],
  },
  {
    key: "losses",
    label: P.groupLosses,
    lines: ["lost_failed", "lost_cancelled", "lost_refunded", "compensations", "shortfalls", "adjustment_credits"],
  },
  { key: "costs", label: P.groupCosts, lines: ["opex"] },
  { key: "back", label: P.groupBack, lines: ["recovered", "penalties"] },
];

/** **كلُّ سطرٍ يفتح صفحتَه** — كلُّها بقدرة `finance.read` نفسِها التي تفتح هذه الصفحة. */
function lineHref(key: string, from: string, to: string): string | null {
  const range = new URLSearchParams();
  if (from) range.set("from", from);
  if (to) range.set("to", to);
  const qs = range.toString();
  switch (key) {
    case "compensations":
      return `/dashboard/compensations${qs ? `?${qs}` : ""}`;
    case "lost_failed":
    case "lost_cancelled":
    case "lost_refunded":
    case "shortfalls":
    case "adjustment_credits":
      return "/dashboard/tickets?tab=losses";
    case "opex":
      return "/dashboard/expenses";
    case "targets":
    case "manual_rewards":
      return "/dashboard/incentives";
    default:
      return null;
  }
}

/** اسم كلّ عمود في تبويبات الأشخاص. */
const COL_LABEL: Record<string, string> = {
  referral: P.colReferral,
  bonus: P.colBonus,
  target_reward: P.colTargetReward,
  manual_reward: P.colManualReward,
  signup: P.colSignup,
  compensation: P.colCompensation,
  penalty: P.colPenalty,
  spent: P.colSpent,
  commission: P.colCommission,
  delivery: P.colDelivery,
  our_margin: P.colOurMargin,
  our_commission: P.colOurCommission,
  gross: P.colSalesItems,
  store_earned: P.colStoreEarnedLedger,
};
/** **الرقمُ الأساسيُّ لكلّ تبويب** — إن لم يقله الخادم. */
const PRIMARY: Record<Exclude<Tab, "platform">, string> = {
  customers: "spent",
  reps: "commission",
  drivers: "delivery",
  merchants: "gross",
};

/** خليّة CSV آمنة — لا معادلة تبدأ بها. */
const DQ = String.fromCharCode(34);
const NEEDS_QUOTE = new RegExp("[" + DQ + ",\\n]");
const FORMULA = new RegExp("^[=+\\-@]");
function cell(v: string | number): string {
  let s = String(v);
  if (typeof v === "string" && FORMULA.test(s)) s = "'" + s;
  return NEEDS_QUOTE.test(s) ? DQ + s.split(DQ).join(DQ + DQ) + DQ : s;
}

function saveCsv(name: string, rows: (string | number)[][]) {
  const text = "﻿" + rows.map((r) => r.map(cell).join(",")).join("\n");
  const url = URL.createObjectURL(new Blob([text], { type: "text/csv;charset=utf-8" }));
  const a = document.createElement("a");
  a.href = url;
  a.download = name;
  a.click();
  URL.revokeObjectURL(url);
}

/** **يومٌ بعد يومٍ بالتقويم** — على نصّ YYYY-MM-DD، بلا منطقةٍ زمنيّة. */
function addDays(day: string, n: number): string {
  return new Date(Date.parse(`${day}T12:00:00Z`) + n * DAY).toISOString().slice(0, 10);
}

/** **مدّةُ زرٍّ سريع** — كلُّها من يوم الحساب لا من يوم التقويم. */
function presetRange(p: Preset, today: string): { from: string; to: string } {
  switch (p) {
    case "today":
      return { from: today, to: today };
    case "yesterday": {
      const y = addDays(today, -1);
      return { from: y, to: y };
    }
    case "week":
      return { from: addDays(today, -6), to: today };
    case "month":
      return { from: `${today.slice(0, 8)}01`, to: today };
    case "lastMonth": {
      const end = addDays(`${today.slice(0, 8)}01`, -1);
      return { from: `${end.slice(0, 8)}01`, to: end };
    }
    default:
      return { from: "", to: "" };
  }
}

/** **ما يقوله الرابط عند الفتح** — `?range=` لزرٍّ سريع، أو `?from=&to=` لمدّةٍ بعينها. */
function readUrl(): { tab: Tab; preset: Preset | ""; from: string; to: string } {
  const q = new URLSearchParams(typeof window === "undefined" ? "" : window.location.search);
  const t = q.get("tab") as Tab | null;
  const tab: Tab = t && TABS.includes(t) ? t : "platform";
  const r = q.get("range") as Preset | null;
  if (r && PRESETS.includes(r)) return { tab, preset: r, from: "", to: "" };
  // **ونظرةُ الخزينة تقول «من أول يوم» بـ `period=all`.**
  if (q.get("period") === "all") return { tab, preset: "all", from: "", to: "" };
  const from = q.get("from") ?? "";
  const to = q.get("to") ?? "";
  if (from || to) return { tab, preset: "", from, to };
  return { tab, preset: "month", from: "", to: "" };
}

function writeUrl(tab: Tab, preset: Preset | "", from: string, to: string) {
  const q = new URLSearchParams();
  if (tab !== "platform") q.set("tab", tab);
  if (preset) {
    if (preset !== "month") q.set("range", preset);
  } else {
    if (from) q.set("from", from);
    if (to) q.set("to", to);
  }
  const s = q.toString();
  window.history.replaceState(null, "", s ? `?${s}` : window.location.pathname);
}

/**
 * **مبلغٌ بإشارته نصّاً** — للبطاقات التي تأخذ نصّاً. **الإشارةُ والرقمُ في عزلٍ يساريّ واحد**
 * (LRI…PDI) والعملةُ بعده، فيُقرأ «−1,000 ل.س» لا «1,000− ل.س» (تدقيقُ ٢٠٢٦-١٠-٢٧).
 */
function signedText(v: number, plus = true): string {
  const sign = v < 0 ? "−" : v > 0 && plus ? "+" : "";
  return `⁦${sign}${fmtNum(Math.abs(v))}⁩ ${CUR}`;
}

/** **نبرةُ الرقم بإشارته** — موجبٌ أخضر، سالبٌ أحمر، وصفرٌ رماديّ. */
function toneOf(v: number): TileTone {
  return v > 0 ? "success" : v < 0 ? "danger" : "muted";
}

export default function ProfitsPage() {
  const { can } = useAuth();
  const canOpen = useCanOpen();
  const router = useRouter();
  const [ready, setReady] = useState(false);
  const [tab, setTab] = useState<Tab>("platform");
  /** **زرٌّ سريعٌ مختار** — مدّتُه تُحسب من يوم الحساب، فتتبعه إن صحّحه الخادم. */
  const [preset, setPreset] = useState<Preset | "">("month");
  const [customFrom, setCustomFrom] = useState("");
  const [customTo, setCustomTo] = useState("");
  /** **يومُ الحساب وساعةُ بدئه من الخادم** — وقبل أوّل ردٍّ تقديرٌ بالساعة التاسعة. */
  const [serverToday, setServerToday] = useState<string | null>(null);
  const [dayHour, setDayHour] = useState<number | null>(null);
  const [page, setPage] = useState(1);
  const [data, setData] = useState<unknown | null>(null);
  const [loading, setLoading] = useState(true);
  const [failed, setFailed] = useState("");
  const [notice, setNotice] = useState("");
  const [exporting, setExporting] = useState(false);
  /** الطلب السابق يُلغى إن تغيّر التاريخ بسرعة — فلا تصل نتيجة قديمة بعد الجديدة. */
  const ctl = useRef<AbortController | null>(null);

  const today = serverToday ?? damascusDay(Date.now() - (dayHour ?? GUESS_HOUR) * HOUR);
  const range = preset ? presetRange(preset, today) : { from: customFrom, to: customTo };
  const { from, to } = range;
  const inverted = from !== "" && to !== "" && from > to;

  useEffect(() => {
    const u = readUrl();
    setTab(u.tab);
    setPreset(u.preset);
    setCustomFrom(u.from);
    setCustomTo(u.to);
    setReady(true);
  }, []);

  useEffect(() => {
    if (ready) writeUrl(tab, preset, customFrom, customTo);
  }, [ready, tab, preset, customFrom, customTo]);

  const load = useCallback(() => {
    if (!ready || inverted) return;
    ctl.current?.abort();
    const c = new AbortController();
    ctl.current = c;
    const qs = new URLSearchParams({ tab, page: String(page) });
    if (from) qs.set("from", from);
    if (to) qs.set("to", to);
    setLoading(true);
    api<DayInfo>(`/api/v1/admin/profits?${qs}`, { signal: c.signal })
      .then((d) => {
        if (c.signal.aborted) return;
        setFailed("");
        setData(d);
        if (typeof d.today === "string" && d.today) setServerToday(d.today);
        if (typeof d.day_start_hour === "number") setDayHour(d.day_start_hour);
      })
      .catch((e) => {
        if (c.signal.aborted) return;
        setFailed(errorText(e, m) || " ");
      })
      .finally(() => {
        if (!c.signal.aborted) setLoading(false);
      });
  }, [ready, tab, page, from, to, inverted]);

  useEffect(() => {
    load();
    return () => ctl.current?.abort();
  }, [load]);

  const tabs: HeroTab<Tab>[] = [
    { key: "platform", label: P.tabPlatform, icon: IconWallet },
    { key: "customers", label: P.tabCustomers, icon: IconUser },
    { key: "reps", label: P.tabReps, icon: IconStatus },
    { key: "drivers", label: P.tabDrivers, icon: IconDriver },
    { key: "merchants", label: P.tabMerchants, icon: IconStore },
  ];

  /** التصدير بالمدّة والتبويب المعروضين — كلّ الصفوف لا الصفحة وحدها. */
  async function exportCsv() {
    const span = `${from || "all"}_${to || today}`;
    setExporting(true);
    setNotice("");
    try {
      if (tab === "platform") {
        const d = data as Platform;
        const rows: (string | number)[][] = [[P.exportLine, `${P.exportAmount} (${CUR})`]];
        for (const l of d.lines) rows.push([LINE_LABEL[l.key] ?? l.key, l.amount]);
        rows.push([P.lineNet, d.net], [P.pending, d.pending], [P.outside, d.outside]);
        if (typeof d.written_off === "number") rows.push([P.writtenOff, d.written_off]);
        saveCsv(`profits-${span}.csv`, rows);
      } else {
        const withPhone = can("users.contact.read");
        const all: Row[] = [];
        let cols: string[] = [];
        for (let p = 1; p < 1000; p++) {
          const qs = new URLSearchParams({ tab, page: String(p), per_page: "100" });
          if (from) qs.set("from", from);
          if (to) qs.set("to", to);
          const d = await api<Party>(`/api/v1/admin/profits?${qs}`);
          cols = d.columns;
          all.push(...d.rows);
          if (all.length >= d.total || d.rows.length === 0) break;
        }
        const head = [P.exportName, ...(withPhone ? [P.exportPhone] : [])];
        head.push(...cols.map((c) => `${COL_LABEL[c] ?? c} (${CUR})`));
        const rows: (string | number)[][] = [head];
        for (const r of all) {
          rows.push([r.name || r.phone || "", ...(withPhone ? [r.phone ?? ""] : []), ...r.values]);
        }
        saveCsv(`profits-${tab}-${span}.csv`, rows);
      }
      setNotice(P.exported);
    } catch (e) {
      setFailed(errorText(e, m) || " ");
    } finally {
      setExporting(false);
    }
  }

  const activePreset: Preset | "" =
    preset ||
    (PRESETS.find((p) => {
      const r = presetRange(p, today);
      return r.from === from && r.to === to;
    }) ??
      "");

  /** **التاريخُ المكتوبُ باليد يبدأ من المدّة المعروضة** — لا من فراغ. */
  function setDates(nextFrom: string, nextTo: string) {
    setPreset("");
    setCustomFrom(nextFrom);
    setCustomTo(nextTo);
    setPage(1);
  }

  // **تصميمٌ احترافيّ (المالك ٢٠٢٦-١٠-٢٦)** — رأسٌ موحّدٌ بتبويبات الأطراف، المدّةُ في كرتٍ واحد،
  // والكشفُ أقسامٌ واضحة (دخل · خصومات بمجموعات · صافي) بدل قائمةٍ طويلة، والأشخاصُ كروتٌ لا جدول.
  return (
    <PageContainer>
      <PageHero<Tab>
        icon={IconWallet}
        title={P.titlePl}
        subtitle={P.hintPl}
        actions={
          <Button
            variant="secondary"
            onClick={() => void exportCsv()}
            disabled={inverted || loading || exporting || data === null}
          >
            {P.export}
          </Button>
        }
        tabs={tabs}
        value={tab}
        onChange={(k) => {
          setTab(k);
          setPage(1);
          setData(null);
        }}
      />

      {notice && (
        <Alert tone="success" className="mb-4" onDismiss={() => setNotice("")}>
          {notice}
        </Alert>
      )}

      {/* ── المدّة في كرتٍ واحد ── */}
      <section className="surface mb-4 space-y-3 p-4">
        <Chips<Preset | "">
          items={PRESETS.map((id) => ({ id, label: PRESET_LABEL[id] }))}
          value={activePreset}
          onChange={(id) => {
            if (!id) return;
            setPreset(id);
            setPage(1);
          }}
          wrap
        />
        <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
          <Input
            id="pf-from"
            type="date"
            icon={<IconDate />}
            label={m.shared.statement.from}
            value={from}
            onChange={(e) => setDates(e.target.value, to)}
          />
          <Input
            id="pf-to"
            type="date"
            icon={<IconDate />}
            label={m.shared.statement.to}
            value={to}
            onChange={(e) => setDates(from, e.target.value)}
          />
        </div>
        <p className="text-2xs text-ink-muted">{P.basis.replace("{h}", fmtNum(dayHour ?? GUESS_HOUR))}</p>
      </section>

      {inverted ? (
        <Alert className="mb-4">{P.inverted}</Alert>
      ) : failed ? (
        <ReloadState onRetry={load} label={failed.trim() || undefined} />
      ) : data === null ? (
        <LoadingState />
      ) : (
        <div aria-busy={loading} className={loading ? "opacity-60" : undefined}>
          {loading && <p className="mb-2 text-2xs text-ink-muted">{P.refreshing}</p>}
          {tab === "platform" ? (
            <PlatformView
              d={data as Platform}
              from={from}
              to={to}
              onOrders={
                can("orders.read")
                  ? () => {
                      const q = new URLSearchParams({ status: "delivered" });
                      if (from) q.set("from", from);
                      if (to) q.set("to", to);
                      router.push(`/dashboard/history?${q}`);
                    }
                  : undefined
              }
            />
          ) : (
            <PartyView
              d={data as Party}
              tab={tab}
              page={page}
              onPage={setPage}
              showPhone={can("users.contact.read")}
              canOpenUser={canOpen.user}
              canOpenStore={canOpen.merchant}
            />
          )}
        </div>
      )}
    </PageContainer>
  );
}

/** مبلغ بإشارته وعملته — الأخضر يزيد الربح والأحمر ينقصه. **الإشارةُ والرقمُ في لفّةٍ واحدة.** */
function Signed({ v, strong }: { v: number; strong?: boolean }) {
  return (
    <span
      className={`whitespace-nowrap ${strong ? "font-bold" : ""} ${v < 0 ? "text-danger" : v > 0 ? "text-success" : "text-ink-muted"}`}
    >
      <span dir="ltr" className="tabular-nums">
        {v < 0 ? "−" : v > 0 ? "+" : ""}
        {fmtNum(Math.abs(v))}
      </span>{" "}
      <span>{CUR}</span>
    </span>
  );
}

/** سطرٌ في الكشف — نقطةٌ بلون أثره، واسمُه (رابطاً لصفحته إن كانت له)، ومبلغُه بإشارته. */
function Line({ label, v, href }: { label: string; v: number; href?: string | null }) {
  return (
    <li className="flex items-center justify-between gap-3 py-2">
      <span className="flex min-w-0 items-center gap-2 text-sm">
        <span
          aria-hidden
          className={`h-2 w-2 shrink-0 rounded-full ${v < 0 ? "bg-danger" : v > 0 ? "bg-success" : "bg-line"}`}
        />
        {href ? (
          <Link href={href} className="min-w-0 text-accent-text underline-offset-4 hover:underline">
            {label}
          </Link>
        ) : (
          <span className="min-w-0">{label}</span>
        )}
      </span>
      <Signed v={v} />
    </li>
  );
}

/** سطرُ المجموع تحت كلّ قسم. */
function Subtotal({ label, v }: { label: string; v: number }) {
  return (
    <div className="mt-2 flex items-center justify-between gap-3 rounded-control bg-field px-3 py-2.5">
      <span className="text-sm font-bold">{label}</span>
      <Signed v={v} strong />
    </div>
  );
}

/** **رقمٌ خارج الصافي** — دائرةٌ ملوّنةٌ واسمٌ ورقمٌ وجملةُ شرح. */
function Aside({
  icon: Icon,
  tone,
  label,
  hint,
  children,
}: {
  icon: typeof IconHourglass;
  tone: string;
  label: string;
  hint: string;
  children: ReactNode;
}) {
  return (
    <div className="flex items-start gap-3 rounded-control bg-field p-3">
      <span className={`inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full ${tone}`}>
        <Icon size={18} />
      </span>
      <div className="min-w-0 flex-1">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <span className="text-sm font-medium">{label}</span>
          {children}
        </div>
        <p className="text-2xs text-ink-muted">{hint}</p>
      </div>
    </div>
  );
}

function PlatformView({
  d,
  from,
  to,
  onOrders,
}: {
  d: Platform;
  from: string;
  to: string;
  onOrders?: () => void;
}) {
  const shown = d.lines.filter((l) => l.amount !== 0 || ALWAYS.has(l.key));
  const income = shown.filter((l) => INCOME_KEYS.has(l.key));
  const rest = shown.filter((l) => !INCOME_KEYS.has(l.key));
  const known = new Set(GROUPS.flatMap((g) => g.lines));
  const groups = GROUPS.map((g) => ({
    ...g,
    items: rest.filter((l) => g.lines.includes(l.key) || (g.key === "costs" && !known.has(l.key))),
  })).filter((g) => g.items.length > 0);
  const out = d.net - d.income;
  const writtenOff = d.written_off ?? 0;
  const tone = d.net > 0 ? "success" : d.net < 0 ? "danger" : "muted";
  const scrollTo = (id: string) => () => document.getElementById(id)?.scrollIntoView({ behavior: "smooth" });

  // **مدّةٌ بلا حركة** (تدقيقُ ٢٠٢٦-١٠-٢٧) — كان يُرسم «ربح 0» أخضرَ وتحته واحدٌ وعشرون صفراً.
  const quiet =
    d.orders === 0 &&
    d.lines.every((l) => l.amount === 0) &&
    d.net === 0 &&
    d.pending === 0 &&
    d.outside === 0 &&
    writtenOff === 0;
  if (quiet) {
    return (
      <section className="surface flex flex-col items-center gap-3 p-8 text-center">
        <span className="inline-flex h-16 w-16 items-center justify-center rounded-full bg-accent text-on-bright">
          <IconWallet size={28} />
        </span>
        <h2 className="heading-section">{P.emptyPl}</h2>
        <p className="max-w-md text-sm text-ink-muted">{P.emptyPlHint}</p>
      </section>
    );
  }

  return (
    <div className="space-y-4">
      {/* ── الصافي بكرتٍ كبير، وتحته فحصُ «مجموع البنود = الصافي» ── */}
      <section
        className={`surface flex flex-wrap items-center gap-4 border-2 p-5 ${
          tone === "success" ? "border-success" : tone === "danger" ? "border-danger" : "border-line-soft"
        }`}
      >
        <span
          className={`inline-flex h-16 w-16 shrink-0 items-center justify-center rounded-full ${
            tone === "success"
              ? "bg-success-solid text-on-solid"
              : tone === "danger"
                ? "bg-danger-solid text-on-solid"
                : "bg-field text-ink-muted"
          }`}
        >
          {d.net < 0 ? <IconTrendDown size={30} /> : d.net > 0 ? <IconTrendUp size={30} /> : <IconBalance size={30} />}
        </span>
        <div className="min-w-0 flex-1">
          <p className="flex flex-wrap items-center gap-2 text-sm text-ink-muted">
            {P.lineNet}
            {tone === "muted" ? (
              <Badge variant="neutral">{P.netZero}</Badge>
            ) : (
              <Badge variant={tone}>{tone === "success" ? P.netProfit : P.netLoss}</Badge>
            )}
          </p>
          <p
            className={`figure ${tone === "success" ? "text-success" : tone === "danger" ? "text-danger" : "text-ink-muted"}`}
          >
            <span dir="ltr" className="tabular-nums">
              {d.net < 0 ? "−" : ""}
              {fmtNum(Math.abs(d.net))}
            </span>{" "}
            <span>{CUR}</span>
          </p>
          <p className="text-xs text-ink-muted">{P.netHint}</p>
        </div>
        <div className="w-full sm:w-auto">
          {d.check_ok ? (
            <span className="inline-flex items-center gap-1.5 rounded-full bg-field px-3 py-1.5 text-sm font-medium text-success">
              <IconCheck size={16} />
              {P.checkOk}
            </span>
          ) : (
            <Alert>{P.checkBad.replace("{d}", signedText(d.check_diff))}</Alert>
          )}
        </div>
      </section>

      {/* ── أربعُ بطاقاتٍ تُضغط: الطلبات تفتح سجلّها، والباقي ينزل لقسمه ── */}
      <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 xl:grid-cols-4">
        <Tile
          icon={IconTrendUp}
          label={P.cardIncome}
          value={signedText(d.income, false)}
          tone={toneOf(d.income)}
          sub={P.sectionTap}
          onClick={scrollTo("pf-income")}
        />
        <Tile
          icon={IconTrendDown}
          label={P.cardOut}
          value={signedText(out, false)}
          tone={toneOf(out)}
          sub={P.sectionTap}
          onClick={scrollTo("pf-out")}
        />
        <Tile
          icon={IconOrder}
          label={P.cardOrders}
          value={fmtNum(d.orders)}
          tone={d.orders > 0 ? "accent" : "muted"}
          sub={`${P.cardSales}: ${signedText(d.sales, false)}`}
          onClick={onOrders}
        />
        <Tile
          icon={IconHourglass}
          label={P.pending}
          value={signedText(d.pending, false)}
          tone={d.pending !== 0 ? "warning" : "muted"}
          sub={P.sectionTap}
          onClick={scrollTo("pf-outside")}
        />
      </div>

      {/* ── الكشف قسمين: دخل ثمّ خصومات بمجموعاتٍ بعناوين ── */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <div id="pf-income" className="scroll-mt-4">
          <Panel title={P.cardIncome} note={P.incomeNote}>
            <ul className="divide-y divide-line-soft">
              {income.map((l) => (
                <Line key={l.key} label={LINE_LABEL[l.key] ?? l.key} v={l.amount} />
              ))}
            </ul>
            <Subtotal label={P.subtotal} v={d.income} />
          </Panel>
        </div>
        <div id="pf-out" className="scroll-mt-4">
          <Panel title={P.cardOut} note={P.costsNote}>
            <div className="space-y-3">
              {groups.map((g) => (
                <div key={g.key}>
                  <p className="text-xs font-bold text-ink-muted">{g.label}</p>
                  <ul className="divide-y divide-line-soft">
                    {g.items.map((l) => (
                      <Line
                        key={l.key}
                        label={LINE_LABEL[l.key] ?? l.key}
                        v={l.amount}
                        href={lineHref(l.key, from, to)}
                      />
                    ))}
                  </ul>
                </div>
              ))}
            </div>
            <Subtotal label={P.subtotal} v={out} />
          </Panel>
        </div>
      </div>

      {/* ── المعادلة: ثلاثةُ أسطرٍ تحت بعض، فلا تتبعثر على الجوّال ── */}
      <section className="surface space-y-1 p-4 text-sm">
        <div className="flex items-center justify-between gap-3">
          <span className="text-ink-muted">{P.cardIncome}</span>
          <Signed v={d.income} />
        </div>
        <div className="flex items-center justify-between gap-3">
          <span className="text-ink-muted">{P.cardOut}</span>
          <Signed v={out} />
        </div>
        <div className="flex items-center justify-between gap-3 border-t border-line-soft pt-2">
          <span className="font-bold">{P.lineNet}</span>
          <Signed v={d.net} strong />
        </div>
      </section>

      {/* ── أرقامٌ برّا الصافي ── */}
      <div id="pf-outside" className="scroll-mt-4">
        <Panel title={P.outsideTitle}>
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
            <Aside icon={IconHourglass} tone="bg-warning-solid text-on-solid" label={P.pending} hint={P.pendingHint}>
              <Money value={d.pending} className="font-bold" />
            </Aside>
            <Aside icon={IconSwap} tone="bg-surface text-ink" label={P.outside} hint={P.outsideHint}>
              <Signed v={d.outside} />
            </Aside>
            <Aside icon={IconBalance} tone="bg-danger-solid text-on-solid" label={P.writtenOff} hint={P.writtenOffHint}>
              <span className="flex items-center gap-2">
                <Money value={writtenOff} className="font-bold" />
                {writtenOff !== 0 && (
                  <Link
                    href="/dashboard/obligations"
                    className="text-2xs text-accent-text underline-offset-4 hover:underline"
                  >
                    {P.details}
                  </Link>
                )}
              </span>
            </Aside>
          </div>
        </Panel>
      </div>
    </div>
  );
}

const PARTY_ICON: Record<Exclude<Tab, "platform">, typeof IconUser> = {
  customers: IconUser,
  reps: IconStatus,
  drivers: IconDriver,
  merchants: IconStore,
};

/**
 * **كرتٌ لكلّ شخص** — الرقمُ الأساسيُّ للتبويب كبير، والباقي خاناتٌ صغيرةٌ **لما ليس صفراً**،
 * وزرٌّ بعرض الكرت يفتح حسابَه.
 */
function PartyView({
  d,
  tab,
  page,
  onPage,
  showPhone,
  canOpenUser,
  canOpenStore,
}: {
  d: Party;
  tab: Exclude<Tab, "platform">;
  page: number;
  onPage: (p: number) => void;
  showPhone: boolean;
  canOpenUser: boolean;
  canOpenStore: boolean;
}) {
  const rows = d.rows ?? [];
  const cols = d.columns ?? [];
  const Icon = PARTY_ICON[tab];
  const store = tab === "merchants";
  const canOpen = store ? canOpenStore : canOpenUser;
  const primaryKey = d.primary ?? PRIMARY[tab];
  const pi = Math.max(0, cols.indexOf(primaryKey));
  if (rows.length === 0) {
    return (
      <section className="surface flex flex-col items-center gap-3 p-8 text-center">
        <span className="inline-flex h-16 w-16 items-center justify-center rounded-full bg-accent text-on-bright">
          <Icon size={28} />
        </span>
        <h2 className="heading-section">{P.empty}</h2>
        <p className="max-w-md text-sm text-ink-muted">{P.emptyHint}</p>
      </section>
    );
  }
  return (
    <>
      <ul className="grid grid-cols-1 gap-3 lg:grid-cols-2 2xl:grid-cols-3">
        {rows.map((x) => {
          const title = x.name || x.phone || "";
          const phone = showPhone && x.phone && x.phone !== title ? x.phone : "";
          const main = x.values[pi] ?? 0;
          const others = cols
            .map((c, j) => ({ c, v: x.values[j] ?? 0, j }))
            .filter((o) => o.j !== pi && o.v !== 0);
          return (
            <li key={x.user_id} className="surface flex flex-col gap-3 p-4">
              <div className="flex items-center gap-3">
                <span className="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-accent text-on-bright">
                  <Icon size={20} />
                </span>
                <span className="min-w-0 flex-1">
                  <span className="block truncate font-bold">{title}</span>
                  {phone && (
                    <span dir="ltr" className="block truncate text-end text-2xs text-ink-muted">
                      {phone}
                    </span>
                  )}
                </span>
              </div>
              {cols.length > 0 && (
                <div className="rounded-control bg-field px-3 py-2.5">
                  <span className="block text-2xs text-ink-muted">{COL_LABEL[cols[pi]!] ?? cols[pi]}</span>
                  <span className={`figure block ${main === 0 ? "text-ink-muted" : ""}`}>
                    <span dir="ltr" className="tabular-nums">
                      {main < 0 ? "−" : ""}
                      {fmtNum(Math.abs(main))}
                    </span>{" "}
                    <span>{CUR}</span>
                  </span>
                </div>
              )}
              {others.length > 0 ? (
                <dl className="grid grid-cols-2 gap-2">
                  {others.map((o) => (
                    <div key={o.c} className="rounded-control border border-line-soft px-3 py-2">
                      <dt className="line-clamp-2 text-2xs text-ink-muted">{COL_LABEL[o.c] ?? o.c}</dt>
                      <dd>
                        <Money value={o.v} small />
                      </dd>
                    </div>
                  ))}
                </dl>
              ) : (
                main === 0 && <p className="text-2xs text-ink-muted">{P.allZero}</p>
              )}
              {canOpen && (
                <ButtonLink
                  variant="secondary"
                  className="mt-auto w-full"
                  href={store ? `/dashboard/merchants/${x.user_id}` : `/dashboard/users/${x.user_id}`}
                >
                  {store ? P.openStore : P.openAccount}
                </ButtonLink>
              )}
            </li>
          );
        })}
      </ul>
      {d.total > d.per_page && (
        <div className="mt-4 flex justify-center">
          <Pagination page={page} total={d.total} perPage={d.per_page} onChange={onPage} />
        </div>
      )}
    </>
  );
}
