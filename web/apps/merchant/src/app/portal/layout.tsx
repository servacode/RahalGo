"use client";

import { useEffect } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  Badge,
  Button,
  IconOrder,
  IconStore,
  IconStatus,
  IconLogout,
  IconWarning,
} from "@rahalgo/ui";
import { useAuth, canAccessPortal } from "@/lib/auth";
import { StoreProvider, useStore } from "@/lib/store";
import { api, mediaUrl } from "@/lib/api";

const m = getMessages(defaultLocale);

const NAV = [
  { href: "/portal", label: m.merchant.nav.orders, icon: IconOrder },
  { href: "/portal/menu", label: m.merchant.nav.menu, icon: IconStore },
  { href: "/portal/reports", label: m.merchant.nav.reports, icon: IconStatus },
];

function PortalChrome({ children }: { children: React.ReactNode }) {
  const { user, loading, logout } = useAuth();
  const { stores, store, loading: storesLoading, select, refresh } = useStore();
  const router = useRouter();
  const pathname = usePathname();

  useEffect(() => {
    if (!loading && !canAccessPortal(user)) router.replace("/login");
  }, [user, loading, router]);

  // تسوّق كزبون: تسليم SSO لتطبيق الزبون بلا كلمة مرور.
  async function shopAsCustomer() {
    try {
      const { code } = await api<{ code: string }>("/api/v1/auth/handoff", { method: "POST" });
      const site = process.env.NEXT_PUBLIC_SITE_URL ?? "http://localhost:3003";
      window.location.href = `${site}/sso?code=${encodeURIComponent(code)}`;
    } catch {
      /* يبقى المستخدم في لوحته */
    }
  }

  if (loading || storesLoading || !canAccessPortal(user)) {
    return (
      <main className="flex min-h-screen items-center justify-center text-ink-muted">
        {m.common.loading}
      </main>
    );
  }

  if (!store) {
    return (
      <main className="flex min-h-screen items-center justify-center p-6 text-center text-ink-muted">
        {m.merchant.noStores}
      </main>
    );
  }

  async function toggleEmergency() {
    if (!store) return;
    await api(`/api/v1/merchant/stores/${store.id}/emergency`, {
      method: "POST",
      body: JSON.stringify({ closed: !store.emergency_closed }),
    });
    await refresh();
  }

  const logo = mediaUrl(store.logo_thumb_url);

  return (
    <div className="flex min-h-screen flex-col">
      <header className="sticky top-0 z-40 border-b border-line bg-surface">
        <div className="mx-auto flex max-w-6xl flex-wrap items-center gap-3 px-4 py-3">
          {logo ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img src={logo} alt="" className="h-10 w-10 rounded-control object-cover" />
          ) : (
            <div className="flex h-10 w-10 items-center justify-center rounded-control bg-primary font-bold text-white">
              ر
            </div>
          )}
          <div className="min-w-0">
            {stores.length > 1 ? (
              <select
                value={store.id}
                onChange={(e) => select(e.target.value)}
                className="max-w-44 rounded-control border border-line bg-surface px-2 py-1 text-sm font-bold"
              >
                {stores.map((s) => (
                  <option key={s.id} value={s.id}>
                    {s.name}
                  </option>
                ))}
              </select>
            ) : (
              <p className="truncate font-bold">
                {store.category_icon} {store.name}
              </p>
            )}
            <Badge variant={store.emergency_closed ? "danger" : "success"} className="mt-0.5">
              {store.emergency_closed ? m.merchant.header.emergencyClosed : m.merchant.header.open}
            </Badge>
          </div>

          <div className="ms-auto flex items-center gap-2">
            <Button
              variant={store.emergency_closed ? "primary" : "danger"}
              onClick={toggleEmergency}
              className="flex items-center gap-1.5"
            >
              <IconWarning size={15} />
              {store.emergency_closed ? m.merchant.header.reopen : m.merchant.header.closeNow}
            </Button>
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
        </div>

        <nav className="mx-auto flex max-w-6xl gap-1 px-4">
          {NAV.map((item) => {
            const active =
              item.href === "/portal" ? pathname === item.href : pathname.startsWith(item.href);
            return (
              <Link
                key={item.href}
                href={item.href}
                className={`flex items-center gap-1.5 border-b-2 px-3 py-2 text-sm transition-colors ${
                  active
                    ? "border-primary font-medium text-primary-dark"
                    : "border-transparent text-ink-muted hover:text-ink"
                }`}
              >
                <item.icon size={16} />
                {item.label}
              </Link>
            );
          })}
          <button
            onClick={shopAsCustomer}
            className="ms-1 flex items-center gap-1.5 rounded-control bg-accent/10 px-3 py-1.5 text-sm font-medium text-accent-dark transition-colors hover:bg-accent/20"
          >
            <IconStore size={16} />
            {m.rep.shopAsCustomer}
          </button>
        </nav>
      </header>

      <main className="mx-auto w-full max-w-6xl flex-1 p-4">{children}</main>
    </div>
  );
}

export default function PortalLayout({ children }: { children: React.ReactNode }) {
  return (
    <StoreProvider>
      <PortalChrome>{children}</PortalChrome>
    </StoreProvider>
  );
}
