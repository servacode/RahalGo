"use client";

import { useEffect, useState } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
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
  IconLogout,
  IconHamburger,
  IconClose,
  IconLink,
  IconWallet,
  IconBalance,
} from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import { useAuth, canAccessPanel } from "@/lib/auth";

const m = getMessages(defaultLocale);
const fmt = new Intl.NumberFormat("ar-SY");

interface MeSummary {
  full_name: string;
  avatar_thumb_url: string | null;
  balance: number;
}

const NAV = [
  { href: "/dashboard", label: m.admin.nav.dashboard, icon: IconDashboard },
  { href: "/dashboard/orders", label: m.admin.nav.orders, icon: IconOrder },
  { href: "/dashboard/customers", label: m.admin.nav.customers, icon: IconUser },
  { href: "/dashboard/tickets", label: m.admin.nav.tickets, icon: IconSupport },
  { href: "/dashboard/drivers", label: m.admin.nav.drivers, icon: IconDriver },
  { href: "/dashboard/sales", label: m.admin.nav.sales, icon: IconUsers },
  { href: "/dashboard/leads", label: m.admin.nav.leads, icon: IconLink },
  { href: "/dashboard/users", label: m.admin.nav.users, icon: IconUsers },
  { href: "/dashboard/merchants", label: m.admin.nav.merchants, icon: IconStore },
  { href: "/dashboard/zones", label: m.admin.nav.zones, icon: IconZones },
  { href: "/dashboard/promos", label: m.admin.nav.promos, icon: IconPromos },
  { href: "/dashboard/commissions", label: m.admin.nav.commissions, icon: IconBalance },
  { href: "/dashboard/reports", label: m.admin.nav.reports, icon: IconStatus },
  { href: "/dashboard/whatsapp", label: m.admin.nav.whatsapp, icon: IconWhatsApp },
  { href: "/dashboard/settings", label: m.admin.nav.settings, icon: IconSettings },
];

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const { user, loading, logout } = useAuth();
  const router = useRouter();
  const pathname = usePathname();
  const [menuOpen, setMenuOpen] = useState(false);
  const [summary, setSummary] = useState<MeSummary | null>(null);

  useEffect(() => {
    if (!loading && !canAccessPanel(user)) router.replace("/login");
  }, [user, loading, router]);

  useEffect(() => {
    if (canAccessPanel(user))
      api<MeSummary>("/api/v1/me/summary").then(setSummary).catch(() => undefined);
  }, [user, pathname]);

  // إغلاق القائمة المنزلقة عند تغيير الصفحة
  useEffect(() => {
    setMenuOpen(false);
  }, [pathname]);

  if (loading || !canAccessPanel(user)) {
    return (
      <main className="flex min-h-screen items-center justify-center text-ink-muted">
        {m.common.loading}
      </main>
    );
  }

  const sidebar = (
    <>
      <div className="flex items-center justify-between border-b border-line p-4">
        <div className="flex items-center gap-2">
          <div className="flex h-9 w-9 items-center justify-center rounded-control bg-primary font-bold text-white">
            ر
          </div>
          <span className="font-bold">{m.common.appName}</span>
        </div>
        <button
          onClick={() => setMenuOpen(false)}
          className="text-ink-muted hover:text-ink lg:hidden"
          aria-label={m.common.cancel}
        >
          <IconClose size={20} />
        </button>
      </div>
      <nav className="flex-1 space-y-1 overflow-y-auto p-3">
        {NAV.map((item) => {
          const active =
            item.href === "/dashboard" ? pathname === item.href : pathname.startsWith(item.href);
          return (
            <Link
              key={item.href}
              href={item.href}
              className={`flex items-center gap-2.5 rounded-control px-3 py-2 text-sm transition-colors ${
                active
                  ? "bg-primary-light font-medium text-primary-dark"
                  : "text-ink-muted hover:bg-page hover:text-ink"
              }`}
            >
              <item.icon size={17} strokeWidth={active ? 2.2 : 1.8} />
              {item.label}
            </Link>
          );
        })}
      </nav>
    </>
  );

  const activeLabel =
    NAV.find((item) =>
      item.href === "/dashboard" ? pathname === item.href : pathname.startsWith(item.href)
    )?.label ?? m.admin.nav.dashboard;

  return (
    <div className="flex min-h-screen bg-page">
      {/* الشريط الجانبي العائم — شاشات كبيرة */}
      <aside className="sticky top-3 m-3 me-0 hidden h-[calc(100vh-1.5rem)] w-60 shrink-0 flex-col overflow-hidden rounded-card border border-line bg-surface shadow-sm lg:flex">
        {sidebar}
      </aside>

      {/* القائمة المنزلقة — الجوال */}
      {menuOpen && (
        <div
          className="fixed inset-0 z-40 bg-ink/40 lg:hidden"
          onClick={() => setMenuOpen(false)}
        />
      )}
      <aside
        className={`fixed inset-y-0 start-0 z-50 flex w-64 flex-col bg-surface shadow-xl transition-transform duration-200 lg:hidden ${
          menuOpen ? "translate-x-0" : "translate-x-full rtl:translate-x-full ltr:-translate-x-full"
        }`}
      >
        {sidebar}
      </aside>

      <div className="flex min-w-0 flex-1 flex-col p-3">
        {/* التوب بار العائم */}
        <header className="mb-3 flex items-center gap-3 rounded-card border border-line bg-surface px-4 py-2.5 shadow-sm">
          <button
            onClick={() => setMenuOpen(true)}
            className="text-ink-muted hover:text-ink lg:hidden"
            aria-label={m.admin.nav.dashboard}
          >
            <IconHamburger size={22} />
          </button>
          <div className="flex items-center gap-2 lg:hidden">
            <div className="flex h-7 w-7 items-center justify-center rounded-control bg-primary text-sm font-bold text-white">
              ر
            </div>
            <span className="font-bold">{m.common.appName}</span>
          </div>
          <h2 className="hidden text-sm font-bold text-ink lg:block">{activeLabel}</h2>
          <div className="ms-auto flex items-center gap-2.5">
            {/* رصيد محفظة المنصة */}
            <Link
              href="/dashboard/account"
              className="flex items-center gap-1.5 rounded-control bg-primary-light px-2.5 py-1.5 text-sm font-bold text-primary-dark hover:bg-primary-light/70"
              title={m.admin.myAccount.title}
            >
              <IconWallet size={15} />
              <span dir="ltr">{fmt.format(summary?.balance ?? 0)}</span>
              <span className="hidden text-xs font-normal sm:inline">{m.common.currency}</span>
            </Link>
            {/* الصورة الشخصية → حسابي */}
            <Link
              href="/dashboard/account"
              title={user?.phone ?? ""}
              className="flex h-8 w-8 items-center justify-center overflow-hidden rounded-full border border-line bg-primary-light text-sm font-bold text-primary-dark"
            >
              {summary?.avatar_thumb_url ? (
                // eslint-disable-next-line @next/next/no-img-element
                <img src={mediaUrl(summary.avatar_thumb_url) ?? ""} alt="" className="h-full w-full object-cover" />
              ) : (
                (summary?.full_name || user?.phone || "؟").slice(0, 1)
              )}
            </Link>
            <button
              onClick={() => {
                logout();
                router.replace("/login");
              }}
              className="flex items-center gap-1.5 rounded-control px-2 py-1.5 text-sm text-danger hover:bg-danger/10"
            >
              <IconLogout size={16} />
              <span className="hidden sm:inline">{m.auth.logout}</span>
            </button>
          </div>
        </header>

        <main className="min-w-0 flex-1 rounded-card border border-line bg-surface p-4 shadow-sm">
          {children}
        </main>
      </div>
    </div>
  );
}
