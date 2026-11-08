"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **سلسلةُ المبيعات لرسم «متجري» — من نقطة التقارير نفسِها بلا حقلٍ جديد**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (التصميمُ المعتمد ٢٠٢٦-١١-٠٨: خطُّ هذا الأسبوع وخطٌّ متقطّعٌ للأسبوع الماضي، والنسبةُ عن أمس وعن الأحد الماضي.)
 *
 * **لا يوجد في المحرّك تفصيلٌ يوميّ**، فكلُّ نقطةٍ نداءُ تقريرٍ لمدّةٍ (`reports?from&to`) — وهي النقطةُ التي
 * تقرؤها «متجري» اليوم. **٧ أيّامٍ = نقطةٌ لكلّ يوم · ٣٠ يوماً = ١٠ نقاطٍ لكلٍّ ثلاثةُ أيّام.**
 * والفترةُ السابقةُ تُقرأ بنداءاتٍ منفصلةٍ بعد الحاضرة، فلا تؤخّر الرسمَ الأوّل. **وفشلُ نقطةٍ يُعدّها صفراً** لا يُسقط الرسم.
 * وتتجدّد بحدث «order» الحيّ كما تتجدّد أرقامُ اليوم.
 */

import { useCallback } from "react";
import { damascusDay } from "@rahalgo/i18n";
import { useLiveData } from "@rahalgo/ui";
import { api } from "@/lib/api";
import type { StoreReport } from "./types";

export type SalesRange = 7 | 30;

export interface SalesRow {
  /** أوّلُ يومٍ في النقطة وآخرُه (`YYYY-MM-DD`). */
  from: string;
  to: string;
  sales: number;
  orders: number;
  cancelled: number;
  commission: number;
}

const DAY = 86_400_000;

/** حدودُ نقاط الفترة المنتهيّة بعد `shiftDays` يوماً من اليوم (٠ = الحاضرة). */
function buckets(range: SalesRange, shiftDays: number): { from: string; to: string }[] {
  const size = range === 7 ? 1 : 3;
  const n = range / size;
  const endMs = Date.now() - shiftDays * DAY;
  return Array.from({ length: n }, (_, i) => {
    const last = endMs - (n - 1 - i) * size * DAY;
    return { from: damascusDay(last - (size - 1) * DAY), to: damascusDay(last) };
  });
}

function readRows(storeId: string, spans: { from: string; to: string }[]): Promise<SalesRow[]> {
  return Promise.all(
    spans.map((s) =>
      api<StoreReport>(`/api/v1/merchant/stores/${storeId}/reports?from=${s.from}&to=${s.to}`).then(
        (r): SalesRow => ({
          ...s,
          sales: r.summary.sales,
          orders: r.summary.orders,
          cancelled: r.summary.cancelled,
          commission: r.summary.platform_commission,
        }),
        (): SalesRow => ({ ...s, sales: 0, orders: 0, cancelled: 0, commission: 0 }),
      ),
    ),
  );
}

/** **الفترةُ الحاضرةُ والسابقةُ لها** — `prev` تأتي بعد `cur` ولا تحجبها. */
export function useSalesSeries(storeId: string, range: SalesRange, enabled = true): { cur: SalesRow[] | null; prev: SalesRow[] | null } {
  const loadCur = useCallback(
    () => (enabled ? readRows(storeId, buckets(range, 0)) : Promise.resolve<SalesRow[]>([])),
    [storeId, range, enabled],
  );
  const { data: cur } = useLiveData(loadCur, ["order"], [storeId, range, enabled]);
  const ready = enabled && cur != null && cur.length > 0;
  const loadPrev = useCallback(
    () => (ready ? readRows(storeId, buckets(range, range)) : Promise.resolve<SalesRow[]>([])),
    [storeId, range, ready],
  );
  const { data: prev } = useLiveData(loadPrev, ["order"], [storeId, range, ready]);
  return { cur: ready ? cur : null, prev: ready && prev && prev.length > 0 && prev.length === cur?.length ? prev : null };
}

/** نسبةُ التغيّر عن قيمةٍ سابقة — `null` حين لا أساسَ للمقارنة (صفرٌ سابقاً). */
export function pctChange(now: number, before: number): number | null {
  if (before <= 0) return null;
  return Math.round(((now - before) / before) * 100);
}
