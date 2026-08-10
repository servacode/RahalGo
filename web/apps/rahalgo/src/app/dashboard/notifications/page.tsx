"use client";

/**
 * الإشعارات — المكوّن المركزي المشترك (@rahalgo/ui).
 *
 * **ولا فرقَ بينها وبين أخواتها الأربع.** كان لوحُ إعلان المنصة يعلوها هنا
 * وحدَها — **فصفحةٌ واحدةٌ تُقرأ صفحتين.** (قرارُ المالك ٢٠٢٦-٠٨-٠٧: «انقله
 * إلى قسمٍ لحاله».) وموضعُه الآن `/dashboard/broadcast`.
 */

import Link from "next/link";
import { NotificationsPage } from "@rahalgo/ui";
import { api } from "@/lib/api";

export default function Page() {
  return <NotificationsPage api={api} Link={Link} />;
}
