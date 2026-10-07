"use client";

/**
 * **عدّادُ زوّار الموقع** (طلبُ المالك ٢٠٢٦-١٠-٠٧)
 *
 * نداءٌ واحدٌ في جلسة المتصفّح إلى `POST /api/v1/public/visit` — **والمحرّكُ
 * يعدّ الشخصَ مرّةً في اليوم** (`site_stats.go`). **ولا يُنتظر ولا يُعرض خطؤه**:
 * عدّادٌ تعثّر لا يكسر صفحة.
 */

import { useEffect } from "react";
import { apiUrl } from "@/lib/config";

const SESSION_KEY = "rahalgo.visit.sent";

export default function VisitBeacon() {
  useEffect(() => {
    try {
      if (sessionStorage.getItem(SESSION_KEY)) return;
      sessionStorage.setItem(SESSION_KEY, "1");
    } catch {
      // @empty-ok — **تخزينٌ محجوبٌ لا يمنع العدّ**؛ والمحرّكُ يتفرّد باليوم.
    }
    const base = apiUrl();
    if (!base) return;
    fetch(`${base}/api/v1/public/visit`, { method: "POST", keepalive: true })
      // @empty-ok — **العدّادُ لا يُبلَّغ خطؤه** للزائر.
      .catch(() => undefined);
  }, []);
  return null;
}
