"use client";

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { useAuth, isRep } from "@/lib/auth";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { BootScreen } from "@rahalgo/ui";

const m = getMessages(defaultLocale);

export default function Home() {
  const { user, loading } = useAuth();
  const router = useRouter();
  useEffect(() => {
    if (loading) return;
    router.replace(isRep(user) ? "/portal" : "/login");
  }, [user, loading, router]);
  return (
    <BootScreen />
  );
}
