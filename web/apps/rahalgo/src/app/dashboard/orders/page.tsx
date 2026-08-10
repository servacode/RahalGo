"use client";

/**
 * **شاشةُ العمل** — الطلباتُ الجارية وحدَها.
 *
 * ما زال مفتوحاً يحتاج قراراً: قبولاً أو تحويلاً أو إسناداً أو متابعة.
 * **وما انتهى ليس عملاً — هو سجلّ**، وموضعُه `/dashboard/history`.
 */

import { Suspense } from "react";
import OrdersScreen from "@/components/admin/orders/OrdersScreen";

export default function OrdersPage() {
  return (
    <Suspense>
      <OrdersScreen mode="live" />
    </Suspense>
  );
}
