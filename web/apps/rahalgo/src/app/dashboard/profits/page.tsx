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
import { getMessages, defaultLocale, fmtNum, errorText, damascusDay } from "@rahalgo/i18n";
import {
  Alert,
  Button,
  Tabs,
  type TabDef,
  Input,
  PageContainer,
  PageHeader,
  EmptyState,
  LoadingState,
  ReloadState,
  Pagination,
  StatGrid,
  StatCard,
  Money,
  IconWallet,
  IconUser,
  IconStore,
  IconDriver,
  IconStatus,
  IconDate,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";

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
  recovered: P.lineRecovered,
  referrals: P.lineReferrals,
  targets: P.lineTargets,
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

  const tabs: TabDef<Tab>[] = [
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

  return (
    <PageContainer>
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <PageHeader icon={IconWallet} title={P.titlePl} subtitle={P.hintPl} />
        <Button
          variant="secondary"
          onClick={() => void exportCsv()}
          disabled={inverted || loading || data === null}
        >
          {P.export}
        </Button>
      </div>

      <Tabs
        items={tabs}
        value={tab}
        onChange={(k) => {
          setTab(k);
          setPage(1);
          setData(null);
        }}
        className="mb-4"
      />

      <div className="mb-2 flex flex-wrap items-end gap-2">
        {presets.map((x) => {
          const on = from === x.from && to === x.to;
          return (
            <Button
              key={x.id}
              variant={on ? "primary" : "secondary"}
              aria-pressed={on}
              onClick={() => {
                setFrom(x.from);
                setTo(x.to);
                setPage(1);
              }}
            >
              {x.label}
            </Button>
          );
        })}
        <div className="min-w-[8.75rem] flex-1 sm:w-40 sm:flex-none">
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
        </div>
        <div className="min-w-[8.75rem] flex-1 sm:w-40 sm:flex-none">
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
      </div>
      <p className="mb-4 text-2xs text-ink-muted">{P.basis}</p>

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
            <PartyView d={data as Party} page={page} onPage={setPage} />
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
      className={`${strong ? "font-bold" : ""} ${v < 0 ? "text-danger" : v > 0 ? "text-success" : "text-ink-muted"}`}
    >
      <span dir="ltr" className="tabular-nums">
        {v < 0 ? "−" : v > 0 ? "+" : ""}
      </span>
      <Money value={Math.abs(v)} />
    </span>
  );
}

function Row2({ label, v, strong }: { label: string; v: number; strong?: boolean }) {
  return (
    <li className="flex items-center justify-between gap-3 px-3 py-2">
      <span className={`text-sm ${strong ? "font-bold" : ""}`}>{label}</span>
      <Signed v={v} strong={strong} />
    </li>
  );
}

function PlatformView({ d }: { d: Platform }) {
  const shown = d.lines.filter((l) => l.amount !== 0 || ALWAYS.has(l.key));
  const income = shown.filter((l) => INCOME_KEYS.has(l.key));
  const rest = shown.filter((l) => !INCOME_KEYS.has(l.key));
  const out = d.net - d.income;
  return (
    <>
      <StatGrid>
        <StatCard
          label={`${P.cardIncome} (${CUR})`}
          value={fmtNum(d.income)}
          icon={IconStore}
          tone="success"
        />
        <StatCard label={`${P.cardOut} (${CUR})`} value={fmtNum(-out)} icon={IconStatus} tone="danger" />
        <StatCard
          label={`${P.cardNet} (${CUR})`}
          value={fmtNum(d.net)}
          icon={IconWallet}
          tone={d.net >= 0 ? "success" : "danger"}
        />
        <StatCard
          label={P.cardOrders}
          value={fmtNum(d.orders)}
          icon={IconStatus}
          sub={`${P.cardSales}: ${fmtNum(d.sales)} ${CUR}`}
        />
      </StatGrid>

      <ul className="mt-4 divide-y divide-line surface">
        {income.map((l) => (
          <Row2 key={l.key} label={LINE_LABEL[l.key] ?? l.key} v={l.amount} />
        ))}
        <Row2 label={P.cardIncome} v={d.income} strong />
        {rest.map((l) => (
          <Row2 key={l.key} label={LINE_LABEL[l.key] ?? l.key} v={l.amount} />
        ))}
        <Row2 label={P.lineNet} v={d.net} strong />
      </ul>

      {d.check_ok ? (
        <p className="mt-2 text-sm font-medium text-success">{P.checkOk}</p>
      ) : (
        <Alert className="mt-2">
          {P.checkBad.replace("{d}", `${fmtNum(d.check_diff)} ${CUR}`)}
        </Alert>
      )}

      <ul className="mt-4 divide-y divide-line surface">
        <li className="px-3 py-2">
          <div className="flex items-center justify-between gap-3">
            <span className="text-sm">{P.pending}</span>
            <Money value={d.pending} className="font-bold" />
          </div>
          <p className="text-2xs text-ink-muted">{P.pendingHint}</p>
        </li>
        <li className="px-3 py-2">
          <div className="flex items-center justify-between gap-3">
            <span className="text-sm">{P.outside}</span>
            <Signed v={d.outside} />
          </div>
          <p className="text-2xs text-ink-muted">{P.outsideHint}</p>
        </li>
      </ul>
    </>
  );
}

function PartyView({
  d,
  page,
  onPage,
}: {
  d: Party;
  page: number;
  onPage: (p: number) => void;
}) {
  const rows = d.rows ?? [];
  const cols = d.columns ?? [];
  if (rows.length === 0) return <EmptyState icon={IconUser} title={P.empty} />;
  return (
    <>
      <div className="surface overflow-x-auto">
        <table className="w-full text-sm">
          <thead className="text-2xs text-ink-muted">
            <tr className="border-b border-line-soft">
              <th className="p-2 text-start">{m.terms.name}</th>
              {cols.map((c) => (
                <th key={c} className="p-2 text-start">
                  {COL_LABEL[c] ?? c}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {rows.map((x) => (
              <tr key={x.user_id} className="border-b border-line-soft">
                <td className="p-2">
                  <span className="block font-medium">{x.name || x.phone}</span>
                  <span dir="ltr" className="block text-2xs text-ink-muted">
                    {x.phone}
                  </span>
                </td>
                {cols.map((c, i) => (
                  <td key={c} className="whitespace-nowrap p-2">
                    <Money value={x.values[i] ?? 0} small className={i === 0 ? "font-bold" : undefined} />
                  </td>
                ))}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      <Pagination page={page} total={d.total} perPage={d.per_page} onChange={onPage} />
    </>
  );
}
