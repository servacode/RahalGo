"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **‏/store/reports — «التقارير»: مبيعاتك على مدًى تختاره**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `SalesScreen` و`SalesViewModel` في `DrawerScreens.kt`.)
 *
 * **أربعةُ مَدَيات**: اليوم · ٧ أيّام · ٣٠ يوماً · مدّةٌ محدّدة (من · لـ)
 * — **والأيّامُ بتوقيت دمشق** (`damascusDay`)، كما يحسبها المحرّك.
 *
 * **ستّةُ أرقام**: الطلبات · تم التسليم · ملغي · المبيعات · عمولةُ المنصّة ·
 * مستحقُّك — **و«طباعة التقرير» طباعةُ المتصفّح للصفحة نفسِها.**
 */

import { useCallback, useState } from "react";
import { getMessages, defaultLocale, damascusDay, fmtCalendarDay, fmtNum } from "@rahalgo/i18n";
import { Button, Chip, EmptyState, IconBlock, IconCheck, IconOrder, IconPrint, IconReceipt, IconTrendUp, IconWallet, Input, Kpi, KpiGrid, LoadingState, ReloadState, useLiveData } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useStore } from "@/lib/store/context";
import type { StoreReport } from "@/lib/store/types";
import { fill } from "@/lib/customer/texts";
import { PageHeader } from "@/components/store/SetupParts";

const m = getMessages(defaultLocale);
const R = m.store.sales;
const K = m.store.console;

type Range = "today" | "week" | "month" | "custom";

/** **يومٌ قبل يومٍ بأيّام** — على `YYYY-MM-DD` بلا منطقةٍ زمنيّة تقلبه. */
function minusDays(day: string, n: number): string {
  const d = new Date(`${day}T12:00:00Z`);
  d.setUTCDate(d.getUTCDate() - n);
  return d.toISOString().slice(0, 10);
}

export default function ReportsPage() {
  const { store } = useStore();
  const id = store?.id ?? "";
  const today = damascusDay();
  const [range, setRange] = useState<Range>("today");
  const [from, setFrom] = useState(`${today.slice(0, 8)}01`);
  const [to, setTo] = useState(today);

  const span =
    range === "today"
      ? { from: today, to: today }
      : range === "week"
        ? { from: minusDays(today, 6), to: today }
        : range === "month"
          ? { from: minusDays(today, 29), to: today }
          : { from: from > to ? to : from, to };

  const load = useCallback(
    () => api<StoreReport>(`/api/v1/merchant/stores/${id}/reports?from=${span.from}&to=${span.to}`),
    [id, span.from, span.to],
  );
  const { data, loading, error, reload } = useLiveData(load, ["order"], [id, span.from, span.to]);
  const s = data?.summary;

  const rate = s && s.orders > 0 ? Math.round((s.delivered / s.orders) * 100) : null;
  const periods: { id: Range; label: string }[] = [
    { id: "today", label: R.today },
    { id: "week", label: R.week },
    { id: "month", label: R.month },
    { id: "custom", label: R.custom },
  ];

  return (
    <div className="console-page" data-testid="store-reports">
      <div className="print:hidden">
        <PageHeader
          title={R.title}
          subtitle={fill(R.range, { from: fmtCalendarDay(span.from), to: fmtCalendarDay(span.to) })}
          actions={
            s && (
              <Button variant="secondary" className="flex min-h-11 items-center justify-center gap-2" onClick={() => window.print()}>
                <IconPrint size={16} />
                {R.print}
              </Button>
            )
          }
        />
      </div>
      {/* ── المدى: رقائق + تاريخان لمدّةٍ محدّدة ── */}
      <div className="flex flex-col gap-3 print:hidden">
        <div className="rail-x -mx-3 px-3 lg:mx-0 lg:px-0" aria-label={R.title}>
          {periods.map((x) => (
            <Chip key={x.id} selected={range === x.id} onClick={() => setRange(x.id)} testId={`reports-range-${x.id}`}>
              {x.label}
            </Chip>
          ))}
        </div>
        {range === "custom" && (
          <div className="panel grid grid-cols-1 gap-3 p-3 sm:grid-cols-2 lg:max-w-xl">
            <Input id="sales-from" type="date" label={R.from} dir="ltr" max={today} value={from} onChange={(e) => e.target.value && setFrom(e.target.value)} wrapperClassName="min-w-0" />
            <Input id="sales-to" type="date" label={R.to} dir="ltr" max={today} value={to} onChange={(e) => e.target.value && setTo(e.target.value)} wrapperClassName="min-w-0" />
          </div>
        )}
      </div>
      <div className="hidden print:block">
        <h2 className="heading-page">{`${R.title} · ${store?.name ?? ""}`}</h2>
        <p className="console-meta">{fill(R.range, { from: fmtCalendarDay(span.from), to: fmtCalendarDay(span.to) })}</p>
      </div>
      {loading && !data ? (
        <LoadingState variant="stats" rows={6} />
      ) : error && !data ? (
        <ReloadState onRetry={() => void reload()} />
      ) : (
        s && (
          <>
            {/* **المال أوّلاً**: المبيعات · العمولة · مستحقّك — ثمّ العدّ. */}
            <KpiGrid cols={3}>
              <Kpi icon={IconTrendUp} label={R.sales} value={fmtNum(s.sales)} unit={m.common.currency} tone="accent" testId="reports-sales" />
              <Kpi icon={IconReceipt} label={R.commission} value={fmtNum(s.platform_commission)} unit={m.common.currency} testId="reports-commission" />
              <Kpi
                icon={IconWallet}
                label={R.due}
                value={fmtNum(s.sales - s.platform_commission)}
                unit={m.common.currency}
                tone="good"
                testId="reports-due"
                className="col-span-2 lg:col-span-1"
              />
            </KpiGrid>
            <KpiGrid cols={3}>
              <Kpi icon={IconOrder} label={R.orders} value={s.orders} testId="reports-orders" />
              <Kpi
                icon={IconCheck}
                label={R.delivered}
                value={s.delivered}
                tone="good"
                sub={rate != null ? fill(K.rpRate, { p: fmtNum(rate) }) : undefined}
                testId="reports-delivered"
              />
              <Kpi icon={IconBlock} label={R.cancelled} value={s.cancelled} tone={s.cancelled > 0 ? "attention" : undefined} testId="reports-cancelled" />
            </KpiGrid>
            {s.orders === 0 && <EmptyState icon={IconReceipt} title={K.rpNone} />}
          </>
        )
      )}
    </div>
  );
}
