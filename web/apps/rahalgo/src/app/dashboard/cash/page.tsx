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
  DataView,
  EmptyState,
  Input,
  LoadingState,
  Modal,
  Money,
  PageContainer,
  ReloadState,
  StatCard,
  StatGrid,
  useLiveData,
  type DataColumn,
  IconArrowOut,
  IconDate,
  IconStatus,
  IconStore,
  IconUser,
  IconWallet,
  IconWarning,
} from "@rahalgo/ui";
import { api, apiFile } from "@/lib/api";
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
type Filter = "all" | "over" | "overdue" | "shift";

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
      : f === "overdue" || f === "shift"
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
 * ما معه للمكتب، وحقُّه بمحفظته، والصافي بجملة، وسقفُه، وزرُّ الاستلام، **وكشفُ نقده حركةً حركة.**
 */
interface DriverRow {
  id: string;
  full_name: string;
  phone: string;
  balance?: number;
  on_shift?: boolean;
}

interface CashEntry {
  id: number;
  amount: number;
  kind: string;
  order_number: number | null;
  note: string;
  created_at: string;
}

function DriversCash({ filter, setFilter }: { filter: Filter; setFilter: (f: Filter) => void }) {
  const { can } = useAuth();
  const canSettle = can("finance.manage");
  const canExport = can("finance.export");
  const canUser = useCanOpen().user;
  const [target, setTarget] = useState<Holder | null>(null);
  const [picked, setPicked] = useState("");
  const [query, setQuery] = useState("");
  const [done, setDone] = useState("");
  const [exportError, setExportError] = useState("");

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
        wallet: u.balance ?? 0,
      });
    }
    const q = query.trim().toLowerCase();
    return rows.filter((h) => {
      if (filter === "over" && !h.over_limit) return false;
      if (filter === "overdue" && !h.overdue) return false;
      if (filter === "shift" && !h.on_shift) return false;
      if (!q) return true;
      return (h.name || "").toLowerCase().includes(q) || h.phone.includes(q);
    });
  }, [holders, drivers.data, byID, data?.limit, filter, query]);

  useEffect(() => {
    if (!picked && list[0]) setPicked(list[0].driver_id);
  }, [list, picked]);

  if (!data && error) return <ReloadState label={C.loadFailed} onRetry={reload} />;
  if (!data) return <LoadingState />;

  const oldest = holders.reduce((mx, h) => Math.max(mx, daysHeld(h.oldest_at)), 0);
  const sel = list.find((h) => h.driver_id === picked) ?? null;

  function download() {
    setExportError("");
    void (async () => {
      try {
        const res = await apiFile("/api/v1/admin/cash/outstanding/export");
        const url = URL.createObjectURL(await res.blob());
        const a = document.createElement("a");
        a.href = url;
        a.download = "driver-cash.csv";
        a.click();
        URL.revokeObjectURL(url);
      } catch (err) {
        setExportError(errorText(err));
      }
    })();
  }

  return (
    <>
      <div className="mb-4 grid grid-cols-2 gap-2 lg:grid-cols-4">
        <Tile icon={IconWallet} label={C.total} value={fmtMoney(data.total)} tone={data.total > 0 ? "warning" : "muted"} onClick={() => setFilter("all")} />
        <Tile icon={IconUser} label={C.holders} value={fmtNum(holders.length)} onClick={() => setFilter("all")} />
        <Tile icon={IconWarning} label={C.overCount} value={fmtNum(data.over_count)} tone={data.over_count > 0 ? "danger" : "muted"} onClick={() => setFilter(filter === "over" ? "all" : "over")} />
        <Tile icon={IconDate} label={C.oldestAge} value={holders.length ? daysLabel(oldest) : "—"} tone={data.overdue_count > 0 ? "danger" : "default"} onClick={() => setFilter(filter === "overdue" ? "all" : "overdue")} />
      </div>

      {done && (
        <Alert tone="success" className="mb-3" onDismiss={() => setDone("")}>
          {done}
        </Alert>
      )}
      {exportError && <Alert className="mb-3">{exportError}</Alert>}

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
                { id: "shift" as Filter, label: C.filterOnShift },
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
          {canExport && (
            <div className="border-t border-line-soft p-2">
              <Button variant="ghost" className="w-full" onClick={download}>
                <span className="flex items-center justify-center gap-1.5">
                  <IconArrowOut size={14} />
                  {C.export}
                </span>
              </Button>
            </div>
          )}
        </section>

        {/* ── السائقُ المختار ── */}
        {sel ? (
          <DriverPanel
            h={sel}
            canUser={canUser}
            canSettle={canSettle}
            onReceive={() => setTarget(sel)}
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

/** **لوحةُ السائق المختار** — الأرقامُ الثلاثة والصافي، والسقف، والاستلام، وكشفُ نقده. */
function DriverPanel({
  h,
  canUser,
  canSettle,
  onReceive,
}: {
  h: Holder;
  canUser: boolean;
  canSettle: boolean;
  onReceive: () => void;
}) {
  const st = useLiveData<{ held: number; limit: number; entries: CashEntry[]; open_cash: number }>(
    () => api(`/api/v1/admin/drivers/${h.driver_id}/cash`),
    ["wallet", "order"],
    [h.driver_id],
  );
  const KIND = m.admin.users.profile.roleTabs.cashKinds as Record<string, string>;
  const held = st.data?.held ?? h.held;
  const net = held - h.wallet;
  const limit = st.data?.limit ?? h.limit;
  const exposure = held + (st.data?.open_cash ?? h.open_cash);
  const pct = limit > 0 ? Math.min(100, Math.round((exposure / limit) * 100)) : 100;
  const d = daysHeld(h.oldest_at);
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
          {held > 0 && <Badge variant={h.overdue ? "danger" : d >= 1 ? "warning" : "neutral"}>{daysLabel(d)}</Badge>}
        </span>
      </div>

      <div className="grid grid-cols-1 gap-2 sm:grid-cols-3">
        <div className="rounded-card border border-line-soft p-4">
          <span className="block text-sm text-ink-muted">{C.cardHeld}</span>
          <span className="figure block text-warning">{fmtMoney(held)}</span>
        </div>
        <div className="rounded-card border border-line-soft p-4">
          <span className="block text-sm text-ink-muted">{C.cardWallet}</span>
          <span className="figure block text-success">{fmtMoney(h.wallet)}</span>
        </div>
        <div
          className={`rounded-card p-4 ${
            net > 0 ? "bg-warning-solid text-on-solid" : net < 0 ? "bg-success-solid text-on-solid" : "border border-line-soft"
          }`}
        >
          <span className="block text-sm opacity-90">{C.cardNet}</span>
          <span className="figure block">{fmtMoney(Math.abs(net))}</span>
          <span className="block text-xs font-bold">{net > 0 ? C.netDriverPaysShort : net < 0 ? C.netOfficePaysShort : C.netEven}</span>
        </div>
      </div>

      <div className="space-y-1">
        <div className="flex items-baseline justify-between text-xs text-ink-muted">
          <span>{C.exposure}</span>
          <span className="tabular-nums" dir="ltr">
            {C.capOf.replace("{t}", fmtNum(exposure)).replace("{cap}", fmtNum(limit))}
          </span>
        </div>
        <span className="block h-2 w-full overflow-hidden rounded-full bg-field" aria-hidden>
          <span
            className={`block h-full rounded-full ${exposure > limit ? "bg-danger" : pct >= 80 ? "bg-warning" : "bg-success"}`}
            style={{ width: `${pct}%` }}
          />
        </span>
      </div>

      <div className="flex flex-wrap items-center justify-between gap-2 border-t border-line-soft pt-3">
        <span className="text-xs text-ink-muted">
          {C.lastSettled}: {h.last_settled_at ? <span dir="ltr">{fmtDate(h.last_settled_at)}</span> : C.never}
        </span>
        {canSettle && held > 0 && <Button onClick={onReceive}>{C.receive}</Button>}
      </div>

      <div className="space-y-2">
        <h3 className="heading-card">{C.statementTitle}</h3>
        {!st.data ? (
          <LoadingState variant="text" />
        ) : st.data.entries.length === 0 ? (
          <p className="text-sm text-ink-muted">{C.statementEmpty}</p>
        ) : (
          <ul className="space-y-1.5">
            {st.data.entries.map((e) => {
              const inn = e.amount >= 0;
              return (
                <li key={e.id} className="flex items-center gap-3 rounded-control bg-field px-3 py-2 text-sm">
                  <span className={`text-xs ${inn ? "text-warning" : "text-success"}`}>{inn ? "▲" : "▼"}</span>
                  <span className="min-w-0 flex-1">
                    <span className="font-medium">{KIND[e.kind] ?? e.kind}</span>
                    {e.order_number != null && <span className="text-ink-muted"> · #{e.order_number}</span>}
                    {e.note && <span className="text-ink-muted"> · {e.note}</span>}
                    <span className="block text-2xs text-ink-muted" dir="ltr">{fmtDateTime(e.created_at)}</span>
                  </span>
                  <span className={`shrink-0 font-bold tabular-nums ${inn ? "text-warning" : "text-success"}`} dir="ltr">
                    {inn ? "+" : "−"}
                    {fmtNum(Math.abs(e.amount))}
                  </span>
                </li>
              );
            })}
          </ul>
        )}
      </div>
    </section>
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

  const columns: DataColumn<StoreDue>[] = [
    {
      id: "name",
      header: C.storeName,
      icon: <IconStore />,
      cell: (d) => (
        <OpenLink allowed={canMerchant} href={`/dashboard/merchants/${d.merchant_id}`} className="font-medium hover:underline">
          {d.name}
        </OpenLink>
      ),
    },
    { id: "due", header: C.storeOutstanding, cell: (d) => <Money value={d.outstanding} small /> },
    { id: "count", header: C.storeCount, cell: (d) => <span dir="ltr">{fmtNum(d.count)}</span> },
    {
      id: "since",
      header: C.storeSince,
      icon: <IconDate />,
      cell: (d) => <Badge variant={daysHeld(d.oldest_at) >= 3 ? "warning" : "neutral"}>{daysLabel(daysHeld(d.oldest_at))}</Badge>,
    },
    {
      id: "method",
      header: C.storeMethod,
      cell: (d) => (
        <Badge variant="neutral">{d.settlement_method === "wallet" ? C.methodWallet : C.methodCash}</Badge>
      ),
    },
    ...(canPayView
      ? [
          {
            id: "pay",
            header: C.storePay,
            cell: (d: StoreDue) => (
              <Button variant="secondary" onClick={() => setPaying(d)}>
                {C.storePay}
              </Button>
            ),
          },
        ]
      : []),
  ];

  return (
    <>
      <StatGrid>
        <StatCard label={C.storesTotal} value={fmtMoney(data.total)} icon={IconStore} />
      </StatGrid>
      <p className="mb-3 text-sm text-ink-muted">{C.storesHint}</p>
      {rows.length === 0 ? (
        <EmptyState icon={IconStatus} title={C.storesEmpty} />
      ) : (
        <DataView items={rows} getKey={(d) => d.merchant_id} columns={columns} view="cards" empty={C.storesEmpty} />
      )}
      {paying && (
        <Modal open size="lg" onClose={() => setPaying(null)} title={C.storePayTitle.replace("{name}", paying.name)}>
          <MerchantSettlement merchantId={paying.merchant_id} method={paying.settlement_method} onChanged={reload} />
        </Modal>
      )}
    </>
  );
}
