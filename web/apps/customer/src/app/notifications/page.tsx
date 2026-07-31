"use client";

/** الإشعارات — المكوّن المركزي المشترك (@rahalgo/ui). */

import Link from "next/link";
import { NotificationsPage } from "@rahalgo/ui";
import { api } from "@/lib/api";

export default function Page() {
  return <NotificationsPage api={api} Link={Link} width="full" />;
}
