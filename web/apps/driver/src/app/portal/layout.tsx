"use client";

/** بوابة السائق — نفس الهيكل العائم المشترك، بثلاثة أقسام لا أكثر. */

import { useEffect } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  DashboardChrome,
  type ChromeNavItem,
  IconOrder,
  IconWallet,
  IconUser,
  IconLocation,
} from "@rahalgo/ui";
import { PasswordGate, FIELD_ROLES_ARE_CUSTOMERS } from "@rahalgo/auth";
import { api, mediaUrl, tokenStore } from "@/lib/api";
import { useAuth, isDriver } from "@/lib/auth";

const m = getMessages(defaultLocale);

// أربعةُ أقسام: كلُّ قسمٍ زائد في تطبيقٍ يُستعمل بيدٍ واحدة ضغطةٌ ضائعة —
// **و«طلبات قادمة» ليست زائدة**: هي أوّلُ ما يفتحه السائقُ في دوامه، وقرارُ
// الأخذ يُتّخذ في ثوانٍ. (قرارُ المالك ٢٠٢٦-٠٨-٠٣)
const NAV: ChromeNavItem[] = [
  { href: "/portal/incoming", label: m.driver.nav.incoming, icon: IconLocation },
  { href: "/portal", label: m.driver.nav.tasks, icon: IconOrder },
  { href: "/portal/wallet", label: m.driver.nav.wallet, icon: IconWallet },
  { href: "/portal/account", label: m.driver.nav.account, icon: IconUser },
];

export default function PortalLayout({ children }: { children: React.ReactNode }) {
  const { user, loading, logout } = useAuth();
  const router = useRouter();
  const pathname = usePathname();

  useEffect(() => {
    if (!loading && !isDriver(user)) router.replace("/login");
  }, [user, loading, router]);

  if (loading || !isDriver(user)) {
    return (
      <main className="flex flex-1 items-center justify-center text-ink-muted">
        {m.common.loading}
      </main>
    );
  }

  return (
    <PasswordGate>
      <DashboardChrome
        brand={m.driver.loginTitle}
        nav={NAV}
        pathname={pathname}
        homeHref="/portal"
        accountHref="/portal/account"
        walletHref="/portal/wallet"
        api={api}
        mediaUrl={mediaUrl}
        Link={Link}
        wsUrl={`${(process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080").replace(/^http/, "ws")}/api/v1/ws`}
        token={tokenStore.access}
        notificationsHref="/portal/notifications"
        phone={user?.phone}
        // التقييم يخصّ السائق مباشرةً — الزبون يقيّم توصيلته لا متجراً
        showRating
        ratingHref="/portal/account"
        ratingLabel={m.terms.myRating}
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
