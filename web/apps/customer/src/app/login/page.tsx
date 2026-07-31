"use client";

/**
 * شاشة تسجيل الدخول الموحّدة للمنصة — الجميع يدخل من هنا (زبون/متجر/مندوب/سائق
 * وموظفو المنصة)، ويُوجَّه كلٌّ إلى لوحته حسب دوره تلقائياً. لا شاشة دخول خاصة
 * بكل لوحة. إنشاء الحساب الذاتي للزبون فقط (يتم تلقائياً عند التحقق برمز واتساب).
 */

import { Suspense, useEffect, useRef } from "react";
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
  const { user, loading, setUser } = useAuth();
  const next = safeNext(useSearchParams().get("next"));
  const sent = useRef(false); // التحويل مرة واحدة — لا يتكرر مع كل إعادة رسم

  // من هو داخل أصلاً لا يرى شاشة الدخول إطلاقاً: يُنقل فوراً إلى مكانه حسب دوره.
  // (لتبديل الحساب: الخروج من الشريط العلوي ثم العودة إلى /login)
  useEffect(() => {
    if (loading || !user || sent.current) return;
    sent.current = true;
    void routeByRole(user, next ?? undefined);
  }, [user, loading, next]);

  if (loading || user) {
    return (
      <main className="flex flex-1 items-center justify-center text-ink-muted">
        {m.common.loading}
      </main>
    );
  }

  return (
    <LoginCard
      title={m.site.loginTitle}
      subtitle={m.site.loginSubtitle}
      methods="both"
      onSuccess={async (u) => {
        setUser(u);
        await routeByRole(u, next ?? undefined);
      }}
    />
  );
}
