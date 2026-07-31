"use client";

/** لا شاشة دخول خاصة بهذه اللوحة — الدخول موحّد من تطبيق المنصة، والتوجيه بالدور. */

import { useEffect } from "react";
import { APP_URLS } from "@rahalgo/auth";
import { getMessages, defaultLocale } from "@rahalgo/i18n";

const m = getMessages(defaultLocale);

export default function LoginRedirect() {
  useEffect(() => {
    window.location.replace(`${APP_URLS.customer()}/login`);
  }, []);
  return (
    <main className="flex min-h-screen items-center justify-center bg-page text-ink-muted">
      {m.common.loading}
    </main>
  );
}
