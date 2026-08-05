"use client";

/** المفضّلة — المكوّن المركزي المشترك (@rahalgo/ui). */

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { FavoritesPage, Button, LoadingState } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth, isLoggedIn } from "@/lib/auth";

const m = getMessages(defaultLocale);

export default function Page() {
  const { user, loading } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (!loading && !isLoggedIn(user)) router.replace("/login?next=/favorites");
  }, [user, loading, router]);

  if (loading || !isLoggedIn(user)) return <LoadingState />;

  return (
    <FavoritesPage
      api={api}
      Link={Link}
      // **وزرُّ تصفّحٍ خيرٌ من جملةٍ يائسة**: من فتح مفضّلةً فارغةً لا يعرف
      // من أين يملؤها، **وبابٌ في الشاشة نفسِها يُغني عن البحث.**
      empty={
        <Button onClick={() => router.push("/")}>{m.customer.favorites.browse}</Button>
      }
    />
  );
}
