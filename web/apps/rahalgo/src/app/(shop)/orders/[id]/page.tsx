"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **تفاصيلُ الطلب — مراحلُه وكودُه وحسابُه وإلغاؤه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **يتحدّث كلَّ عشر ثوانٍ** ما دام جارياً والصفحةُ ظاهرة — ويسكن إذا انتهى.
 *
 * **وزرُّ الإلغاء ما دامت المهلة** (`cancel_seconds_left` من المحرّك).
 * **وكودُ الاستلام حين يصير الطلبُ بيد السائق.** **ونافذةُ السعر** (للطلب
 * الخاصّ) **تُفتح من القشرة** في أيّ شاشة — فلا تُكرَّر هنا.
 */

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { getMessages, defaultLocale, errorText, fmtDateTime, fmtMoney, fmtNum, fmtRef } from "@rahalgo/i18n";
import { Alert, Badge, LoadingState, Money, ReloadState, IconPrev } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { fill, statusLabel } from "@/lib/customer/texts";
import { ENDED, type MyOrder } from "@/lib/customer/types";
import { CancelOrder, DeliveryCode, OrderStages, QuoteAmounts } from "@/components/customer/orderParts";
import { statusTone } from "@/components/customer/OrderSummary";
import { NeedLogin } from "@/components/customer/NeedLogin";
import { useShopToast } from "@/components/customer/toast";

const m = getMessages(defaultLocale);
const O = m.shop.orders;
const EVERY_MS = 10_000;

function Row({ label, children }: { label: string; children: React.ReactNode }) {
  return (
    <div className="flex items-start justify-between gap-3 text-sm">
      <dt className="shrink-0 text-ink-muted">{label}</dt>
      <dd className="min-w-0 text-end">{children}</dd>
    </div>
  );
}

export default function OrderPage() {
  const params = useParams<{ id: string }>();
  const id = params?.id ?? "";
  const { user, loading } = useAuth();
  const toast = useShopToast();
  const [order, setOrder] = useState<MyOrder | null>(null);
  const [error, setError] = useState("");

  const load = useCallback(() => {
    if (!user || !id) return;
    api<MyOrder>(`/api/v1/my/orders/${id}`)
      .then((o) => {
        setOrder(o);
        setError("");
      })
      .catch((err) => setError(errorText(err)));
  }, [user, id]);

  const ended = order ? ENDED.has(order.status) : false;
  useEffect(() => {
    load();
    if (ended) return;
    const t = setInterval(() => {
      if (document.visibilityState === "visible") load();
    }, EVERY_MS);
    return () => clearInterval(t);
  }, [load, ended]);

  if (loading) return <LoadingState variant="list" rows={3} />;
  if (!user) return <NeedLogin />;

  const back = (
    <Link href="/orders" className="flex w-fit items-center gap-1 text-sm text-accent-text">
      <IconPrev size={16} />
      {O.back}
    </Link>
  );

  if (!order) {
    return (
      <div className="flex flex-col gap-3">
        {back}
        {error ? <ReloadState label={error} onRetry={load} /> : <LoadingState variant="list" rows={3} />}
      </div>
    );
  }

  const custom = order.kind === "custom";

  return (
    <div className="mx-auto flex w-full max-w-2xl flex-col gap-3">
      {back}
      <div className="surface flex flex-col gap-4 p-4">
        <div className="flex items-center justify-between gap-2">
          <h1 className="heading-page tabular-nums" dir="ltr">
            #{fmtRef(order.number)}
          </h1>
          <Badge variant={statusTone(order.status)}>{statusLabel(order.status)}</Badge>
        </div>
        {!ended && (
          <>
            <OrderStages order={order} />
            <p className="text-xs text-ink-muted">{O.live}</p>
          </>
        )}
        {custom && order.custom_request && (
          <p className="text-sm">
            {O.wanted} {order.custom_request}
          </p>
        )}
        {order.cancel_reason && <Alert>{order.cancel_reason}</Alert>}
        {order.quote_rejected_at && !ended && <Alert tone="info">{O.rejectedNote}</Alert>}
      </div>

      <DeliveryCode order={order} />

      <dl className="surface flex flex-col gap-2 p-4">
        {order.driver_name && !ended && <Row label={O.driver}>{order.driver_name}</Row>}
        {order.items_preview && <Row label={fill(O.items, { n: fmtNum(order.items_count) })}>{order.items_preview}</Row>}
        <Row label={O.placed}>{fmtDateTime(order.created_at)}</Row>
        <Row label={O.payment}>{order.payment_method === "wallet" ? m.shop.cart.wallet : m.shop.cart.cash}</Row>
        {order.address_text && <Row label={O.address}>{order.address_text}</Row>}
        {order.notes && <Row label={O.notes}>{order.notes}</Row>}
      </dl>

      <dl className="surface flex flex-col gap-2 p-4">
        {custom ? (
          order.custom_goods_amount == null ? (
            <>
              <Row label={O.delivery}>{order.custom_fee != null ? <Money value={order.custom_fee} /> : O.deliveryOnDeal}</Row>
              <p className="text-xs text-ink-muted">{O.awaitingQuote}</p>
            </>
          ) : (
            <>
              <QuoteAmounts order={order} />
              {order.custom_reserved_amount > 0 && (
                <p className="text-xs text-ink-muted">{fill(O.reserved, { amount: fmtMoney(order.custom_reserved_amount) })}</p>
              )}
            </>
          )
        ) : (
          <>
            {order.subtotal > 0 && (
              <Row label={O.subtotal}>
                <Money value={order.subtotal} />
              </Row>
            )}
            <Row label={O.delivery}>
              <Money value={order.delivery_fee} />
            </Row>
            {order.discount > 0 && (
              <Row label={O.discount}>
                <span className="text-success">
                  {"− "}
                  <Money value={order.discount} />
                </span>
              </Row>
            )}
            <div className="flex justify-between border-t border-line-soft pt-2 font-bold">
              <span>{O.total}</span>
              <Money value={order.total} className="text-accent-text" />
            </div>
          </>
        )}
        {order.wallet_paid > 0 && (
          <Row label={O.walletPaid}>
            <Money value={order.wallet_paid} />
          </Row>
        )}
        {order.cash_due > 0 && !ended && (
          <Row label={O.cashDue}>
            <Money value={order.cash_due} />
          </Row>
        )}
      </dl>

      <CancelOrder
        order={order}
        onDone={() => {
          toast(O.cancelled);
          load();
        }}
      />
    </div>
  );
}
