"use client";

/** استلام جلسة SSO — يفتحها الموظف من لوحته ("تسوّق كزبون") بلا كلمة مرور. */

import { Suspense, useEffect, useState } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { authApi, tokenStore } from "@/lib/api";
import { useAuth } from "@/lib/auth";

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
  const { setUser } = useAuth();
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
        tokenStore.set(res.tokens);
        setUser(res.user);
        router.replace("/");
      })
      .catch(() => setFailed(true));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [code]);

  return (
    <div className="flex min-h-[60vh] flex-col items-center justify-center gap-3 text-center">
      {failed ? (
        <>
          <p className="text-lg font-bold text-danger">{m.site.sso.failed}</p>
          <button
            onClick={() => router.replace("/login")}
            className="rounded-control bg-primary px-4 py-2 text-sm font-medium text-white"
          >
            {m.auth.login}
          </button>
        </>
      ) : (
        <p className="text-ink-muted">{m.site.sso.loading}</p>
      )}
    </div>
  );
}
