"use client";

/** استلام جلسة SSO — نسخة واحدة يستعملها كل تطبيق (كانت مكرّرة 4 مرات). */

import { Suspense, useEffect, useState } from "react";
import { useSearchParams } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Button } from "@rahalgo/ui";
import { authApi, tokenStore } from "./client";
import { safeNext } from "./routing";

const m = getMessages(defaultLocale);

export function SsoPage({ loginPath = "/login" }: { loginPath?: string }) {
  return (
    <Suspense>
      <Sso loginPath={loginPath} />
    </Suspense>
  );
}

function Sso({ loginPath }: { loginPath: string }) {
  const params = useSearchParams();
  const code = params.get("code") ?? "";
  const next = safeNext(params.get("next")) ?? "/";
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
        // تحميل كامل كي تلتقط AuthProvider الجلسة من التوكن المخزّن
        window.location.replace(next);
      })
      .catch(() => setFailed(true));
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [code]);

  return (
    <div className="flex flex-1 flex-col items-center justify-center gap-3 text-center">
      {failed ? (
        <>
          <p className="figure text-danger">{m.shared.sso.failed}</p>
          <Button onClick={() => window.location.replace(loginPath)}>{m.auth.login}</Button>
        </>
      ) : (
        <p className="text-ink-muted">{m.shared.sso.loading}</p>
      )}
    </div>
  );
}
