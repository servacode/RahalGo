"use client";

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtNum, errorText, fmtMoney, damascusDay } from "@rahalgo/i18n";
import {
  Alert,
  PageHeader,
  Input,
  Button,
  Select,
  IconOrder,
  IconWallet,
  IconStore,
  IconDriver,
  IconStatus,
  IconDate,
  Badge,
  IconTile,
  IconUsers,
  Money,
  StatCard,
  StatGrid,
  SkeletonStats,
  SkeletonList,
} from "@rahalgo/ui";
import { api, apiFile } from "@/lib/api";
import { useAuth } from "@/lib/auth";

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

/** **نسبةُ التغيّر عن الفترة السابقة** — نصٌّ جاهز، أو لا شيء. */
function change(cur: number | undefined, prev: number | undefined): string | undefined {
  const r = m.admin.reports;
  if (cur === undefined || prev === undefined) return undefined;
  if (prev === 0) return cur === 0 ? undefined : r.previousNone;
  const pct = Math.round(((cur - prev) / Math.abs(prev)) * 100);
  if (pct === 0) return r.previousSame;
  const d = `${pct > 0 ? "+" : "−"}${r.pct.replace("{n}", fmtNum(Math.abs(pct)))}`;
  return r.vsPrevious.replace("{d}", d);
}

/** نسبةٌ من الطلبات. */
function share(part: number, total: number): string | undefined {
  if (total <= 0) return undefined;
  return m.admin.reports.summary.ofOrders.replace("{p}", fmtNum(Math.round((part / total) * 100)));
}

/**
 * مخطط أعمدة يومي — سلسلةٌ واحدةٌ بلونٍ واحد (العنوانُ يسمّيها)، وتلميحٌ لكلّ عمود.
 */
function DailyBars({
  title,
  data,
  value,
  format,
}: {
  title: string;
  data: DailyPoint[];
  value: (d: DailyPoint) => number;
  format: (v: number) => string;
}) {
  const [hover, setHover] = useState<number | null>(null);
  const max = Math.max(1, ...data.map(value));

  return (
    <section className="surface p-5">
      <h2 className="mb-4 text-sm font-bold">{title}</h2>
      <div className="relative">
        <div className="pointer-events-none absolute inset-x-0 bottom-6 top-0">
          {[0.25, 0.5, 0.75].map((f) => (
            <div
              key={f}
              className="absolute inset-x-0 border-t border-line-soft"
              style={{ bottom: `${f * 100}%` }}
            />
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
    </section>
  );
}

function Stat({
  icon: Icon,
  label,
  value,
  sub,
  note,
}: {
  icon: React.ComponentType<{ size?: number; className?: string }>;
  label: string;
  value: React.ReactNode;
  sub?: string;
  /** سطرٌ ثانٍ — مقارنةٌ أو تنبيه. */
  note?: string;
}) {
  return (
    <div className="flex items-center gap-3 surface p-4">
      <IconTile>
        <Icon size={18} className="text-primary-dark" />
      </IconTile>
      <div className="min-w-0">
        <p className="figure leading-tight">{value}</p>
        <p className="text-xs text-ink-muted">
          {label}
          {sub && ` · ${sub}`}
        </p>
        {note && <p className="text-2xs text-ink-muted">{note}</p>}
      </div>
    </div>
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
    <div>
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <PageHeader icon={IconStatus} title={r.title} />
        {canExport && (
          <div className="flex flex-wrap gap-2">
            <Button variant="secondary" onClick={() => download("orders")} disabled={inverted}>
              {r.exportOrders}
            </Button>
            <Button variant="secondary" onClick={() => download("ledger")} disabled={inverted}>
              {r.exportLedger}
            </Button>
          </div>
        )}
      </div>

      {/* **شريطُ المدّة والفلاتر** — يلتفّ على الجوال ولا يُزيح الصفحة. */}
      <div className="mb-4 flex flex-wrap items-end gap-2">
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
              }}
            >
              {x.label}
            </Button>
          );
        })}
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
        <p className="w-full text-2xs text-ink-muted">{r.byOrderDay}</p>
      </div>

      {error && (
        <Alert className="mb-4">
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
        <div className={loading ? "opacity-60 transition-opacity" : "transition-opacity"}>
          {/* ── أرقامُ الطلبات — لكلّ من يفتح الصفحة ── */}
          <div className="mb-4 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
            <Stat
              icon={IconOrder}
              label={r.summary.ordersTotal}
              value={fmtNum(s.orders_total)}
              note={change(s.orders_total, p.orders_total)}
            />
            <Stat
              icon={IconOrder}
              label={r.summary.delivered}
              value={fmtNum(s.delivered)}
              sub={share(s.delivered, s.orders_total)}
              note={change(s.delivered, p.delivered)}
            />
            <Stat
              icon={IconOrder}
              label={r.summary.cancelled}
              value={fmtNum(s.cancelled)}
              sub={share(s.cancelled, s.orders_total)}
              note={change(s.cancelled, p.cancelled)}
            />
            <Stat
              icon={IconOrder}
              label={r.summary.refunded}
              value={fmtNum(s.refunded)}
              sub={share(s.refunded, s.orders_total)}
              note={change(s.refunded, p.refunded)}
            />
            <Stat
              icon={IconStatus}
              label={r.summary.avgOrderToDelivery}
              value={`${fmtNum(s.avg_delivery_min)} ${r.summary.minutes}`}
            />
            <Stat
              icon={IconUsers}
              label={r.summary.activeCustomers}
              value={fmtNum(s.active_customers)}
              note={change(s.active_customers, p.active_customers)}
            />
          </div>

          {/* ── المال — يصل من الخادم لمن يقرؤه وحدَه ── */}
          {money ? (
            <div className="mb-6 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-3">
              <Stat
                icon={IconWallet}
                label={r.summary.grossSales}
                value={<Money value={s.gross_sales ?? 0} />}
                note={change(s.gross_sales, p.gross_sales)}
              />
              <Stat
                icon={IconWallet}
                label={r.summary.platformProfit}
                value={<Money value={s.platform_profit ?? 0} />}
                sub={report.profit_unfiltered ? r.profitAllPlatform : r.profitHint}
                note={change(s.platform_profit, p.platform_profit)}
              />
              <Stat
                icon={IconWallet}
                label={r.summary.storeCommission}
                value={<Money value={s.commissions ?? 0} />}
                note={change(s.commissions, p.commissions)}
              />
              <Stat
                icon={IconWallet}
                label={r.summary.deliveryFees}
                value={<Money value={s.delivery_fees ?? 0} />}
                note={change(s.delivery_fees, p.delivery_fees)}
              />
              <Stat
                icon={IconWallet}
                label={r.summary.walletPaid}
                value={<Money value={s.wallet_paid ?? 0} />}
                note={change(s.wallet_paid, p.wallet_paid)}
              />
              <Stat
                icon={IconWallet}
                label={r.summary.cashCollected}
                value={<Money value={s.cash_collected ?? 0} />}
                note={change(s.cash_collected, p.cash_collected)}
              />
            </div>
          ) : (
            <p className="mb-6 text-xs text-ink-muted">{r.moneyHidden}</p>
          )}

          <div className="mb-6 grid grid-cols-1 gap-4 lg:grid-cols-2">
            <DailyBars
              title={r.dailyOrders}
              data={report.daily}
              value={(d) => d.orders}
              format={(v) => fmtNum(v)}
            />
            {money && (
              <DailyBars
                title={r.dailySales}
                data={report.daily}
                value={(d) => d.sales ?? 0}
                format={(v) => fmtMoney(v)}
              />
            )}
          </div>

          <div className="mb-6 grid grid-cols-1 gap-4 lg:grid-cols-2">
            <section className="surface p-5">
              <h2 className="mb-1 flex items-center gap-2 text-sm font-bold">
                <IconStore size={16} className="text-primary" />
                {r.topMerchants}
              </h2>
              {money && <p className="mb-3 text-2xs text-ink-muted">{r.storeSalesHint}</p>}
              {report.top_merchants.length === 0 ? (
                <p className="py-4 text-center text-sm text-ink-muted">{r.noData}</p>
              ) : (
                <ol className="space-y-2">
                  {report.top_merchants.map((t, i) => (
                    <li key={t.id} className="flex items-center justify-between gap-2 text-sm">
                      <span className="min-w-0">
                        <Badge variant="primary" className="me-2 h-5 w-5 justify-center px-0 font-bold">
                          {fmtNum(i + 1)}
                        </Badge>
                        {t.name}
                        <span className="text-xs text-ink-muted">
                          {" "}
                          · {fmtNum(t.delivered)} {r.deliveries}
                        </span>
                      </span>
                      {t.sales !== undefined && (
                        <span className="shrink-0 font-bold text-primary-dark">
                          <Money value={t.sales} />
                        </span>
                      )}
                    </li>
                  ))}
                </ol>
              )}
            </section>

            <section className="surface p-5">
              <h2 className="mb-3 flex items-center gap-2 text-sm font-bold">
                <IconDriver size={16} className="text-primary" />
                {r.topDrivers}
              </h2>
              {report.top_drivers.length === 0 ? (
                <p className="py-4 text-center text-sm text-ink-muted">{r.noData}</p>
              ) : (
                <ol className="space-y-2">
                  {report.top_drivers.map((t, i) => (
                    <li key={t.id} className="flex items-center justify-between gap-2 text-sm">
                      <span className="min-w-0">
                        <Badge variant="primary" className="me-2 h-5 w-5 justify-center px-0 font-bold">
                          {fmtNum(i + 1)}
                        </Badge>
                        {t.name || (t.phone ? <span dir="ltr">{t.phone}</span> : r.noName)}
                      </span>
                      <span className="shrink-0">
                        <span className="font-bold">{fmtNum(t.delivered)}</span>{" "}
                        <span className="text-xs text-ink-muted">{r.deliveries}</span>
                        {t.cash !== undefined && (
                          <span className="text-xs text-ink-muted">
                            {" · "}
                            {r.summary.cashCollected} <Money value={t.cash} small />
                          </span>
                        )}
                      </span>
                    </li>
                  ))}
                </ol>
              )}
            </section>
          </div>

          {/* **الزوّار** — أجهزةُ تطبيق الزبون وحدَها، ولا يتبعون المدّة. */}
          <section className="mb-6">
            <h2 className="mb-1 text-sm font-bold">{m.admin.dash.visitors}</h2>
            <p className="mb-2 text-2xs text-ink-muted">{r.visitorsHint}</p>
            {report.visitors ? (
              <StatGrid>
                <StatCard icon={IconUsers} label={m.admin.dash.devicesToday} value={fmtNum(report.visitors.devices_today)} />
                <StatCard icon={IconUsers} label={m.admin.dash.devices7} value={fmtNum(report.visitors.devices_7d)} />
                <StatCard icon={IconUsers} label={m.admin.dash.devices30} value={fmtNum(report.visitors.devices_30d)} />
                <StatCard icon={IconOrder} label={m.admin.dash.opensToday} value={fmtNum(report.visitors.opens_today)} />
                <StatCard icon={IconOrder} label={m.admin.dash.opens7} value={fmtNum(report.visitors.opens_7d)} />
                <StatCard icon={IconOrder} label={m.admin.dash.opens30} value={fmtNum(report.visitors.opens_30d)} />
              </StatGrid>
            ) : (
              <Alert tone="warning">{r.visitorsFailed}</Alert>
            )}
          </section>
        </div>
      )}
    </div>
  );
}
