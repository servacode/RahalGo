"use client";

/** بوابة المتجر — تستخدم الهيكل العائم المشترك (موحّد مع الإدارة والمندوب). */

import { useEffect } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  Badge,
  Button,
  DashboardChrome,
  type ChromeNavItem,
  IconOrder,
  IconStore,
  IconStatus,
  IconStar,
  IconSupport,
  IconUser,
  IconWarning,
} from "@rahalgo/ui";
import { PasswordGate } from "@rahalgo/auth";
import { useAuth, canAccessPortal } from "@/lib/auth";
import { StoreProvider, useStore } from "@/lib/store";
import { api, mediaUrl, tokenStore } from "@/lib/api";

const m = getMessages(defaultLocale);

const NAV: ChromeNavItem[] = [
  { href: "/portal", label: m.terms.orders, icon: IconOrder },
  { href: "/portal/menu", label: m.terms.menu, icon: IconStore },
  { href: "/portal/reports", label: m.terms.reports, icon: IconStatus },
  { href: "/portal/reviews", label: m.terms.ratings, icon: IconStar },
  { href: "/portal/complaints", label: m.terms.complaints, icon: IconSupport },
  { href: "/portal/account", label: m.terms.account, icon: IconUser },
];

function PortalChrome({ children }: { children: React.ReactNode }) {
  const { user, loading, logout } = useAuth();
  const { stores, store, loading: storesLoading, select, refresh } = useStore();
  const router = useRouter();
  const pathname = usePathname();

  useEffect(() => {
    if (!loading && !canAccessPortal(user)) router.replace("/login");
  }, [user, loading, router]);

  if (loading || storesLoading || !canAccessPortal(user)) {
    return (
      <main className="flex flex-1 items-center justify-center text-ink-muted">
        {m.common.loading}
      </main>
    );
  }

  if (!store) {
    return (
      <main className="flex flex-1 items-center justify-center p-6 text-center text-ink-muted">
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

  // خاص بالمتجر: اختيار المتجر وحالته وزر الإغلاق الطارئ — يظهر في التوب بار.
  const storeControls = (
    <div className="flex flex-wrap items-center gap-2">
      {stores.length > 1 ? (
        <select
          value={store.id}
          onChange={(e) => select(e.target.value)}
          className="max-w-40 rounded-control border border-line bg-surface px-2 py-1 text-sm font-bold"
        >
          {stores.map((s) => (
            <option key={s.id} value={s.id}>
              {s.name}
            </option>
          ))}
        </select>
      ) : (
        <span className="truncate text-sm font-bold">
          {store.category_icon} {store.name}
        </span>
      )}
      <Badge variant={store.emergency_closed ? "danger" : "success"}>
        {store.emergency_closed ? m.merchant.header.emergencyClosed : m.merchant.header.open}
      </Badge>
      <Button
        variant={store.emergency_closed ? "primary" : "danger"}
        onClick={toggleEmergency}
        className="flex items-center gap-1.5"
      >
        <IconWarning size={15} />
        <span className="hidden sm:inline">
          {store.emergency_closed ? m.merchant.header.reopen : m.merchant.header.closeNow}
        </span>
      </Button>
    </div>
  );

  return (
    <DashboardChrome
      brand={m.merchant.brand}
      nav={NAV}
      pathname={pathname}
      homeHref="/portal"
      accountHref="/portal/account"
      ratingHref="/portal/reviews"
      showRating
      api={api}
      mediaUrl={mediaUrl}
      Link={Link}
      wsUrl={`${(process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080").replace(/^http/, "ws")}/api/v1/ws`}
      token={tokenStore.access}
      notificationsHref="/portal/notifications"
      phone={user?.phone}
      shopUrl={process.env.NEXT_PUBLIC_SITE_URL ?? "http://localhost:3003"}
      shopLabel={m.shared.shopAsCustomer}
      topbarStart={storeControls}
      onLogout={() => {
        logout();
        router.replace("/login");
      }}
    >
      {children}
    </DashboardChrome>
  );
}

export default function PortalLayout({ children }: { children: React.ReactNode }) {
  // كلمة المرور المؤقتة تُبدَّل قبل أي شاشة — البوابة تحجب اللوحة حتى ذلك
  return (
    <PasswordGate>
      <StoreProvider>
        <PortalChrome>{children}</PortalChrome>
      </StoreProvider>
    </PasswordGate>
  );
}
