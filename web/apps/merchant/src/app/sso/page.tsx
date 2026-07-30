"use client";

/** استلام جلسة SSO — "العودة إلى لوحتي" من تطبيق الزبون بلا كلمة مرور. */

import { Suspense, useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { authApi, tokenStore } from "@/lib/api";

const m = getMessages(defaultLocale);

export default function SSOPage() {
  return (
    <Suspense>
      <SSO />
    </Suspense>
  );
}

function SSO() {
  const router = useRouter();
  const code = useSearchParams().get("code") ?? "";
  const [failed, setFailed] = useState(false);

  useEffect(() => {
    if (!code) {
      setFailed(true);
      return;
    }
    authApi
      .sso(code)
      .then((res) => {
        tokenStore.set(res.tokens as never);
        // تحميل كامل كي تلتقط AuthProvider الجلسة من التوكن المخزّن
        window.location.replace("/");
      })
      .catch(() => setFailed(true));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [code]);

  return (
    <main className="flex min-h-screen flex-col items-center justify-center gap-3 bg-page text-center">
      {failed ? (
        <>
          <p className="text-lg font-bold text-danger">{m.errors.unauthorized}</p>
          <button
            onClick={() => router.replace("/login")}
            className="rounded-control bg-primary px-4 py-2 text-sm font-medium text-white"
          >
            {m.auth.login}
          </button>
        </>
      ) : (
        <p className="text-ink-muted">{m.common.loading}</p>
      )}
    </main>
  );
}
