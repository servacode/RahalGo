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
import { OrderStages } from "./orderParts";

const m = getMessages(defaultLocale);
const O = m.shop.orders;

export function statusTone(status: string): "success" | "danger" | "warning" | "info" {
  if (status === "delivered") return "success";
  if (["cancelled", "failed", "rejected", "refunded"].includes(status)) return "danger";
  if (status === "pending") return "warning";
  return "info";
}

export function OrderSummary({ order }: { order: MyOrder }) {
  const live = !ENDED.has(order.status);
  return (
    <Link href={`/orders/${order.id}`} className="surface flex flex-col gap-3 p-4 transition-transform active:scale-[0.99]">
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
      {order.items_preview && <p className="line-clamp-2 text-xs text-ink-muted">{order.items_preview}</p>}
      <div className="flex items-center justify-between gap-2 text-sm">
        <span className="text-xs text-ink-muted">{fmtDateTime(order.created_at)}</span>
        <span className="flex items-center gap-1 font-bold text-accent-text">
          <Money value={order.total} />
          <IconNext size={16} className="text-ink-muted" />
        </span>
      </div>
    </Link>
  );
}
