"use client";

/**
 * **«هذا القسم يحتاج حساباً»** — للزائر في السلّة والطلبات والحساب.
 *
 * **والتصفّحُ مفتوحٌ له** (مرآةُ وضع الزائر في أندرويد) — **ويُعاد بعد
 * الدخول إلى حيث كان** (`next`).
 */

import { usePathname } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { ButtonLink, EmptyState, IconUser } from "@rahalgo/ui";

const m = getMessages(defaultLocale);
const A = m.shop.account;

export function NeedLogin() {
  const path = usePathname() ?? "/shop";
  return (
    <EmptyState
      icon={IconUser}
      title={A.guestTitle}
      action={
        <div className="flex flex-col items-center gap-3">
          <p className="text-xs text-ink-muted">{A.guestHint}</p>
          <ButtonLink href={`/login?next=${encodeURIComponent(path)}`}>{A.guestEnter}</ButtonLink>
        </div>
      }
    />
  );
}
