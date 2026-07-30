"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuth, canAccessPortal } from "@/lib/auth";
import { getMessages, defaultLocale } from "@rahalgo/i18n";

const m = getMessages(defaultLocale);

export default function Home() {
  const { user, loading } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (loading) return;
    router.replace(canAccessPortal(user) ? "/portal" : "/login");
  }, [user, loading, router]);

  return (
    <main className="flex min-h-screen items-center justify-center text-ink-muted">
      {m.common.loading}
    </main>
  );
}
