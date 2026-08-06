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
import { isRep } from "@/lib/auth";

const m = getMessages(defaultLocale);

export default function ForgotPage() {
  const router = useRouter();
  return (
    <PanelLogin
      initialMode="reset"
      onModeChange={(mo) => router.replace(mo === "reset" ? "/forgot" : "/login")}
      title={m.rep.loginTitle}
      allows={isRep}
      notAllowed={m.rep.notAllowed}
      home="/portal"
      replace={(href) => router.replace(href)}
    />
  );
}
