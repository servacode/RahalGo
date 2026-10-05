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
 */

import { useEffect, useMemo, useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtMoney, fmtDate, errorText } from "@rahalgo/i18n";
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
  PageHeader,
  ReloadState,
  StatCard,
  StatGrid,
  Tabs,
  ViewToggle,
  useLiveData,
  useViewMode,
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
  oldest_at: string | null;
  overdue: boolean;
  last_settled_at: string | null;
  on_shift: boolean;
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
      <PageHeader icon={IconWallet} title={C.tabTitle} subtitle={C.tabHint} />
      <Tabs
        className="mb-4"
        items={[
          { key: "drivers" as Tab, label: C.tabDrivers, icon: IconUser },
          { key: "stores" as Tab, label: C.tabStores, icon: IconStore },
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

function DriversCash({ filter, setFilter }: { filter: Filter; setFilter: (f: Filter) => void }) {
  const { can } = useAuth();
  // **وتسويةُ نقدِ السائق قيدٌ ماليّ** — بقدرة بابها لا باسم دور.
  const canSettle = can("finance.manage");
  const canExport = can("finance.export");
  // **وملفُّ السائق لمن يفتح الحسابات** (قرارُ المالك ٢٠٢٦-١٠-٠٥) — والتسويةُ هنا.
  const canUser = useCanOpen().user;
  const [view, setView] = useViewMode("cash-outstanding");
  const [target, setTarget] = useState<Holder | null>(null);
  const [query, setQuery] = useState("");
  const [done, setDone] = useState("");
  const [exportError, setExportError] = useState("");

  const { data, error, reload } = useLiveData<CashData>(
    () => api("/api/v1/admin/cash/outstanding"),
    ["wallet", "order"],
  );

  const all = useMemo(() => data?.holders ?? [], [data]);
  const holders = useMemo(() => {
    const q = query.trim().toLowerCase();
    return all.filter((h) => {
      if (filter === "over" && !h.over_limit) return false;
      if (filter === "overdue" && !h.overdue) return false;
      if (filter === "shift" && !h.on_shift) return false;
      if (!q) return true;
      return h.name.toLowerCase().includes(q) || h.phone.includes(q);
    });
  }, [all, filter, query]);

  // **الخطأُ يُعرض ولا يبقى «جارٍ التحميل» للأبد** (المشكلة ٨).
  if (!data && error) return <ReloadState label={C.loadFailed} onRetry={reload} />;
  if (!data) return <LoadingState />;

  const oldest = all.reduce((mx, h) => Math.max(mx, daysHeld(h.oldest_at)), 0);

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

  const columns: DataColumn<Holder>[] = [
    {
      id: "driver",
      header: m.terms.driver,
      icon: <IconUser />,
      cell: (h) => (
        <OpenLink allowed={canUser} href={`/dashboard/users/${h.driver_id}`} className="flex flex-col hover:underline">
          <span className="font-medium">{h.name || h.phone}</span>
          <span className="text-2xs text-ink-muted" dir="ltr">
            {h.phone}
          </span>
        </OpenLink>
      ),
    },
    {
      id: "held",
      header: C.held,
      icon: <IconWallet />,
      cell: (h) => <Money value={h.held} small />,
    },
    {
      id: "open",
      header: C.openCash,
      cell: (h) => (h.open_cash > 0 ? <Money value={h.open_cash} small /> : <span className="text-ink-muted">—</span>),
    },
    {
      id: "cap",
      header: C.exposure,
      cell: (h) => {
        const pct = h.limit > 0 ? Math.min(100, Math.round((h.exposure / h.limit) * 100)) : 100;
        return (
          <span className="flex min-w-32 flex-col gap-1">
            <span dir="ltr" className={`text-xs tabular-nums ${h.over_limit ? "text-danger" : "text-ink-muted"}`}>
              {C.capOf.replace("{t}", fmtNum(h.exposure)).replace("{cap}", fmtNum(h.limit))}
            </span>
            <span className="h-1.5 w-full overflow-hidden rounded-full bg-field" aria-hidden>
              <span
                className={`block h-full rounded-full ${h.over_limit ? "bg-danger" : pct >= 80 ? "bg-warning" : "bg-success"}`}
                style={{ width: `${pct}%` }}
              />
            </span>
          </span>
        );
      },
    },
    {
      id: "age",
      header: C.age,
      icon: <IconDate />,
      // **القِدَمُ يُلوَّن لا المقدار**، وحدُّ الأحمر هو حدُّ التنبيه في الإعدادات.
      cell: (h) => {
        const d = daysHeld(h.oldest_at);
        return (
          <Badge variant={h.overdue ? "danger" : d >= 1 ? "warning" : "neutral"}>{daysLabel(d)}</Badge>
        );
      },
    },
    {
      id: "last",
      header: C.lastSettled,
      cell: (h) =>
        h.last_settled_at ? (
          <span dir="ltr" className="text-xs text-ink-muted">
            {fmtDate(h.last_settled_at)}
          </span>
        ) : (
          <span className="text-xs text-ink-muted">{C.never}</span>
        ),
    },
    {
      id: "shift",
      header: C.shift,
      cell: (h) => (
        <Badge variant={h.on_shift ? "success" : "neutral"}>{h.on_shift ? C.onShift : C.offShift}</Badge>
      ),
    },
    {
      id: "act",
      header: "",
      cell: (h) =>
        canSettle ? (
          <Button variant="secondary" onClick={() => setTarget(h)}>
            {C.receive}
          </Button>
        ) : null,
    },
  ];

  return (
    <>
      <StatGrid>
        <StatCard
          label={C.total}
          value={fmtMoney(data.total)}
          icon={IconWallet}
          tone={data.total > 0 ? "danger" : "default"}
        />
        <StatCard
          label={C.holders}
          value={fmtNum(all.length)}
          icon={IconUser}
          selected={filter === "all"}
          onClick={() => setFilter("all")}
        />
        <StatCard
          label={C.overCount}
          value={fmtNum(data.over_count)}
          icon={IconWarning}
          tone={data.over_count > 0 ? "danger" : "muted"}
          selected={filter === "over"}
          onClick={() => setFilter(filter === "over" ? "all" : "over")}
        />
        <StatCard
          label={C.oldestAge}
          value={all.length ? daysLabel(oldest) : "—"}
          icon={IconDate}
          tone={data.overdue_count > 0 ? "danger" : "default"}
          selected={filter === "overdue"}
          onClick={() => setFilter(filter === "overdue" ? "all" : "overdue")}
        />
      </StatGrid>

      {done && (
        <Alert tone="success" className="mb-3">
          {done}
        </Alert>
      )}
      {exportError && <Alert className="mb-3">{exportError}</Alert>}

      <div className="mb-3 flex flex-wrap items-end gap-3">
        <Chips
          items={[
            { id: "all" as Filter, label: C.filterAll, count: all.length },
            { id: "over" as Filter, label: C.filterOver, count: data.over_count },
            {
              id: "overdue" as Filter,
              label: C.overdueCount.replace("{n}", fmtNum(data.overdue_days)),
              count: data.overdue_count,
            },
            { id: "shift" as Filter, label: C.filterOnShift, count: all.filter((h) => h.on_shift).length },
          ]}
          value={filter}
          onChange={setFilter}
        />
        <div className="min-w-48 flex-1">
          <Input
            id="cash-search"
            label={C.search}
            value={query}
            onChange={(e) => setQuery(e.target.value)}
          />
        </div>
        {canExport && (
          <Button variant="secondary" onClick={download}>
            <span className="flex items-center gap-1.5">
              <IconArrowOut size={14} />
              {C.export}
            </span>
          </Button>
        )}
        <ViewToggle
          view={view}
          onChange={setView}
          tableLabel={m.common.viewTable}
          cardsLabel={m.common.viewCards}
        />
      </div>

      {all.length === 0 ? (
        <EmptyState icon={IconStatus} title={C.empty} />
      ) : holders.length === 0 ? (
        <EmptyState icon={IconStatus} title={C.noMatch} />
      ) : (
        <DataView items={holders} getKey={(h) => h.driver_id} columns={columns} view={view} empty={C.noMatch} />
      )}

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
        <DataView items={rows} getKey={(d) => d.merchant_id} columns={columns} view="table" empty={C.storesEmpty} />
      )}
      {paying && (
        <Modal open size="lg" onClose={() => setPaying(null)} title={C.storePayTitle.replace("{name}", paying.name)}>
          <MerchantSettlement merchantId={paying.merchant_id} method={paying.settlement_method} onChanged={reload} />
        </Modal>
      )}
    </>
  );
}
