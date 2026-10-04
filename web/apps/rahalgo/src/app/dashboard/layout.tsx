"use client";

/** لوحة الإدارة — تستخدم الهيكل العائم المشترك (نسخة واحدة مركزية). */

import { useEffect, useMemo, useState } from "react";
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
  IconLocation,
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
  IconShieldCheck,
  wsBase,
  useLiveData,
} from "@rahalgo/ui";
import { PasswordGate } from "@rahalgo/auth";
import { api, mediaUrl, tokenStore } from "@/lib/api";
import { useAuth, canAccessPanel } from "@/lib/auth";
import { EmergencyBanner } from "@/components/admin/EmergencyBanner";
import { OutageBanner } from "@/components/admin/OutageBanner";

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
  // **الرئيسيّةُ لمدير المنصّة وحدَه** (قرارُ المالك ٢٠٢٦-١٠-٠٤) — فيها المالُ
  // كلُّه، **وبقيّةُ الموظّفين يهبطون على أوّل بابٍ يملكونه** كما كانوا.
  { href: "/dashboard", label: m.terms.dashboard, icon: IconDashboard,
    caps: ["platform.overview"] },
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
    caps: ["emergencies.manage"] },
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
  // **«طلباتُ التوسّع» قسمٌ مستقلّ** — «مو مخفيّة تحت الخريطة» (قرارُ المالك
  // ٢٠٢٦-١٠-٠٤). **وقراءتُها تحليليّة** كبابها في المحرّك.
  { href: "/dashboard/expansion", label: m.admin.nav.expansion, icon: IconLocation,
    caps: ["analytics.read"] },
  // **خزينةُ المنصة — أصلُ كلّ حركة.**
  //
  // **لا يُدفع لأحدٍ إلّا وخرج منها، ولا يدخل مالٌ إلّا ودخلها.** (قرارُ
  // المالك ٢٠٢٦-٠٨-٠٤.) وهي محفظةُ الحساب الحامل لها — فيراها صاحبُها
  // كشفاً كاملاً، **ويرى غيرُه محفظتَه هو.**
  //
  // **وصار قسماً موحّداً** (قرارُ المالك ٢٠٢٦-١٠-٠٤ — الخزينة): نظرةٌ عامّة · كشفٌ برصيدٍ
  // جارٍ · النقدُ والصندوق · الموافقات · صحّةُ الدفتر، **وصفحاتُ المال تُفتح منه** —
  // المصروفاتُ والأرباحُ والنقدُ والسحوباتُ والتعويضاتُ لم تعد بنوداً في القائمة.
  { href: "/dashboard/treasury", label: m.admin.treasury.title, icon: IconWallet,
    caps: ["finance.read"] },
  // **والالتزاماتُ الماليّة تُقرأ هنا** — الدَّينُ على المتاجر والمناديب
  // (`financial_obligations`): على من وكم ومن أين وكم بقي. **قراءةٌ فقط**،
  // والتسويةُ من بابها (استرداد أو نزاع).
  { href: "/dashboard/obligations", label: m.admin.nav.obligations, icon: IconBalance,
    caps: ["finance.read"] },
  // **الخسارةُ والمطالبةُ وجها واقعةٍ واحدة** — طلبٌ يفشل فيُعوَّض السائقُ
  // (خسارة) ثمّ يُفتح نزاعٌ مع المتجر (مطالبة). (قرارُ المالك ٢٠٢٦-٠٨-٠٨:
  // «النزاعات تكون مع الخسائر لأنّها هي بسبب الخسائر».)
  //
  // **والصلاحيّةُ أوسعُهما** — وتبويبُ الخسائر لا يُرسَم إلّا لمن يملكه،
  // فلا يوسّع البابُ على أحدٍ ما كان يراه.
  { href: "/dashboard/losses", label: m.admin.nav.moneyLost, icon: IconBalance,
    caps: ["finance.read", "disputes.manage"] },
  { href: "/dashboard/audit", label: m.admin.audit.title, icon: IconShieldCheck,
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
  // **وصارت شاشةَ المراقب** (قرارُ المالك ٢٠٢٦-١٠-٠٤): سيرُ الطلبات لمن يملك
  // `orders.read` — موظّفُ العمليّات — والتفاصيلُ التقنيّةُ داخلها لـ`observability.read`.
  {
    href: "/dashboard/ops",
    label: m.admin.ops.navTitle,
    icon: IconStatus,
    caps: ["orders.read", "observability.read"],
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

  // **عدّادُ «المضافُ حديثاً» على بند السوق** (قرارُ المالك ٢٠٢٦-١٠-٠٤) —
  // ما أُضيف بعد آخر فتحٍ للسوق، **ويختفي حين يُفتح.** (إضافةٌ من دفعة السوق.)
  const canMarket = capabilities.includes("content.manage");
  const { data: marketNew } = useLiveData<{ count: number }>(
    () => (canMarket ? api("/api/v1/admin/market/new-count") : Promise.resolve({ count: 0 })),
    ["menu", "market_seen"],
    [canMarket],
  );
  const inMarket = pathname.startsWith("/dashboard/sections");
  // **عدّادُ الطوارئ المفتوحة** (قرارُ المالك ٢٠٢٦-١٠-٠٤ — غرفةُ الطوارئ): **ويحمرّ** إن
  // بقي طارئٌ بلا مستلِمٍ أطولَ من مهلته (`ops.emergency_unacked_red_min`).
  const canEmergencies = capabilities.includes("emergencies.manage");
  // **والاحمرارُ بمرور الوقت لا بحدث** — فيُعاد السؤالُ كلَّ دقيقة.
  const [minuteTick, setMinuteTick] = useState(0);
  useEffect(() => {
    if (!canEmergencies) return;
    const t = window.setInterval(() => setMinuteTick((x) => x + 1), 60_000);
    return () => window.clearInterval(t);
  }, [canEmergencies]);
  const { data: emergencyCount } = useLiveData<{ open: number; stale: number }>(
    () =>
      canEmergencies
        ? api("/api/v1/admin/emergencies/count")
        : Promise.resolve({ open: 0, stale: 0 }),
    ["emergency"],
    [canEmergencies, minuteTick],
  );
  const navShown = useMemo(
    () =>
      nav.map((i) => {
        if (i.href === "/dashboard/sections")
          return { ...i, badge: inMarket ? 0 : (marketNew?.count ?? 0) };
        if (i.href === "/dashboard/emergencies")
          return {
            ...i,
            badge: emergencyCount?.open ?? 0,
            badgeTone: (emergencyCount?.stale ?? 0) > 0 ? ("danger" as const) : undefined,
          };
        return i;
      }),
    [nav, marketNew, inMarket, emergencyCount],
  );

  useEffect(() => {
    if (!loading && capsLoaded && !canAccessPanel(user, capabilities))
      router.replace("/adminrahalgo");
  }, [user, capabilities, capsLoaded, loading, router]);

  // ══════════════════════════════════════════════════════════════════
  // **ومن هبط على بابٍ لا يملكه يُنزَل على أوّلِ ما يملك**
  // ══════════════════════════════════════════════════════════════════
  //
  // **و«الرئيسيّة» تنادي `/admin/overview` فتُردّ ٤٠٣** لمن لا يملك
  // `platform.overview` — **فمن هبط عليها رأى عطباً لا شاشة.**
  //
  // **والشرطُ من القائمة لا من اسم دور**: **ما ليس في قائمته لا
  // يملكه** — **ودورٌ مخصَّصٌ يُنشَأ غداً يُنزَل على بابه بلا سطرٍ
  // يُكتب له.**
  // **وكلَّ مرّةٍ لا أوّلَها فقط**: الصفحةُ لا تُرسَم ما دام البابُ ليس له
  // (`mustLand` تحت)، فبابٌ ثانٍ يُفتح برابطٍ مباشر يُحوَّل كذلك ولا يبقى
  // على شاشة الإقلاع. **ولا حلقة**: الوجهةُ من قائمته.
  useEffect(() => {
    if (loading || !capsLoaded) return;
    const first = nav[0];
    // **والردُّ لبابٍ في القائمة لا يملكه وحدَه** (تدقيقُ اللوحة ٢٠٢٦-١٠-٠٣): كان الشرطُ «ليس في
    // قائمته» فرُدّ كلُّ بابِ تفصيلٍ — ملفُّ حساب، متجر، قسم، حسابي، الإشعارات — **إلى الرئيسيّة عند
    // أوّل فتحٍ في الجلسة**: الضغطةُ الأولى على ملفٍّ «لا تفعل شيئاً»، والتحديثُ يرمي إلى البيت.
    // **وأبوابُ التفصيل يحرسها المحرّك** (٤٠٣) كما يحرس كلَّ باب.
    if (!first || !ALL_NAV.some((i) => i.href === pathname)) return;
    if (nav.some((i) => i.href === pathname)) return;
    router.replace(first.href);
  }, [loading, capsLoaded, pathname, nav, router]);

  // **ولا حكمَ بالغياب قبل وصول القدرات** — **وإلّا رُدَّ صاحبُ
  // القدرةِ إلى الباب ثمّ أُدخِل، فيرى وميضَ رفضٍ لا معنى له.**
  // **وصفحةٌ لا يملكها الدورُ لا تُرسَم قبل التحويل** (فحصُ المتصفّح
  // ٢٠٢٦-١٠-٠٥): كانت الرئيسيةُ تُركَّب لحظةً عند المالية والدعم فتنادي
  // `overview` و`settings` فيردّها المحرّكُ ٤٠٣ — **ضجيجٌ وأخطاءُ طرفيّةٍ
  // في كلّ دخول.** فتبقى شاشةُ الإقلاع حتى يصل التحويل.
  const mustLand =
    !!nav[0] && ALL_NAV.some((i) => i.href === pathname) && !nav.some((i) => i.href === pathname);

  if (loading || !capsLoaded || !canAccessPanel(user, capabilities) || mustLand) {
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
      nav={navShown}
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
      {/* **شريطُ الطوارئ أعلى كلّ صفحة** (قرارُ المالك ٢٠٢٦-١٠-٠٤) — لمن يملكها. */}
      {/* **وشريطُ تعطّل الخادم لكلّ موظّف** (قرارُ المالك ٢٠٢٦-١٠-٠٤ — «مراقبة التشغيل»). */}
      <OutageBanner
        canOpen={capabilities.includes("orders.read") || capabilities.includes("observability.read")}
      />
      {capabilities.includes("emergencies.manage") && <EmergencyBanner />}
      {children}
    </DashboardChrome>
    </PasswordGate>
  );
}
