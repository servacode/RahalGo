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
import { getMessages, defaultLocale, damascusDay, fmtCalendarDay, fmtMoney, fmtNum } from "@rahalgo/i18n";
import { Button, Chips, IconPrint, Input, LoadingState, ReloadState, StatCard, StatGrid, useLiveData } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useStore } from "@/lib/store/context";
import type { StoreReport } from "@/lib/store/types";
import { fill } from "@/lib/customer/texts";
import { PageHeader } from "@/components/store/SetupParts";

const m = getMessages(defaultLocale);
const R = m.store.sales;

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

  return (
    <div className="flex flex-col gap-3">
      <div className="print:hidden">
        <PageHeader title={R.title} />
      </div>
      <div className="print:hidden">
        <Chips
          wrap
          items={[
            { id: "today", label: R.today },
            { id: "week", label: R.week },
            { id: "month", label: R.month },
            { id: "custom", label: R.custom },
          ]}
          value={range}
          onChange={setRange}
        />
      </div>
      {range === "custom" && (
        <div className="flex flex-wrap gap-2 print:hidden">
          <Input id="sales-from" type="date" label={R.from} dir="ltr" max={today} value={from} onChange={(e) => e.target.value && setFrom(e.target.value)} wrapperClassName="min-w-0 flex-1" />
          <Input id="sales-to" type="date" label={R.to} dir="ltr" max={today} value={to} onChange={(e) => e.target.value && setTo(e.target.value)} wrapperClassName="min-w-0 flex-1" />
        </div>
      )}
      <div className="hidden print:block">
        <h2 className="heading-page">{`${R.title} · ${store?.name ?? ""}`}</h2>
      </div>
      {/* **والمدى بالأيّام لا بصيغة الآلة** (مرآةُ `Dates.ymd`، تدقيقُ ٢٠٢٦-١٠-٢٧). */}
      <p className="text-xs text-ink-muted">
        {fill(R.range, { from: fmtCalendarDay(span.from), to: fmtCalendarDay(span.to) })}
      </p>
      {loading && !data ? (
        <LoadingState variant="stats" rows={6} />
      ) : error && !data ? (
        <ReloadState onRetry={() => void reload()} />
      ) : (
        s && (
          <StatGrid>
            <StatCard label={R.orders} value={fmtNum(s.orders)} />
            <StatCard label={R.delivered} value={fmtNum(s.delivered)} tone="success" />
            <StatCard label={R.cancelled} value={fmtNum(s.cancelled)} tone="muted" />
            <StatCard label={R.sales} value={fmtMoney(s.sales)} />
            <StatCard label={R.commission} value={fmtMoney(s.platform_commission)} tone="muted" />
            <StatCard label={R.due} value={fmtMoney(s.sales - s.platform_commission)} tone="accent" emphasis />
          </StatGrid>
        )
      )}
      {s && (
        <Button className="flex w-full items-center justify-center gap-2 print:hidden" onClick={() => window.print()}>
          <IconPrint size={16} />
          {R.print}
        </Button>
      )}
    </div>
  );
}
