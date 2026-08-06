"use client";

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";
import {
  Alert,
  PageHeader,
  Input,
  Button,
  IconOrder,
  IconWallet,
  IconStore,
  IconDriver,
  IconUser,
  IconStatus,
  IconDate,
} from "@rahalgo/ui";
import { api, ApiError, tokenStore } from "@/lib/api";

const m = getMessages(defaultLocale);

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
    <section className="surface p-5">
      <h2 className="mb-4 text-sm font-bold">{title}</h2>
      <div className="relative">
        {/* شبكة خافتة: خطا الربع والنصف والثلاثة أرباع */}
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
                {/* تلميح التحويم */}
                {hover === i && (
                  <div className="pointer-events-none absolute bottom-full mb-1 whitespace-nowrap rounded-control border border-line bg-surface px-2.5 py-1.5 text-xs elev-2">
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
}: {
  icon: React.ComponentType<{ size?: number; className?: string }>;
  label: string;
  value: string;
  sub?: string;
}) {
  return (
    <div className="flex items-center gap-3 surface p-4">
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

  /**
   * تنزيلُ CSV — **بالمدّة المعروضة نفسِها.**
   *
   * **ولا يُطلب تاريخان مرّةً ثانية**: من ضبط المدّةَ ليرى التقرير يريد
   * تصديرَ ما يراه، **وحقلان ثانيان يجعلان الملفَّ يخالف الشاشة بلا أن يُلاحظ.**
   */
  function download(kind: "orders" | "ledger") {
    const base = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
    const path = kind === "orders" ? "orders/export" : "ledger/export";
    void (async () => {
      try {
        const res = await fetch(`${base}/api/v1/admin/${path}?from=${from}&to=${to}`, {
          headers: { Authorization: `Bearer ${tokenStore.access ?? ""}` },
        });
        if (!res.ok) {
          setError(m.errors.internal);
          return;
        }
        const blob = await res.blob();
        const url = URL.createObjectURL(blob);
        const a = document.createElement("a");
        a.href = url;
        a.download = `${kind}-${from}_${to}.csv`;
        a.click();
        URL.revokeObjectURL(url);
      } catch {
        setError(m.errors.internal);
      }
    })();
  }

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
        <PageHeader icon={IconStatus} title={r.title} />
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
          {/* **وتصديرُ الأصل لا المجاميع.**

              التقاريرُ تُخرج المجاميع، **وما ينقص هو السطور التي تُبنى عليها**:
              من دفع كم، ولمن ذهب، وماذا بقي. **ومن شكّ في مجموعٍ عاد إلى
              السطور، ومن لا يملكها يُصدّق أو يشكّ بلا سبيل.**

              وكان التصديرُ للحسابات وحدَها — **ومحاسبٌ يريد كشفاً شهرياً لا
              يجد ما يأخذه**، فينسخ من الشاشة صفحةً صفحة. */}
          <Button variant="secondary" onClick={() => download("orders")}>
            {r.exportOrders}
          </Button>
          <Button variant="secondary" onClick={() => download("ledger")}>
            {r.exportLedger}
          </Button>
        </div>
      </div>

      {error && (
        <Alert className="mb-4">{error}</Alert>
      )}

      {s && (
        <>
          <div className="mb-6 grid gap-3 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-5">
            <Stat icon={IconOrder} label={r.summary.ordersTotal} value={fmtNum(s.orders_total)} />
            <Stat
              icon={IconOrder}
              label={r.summary.delivered}
              value={fmtNum(s.delivered)}
              sub={`${fmtNum(s.cancelled)} ${r.summary.cancelled}`}
            />
            <Stat
              icon={IconWallet}
              label={r.summary.grossSales}
              value={`${fmtNum(s.gross_sales)} ${m.common.currency}`}
            />
            <Stat
              icon={IconWallet}
              label={r.summary.commissions}
              value={`${fmtNum(s.commissions)} ${m.common.currency}`}
              sub={`${r.summary.deliveryFees}: ${fmtNum(s.delivery_fees)}`}
            />
            <Stat
              icon={IconStatus}
              label={r.summary.avgDelivery}
              value={`${fmtNum(s.avg_delivery_min)} ${r.summary.minutes}`}
              sub={`${fmtNum(s.active_customers)} ${r.summary.activeCustomers}`}
            />
          </div>

          <div className="mb-6 grid gap-4 lg:grid-cols-2">
            <DailyBars
              title={r.dailyOrders}
              data={report.daily}
              value={(d) => d.orders}
              format={(v) => fmtNum(v)}
            />
            <DailyBars
              title={r.dailySales}
              data={report.daily}
              value={(d) => d.sales}
              format={(v) => `${fmtNum(v)} ${m.common.currency}`}
            />
          </div>

          <div className="grid gap-4 lg:grid-cols-2">
            <section className="surface p-5">
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
                        <span className="text-xs text-ink-muted"> · {fmtNum(t.delivered)} {r.deliveries}</span>
                      </span>
                      <span className="font-bold text-primary-dark">
                        {fmtNum(t.sales)} {m.common.currency}
                      </span>
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
                    <li key={t.phone} className="flex items-center justify-between text-sm">
                      <span>
                        <span className="me-2 inline-flex h-5 w-5 items-center justify-center rounded-badge bg-primary-light text-xs font-bold text-primary-dark">
                          {i + 1}
                        </span>
                        {t.name || <span dir="ltr">{t.phone}</span>}
                      </span>
                      <span>
                        <span className="font-bold">{fmtNum(t.delivered)}</span>{" "}
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
