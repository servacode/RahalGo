"use client";

/** تقارير المتجر: ملخص بمدى زمني + أعمدة يومية (منهجية dataviz الموحدة). */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";
import { IconStatus, IconOrder, IconSuccess, IconError, IconWallet, Input } from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";
import { useStore } from "@/lib/store";

const m = getMessages(defaultLocale);

interface Report {
  summary: {
    orders: number;
    delivered: number;
    cancelled: number;
    sales: number;
    platform_commission: number;
  };
  days: { date: string; orders: number; delivered: number; sales: number }[];
}

function isoDaysAgo(n: number): string {
  const d = new Date();
  d.setDate(d.getDate() - n);
  return d.toISOString().slice(0, 10);
}

export default function MerchantReportsPage() {
  const { store } = useStore();
  const [from, setFrom] = useState(isoDaysAgo(6));
  const [to, setTo] = useState(isoDaysAgo(0));
  const [report, setReport] = useState<Report | null>(null);
  const [error, setError] = useState("");

  const load = useCallback(async () => {
    if (!store) return;
    try {
      setReport(
        await api<Report>(`/api/v1/merchant/stores/${store.id}/reports?from=${from}&to=${to}`)
      );
      setError("");
    } catch (err) {
      setError(err instanceof ApiError ? m.errors.internal : m.errors.internal);
    }
  }, [store, from, to]);

  useEffect(() => {
    void load();
  }, [load]);

  const s = report?.summary;
  const net = s ? s.sales - s.platform_commission : 0;
  const maxSales = Math.max(1, ...(report?.days.map((d) => d.sales) ?? [1]));

  const cards = s
    ? [
        { label: m.merchant.reports.orders, value: fmtNum(s.orders), icon: <IconOrder /> },
        {
          label: m.merchant.reports.delivered,
          value: fmtNum(s.delivered),
          icon: <IconSuccess className="text-success" />,
        },
        {
          label: m.merchant.reports.cancelled,
          value: fmtNum(s.cancelled),
          icon: <IconError className="text-danger" />,
        },
        {
          label: `${m.merchant.reports.sales} (${m.common.currency})`,
          value: fmtNum(s.sales),
          icon: <IconWallet />,
        },
        {
          label: `${m.merchant.reports.net} (${m.common.currency})`,
          value: fmtNum(net),
          icon: <IconWallet className="text-success" />,
        },
      ]
    : [];

  return (
    <div>
      <div className="mb-5 flex flex-wrap items-end justify-between gap-3">
        <h1 className="flex items-center gap-2 text-xl font-bold">
          <IconStatus className="text-primary" />
          {m.merchant.reports.title}
        </h1>
        <div className="flex gap-2">
          <Input
            id="from"
            type="date"
            dir="ltr"
            value={from}
            onChange={(e) => setFrom(e.target.value)}
          />
          <Input id="to" type="date" dir="ltr" value={to} onChange={(e) => setTo(e.target.value)} />
        </div>
      </div>

      {error && (
        <p className="mb-4 rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>
      )}

      <div className="mb-6 grid grid-cols-2 gap-3 md:grid-cols-5">
        {cards.map((c) => (
          <div key={c.label} className="rounded-card border border-line bg-surface p-4">
            <div className="mb-1 flex items-center gap-2 text-ink-muted">{c.icon}</div>
            <p className="text-xl font-bold">{c.value}</p>
            <p className="text-xs text-ink-muted">{c.label}</p>
          </div>
        ))}
      </div>

      {report && (
        <section className="rounded-card border border-line bg-surface p-4">
          <h2 className="mb-4 font-bold">{m.merchant.reports.daily}</h2>
          <div className="flex items-end gap-2" style={{ height: 160 }}>
            {report.days.map((d) => (
              <div key={d.date} className="group relative flex-1">
                <div
                  className="mx-auto w-3/5 rounded-t-[4px] bg-primary transition-colors group-hover:bg-primary-dark"
                  style={{ height: Math.max(3, (d.sales / maxSales) * 140) }}
                />
                <div className="pointer-events-none absolute bottom-full start-1/2 z-10 mb-1 hidden -translate-x-1/2 whitespace-nowrap rounded-control bg-ink px-2 py-1 text-xs text-white group-hover:block rtl:translate-x-1/2">
                  {fmtNum(d.sales)} {m.common.currency} — {fmtNum(d.delivered)}/
                  {fmtNum(d.orders)}
                </div>
              </div>
            ))}
          </div>
          <div className="mt-1 flex gap-2 border-t border-line pt-1">
            {report.days.map((d) => (
              <span key={d.date} className="flex-1 text-center text-[10px] text-ink-muted" dir="ltr">
                {d.date.slice(8)}
              </span>
            ))}
          </div>
        </section>
      )}
    </div>
  );
}
