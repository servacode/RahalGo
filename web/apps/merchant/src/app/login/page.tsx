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
import { canAccessPortal } from "@/lib/auth";

const m = getMessages(defaultLocale);

export default function LoginPage() {
  const router = useRouter();
  return (
    <PanelLogin
      title={m.merchant.loginTitle}
      allows={canAccessPortal}
      notAllowed={m.merchant.notAllowed}
      home="/portal"
      replace={(href) => router.replace(href)}
      /* **ولكلّ شاشةٍ عنوان** — (قرارُ المالك ٢٠٢٦-٠٨-٠٦). */
      onModeChange={(mo) => router.replace(mo === "reset" ? "/forgot" : "/login")}
    />
  );
}
