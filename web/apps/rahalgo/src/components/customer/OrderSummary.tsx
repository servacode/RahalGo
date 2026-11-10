"use client";

/**
 * **بطاقةُ الطلب في القائمة** — رقمُه وحالُه ومراحلُه وما فيه وإجماليُّه.
 *
 * **ولا اسمَ متجرٍ في طلب الزبون** — يُنقّى في المحرّك (`redactAllForCustomer`)،
 * **وما في الطلب نصٌّ مختصرٌ يرسله المحرّك** (`items_preview`).
 */

import Link from "next/link";
import { useState } from "react";
import { getMessages, defaultLocale, fmtDateTime, fmtRef, fmtNum } from "@rahalgo/i18n";
import { Badge, Money, IconNext, IconChevronDown } from "@rahalgo/ui";
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

/**
 * **عنوانُ البطاقة** (تدقيق D3) — مرآةُ `orderTitle` في `OrderCard.kt`: اسمُ المتجر إن وصل، وللخاصّ لفظُه،
 * وإلّا أوّلُ صنفٍ و«+ كذا». **والمحرّكُ اليوم لا يرسل اسمَ المتجر للزبون** (خارجَ مسموحات
 * `AudienceCustomer`) فلا يُختلَق ولا يُغيَّر العقد؛ وحين يُسمح به يظهر هنا بلا تعديلٍ ثانٍ.
 */
export function orderTitle(order: MyOrder): string {
  if (order.kind === "custom") return order.custom_mode === "amanah" ? O.titleAmanah : O.titleCustom;
  if (order.merchant_name) return order.merchant_name;
  const first = (order.items ?? [])[0]?.name || (order.items_preview ?? "").split(m.common.listSep.trim())[0]?.trim();
  if (!first) return O.titleFallback;
  const more = (order.items_count ?? 0) - 1;
  return more > 0 ? `${first} +${fmtNum(more)}` : first;
}

export function OrderSummary({ order }: { order: MyOrder }) {
  const live = !ENDED.has(order.status);
  /* **المنتهي بطاقةٌ مطويّةٌ تُفتح بلمسة** (قرارُ المالك ٢٠٢٦-١١-١١، تدقيق D3) — متجرٌ وحالٌ وإجماليٌّ وتاريخ،
     **بلا شريطِ مراحل** (لا مراحلَ لطلبٍ انتهى)؛ والجاري مفتوحٌ كما كان. */
  const [expanded, setExpanded] = useState(live);
  const collapsed = !live && !expanded;
  const href = `/orders/${order.id}`;
  /* **فاتورةٌ جدولٌ كاملٌ في البطاقة** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — بدل «شاورما دجاج، شاورما دجاج»،
     بنودُها ومجاميعُها في الأعمدة نفسِها (`OrderInvoice`). */
  const invoice = order.kind !== "custom" && (order.items ?? []).length > 0;
  if (collapsed) {
    return (
      <button
        type="button"
        onClick={() => setExpanded(true)}
        aria-expanded={false}
        aria-label={`${orderTitle(order)} — ${O.expand}`}
        className="surface flex w-full flex-col gap-2 p-4 text-start"
      >
        <span className="flex items-center justify-between gap-2">
          <span className="min-w-0 truncate text-base font-bold">{orderTitle(order)}</span>
          <span className="flex shrink-0 items-center gap-1.5 text-sm font-medium tabular-nums" dir="ltr">
            #{fmtRef(order.number)}
            <IconChevronDown size={16} className="text-ink-muted" />
          </span>
        </span>
        <span className="flex items-center justify-between gap-2">
          <Badge variant={statusTone(order.status)}>{statusLabel(order.status)}</Badge>
          <Money value={order.total} className="font-bold text-accent-text" />
        </span>
        <span className="text-xs text-ink-muted">{fmtDateTime(order.created_at)}</span>
      </button>
    );
  }
  return (
    <div className="surface flex flex-col gap-3 p-4">
      <Link href={href} className="flex flex-col gap-3 transition-transform active:scale-[0.99]">
        <div className="flex items-center justify-between gap-2">
          <span className="min-w-0 truncate text-base font-bold">{orderTitle(order)}</span>
          <span className="shrink-0 text-sm font-medium tabular-nums" dir="ltr">
            #{fmtRef(order.number)}
          </span>
        </div>
        <div className="flex items-center justify-between gap-2">
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
      {!live && (
        <button type="button" onClick={() => setExpanded(false)} className="self-start text-xs text-ink-muted underline">
          {O.collapse}
        </button>
      )}
    </div>
  );
}
