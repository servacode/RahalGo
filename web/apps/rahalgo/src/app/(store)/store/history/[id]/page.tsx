"use client";

/**
 * **‏/store/history/[id] — تفاصيلُ طلبٍ انتهى** (`/merchant/orders/{id}`):
 * الأصنافُ بخياراتها وملاحظاتها · ملاحظةُ الزبون · المالُ · الحالةُ ووقتُها
 * **وسببُ الإلغاء إن أُلغي.** — «يُفتح عند سؤال» كما يقول السجلّ.
 */

import { useCallback } from "react";
import { useParams } from "next/navigation";
import { getMessages, defaultLocale, fmtNum, fmtRelStamp } from "@rahalgo/i18n";
import { Badge, InvoiceLines, LoadingState, ReloadState, useLiveData } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { noPayout, type MerchantOrder } from "@/lib/store/types";
import { fill } from "@/lib/customer/texts";
import { MoneyLines, orderStatus } from "@/components/store/HistoryParts";
import { PageHeader } from "@/components/store/SetupParts";

const m = getMessages(defaultLocale);
const S = m.store;
const K = S.console;
const H = S.history;

export default function HistoryOrderPage() {
  const params = useParams<{ id: string }>();
  const id = params?.id ?? "";
  const load = useCallback(() => api<MerchantOrder>(`/api/v1/merchant/orders/${id}`), [id]);
  const { data: o, loading, error, reload } = useLiveData(load, ["order"], [id]);

  if (loading && !o) return <LoadingState variant="list" rows={3} />;
  if (error && !o) return <ReloadState onRetry={() => void reload()} />;
  if (!o) return null;

  return (
    <div className="console-page" data-testid="history-detail">
      <PageHeader
        title={fill(H.orderTitle, { n: o.number })}
        href="/store/history"
        subtitle={fmtRelStamp(o.closed_at ?? o.created_at)}
        actions={<Badge variant={o.status === "delivered" ? "success" : noPayout(o.status) ? "danger" : "neutral"}>{orderStatus(o.status)}</Badge>}
      />
      {(o.shared_stops ?? 0) >= 2 && (
        <p className="text-caption font-bold text-accent-text">{fill(S.orders.sharedStops, { n: fmtNum(o.shared_stops ?? 0) })}</p>
      )}
      <div className="grid grid-cols-1 gap-rg-lg md:grid-cols-[minmax(0,1.4fr)_minmax(0,1fr)] md:items-start md:gap-5">
      <section className="panel flex flex-col gap-rg-sm p-rg-lg">
        <h2 className="console-h2">{S.orders.items}</h2>
        {/* **سطرٌ لكلّ عنصر** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — «شاورما دبل × 3 ...... 90 ل.س». */}
        <InvoiceLines items={o.items} />
        {o.notes && (
          <p className="rounded-control bg-field p-rg-sm text-body">
            <span className="font-bold">{S.orders.note}: </span>
            {o.notes}
          </p>
        )}
      </section>
      <section className="panel flex flex-col gap-rg-sm p-rg-lg">
        <h2 className="console-h2">{K.histMoney}</h2>
        <MoneyLines o={o} />
      </section>
      </div>
      {o.cancel_reason && (
        <p className="text-body text-ink-muted">
          <span className="font-bold">{H.cancelReason}: </span>
          {o.cancel_reason}
        </p>
      )}
    </div>
  );
}
