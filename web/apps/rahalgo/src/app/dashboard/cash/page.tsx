"use client";

/**
 * **النقد والصندوق** — تبويبُ الخزينة لِما في أيدي السائقين ولِما للمتاجر نقداً.
 *
 * (قراراتُ المالك ٢٠٢٦-١٠-٠٤.) الرابطُ باقٍ `/dashboard/cash`، والخزينةُ تفتحه
 * تبويباً. وصندوقُ المكتب والإغلاقُ اليوميّ في صفحة الخزينة لا هنا.
 *
 * # والقِدَمُ هو الإشارة لا المقدار
 *
 * خمسون ألفاً قُبضت قبل ساعة **عملٌ يجري**، وخمسون ألفاً منذ أسبوعٍ **مسألةٌ
 * أخرى**. و«منذ» تُحسب من أقدم مالٍ باقٍ بيده — والأقدمُ يُسدَّد أوّلاً، فلا
 * تُصفّرها تسليمةٌ جزئيّة.
 *
 * # والسقفُ كما يمنع
 *
 * ما بالجيب + نقدُ طلباتٍ بيده لم تُغلق، مقابلَ سقفه الخاصّ أو العامّ — وهو
 * ما يقارنه حارسُ الإسناد. فلا يُرى «تحت السقف» من مُنع فعلاً.
 *
 * **وقراراتُ ٢٠٢٦-١٠-١٥:** من جيبُه فارغٌ ونقدُ طلباته المفتوحة يبلغ سقفَه يظهر هنا
 * ويُعدّ · و«فوق السقف» أكبرُ منه لا يساويه (الحارسُ يقبل حتّى السقف) — ومن كان عنده
 * تماماً يُعلَّم «عند السقف».
 */

import { useEffect, useMemo, useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtMoney, fmtDate, fmtDateTime, errorText } from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  Chips,
  EmptyState,
  FormActions,
  Input,
  LoadingState,
  Modal,
  Money,
  PageContainer,
  ReloadState,
  useLiveData,
  IconDate,
  IconStatus,
  IconStore,
  IconUser,
  IconWallet,
  IconWarning,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { DriverCashReceive } from "@/components/admin/DriverCashReceive";
import { MerchantSettlement } from "@/components/admin/MerchantSettlement";
import { OpenLink } from "@/components/admin/OpenLink";
import { useCanOpen } from "@/lib/policy";
import { PageHero } from "@/components/admin/PageHero";
import { Tile } from "@/components/admin/DashParts";

const m = getMessages(defaultLocale);
const C = m.admin.cashOutstanding;

interface Holder {
  driver_id: string;
  name: string;
  phone: string;
  held: number;
  open_cash: number;
  exposure: number;
  limit: number;
  over_limit: boolean;
  /** **عند السقف تماماً** — ليس فوقه، ولا يأخذ نقديّاً جديداً. */
  at_limit: boolean;
  oldest_at: string | null;
  overdue: boolean;
  last_settled_at: string | null;
  on_shift: boolean;
  /** **حقُّ السائق بمحفظته** — أجورُه، ليُقرأ الصافي (٢٠٢٦-١٠-٢٦). */
  wallet: number;
}

interface CashData {
  holders: Holder[];
  total: number;
  limit: number;
  over_count: number;
  overdue_count: number;
  overdue_days: number;
  /** **ساعةُ فتح الدوام** — بدايةُ يوم الحساب (`platform.day_start_hour`). */
  day_start_hour?: number;
}

interface StoreDue {
  merchant_id: string;
  name: string;
  settlement_method: "cash" | "wallet";
  outstanding: number;
  count: number;
  oldest_at: string | null;
}

type Tab = "drivers" | "stores";
type Filter = "all" | "over" | "overdue";

/** أيامٌ مضت على أقدم مالٍ باقٍ — و`0` إن لا تاريخ. */
function daysHeld(iso: string | null): number {
  if (!iso) return 0;
  return Math.max(0, Math.floor((Date.now() - new Date(iso).getTime()) / 86_400_000));
}

/** «اليوم» · «يوم واحد» · «يومان» · «٣ أيام» · «١١ يوماً» — بأرقامٍ غربيّة. */
function daysLabel(n: number): string {
  if (n <= 0) return C.today;
  if (n === 1) return C.day1;
  if (n === 2) return C.day2;
  if (n <= 10) return C.daysFew.replace("{n}", fmtNum(n));
  return C.daysMany.replace("{n}", fmtNum(n));
}

/** قراءةُ الحال من الرابط — و`over=1` القديمُ يبقى يعمل. */
function readURL(): { tab: Tab; filter: Filter } {
  if (typeof window === "undefined") return { tab: "drivers", filter: "all" };
  const q = new URLSearchParams(window.location.search);
  const tab: Tab = q.get("tab") === "stores" ? "stores" : "drivers";
  const f = q.get("filter");
  const filter: Filter =
    q.get("over") === "1" || f === "over"
      ? "over"
      : f === "overdue"
        ? f
        : "all";
  return { tab, filter };
}

function writeURL(tab: Tab, filter: Filter) {
  if (typeof window === "undefined") return;
  const q = new URLSearchParams();
  if (tab !== "drivers") q.set("tab", tab);
  if (filter !== "all") q.set("filter", filter);
  const s = q.toString();
  window.history.replaceState(null, "", s ? `?${s}` : window.location.pathname);
}

export default function CashPage() {
  const [tab, setTab] = useState<Tab>(() => readURL().tab);
  const [filter, setFilter] = useState<Filter>(() => readURL().filter);
  useEffect(() => writeURL(tab, filter), [tab, filter]);

  return (
    <PageContainer>
      <PageHero<Tab>
        icon={IconWallet}
        title={C.tabTitle}
        subtitle={C.tabHint}
        tabs={[
          { key: "drivers", label: C.tabDrivers, icon: IconUser },
          { key: "stores", label: C.tabStores, icon: IconStore },
        ]}
        value={tab}
        onChange={setTab}
      />
      {tab === "drivers" ? (
        <DriversCash filter={filter} setFilter={setFilter} />
      ) : (
        <StoresCash />
      )}
    </PageContainer>
  );
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **نقدُ السائقين — قائمةٌ وسائقٌ مختار** (قرارُ المالك ٢٠٢٦-١٠-٢٦: «التصميم لازم يكون أقوى وواضح —
 * صندوق النقد يخصّ السائقين، ولازم أقدر أحدّد أي سائق»)
 * ══════════════════════════════════════════════════════════════════════
 *
 * يميناً كلُّ السائقين (بحثٌ باسمٍ أو رقم، ومن معه نقدٌ أوّلاً)، **ويساراً السائقُ المختار بلوحةٍ كاملة**:
 * ما يسلّمه للمكتب مفصّلاً، وسقفُه بكلمات، وزرُّ الاستلام، **وكشفُ نقده طلباً طلباً.**
 *
 * **ولا محفظةَ هنا** (المالك ٢٠٢٦-١٠-٢٦: «مالنا علاقة بمحفظته») — أجرتُه نزلت على محفظته
 * لحالها، والكاشُ للمكتب أجرةُ التوصيل والعمولة. **ولا «على الدوام» ولا «تصدير»** (طلبُه).
 */
interface DriverRow {
  id: string;
  full_name: string;
  phone: string;
  on_shift?: boolean;
}

interface CashEntry {
  id: number;
  amount: number;
  kind: string;
  order_number: number | null;
  note: string;
  created_at: string;
  /** حقولُ قيود الطلب (٢٠٢٦-١٠-٢٦) — تغيب عن غيرها. */
  order_id?: string;
  customer_name?: string;
  merchant_name?: string;
  fee_part?: number;
  commission_part?: number;
  commission_waived?: number;
  waived_reason?: string;
  /** **ما أُعفي منه الزبونُ من الأجرة** — فوق الصفر: «توصيل مجاني» على الكرت. */
  free_waived?: number;
}

type Statement = { held: number; limit: number; entries: CashEntry[]; open_cash: number };

/** «من ٥ دقائق» · «من ٣ ساعات» · «من يومين» — مدّةٌ تُقرأ لا كلمةُ «اليوم» وحدها. */
function sinceLabel(iso: string | null): string {
  if (!iso) return "—";
  const mins = Math.max(0, Math.floor((Date.now() - new Date(iso).getTime()) / 60_000));
  if (mins < 60) return C.sinceMinutes.replace("{n}", fmtNum(Math.max(1, mins)));
  const hours = Math.floor(mins / 60);
  if (hours < 24) return C.sinceHours.replace("{n}", fmtNum(hours));
  return C.since.replace("{d}", daysLabel(Math.floor(hours / 24)));
}

/**
 * **ما بيده مفصّلاً** — أجرةُ توصيلٍ وعمولة. والتسليمُ يُسدّد الأقدمَ أوّلاً، فالباقي بيده
 * أحدثُ قيود الطلبات: تُجمع من الأحدث حتّى تبلغ ما بيده. **ويُعرض التفصيلُ إن طابق تماماً** —
 * وتسليمٌ جزئيٌّ قسم قيداً لا يُقسَم تفصيلُه تخميناً.
 */
function heldSplit(
  entries: CashEntry[],
  held: number,
): { fee: number; commission: number; orders: number; free: number } | null {
  if (held <= 0) return null;
  let sum = 0;
  let fee = 0;
  let commission = 0;
  let orders = 0;
  let free = 0;
  for (const e of entries) {
    if (e.fee_part == null && e.commission_part == null && e.commission_waived == null) continue;
    if (e.kind === "order_collection") {
      const waived = e.commission_waived ?? 0;
      sum += e.amount - waived;
      fee += e.fee_part ?? 0;
      commission += (e.commission_part ?? 0) - waived;
      orders += 1;
      if ((e.free_waived ?? 0) > 0) free += 1;
    }
    if (sum >= held) break;
  }
  return sum === held ? { fee, commission, orders, free } : null;
}

function DriversCash({ filter, setFilter }: { filter: Filter; setFilter: (f: Filter) => void }) {
  const { can } = useAuth();
  const canSettle = can("finance.manage");
  const canUser = useCanOpen().user;
  const [target, setTarget] = useState<Holder | null>(null);
  const [picked, setPicked] = useState("");
  const [query, setQuery] = useState("");
  const [done, setDone] = useState("");

  const { data, error, reload } = useLiveData<CashData>(() => api("/api/v1/admin/cash/outstanding"), ["wallet", "order"]);
  const drivers = useLiveData<{ users: DriverRow[] }>(() => api("/api/v1/admin/users?role=driver&per_page=200"), ["user", "wallet"]);

  const holders = useMemo(() => data?.holders ?? [], [data]);
  const byID = useMemo(() => new Map(holders.map((h) => [h.driver_id, h])), [holders]);

  // **كلُّ السائقين في القائمة** — من معه نقدٌ أوّلاً (بأقدمه)، ثمّ الباقون بأسمائهم.
  const list = useMemo(() => {
    const rows: Holder[] = [...holders];
    for (const u of drivers.data?.users ?? []) {
      if (byID.has(u.id)) continue;
      rows.push({
        driver_id: u.id,
        name: u.full_name,
        phone: u.phone,
        held: 0,
        open_cash: 0,
        exposure: 0,
        limit: data?.limit ?? 0,
        over_limit: false,
        at_limit: false,
        oldest_at: null,
        overdue: false,
        last_settled_at: null,
        on_shift: !!u.on_shift,
        wallet: 0,
      });
    }
    const q = query.trim().toLowerCase();
    return rows.filter((h) => {
      if (filter === "over" && !h.over_limit) return false;
      if (filter === "overdue" && !h.overdue) return false;
      if (!q) return true;
      return (h.name || "").toLowerCase().includes(q) || h.phone.includes(q);
    });
  }, [holders, drivers.data, byID, data?.limit, filter, query]);

  useEffect(() => {
    if (!picked && list[0]) setPicked(list[0].driver_id);
  }, [list, picked]);

  if (!data && error) return <ReloadState label={C.loadFailed} onRetry={reload} />;
  if (!data) return <LoadingState />;

  // **أقدمُ مبلغٍ باقٍ بمدّته** — من الأقدم بين الحاملين.
  const oldestAt = holders.reduce<string | null>(
    (o, h) => (h.oldest_at && (!o || h.oldest_at < o) ? h.oldest_at : o),
    null,
  );
  const oldestDays = daysHeld(oldestAt);
  const sel = list.find((h) => h.driver_id === picked) ?? null;

  return (
    <>
      <div className="mb-4 grid grid-cols-2 gap-2 lg:grid-cols-4">
        <Tile icon={IconWallet} label={C.total} value={fmtMoney(data.total)} tone={data.total > 0 ? "warning" : "muted"} onClick={() => setFilter("all")} />
        <Tile icon={IconUser} label={C.holders} value={fmtNum(holders.length)} onClick={() => setFilter("all")} />
        <Tile icon={IconWarning} label={C.overCount} value={fmtNum(data.over_count)} tone={data.over_count > 0 ? "danger" : "muted"} onClick={() => setFilter(filter === "over" ? "all" : "over")} />
        <Tile
          icon={IconDate}
          label={C.oldestAge}
          value={oldestAt ? sinceLabel(oldestAt) : "—"}
          tone={data.overdue_count > 0 ? "danger" : oldestDays >= 1 ? "warning" : oldestAt ? "default" : "muted"}
          onClick={() => setFilter(filter === "overdue" ? "all" : "overdue")}
        />
      </div>

      {done && (
        <Alert tone="success" className="mb-3" onDismiss={() => setDone("")}>
          {done}
        </Alert>
      )}

      <div className="grid grid-cols-1 gap-4 lg:grid-cols-[minmax(16rem,22rem)_1fr]">
        {/* ── قائمةُ السائقين ── */}
        <section className="surface flex max-h-[75vh] flex-col overflow-hidden">
          <div className="space-y-2 border-b border-line-soft p-3">
            <Input id="cash-search" icon={<IconUser size={15} />} placeholder={C.search} value={query} onChange={(e) => setQuery(e.target.value)} />
            <Chips
              wrap
              items={[
                { id: "all" as Filter, label: C.filterAll },
                { id: "over" as Filter, label: C.filterOver, count: data.over_count },
                { id: "overdue" as Filter, label: C.overdueCount.replace("{n}", fmtNum(data.overdue_days)), count: data.overdue_count },
              ]}
              value={filter}
              onChange={setFilter}
            />
          </div>
          {list.length === 0 ? (
            <div className="p-4">
              <EmptyState icon={IconStatus} title={C.noMatch} />
            </div>
          ) : (
            <ul className="flex-1 overflow-y-auto">
              {list.map((h) => {
                const on = h.driver_id === picked;
                return (
                  <li key={h.driver_id}>
                    <button
                      type="button"
                      onClick={() => setPicked(h.driver_id)}
                      className={`flex w-full items-center gap-3 border-b border-line-soft px-3 py-2.5 text-start transition-colors ${
                        on ? "bg-row-hover" : "hover:bg-row-hover"
                      }`}
                    >
                      <span className={`h-2.5 w-2.5 shrink-0 rounded-full ${h.on_shift ? "bg-success" : "bg-field"}`} />
                      <span className="min-w-0 flex-1">
                        <span className={`block truncate ${on ? "font-bold" : "font-medium"}`}>{h.name || h.phone}</span>
                        <span className="block text-2xs text-ink-muted" dir="ltr">{h.phone}</span>
                      </span>
                      <span className={`shrink-0 text-sm font-bold tabular-nums ${h.held > 0 ? (h.over_limit ? "text-danger" : "text-warning") : "text-ink-muted"}`}>
                        {fmtMoney(h.held)}
                      </span>
                    </button>
                  </li>
                );
              })}
            </ul>
          )}
        </section>

        {/* ── السائقُ المختار ── */}
        {sel ? (
          <DriverPanel
            h={sel}
            canUser={canUser}
            canSettle={canSettle}
            onReceive={() => setTarget(sel)}
            dayHour={data.day_start_hour ?? 9}
            onWaived={(msg) => {
              setDone(msg);
              reload();
            }}
          />
        ) : (
          <section className="surface p-6">
            <EmptyState icon={IconUser} title={C.pickDriver} />
          </section>
        )}
      </div>

      {target && (
        <DriverCashReceive
          driverID={target.driver_id}
          driverName={target.name || target.phone}
          held={target.held}
          onClose={() => setTarget(null)}
          onDone={(rest) => {
            setTarget(null);
            setDone(C.receiveDone.replace("{rest}", fmtMoney(rest)));
            reload();
          }}
        />
      )}
    </>
  );
}

/** **لوحةُ السائق المختار** — ما يسلّمه للمكتب مفصّلاً، والسقف بكلمات، والاستلام، وكشفُ نقده طلباً طلباً. */
function DriverPanel({
  h,
  canUser,
  canSettle,
  onReceive,
  onWaived,
  dayHour,
}: {
  h: Holder;
  canUser: boolean;
  canSettle: boolean;
  onReceive: () => void;
  onWaived: (msg: string) => void;
  dayHour: number;
}) {
  const st = useLiveData<Statement>(
    () => api(`/api/v1/admin/drivers/${h.driver_id}/cash`),
    ["wallet", "order"],
    [h.driver_id],
  );
  const [waive, setWaive] = useState<CashEntry | null>(null);
  const KIND = m.admin.users.profile.roleTabs.cashKinds as Record<string, string>;
  const held = st.data?.held ?? h.held;
  const open = st.data?.open_cash ?? h.open_cash;
  const limit = st.data?.limit ?? h.limit;
  const exposure = held + open;
  const ratio = limit > 0 ? exposure / limit : 1;
  const pct = Math.min(100, Math.round(ratio * 100));
  const left = Math.max(0, limit - exposure);
  const capTone = ratio > 0.9 ? "danger" : ratio > 0.7 ? "warning" : "success";
  const split = st.data ? heldSplit(st.data.entries, held) : null;
  const d = daysHeld(h.oldest_at);
  const [capA, capRest = ""] = C.capLine1.split("{x}");
  const [capB, capC = ""] = capRest.split("{cap}");
  return (
    <section className="surface space-y-4 p-5">
      <div className="flex flex-wrap items-start justify-between gap-3">
        <div className="min-w-0">
          <OpenLink allowed={canUser} href={`/dashboard/users/${h.driver_id}`} className="hover:underline">
            <h2 className="heading-section truncate">{h.name || h.phone}</h2>
          </OpenLink>
          <p className="text-sm text-ink-muted" dir="ltr">{h.phone}</p>
        </div>
        <span className="flex flex-wrap gap-1">
          <Badge variant={h.on_shift ? "success" : "neutral"}>{h.on_shift ? C.onShift : C.offShift}</Badge>
          {held > 0 && <Badge variant={h.overdue ? "danger" : d >= 1 ? "warning" : "neutral"}>{sinceLabel(h.oldest_at)}</Badge>}
        </span>
      </div>

      {/* ── ما يسلّمه للمكتب — رقمٌ واحدٌ كبيرٌ مفصّل ── */}
      <div className={`flex flex-wrap items-center gap-4 rounded-card p-5 ${held > 0 ? "bg-warning-tint" : "bg-field"}`}>
        <span
          className={`inline-flex h-14 w-14 shrink-0 items-center justify-center rounded-full ${
            held > 0 ? "bg-warning-solid text-on-solid" : "bg-surface text-ink-muted"
          }`}
        >
          <IconWallet size={26} />
        </span>
        <div className="min-w-0 flex-1">
          <span className="block text-sm text-ink-muted">{C.mustHand}</span>
          <span className={`figure block ${held > 0 ? "text-warning" : "text-ink-muted"}`}>{fmtMoney(held)}</span>
          {split && (
            <span className="mt-1 block text-sm font-bold">
              {C.splitOrders.replace("{n}", fmtNum(split.orders))}
              {split.free > 0 && <span className="text-success"> · {C.splitFreeCount.replace("{n}", fmtNum(split.free))}</span>}
            </span>
          )}
          {split && (
            <span className="mt-1 block text-sm">
              {C.splitFee} <b className="tabular-nums">{fmtNum(split.fee)}</b>
              <span className="text-ink-muted"> · </span>
              {C.splitCommission} <b className="tabular-nums">{fmtNum(split.commission)}</b>
            </span>
          )}
          <span className="mt-1 block text-xs text-ink-muted">
            {C.lastSettled}: {h.last_settled_at ? <span dir="ltr">{fmtDate(h.last_settled_at)}</span> : C.never}
          </span>
        </div>
        {canSettle && held > 0 && (
          <Button size="lg" onClick={onReceive}>
            {C.receiveAmount.replace("{n}", fmtMoney(held))}
          </Button>
        )}
      </div>

      {/* ── السقفُ بكلمات ── */}
      <div className="space-y-1.5 rounded-card border border-line-soft p-4">
        <p className="text-sm">
          {capA}
          <b className="tabular-nums">{fmtMoney(exposure)}</b>
          {capB}
          <b className="tabular-nums">{fmtMoney(limit)}</b>
          {capC}
        </p>
        <span className="block h-2.5 w-full overflow-hidden rounded-full bg-field" aria-hidden>
          <span
            className={`block h-full rounded-full ${capTone === "danger" ? "bg-danger" : capTone === "warning" ? "bg-warning" : "bg-success"}`}
            style={{ width: `${Math.max(2, pct)}%` }}
          />
        </span>
        <p className={`text-xs ${capTone === "danger" ? "font-bold text-danger" : "text-ink-muted"}`}>
          {capTone === "danger" ? C.capMustHand : C.capLeft.replace("{n}", fmtMoney(left))}
          {open > 0 && <span className="text-ink-muted"> · {C.capOpen.replace("{n}", fmtMoney(open))}</span>}
        </p>
      </div>

      {/* ── كشفُ نقده — كرتٌ لكلّ طلب ── */}
      <div className="space-y-2">
        <h3 className="heading-card">{C.statementTitle}</h3>
        {!st.data ? (
          <LoadingState variant="text" />
        ) : st.data.entries.length === 0 ? (
          <p className="text-sm text-ink-muted">{C.statementEmpty}</p>
        ) : (
          <div className="space-y-4">
            {groupByAccountDay(st.data.entries.filter((e) => e.kind !== "commission_waived"), dayHour).map((g) => (
              <div key={g.key} className="space-y-2">
                <div className={`flex flex-wrap items-center justify-between gap-2 rounded-control px-3 py-2 ${g.current ? "bg-field" : "bg-warning-tint"}`}>
                  <span className="min-w-0">
                    <span className="block text-sm font-bold">{dayRangeLabel(g.key, dayHour)}</span>
                    <span className="block text-2xs text-ink-muted">
                      {(g.current ? C.dayOpen : C.dayDue).replaceAll("{h}", fmtNum(dayHour))}
                    </span>
                  </span>
                  <span className="shrink-0 text-sm">
                    {C.dayOrders.replace("{n}", fmtNum(g.orders))}
                    {g.free > 0 && <span className="text-success"> · {C.splitFreeCount.replace("{n}", fmtNum(g.free))}</span>}
                    <span className="text-ink-muted"> · </span>
                    <b className="tabular-nums text-warning">{fmtMoney(g.cash)}</b>
                  </span>
                </div>
                <ul className="grid grid-cols-1 gap-2 xl:grid-cols-2">
                  {g.entries.map((e) =>
                    e.kind === "order_collection" ? (
                      <OrderCashCard key={e.id} e={e} canWaive={canSettle} onWaive={() => setWaive(e)} />
                    ) : (
                      <li key={e.id} className="flex items-center gap-3 rounded-card border border-line-soft bg-field px-4 py-3 text-sm">
                        <span
                          className={`inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full ${
                            e.amount >= 0 ? "bg-warning-tint text-warning" : "bg-success-tint text-success"
                          }`}
                        >
                          <IconWallet size={16} />
                        </span>
                        <span className="min-w-0 flex-1">
                          <span className="block font-bold">{KIND[e.kind] ?? e.kind}</span>
                          {e.note && <span className="block truncate text-xs text-ink-muted">{e.note}</span>}
                          <span className="block text-2xs text-ink-muted" dir="ltr">{fmtDateTime(e.created_at)}</span>
                        </span>
                        <span className={`shrink-0 font-bold tabular-nums ${e.amount >= 0 ? "text-warning" : "text-success"}`} dir="ltr">
                          {e.amount >= 0 ? "+" : "−"}
                          {fmtNum(Math.abs(e.amount))}
                        </span>
                      </li>
    
                    ),
                  )}
                </ul>
              </div>
            ))}
          </div>
        )}
      </div>

      {waive && (
        <WaiveCommission
          e={waive}
          onClose={() => setWaive(null)}
          onDone={(amount) => {
            setWaive(null);
            st.reload();
            onWaived(C.waiveDone.replace("{n}", fmtMoney(amount)).replace("{o}", String(waive.order_number ?? "")));
          }}
        />
      )}
    </section>
  );
}

/**
 * **يومُ الحساب يبدأ بساعة فتح الدوام** (المالك ٢٠٢٦-١٠-٢٦: «كل يوم الساعة 9 صبح نستلم من
 * السائقين… اذا فتحنا 8 يصير الحساب ع8») — من `platform.day_start_hour` بتوقيت دمشق (+3 طوال
 * السنة). وما قُبض قبلها يُحسب على اليوم السابق.
 */
const DAMASCUS_OFFSET_H = 3;

/** مفتاحُ يوم الحساب «YYYY-MM-DD» — دمشق +3 ثابتةٌ بلا توقيتٍ صيفيّ منذ ٢٠٢٢. */
function accountDay(iso: string, hour: number): string {
  const t = new Date(iso).getTime() + (DAMASCUS_OFFSET_H - hour) * 3_600_000;
  return new Date(t).toISOString().slice(0, 10);
}

/** «من ٩ الصبح ٢٦/١٠ لـ ٩ الصبح ٢٧/١٠». */
function dayRangeLabel(key: string, hour: number): string {
  const from = new Date(`${key}T${String(hour).padStart(2, "0")}:00:00+03:00`);
  const to = new Date(from.getTime() + 86_400_000);
  return C.dayRange
    .replaceAll("{h}", fmtNum(hour))
    .replace("{from}", fmtDate(from.toISOString()))
    .replace("{to}", fmtDate(to.toISOString()));
}

function groupByAccountDay(entries: CashEntry[], hour: number) {
  const today = accountDay(new Date().toISOString(), hour);
  const out: { key: string; current: boolean; entries: CashEntry[]; orders: number; free: number; cash: number }[] = [];
  for (const e of entries) {
    const key = accountDay(e.created_at, hour);
    let g = out.find((x) => x.key === key);
    if (!g) {
      g = { key, current: key === today, entries: [], orders: 0, free: 0, cash: 0 };
      out.push(g);
    }
    g.entries.push(e);
    if (e.kind === "order_collection") {
      g.orders += 1;
      if ((e.free_waived ?? 0) > 0) g.free += 1;
      g.cash += e.amount - (e.commission_waived ?? 0);
    }
  }
  return out;
}

/** **كرتُ طلبٍ في كشف النقد** — الزبون والمتجر ورقمُ الطلب، والتاريخ، وأجرةُ التوصيل والعمولة والمجموع. */
function OrderCashCard({ e, canWaive, onWaive }: { e: CashEntry; canWaive: boolean; onWaive: () => void }) {
  const commission = e.commission_part ?? 0;
  const waived = e.commission_waived ?? 0;
  const hasSplit = e.fee_part != null || e.commission_part != null;
  const total = e.amount - waived;
  // **التوصيلةُ المجانيّة تُقال** (المالك ٢٠٢٦-١٠-٢٦: «لازم يذكر التوصيلة ويكتب مجانية 0»).
  const free = (e.free_waived ?? 0) > 0;
  return (
    <li className="space-y-2 rounded-card border border-line-soft bg-field p-4 text-sm">
      <div className="flex items-start gap-3">
        <span className="min-w-0 flex-1">
          <span className="block truncate font-bold">{e.customer_name || C.customerFallback}</span>
          <span className="block truncate text-xs text-ink-muted">
            {e.merchant_name && <>{e.merchant_name} · </>}
            {e.order_number != null && <span dir="ltr">#{e.order_number}</span>}
          </span>
        </span>
        <span className="shrink-0 text-2xs text-ink-muted" dir="ltr">
          {fmtDateTime(e.created_at)}
        </span>
      </div>
      {hasSplit ? (
        <div className="grid grid-cols-1 gap-1.5 sm:grid-cols-3 text-center">
          <span className={`rounded-control px-2 py-1.5 ${free ? "bg-success-tint" : "bg-surface"}`}>
            <span className="block text-2xs text-ink-muted">{free ? C.splitFeeFree : C.splitFee}</span>
            <b className={`tabular-nums ${free ? "text-success" : ""}`}>{fmtNum(e.fee_part ?? 0)}</b>
          </span>
          <span className="rounded-control bg-surface px-2 py-1.5">
            <span className="block text-2xs text-ink-muted">{C.splitCommission}</span>
            <b className="tabular-nums">{fmtNum(commission - waived)}</b>
          </span>
          <span className="rounded-control bg-warning-tint px-2 py-1.5">
            <span className="block text-2xs text-ink-muted">{C.splitTotal}</span>
            <b className="tabular-nums text-warning">{fmtNum(total)}</b>
          </span>
        </div>
      ) : (
        <p className="flex justify-between">
          <span className="text-ink-muted">{C.splitTotal}</span>
          <b className="tabular-nums text-warning">{fmtNum(total)}</b>
        </p>
      )}
      {waived > 0 && (
        <p className="text-xs text-ink-muted">
          {C.waivedNote}
          {e.waived_reason && <> · {e.waived_reason}</>}
        </p>
      )}
      {canWaive && commission > 0 && waived === 0 && e.order_id && (
        <div className="flex justify-end">
          <Button variant="ghost" onClick={onWaive}>
            {C.waiveBtn}
          </Button>
        </div>
      )}
    </li>
  );
}

/** **«ما استلمت العمولة»** — قرارُ المالك ٢٠٢٦-١٠-٢٦: تُمسح نهائيّاً بقيدٍ عكسيّ، وبسببٍ مكتوب. */
function WaiveCommission({ e, onClose, onDone }: { e: CashEntry; onClose: () => void; onDone: (amount: number) => void }) {
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const commission = e.commission_part ?? 0;
  return (
    <Modal open onClose={onClose} title={C.waiveTitle.replace("{o}", String(e.order_number ?? ""))}>
      <form
        className="space-y-3"
        onSubmit={(ev) => {
          ev.preventDefault();
          if (!reason.trim()) {
            setError(C.waiveNeedReason);
            return;
          }
          setBusy(true);
          setError("");
          void (async () => {
            try {
              await api(`/api/v1/admin/orders/${e.order_id}/waive-commission`, {
                method: "POST",
                body: JSON.stringify({ reason: reason.trim() }),
              });
              onDone(commission);
            } catch (err) {
              setError(errorText(err));
              setBusy(false);
            }
          })();
        }}
      >
        <p className="flex items-center justify-between rounded-control bg-field px-3 py-2 text-sm">
          <span className="text-ink-muted">{C.splitCommission}</span>
          <Money value={commission} small />
        </p>
        <p className="text-sm text-ink-muted">{C.waiveHint}</p>
        <Input
          id="waive-reason"
          label={C.waiveReason}
          placeholder={C.waiveReasonPh}
          required
          autoFocus
          maxLength={300}
          value={reason}
          onChange={(ev) => setReason(ev.target.value)}
        />
        {error && <Alert>{error}</Alert>}
        <FormActions submit onCancel={onClose} busy={busy} saveLabel={C.waiveConfirm} />
      </form>
    </Modal>
  );
}

/**
 * **مستحقّاتُ المتاجر نقداً — كلُّها في مكانٍ واحد.**
 *
 * أيّاً كانت طريقةُ المتجر اليوم: متجرٌ له مستحقٌّ نقديٌّ ثمّ حُوّل إلى المحفظة
 * يبقى هنا حتّى يُدفع (المشكلة ٣). **والدفعُ من هنا** بزرّ «المستحقات» (٢٠٢٦-١٠-٠٥).
 */
function StoresCash() {
  const { can } = useAuth();
  // **ومستحقّاتُ المتجر تُدفع من هنا** (قرارُ المالك ٢٠٢٦-١٠-٠٥): كانت «تُدفع من ملفّ
  // المتجر» — **والماليّةُ لا تفتح ملفَّ المتجر** (`merchants.read`)، فبقي الزرُّ وراء ٤٠٣.
  const canMerchant = useCanOpen().merchant;
  const canPayView = can("finance.read");
  const [paying, setPaying] = useState<StoreDue | null>(null);
  const { data, error, reload } = useLiveData<{ merchants: StoreDue[]; total: number }>(
    () => api("/api/v1/admin/cash/merchant-dues"),
    ["wallet", "order"],
  );
  if (!data && error) return <ReloadState label={C.loadFailed} onRetry={reload} />;
  if (!data) return <LoadingState />;
  const rows = data.merchants ?? [];
  const orders = rows.reduce((n, d) => n + d.count, 0);
  const oldestAt = rows.reduce<string | null>(
    (o, d) => (d.oldest_at && (!o || d.oldest_at < o) ? d.oldest_at : o),
    null,
  );
  const oldestDays = daysHeld(oldestAt);

  // **تصميمٌ احترافيّ** (المالك ٢٠٢٦-١٠-٢٦) — أرقامٌ بنظرة، وكرتٌ لكلّ متجرٍ بمبلغه وزرِّ دفعه.
  return (
    <>
      <div className="mb-4 grid grid-cols-2 gap-2 lg:grid-cols-4">
        <Tile icon={IconWallet} label={C.storesTotal} value={fmtMoney(data.total)} tone={data.total > 0 ? "warning" : "muted"} />
        <Tile icon={IconStore} label={C.storesCountLabel} value={fmtNum(rows.length)} tone={rows.length ? "default" : "muted"} />
        <Tile icon={IconStatus} label={C.storesOrdersLabel} value={fmtNum(orders)} tone={orders ? "default" : "muted"} />
        <Tile
          icon={IconDate}
          label={C.storesOldest}
          value={oldestAt ? sinceLabel(oldestAt) : "—"}
          tone={oldestDays >= 3 ? "danger" : oldestDays >= 1 ? "warning" : oldestAt ? "default" : "muted"}
        />
      </div>

      <section className="surface mb-4 flex items-start gap-3 p-4">
        <span className="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-field text-ink-muted">
          <IconStore size={18} />
        </span>
        <span className="min-w-0">
          <span className="block font-bold">{C.storesWhatTitle}</span>
          <span className="block text-sm text-ink-muted">{C.storesHint}</span>
        </span>
      </section>

      {rows.length === 0 ? (
        <section className="surface flex flex-col items-center gap-3 p-8 text-center">
          <span className="inline-flex h-16 w-16 items-center justify-center rounded-full bg-success-solid text-on-solid">
            <IconStore size={28} />
          </span>
          <h2 className="heading-section text-success">{C.storesEmpty}</h2>
          <p className="max-w-md text-sm text-ink-muted">{C.storesEmptyHint}</p>
        </section>
      ) : (
        <ul className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3">
          {rows.map((d) => {
            const days = daysHeld(d.oldest_at);
            return (
              <li key={d.merchant_id} className="surface flex flex-col gap-3 p-4">
                <div className="flex items-start gap-3">
                  <span className="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-warning-tint text-warning">
                    <IconStore size={20} />
                  </span>
                  <span className="min-w-0 flex-1">
                    <OpenLink allowed={canMerchant} href={`/dashboard/merchants/${d.merchant_id}`} className="block truncate font-bold hover:underline">
                      {d.name}
                    </OpenLink>
                    <span className="mt-1 flex flex-wrap gap-1">
                      <Badge variant="neutral">{d.settlement_method === "wallet" ? C.methodWallet : C.methodCash}</Badge>
                      <Badge variant={days >= 3 ? "danger" : days >= 1 ? "warning" : "neutral"}>{sinceLabel(d.oldest_at)}</Badge>
                    </span>
                  </span>
                </div>
                <div className="flex items-end justify-between gap-2 rounded-control bg-field px-3 py-2.5">
                  <span>
                    <span className="block text-2xs text-ink-muted">{C.storeOutstanding}</span>
                    <span className="figure block text-warning">{fmtMoney(d.outstanding)}</span>
                  </span>
                  <span className="text-end text-sm">
                    <span className="block text-2xs text-ink-muted">{C.storeCount}</span>
                    <b className="tabular-nums">{fmtNum(d.count)}</b>
                  </span>
                </div>
                {canPayView && (
                  <Button className="w-full" onClick={() => setPaying(d)}>
                    {C.storePayAmount.replace("{n}", fmtMoney(d.outstanding))}
                  </Button>
                )}
              </li>
            );
          })}
        </ul>
      )}
      {paying && (
        <Modal open size="lg" onClose={() => setPaying(null)} title={C.storePayTitle.replace("{name}", paying.name)}>
          <MerchantSettlement merchantId={paying.merchant_id} method={paying.settlement_method} onChanged={reload} />
        </Modal>
      )}
    </>
  );
}
