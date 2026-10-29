"use client";

/**
 * **الإشعارات — صندوقُه كما في التطبيق** (`/me/notifications`).
 *
 * **ولا إشعاراتِ دفعٍ للمتصفّح بعد** — المحرّكُ يدفع لأندرويد وآيفون وحدَهما
 * (`push.PlatformAndroid`/`PlatformIOS`)، **فيبقى الصندوقُ والجرسُ وقناةُ البثّ
 * ما دامت الصفحةُ مفتوحة.**
 */

import Link from "next/link";
import { LoadingState, NotificationsPage } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { NeedLogin } from "@/components/customer/NeedLogin";

export default function Page() {
  const { user, loading } = useAuth();
  if (loading) return <LoadingState variant="list" rows={3} />;
  if (!user) return <NeedLogin />;
  return <NotificationsPage api={api} Link={Link} app="customer" />;
}
