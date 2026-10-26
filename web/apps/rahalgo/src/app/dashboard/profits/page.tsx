"use client";

/**
 * الأرباح والخسائر — تبويب في قسم الخزينة (قرارات المالك ٢٠٢٦-١٠-٠٤).
 *
 * - الأرقام حسب يوم القيد في دفتر الخزينة بتوقيت دمشق، والشاشة تقول ذلك.
 * - كشف يتجمّع بالضبط، وتحته سطر التحقّق «مجموع البنود = الصافي».
 * - الربح المعلّق والحركات خارج الربح (سحب وإيداع يدويّ) رقمان خارج الصافي.
 * - تبويبات الأشخاص: كلّ عمود نوع قيد واحد، والأعمدة يرسلها المحرّك.
 */

import { useCallback, useEffect, useRef, useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtMoney, errorText, damascusDay } from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
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
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { PageHero, type HeroTab } from "@/components/admin/PageHero";
import { Panel, Tile } from "@/components/admin/DashParts";

const m = getMessages(defaultLocale);
const P = m.admin.profits;
const CUR = m.common.currency;
const DAY = 86_400_000;

type Tab = "platform" | "customers" | "reps" | "drivers" | "merchants";

interface Line {
  key: string;
  amount: number;
}
interface Platform {
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
}
interface Row {
  user_id: string;
  name: string;
  phone: string;
  values: number[];
}
interface Party {
  columns: string[];
  rows: Row[];
  total: number;
  per_page: number;
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
  // **ونقصُ الصندوق والتسويةُ اليدويّةُ إيداعاً والمشطوبُ سطورٌ لحالها** (٢٠٢٦-١٠-١٥).
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

export default function ProfitsPage() {
  const { can } = useAuth();
  const [tab, setTab] = useState<Tab>("platform");
  const today = damascusDay();
  const [from, setFrom] = useState(`${today.slice(0, 8)}01`);
  const [to, setTo] = useState(today);
  const [page, setPage] = useState(1);
  const [data, setData] = useState<unknown | null>(null);
  const [loading, setLoading] = useState(true);
  const [failed, setFailed] = useState("");
  /** الطلب السابق يُلغى إن تغيّر التاريخ بسرعة — فلا تصل نتيجة قديمة بعد الجديدة. */
  const ctl = useRef<AbortController | null>(null);

  const inverted = from !== "" && to !== "" && from > to;

  const load = useCallback(() => {
    if (inverted) return;
    ctl.current?.abort();
    const c = new AbortController();
    ctl.current = c;
    const qs = new URLSearchParams({ tab, page: String(page) });
    if (from) qs.set("from", from);
    if (to) qs.set("to", to);
    setLoading(true);
    api<unknown>(`/api/v1/admin/profits?${qs}`, { signal: c.signal })
      .then((d) => {
        if (c.signal.aborted) return;
        setFailed("");
        setData(d);
      })
      .catch((e) => {
        if (c.signal.aborted) return;
        setFailed(errorText(e, m) || " ");
      })
      .finally(() => {
        if (!c.signal.aborted) setLoading(false);
      });
  }, [tab, page, from, to, inverted]);

  useEffect(() => {
    load();
    return () => ctl.current?.abort();
  }, [load]);

  const presets = [
    { id: "today", label: P.presetToday, from: today, to: today },
    { id: "week", label: P.presetWeek, from: damascusDay(Date.now() - 6 * DAY), to: today },
    { id: "month", label: P.presetMonth, from: `${today.slice(0, 8)}01`, to: today },
    { id: "all", label: P.presetAll, from: "", to: "" },
  ];

  const tabs: HeroTab<Tab>[] = [
    { key: "platform", label: P.tabPlatform, icon: IconWallet },
    { key: "customers", label: P.tabCustomers, icon: IconUser },
    { key: "reps", label: P.tabReps, icon: IconStatus },
    { key: "drivers", label: P.tabDrivers, icon: IconDriver },
    { key: "merchants", label: P.tabMerchants, icon: IconStore },
  ];

  /** التصدير بالمدّة والتبويب المعروضين — كلّ الصفوف لا الصفحة وحدها. */
  async function exportCsv() {
    const range = `${from || "start"}_${to || today}`;
    try {
      if (tab === "platform") {
        const d = data as Platform;
        const rows: (string | number)[][] = [[P.exportLine, `${P.exportAmount} (${CUR})`]];
        for (const l of d.lines) rows.push([LINE_LABEL[l.key] ?? l.key, l.amount]);
        rows.push([P.lineNet, d.net], [P.pending, d.pending], [P.outside, d.outside]);
        saveCsv(`profits-${range}.csv`, rows);
        return;
      }
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
        rows.push([r.name || r.phone, ...(withPhone ? [r.phone] : []), ...r.values]);
      }
      saveCsv(`profits-${tab}-${range}.csv`, rows);
    } catch (e) {
      setFailed(errorText(e, m) || " ");
    }
  }

  const activePreset = presets.find((x) => from === x.from && to === x.to)?.id ?? "";

  // **تصميمٌ احترافيّ (المالك ٢٠٢٦-١٠-٢٦)** — رأسٌ موحّدٌ بتبويبات الأطراف، المدّةُ في كرتٍ واحد،
  // والكشفُ أقسامٌ واضحة (دخل · خصومات · صافي) بدل قائمةٍ طويلة، والأشخاصُ كروتٌ لا جدول.
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
            disabled={inverted || loading || data === null}
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

      {/* ── المدّة في كرتٍ واحد ── */}
      <section className="surface mb-4 space-y-3 p-4">
        <Chips
          items={presets.map((x) => ({ id: x.id, label: x.label }))}
          value={activePreset}
          onChange={(id) => {
            const x = presets.find((p) => p.id === id);
            if (!x) return;
            setFrom(x.from);
            setTo(x.to);
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
            onChange={(e) => {
              setFrom(e.target.value);
              setPage(1);
            }}
          />
          <Input
            id="pf-to"
            type="date"
            icon={<IconDate />}
            label={m.shared.statement.to}
            value={to}
            onChange={(e) => {
              setTo(e.target.value);
              setPage(1);
            }}
          />
        </div>
        <p className="text-2xs text-ink-muted">{P.basis}</p>
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
            <PlatformView d={data as Platform} />
          ) : (
            <PartyView d={data as Party} tab={tab} page={page} onPage={setPage} />
          )}
        </div>
      )}
    </PageContainer>
  );
}

/** مبلغ بإشارته وعملته — الأخضر يزيد الربح والأحمر ينقصه. */
function Signed({ v, strong }: { v: number; strong?: boolean }) {
  return (
    <span
      className={`whitespace-nowrap ${strong ? "font-bold" : ""} ${v < 0 ? "text-danger" : v > 0 ? "text-success" : "text-ink-muted"}`}
    >
      <span dir="ltr" className="tabular-nums">
        {v < 0 ? "−" : v > 0 ? "+" : ""}
      </span>
      <Money value={Math.abs(v)} />
    </span>
  );
}

/** سطرٌ في الكشف — نقطةٌ بلون أثره، واسمُه، ومبلغُه بإشارته. */
function Line({ label, v }: { label: string; v: number }) {
  return (
    <li className="flex items-center justify-between gap-3 py-2">
      <span className="flex min-w-0 items-center gap-2 text-sm">
        <span
          aria-hidden
          className={`h-2 w-2 shrink-0 rounded-full ${v < 0 ? "bg-danger" : v > 0 ? "bg-success" : "bg-line"}`}
        />
        <span className="min-w-0">{label}</span>
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

function PlatformView({ d }: { d: Platform }) {
  const shown = d.lines.filter((l) => l.amount !== 0 || ALWAYS.has(l.key));
  const income = shown.filter((l) => INCOME_KEYS.has(l.key));
  const rest = shown.filter((l) => !INCOME_KEYS.has(l.key));
  const out = d.net - d.income;
  const gain = d.net >= 0;
  return (
    <div className="space-y-4">
      {/* ── الصافي بكرتٍ كبير، وتحته فحصُ «مجموع البنود = الصافي» ── */}
      <section
        className={`surface flex flex-wrap items-center gap-4 border-2 p-5 ${gain ? "border-success" : "border-danger"}`}
      >
        <span
          className={`inline-flex h-16 w-16 shrink-0 items-center justify-center rounded-full text-on-solid ${
            gain ? "bg-success-solid" : "bg-danger-solid"
          }`}
        >
          {gain ? <IconTrendUp size={30} /> : <IconTrendDown size={30} />}
        </span>
        <div className="min-w-0 flex-1">
          <p className="flex flex-wrap items-center gap-2 text-sm text-ink-muted">
            {P.lineNet}
            <Badge variant={gain ? "success" : "danger"}>{gain ? P.netProfit : P.netLoss}</Badge>
          </p>
          <p className={`figure ${gain ? "text-success" : "text-danger"}`}>
            <span dir="ltr">
              {d.net < 0 ? "−" : ""}
              {fmtMoney(Math.abs(d.net))}
            </span>
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
            <Alert>{P.checkBad.replace("{d}", `${fmtNum(d.check_diff)} ${CUR}`)}</Alert>
          )}
        </div>
      </section>

      <div className="grid grid-cols-1 gap-2 sm:grid-cols-3">
        <Tile icon={IconStore} label={P.cardIncome} value={fmtMoney(d.income)} tone="success" />
        <Tile icon={IconStatus} label={P.cardOut} value={fmtMoney(-out)} tone="danger" />
        <Tile
          icon={IconOrder}
          label={P.cardOrders}
          value={fmtNum(d.orders)}
          sub={`${P.cardSales}: ${fmtMoney(d.sales)}`}
        />
      </div>

      {/* ── الكشف قسمين: دخل ثمّ خصومات، وتحتهما المعادلة ── */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
        <Panel title={P.cardIncome} note={P.incomeNote}>
          <ul className="divide-y divide-line-soft">
            {income.map((l) => (
              <Line key={l.key} label={LINE_LABEL[l.key] ?? l.key} v={l.amount} />
            ))}
          </ul>
          <Subtotal label={P.subtotal} v={d.income} />
        </Panel>
        <Panel title={P.cardOut} note={P.costsNote}>
          <ul className="divide-y divide-line-soft">
            {rest.map((l) => (
              <Line key={l.key} label={LINE_LABEL[l.key] ?? l.key} v={l.amount} />
            ))}
          </ul>
          <Subtotal label={P.subtotal} v={out} />
        </Panel>
      </div>

      <section className="surface flex flex-wrap items-center justify-center gap-x-3 gap-y-1 p-4 text-sm">
        <Signed v={d.income} />
        <span className="text-ink-muted">+</span>
        <Signed v={out} />
        <span className="text-ink-muted">=</span>
        <span className="font-bold">{P.lineNet}</span>
        <Signed v={d.net} strong />
      </section>

      {/* ── أرقامٌ برّا الصافي ── */}
      <Panel title={P.outsideTitle}>
        <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
          <div className="flex items-start gap-3 rounded-control bg-field p-3">
            <span className="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-warning-solid text-on-solid">
              <IconHourglass size={18} />
            </span>
            <div className="min-w-0 flex-1">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <span className="text-sm font-medium">{P.pending}</span>
                <Money value={d.pending} className="font-bold" />
              </div>
              <p className="text-2xs text-ink-muted">{P.pendingHint}</p>
            </div>
          </div>
          <div className="flex items-start gap-3 rounded-control bg-field p-3">
            <span className="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-surface text-ink">
              <IconSwap size={18} />
            </span>
            <div className="min-w-0 flex-1">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <span className="text-sm font-medium">{P.outside}</span>
                <Signed v={d.outside} />
              </div>
              <p className="text-2xs text-ink-muted">{P.outsideHint}</p>
            </div>
          </div>
        </div>
      </Panel>
    </div>
  );
}

const PARTY_ICON: Record<Exclude<Tab, "platform">, typeof IconUser> = {
  customers: IconUser,
  reps: IconStatus,
  drivers: IconDriver,
  merchants: IconStore,
};

/** **كرتٌ لكلّ شخص** — أوّلُ عمودٍ رقمُه الكبير، والباقي خاناتٌ صغيرة. */
function PartyView({
  d,
  tab,
  page,
  onPage,
}: {
  d: Party;
  tab: Tab;
  page: number;
  onPage: (p: number) => void;
}) {
  const rows = d.rows ?? [];
  const cols = d.columns ?? [];
  const Icon = tab === "platform" ? IconUser : PARTY_ICON[tab];
  if (rows.length === 0) {
    return (
      <section className="surface flex flex-col items-center gap-3 p-8 text-center">
        <span className="inline-flex h-16 w-16 items-center justify-center rounded-full bg-success-solid text-on-solid">
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
        {rows.map((x) => (
          <li key={x.user_id} className="surface flex flex-col gap-3 p-4">
            <div className="flex items-center gap-3">
              <span className="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-accent text-on-bright">
                <Icon size={20} />
              </span>
              <span className="min-w-0 flex-1">
                <span className="block truncate font-bold">{x.name || x.phone}</span>
                <span dir="ltr" className="block truncate text-end text-2xs text-ink-muted">
                  {x.phone}
                </span>
              </span>
            </div>
            {cols.length > 0 && (
              <div className="rounded-control bg-field px-3 py-2.5">
                <span className="block text-2xs text-ink-muted">{COL_LABEL[cols[0]!] ?? cols[0]}</span>
                <span className="figure block">{fmtMoney(x.values[0] ?? 0)}</span>
              </div>
            )}
            {cols.length > 1 && (
              <dl className="grid grid-cols-2 gap-2">
                {cols.slice(1).map((c, j) => (
                  <div key={c} className="rounded-control border border-line-soft px-3 py-2">
                    <dt className="truncate text-2xs text-ink-muted">{COL_LABEL[c] ?? c}</dt>
                    <dd>
                      <Money value={x.values[j + 1] ?? 0} small />
                    </dd>
                  </div>
                ))}
              </dl>
            )}
          </li>
        ))}
      </ul>
      {d.total > d.per_page && (
        <div className="mt-4 flex justify-center">
          <Pagination page={page} total={d.total} perPage={d.per_page} onChange={onPage} />
        </div>
      )}
    </>
  );
}
