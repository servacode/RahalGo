"use client";

/** لوحة الإدارة — تستخدم الهيكل العائم المشترك (نسخة واحدة مركزية). */

import { useEffect, useMemo, useRef } from "react";
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
  IconRoles,
  wsBase,
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
// ══════════════════════════════════════════════════════════════════════
// **وبابٌ يُفتح بقدرةٍ لا باسم دور** (٢٠٢٦-٠٩-١٢)
// ══════════════════════════════════════════════════════════════════════
//
// **و`roles` باقيةٌ لما لم يُصنَّف بعد** — **و`caps` هي الوجهةُ**:
// **دورٌ جديدٌ يحمل قدرةً يرى بابَها بلا أن يُكتب اسمُه هنا.**
//
// **وذاك `ADG-1` في العرض**: اسمُ الدور حقيقةٌ ثانيةٌ في العميل تفترق
// يوماً — **وقد افترقت في الوجود من قبل.**
type NavItem = ChromeNavItem & { roles?: string[]; caps?: string[] };

const ALL_NAV: NavItem[] = [
  { href: "/dashboard", label: m.terms.dashboard, icon: IconDashboard,
    caps: ["analytics.read"] },
  // التشغيل اليومي — مشتركٌ بين الثلاثة
  { href: "/dashboard/orders", label: m.terms.orders, icon: IconOrder,
    caps: ["orders.read"] },
  // **والسجلُّ بابٌ ثانٍ** — «ماذا جرى؟» سؤالٌ غيرُ «ما الذي يحتاجني الآن؟».
  { href: "/dashboard/history", label: m.admin.nav.history, icon: IconStatus,
    caps: ["orders.read"] },
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
  { href: "/dashboard/users", label: m.terms.accounts, icon: IconUsers,
    caps: ["users.read"] },
  // **والأدوارُ تحت الحسابات** — **هي من يملك ماذا، لا من هو** (٧٠ب-و١).
  //
  // **والمنعُ في المحرّك** (`roles.manage` في جدول السياسة) — **وهذا
  // البندُ يُخفي ما لا يخصّ صاحبَه لطفاً بالعين لا حراسةً.**
  { href: "/dashboard/roles", label: m.admin.roles.navTitle, icon: IconRoles,
    caps: ["roles.manage"] },
  // **والسوقُ يليها** — ما يُعرض وما نفد وما ينتظر المراجعة.
  { href: "/dashboard/sections", label: m.admin.nav.sections, icon: IconStore,
    caps: ["content.manage"] },
  // **الشكاوى والتقييماتُ بابٌ واحد** — جوابان لسؤالٍ واحد: «ما رأيُ الناس
  // بنا؟». ومن رأى سائقاً هبط تقييمُه يقرأ شكاواه في المكان نفسِه.
  // (قرارُ المالك ٢٠٢٦-٠٨-٠٨.)
  { href: "/dashboard/tickets", label: m.admin.nav.support, icon: IconSupport,
    caps: ["support.manage"] },
  // **الطارئُ يبقى ظاهراً حتى يُغلقه إنسان** — والوقتُ لا يطمئنّ على أحد.
  { href: "/dashboard/emergencies", label: m.admin.nav.emergencies, icon: IconWarning,
    caps: ["support.manage"] },
  { href: "/dashboard/leads", label: m.terms.leads, icon: IconLink,
    caps: ["merchants.verify"] },
  // ══════════════════════════════════════════════════════════════════
  // **خريطةُ العمليات — «أين» لا «كم»**
  // ══════════════════════════════════════════════════════════════════
  //
  // **والجداولُ كلُّها تجيب «كم»** — وسؤالُ «أين السائقون الآن؟ وأيُّ
  // طلبٍ ينتظر بعيداً عنهم؟» **لا يُجاب بجدول.**
  //
  // **وموضعُها بعد التشغيل اليوميّ وقبل المال**: تُفتح مع الطلبات لا
  // مع التقارير.
  //
  // **والصلاحيّةُ في الخادم لا هنا** — `opsmap.Perm`. **وهذا سطرُ
  // رسمٍ فقط**، ومن رآه ولا يملك شيئاً فيه رأى صفحةَ «لا صلاحية».
  { href: "/dashboard/opsmap", label: m.admin.nav.opsMap, icon: IconZones,
    caps: ["orders.read"] },
  // **خزينةُ المنصة — أصلُ كلّ حركة.**
  //
  // **لا يُدفع لأحدٍ إلّا وخرج منها، ولا يدخل مالٌ إلّا ودخلها.** (قرارُ
  // المالك ٢٠٢٦-٠٨-٠٤.) وهي محفظةُ الحساب الحامل لها — فيراها صاحبُها
  // كشفاً كاملاً، **ويرى غيرُه محفظتَه هو.**
  { href: "/dashboard/wallet", label: m.admin.nav.treasury, icon: IconWallet,
    caps: ["finance.read"] },
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
    caps: ["finance.read"] },
  // **والأرباحُ تُقرأ بعدهما** — (قرارُ المالك ٢٠٢٦-٠٨-١٦): **دخلُ الطلبات
  // ناقصَ الخسائر والمصاريف والدعوات**، ولكلِّ إنسانٍ نصيبُه في تبويبه.
  { href: "/dashboard/profits", label: m.admin.nav.profits, icon: IconWallet,
    caps: ["finance.read"] },
  // **ما في الشارع مجموعاً** — مالٌ لا يُرى مجموعاً لا يُطالَب به.
  { href: "/dashboard/cash", label: m.admin.nav.cash, icon: IconWallet,
    caps: ["finance.read"] },
  // **الخسارةُ والمطالبةُ وجها واقعةٍ واحدة** — طلبٌ يفشل فيُعوَّض السائقُ
  // (خسارة) ثمّ يُفتح نزاعٌ مع المتجر (مطالبة). (قرارُ المالك ٢٠٢٦-٠٨-٠٨:
  // «النزاعات تكون مع الخسائر لأنّها هي بسبب الخسائر».)
  //
  // **والصلاحيّةُ أوسعُهما** — وتبويبُ الخسائر لا يُرسَم إلّا لمن يملكه،
  // فلا يوسّع البابُ على أحدٍ ما كان يراه.
  { href: "/dashboard/losses", label: m.admin.nav.moneyLost, icon: IconBalance,
    caps: ["finance.read", "support.manage"] },
  { href: "/dashboard/payouts", label: m.shared.payout.title, icon: IconWallet,
    caps: ["finance.read"] },
  { href: "/dashboard/audit", label: m.admin.audit.title, icon: IconStatus,
    caps: ["audit.read"] },
  { href: "/dashboard/reports", label: m.terms.reports, icon: IconStatus,
    caps: ["analytics.read"] },
  // **الأهدافُ والمكافآت** — الشاشةُ تقول من بلغ، **والمكافأةُ بيدٍ لا بمعادلة.**
  { href: "/dashboard/incentives", label: m.admin.incentives.title, icon: IconStar,
    caps: ["finance.read"] },
  // **صفحةٌ واحدةٌ لثلاثة أشكال**: كودٌ يُكتب · ولافتةٌ تُرى · وخصمٌ
  // يُطبَّق في الدفتر. **وشاشتان لغرضٍ واحدٍ تجعلان من يبحث يفتح الاثنتين.**
  { href: "/dashboard/promos", label: m.admin.promos.title, icon: IconPromos,
    caps: ["content.manage"] },
  // الإعدادات تبقى للجميع **للقراءة**: العمليات تحتاج أن تعرف المهل التي
  // تُحاسَب عليها، وإخفاؤها يجعلها تعمل بقواعد لا تراها. والتعديل للأدمن وحده
  // ويُحرسه الخادم.
  // ══════════════════════════════════════════════════════════════════
  // **مراقبةُ التشغيل — «هل النظام يعمل؟»**
  // ══════════════════════════════════════════════════════════════════
  //
  // **وبابُها القدرةُ لا اسمُ الدور** (`observability.read`): **فأيُّ
  // دورٍ يُمنَح القدرةَ غداً يرى البابَ**، ولا يُكتب اسمُه هنا.
  //
  // **وموضعُها قبل الإعدادات**: سؤالُ صحّةٍ لا سؤالُ تهيئة — **ويُفتح
  // عند الشكوى لا كلَّ يوم.**
  {
    href: "/dashboard/ops",
    label: m.admin.ops.navTitle,
    icon: IconStatus,
    caps: ["observability.read"],
  },
  { href: "/dashboard/settings", label: m.terms.settings, icon: IconSettings,
    caps: ["settings.general.manage", "settings.financial.manage", "settings.security.manage"] },
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
function navFor(
  roles: string[] | undefined,
  caps: readonly string[],
): ChromeNavItem[] {
  void roles;
  // ══════════════════════════════════════════════════════════════════
  // **كلُّ بابٍ بقدرته — ولا اسمَ دورٍ في شرطِ ظهور** (٢٠٢٦-٠٩-١٣)
  // ══════════════════════════════════════════════════════════════════
  //
  // **وكانت اثنتان وعشرون بنداً تُبوَّب بالاسم**: تسعةٌ بلا شرطٍ
  // أصلاً — **يراها كلُّ من دخل اللوحة** — **وبندا «النقد» و«الخسائر»
  // بـ`ops`**، **وهو دورٌ صفرُ حامليه في الإنتاج** (قِيس ٢٠٢٦-٠٩-١٣)،
  // **فموظّفُ `operations` لا يراهما وهما له في الورق.**
  //
  // **ودورٌ مخصَّصٌ يُنشَأ غداً بالقدرة نفسِها يرى بابَها** — **ولا
  // يُنتظَر مهندسٌ يضيف اسمَه في مصفوفة.** (وهو عقدُ المالك:
  // `NO-CODE FOR OPERATIONS`.)
  //
  // **ورؤيةُ بابٍ لا يُفتح أسوأُ من عدم رؤيته** (`R-34`) — **وقِيس
  // ثلاثةٌ منها**: سجلُّ التدقيق للماليّة، وزرُّ إغلاق التذكرة لها،
  // **وكلاهما يُردّ ٤٠٣ من المحرّك.**
  //
  // **والمحرّكُ هو الحارس** — **وهذه عينٌ لا يد** (`ADG-2`).
  const can = (c: string) => caps.includes(c);
  return ALL_NAV.filter((i) => !!i.caps && i.caps.some(can));
}


export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const { user, capabilities, capsLoaded, loading, logout } = useAuth();
  const router = useRouter();
  const pathname = usePathname();
  const nav = useMemo(() => navFor(user?.roles, capabilities), [user?.roles, capabilities]);

  useEffect(() => {
    if (!loading && capsLoaded && !canAccessPanel(user, capabilities))
      router.replace("/adminrahalgo");
  }, [user, capabilities, capsLoaded, loading, router]);

  // ══════════════════════════════════════════════════════════════════
  // **ومن هبط على بابٍ لا يملكه يُنزَل على أوّلِ ما يملك**
  // ══════════════════════════════════════════════════════════════════
  //
  // **و«الرئيسيّة» تنادي `/admin/stats` فتُردّ ٤٠٣** لمن لا يملك
  // `analytics.read` (قِيس) — **فمن هبط عليها رأى عطباً لا شاشة.**
  //
  // **والشرطُ من القائمة لا من اسم دور**: **ما ليس في قائمته لا
  // يملكه** — **ودورٌ مخصَّصٌ يُنشَأ غداً يُنزَل على بابه بلا سطرٍ
  // يُكتب له.**
  const landed = useRef(false);
  useEffect(() => {
    if (loading || !capsLoaded || landed.current) return;
    const first = nav[0];
    if (!first || nav.some((i) => i.href === pathname)) return;
    landed.current = true;
    router.replace(first.href);
  }, [loading, capsLoaded, pathname, nav, router]);

  // **ولا حكمَ بالغياب قبل وصول القدرات** — **وإلّا رُدَّ صاحبُ
  // القدرةِ إلى الباب ثمّ أُدخِل، فيرى وميضَ رفضٍ لا معنى له.**
  if (loading || !capsLoaded || !canAccessPanel(user, capabilities)) {
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
      // **وقناةُ البثّ مُشتَقّةٌ من تهيئة التشغيل** — دورةُ ٧١و:
      // **وكان العنوانُ يُخبَز وقتَ البناء.**
      wsUrl={wsBase()}
      token={tokenStore.access}
      notificationsHref="/dashboard/notifications"
      phone={user?.phone}
      /* **ولا زرَّ «تسوّق» هنا** — (قرارُ المالك ٢٠٢٦-٠٨-٣١: «احذف زرَّ
         تسوّق من لوحة الإدارة»).

         **وكان أُضيف ٢٠٢٦-٠٨-١٠** («وصاحبُ المنصّة أيضاً، والموظّفون
         أيضاً») ليرى صاحبُ المنصّة ما يراه زبائنُه. **وصار للزبون
         تطبيقُه** — والويبُ للإدارة والموظّفين وحدَهم. */
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
