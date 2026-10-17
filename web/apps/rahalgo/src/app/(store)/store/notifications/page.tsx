"use client";

/**
 * **إشعاراتُ صاحب المتجر** — الصندوقُ نفسُه (`/me/notifications`) كما في
 * تطبيق المتجر (`InboxSheet`). **ولا إشعاراتِ دفعٍ للمتصفّح** — فيبقى
 * الصندوقُ والجرسُ وقناةُ البثّ ما دامت الصفحةُ مفتوحة.
 */

import Link from "next/link";
import { NotificationsPage } from "@rahalgo/ui";
import { api } from "@/lib/api";

export default function Page() {
  return <NotificationsPage api={api} Link={Link} />;
}
