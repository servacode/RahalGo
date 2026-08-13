"use client";

/** بوابة السائق — نفس الهيكل العائم المشترك، بثلاثة أقسام لا أكثر. */

import { useEffect } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  DashboardChrome,
  MobileNav,
  ThemeToggle,
  useDriverSkin,
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
  IconChat,
  IconLock,
  IconNote,
  IconPhone,
  BootScreen,
} from "@rahalgo/ui";
import { PasswordGate, FIELD_ROLES_ARE_CUSTOMERS, PANEL_PATHS } from "@rahalgo/auth";
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
const BOTTOM = ["/driver", "/driver/incoming", "/driver/history", "/driver/cash"] as const;

const NAV: ChromeNavItem[] = [
  { href: "/driver/incoming", label: m.driver.nav.incoming, icon: IconLocation },
  { href: "/driver", label: m.driver.nav.tasks, icon: IconOrder },
  // **وسجلُّه** — كان «ما انتهى اختفى»، فلا يجد طلباً يتذكّره ليُبلّغ عنه.
  { href: "/driver/history", label: m.driver.nav.history, icon: IconCheck },
  { href: "/driver/wallet", label: m.driver.nav.wallet, icon: IconWallet },
  // **ومالٌ في ذمّته يُقرأ مفصَّلاً** — لا مجموعاً في رأس الشاشة.
  { href: "/driver/cash", label: m.driver.cashbox.title, icon: IconBalance },
  // **تقييماتُه وشكاواه** — كانتا لا بابَ لهما في لوحته: الرقمُ في الشريط
  // وحدَه، **ومن اشتُكي عليه ولا يعلم لا يُصلح شيئاً.**
  // **هدفُه ومكافآتُه** — وحافزٌ لا يُرى لا يحفّز.
  { href: "/driver/incentives", label: m.driver.nav.incentives, icon: IconStar },
  { href: "/driver/reviews", label: m.driver.nav.ratings, icon: IconStar },
  { href: "/driver/complaints", label: m.driver.nav.complaints, icon: IconSupport },
  // **وسجلُّ محادثاته** — كالزبون: تُغلق بالتسليم، **والحجّةُ تُطلب بعده.**
  { href: "/driver/chats", label: m.driver.nav.chats, icon: IconChat },
  { href: "/driver/account", label: m.driver.nav.account, icon: IconUser },
  // ══════════════════════════════════════════════════════════════════════
  // **وأبوابُ المنصّة — كما في قائمة التطبيق**
  // ══════════════════════════════════════════════════════════════════════
  //
  // (أمرُ المالك ٢٠٢٦-٠٨-١٣: «يجب أن توحّد الويبَ بنفس الطريقة المتّبعة
  //  بالتطبيق، ليكون التطبيقُ والويبُ متوافقين… نفس النموذج والتسميات
  //  والشكل والأفعال والأسماء وكلّ شيء».)
  //
  // **وكانت لا بابَ لها في لوحته** — يقرؤها في التطبيق ولا يجدها في
  // الموقع، **فيُقرآن منصّتين.**
  //
  // **والتعليماتُ تعليماتُ سائقٍ لا تعليماتُ زبون**: «كيف أطلب؟ اختر
  // متجراً وأضِف إلى السلّة» تُقال لمن يشتري، **لا لمن يقود.**
  { href: "/driver/help", label: m.driver.nav.help, icon: IconSupport },
  { href: "/about", label: m.driver.nav.about, icon: IconUser },
  { href: "/contact", label: m.driver.nav.contact, icon: IconPhone },
  { href: "/terms", label: m.driver.nav.terms, icon: IconNote },
  { href: "/privacy", label: m.driver.nav.privacy, icon: IconLock },
];

export default function PortalLayout({ children }: { children: React.ReactNode }) {
  const { user, loading, logout } = useAuth();
  // ══════════════════════════════════════════════════════════════════
  // **ولوحتُه كتطبيقه — لونًا بلون**
  // ══════════════════════════════════════════════════════════════════
  //
  // (قرارُ المالك ٢٠٢٦-٠٨-١٣: «يجب أن تكون نسخةُ الويب مطابقةً لنسخة
  //  التطبيق بشكلٍ كامل، حتّى الثيمُ والستايلُ والألوان».)
  //
  // **والجلدُ على جذر الصفحة لا على حاويةٍ داخليّة** — النوافذُ تُرسم
  // خارج الشجرة، **فجلدٌ داخليٌّ لا يصلها.**
  const [theme, flipTheme] = useDriverSkin();
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
        homeHref="/driver"
        accountHref="/driver/account"
        walletHref="/driver/wallet"
        api={api}
        mediaUrl={mediaUrl}
        Link={Link}
        wsUrl={`${(process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080").replace(/^http/, "ws")}/api/v1/ws`}
        token={tokenStore.access}
        notificationsHref="/driver/notifications"
        phone={user?.phone}
        // التقييم يخصّ السائق مباشرةً — الزبون يقيّم توصيلته لا متجراً
        topExtra={
          <ThemeToggle theme={theme} onFlip={flipTheme} label={m.driver.themeToggle} />
        }
        showRating
        ratingHref="/driver/reviews"
        ratingLabel={m.terms.myRating}
        // زرّ «تسوّق» يتبع دورَ الزبون: بلا الدور لا يستطيع صاحبه أن يطلب
        shopUrl={
          /* **والسوقُ في البيت نفسِه** — مسارٌ لا عنوان. */
          FIELD_ROLES_ARE_CUSTOMERS ? PANEL_PATHS.customer : undefined
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
