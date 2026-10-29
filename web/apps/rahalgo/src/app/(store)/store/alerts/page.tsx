"use client";

/**
 * **‏/store/alerts — «الإنذارات»**: ما على متجري، يُقرأ قبل أن يُحظر.
 *
 * (مرآةُ `WarningsScreen` في `DrawerScreens.kt` — `/merchant/warnings`.)
 *
 * **والسببُ رمزٌ من المحرّك** يقوله المعجمُ بالعربيّة (`merchant.warnings.
 * reasons` — معجمُ البوّابة القديمة نفسُه)، **وما لا لفظَ له يُعرض كما جاء.**
 */

import { useCallback } from "react";
import { getMessages, defaultLocale, fmtRelDay } from "@rahalgo/i18n";
import { EmptyState, IconWarning, LoadingState, ReloadState, useLiveData } from "@rahalgo/ui";
import { api } from "@/lib/api";
import type { Warning } from "@/lib/store/types";
import { fill } from "@/lib/customer/texts";
import { PageHeader } from "@/components/store/SetupParts";

const m = getMessages(defaultLocale);
const W = m.store.alerts;
const REASONS = m.merchant.warnings.reasons as Record<string, string>;

export default function AlertsPage() {
  const load = useCallback(() => api<{ warnings: Warning[] }>("/api/v1/merchant/warnings"), []);
  const { data, loading, error, reload } = useLiveData(load, ["merchant"], []);

  if (loading && !data) return <LoadingState variant="list" rows={3} />;
  if (error && !data) return <ReloadState onRetry={() => void reload()} />;
  const rows = data?.warnings ?? [];

  return (
    <div className="flex flex-col gap-3">
      <PageHeader title={W.title} />
      {rows.length === 0 ? (
        <EmptyState icon={IconWarning} title={W.empty} tone="success" />
      ) : (
        <ul className="surface flex flex-col divide-y divide-line rounded-card px-4">
          {rows.map((w) => (
            <li key={w.id} className="flex flex-col gap-0.5 py-3" data-testid="store-warning">
              <span className="font-bold">{REASONS[w.reason] ?? w.reason}</span>
              {w.note && <span className="text-sm text-ink-muted">{w.note}</span>}
              <span className="flex gap-3 text-2xs text-ink-muted">
                <span>{fmtRelDay(w.created_at)}</span>
                {w.order_number != null && <span>{fill(W.onOrder, { n: `#${w.order_number}` })}</span>}
              </span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
}
