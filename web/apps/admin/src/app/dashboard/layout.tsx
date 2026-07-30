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
  IconSettings,
  IconLogout,
  IconHamburger,
  IconClose,
} from "@rahalgo/ui";
import { useAuth, canAccessPanel } from "@/lib/auth";

const m = getMessages(defaultLocale);

const NAV = [
  { href: "/dashboard", label: m.admin.nav.dashboard, icon: IconDashboard },
  { href: "/dashboard/orders", label: m.admin.nav.orders, icon: IconOrder },
  { href: "/dashboard/customers", label: m.admin.nav.customers, icon: IconUser },
  { href: "/dashboard/drivers", label: m.admin.nav.drivers, icon: IconDriver },
  { href: "/dashboard/sales", label: m.admin.nav.sales, icon: IconUsers },
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
  const [menuOpen, setMenuOpen] = useState(false);

  useEffect(() => {
    if (!loading && !canAccessPanel(user)) router.replace("/login");
  }, [user, loading, router]);

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
      <div className="border-t border-line p-3">
        <p dir="ltr" className="truncate px-3 pb-2 text-end text-xs text-ink-muted">
          {user?.phone}
        </p>
        <button
          onClick={() => {
            logout();
            router.replace("/login");
          }}
          className="flex w-full items-center gap-2.5 rounded-control px-3 py-2 text-start text-sm text-danger hover:bg-danger/10"
        >
          <IconLogout size={17} strokeWidth={1.8} />
          {m.auth.logout}
        </button>
      </div>
    </>
  );

  return (
    <div className="flex min-h-screen">
      {/* الشريط الجانبي الثابت — شاشات كبيرة */}
      <aside className="hidden w-60 shrink-0 flex-col border-e border-line bg-surface lg:flex">
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

      <div className="flex min-w-0 flex-1 flex-col">
        {/* الشريط العلوي — الجوال فقط */}
        <header className="flex items-center gap-3 border-b border-line bg-surface px-4 py-3 lg:hidden">
          <button
            onClick={() => setMenuOpen(true)}
            className="text-ink-muted hover:text-ink"
            aria-label={m.admin.nav.dashboard}
          >
            <IconHamburger size={22} />
          </button>
          <div className="flex items-center gap-2">
            <div className="flex h-7 w-7 items-center justify-center rounded-control bg-primary text-sm font-bold text-white">
              ر
            </div>
            <span className="font-bold">{m.common.appName}</span>
          </div>
        </header>

        <main className="min-w-0 flex-1 p-4 lg:p-6">{children}</main>
      </div>
    </div>
  );
}
