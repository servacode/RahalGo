"use client";

/**
 * **رابطٌ لمن يُفتح له — ونصٌّ لغيره** (قرارُ المالك ٢٠٢٦-١٠-٠٥).
 *
 * الماليّةُ لا ترى الطلباتِ ولا الحسابات: **رابطُ «ملفّ الحساب» أو «الطلب» في
 * صفحات المال كان يفتح لها صفحةً تُردّ ٤٠٣.** والحكمُ من `useCanOpen` —
 * جدولُ المحرّك نفسُه — **لا من اسم دور.**
 */

import type { ReactNode } from "react";
import Link from "next/link";

export function OpenLink({
  allowed,
  href,
  className,
  title,
  dir,
  children,
}: {
  allowed: boolean;
  href: string;
  className?: string;
  title?: string;
  dir?: "ltr" | "rtl";
  children: ReactNode;
}) {
  if (!allowed) {
    return (
      <span dir={dir} className={className?.replace(/\bhover:underline\b/g, "").trim()}>
        {children}
      </span>
    );
  }
  return (
    <Link href={href} className={className} title={title} dir={dir}>
      {children}
    </Link>
  );
}
