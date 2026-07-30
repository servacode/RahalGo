"use client";

import { useEffect } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { useAuth, canAccessPanel } from "@/lib/auth";

const m = getMessages(defaultLocale);

const NAV = [
  { href: "/dashboard", label: m.admin.nav.dashboard },
  { href: "/dashboard/users", label: m.admin.nav.users },
  { href: "/dashboard/merchants", label: m.admin.nav.merchants },
  { href: "/dashboard/zones", label: m.admin.nav.zones },
  { href: "/dashboard/promos", label: m.admin.nav.promos },
  { href: "/dashboard/whatsapp", label: m.admin.nav.whatsapp },
  { href: "/dashboard/settings", label: m.admin.nav.settings },
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
    <div className="flex min-h-screen">
      <aside className="flex w-60 shrink-0 flex-col border-e border-line bg-surface">
        <div className="flex items-center gap-2 border-b border-line p-4">
          <div className="flex h-9 w-9 items-center justify-center rounded-control bg-primary font-bold text-white">
            ر
          </div>
          <span className="font-bold">{m.common.appName}</span>
        </div>
        <nav className="flex-1 space-y-1 p-3">
          {NAV.map((item) => {
            const active =
              item.href === "/dashboard" ? pathname === item.href : pathname.startsWith(item.href);
            return (
              <Link
                key={item.href}
                href={item.href}
                className={`block rounded-control px-3 py-2 text-sm transition-colors ${
                  active
                    ? "bg-primary-light font-medium text-primary-dark"
                    : "text-ink-muted hover:bg-page hover:text-ink"
                }`}
              >
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
            className="w-full rounded-control px-3 py-2 text-start text-sm text-danger hover:bg-danger/10"
          >
            {m.auth.logout}
          </button>
        </div>
      </aside>
      <main className="flex-1 p-6">{children}</main>
    </div>
  );
}
