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
  IconStar,
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
  // **والسجلُّ بابٌ ثانٍ** — «ماذا جرى؟» سؤالٌ غيرُ «ما الذي يحتاجني الآن؟».
  { href: "/dashboard/history", label: m.admin.nav.history, icon: IconStatus },
  // **والسوقُ ثالثاً — أهمُّ ما بعد الطلبات.**
  //
  // كان في آخر القائمة مع «البناء»: الحسابات والعروض والإعدادات. **وهي أبوابٌ
  // تُفتح مرّةً وتُترك**، والسوقُ يُفتح كلَّ يوم — ما يُعرض وما نفد وما ينتظر
  // المراجعة.
  //
  // **وترتيبُ القائمة يقول ما يهمّ**: من وضع بابَه اليوميَّ في آخرها مرّر
  // عينَه على عشرة أبوابٍ ليصل إليه، **ثمّ حفظ موضعَه فصار لا يقرأ القائمة
  // أصلاً.**
  //
  // (قرارُ المالك ٢٠٢٦-٠٨-٠٤: «انقل السوق تحت سجلّ الطلبات، لأنّه أكثرُ شيءٍ
  // يهمّنا بعد الطلبات».)
  { href: "/dashboard/sections", label: m.admin.nav.sections, icon: IconStore, roles: ["admin"] },
  { href: "/dashboard/tickets", label: m.terms.complaints, icon: IconSupport },
  // **وما يقوله الناس مجموعاً** — «أيُّ سائقٍ يشكو منه الناس؟» سؤالٌ لا جوابَ
  // له إلّا بفتح عشرين ملفّاً، فلا يُفتح فلا يُعرف.
  { href: "/dashboard/ratings", label: m.admin.nav.ratings, icon: IconStar },
  // **الطارئُ يبقى ظاهراً حتى يُغلقه إنسان** — والوقتُ لا يطمئنّ على أحد.
  { href: "/dashboard/emergencies", label: m.admin.nav.emergencies, icon: IconWarning },
  { href: "/dashboard/leads", label: m.terms.leads, icon: IconLink },
  // **خزينةُ المنصة — أصلُ كلّ حركة.**
  //
  // **لا يُدفع لأحدٍ إلّا وخرج منها، ولا يدخل مالٌ إلّا ودخلها.** (قرارُ
  // المالك ٢٠٢٦-٠٨-٠٤.) وهي محفظةُ الحساب الحامل لها — فيراها صاحبُها
  // كشفاً كاملاً، **ويرى غيرُه محفظتَه هو.**
  { href: "/dashboard/wallet", label: m.admin.nav.treasury, icon: IconWallet,
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
  //
  // **وبابٌ واحدٌ لكلّ من في المنصة**: الزبائنُ والمتاجرُ والسائقون والمندوبون
  // صاروا تبويباتٍ فيه — **بجداولهم كما هي، لا بجدولٍ واحدٍ يُفقد أعمدتَهم.**
  { href: "/dashboard/users", label: m.terms.accounts, icon: IconUsers, roles: ["admin"] },
  { href: "/dashboard/promos", label: m.terms.promos, icon: IconPromos, roles: ["admin"] },
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
    {/* **وشارةُ المحفظة عادت.**

        كانت القاعدةُ: «موظّفو المنصة لا محافظ لهم — وشارةٌ برصيد صفرٍ تَعِد
        بما لا يملكه صاحبها» (R-34). **وكانت صحيحةً يومَها**: المحفظةُ تُنشأ
        عند أوّل حركة، وحسابُ الأدمن لا يقبض ولا يُخصم منه.

        **وقد بطل سببُها**: صار لكلّ حسابٍ محفظةٌ تُخلق معه، وصارت محفظةُ
        الأدمن هي خزينةَ المنصة — **فالرقمُ فيها أهمُّ رقمٍ في اللوحة**، لا
        صفراً يُخفى. (قرارُ المالك ٢٠٢٦-٠٨-٠٤.)

        **والشريطُ يبقى خريطةَ ما يملكه المستخدم** — والقاعدةُ لم تُنقض، بل
        صار المستخدمُ يملك. */}
    <DashboardChrome
      walletHref="/dashboard/wallet"
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
