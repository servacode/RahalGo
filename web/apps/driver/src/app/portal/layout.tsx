"use client";

/** بوابة السائق — نفس الهيكل العائم المشترك، بثلاثة أقسام لا أكثر. */

import { useEffect } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  DashboardChrome,
  MobileNav,
  MobileNavSpacer,
  type ChromeNavItem,
  IconOrder,
  IconCheck,
  IconWallet,
  IconBalance,
  IconUser,
  IconLocation,
  IconStar,
  IconSupport,
  BootScreen,
} from "@rahalgo/ui";
import { PasswordGate, FIELD_ROLES_ARE_CUSTOMERS } from "@rahalgo/auth";
import { api, mediaUrl, tokenStore } from "@/lib/api";
import { useAuth, isDriver } from "@/lib/auth";

const m = getMessages(defaultLocale);

// أربعةُ أقسام: كلُّ قسمٍ زائد في تطبيقٍ يُستعمل بيدٍ واحدة ضغطةٌ ضائعة —
// **و«طلبات قادمة» ليست زائدة**: هي أوّلُ ما يفتحه السائقُ في دوامه، وقرارُ
// الأخذ يُتّخذ في ثوانٍ. (قرارُ المالك ٢٠٢٦-٠٨-٠٣)
/**
 * **وأربعةٌ منها تنزل إلى الشريط السفليّ.**
 *
 * **السائقُ أشدُّ من يحتاجه**: يمسك هاتفَه بيدٍ وهو واقفٌ في الشارع، **واليدُ
 * الأخرى على الدرّاجة** — والقائمةُ الجانبيّةُ تحتاج فتحاً ثمّ اختياراً ثمّ
 * إغلاقاً، **ثلاثُ لمساتٍ لِما يُفتح كلَّ دقيقتين.**
 *
 * **وهي الأربعةُ التي يفتحها في دوامه**: ما بيده الآن، وما يُعرض عليه،
 * وما سلّمه، وما في ذمّته من نقد. **والباقي يبقى في القائمة الجانبيّة.**
 */
const BOTTOM = ["/portal", "/portal/incoming", "/portal/history", "/portal/cash"] as const;

const NAV: ChromeNavItem[] = [
  { href: "/portal/incoming", label: m.driver.nav.incoming, icon: IconLocation },
  { href: "/portal", label: m.driver.nav.tasks, icon: IconOrder },
  // **وسجلُّه** — كان «ما انتهى اختفى»، فلا يجد طلباً يتذكّره ليُبلّغ عنه.
  { href: "/portal/history", label: m.driver.nav.history, icon: IconCheck },
  { href: "/portal/wallet", label: m.driver.nav.wallet, icon: IconWallet },
  // **ومالٌ في ذمّته يُقرأ مفصَّلاً** — لا مجموعاً في رأس الشاشة.
  { href: "/portal/cash", label: m.driver.cashbox.title, icon: IconBalance },
  // **تقييماتُه وشكاواه** — كانتا لا بابَ لهما في لوحته: الرقمُ في الشريط
  // وحدَه، **ومن اشتُكي عليه ولا يعلم لا يُصلح شيئاً.**
  // **هدفُه ومكافآتُه** — وحافزٌ لا يُرى لا يحفّز.
  { href: "/portal/incentives", label: m.driver.nav.incentives, icon: IconStar },
  { href: "/portal/reviews", label: m.terms.ratings, icon: IconStar },
  { href: "/portal/complaints", label: m.terms.complaints, icon: IconSupport },
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
      <BootScreen />
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
        ratingHref="/portal/reviews"
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
        {/* **وفراغٌ بارتفاع الشريط** — وبلاه يختفي زرُّ «سلّمت» خلفه،
            وهو آخرُ ما في الشاشة وأهمُّ ما فيها. */}
        <MobileNavSpacer />
      </DashboardChrome>
      {/* **أقسامُه الأربعةُ حيث يصل إبهامُه** — على الجوّال وحدَه. */}
      <MobileNav
        items={BOTTOM.map((h) => NAV.find((n) => n.href === h)!).filter(Boolean)}
        active={pathname}
        Link={Link}
      />
    </PasswordGate>
  );
}
