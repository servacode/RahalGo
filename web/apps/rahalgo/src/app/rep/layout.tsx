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
  IconStar,
  IconOrder,
  IconUser,
  BootScreen,
} from "@rahalgo/ui";
import { PasswordGate, FIELD_ROLES_ARE_CUSTOMERS } from "@rahalgo/auth";
import { api, mediaUrl, tokenStore } from "@/lib/api";
import { useAuth, isRep } from "@/lib/auth";

const m = getMessages(defaultLocale);

const NAV: ChromeNavItem[] = [
  { href: "/rep", label: m.rep.nav.overview, icon: IconOverview },
  { href: "/rep/link", label: m.rep.nav.link, icon: IconLink },
  // لا قسم مستقل لطلبات الانضمام: العميل المعلّق يظهر في "عملائي" بحالته
  { href: "/rep/merchants", label: m.terms.clients, icon: IconStore },
  // **وهدفُه كهدف السائق** — المقياسُ يختلف والمعنى واحد.
  { href: "/rep/incentives", label: m.rep.nav.incentives, icon: IconStar },
  { href: "/rep/wallet", label: m.terms.wallet, icon: IconWallet },
  { href: "/rep/account", label: m.terms.account, icon: IconUser },
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
      <BootScreen />
    );
  }

  return (
    <PasswordGate>
    <DashboardChrome
      brand={m.rep.loginTitle}
      nav={NAV}
      pathname={pathname}
      homeHref="/rep"
      accountHref="/rep/account"
      walletHref="/rep/wallet"
      api={api}
      mediaUrl={mediaUrl}
      Link={Link}
      wsUrl={`${(process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080").replace(/^http/, "ws")}/api/v1/ws`}
      token={tokenStore.access}
      notificationsHref="/rep/notifications"
      phone={user?.phone}
      // زرّ «تسوّق» يتبع دورَ الزبون: بلا الدور لا يستطيع صاحبه أن يطلب
      shopUrl={
        FIELD_ROLES_ARE_CUSTOMERS
          ? (process.env.NEXT_PUBLIC_SITE_URL ?? "http://localhost:3003")
          : undefined
      }
      shopLabel={m.shared.shopAsCustomer}
      onLogout={() => {
        logout();
        router.replace("/login");
      }}
    >
      {children}
    </DashboardChrome>
    </PasswordGate>
  );
}
