"use client";

/**
 * **«اطلب مرة تانية»** (قرارُ المالك ٢٠٢٦-١١-١١) — للمُسلَّم العاديّ وحدَه. تعيد أصنافَه إلى السلّة
 * من كتالوج اليوم (`reorderInto`) وتفتحها؛ وما ما عاد متوفراً يُقال بجملةٍ قصيرة.
 */

import { useState } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Button } from "@rahalgo/ui";
import { useCart } from "@/lib/customer/cart";
import { reorderInto } from "@/lib/customer/reorder";
import type { MyOrder } from "@/lib/customer/types";
import { useShopToast } from "./toast";

const O = getMessages(defaultLocale).shop.orders;

export function canReorder(order: MyOrder): boolean {
  return order.status === "delivered" && order.kind !== "custom" && (order.items ?? []).length > 0;
}

export function ReorderButton({ order, className = "" }: { order: MyOrder; className?: string }) {
  const cart = useCart();
  const toast = useShopToast();
  const router = useRouter();
  const [busy, setBusy] = useState(false);
  if (!canReorder(order)) return null;

  const run = async () => {
    setBusy(true);
    try {
      const r = await reorderInto(order, (item, qty, options, drinks) => cart.add(item, qty, options, drinks));
      if (r.added === 0) {
        toast(O.reorderNone, "error");
        return;
      }
      if (r.skipped > 0) toast(O.reorderPartial, "warning");
      router.push("/cart");
    } finally {
      setBusy(false);
    }
  };

  return (
    <Button variant="secondary" className={`w-full ${className}`} onClick={run} disabled={busy}>
      {O.reorder}
    </Button>
  );
}
