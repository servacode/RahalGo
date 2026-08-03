"use client";

/**
 * الإشعارات — المكوّن المركزي المشترك (@rahalgo/ui).
 *
 * **ولوحةُ الإدارة وحدَها تُرسل أيضاً.** فالإعلانُ إشعارٌ **لا واقعةَ
 * تُولّده**، وموضعُه حيث تُقرأ الإشعارات — لا في قسمٍ ثالثٍ يُبحث عنه.
 */

import Link from "next/link";
import { NotificationsPage } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import BroadcastPanel from "@/components/BroadcastPanel";

export default function Page() {
  const { user } = useAuth();
  const isAdmin = !!user?.roles.includes("admin");
  return (
    <div className="space-y-6">
      {/* **صوتُ المنصة لا يُعار** — الأدمنُ وحدَه يراه، والخادمُ يحرسه. */}
      {isAdmin && <BroadcastPanel />}
      <NotificationsPage api={api} Link={Link} />
    </div>
  );
}
