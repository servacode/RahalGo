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
import { api, mediaUrl, tokenStore } from "@/lib/api";
import { useAuth, isRep } from "@/lib/auth";

const m = getMessages(defaultLocale);

const NAV: ChromeNavItem[] = [
  { href: "/portal", label: m.rep.nav.overview, icon: IconOverview },
  { href: "/portal/link", label: m.rep.nav.link, icon: IconLink },
  { href: "/portal/leads", label: m.terms.leads, icon: IconOrder },
  { href: "/portal/merchants", label: m.terms.clients, icon: IconStore },
  { href: "/portal/wallet", label: m.terms.wallet, icon: IconWallet },
  { href: "/portal/reviews", label: m.terms.ratings, icon: IconStar },
  { href: "/portal/complaints", label: m.terms.complaints, icon: IconSupport },
  { href: "/portal/account", label: m.terms.account, icon: IconUser },
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
      <main className="flex flex-1 items-center justify-center text-ink-muted">
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
      wsUrl={`${(process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080").replace(/^http/, "ws")}/api/v1/ws`}
      token={tokenStore.access}
      phone={user?.phone}
      shopUrl={process.env.NEXT_PUBLIC_SITE_URL ?? "http://localhost:3003"}
      shopLabel={m.shared.shopAsCustomer}
      onLogout={() => {
        logout();
        router.replace("/login");
      }}
    >
      {children}
    </DashboardChrome>
  );
}
