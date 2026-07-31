"use client";

/**
 * شاشة تسجيل الدخول الموحّدة للمنصة — الجميع يدخل من هنا (زبون/متجر/مندوب/سائق
 * وموظفو المنصة)، ويُوجَّه كلٌّ إلى لوحته حسب دوره تلقائياً. لا شاشة دخول خاصة
 * بكل لوحة. إنشاء الحساب الذاتي للزبون فقط (يتم تلقائياً عند التحقق برمز واتساب).
 */

import { Suspense } from "react";
import { useSearchParams } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { LoginCard, routeByRole, safeNext, useAuth } from "@rahalgo/auth";

const m = getMessages(defaultLocale);

export default function LoginPage() {
  return (
    <Suspense>
      <Login />
    </Suspense>
  );
}

function Login() {
  const { setUser } = useAuth();
  const next = safeNext(useSearchParams().get("next"));
  return (
    <LoginCard
      title={m.site.loginTitle}
      subtitle={m.site.loginSubtitle}
      methods="both"
      onSuccess={async (user) => {
        setUser(user);
        await routeByRole(user, next ?? undefined);
      }}
    />
  );
}
