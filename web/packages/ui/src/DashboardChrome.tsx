"use client";

/**
 * الهيكل العائم الموحّد (سايدبار + توب بار) — نسخة واحدة مركزية لكل بوابات اللوحة
 * (إدارة/مندوب/متجر). يُحقن لها api و Link و mediaUrl الخاصة بالتطبيق (حقن تبعية)
 * فتبقى @rahalgo/ui غير مقيّدة بإطار معيّن.
 */

import { useCallback, useEffect, useRef, useState, type ComponentType, type ReactNode } from "react";
import { createPortal } from "react-dom";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { LiveNotifications, useLiveRefresh } from "./Notifications";
import { BrandMark, usePlatform } from "./platform";
import { Money } from "./money";
import { isStagingEnv } from "./runtimeconfig";
import { CountBadge } from "./components";
import {
  TopBar,
  TopBarChip,
  TopBarLink,
  TopBarActions,
  TOPBAR_ICON,
  type AccountMenuItem,
} from "./topbar";
import {
  IconWallet,
  IconStar,
  IconTrendUp,
  IconTrendDown,
  IconHamburger,
  IconClose,
  IconStore,
  IconBalance,
} from "./icons";

const m = getMessages(defaultLocale);

/**
 * **قائمةُ حساب اللوحة — بلا بنودٍ زائدة.**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «طبّق الاسم والقائمة مع باقي اللوحات وأضِف فقط
 *  تسجيل الخروج للقائمة».)
 *
 * **وأبوابُ اللوحة في سايدبارها** — اثنان وعشرون بنداً بأسمائها،
 * **فقائمةٌ تُكرّر منها بعضاً تصير باباً ثانياً لِما له باب.**
 *
 * **وخارجَ المكوّن لا داخلَه**: مصفوفةٌ تُبنى في كلّ طلاءٍ مرجعٌ جديدٌ في
 * كلّ مرّة — **وهي تُمرَّر إلى `useEffect` عبر `items` في القائمة.**
 */
const ACCOUNT_MENU: readonly AccountMenuItem[] = [];

type ApiFn = <T>(path: string, init?: RequestInit) => Promise<T>;
type IconType = ComponentType<{ size?: number; strokeWidth?: number; className?: string }>;
type LinkType = ComponentType<{ href: string; className?: string; title?: string; children: ReactNode; onClick?: () => void }>;

export interface ChromeNavItem {
  href: string;
  label: string;
  icon: IconType;
  /**
   * **عنوانُ المجموعة التي يبدؤها هذا البند** — واختياريّ.
   *
   * # لماذا وُجد
   *
   * (**ولا يُستعمل في لوحة الإدارة** — رفض المالك عناوينَ المجموعات ٢٠٢٦-٠٨-٠٨؛ يبقى للبوّابات.)
   *
   * كانت قائمةُ الادمن **بنوداً مسطَّحة**: الطلباتُ ثمّ السجلُّ ثمّ
   * الأقسامُ ثمّ الشكاوى ثمّ التقييماتُ ثمّ الطوارئ ثمّ الخزينةُ ثمّ
   * الصندوقُ… **بلا فاصلٍ ولا عنوان.**
   *
   * **والعينُ تمسح تسعةَ عشرَ سطراً في كلّ مرّة** لتجد ما تريد — ولا
   * تتعلّم مواضعَها لأنّ لا شيءَ يجمعها.
   *
   * **والمجموعاتُ في التعليقات أصلاً** («التشغيل اليوميّ» · «المال» ·
   * «البناء») — **كُتبت لقارئ الشيفرة ولم تصل إلى الشاشة.**
   *
   * **ويُوضع على أوّل بندٍ فيها لا على كلٍّ** — فالقائمةُ تبقى مصفوفةً
   * واحدة، **وترتيبٌ يُبنى من كائناتٍ متداخلةٍ يُخطئ فيه من يُضيف بنداً.**
   */
  group?: string;
  /** يُخفى عن قائمة الجوّال — **لِما يُفتح من داخل صفحةٍ أخرى.** */
  hideOnMobile?: boolean;
  /**
   * **عدّادٌ بجانب البند** — ما ينتظر النظرَ فيه (مثلاً «المضافُ حديثاً» في
   * السوق). **والصفرُ لا يُرسم**: عدّادٌ يختفي حين يُفتح البابُ هو الخبر.
   */
  badge?: number;
  /**
   * **نبرةُ العدّاد** — `danger` لما ينتظر يداً الآن (طارئٌ بلا مستلِمٍ بعد مهلته،
   * قرارُ المالك ٢٠٢٦-١٠-٠٤). وافتراضُه النبرةُ العاديّة.
   */
  badgeTone?: "accent" | "danger";
  /**
   * **مساراتٌ أخرى تلوّن هذا البند** (تدقيقُ السايدبار ٢٠٢٦-١٠-٢٧) — صفحةٌ فرعيّةٌ تنتمي إليه
   * بغير بادئته: ملفُّ المتجر للحسابات، والأدوارُ للإعدادات. **وبدونها يفقد الموظّفُ مكانَه.**
   */
  activeAlso?: string[];
}

interface Summary {
  full_name: string;
  avatar_thumb_url: string | null;
  balance: number;
  /**
   * **المتوقَّعُ في درج المكتب** (قرارُ المالك ٢٠٢٦-١٠-١٥) — يرسله المحرّكُ لمن يملك
   * `finance.read` وحدَه، **وغيابُه يُخفي الشريحة** فلا يُخمَّن الحقُّ في الواجهة.
   */
  drawer_expected?: number;
  /** **رصيدُ محفظة الخزينة نفسِها** — لمن يملك `finance.read` وحدَه (تدقيقُ الشريط ٢٠٢٦-١٠-٢٧). */
  treasury_balance?: number;
}
interface Rep {
  rating: { avg: number; count: number; trend: "up" | "down" | "flat" };
}

/**
 * **أبيئةُ تجربة؟** — `P-0` البند ٤٥.
 *
 * **ويُقرأ وقتَ التشغيل من تهيئةٍ مركزيّة** (دورةُ ٧١و) — **وكان
 * يُخبَز وقتَ البناء**، **فكانت صورةُ التجهيز غيرَ صورة الإنتاج ولو
 * من التزامٍ واحد.**
 *
 * **ولا يظهر في الإنتاج بغلطة**: **الرايةُ تتبع `environment` الذي
 * يضبطه `compose`** — **وقيمةٌ ناقصةٌ لا تعني «إنتاج»، تعني فراغاً،
 * فلا رايةَ ولا كذب.**
 *
 * **ودالّةٌ لا ثابت**: **ثابتٌ على مستوى الملفّ يُحسَب عند التحميل**
 * — **وقد يسبق وصولَ التهيئة.**
 */

/** **نصُّ الراية** — من المعجم المركزيّ لا من الشيفرة. */
const stagingLabel = m.shared.stagingBanner;

export function DashboardChrome({
  brand,
  nav,
  pathname,
  homeHref,
  accountHref,
  walletHref,
  drawerHref,
  ratingHref,
  api,
  mediaUrl,
  Link,
  onLogout,
  phone,
  shopUrl,
  shopLabel,
  showRating = false,
  showBrand = true,
  ratingLabel,
  topbarStart,
  wsUrl,
  token,
  notificationsHref,
  treasuryMode = false,
  children,
}: {
  brand: string;
  nav: ChromeNavItem[];
  pathname: string;
  homeHref: string;
  accountHref: string;
  walletHref?: string;
  /** **شريحةُ الدرج بجانب الرصيد** — تظهر حين يرسل الملخّصُ `drawer_expected`. */
  drawerHref?: string;
  ratingHref?: string;
  api: ApiFn;
  mediaUrl: (p: string | null | undefined) => string | null;
  Link: LinkType;
  onLogout: () => void;
  phone?: string;
  shopUrl?: string;
  shopLabel?: string;
  showRating?: boolean;
  /** **علامةُ المنصة في رأس السايدبار** — تُطفأ حيث لا تُفيد. (٢٠٢٦-٠٨-٠٨.) */
  showBrand?: boolean;
  /** تسمية شارة التقييم — تختلف بالدور (تقييمي للمتجر، تقييم متاجري للمندوب) */
  ratingLabel?: string;
  topbarStart?: ReactNode;
  /** عنوان قناة البث الحي (ws://…/api/v1/ws) */
  wsUrl?: string;
  /** توكن الوصول للبث — بلا ترويسات في WebSocket */
  token?: string | null;
  /** مسار صفحة الإشعارات الكاملة في هذا التطبيق */
  notificationsHref?: string;
  /**
   * **لوحةُ الإدارة: الرصيدُ رصيدُ الخزينة لا محفظةُ الفاتح** (تدقيقُ الشريط ٢٠٢٦-١٠-٢٧) — يُقرأ من
   * `treasury_balance`، **وغيابُه يُخفي الشريحة**. وعلى الجوّال تصير الشريحتان زرّاً واحداً «المال».
   */
  treasuryMode?: boolean;
  children: ReactNode;
}) {
  const [menuOpen, setMenuOpen] = useState(false);
  const [summary, setSummary] = useState<Summary | null>(null);
  // **ولا صفرَ كاذبٌ قبل الوصول** — «—» حتّى يصل الملخّص، وتلميحٌ صريحٌ إن فشل.
  const [summaryFailed, setSummaryFailed] = useState(false);
  const [rep, setRep] = useState<Rep | null>(null);
  const { name: platformName } = usePlatform();

  const loadSummary = useCallback(() => {
    api<Summary>("/api/v1/me/summary")
      .then((v) => {
        setSummary(v);
        setSummaryFailed(false);
      })
      .catch(() => setSummaryFailed(true));
    if (showRating) api<Rep>("/api/v1/me/reputation").then(setRep).catch(() => undefined);
  }, [api, showRating]);

  useEffect(() => {
    loadSummary();
  }, [loadSummary, pathname]);

  // الرصيد والتقييم والصورة في الشريط العلوي تتحدّث لحظياً بلا إعادة تحميل
  // ("profile" حدث محلي يبثّه AccountSettings عند تغيير الصورة أو الرقم)
  // **ودرجُ المكتب معها** — كلُّ حركةٍ في الصندوق تبثّ `wallet` (`s.touch("wallet", ...)`)،
  // فالإشارةُ نفسُها تحدّث الرصيدَ والدرجَ بنداءٍ واحد.
  useLiveRefresh(["wallet", "rating", "profile"], loadSummary);

  useEffect(() => setMenuOpen(false), [pathname]);

  // ══════════════════════════════════════════════════════════════════
  // **والمطابقةُ بالمسار كاملاً أو بما بعده «/»** (تدقيقُ السايدبار ٢٠٢٦-١٠-٢٧)
  // ══════════════════════════════════════════════════════════════════
  //
  // كانت `startsWith(href)` — **و`/dashboard/opsmap` يبدأ بـ`/dashboard/ops`**، فتلوّن «خريطةُ
  // العمليات» و«مراقبةُ التشغيل» معاً. **والرئيسيّةُ بمطابقةٍ كاملةٍ** كما كانت.
  const under = (href: string) => pathname === href || pathname.startsWith(href + "/");
  const isActive = (item: ChromeNavItem) =>
    item.href === homeHref
      ? pathname === item.href
      : under(item.href) || !!item.activeAlso?.some(under);
  // **واسمُ القسم المفتوح عنوانُ لسان المتصفّح** — لا في الشريط (قرارُ المالك ٢٠٢٦-٠٨-١١)،
  // **ومن فتح أقساماً في ألسنةٍ يميّز «الطلبات» من «الخزينة».**
  const activeLabel = nav.find((i) => isActive(i))?.label ?? "";
  const baseTitle = platformName || brand;
  useEffect(() => {
    if (typeof document === "undefined") return;
    document.title = activeLabel ? `${activeLabel} · ${baseTitle}` : baseTitle;
  }, [activeLabel, baseTitle]);

  // ══════════════════════════════════════════════════════════════════
  // **ودُرجُ الجوّال: الهروبُ يُغلقه، والصفحةُ لا تتحرّك تحته، والبندُ المفتوحُ يُرى**
  // ══════════════════════════════════════════════════════════════════
  const drawerRef = useRef<HTMLElement>(null);
  useEffect(() => {
    if (!menuOpen) return;
    const esc = (e: KeyboardEvent) => e.key === "Escape" && setMenuOpen(false);
    document.addEventListener("keydown", esc);
    const prev = document.body.style.overflow;
    document.body.style.overflow = "hidden";
    drawerRef.current?.querySelector('[aria-current="page"]')?.scrollIntoView({ block: "center" });
    return () => {
      document.removeEventListener("keydown", esc);
      document.body.style.overflow = prev;
    };
  }, [menuOpen]);
  const staging = isStagingEnv();

  /**
   * **بابُ التسوّق — وصار تنقّلاً لا عبوراً.**
   *
   * (قرارُ المالك ٢٠٢٦-٠٨-١٠: «بابٌ واحدٌ للجميع».)
   *
   * **كان يسلّم الجلسةَ برمزٍ لمرّة واحدة** لأنّ السوقَ كان أصلاً آخر
   * (منفذاً آخر)، **والتخزينُ المحلّيُّ لا يُشارَك بين أصلين.**
   *
   * **وصار السوقُ ولوحتُه في بيتٍ واحد** — فالجلسةُ نفسُها، **ونداءُ
   * تسليمٍ لا لزومَ له يضيف تأخّراً ونقطةَ فشلٍ في زرٍّ يُضغط كلَّ يوم.**
   *
   * **ويبقى التسليمُ لعنوانٍ خارجيّ** — يحتاجه تطبيقُ أندرويد.
   */
  async function shopAsCustomer() {
    if (!shopUrl) return;
    const here = typeof window !== "undefined" ? window.location.origin : "";
    if (shopUrl.startsWith("/") || shopUrl === here) {
      window.location.href = shopUrl.startsWith("/") ? shopUrl : "/";
      return;
    }
    try {
      const { code } = await api<{ code: string }>("/api/v1/auth/handoff", { method: "POST" });
      window.location.href = `${shopUrl}/sso?code=${encodeURIComponent(code)}`;
    } catch {
      /* يبقى في لوحته */
    }
  }

  const sidebar = (
    <>
      {/* ══════════════════════════════════════════════════════════════
          **ولا خطَّ يفصل العلامةَ عن بنودها**
          ══════════════════════════════════════════════════════════════

          (قرارُ المالك ٢٠٢٦-٠٨-٠٧: «ألغِ الخطَّ الأبيض من السايدبار،
           واللوغو خلّيه بالوسط وخلّيه أكبر من هيك يكون واضح».)

          **السايدبارُ زجاجٌ واحدٌ متّصل** — وخطٌّ يعبره يقطعه قطعتين، **وهو
          العيبُ نفسُه الذي أُزيل من الشريط العلويّ والذيل.**

          **والعلامةُ في الوسط**: هي وحدَها في سطرها، **وشيءٌ واحدٌ يلتصق
          بحافّةٍ يُقرأ منسيّاً هناك** لا مقصوداً.

          **وزرُّ الإغلاق مطلقٌ لا شريكٌ في الصفّ** — وإلّا أزاح العلامةَ عن
          الوسط على الجوّال وحدَه، **فتُرى الشاشتان مختلفتين.** */}
      {/* **ورأسٌ بارتفاعٍ يسع زرَّ الإغلاق على الجوّال** (تدقيقُ السايدبار ٢٠٢٦-١٠-٢٧) — كان ستّةَ
          عشرَ بكسلاً بلا علامة، فيركب الزرُّ طرفَ أوّل بند **ومن ضغط البندَ أغلق القائمة.** */}
      <div className={`relative flex items-center justify-center ${showBrand ? "p-4" : "h-12 lg:hidden"}`}>
        <div className="flex items-center gap-2">
          {/* ══════════════════════════════════════════════════════
              **اللوغو وحدَه — لا اسمٌ ولا تسميةُ بوّابة**
              ══════════════════════════════════════════════════════

              (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «لا يوجد داعٍ لكتابة بوّابة السائق
               والمندوب والمتجر والادمن — فقط يظهر لوغو المنصة بالسايدبار
               بدون اسمٍ للمنصة أو شيءٍ آخر».)

              **وكانت الأربعُ تكتب أربعةَ أسماءٍ مختلفةً في شيفرتها**، ثمّ
              صارت تكتب اسمَ المنصة من الإعدادات — **والاثنان زائدان**:
              من فتح بوّابتَه يعرف أيَّها فتح، **والسايدبارُ تحتها يقول
              دورَه بتسعةَ عشرَ بنداً.**

              **ويُطفأ في لوحة الإدارة** (قرارُ المالك ٢٠٢٦-٠٨-٠٨): من يفتحها
              يفتحها عشرَ مرّاتٍ في اليوم، **والعلامةُ تقول له ما يعرف** —
              وتأخذ سطراً من قائمةٍ طويلة. **ويبقى في البوّابات الأربع**:
              يفتحها صاحبُها مرّةً في اليوم فتقول له أين هو. */}
          {showBrand && <BrandMark size={92} disc />}
        </div>
        <button
          onClick={() => setMenuOpen(false)}
          type="button"
          className={`absolute end-3 flex items-center rounded-control p-2 text-ink-muted hover:bg-row-hover hover:text-ink lg:hidden ${
            showBrand ? "top-3" : "top-1/2 -translate-y-1/2"
          }`}
          aria-label={m.shared.closeMenu}
          title={m.shared.closeMenu}
        >
          <IconClose size={20} />
        </button>
      </div>

      {/* ══════════════════════════════════════════════════════════════
          **وبابُ التسوّق بندٌ عاديّ — لا صندوقٌ ممتلئ**
          ══════════════════════════════════════════════════════════════

          (شكوى المالك ٢٠٢٦-٠٨-١٠: «خلّي كلَّ الأزرار عاديّ، بس القسمُ الذي
           نقف عليه يكون واضحاً — مو مثل الصور».)

          **كان `bg-accent-tint` مصمتاً** — وصندوقُ القسم الحاليّ
          `bg-primary-tint`. **صندوقان ممتلئان في قائمةٍ واحدةٍ بوزنٍ
          متقارب**، فيُقرآن اثنين مفتوحين.

          **ومن نظر لا يعرف أين هو** — يبحث عن القسم الذي يقف عليه فيجد
          اثنين يشبهانه. **وأمارةُ «أنت هنا» لا تُقرأ إلّا إن كانت وحدَها.**

          **وهو ليس قسماً أصلاً**: رابطٌ يخرج من اللوحة إلى الموقع.

          # ولا يُساوى بالبنود الساكنة أيضاً

          (تصحيحُ المالك في اللحظة نفسِها: «وقسم تسوّق خلّيه مثل زرّ بلونٍ
           مختلفٍ مميّز مشان يكون مبيّناً أنّه زرّ وليس قسماً».)

          **فجُرّب ساكناً فصار يُقرأ قسماً ساكناً** — بندٌ في القائمة لم
          يُفتح بعد. **وهو ليس قسماً يُفتح، هو فعلٌ يُخرجك من اللوحة.**

          **والفرقُ بين «أنت هنا» و«اضغط لتخرج» فرقُ شكلٍ لا درجة**: القسمُ
          الحاليُّ **صبغةٌ وشريطٌ جانبيّ** — سطرٌ في قائمة. وهذا **لونٌ
          مصمتٌ وظلٌّ** — جسمٌ يعلو القائمةَ ولا ينتمي إليها.

          **ونبرةٌ ثانيةٌ لا نبرةُ القسم**: الأخضرُ للخروج إلى السوق،
          **والأزرقُ لموضعك في اللوحة** — فلا يلتبسان ولو تجاورا. */}
      {shopUrl && (
        <div className="p-3 pb-0">
          <button
            onClick={shopAsCustomer}
            className="flex w-full items-center justify-center gap-2 rounded-control bg-accent px-3 py-2.5 text-sm font-bold text-on-bright elev-1 transition-opacity hover:opacity-90"
          >
            <IconStore size={17} />
            {shopLabel}
          </button>
        </div>
      )}

      <nav className="flex-1 space-y-1 overflow-y-auto p-3">
        {nav.map((item) => {
          const active = isActive(item);
          const Icon = item.icon;
          return (
            <div key={item.href}>
              {/* **وعنوانُ المجموعة يفصل ولا يُضغط.**

                  **ولا خطَّ فاصلاً معه**: العنوانُ وحدَه يفصل، **وخطٌّ فوق
                  كلّ مجموعةٍ يجعل القائمةَ سلسلةَ صناديق** — وهي عينُ ما
                  خرجنا منه في الشريط العلويّ. */}
              {item.group && (
                <p className="px-3 pt-4 pb-1.5 text-2xs font-bold tracking-wider text-ink-muted first:pt-0">
                  {item.group}
                </p>
              )}
              {/* ══════════════════════════════════════════════════════
                  **وأمارةُ «أنت هنا» ثلاثُ إشاراتٍ لا واحدة**
                  ══════════════════════════════════════════════════════

                  **صبغةٌ وحدَها لا تكفي في قائمةٍ طويلة**: العينُ تمرّ على
                  خمسةَ عشرَ بنداً، **ودرجةُ لونٍ خفيفةٌ تُقرأ ظلّاً لا
                  علامة.**

                  **فشريطٌ في جهة البداية** يقطع الصفَّ فيُرى قبل أن يُقرأ،
                  **وخطٌّ عريضٌ وحرفٌ كامل** — ثلاثتُها معاً لا تلتبس ببندٍ
                  ساكن.

                  **والشريطُ محجوزٌ للجميع** (`border-transparent` للساكن)
                  — **وبدونه يقفز الصفُّ بكسلين كلّما بُدّل القسم**، وهي
                  العلّةُ نفسُها التي حُرست في شريط التبويب. */}
              <Link
                href={item.href}
                aria-current={active ? "page" : undefined}
                className={`flex items-center gap-2.5 rounded-control border-s-2 px-3 py-2 text-sm transition-colors ${
                  active
                    ? "border-primary bg-primary-tint font-bold text-ink"
                    : "border-transparent text-ink-muted hover:bg-row-hover hover:text-ink"
                }`}
              >
                <Icon size={17} strokeWidth={active ? 2.4 : 1.8} />
                <span className="min-w-0 flex-1 truncate">{item.label}</span>
                {!!item.badge && <CountBadge count={item.badge} tone={item.badgeTone} />}
              </Link>
            </div>
          );
        })}
      </nav>
    </>
  );

  return (
    /* **ولا لونَ مصمتٌ فوق خلفيّة الصفحة.**

       (المرحلةُ الأولى من ورشة ٢٠٢٦-٠٨-٠٦: خلفيّةٌ مرسومةٌ بالطبقات على
        `body`.)

       كان `bg-shell` — **لوحاً مصمتاً يغطّي الشاشةَ كلَّها**، فتُرسم الخلفيّةُ
       تحته ولا تُرى. **ولوحةُ التحكّم أربعةُ تطبيقاتٍ من خمسة**، فتبقى
       الخلفيّةُ في الموقع وحدَه. */
    <div className={`flex min-h-screen ${staging ? "pt-6" : ""}`}>
      {/* ══════════════════════════════════════════════════════════════
          **رايةُ بيئة التجربة — سلامةٌ لا زينة** (`P-0` البند ٤٥)
          ══════════════════════════════════════════════════════════════

          **ومن جرّب على «التجهيز» وهو الإنتاج لا يكتشف ذلك إلّا بعد
          أن يحذف شيئاً.** **واللوحتان متطابقتان حرفاً** — لا شيءَ في
          الشاشة يقول أيَّهما أمامه.

          **وتُقرأ من `NEXT_PUBLIC_ENVIRONMENT` وقتَ البناء** — **فلا
          يُطفئها متغيّرُ تشغيلٍ على الخادم**، ولا تظهر في الإنتاج
          بغلطة: **القيمةُ مدفونةٌ في الحزمة.**

          **وشريطٌ ثابتٌ في أعلى الشاشة لا شارةٌ في زاوية** — **والزاويةُ
          تُنسى بعد ساعة.** */}
      {/* **ويأخذ مكانَه ولا يغطّي** (تدقيقُ الشريط ٢٠٢٦-١٠-٢٧): كان ثابتاً بلا مسافةٍ تحته، فيُدفن
          رأسُ السايدبار والشريطِ وعدّادُ الجرس. **فالغلافُ ينزل بقدره** (`pt-6` = ارتفاعُه). */}
      {staging && (
        <div
          role="status"
          className="fixed inset-x-0 top-0 z-[60] h-6 bg-warning text-center text-xs font-bold leading-6 text-on-bright"
        >
          {stagingLabel}
        </div>
      )}
      <aside
        className={`sticky m-3 me-0 hidden w-60 shrink-0 flex-col surface-lit chrome overflow-hidden rounded-card border border-line lg:flex ${
          staging ? "top-9 h-[calc(100vh-3rem)]" : "top-3 h-[calc(100vh-1.5rem)]"
        }`}
      >
        {sidebar}
      </aside>

      {menuOpen && (
        <div className="fixed inset-0 z-40 scrim lg:hidden" onClick={() => setMenuOpen(false)} />
      )}
      <aside
        ref={drawerRef}
        inert={!menuOpen}
        aria-hidden={!menuOpen}
        /* **ومغلقاً لا يُمسَك بالتنقّل بلوحة المفاتيح** (`inert`) — روابطُه خارج الشاشة.

           **والدُّرجُ الجوّاليُّ يُضبّب ويثقل — وهو يعلو المحتوى.**

           (المرحلةُ الثالثة ٢٠٢٦-٠٨-٠٦.)

           كان `bg-surface` بلا تضبيب — **وكان مصمتاً يوم كان السطحُ مصمتاً.**
           ولمّا صار السطحُ زجاجاً بخمسةٍ وستّين **صار محتوى الصفحة يُرى من
           خلفه حادّاً**، فتُقرأ بنودُ القائمة فوق جدولٍ يتحرّك.

           **و`raised` لا `surface`**: ما يعلو كلَّ شيءٍ يبقى أثقل — كالنوافذ
           والقوائم، **وقائمةُ تنقّلٍ لا تُقرأ ليست قائمة.** */
        className={`surface-lit chrome fixed bottom-0 start-0 z-50 flex w-64 ${staging ? "top-6" : "top-0"} flex-col elev-4 transition-transform duration-200 lg:hidden ${
          menuOpen ? "translate-x-0" : "translate-x-full rtl:translate-x-full ltr:-translate-x-full"
        }`}
      >
        {sidebar}
      </aside>

      {/* **حشوةٌ واحدةٌ لا اثنتان.**

          كانت `p-3` هنا و`p-4` في `main` — **ثمانيةٌ وعشرون بكسلاً من كلّ
          جانبٍ قبل أن يبدأ المحتوى**، وجداولُ اللوحة تُقصّ أعمدتَها لتتّسع.
          فبقيت واحدةٌ صغيرةٌ تفصل البطاقةَ عن حافّة الشاشة، **والمحتوى يتمدّد.**
          (قرارُ المالك ٢٠٢٦-٠٨-٠٣.) */}
      {/* **وفجوةٌ بين اللوحين.**

          الشريطُ لوحٌ مدوَّرٌ والمحتوى لوحٌ مدوَّر — **وملتصقان يُقرآن لوحاً
          واحداً بخطٍّ في وسطه**، فتضيع الزاويةُ التي جُعلت لتُرى. */}
      <div className="flex min-w-0 flex-1 flex-col gap-2 p-2 lg:gap-3 lg:p-3">
        <TopBar
          /* **ولوحةُ التحكّم لوحٌ مدوَّرٌ لا شريطٌ ممتدّ.** (قرارُ المالك
             ٢٠٢٦-٠٨-٠٦: «باقي اللوحات خلّيها بحواف مستديرة مع بادينك».)

             **والفرقُ ليس ذوقاً**: الشريطُ الممتدُّ هنا يلتصق بالسايدبار
             **فيصيران كتلةً واحدة** — والسايدبارُ نفسُه لوحٌ مدوَّرٌ يطفو. */
          shape="card"
          start={
            <>
              <TopBarChip
                type="button"
                onClick={() => setMenuOpen(true)}
                className="lg:hidden"
                aria-label={m.shared.openMenu}
                aria-expanded={menuOpen}
                title={m.shared.openMenu}
              >
                <IconHamburger size={22} />
              </TopBarChip>
              {/* **ولا اسمَ قسمٍ في الشريط** — (قرارُ المالك ٢٠٢٦-٠٨-١١:
                  «لا يوجد داعٍ لكتابة اسم القسم بالتوب بار أصلاً»).

                  **والقسمُ مُعلَّمٌ في الشريط الجانبيّ** بحدٍّ ولونٍ وخطٍّ
                  عريض — **واسمُه في الشريط العلويِّ تكرارٌ يأكل عرضاً**
                  هو أضيقُ ما يُنازع عليه في الجوّال.

                  **و`activeLabel` يقرؤه عنوانُ لسان المتصفّح** (`document.title` فوق). */}
              {topbarStart}
            </>
          }
        >
          <TopBarActions
            Link={Link}
            notifications={
              <LiveNotifications
                api={api}
                wsUrl={wsUrl ?? ""}
                token={token ?? null}
                Link={Link}
                allHref={notificationsHref}
              />
            }
            menu={ACCOUNT_MENU}
            walletHref={treasuryMode ? undefined : walletHref}
            balance={summary ? summary.balance : null}
            walletIcon={<IconWallet size={TOPBAR_ICON} />}
            accountHref={accountHref}
            accountLabel={m.terms.account}
            avatarUrl={mediaUrl(summary?.avatar_thumb_url)}
            name={summary?.full_name || phone || ""}
            onLogout={onLogout}
            logoutLabel={m.auth.logout}
            active={pathname}
            extras={
              <>
              {treasuryMode && (walletHref || drawerHref) && (
                <MoneyChips
                  Link={Link}
                  walletHref={walletHref}
                  drawerHref={drawerHref}
                  wallet={summary?.treasury_balance}
                  drawer={summary?.drawer_expected}
                  state={summary ? "ready" : summaryFailed ? "failed" : "loading"}
                  pathname={pathname}
                />
              )}
              {/* **والعضوُ يُحرَس كما يُحرَس الكائن.**

                 كان `rep && rep.rating.count` — **يسأل عن الكائن ويثق بعضوه.**
                 وردٌّ ناقصُ `rating` (٢٠٠ بجسمٍ غير متوقّع) يرمي هنا،
                 **والرميةُ في `DashboardChrome` تُبيّض اللوحةَ كلَّها** — لا
                 شريطَ ولا قائمةَ ولا محتوى، بل «حدث خطأ في التطبيق».
                 (وقع فعلاً في فحصٍ بمتصفّح ٢٠٢٦-٠٨-٠٦.)

                 **وخسارةُ نجمةٍ في الشريط أهونُ من خسارة اللوحة.** */}
              {showRating &&
              rep?.rating &&
              rep.rating.count > 0 && (
                <TopBarLink
                  Link={Link}
                  href={ratingHref ?? accountHref}
                  tone="accent"
                  title={ratingLabel ?? m.terms.myRating}
                >
                  <IconStar size={TOPBAR_ICON} className="fill-accent text-accent-text" />
                  <span dir="ltr">{rep.rating.avg.toFixed(1)}</span>
                  {rep.rating.trend === "up" && <IconTrendUp size={TOPBAR_ICON} className="text-success" />}
                  {rep.rating.trend === "down" && <IconTrendDown size={TOPBAR_ICON} className="text-danger" />}
                </TopBarLink>
              )}
              </>
            }
          />
        </TopBar>

        {/* ══════════════════════════════════════════════════════════
            **ولا كرتَ حول المحتوى — سايدبارٌ وتوب بارٌ ومحتوًى**
            ══════════════════════════════════════════════════════════

            (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «أهمُّ شيء الكرت الخاصُّ بالمحتوى
             ألغِه — فقط سايدبار وتوب بار ومحتوى بدون كرت أفضل».)

            **وهو القرارُ نفسُه الذي اتُّخذ لموقع الزبون** قبلَ ذلك: المحتوى
            يقف على الصفحة لا داخلَ صندوق.

            # ولماذا يصحّ هنا

            **محتوى اللوحة ألواحٌ أصلاً**: جداولٌ وبطاقاتٌ ولوحاتُ إحصاء.
            **فلوحٌ يلفّها كلَّها يصير صندوقاً في صندوق** — والعينُ تعدّ
            الحدودَ قبل أن تقرأ ما فيها.

            **والحشوةُ تبقى** — هي ما يفصل المحتوى عن حافّة الشاشة، **وذهابُ
            اللوح لا يعني التصاقَ الجدول بالحافّة.** */}
        <main className="min-w-0 flex-1 p-3 sm:p-4">
          {children}
        </main>
      </div>
    </div>
  );
}

// ---------- شريحتا المال — لوحة الإدارة ----------

type MoneyState = "loading" | "ready" | "failed";

/**
 * **رقمٌ بعملته — والسالبُ أحمرُ في الشريحتين بقاعدةٍ واحدة** (تدقيقُ الشريط ٢٠٢٦-١٠-٢٧).
 *
 * كان الدرجُ السالبُ أحمرَ بإشارةٍ تُكتب باليد والمحفظةُ السالبةُ بلونٍ عاديٍّ وبلا عملةٍ على
 * الجوّال. **و«—» قبل الوصول لا صفرٌ كاذب** — وتلميحٌ صريحٌ إن فشل الطلب.
 */
function MoneyFigure({ value, state }: { value: number | undefined; state: MoneyState }) {
  if (typeof value !== "number") {
    return (
      <span
        className="font-bold text-ink-muted"
        title={state === "failed" ? m.shared.balanceFailed : m.shared.balanceLoading}
      >
        —
      </span>
    );
  }
  return (
    <Money
      value={value}
      small
      className={`whitespace-nowrap font-bold ${value < 0 ? "text-danger" : "text-ink"}`}
    />
  );
}

/**
 * **«محفظة المنصة» و«درج المكتب» — شريحتان على الواسع، وزرٌّ واحدٌ «المال» على الجوّال.**
 *
 * (تدقيقُ الشريط ٢٠٢٦-١٠-٢٧: على الجوّال كان رقمان متجاوران بلا اسمٍ ولا عملة، فلا يُعرف أيُّهما
 *  المحفظةُ وأيُّهما الدرج، والصفُّ يفيض فتختبئ الصورةُ خلف التمرير.) **فالزرُّ يفتح لوحاً صغيراً
 * بسطرين واضحين**: الاسمُ والشرحُ والرقمُ بعملته، وكلُّ سطرٍ يفتح كشفَه.
 */
function MoneyChips({
  Link,
  walletHref,
  drawerHref,
  wallet,
  drawer,
  state,
  pathname,
}: {
  Link: LinkType;
  walletHref?: string;
  drawerHref?: string;
  wallet: number | undefined;
  drawer: number | undefined;
  state: MoneyState;
  pathname: string;
}) {
  const [open, setOpen] = useState(false);
  const [top, setTop] = useState(0);
  const btn = useRef<HTMLSpanElement>(null);
  const panel = useRef<HTMLDivElement>(null);

  useEffect(() => setOpen(false), [pathname]);
  useEffect(() => {
    if (!open) return;
    const away = (e: PointerEvent) => {
      const t = e.target as Node;
      if (btn.current?.contains(t) || panel.current?.contains(t)) return;
      setOpen(false);
    };
    const esc = (e: KeyboardEvent) => e.key === "Escape" && setOpen(false);
    const close = () => setOpen(false);
    document.addEventListener("pointerdown", away);
    document.addEventListener("keydown", esc);
    window.addEventListener("resize", close);
    window.addEventListener("scroll", close, true);
    return () => {
      document.removeEventListener("pointerdown", away);
      document.removeEventListener("keydown", esc);
      window.removeEventListener("resize", close);
      window.removeEventListener("scroll", close, true);
    };
  }, [open]);

  // **والمحفظةُ تُرى «—» حتّى يصل الملخّص** — ثمّ تُخفى إن لم يرسل المحرّكُ رصيدَ الخزينة.
  const showWallet = !!walletHref && (state !== "ready" || typeof wallet === "number");
  const showDrawer = !!drawerHref && typeof drawer === "number";
  if (!showWallet && !showDrawer) return null;

  const drawerHint = typeof drawer === "number" && drawer < 0 ? m.shared.drawerNegative : m.shared.drawerChip;

  const toggle = () => {
    if (!open) {
      const bar = btn.current?.closest("header")?.getBoundingClientRect();
      setTop((bar?.bottom ?? 64) + 8);
    }
    setOpen((v) => !v);
  };

  const row = "flex items-center gap-3 px-4 py-3 text-sm transition-colors hover:bg-row-hover";

  return (
    <>
      <span className="hidden sm:contents">
        {showWallet && walletHref && (
          <TopBarLink
            Link={Link}
            href={walletHref}
            title={m.shared.platformWalletHint}
            aria-label={m.shared.platformWalletChip}
          >
            <IconWallet size={TOPBAR_ICON} />
            <span className="hidden text-xs text-ink-muted md:inline">{m.shared.platformWalletChip}</span>
            <MoneyFigure value={wallet} state={state} />
          </TopBarLink>
        )}
        {showDrawer && drawerHref && (
          <TopBarLink Link={Link} href={drawerHref} title={drawerHint} aria-label={m.shared.drawerChipShort}>
            <IconBalance size={TOPBAR_ICON} />
            <span className="hidden text-xs text-ink-muted md:inline">{m.shared.drawerChipShort}</span>
            <MoneyFigure value={drawer} state={state} />
          </TopBarLink>
        )}
      </span>

      <span ref={btn} className="flex shrink-0 sm:hidden">
        <TopBarChip
          type="button"
          onClick={toggle}
          aria-haspopup="dialog"
          aria-expanded={open}
          aria-label={m.shared.moneyPanel}
          title={m.shared.moneyPanel}
          className={open ? "text-ink" : ""}
        >
          <IconWallet size={TOPBAR_ICON} />
          <span className="text-sm font-medium">{m.shared.moneyChip}</span>
          {typeof drawer === "number" && drawer < 0 && (
            <span className="h-2 w-2 rounded-full bg-danger-solid" aria-hidden />
          )}
        </TopBarChip>
      </span>

      {open &&
        createPortal(
          <div
            ref={panel}
            role="dialog"
            aria-label={m.shared.moneyPanel}
            style={{ top }}
            className="fixed inset-x-3 z-[80] overflow-hidden surface-raised elev-3 sm:hidden"
          >
            {showWallet && walletHref && (
              <Link href={walletHref} className={row} onClick={() => setOpen(false)}>
                <IconWallet size={20} className="shrink-0 text-primary" />
                <span className="min-w-0 flex-1">
                  <span className="block font-bold text-ink">{m.shared.platformWalletChip}</span>
                  <span className="block text-xs text-ink-muted">{m.shared.platformWalletHint}</span>
                </span>
                <MoneyFigure value={wallet} state={state} />
              </Link>
            )}
            {showDrawer && drawerHref && (
              <Link
                href={drawerHref}
                className={`${row} ${showWallet ? "border-t border-line-soft" : ""}`}
                onClick={() => setOpen(false)}
              >
                <IconBalance size={20} className="shrink-0 text-primary" />
                <span className="min-w-0 flex-1">
                  <span className="block font-bold text-ink">{m.shared.drawerChipShort}</span>
                  <span className={`block text-xs ${typeof drawer === "number" && drawer < 0 ? "text-danger" : "text-ink-muted"}`}>
                    {drawerHint}
                  </span>
                </span>
                <MoneyFigure value={drawer} state={state} />
              </Link>
            )}
          </div>,
          document.body,
        )}
    </>
  );
}
