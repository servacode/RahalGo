"use client";

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  Input,
  IconOrder,
  IconWallet,
  IconStore,
  IconDriver,
  IconUser,
  IconStatus,
  IconDate,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";

const m = getMessages(defaultLocale);
const fmt = new Intl.NumberFormat("ar-SY");

interface DailyPoint {
  day: string;
  orders: number;
  delivered: number;
  sales: number;
}

interface Report {
  from: string;
  to: string;
  summary: {
    orders_total: number;
    delivered: number;
    cancelled: number;
    gross_sales: number;
    delivery_fees: number;
    commissions: number;
    wallet_paid: number;
    cash_collected: number;
    avg_delivery_min: number;
    active_customers: number;
  };
  daily: DailyPoint[];
  top_merchants: { name: string; delivered: number; sales: number }[];
  top_drivers: { name: string; phone: string; delivered: number; cash: number }[];
}

function errText(err: unknown): string {
  return err instanceof ApiError ? m.errors.internal : m.errors.internal;
}

function iso(d: Date): string {
  return d.toISOString().slice(0, 10);
}

/**
 * مخطط أعمدة يومي — وفق منهجية التصميم البياني:
 * سلسلة واحدة بلون واحد (العنوان يسميها — لا مفتاح)، أعمدة رفيعة بنهايات دائرية
 * مثبتة على خط الأساس، فجوة 2px، شبكة خافتة، وتلميح عند التحويم لكل عمود.
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
    <section className="rounded-card border border-line bg-surface p-5">
      <h2 className="mb-4 text-sm font-bold">{title}</h2>
      <div className="relative">
        {/* شبكة خافتة: خطا الربع والنصف والثلاثة أرباع */}
        <div className="pointer-events-none absolute inset-x-0 bottom-6 top-0">
          {[0.25, 0.5, 0.75].map((f) => (
            <div
              key={f}
              className="absolute inset-x-0 border-t border-line/60"
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
                {/* تلميح التحويم */}
                {hover === i && (
                  <div className="pointer-events-none absolute bottom-full mb-1 whitespace-nowrap rounded-control border border-line bg-surface px-2.5 py-1.5 text-xs shadow-md">
                    <span className="font-bold">{format(v)}</span>
                    <span className="text-ink-muted"> · {d.day.slice(5)}</span>
                  </div>
                )}
              </div>
            );
          })}
        </div>
        {/* محور الأيام */}
        <div className="mt-1 flex gap-0.5 border-t border-line pt-1" dir="ltr">
          {data.map((d) => (
            <div key={d.day} className="flex-1 text-center text-[10px] text-ink-muted">
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
}: {
  icon: React.ComponentType<{ size?: number; className?: string }>;
  label: string;
  value: string;
  sub?: string;
}) {
  return (
    <div className="flex items-center gap-3 rounded-card border border-line bg-surface p-4">
      <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-card bg-primary-light">
        <Icon size={18} className="text-primary-dark" />
      </div>
      <div className="min-w-0">
        <p className="text-lg font-bold leading-tight">{value}</p>
        <p className="truncate text-xs text-ink-muted">
          {label}
          {sub && ` · ${sub}`}
        </p>
      </div>
    </div>
  );
}

export default function ReportsPage() {
  const today = new Date();
  const weekAgo = new Date(today.getTime() - 6 * 86400000);
  const [from, setFrom] = useState(iso(weekAgo));
  const [to, setTo] = useState(iso(today));
  const [report, setReport] = useState<Report | null>(null);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    try {
      setReport(await api<Report>(`/api/v1/admin/reports?from=${from}&to=${to}`));
      setError("");
    } catch (err) {
      setError(errText(err));
    }
  }, [from, to]);

  useEffect(() => {
    void load();
  }, [load]);

  const s = report?.summary;
  const r = m.admin.reports;

  return (
    <div>
      <div className="mb-6 flex flex-wrap items-center justify-between gap-3">
        <h1 className="flex items-center gap-2 text-2xl font-bold">
          <IconStatus className="text-primary" />
          {r.title}
        </h1>
        <div className="flex items-end gap-2">
          <div className="w-40">
            <Input
              id="r-from"
              label={r.from}
              icon={<IconDate />}
              type="date"
              value={from}
              onChange={(e) => setFrom(e.target.value)}
            />
          </div>
          <div className="w-40">
            <Input
              id="r-to"
              label={r.to}
              icon={<IconDate />}
              type="date"
              value={to}
              onChange={(e) => setTo(e.target.value)}
            />
          </div>
        </div>
      </div>

      {error && (
        <p className="mb-4 rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>
      )}

      {s && (
        <>
          <div className="mb-6 grid gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5">
            <Stat icon={IconOrder} label={r.summary.ordersTotal} value={fmt.format(s.orders_total)} />
            <Stat
              icon={IconOrder}
              label={r.summary.delivered}
              value={fmt.format(s.delivered)}
              sub={`${fmt.format(s.cancelled)} ${r.summary.cancelled}`}
            />
            <Stat
              icon={IconWallet}
              label={r.summary.grossSales}
              value={`${fmt.format(s.gross_sales)} ${m.common.currency}`}
            />
            <Stat
              icon={IconWallet}
              label={r.summary.commissions}
              value={`${fmt.format(s.commissions)} ${m.common.currency}`}
              sub={`${r.summary.deliveryFees}: ${fmt.format(s.delivery_fees)}`}
            />
            <Stat
              icon={IconStatus}
              label={r.summary.avgDelivery}
              value={`${fmt.format(s.avg_delivery_min)} ${r.summary.minutes}`}
              sub={`${fmt.format(s.active_customers)} ${r.summary.activeCustomers}`}
            />
          </div>

          <div className="mb-6 grid gap-4 lg:grid-cols-2">
            <DailyBars
              title={r.dailyOrders}
              data={report.daily}
              value={(d) => d.orders}
              format={(v) => fmt.format(v)}
            />
            <DailyBars
              title={r.dailySales}
              data={report.daily}
              value={(d) => d.sales}
              format={(v) => `${fmt.format(v)} ${m.common.currency}`}
            />
          </div>

          <div className="grid gap-4 lg:grid-cols-2">
            <section className="rounded-card border border-line bg-surface p-5">
              <h2 className="mb-3 flex items-center gap-2 text-sm font-bold">
                <IconStore size={16} className="text-primary" />
                {r.topMerchants}
              </h2>
              {report.top_merchants.length === 0 ? (
                <p className="py-4 text-center text-sm text-ink-muted">{r.noData}</p>
              ) : (
                <ol className="space-y-2">
                  {report.top_merchants.map((t, i) => (
                    <li key={t.name} className="flex items-center justify-between text-sm">
                      <span>
                        <span className="me-2 inline-flex h-5 w-5 items-center justify-center rounded-badge bg-primary-light text-xs font-bold text-primary-dark">
                          {i + 1}
                        </span>
                        {t.name}
                        <span className="text-xs text-ink-muted"> · {fmt.format(t.delivered)} {r.deliveries}</span>
                      </span>
                      <span className="font-bold text-primary-dark">
                        {fmt.format(t.sales)} {m.common.currency}
                      </span>
                    </li>
                  ))}
                </ol>
              )}
            </section>

            <section className="rounded-card border border-line bg-surface p-5">
              <h2 className="mb-3 flex items-center gap-2 text-sm font-bold">
                <IconDriver size={16} className="text-primary" />
                {r.topDrivers}
              </h2>
              {report.top_drivers.length === 0 ? (
                <p className="py-4 text-center text-sm text-ink-muted">{r.noData}</p>
              ) : (
                <ol className="space-y-2">
                  {report.top_drivers.map((t, i) => (
                    <li key={t.phone} className="flex items-center justify-between text-sm">
                      <span>
                        <span className="me-2 inline-flex h-5 w-5 items-center justify-center rounded-badge bg-primary-light text-xs font-bold text-primary-dark">
                          {i + 1}
                        </span>
                        {t.name || <span dir="ltr">{t.phone}</span>}
                      </span>
                      <span>
                        <span className="font-bold">{fmt.format(t.delivered)}</span>{" "}
                        <span className="text-xs text-ink-muted">{r.deliveries}</span>
                      </span>
                    </li>
                  ))}
                </ol>
              )}
            </section>
          </div>
        </>
      )}
    </div>
  );
}
