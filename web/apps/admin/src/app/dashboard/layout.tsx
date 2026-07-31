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
import { api, mediaUrl } from "@/lib/api";
import { useAuth, canAccessPanel } from "@/lib/auth";

const m = getMessages(defaultLocale);

const NAV: ChromeNavItem[] = [
  { href: "/dashboard", label: m.admin.nav.dashboard, icon: IconDashboard },
  { href: "/dashboard/orders", label: m.admin.nav.orders, icon: IconOrder },
  { href: "/dashboard/customers", label: m.admin.nav.customers, icon: IconUser },
  { href: "/dashboard/tickets", label: m.admin.nav.tickets, icon: IconSupport },
  { href: "/dashboard/drivers", label: m.admin.nav.drivers, icon: IconDriver },
  { href: "/dashboard/sales", label: m.admin.nav.sales, icon: IconUsers },
  { href: "/dashboard/leads", label: m.admin.nav.leads, icon: IconLink },
  { href: "/dashboard/commissions", label: m.admin.nav.commissions, icon: IconBalance },
  { href: "/dashboard/users", label: m.admin.nav.users, icon: IconUsers },
  { href: "/dashboard/merchants", label: m.admin.nav.merchants, icon: IconStore },
  { href: "/dashboard/zones", label: m.admin.nav.zones, icon: IconZones },
  { href: "/dashboard/promos", label: m.admin.nav.promos, icon: IconPromos },
  { href: "/dashboard/reports", label: m.admin.nav.reports, icon: IconStatus },
  { href: "/dashboard/whatsapp", label: m.admin.nav.whatsapp, icon: IconWhatsApp },
  { href: "/dashboard/settings", label: m.admin.nav.settings, icon: IconSettings },
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
      <main className="flex min-h-screen items-center justify-center text-ink-muted">
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
