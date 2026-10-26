"use client";

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtNum, errorText, fmtMoney, damascusDay } from "@rahalgo/i18n";
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
  IconStatus,
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
  Money,
  SkeletonStats,
  SkeletonList,
} from "@rahalgo/ui";
import { api, apiFile } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { PageHero } from "@/components/admin/PageHero";
import { Panel, type TileTone } from "@/components/admin/DashParts";

const m = getMessages(defaultLocale);

/**
 * ══════════════════════════════════════════════════════════════════════
 * **التقارير — قرارُ المالك ٢٠٢٦-١٠-٠٤**
 * ══════════════════════════════════════════════════════════════════════
 *
 * - **اليومُ يومُ دمشق** — في الأزرار والافتراض والمخطّط (والمحرّكُ كذلك).
 * - **أرقامُ المال يحذفها الخادمُ** عمّن ليس من الماليّة أو مدير المنصّة؛
 *   والصفحةُ تعرض ما وصلها فقط (`money_visible`).
 * - **«ربح المنصّة» رقمُ صفحة الأرباح نفسُه**، و«عمولة المتاجر» بطاقةٌ لحالها.
 * - **المستردُّ بطاقةٌ لحالها**، **ومقارنةٌ بالفترة السابقة المساوية.**
 * - **الزوّارُ أجهزةُ الزبائن وحدَها** — وعطبُهم يُقال لا يُخفى.
 *
 * **تصميمٌ احترافيّ (المالك ٢٠٢٦-١٠-٢٦)** — رأسٌ موحّد، والفلاترُ ببطاقةٍ واحدة، والأرقامُ
 * بلوحتين («الطلبات» و«المال») ومقارنةٌ ملوّنةٌ قصيرة، والمخطّطاتُ والأفضلُ بلوحات — بلا جداول.
 * **العرضُ وحدَه تغيّر** — النداءاتُ والحسابُ والتصديرُ والصلاحيّاتُ كما كانت.
 * **ويومُ الحساب يبدأ بساعة فتح الدوام** (`platform.day_start_hour`) — والخادمُ يطبّقه.
 */

interface DailyPoint {
  day: string;
  orders: number;
  delivered: number;
  sales?: number;
}

interface Summary {
  orders_total: number;
  delivered: number;
  cancelled: number;
  refunded: number;
  avg_delivery_min: number;
  active_customers: number;
  gross_sales?: number;
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

const DAY = 86_400_000;

/** **الشهرُ الماضي بيوم دمشق** — من أوّله إلى آخره. */
function lastMonth(today: string): { from: string; to: string } {
  const y = Number(today.slice(0, 4));
  const mo = Number(today.slice(5, 7));
  const py = mo === 1 ? y - 1 : y;
  const pm = mo === 1 ? 12 : mo - 1;
  const last = new Date(Date.UTC(py, pm, 0)).getUTCDate();
  const mm = String(pm).padStart(2, "0");
  return { from: `${py}-${mm}-01`, to: `${py}-${mm}-${String(last).padStart(2, "0")}` };
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
 * ملوّناً: أخضرُ للأحسن وأحمرُ للأسوأ — **والأحسنُ في الملغى والمستردّ النزول** (`lowerIsBetter`).
 */
function StatTile({
  icon: Icon,
  label,
  value,
  sub,
  delta,
  tone = "default",
  lowerIsBetter = false,
}: {
  icon: React.ComponentType<{ size?: number; className?: string }>;
  label: string;
  value: React.ReactNode;
  sub?: string;
  delta?: Delta;
  tone?: TileTone;
  lowerIsBetter?: boolean;
}) {
  const good = delta && ((delta.dir === "up" && !lowerIsBetter) || (delta.dir === "down" && lowerIsBetter));
  const bad = delta && ((delta.dir === "down" && !lowerIsBetter) || (delta.dir === "up" && lowerIsBetter));
  const TrendIcon = delta?.dir === "up" ? IconTrendUp : delta?.dir === "down" ? IconTrendDown : null;
  return (
    <div className="flex items-center gap-3 rounded-card border border-line-soft p-3">
      <span className={`inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full ${TONE_ICON[tone]}`}>
        <Icon size={18} />
      </span>
      <span className="min-w-0">
        <span className="figure block leading-none">{value}</span>
        <span className="mt-1 block text-xs text-ink-muted">{label}</span>
        {sub && <span className="block text-2xs text-ink-muted">{sub}</span>}
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
    </div>
  );
}

/**
 * مخطط أعمدة يومي — سلسلةٌ واحدةٌ بلونٍ واحد (عنوانُ اللوحة يسمّيها)، وتلميحٌ لكلّ عمود.
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
  const [hover, setHover] = useState<number | null>(null);
  const max = Math.max(1, ...data.map(value));

  return (
    <div className="relative">
      <div className="pointer-events-none absolute inset-x-0 bottom-6 top-0">
        {[0.25, 0.5, 0.75].map((f) => (
          <div key={f} className="absolute inset-x-0 border-t border-line-soft" style={{ bottom: `${f * 100}%` }} />
        ))}
      </div>

      <div className="flex h-40 items-end gap-0.5" dir="ltr">
        {data.map((d, i) => {
          const v = value(d);
          const h = Math.max(v > 0 ? 6 : 2, (v / max) * 100);
          return (
            <div
              key={d.day}
              className="group relative flex h-full flex-1 items-end justify-center"
              onMouseEnter={() => setHover(i)}
              onMouseLeave={() => setHover(null)}
            >
              <div
                className={`w-full max-w-6 rounded-t transition-colors ${
                  v > 0 ? (hover === i ? "bg-primary-dark" : "bg-primary") : "bg-line"
                }`}
                style={{ height: `${h}%`, borderRadius: "4px 4px 0 0" }}
              />
              {hover === i && (
                <div className="pointer-events-none absolute bottom-full mb-1 whitespace-nowrap surface-inset px-2.5 py-1.5 text-xs elev-2">
                  <span className="font-bold">{format(v)}</span>
                  <span className="text-ink-muted"> · {d.day.slice(5)}</span>
                </div>
              )}
            </div>
          );
        })}
      </div>
      <div className="mt-1 flex gap-0.5 border-t border-line-soft pt-1" dir="ltr">
        {data.map((d) => (
          <div key={d.day} className="flex-1 text-center text-2xs text-ink-muted">
            {d.day.slice(8)}
          </div>
        ))}
      </div>
    </div>
  );
}

/** **سطرُ ترتيبٍ في «الأفضل»** — رقمُ المرتبة بدائرة، والاسمُ وسطرُه، والرقمُ على الطرف. */
function RankRow({
  rank,
  name,
  sub,
  end,
}: {
  rank: number;
  name: React.ReactNode;
  sub?: React.ReactNode;
  end?: React.ReactNode;
}) {
  return (
    <li className="flex items-center gap-3 rounded-control px-2 py-2 transition-colors hover:bg-row-hover">
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
    </li>
  );
}

export default function ReportsPage() {
  const { can } = useAuth();
  const canExport = can("finance.export");
  const r = m.admin.reports;

  const [from, setFrom] = useState(() => damascusDay(Date.now() - 6 * DAY));
  const [to, setTo] = useState(() => damascusDay());
  const [kind, setKind] = useState("");
  const [city, setCity] = useState("");
  /** **والفلترُ من الرابط** — يُقرأ بعد التركيب فلا تختلف الصفحةُ بين الخادم والمتصفّح. */
  const [ready, setReady] = useState(false);
  const [cities, setCities] = useState<City[]>([]);
  const [report, setReport] = useState<Report | null>(null);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");

  useEffect(() => {
    const q = new URLSearchParams(window.location.search);
    const f = q.get("from");
    const t = q.get("to");
    if (f) setFrom(f);
    if (t) setTo(t);
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

  const today = damascusDay();
  const lm = lastMonth(today);
  const presets = [
    { id: "today", label: r.range.today, from: today, to: today },
    {
      id: "yesterday",
      label: r.range.yesterday,
      from: damascusDay(Date.now() - DAY),
      to: damascusDay(Date.now() - DAY),
    },
    { id: "last7", label: r.range.last7, from: damascusDay(Date.now() - 6 * DAY), to: today },
    { id: "month", label: r.range.month, from: `${today.slice(0, 8)}01`, to: today },
    { id: "lastMonth", label: r.range.lastMonth, from: lm.from, to: lm.to },
  ];
  const activePreset = presets.find((x) => from === x.from && to === x.to)?.id ?? "";

  const query = new URLSearchParams({ from, to });
  if (kind) query.set("kind", kind);
  if (city) query.set("city_id", city);
  const qs = query.toString();

  /** تنزيلُ CSV — **بالمدّة المعروضة نفسِها** (والنوعُ لتصدير الطلبات). */
  function download(which: "orders" | "ledger") {
    const path = which === "orders" ? "orders/export" : "ledger/export";
    const q = new URLSearchParams({ from, to });
    if (which === "orders" && kind) q.set("kind", kind);
    void (async () => {
      try {
        const res = await apiFile(`/api/v1/admin/${path}?${q.toString()}`);
        const blob = await res.blob();
        const url = URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = `${which}-${from}_${to}.csv`;
        a.click();
        URL.revokeObjectURL(url);
      } catch (err) {
        setError(errorText(err));
      }
    })();
  }

  const load = useCallback(async () => {
    setLoading(true);
    try {
      setReport(await api<Report>(`/api/v1/admin/reports?${qs}`));
      setError("");
    } catch (err) {
      setReport(null);
      setError(errorText(err));
    } finally {
      setLoading(false);
    }
  }, [qs]);

  useEffect(() => {
    if (!ready) return;
    try {
      window.history.replaceState(null, "", `${window.location.pathname}?${qs}`);
    } catch {
      /* الرابطُ راحةٌ لا شرط */
    }
    void load();
  }, [load, qs, ready]);

  const inverted = from !== "" && to !== "" && from > to;
  const s = report?.summary;
  const p = report?.previous;
  const money = report?.money_visible ?? false;

  return (
    <PageContainer>
      {/* **رأسٌ موحّد** — والتصديرُ لمن يملك `finance.export` وحدَه، كما كان. */}
      <PageHero
        icon={IconStatus}
        title={r.title}
        subtitle={r.subtitle}
        actions={
          canExport ? (
            <div className="flex flex-wrap gap-2">
              <Button variant="secondary" onClick={() => download("orders")} disabled={inverted}>
                {r.exportOrders}
              </Button>
              <Button variant="secondary" onClick={() => download("ledger")} disabled={inverted}>
                {r.exportLedger}
              </Button>
            </div>
          ) : undefined
        }
      />

      {/* **الفترةُ والفلاترُ ببطاقةٍ واحدة** — تلتفّ على الجوال ولا تُزيح الصفحة. */}
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
        <div className="flex flex-wrap items-end gap-2">
          <div className="min-w-[8.75rem] flex-1 sm:w-40 sm:flex-none">
            <Input
              id="r-from"
              label={r.from}
              icon={<IconDate />}
              type="date"
              value={from}
              onChange={(e) => setFrom(e.target.value)}
            />
          </div>
          <div className="min-w-[8.75rem] flex-1 sm:w-40 sm:flex-none">
            <Input
              id="r-to"
              label={r.to}
              icon={<IconDate />}
              type="date"
              value={to}
              onChange={(e) => setTo(e.target.value)}
            />
          </div>
          <div className="min-w-[9rem] flex-1 sm:w-44 sm:flex-none">
            <Select aria-label={r.kindAll} value={kind} onChange={(e) => setKind(e.target.value)}>
              <option value="">{r.kindAll}</option>
              <option value="standard">{r.kindStandard}</option>
              <option value="custom">{r.kindCustom}</option>
              <option value="merchant_delivery">{r.kindDelivery}</option>
            </Select>
          </div>
          {cities.length > 0 && (
            <div className="min-w-[9rem] flex-1 sm:w-44 sm:flex-none">
              <Select aria-label={r.cityAll} value={city} onChange={(e) => setCity(e.target.value)}>
                <option value="">{r.cityAll}</option>
                {cities.map((c) => (
                  <option key={c.id} value={c.id}>
                    {c.name}
                  </option>
                ))}
              </Select>
            </div>
          )}
        </div>
        {/* **ويومُ الحساب من ساعة فتح الدوام** — التقريرُ لا يحمل الساعة، فاللفظُ بلا رقم. */}
        <p className="flex items-center gap-1.5 text-2xs text-ink-muted">
          <IconHourglass size={12} />
          {r.byOrderDay}
        </p>
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
          {/* ── أرقامُ الطلبات — لكلّ من يفتح الصفحة ── */}
          <Panel title={r.secOrders}>
            <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
              <StatTile
                icon={IconOrder}
                tone="accent"
                label={r.summary.ordersTotal}
                value={fmtNum(s.orders_total)}
                delta={change(s.orders_total, p.orders_total)}
              />
              <StatTile
                icon={IconCheck}
                tone={s.delivered > 0 ? "success" : "muted"}
                label={r.summary.delivered}
                value={fmtNum(s.delivered)}
                sub={share(s.delivered, s.orders_total)}
                delta={change(s.delivered, p.delivered)}
              />
              <StatTile
                icon={IconBlock}
                tone={s.cancelled > 0 ? "danger" : "muted"}
                label={r.summary.cancelled}
                value={fmtNum(s.cancelled)}
                sub={share(s.cancelled, s.orders_total)}
                delta={change(s.cancelled, p.cancelled)}
                lowerIsBetter
              />
              <StatTile
                icon={IconSwap}
                tone={s.refunded > 0 ? "warning" : "muted"}
                label={r.summary.refunded}
                value={fmtNum(s.refunded)}
                sub={share(s.refunded, s.orders_total)}
                delta={change(s.refunded, p.refunded)}
                lowerIsBetter
              />
              <StatTile
                icon={IconHourglass}
                label={r.summary.avgOrderToDelivery}
                value={`${fmtNum(s.avg_delivery_min)} ${r.summary.minutes}`}
              />
              <StatTile
                icon={IconUsers}
                label={r.summary.activeCustomers}
                value={fmtNum(s.active_customers)}
                delta={change(s.active_customers, p.active_customers)}
              />
            </div>
          </Panel>

          {/* ── المال — يصل من الخادم لمن يقرؤه وحدَه ── */}
          {money ? (
            <Panel title={r.secMoney}>
              <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-3">
                <StatTile
                  icon={IconReceipt}
                  tone="accent"
                  label={r.summary.grossSales}
                  value={<Money value={s.gross_sales ?? 0} />}
                  delta={change(s.gross_sales, p.gross_sales)}
                />
                <StatTile
                  icon={IconWallet}
                  tone="success"
                  label={r.summary.platformProfit}
                  value={<Money value={s.platform_profit ?? 0} />}
                  sub={report.profit_unfiltered ? r.profitAllPlatform : r.profitHint}
                  delta={change(s.platform_profit, p.platform_profit)}
                />
                <StatTile
                  icon={IconStore}
                  label={r.summary.storeCommission}
                  value={<Money value={s.commissions ?? 0} />}
                  delta={change(s.commissions, p.commissions)}
                />
                <StatTile
                  icon={IconMoto}
                  label={r.summary.deliveryFees}
                  value={<Money value={s.delivery_fees ?? 0} />}
                  delta={change(s.delivery_fees, p.delivery_fees)}
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
                <DailyBars data={report.daily} value={(d) => d.sales ?? 0} format={(v) => fmtMoney(v)} />
              </Panel>
            )}
          </div>

          {/* ── الأفضلُ بالفترة — قوائمُ مرتّبةٌ لا جداول ── */}
          <div className="grid grid-cols-1 gap-4 lg:grid-cols-2">
            <Panel title={r.topMerchants} note={money ? r.storeSalesHint : undefined}>
              {report.top_merchants.length === 0 ? (
                <p className="flex flex-col items-center gap-2 py-6 text-center text-sm text-ink-muted">
                  <IconStore size={22} />
                  {r.noData}
                </p>
              ) : (
                <ol className="space-y-0.5">
                  {report.top_merchants.map((t, i) => (
                    <RankRow
                      key={t.id}
                      rank={i + 1}
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
                <p className="flex flex-col items-center gap-2 py-6 text-center text-sm text-ink-muted">
                  <IconDriver size={22} />
                  {r.noData}
                </p>
              ) : (
                <ol className="space-y-0.5">
                  {report.top_drivers.map((t, i) => (
                    <RankRow
                      key={t.id}
                      rank={i + 1}
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

          {/* **الزوّار** — أجهزةُ تطبيق الزبون وحدَها، ولا يتبعون المدّة. */}
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
