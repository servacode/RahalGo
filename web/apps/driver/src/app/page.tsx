"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuth, isDriver } from "@/lib/auth";
import { getMessages, defaultLocale } from "@rahalgo/i18n";

const m = getMessages(defaultLocale);

export default function Home() {
  const { user, loading } = useAuth();
  const router = useRouter();
  useEffect(() => {
    if (loading) return;
    router.replace(isDriver(user) ? "/portal" : "/login");
  }, [user, loading, router]);
  return (
    <main className="flex flex-1 items-center justify-center text-ink-muted">
      {m.common.loading}
    </main>
  );
}
