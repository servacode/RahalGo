"use client";

/**
 * **مشروباتُ المتجر للمطاعم وحدَها** (قرارُ المالك ٢٠٢٦-١٠-٣٠).
 *
 * المحرّكُ يقول `allowed` مع قائمة المشروبات؛ **وغيابُه = مسموح** (محرّكٌ أقدم).
 * و`null` = لم يُعرف بعد — فلا تظهر أداةٌ ثمّ تختفي. **وتعثّرُ النداء لا يخفيها.**
 */

import { useEffect, useState } from "react";
import { api } from "@/lib/api";

export function useDrinksAllowed(storeId: string): boolean | null {
  const [allowed, setAllowed] = useState<boolean | null>(null);
  useEffect(() => {
    if (!storeId) return;
    let live = true;
    api<{ allowed?: boolean }>(`/api/v1/merchant/stores/${storeId}/drinks`)
      .then((r) => live && setAllowed(r.allowed !== false))
      .catch(() => live && setAllowed(true));
    return () => {
      live = false;
    };
  }, [storeId]);
  return allowed;
}
