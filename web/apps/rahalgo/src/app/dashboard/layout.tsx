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
  IconWarning,
  IconUsers,
  IconStore,
  IconZones,
  IconRoute,
  IconLocation,
  IconPromos,
  IconSupport,
  IconSettings,
  IconWallet,
  IconBalance,
  IconHistory,
  IconReports,
  IconMonitor,
  IconMoto,
  IconArrowOut,
  IconTrendUp,
  IconReceipt,
  IconTarget,
  IconSignup,
  BootScreen,
  IconShieldCheck,
  wsBase,
  useLiveData,
  AdminLinkProvider,
} from "@rahalgo/ui";
import { PasswordGate } from "@rahalgo/auth";
import { api, mediaUrl, tokenStore } from "@/lib/api";
import { useAuth, canAccessPanel } from "@/lib/auth";
import { EmergencyBanner } from "@/components/admin/EmergencyBanner";
import { OutageBanner } from "@/components/admin/OutageBanner";
import { AdminSearch } from "@/components/admin/AdminSearch";
import { MenuCountsProvider, useMenuCounts } from "@/components/admin/MenuCounts";

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
/** **المجموعاتُ الخمس** (قرارُ المالك ٢٠٢٦-١١-٠٣) — عنوانُها يركب أوّلَ بندٍ ظاهرٍ منها (`navFor`). */
type NavSection = "daily" | "people" | "money" | "growth" | "system";
type NavItem = ChromeNavItem & { roles?: string[]; caps?: string[]; section: NavSection };

const SECTION_TITLE: Record<NavSection, string> = {
  daily: m.admin.kit.nav.groupDaily,
  people: m.admin.kit.nav.groupPeople,
  money: m.admin.kit.nav.groupMoney,
  growth: m.admin.kit.nav.groupGrowth,
  system: m.admin.kit.nav.groupSystem,
};

/**
 * **من يرى الرئيسيّةَ**: كلُّ من يملك كتلةً فيها (انتباه · نبض · مال) — لا مديرُ المنصّة وحدَه.
 * (كان القرار ٢٠٢٦-١٠-٠٤ «الرئيسيّةُ للمدير وحدَه» — **نُسخ بإعادة تصميم اللوحة ٢٠٢٦-١١-٠٣**: لكلّ دورٍ
 * رئيسيّتُه بكتله فقط، وكلُّ كتلةٍ بقدرة بياناتها.)
 */
const HOME_CAPS = [
  "platform.overview",
  "orders.read",
  "finance.read",
  "support.manage",
  "emergencies.manage",
  "merchants.verify",
];

/**
 * **خريطةُ ما يملكه من يقف أمامها، في خمس مجموعات** — الشغلُ اليوميّ · الناسُ والمتاجر · المال · النموّ · النظام.
 *
 * **وبابٌ بقدرته لا باسم دور** (٢٠٢٦-٠٩-١٣): دورٌ جديدٌ يحمل قدرةً يرى بابَها بلا أن يُكتب اسمُه هنا.
 * **ورؤيةُ بابٍ لا يُفتح أسوأُ من عدم رؤيته** (`R-34`) — والمحرّكُ هو الحارس، وهذه عينٌ لا يد (`ADG-2`).
 *
 * **والتفاصيلُ بلا بندٍ خاصٍّ** تلوّن أمَّها بـ`activeAlso`: ملفُّ الحساب والمتجر للحسابات، التعويضاتُ والخسائرُ
 * للمشاكل، والأدوارُ للإعدادات. **والإشعاراتُ في الجرس** لا في القائمة.
 */
const ALL_NAV: NavItem[] = [
  // ── الشغل اليومي ──
  { section: "daily", href: "/dashboard", label: m.terms.dashboard, icon: IconDashboard, caps: HOME_CAPS },
  { section: "daily", href: "/dashboard/orders", label: m.terms.orders, icon: IconOrder, caps: ["orders.read"] },
  // **والسجلُّ بابٌ ثانٍ** — «ماذا جرى؟» سؤالٌ غيرُ «ما الذي يحتاجني الآن؟».
  { section: "daily", href: "/dashboard/history", label: m.admin.nav.history, icon: IconHistory, caps: ["orders.read"] },
  // **خريطةُ العمليات — «أين» لا «كم»**، والصلاحيةُ في الخادم (`opsmap.Perm`).
  { section: "daily", href: "/dashboard/opsmap", label: m.admin.nav.opsMap, icon: IconZones, caps: ["orders.read"] },
  // **مراقبةُ حركة السائقين** (قرارُ المالك ٢٠٢٦-١١-٠٣) — بقدرتها وحدَها، **أثرٌ شخصيٌّ حسّاس**.
  { section: "daily", href: "/dashboard/driver-monitor", label: m.admin.nav.driverMonitor, icon: IconRoute, caps: ["drivers.monitor"] },
  // **الخريطة من السائقين** (قرارُ المالك ٢٠٢٦-١١-٠٣) — كثافةُ المشي وشوارعُ وأماكنُ مرشَّحة للموافقة.
  { section: "daily", href: "/dashboard/map-learning", label: m.admin.nav.mapLearning, icon: IconRoute, caps: ["orders.read"] },
  // **«المشاكل والتعويضات»** (قرارُ المالك ٢٠٢٦-١٠-٢٥) — الشكاوى والتقييماتُ والخسائرُ والنزاعاتُ وتعويضُ البضاعة.
  {
    section: "daily",
    href: "/dashboard/tickets",
    label: m.admin.nav.problems,
    icon: IconSupport,
    caps: ["support.manage", "finance.read", "disputes.manage"],
    activeAlso: ["/dashboard/compensations", "/dashboard/losses"],
  },
  // **الطارئُ يبقى ظاهراً حتى يُغلقه إنسان.**
  { section: "daily", href: "/dashboard/emergencies", label: m.admin.nav.emergencies, icon: IconWarning, caps: ["emergencies.manage"] },

  // ── الناس والمتاجر ──
  // **بابٌ واحدٌ لكلّ من في المنصة** (زبائن · متاجر · سائقون · مناديب) — **وملفُّ المتجر تابعٌ له** (`activeAlso`).
  {
    section: "people",
    href: "/dashboard/users",
    label: m.terms.accounts,
    icon: IconUsers,
    caps: ["users.read"],
    activeAlso: ["/dashboard/merchants"],
  },
  // **السوقُ بقدرته لا بقدرة المحتوى** (قرارُ المالك ٢٠٢٦-١٠-٠٥) — موظّفُ العمليّات يرتّبه.
  { section: "people", href: "/dashboard/sections", label: m.admin.nav.sections, icon: IconStore, caps: ["market.manage"] },
  { section: "people", href: "/dashboard/leads", label: m.terms.leads, icon: IconSignup, caps: ["merchants.verify"] },

  // ── المال ──
  // **خزينةُ المنصة — أصلُ كلّ حركة** (قرارُ المالك ٢٠٢٦-٠٨-٠٤). **وصفحاتُ المال بنودٌ بالقائمة** (٢٠٢٦-١٠-٢٧).
  { section: "money", href: "/dashboard/treasury", label: m.admin.treasury.title, icon: IconWallet, caps: ["finance.read"], activeAlso: ["/dashboard/wallet"] },
  { section: "money", href: "/dashboard/cash", label: m.admin.treasury.links.cash, icon: IconMoto, caps: ["finance.read"] },
  { section: "money", href: "/dashboard/payouts", label: m.admin.treasury.links.payouts, icon: IconArrowOut, caps: ["finance.read"] },
  { section: "money", href: "/dashboard/obligations", label: m.admin.treasury.links.obligations, icon: IconBalance, caps: ["finance.read"] },
  { section: "money", href: "/dashboard/profits", label: m.admin.treasury.links.profits, icon: IconTrendUp, caps: ["finance.read"] },
  { section: "money", href: "/dashboard/expenses", label: m.admin.treasury.links.expenses, icon: IconReceipt, caps: ["finance.read"] },
  { section: "money", href: "/dashboard/reports", label: m.terms.reports, icon: IconReports, caps: ["analytics.read"] },

  // ── النموّ ──
  // **كودٌ يُكتب · ولافتةٌ تُرى · وخصمٌ يُطبَّق** في صفحةٍ واحدة، **بقدرتها** (قرارُ المالك ٢٠٢٦-١٠-٠٥).
  { section: "growth", href: "/dashboard/promos", label: m.admin.promos.title, icon: IconPromos, caps: ["offers.manage"] },
  // **الأهدافُ والمكافآت** — الشاشةُ تقول من بلغ، والمكافأةُ بيدٍ لا بمعادلة.
  { section: "growth", href: "/dashboard/incentives", label: m.admin.incentives.title, icon: IconTarget, caps: ["finance.read"] },
  // **«طلباتُ التوسّع» قسمٌ مستقلّ** (قرارُ المالك ٢٠٢٦-١٠-٠٤) بـ`orders.read` كبابها في المحرّك.
  { section: "growth", href: "/dashboard/expansion", label: m.admin.nav.expansion, icon: IconLocation, caps: ["orders.read"] },

  // ── النظام ──
  // **الإعداداتُ لمن يملك تعديلَ قسمٍ منها** — والأدوارُ موضوعٌ فيها (٢٠٢٦-١٠-٢٥) فتلوّنها.
  {
    section: "system",
    href: "/dashboard/settings",
    label: m.terms.settings,
    icon: IconSettings,
    caps: ["settings.general.manage", "settings.financial.manage", "settings.security.manage"],
    activeAlso: ["/dashboard/roles"],
  },
  { section: "system", href: "/dashboard/audit", label: m.admin.audit.title, icon: IconShieldCheck, caps: ["audit.read"] },
  // **مراقبةُ التشغيل صارت صفحةَ المهندس وحدَه** (إعادةُ التصميم ٢٠٢٦-١١-٠٣): ما كان يراه موظّفُ العمليّات
  // (سيرُ الطلبات والعالق) انتقل إلى الرئيسيّة، **وبقي هنا التفصيلُ التقنيّ** بقدرة الرصد (`observability.read`).
  { section: "system", href: "/dashboard/ops", label: m.admin.ops.navTitle, icon: IconMonitor, caps: ["observability.read"] },
];

/**
 * navFor ما يراه صاحبُ هذه القدرات — **وعناوينُ المجموعات الخمس** (قرارُ المالك ٢٠٢٦-١١-٠٣،
 * **وهو ينسخ «بلا عناوين مجموعات» ٢٠٢٦-٠٨-٠٨**).
 *
 * **وعنوانُ المجموعة يركب أوّلَ بندٍ من المجموعة بعد الترشيح**: لو حُجب أوّلُ بندٍ عن دورٍ ضاع العنوانُ
 * وبقي ما تحته معلَّقاً بلا رأس — فيُلصق هنا بعد الترشيح لا قبله. **ومجموعةٌ لا يبقى منها بندٌ لا عنوانَ لها.**
 *
 * **كلُّ بابٍ بقدرته — ولا اسمَ دورٍ في شرطِ ظهور** (٢٠٢٦-٠٩-١٣).
 */
function visibleNav(caps: readonly string[]): NavItem[] {
  const can = (c: string) => caps.includes(c);
  return ALL_NAV.filter((i) => !!i.caps && i.caps.some(can));
}

function navFor(
  roles: string[] | undefined,
  caps: readonly string[],
): ChromeNavItem[] {
  void roles;
  const seen = new Set<NavSection>();
  return visibleNav(caps).map(({ section, ...item }) => {
    if (seen.has(section)) return item;
    seen.add(section);
    return { ...item, group: SECTION_TITLE[section] };
  });
}

/**
 * **بابُ الهبوط — الشغلُ الأساسيُّ قبل أوّل بندٍ بالترتيب** (تدقيقُ السايدبار ٢٠٢٦-١٠-٢٧).
 *
 * كان أوّلَ ما يملكه — **فموظّفُ الماليّة يهبط على «المشاكل والتعويضات»** لأنّها قبل الخزينة
 * بالترتيب، وشغلُه الخزينة. **فالرئيسيّةُ ثمّ الطلباتُ ثمّ الخزينة**، وإلّا أوّلُ ما يملك.
 */
const LANDING_ORDER = ["/dashboard", "/dashboard/orders", "/dashboard/treasury"];
function landingFor(nav: ChromeNavItem[]): ChromeNavItem | undefined {
  for (const href of LANDING_ORDER) {
    const hit = nav.find((i) => i.href === href);
    if (hit) return hit;
  }
  return nav[0];
}

export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const { user, capabilities, capsLoaded, loading } = useAuth();
  // **الأرقامُ لا تُطلب قبل وصول القدرات ولا لمن لا يدخل اللوحة** — فلا ضجيجَ ٤٠٣ عند الإقلاع.
  const enabled = !loading && capsLoaded && canAccessPanel(user, capabilities);
  return (
    <MenuCountsProvider enabled={enabled}>
      <DashboardShell>{children}</DashboardShell>
    </MenuCountsProvider>
  );
}

function DashboardShell({ children }: { children: React.ReactNode }) {
  const { user, capabilities, capsLoaded, loading, logout } = useAuth();
  const router = useRouter();
  const pathname = usePathname();
  const nav = useMemo(() => navFor(user?.roles, capabilities), [user?.roles, capabilities]);

  // **عدّادُ «المضافُ حديثاً» على بند السوق** (قرارُ المالك ٢٠٢٦-١٠-٠٤) —
  // ما أُضيف بعد آخر فتحٍ للسوق، **ويختفي حين يُفتح.** (إضافةٌ من دفعة السوق.)
  const canMarket = capabilities.includes("market.manage");
  const canFinance = capabilities.includes("finance.read");
  const { data: marketNew } = useLiveData<{ count: number }>(
    () => (canMarket ? api("/api/v1/admin/market/new-count") : Promise.resolve({ count: 0 })),
    ["menu", "market_seen"],
    [canMarket],
  );
  const inMarket = pathname.startsWith("/dashboard/sections");
  // **عدّاداتُ القائمة كلُّها من نداءٍ واحد** (`GET /admin/menu-counts`) — يغذّي معها شريطَ الانتباه في الرئيسيّة
  // (`MenuCountsProvider`). **والصفرُ لا يُرسم**، **والأحمرُ لما ينتظر يداً الآن**: عالقٌ، نقدٌ فوق السقف،
  // شكوى متأخرة، طارئٌ ما استلمه أحد (قرارُ المالك ٢٠٢٦-١٠-٢٧).
  const { data: counts } = useMenuCounts();
  const navShown = useMemo(() => {
    const a = counts?.attention;
    const n = (v: number | null | undefined) => v ?? 0;
    return nav.map((i) => {
      switch (i.href) {
        case "/dashboard/sections":
          return { ...i, badge: inMarket ? 0 : (marketNew?.count ?? 0) };
        case "/dashboard/orders":
          return { ...i, badge: n(a?.stuck_orders), badgeTone: n(a?.stuck_orders) > 0 ? ("danger" as const) : undefined };
        case "/dashboard/tickets":
          return {
            ...i,
            badge: n(a?.tickets_open) + n(a?.compensations_pending),
            badgeTone: n(a?.tickets_late) > 0 ? ("danger" as const) : undefined,
          };
        case "/dashboard/emergencies":
          return { ...i, badge: n(a?.emergencies_open), badgeTone: n(a?.emergencies_unacked) > 0 ? ("danger" as const) : undefined };
        case "/dashboard/leads":
          return { ...i, badge: n(a?.leads_new) };
        case "/dashboard/cash":
          return { ...i, badge: n(a?.cash_over), badgeTone: "danger" as const };
        case "/dashboard/payouts":
          return { ...i, badge: n(a?.payouts_pending) };
        case "/dashboard/obligations":
          return { ...i, badge: n(a?.obligations_overdue) + n(a?.obligation_requests) };
        case "/dashboard/expansion":
          return { ...i, badge: n(a?.expansion_waiting) };
        default:
          return i;
      }
    });
  }, [nav, marketNew, inMarket, counts]);

  useEffect(() => {
    if (!loading && capsLoaded && !canAccessPanel(user, capabilities))
      router.replace("/adminrahalgo");
  }, [user, capabilities, capsLoaded, loading, router]);

  // ══════════════════════════════════════════════════════════════════
  // **ومن هبط على بابٍ لا يملكه يُنزَل على أوّلِ ما يملك**
  // ══════════════════════════════════════════════════════════════════
  //
  // **والرئيسيّةُ صارت لكلّ دورٍ يملك كتلةً فيها** (`HOME_CAPS`) — ومن لا يملك أيَّ كتلةٍ لا بابَ له فيها،
  // **ومن هبط على بابٍ لا يملكه رأى عطباً لا شاشة.**
  //
  // **والشرطُ من القائمة لا من اسم دور**: **ما ليس في قائمته لا
  // يملكه** — **ودورٌ مخصَّصٌ يُنشَأ غداً يُنزَل على بابه بلا سطرٍ
  // يُكتب له.**
  // **وكلَّ مرّةٍ لا أوّلَها فقط**: الصفحةُ لا تُرسَم ما دام البابُ ليس له
  // (`mustLand` تحت)، فبابٌ ثانٍ يُفتح برابطٍ مباشر يُحوَّل كذلك ولا يبقى
  // على شاشة الإقلاع. **ولا حلقة**: الوجهةُ من قائمته.
  useEffect(() => {
    if (loading || !capsLoaded) return;
    const first = landingFor(nav);
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
    !!landingFor(nav) && ALL_NAV.some((i) => i.href === pathname) && !nav.some((i) => i.href === pathname);

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
        صار المستخدمُ يملك.

        **والرصيدُ يفتح كشفَ الخزينة، وبجانبه المتوقَّعُ في الدرج** (قرارُ المالك
        ٢٠٢٦-١٠-١٥) — وصفحةُ «محفظتي» أُزيلت: الخزينةُ هي محفظةُ الأدمن، وكشفُها
        في قسمها. **وشريحةُ الدرج لمن يملك `finance.read`** — يقرّرها المحرّكُ في الملخّص. */}
    <DashboardChrome
      /* **وشريحتا المال لمن يملك `finance.read` وحدَه** (تدقيقُ الشريط ٢٠٢٦-١٠-٢٧) — كانت «محفظة
         المنصة» تظهر لموظّف الدعم فيضغطها فيُردّ إلى أوّل بابه: زرٌّ ميت. **والرقمُ رصيدُ الخزينة
         نفسِها** (`treasury_balance`) لا محفظةُ الفاتح. */
      walletHref={canFinance ? "/dashboard/treasury?tab=statement" : undefined}
      drawerHref={canFinance ? "/dashboard/treasury?tab=cashbox" : undefined}
      treasuryMode
      /* **و`brand` تسميةٌ احتياطيّةٌ للوحة لا اسمُ المنصة** — تُعرض حين
         لا يكون الاسمُ مضبوطاً في الإعدادات، **وتُقرأ للقارئ الصوتيّ على
         زرّ القائمة.** (قرارُ المالك ٢٠٢٦-٠٨-٠٦.) */
      brand={m.admin.nav.dashboard}
      /* **ولا علامةَ في رأس سايدبار الإدارة** (قرارُ المالك ٢٠٢٦-٠٨-٠٨).
         من يفتحها يفتحها عشرَ مرّاتٍ في اليوم، **فالعلامةُ تقول له ما
         يعرف** وتأخذ سطراً من قائمةٍ طويلة. **وتبقى في البوّابات الأربع.** */
      showBrand={false}
      nav={navShown}
      /* **البحثُ السريع (Ctrl+K)** في شريط الصفحة — رقمُ طلب، تلفون، اسمُ متجرٍ أو شخص، أو صفحة. */
      topbarStart={<AdminSearch nav={nav} />}
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
        canOpen={capabilities.includes("observability.read")}
      />
      {/* **وهو الموضعُ الوحيدُ لشريط الطوارئ** — كان يُرسم ثانيةً في لوحة الطلبات لمن لا يملك الدعم (دفعةُ التصميم ١). */}
      {capabilities.includes("emergencies.manage") && <EmergencyBanner />}
      {/* **`data-admin` يحصر شكلَ اللوحة الجديد فيها** — `/shop` و`/store` لا يحملانه (دفعةُ التصميم ١). */}
      {/* **روابطُ العُدّة (أرقامٌ تنكبس، بطاقاتٌ تُفتح) تمرّ من `next/link`** — إعادةُ تصميم اللوحة ٢٠٢٦-١١-٠٣. */}
      <AdminLinkProvider Link={Link}>
        <div data-admin>{children}</div>
      </AdminLinkProvider>
    </DashboardChrome>
    </PasswordGate>
  );
}
