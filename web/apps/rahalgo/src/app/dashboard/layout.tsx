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
  IconArrowOut,
  IconTrendUp,
  IconTrendDown,
  IconReceipt,
  IconList,
  wsBase,
  useLiveData,
} from "@rahalgo/ui";
import { PasswordGate } from "@rahalgo/auth";
import { api, mediaUrl, tokenStore } from "@/lib/api";
import { useAuth, canAccessPanel } from "@/lib/auth";
import { EmergencyBanner } from "@/components/admin/EmergencyBanner";
import { OutageBanner } from "@/components/admin/OutageBanner";
import { GlobalSearch } from "@/components/admin/GlobalSearch";

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

// ══════════════════════════════════════════════════════════════════════
// **القائمةُ بستّ مجموعات** (تصميمُ اللوحة ٢٠٢٦-١٠-٢٥ — بقرار المالك: «نفّذ اللوحة»)
// ══════════════════════════════════════════════════════════════════════
//
// **كانت عشرين بنداً مسطّحة** وخمسُ صفحاتٍ ماليّةٍ لا تُبلغ إلّا من شريط الخزينة.
// **والآن: الرئيسيّة · الطلبات · المتاجر والسوق · الناس · المال · النظام** — ولكلّ
// صفحةٍ ماليّةٍ تُفتح كلَّ يومٍ بندُها. **وعنوانُ المجموعة يركب أوّلَ بندٍ ظاهرٍ منها**
// (`navFor`)، فلا يضيع إن حُجب أوّلُها عن دور. والقدرةُ تحكم الظهور كما كانت (`ADG-1`).
type Group = "orders" | "stores" | "people" | "money" | "system";
const GROUP_LABEL: Record<Group, string> = {
  orders: m.admin.nav.groupOrders,
  stores: m.admin.nav.groupStores,
  people: m.admin.nav.groupPeople,
  money: m.admin.nav.groupMoney,
  system: m.admin.nav.groupSystem,
};

const ALL_NAV: (NavItem & { g?: Group })[] = [
  // **الرئيسيّةُ لمدير المنصّة وحدَه** (قرارُ المالك ٢٠٢٦-١٠-٠٤).
  { href: "/dashboard", label: m.terms.dashboard, icon: IconDashboard,
    caps: ["platform.overview"] },

  // ── الطلبات — أهمُّ ما في اللوحة ──
  { g: "orders", href: "/dashboard/orders", label: m.admin.nav.liveOrders, icon: IconOrder,
    caps: ["orders.read"] },
  { g: "orders", href: "/dashboard/history", label: m.admin.nav.history, icon: IconStatus,
    caps: ["orders.read"] },
  { g: "orders", href: "/dashboard/emergencies", label: m.admin.nav.emergencies, icon: IconWarning,
    caps: ["emergencies.manage"] },
  { g: "orders", href: "/dashboard/tickets", label: m.admin.nav.support, icon: IconSupport,
    caps: ["support.manage"] },
  { g: "orders", href: "/dashboard/opsmap", label: m.admin.nav.opsMap, icon: IconZones,
    caps: ["orders.read"] },
  { g: "orders", href: "/dashboard/ops", label: m.admin.ops.navTitle, icon: IconShieldCheck,
    caps: ["orders.read", "observability.read"] },

  // ── المتاجر والسوق ──
  { g: "stores", href: "/dashboard/sections", label: m.admin.nav.marketSections, icon: IconStore,
    caps: ["market.manage"] },
  { g: "stores", href: "/dashboard/leads", label: m.terms.leads, icon: IconLink,
    caps: ["merchants.verify"] },
  { g: "stores", href: "/dashboard/promos", label: m.admin.promos.title, icon: IconPromos,
    caps: ["offers.manage"] },
  { g: "stores", href: "/dashboard/expansion", label: m.admin.nav.expansion, icon: IconLocation,
    caps: ["orders.read"] },

  // ── الناس ──
  { g: "people", href: "/dashboard/users", label: m.terms.accounts, icon: IconUsers,
    caps: ["users.read"] },
  { g: "people", href: "/dashboard/roles", label: m.admin.roles.navTitle, icon: IconRoles,
    caps: ["roles.manage"] },
  { g: "people", href: "/dashboard/incentives", label: m.admin.incentives.title, icon: IconStar,
    caps: ["finance.read"] },

  // ── المال — كلُّ صفحةٍ يوميّةٍ ببندها ──
  { g: "money", href: "/dashboard/treasury", label: m.admin.treasury.title, icon: IconWallet,
    caps: ["finance.read"] },
  { g: "money", href: "/dashboard/cash", label: m.admin.nav.driversCash, icon: IconDriver,
    caps: ["finance.read"] },
  { g: "money", href: "/dashboard/payouts", label: m.admin.nav.payoutRequests, icon: IconArrowOut,
    caps: ["finance.read"] },
  { g: "money", href: "/dashboard/obligations", label: m.admin.nav.obligations, icon: IconBalance,
    caps: ["finance.read"] },
  { g: "money", href: "/dashboard/profits", label: m.admin.treasury.links.profits, icon: IconTrendUp,
    caps: ["finance.read"] },
  { g: "money", href: "/dashboard/losses", label: m.admin.nav.moneyLost, icon: IconTrendDown,
    caps: ["finance.read", "disputes.manage"] },
  { g: "money", href: "/dashboard/reports", label: m.terms.reports, icon: IconReceipt,
    caps: ["analytics.read"] },

  // ── النظام ──
  { g: "system", href: "/dashboard/audit", label: m.admin.nav.auditLog, icon: IconList,
    caps: ["audit.read"] },
  // الإعدادات للقراءة لمن يملك أحد أبوابها، والتعديلُ يحرسه الخادم.
  { g: "system", href: "/dashboard/settings", label: m.terms.settings, icon: IconSettings,
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
  return withGroupTitles(visibleNav(caps));
}

/** **ما يملكه صاحبُ القدرات** — كلُّ بابٍ بقدرته وحدَها (`R-34`). */
function visibleNav(caps: readonly string[]): (NavItem & { g?: Group })[] {
  const can = (c: string) => caps.includes(c);
  return ALL_NAV.filter((i) => !!i.caps && i.caps.some(can));
}

/** **عنوانُ المجموعة على أوّل بندٍ ظاهرٍ منها** — فلا يضيع إن حُجب أوّلُها. */
function withGroupTitles(items: (NavItem & { g?: Group })[]): ChromeNavItem[] {
  const seen = new Set<Group>();
  return items.map(({ g, ...i }) => {
    if (!g || seen.has(g)) return i;
    seen.add(g);
    return { ...i, group: GROUP_LABEL[g] };
  });
}


export default function DashboardLayout({ children }: { children: React.ReactNode }) {
  const { user, capabilities, capsLoaded, loading, logout } = useAuth();
  const router = useRouter();
  const pathname = usePathname();
  const nav = useMemo(() => navFor(user?.roles, capabilities), [user?.roles, capabilities]);

  // **عدّادُ «المضافُ حديثاً» على بند السوق** (قرارُ المالك ٢٠٢٦-١٠-٠٤) —
  // ما أُضيف بعد آخر فتحٍ للسوق، **ويختفي حين يُفتح.** (إضافةٌ من دفعة السوق.)
  const canMarket = capabilities.includes("market.manage");
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
  // **عدّادُ ما يحتاج تدخّلاً في الطلبات** (تصميمُ اللوحة ٢٠٢٦-١٠-٢٥) — العالقُ والبلاغات، بشرط
  // عدّادات اللوحة نفسِها (`/orders/board`)؛ يحمرّ فيُرى من أيّ صفحة.
  const canOrders = capabilities.includes("orders.read");
  const { data: board } = useLiveData<{ counts: Record<string, number> }>(
    () => (canOrders ? api("/api/v1/admin/orders/board") : Promise.resolve({ counts: {} })),
    ["order", "alerts"],
    [canOrders],
  );
  const ordersHot = (board?.counts?.stuck ?? 0) + (board?.counts?.reports ?? 0);
  const navShown = useMemo(
    () =>
      nav.map((i) => {
        if (i.href === "/dashboard/orders")
          return { ...i, badge: ordersHot, badgeTone: "danger" as const };
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
    [nav, marketNew, inMarket, emergencyCount, ordersHot],
  );

  // **أرضُ اللوحة الثابتة** (تصميمُ اللوحة ٢٠٢٦-١٠-٢٥) — صنفٌ على `html` يقرؤه `theme.css`،
  // ويُرفع عند الخروج من اللوحة فلا يلحق بموقع الزبون.
  useEffect(() => {
    document.documentElement.classList.add("admin-shell");
    return () => document.documentElement.classList.remove("admin-shell");
  }, []);

  // **المسارُ الصغير**: أطولُ بابٍ يطابق الصفحة، ومجموعتُه — ولا شيءَ للرئيسيّة.
  const crumb = useMemo(() => {
    const hit = ALL_NAV.filter(
      (i) => i.g && (pathname === i.href || pathname.startsWith(i.href + "/")),
    ).sort((a, b) => b.href.length - a.href.length)[0];
    if (!hit && pathname.startsWith("/dashboard/merchants/"))
      return { group: GROUP_LABEL.stores, label: m.admin.nav.merchants };
    return hit?.g ? { group: GROUP_LABEL[hit.g], label: hit.label } : null;
  }, [pathname]);

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
        صار المستخدمُ يملك.

        **والرصيدُ يفتح كشفَ الخزينة، وبجانبه المتوقَّعُ في الدرج** (قرارُ المالك
        ٢٠٢٦-١٠-١٥) — وصفحةُ «محفظتي» أُزيلت: الخزينةُ هي محفظةُ الأدمن، وكشفُها
        في قسمها. **وشريحةُ الدرج لمن يملك `finance.read`** — يقرّرها المحرّكُ في الملخّص. */}
    <DashboardChrome
      walletHref="/dashboard/treasury?tab=statement"
      drawerHref="/dashboard/treasury?tab=cashbox"
      /* **و`brand` تسميةٌ احتياطيّةٌ للوحة لا اسمُ المنصة** — تُعرض حين
         لا يكون الاسمُ مضبوطاً في الإعدادات، **وتُقرأ للقارئ الصوتيّ على
         زرّ القائمة.** (قرارُ المالك ٢٠٢٦-٠٨-٠٦.) */
      brand={m.admin.nav.dashboard}
      /* **ولا علامةَ في رأس سايدبار الإدارة** (قرارُ المالك ٢٠٢٦-٠٨-٠٨).
         من يفتحها يفتحها عشرَ مرّاتٍ في اليوم، **فالعلامةُ تقول له ما
         يعرف** وتأخذ سطراً من قائمةٍ طويلة. **وتبقى في البوّابات الأربع.** */
      showBrand={false}
      /* **والبحثُ العامّ في الشريط** (تصميمُ اللوحة ٢٠٢٦-١٠-٢٥) — من أيّ صفحة. */
      topbarStart={<GlobalSearch />}
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
      {/* **«أنت هنا»** (تصميمُ اللوحة ٢٠٢٦-١٠-٢٥) — المجموعةُ ثمّ الباب، فوق كلّ صفحة. */}
      {crumb && (
        <p className="mb-2 flex items-center gap-1.5 text-xs text-ink-muted">
          <span>{crumb.group}</span>
          <span aria-hidden>{m.admin.nav.crumbSep}</span>
          <span className="font-medium text-ink">{crumb.label}</span>
        </p>
      )}
      {children}
    </DashboardChrome>
    </PasswordGate>
  );
}
