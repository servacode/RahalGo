"use client";

/** لوحة الإدارة — تستخدم الهيكل العائم المشترك (نسخة واحدة مركزية). */

import { useEffect } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  DashboardChrome,
  type ChromeNavItem,
  IconDashboard,
  IconOrder,
  IconUser,
  IconDriver,
  IconUsers,
  IconStore,
  IconZones,
  IconPromos,
  IconWhatsApp,
  IconStatus,
  IconSupport,
  IconSettings,
  IconLink,
  IconBalance,
} from "@rahalgo/ui";
import { api, mediaUrl, tokenStore } from "@/lib/api";
import { useAuth, canAccessPanel } from "@/lib/auth";

const m = getMessages(defaultLocale);

const NAV: ChromeNavItem[] = [
  { href: "/dashboard", label: m.terms.dashboard, icon: IconDashboard },
  { href: "/dashboard/orders", label: m.terms.orders, icon: IconOrder },
  { href: "/dashboard/customers", label: m.terms.customers, icon: IconUser },
  { href: "/dashboard/tickets", label: m.terms.complaints, icon: IconSupport },
  { href: "/dashboard/drivers", label: m.terms.drivers, icon: IconDriver },
  { href: "/dashboard/sales", label: m.terms.reps, icon: IconUsers },
  { href: "/dashboard/leads", label: m.terms.leads, icon: IconLink },
  { href: "/dashboard/commissions", label: m.terms.commissions, icon: IconBalance },
  { href: "/dashboard/users", label: m.terms.accounts, icon: IconUsers },
  { href: "/dashboard/merchants", label: m.terms.merchants, icon: IconStore },
  { href: "/dashboard/zones", label: m.terms.zones, icon: IconZones },
  { href: "/dashboard/promos", label: m.terms.promos, icon: IconPromos },
  { href: "/dashboard/reports", label: m.terms.reports, icon: IconStatus },
  { href: "/dashboard/whatsapp", label: m.admin.nav.whatsapp, icon: IconWhatsApp },
  { href: "/dashboard/settings", label: m.terms.settings, icon: IconSettings },
];

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const { user, loading, logout } = useAuth();
  const router = useRouter();
  const pathname = usePathname();

  useEffect(() => {
    if (!loading && !canAccessPanel(user)) router.replace("/login");
  }, [user, loading, router]);

  if (loading || !canAccessPanel(user)) {
    return (
      <main className="flex flex-1 items-center justify-center text-ink-muted">
        {m.common.loading}
      </main>
    );
  }

  return (
    <DashboardChrome
      brand={m.common.appName}
      nav={NAV}
      pathname={pathname}
      homeHref="/dashboard"
      accountHref="/dashboard/account"
      walletHref="/dashboard/account"
      api={api}
      mediaUrl={mediaUrl}
      Link={Link}
      wsUrl={`${(process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080").replace(/^http/, "ws")}/api/v1/ws`}
      token={tokenStore.access}
      phone={user?.phone}
      onLogout={() => {
        logout();
        router.replace("/login");
      }}
    >
      {children}
    </DashboardChrome>
  );
}
