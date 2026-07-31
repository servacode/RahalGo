"use client";

/** بوابة المندوب — تستخدم الهيكل العائم المشترك (نسخة واحدة مركزية). */

import { useEffect } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  DashboardChrome,
  type ChromeNavItem,
  IconOverview,
  IconLink,
  IconStore,
  IconWallet,
  IconOrder,
  IconUser,
  IconStar,
  IconSupport,
} from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import { useAuth, isRep } from "@/lib/auth";

const m = getMessages(defaultLocale);

const NAV: ChromeNavItem[] = [
  { href: "/portal", label: m.rep.nav.overview, icon: IconOverview },
  { href: "/portal/link", label: m.rep.nav.link, icon: IconLink },
  { href: "/portal/leads", label: m.rep.nav.leads, icon: IconOrder },
  { href: "/portal/merchants", label: m.rep.nav.merchants, icon: IconStore },
  { href: "/portal/wallet", label: m.rep.nav.wallet, icon: IconWallet },
  { href: "/portal/reviews", label: m.rep.nav.reviews, icon: IconStar },
  { href: "/portal/complaints", label: m.rep.nav.complaints, icon: IconSupport },
  { href: "/portal/account", label: m.rep.nav.account, icon: IconUser },
];

export default function PortalLayout({ children }: { children: React.ReactNode }) {
  const { user, loading, logout } = useAuth();
  const router = useRouter();
  const pathname = usePathname();

  useEffect(() => {
    if (!loading && !isRep(user)) router.replace("/login");
  }, [user, loading, router]);

  if (loading || !isRep(user)) {
    return (
      <main className="flex min-h-screen items-center justify-center text-ink-muted">
        {m.common.loading}
      </main>
    );
  }

  return (
    <DashboardChrome
      brand={m.rep.loginTitle}
      nav={NAV}
      pathname={pathname}
      homeHref="/portal"
      accountHref="/portal/account"
      walletHref="/portal/wallet"
      ratingHref="/portal/reviews"
      showRating
      api={api}
      mediaUrl={mediaUrl}
      Link={Link}
      phone={user?.phone}
      shopUrl={process.env.NEXT_PUBLIC_SITE_URL ?? "http://localhost:3003"}
      shopLabel={m.rep.shopAsCustomer}
      onLogout={() => {
        logout();
        router.replace("/login");
      }}
    >
      {children}
    </DashboardChrome>
  );
}
