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
  IconStatus,
  IconSupport,
  IconStar,
  IconSettings,
  IconLink,
  IconBalance,
  IconWallet,
  BootScreen,
} from "@rahalgo/ui";
import { PasswordGate, FIELD_ROLES_ARE_CUSTOMERS, PANEL_PATHS } from "@rahalgo/auth";
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
  // ══════════════════════════════════════════════════════════════════
  // **والحساباتُ ثالثاً — أهمُّ ما بعد الطلبات اليوم**
  // ══════════════════════════════════════════════════════════════════
  //
  // (قرارُ المالك ٢٠٢٦-٠٨-٠٨: «الحسابات انقلها فوق السوق لأنّها تهمّنا أكثرَ
  //  شيءٍ بعد الطلبات».)
  //
  // **وكان السوقُ هنا بقرارٍ سابقٍ صحيحٍ يومَه** (٢٠٢٦-٠٨-٠٤: «انقل السوق
  // تحت سجلّ الطلبات، لأنّه أكثرُ شيءٍ يهمّنا بعد الطلبات») — **والمنصّةُ
  // يومَها تعمل، واليومَ تُبنى**: تُفتح الحساباتُ كلَّ يومٍ لإنشاء مندوبٍ
  // وسائقٍ ومتجر، **والسوقُ ينتظر أصنافاً لم تُضَف بعد.**
  //
  // **وترتيبُ القائمة يقول ما يهمّ** — ومن وضع بابَه اليوميَّ في آخرها مرّر
  // عينَه على عشرة أبوابٍ ليصل إليه، **ثمّ حفظ موضعَه فصار لا يقرأ القائمة
  // أصلاً.**
  //
  // **وبابٌ واحدٌ لكلّ من في المنصة**: الزبائنُ والمتاجرُ والسائقون
  // والمندوبون تبويباتٌ فيه — بجداولهم كما هي.
  { href: "/dashboard/users", label: m.terms.accounts, icon: IconUsers, roles: ["admin"] },
  // **والسوقُ يليها** — ما يُعرض وما نفد وما ينتظر المراجعة.
  { href: "/dashboard/sections", label: m.admin.nav.sections, icon: IconStore, roles: ["admin"] },
  // **الشكاوى والتقييماتُ بابٌ واحد** — جوابان لسؤالٍ واحد: «ما رأيُ الناس
  // بنا؟». ومن رأى سائقاً هبط تقييمُه يقرأ شكاواه في المكان نفسِه.
  // (قرارُ المالك ٢٠٢٦-٠٨-٠٨.)
  { href: "/dashboard/tickets", label: m.admin.nav.support, icon: IconSupport },
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
  // ══════════════════════════════════════════════════════════════════
  // **ومصروفاتُ التشغيل تليها** — (قرارُ المالك ٢٠٢٦-٠٨-١٦)
  // ══════════════════════════════════════════════════════════════════
  //
  // **إيجارُ المكتب والرواتبُ والكهرباء** — تخرج من الخزينة، **فموضعُها
  // بعدها مباشرةً**: من قرأ رصيدَها سأل «وأين ذهب؟».
  //
  // **وليست في «الخسائر»**: الخسارةُ ما لم يكن يجب أن يقع، **وهذه كلفةُ
  // تشغيلٍ مخطَّطة** — وخلطُهما يضخّم تقريرَ الخسائر بالإيجار.
  { href: "/dashboard/expenses", label: m.admin.nav.expenses, icon: IconWallet,
    roles: ["admin", "finance"] },
  // **والأرباحُ تُقرأ بعدهما** — (قرارُ المالك ٢٠٢٦-٠٨-١٦): **دخلُ الطلبات
  // ناقصَ الخسائر والمصاريف والدعوات**، ولكلِّ إنسانٍ نصيبُه في تبويبه.
  { href: "/dashboard/profits", label: m.admin.nav.profits, icon: IconWallet,
    roles: ["admin", "finance"] },
  // **ما في الشارع مجموعاً** — مالٌ لا يُرى مجموعاً لا يُطالَب به.
  { href: "/dashboard/cash", label: m.admin.nav.cash, icon: IconWallet,
    roles: ["admin", "finance", "ops"] },
  // **الخسارةُ والمطالبةُ وجها واقعةٍ واحدة** — طلبٌ يفشل فيُعوَّض السائقُ
  // (خسارة) ثمّ يُفتح نزاعٌ مع المتجر (مطالبة). (قرارُ المالك ٢٠٢٦-٠٨-٠٨:
  // «النزاعات تكون مع الخسائر لأنّها هي بسبب الخسائر».)
  //
  // **والصلاحيّةُ أوسعُهما** — وتبويبُ الخسائر لا يُرسَم إلّا لمن يملكه،
  // فلا يوسّع البابُ على أحدٍ ما كان يراه.
  { href: "/dashboard/losses", label: m.admin.nav.moneyLost, icon: IconBalance,
    roles: ["admin", "finance", "ops"] },
  { href: "/dashboard/payouts", label: m.shared.payout.title, icon: IconWallet,
    roles: ["admin", "finance"] },
  { href: "/dashboard/audit", label: m.admin.audit.title, icon: IconStatus,
    roles: ["admin", "finance"] },
  { href: "/dashboard/reports", label: m.terms.reports, icon: IconStatus,
    roles: ["admin", "finance"] },
  // **الأهدافُ والمكافآت** — الشاشةُ تقول من بلغ، **والمكافأةُ بيدٍ لا بمعادلة.**
  { href: "/dashboard/incentives", label: m.admin.incentives.title, icon: IconStar,
    roles: ["admin", "finance"] },
  // **صفحةٌ واحدةٌ لثلاثة أشكال**: كودٌ يُكتب · ولافتةٌ تُرى · وخصمٌ
  // يُطبَّق في الدفتر. **وشاشتان لغرضٍ واحدٍ تجعلان من يبحث يفتح الاثنتين.**
  { href: "/dashboard/promos", label: m.admin.promos.title, icon: IconPromos, roles: ["admin"] },
  // الإعدادات تبقى للجميع **للقراءة**: العمليات تحتاج أن تعرف المهل التي
  // تُحاسَب عليها، وإخفاؤها يجعلها تعمل بقواعد لا تراها. والتعديل للأدمن وحده
  // ويُحرسه الخادم.
  { href: "/dashboard/settings", label: m.terms.settings, icon: IconSettings },
];

/**
 * navFor ما يراه صاحبُ هذه الأدوار.
 *
 * # ولا عناوينَ مجموعاتٍ بعد اليوم
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٨: «التشغيلُ اليوميّ · المال · البناءُ والإعداد —
 *  لا داعيَ لهما، احذفها من السايدبار».)
 *
 * **وسبعةَ عشرَ بنداً مرتّبةً لا تحتاج ثلاثةَ عناوينَ تفصلها** — والترتيبُ
 * نفسُه يقول ما يهمّ: الطلباتُ أوّلاً والإعداداتُ آخراً.
 *
 * **ومعها ذهب فخٌّ كان يلزم إغلاقه**: عنوانُ المجموعة يركب أوّلَ بندٍ منها،
 * فإن حُجب البندُ عن دورٍ **ضاع العنوانُ وبقي ما تحته معلَّقاً بلا رأس** —
 * فكان يُنقل إلى أوّل من نجا. **ولا عنوانَ اليومَ فلا فخّ**، والترشيحُ سطرٌ.
 */
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
    if (!loading && !canAccessPanel(user)) router.replace("/adminrahalgo");
  }, [user, loading, router]);

  if (loading || !canAccessPanel(user)) {
    return (
      <BootScreen />
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
      /* **و`brand` تسميةٌ احتياطيّةٌ للوحة لا اسمُ المنصة** — تُعرض حين
         لا يكون الاسمُ مضبوطاً في الإعدادات، **وتُقرأ للقارئ الصوتيّ على
         زرّ القائمة.** (قرارُ المالك ٢٠٢٦-٠٨-٠٦.) */
      brand={m.admin.nav.dashboard}
      /* **ولا علامةَ في رأس سايدبار الإدارة** (قرارُ المالك ٢٠٢٦-٠٨-٠٨).
         من يفتحها يفتحها عشرَ مرّاتٍ في اليوم، **فالعلامةُ تقول له ما
         يعرف** وتأخذ سطراً من قائمةٍ طويلة. **وتبقى في البوّابات الأربع.** */
      showBrand={false}
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
      /* **وزرُّ «تسوّق» في لوحة الإدارة أيضاً** — (قرارُ المالك ٢٠٢٦-٠٨-١٠:
         «وصاحبُ المنصّة أيضاً، والموظّفون أيضاً»).

         **كان في الثلاث ولم يكن هنا** — والسائقُ والمتجرُ والمندوبُ يتسوّقون
         وصاحبُ المنصّة لا. **ومن لا يستطيع أن يطلب من منصّته لا يرى ما يراه
         زبائنُه** — وهو أوّلُ من يجب أن يراه.

         **ويتبع دورَ الزبون**: بلا الدور يصل صاحبُه فيتصفّح ولا يستطيع أن
         يطلب — **وزرٌّ يقود إلى بابٍ لا يُفتح أسوأ من غيابه.** */
      shopUrl={
        /* **والسوقُ في البيت نفسِه** — مسارٌ لا عنوان. */
        FIELD_ROLES_ARE_CUSTOMERS ? PANEL_PATHS.customer : undefined
      }
      shopLabel={m.shared.shopAsCustomer}
      onLogout={() => {
        logout();
        router.replace("/adminrahalgo");
      }}
    >
      {children}
    </DashboardChrome>
    </PasswordGate>
  );
}
