"use client";

/**
 * **‏/store/history/[id] — تفاصيلُ طلبٍ انتهى** (`/merchant/orders/{id}`):
 * الأصنافُ بخياراتها وملاحظاتها · ملاحظةُ الزبون · المالُ · الحالةُ ووقتُها
 * **وسببُ الإلغاء إن أُلغي.** — «يُفتح عند سؤال» كما يقول السجلّ.
 */

import { useCallback } from "react";
import { useParams } from "next/navigation";
import { getMessages, defaultLocale, fmtMoney, fmtRelStamp } from "@rahalgo/i18n";
import { LoadingState, ReloadState, useLiveData } from "@rahalgo/ui";
import { api } from "@/lib/api";
import type { MerchantOrder } from "@/lib/store/types";
import { fill } from "@/lib/customer/texts";
import { MoneyLines, orderStatus } from "@/components/store/HistoryParts";
import { PageHeader } from "@/components/store/SetupParts";

const m = getMessages(defaultLocale);
const S = m.store;
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
    <div className="flex flex-col gap-3" data-testid="history-detail">
      <PageHeader
        title={fill(H.orderTitle, { n: o.number })}
        href="/store/history"
        actions={<span className="shrink-0 text-sm text-ink-muted">{orderStatus(o.status)}</span>}
      />
      <p className="text-xs text-ink-muted">{fmtRelStamp(o.closed_at ?? o.created_at)}</p>
      <section className="surface flex flex-col gap-2 rounded-card p-3">
        <h3 className="heading-card">{S.orders.items}</h3>
        <ul className="flex flex-col divide-y divide-line">
          {o.items.map((l) => (
            <li key={l.id} className="flex flex-col gap-0.5 py-2 text-sm">
              <span className="flex justify-between gap-2">
                <span className="font-bold">{`${l.qty}× ${l.name}`}</span>
                <span>{fmtMoney(l.unit_price * l.qty)}</span>
              </span>
              {l.options.length > 0 && (
                <span className="text-xs text-ink-muted">
                  {l.options.map((op) => `${op.group}: ${op.name}`).join(m.common.listSep)}
                </span>
              )}
              {l.note && <span className="text-xs text-ink-muted">{l.note}</span>}
            </li>
          ))}
        </ul>
        {o.notes && (
          <p className="rounded-control bg-field p-2 text-sm">
            <span className="font-bold">{S.orders.note}: </span>
            {o.notes}
          </p>
        )}
      </section>
      <section className="surface rounded-card p-3">
        <MoneyLines o={o} />
      </section>
      {o.cancel_reason && (
        <p className="text-sm text-ink-muted">
          <span className="font-bold">{H.cancelReason}: </span>
          {o.cancel_reason}
        </p>
      )}
    </div>
  );
}
