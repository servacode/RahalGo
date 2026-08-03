"use client";

/**
 * **سجلُّ الطلبات** — ما انتهى أمرُه.
 *
 * يُفتح عند سؤال: **«ماذا جرى لهذا الطلب؟»** — شكوى، أو خلافٌ على مال، أو
 * مراجعةُ متجرٍ يُكثر الرفض. **وله وحدَه مُرشِّحُ الحالة**، لأنّ سؤالَه يبدأ
 * من نوع النهاية لا من رقم الطلب.
 */

import { Suspense } from "react";
import OrdersScreen from "@/components/orders/OrdersScreen";

export default function HistoryPage() {
  return (
    <Suspense>
      <OrdersScreen mode="history" />
    </Suspense>
  );
}
