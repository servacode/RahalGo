"use client";

/**
 * باب هذه اللوحة — غلافٌ رفيع حول `PanelLogin` المركزي.
 *
 * كان تحويلاً إلى بوابة الزبون، فاستحال فتحُ لوحتين بحسابين: البوابة لا تعرض
 * النموذج لمن هو داخلٌ أصلاً، والخروجُ لتفريغها يقتل الجلسة التي سلّمَت إليها.
 * والتعليل الكامل في `packages/auth/src/PanelLogin.tsx`.
 */

import { useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { PanelLogin } from "@rahalgo/auth";
import { canAccessPanel } from "@/lib/auth";

const m = getMessages(defaultLocale);

export default function LoginPage() {
  const router = useRouter();
  return (
    <PanelLogin
      title={m.admin.loginTitle}
      subtitle={m.admin.loginSubtitle}
      allows={canAccessPanel}
      notAllowed={m.admin.notAllowed}
      home="/dashboard"
      replace={(href) => router.replace(href)}
    />
  );
}
