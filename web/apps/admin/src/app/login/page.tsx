"use client";

/**
 * دخول لوحة الإدارة — معزول عمداً (صاحب المنصة وموظفوها) لكنه مبنيّ على نفس
 * البطاقة المركزية، فلا يختلف شكلاً ولا سلوكاً عن دخول بقية المنصة.
 */

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { LoginCard, tokenStore, hasRole, PANEL_ROLES, useAuth } from "@rahalgo/auth";

const m = getMessages(defaultLocale);

export default function AdminLoginPage() {
  const router = useRouter();
  const { user, loading, setUser } = useAuth();
  const signedIn = hasRole(user, ...PANEL_ROLES);

  // موظف داخل أصلاً لا يرى شاشة الدخول — يُنقل للوحته مباشرة.
  useEffect(() => {
    if (!loading && signedIn) router.replace("/dashboard");
  }, [loading, signedIn, router]);

  if (loading || signedIn) {
    return (
      <main className="flex flex-1 items-center justify-center text-ink-muted">
        {m.common.loading}
      </main>
    );
  }

  return (
    <LoginCard
      title={m.admin.loginTitle}
      subtitle={m.admin.loginSubtitle}
      methods="both"
      onSuccess={(user) => {
        // التحقق من الصلاحية قبل أي انتقال — التوكن يُمحى فوراً لغير المخوّلين.
        if (!hasRole(user, ...PANEL_ROLES)) {
          tokenStore.clear();
          throw new Error(m.admin.notAllowed);
        }
        setUser(user);
        router.replace("/dashboard");
      }}
    />
  );
}
