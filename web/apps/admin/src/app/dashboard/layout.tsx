"use client";

/** لوحة الإدارة — تستخدم الهيكل العائم المشترك (نسخة واحدة مركزية). */

import { useEffect, useMemo } from "react";
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
  IconWarning,
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
  IconWallet,
} from "@rahalgo/ui";
import { PasswordGate } from "@rahalgo/auth";
import { api, mediaUrl, tokenStore } from "@/lib/api";
import { useAuth, canAccessPanel } from "@/lib/auth";

const m = getMessages(defaultLocale);

/**
 * القائمة الجانبية — **خريطةُ ما يملكه من يقف أمامها**، لا قائمة ثابتة.
 *
 * كانت ستّة عشر عنصراً يراها الثلاثة: المالية ترى «المناطق» و«الأكواد
 * الترويجية» و«واتساب» ولا تملك فيها زرّاً واحداً، والعمليات تفتح «الإعدادات»
 * فتجدها للقراءة. **ورؤيةُ بابٍ لا يُفتح أسوأ من عدم رؤيته**: من رآه ظنّ أن
 * له فيه شأناً فأضاع وقته يبحث عن الزرّ.
 *
 * والقاعدة مسجّلة في هذا المشروع نفسه منذ R-34 — طُبّقت على شارة المحفظة
 * ونُسيت هنا. وهذا تطبيقها حيث تنتمي أصلاً.
 */
type NavItem = ChromeNavItem & { roles?: string[] };

const ALL_NAV: NavItem[] = [
  { href: "/dashboard", label: m.terms.dashboard, icon: IconDashboard },
  // التشغيل اليومي — مشتركٌ بين الثلاثة
  { href: "/dashboard/orders", label: m.terms.orders, icon: IconOrder },
  { href: "/dashboard/customers", label: m.terms.customers, icon: IconUser },
  { href: "/dashboard/tickets", label: m.terms.complaints, icon: IconSupport },
  { href: "/dashboard/drivers", label: m.terms.drivers, icon: IconDriver },
  // **الطارئُ يبقى ظاهراً حتى يُغلقه إنسان** — والوقتُ لا يطمئنّ على أحد.
  { href: "/dashboard/emergencies", label: m.admin.nav.emergencies, icon: IconWarning },
  { href: "/dashboard/sales", label: m.terms.reps, icon: IconUsers },
  { href: "/dashboard/leads", label: m.terms.leads, icon: IconLink },
  // المال — الأدمن والمالية. والعمليات ليست طرفاً فيه.
  { href: "/dashboard/commissions", label: m.terms.commissions, icon: IconBalance,
    roles: ["admin", "finance"] },
  // **ما في الشارع مجموعاً** — مالٌ لا يُرى مجموعاً لا يُطالَب به.
  { href: "/dashboard/cash", label: m.admin.nav.cash, icon: IconWallet,
    roles: ["admin", "finance", "ops"] },
  // **ما دفعناه بسبب متجر** — والسائقُ عُوّض فوراً، والحسمُ هنا.
  { href: "/dashboard/claims", label: m.admin.nav.claims, icon: IconStore,
    roles: ["admin", "finance", "ops"] },
  // **الخسارةُ الفعلية** — لا الافتراضية التي لم تُدفع.
  { href: "/dashboard/losses", label: m.admin.nav.losses, icon: IconBalance,
    roles: ["admin", "finance"] },
  { href: "/dashboard/payouts", label: m.shared.payout.title, icon: IconWallet,
    roles: ["admin", "finance"] },
  { href: "/dashboard/audit", label: m.admin.audit.title, icon: IconStatus,
    roles: ["admin", "finance"] },
  { href: "/dashboard/reports", label: m.terms.reports, icon: IconStatus,
    roles: ["admin", "finance"] },
  // البناء — الأدمن وحده يملك أزراره
  { href: "/dashboard/users", label: m.terms.accounts, icon: IconUsers, roles: ["admin"] },
  // **أقسامُ المنصة قبل المتاجر** — الزبونُ يتصفّحها، والمتاجرُ خلفها.
  { href: "/dashboard/sections", label: m.admin.nav.sections, icon: IconStore, roles: ["admin"] },
  { href: "/dashboard/merchants", label: m.terms.merchants, icon: IconStore, roles: ["admin"] },
  { href: "/dashboard/zones", label: m.terms.zones, icon: IconZones, roles: ["admin"] },
  { href: "/dashboard/promos", label: m.terms.promos, icon: IconPromos, roles: ["admin"] },
  { href: "/dashboard/whatsapp", label: m.admin.nav.whatsapp, icon: IconWhatsApp, roles: ["admin"] },
  // الإعدادات تبقى للجميع **للقراءة**: العمليات تحتاج أن تعرف المهل التي
  // تُحاسَب عليها، وإخفاؤها يجعلها تعمل بقواعد لا تراها. والتعديل للأدمن وحده
  // ويُحرسه الخادم.
  { href: "/dashboard/settings", label: m.terms.settings, icon: IconSettings },
];

function navFor(roles: string[] | undefined): ChromeNavItem[] {
  const has = (r: string) => !!roles?.includes(r);
  // الأدمن يرى كل شيء بلا استثناء — لا حاجة لفحص كل سطر
  if (has("admin")) return ALL_NAV;
  return ALL_NAV.filter((i) => !i.roles || i.roles.some(has));
}

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const { user, loading, logout } = useAuth();
  const router = useRouter();
  const pathname = usePathname();
  const nav = useMemo(() => navFor(user?.roles), [user?.roles]);

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
    <PasswordGate>
    {/* لا walletHref: موظّفو المنصة (أدمن/عمليات/مالية) لا محافظ لهم — وشارةٌ
        برصيد صفر تشير إلى صفحة الحساب تَعِد بما لا يملكه صاحبها. الشريط خريطة ما
        يملكه المستخدم لا قائمة ثابتة (R-34). */}
    <DashboardChrome
      brand={m.common.appName}
      nav={nav}
      pathname={pathname}
      homeHref="/dashboard"
      accountHref="/dashboard/account"
      api={api}
      mediaUrl={mediaUrl}
      Link={Link}
      wsUrl={`${(process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080").replace(/^http/, "ws")}/api/v1/ws`}
      token={tokenStore.access}
      notificationsHref="/dashboard/notifications"
      phone={user?.phone}
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
