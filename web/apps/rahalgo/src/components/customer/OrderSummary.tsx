"use client";

/**
 * **بطاقةُ الطلب في القائمة** — رقمُه وحالُه ومراحلُه وما فيه وإجماليُّه.
 *
 * **ولا اسمَ متجرٍ في طلب الزبون** — يُنقّى في المحرّك (`redactAllForCustomer`)،
 * **وما في الطلب نصٌّ مختصرٌ يرسله المحرّك** (`items_preview`).
 */

import Link from "next/link";
import { getMessages, defaultLocale, fmtDateTime, fmtRef } from "@rahalgo/i18n";
import { Badge, Money, IconNext } from "@rahalgo/ui";
import { statusLabel } from "@/lib/customer/texts";
import { ENDED, type MyOrder } from "@/lib/customer/types";
import { OrderEta, OrderInvoice, OrderStages } from "./orderParts";

const m = getMessages(defaultLocale);
const O = m.shop.orders;

/**
 * **لونُ الشارة يقول الحالَ قبل أن تُقرأ** — مرآةُ `statusColor` في `Status.kt` (٢٠٢٦-١٠-٢٧):
 * الانتظارُ برتقاليّ، والعملُ عليه بلون العلامة، والأخضرُ للتسليم وحدَه، والأحمرُ لما سقط.
 */
const ACTIVE = new Set(["accepted", "confirmed", "preparing", "assigned", "at_pickup", "picked_up", "on_the_way", "at_dropoff"]);

export function statusTone(status: string): "success" | "danger" | "warning" | "primary" | "neutral" {
  if (status === "delivered") return "success";
  if (status === "cancelled" || status === "rejected" || status === "failed") return "danger";
  if (status === "pending" || status === "dispatching") return "warning";
  if (ACTIVE.has(status)) return "primary";
  return "neutral";
}

export function OrderSummary({ order }: { order: MyOrder }) {
  const live = !ENDED.has(order.status);
  const href = `/orders/${order.id}`;
  /* **فاتورةٌ مفصّلةٌ في البطاقة** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — بدل «شاورما دجاج، شاورما دجاج».
     **وزرُّ «الفاتورة كاملة» لا يكون داخل رابط** — فالبطاقةُ رابطان حوله. */
  const invoice = order.kind !== "custom" && (order.items ?? []).length > 0;
  return (
    <div className="surface flex flex-col gap-3 p-4">
      <Link href={href} className="flex flex-col gap-3 transition-transform active:scale-[0.99]">
        <div className="flex items-center justify-between gap-2">
          <span className="font-bold tabular-nums" dir="ltr">
            #{fmtRef(order.number)}
          </span>
          <Badge variant={statusTone(order.status)}>{statusLabel(order.status)}</Badge>
        </div>
        {order.kind === "custom" && order.custom_request && (
          <p className="line-clamp-2 text-sm">
            {O.wanted} {order.custom_request}
          </p>
        )}
        {live && <OrderStages order={order} />}
        {live && <OrderEta order={order} />}
        {!invoice && order.items_preview && <p className="line-clamp-2 text-xs text-ink-muted">{order.items_preview}</p>}
      </Link>
      {invoice && <OrderInvoice order={order} totals />}
      <Link href={href} className="flex items-center justify-between gap-2 text-sm">
        <span className="text-xs text-ink-muted">{fmtDateTime(order.created_at)}</span>
        <span className="flex items-center gap-1 font-bold text-accent-text">
          {!invoice && <Money value={order.total} />}
          <IconNext size={16} className="text-ink-muted" />
        </span>
      </Link>
    </div>
  );
}
