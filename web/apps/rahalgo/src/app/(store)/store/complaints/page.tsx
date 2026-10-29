"use client";

/**
 * **‏/store/complaints — «الشكاوي والبلاغات»**: ما رفعتُه وما رُفع عليّ.
 *
 * (مرآةُ `MyReportsScreen` في `DrawerScreens.kt` و`TicketsScreen` في `:ui`:
 *  تبويبان «شكاوى رفعتها» (`/merchant/my-reports`) و«شكاوى عليّ»
 *  (`/stores/{id}/reports-against`) — **والبلاغُ نفسُه يُرفع من سجلّ الطلبات.**)
 */

import { useCallback, useState } from "react";
import { getMessages, defaultLocale, fmtRelDay } from "@rahalgo/i18n";
import { EmptyState, IconChat, LoadingState, ReloadState, Tabs, useLiveData } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useStore } from "@/lib/store/context";
import type { MerchantReport } from "@/lib/store/types";
import { fill } from "@/lib/customer/texts";
import { reasonText } from "@/components/store/HistoryParts";
import { PageHeader } from "@/components/store/SetupParts";

const m = getMessages(defaultLocale);
const C = m.store.complaints;

function Row({ t }: { t: MerchantReport }) {
  const status = (C.status as Record<string, string>)[t.status] ?? t.status;
  return (
    <li className="flex flex-col gap-1 py-3" data-testid="store-ticket">
      <span className="flex items-center justify-between gap-2">
        <span className="font-bold">{`#${t.number} · ${reasonText(t.reason)}`}</span>
        <span className={`shrink-0 text-xs ${t.status === "resolved" ? "text-success" : "text-ink-muted"}`}>{status}</span>
      </span>
      {t.order_number != null && <span className="text-xs text-ink-muted">{fill(C.onOrder, { n: `#${t.order_number}` })}</span>}
      {t.resolution && <span className="text-sm">{t.resolution}</span>}
      <span className="text-2xs text-ink-muted">{fmtRelDay(t.created_at)}</span>
    </li>
  );
}

export default function ComplaintsPage() {
  const { store } = useStore();
  const id = store?.id ?? "";
  const [tab, setTab] = useState<"mine" | "onMe">("mine");
  const load = useCallback(async () => {
    const [mine, against] = await Promise.all([
      api<{ reports: MerchantReport[] }>("/api/v1/merchant/my-reports"),
      api<{ reports: MerchantReport[] }>(`/api/v1/merchant/stores/${id}/reports-against`),
    ]);
    return { mine: mine.reports ?? [], against: against.reports ?? [] };
  }, [id]);
  const { data, loading, error, reload } = useLiveData(load, ["support", "merchant"], [id]);

  if (loading && !data) return <LoadingState variant="list" rows={3} />;
  if (error && !data) return <ReloadState onRetry={() => void reload()} />;
  const rows = tab === "mine" ? (data?.mine ?? []) : (data?.against ?? []);

  return (
    <div className="flex flex-col gap-3">
      <PageHeader title={C.title} />
      <Tabs
        items={[
          { key: "mine", label: C.mine, count: data?.mine.length ?? 0 },
          { key: "onMe", label: C.onMe, count: data?.against.length ?? 0 },
        ]}
        value={tab}
        onChange={setTab}
      />
      {rows.length === 0 ? (
        <EmptyState icon={IconChat} title={tab === "mine" ? C.mineEmpty : C.onMeEmpty} tone={tab === "onMe" ? "success" : "muted"} />
      ) : (
        <ul className="surface flex flex-col divide-y divide-line rounded-card px-4">
          {rows.map((t) => (
            <Row key={t.id} t={t} />
          ))}
        </ul>
      )}
    </div>
  );
}
