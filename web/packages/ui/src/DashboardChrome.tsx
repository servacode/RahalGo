"use client";

/**
 * الهيكل العائم الموحّد (سايدبار + توب بار) — نسخة واحدة مركزية لكل بوابات اللوحة
 * (إدارة/مندوب/متجر). يُحقن لها api و Link و mediaUrl الخاصة بالتطبيق (حقن تبعية)
 * فتبقى @rahalgo/ui غير مقيّدة بإطار معيّن.
 */

import { useCallback, useEffect, useState, type ComponentType, type ReactNode } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { LiveNotifications, useLiveRefresh } from "./Notifications";
import { BrandMark } from "./platform";
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
  IconLogout,
  IconHamburger,
  IconClose,
  IconStore,
} from "./icons";

const m = getMessages(defaultLocale);

/**
 * **قائمةُ حساب اللوحة — بلا بنودٍ زائدة.**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «طبّق الاسم والقائمة مع باقي اللوحات وأضِف فقط
 *  تسجيل الخروج للقائمة».)
 *
 * **وأبوابُ اللوحة في سايدبارها** — تسعةَ عشرَ بنداً بأسمائها ومجموعاتها،
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
   * قائمةُ الادمن **تسعةَ عشرَ بنداً مسطَّحة**: الطلباتُ ثمّ السجلُّ ثمّ
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
}

interface Summary {
  full_name: string;
  avatar_thumb_url: string | null;
  balance: number;
}
interface Rep {
  rating: { avg: number; count: number; trend: "up" | "down" | "flat" };
}

export function DashboardChrome({
  brand,
  nav,
  pathname,
  homeHref,
  accountHref,
  walletHref,
  ratingHref,
  api,
  mediaUrl,
  Link,
  onLogout,
  phone,
  shopUrl,
  shopLabel,
  showRating = false,
  ratingLabel,
  topbarStart,
  wsUrl,
  token,
  notificationsHref,
  children,
}: {
  brand: string;
  nav: ChromeNavItem[];
  pathname: string;
  homeHref: string;
  accountHref: string;
  walletHref?: string;
  ratingHref?: string;
  api: ApiFn;
  mediaUrl: (p: string | null | undefined) => string | null;
  Link: LinkType;
  onLogout: () => void;
  phone?: string;
  shopUrl?: string;
  shopLabel?: string;
  showRating?: boolean;
  /** تسمية شارة التقييم — تختلف بالدور (تقييمي للمتجر، تقييم متاجري للمندوب) */
  ratingLabel?: string;
  topbarStart?: ReactNode;
  /** عنوان قناة البث الحي (ws://…/api/v1/ws) */
  wsUrl?: string;
  /** توكن الوصول للبث — بلا ترويسات في WebSocket */
  token?: string | null;
  /** مسار صفحة الإشعارات الكاملة في هذا التطبيق */
  notificationsHref?: string;
  children: ReactNode;
}) {
  const [menuOpen, setMenuOpen] = useState(false);
  const [summary, setSummary] = useState<Summary | null>(null);
  const [rep, setRep] = useState<Rep | null>(null);

  const loadSummary = useCallback(() => {
    api<Summary>("/api/v1/me/summary").then(setSummary).catch(() => undefined);
    if (showRating) api<Rep>("/api/v1/me/reputation").then(setRep).catch(() => undefined);
  }, [api, showRating]);

  useEffect(() => {
    loadSummary();
  }, [loadSummary, pathname]);

  // الرصيد والتقييم والصورة في الشريط العلوي تتحدّث لحظياً بلا إعادة تحميل
  // ("profile" حدث محلي يبثّه AccountSettings عند تغيير الصورة أو الرقم)
  useLiveRefresh(["wallet", "rating", "profile"], loadSummary);

  useEffect(() => setMenuOpen(false), [pathname]);

  const isActive = (href: string) => (href === homeHref ? pathname === href : pathname.startsWith(href));
  // **وعنوانُ الشريط اسمُ القسم المفتوح** — **وفارغٌ إن لم يُطابق شيء**،
  // ولا يسقط إلى تسمية البوّابة (قرارُ المالك ٢٠٢٦-٠٨-٠٦).
  const activeLabel = nav.find((i) => isActive(i.href))?.label ?? "";

  async function shopAsCustomer() {
    if (!shopUrl) return;
    try {
      const { code } = await api<{ code: string }>("/api/v1/auth/handoff", { method: "POST" });
      window.location.href = `${shopUrl}/sso?code=${encodeURIComponent(code)}`;
    } catch {
      /* يبقى في لوحته */
    }
  }

  const sidebar = (
    <>
      <div className="flex items-center justify-between border-b border-line p-4">
        <div className="flex items-center gap-2">
          {/* **علامةُ المنصة من الإعدادات** — شعارٌ إن رُفع وإلّا أوّلُ حرفٍ
              من الاسم. **والاسمُ بجانبها اسمُ المنصة لا اسمُ اللوحة**: كانت
              الأربعُ تكتب أربعةَ أسماءٍ مختلفةً في شيفرتها — «رحّال غو»
              و«بوّابة المتجر» وعنوانَي دخولِ السائق والمندوب.
              (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «يجب أن يأتي من الإعدادات فقط».) */}
          {/* ══════════════════════════════════════════════════════
              **اللوغو وحدَه — لا اسمٌ ولا تسميةُ بوّابة**
              ══════════════════════════════════════════════════════

              (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «لا يوجد داعٍ لكتابة بوّابة السائق
               والمندوب والمتجر والادمن — فقط يظهر لوغو المنصة بالسايدبار
               بدون اسمٍ للمنصة أو شيءٍ آخر».)

              **وكانت الأربعُ تكتب أربعةَ أسماءٍ مختلفةً في شيفرتها**، ثمّ
              صارت تكتب اسمَ المنصة من الإعدادات — **والاثنان زائدان**:
              من فتح بوّابتَه يعرف أيَّها فتح، **والسايدبارُ تحتها يقول
              دورَه بتسعةَ عشرَ بنداً.** */}
          <BrandMark size={36} />
        </div>
        <button
          onClick={() => setMenuOpen(false)}
          className="text-ink-muted hover:text-ink lg:hidden"
          aria-label={m.common.cancel}
        >
          <IconClose size={20} />
        </button>
      </div>

      {shopUrl && (
        <div className="p-3 pb-0">
          <button
            onClick={shopAsCustomer}
            className="flex w-full items-center gap-2.5 rounded-control bg-accent/10 px-3 py-2 text-sm font-medium text-accent-dark transition-colors hover:bg-accent/20"
          >
            <IconStore size={17} />
            {shopLabel}
          </button>
        </div>
      )}

      <nav className="flex-1 space-y-1 overflow-y-auto p-3">
        {nav.map((item) => {
          const active = isActive(item.href);
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
              <Link
                href={item.href}
                className={`flex items-center gap-2.5 rounded-control px-3 py-2 text-sm transition-colors ${
                  active
                    ? "bg-primary-light font-medium text-primary-strong"
                    : "text-ink-muted hover:bg-page hover:text-ink"
                }`}
              >
                <Icon size={17} strokeWidth={active ? 2.2 : 1.8} />
                {item.label}
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
    <div className="flex min-h-screen">
      <aside className="sticky top-3 m-3 me-0 hidden h-[calc(100vh-1.5rem)] w-60 shrink-0 flex-col surface-lit overflow-hidden surface lg:flex">
        {sidebar}
      </aside>

      {menuOpen && (
        <div className="fixed inset-0 z-40 scrim lg:hidden" onClick={() => setMenuOpen(false)} />
      )}
      <aside
        /* **والدُّرجُ الجوّاليُّ يُضبّب ويثقل — وهو يعلو المحتوى.**

           (المرحلةُ الثالثة ٢٠٢٦-٠٨-٠٦.)

           كان `bg-surface` بلا تضبيب — **وكان مصمتاً يوم كان السطحُ مصمتاً.**
           ولمّا صار السطحُ زجاجاً بخمسةٍ وستّين **صار محتوى الصفحة يُرى من
           خلفه حادّاً**، فتُقرأ بنودُ القائمة فوق جدولٍ يتحرّك.

           **و`raised` لا `surface`**: ما يعلو كلَّ شيءٍ يبقى أثقل — كالنوافذ
           والقوائم، **وقائمةُ تنقّلٍ لا تُقرأ ليست قائمة.** */
        className={`surface-lit fixed inset-y-0 start-0 z-50 flex w-64 flex-col bg-raised elev-4 transition-transform duration-200 lg:hidden ${
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
                onClick={() => setMenuOpen(true)}
                className="lg:hidden"
                aria-label={brand}
              >
                <IconHamburger size={22} />
              </TopBarChip>
              <h2 className="text-sm font-bold text-ink">{activeLabel}</h2>
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
            walletHref={walletHref}
            balance={summary?.balance ?? 0}
            walletIcon={<IconWallet size={TOPBAR_ICON} />}
            accountHref={accountHref}
            accountLabel={m.terms.account}
            avatarUrl={mediaUrl(summary?.avatar_thumb_url)}
            name={summary?.full_name || phone || ""}
            onLogout={onLogout}
            logoutLabel={m.auth.logout}
            active={pathname}
            extras={
              /* **والعضوُ يُحرَس كما يُحرَس الكائن.**

                 كان `rep && rep.rating.count` — **يسأل عن الكائن ويثق بعضوه.**
                 وردٌّ ناقصُ `rating` (٢٠٠ بجسمٍ غير متوقّع) يرمي هنا،
                 **والرميةُ في `DashboardChrome` تُبيّض اللوحةَ كلَّها** — لا
                 شريطَ ولا قائمةَ ولا محتوى، بل «حدث خطأ في التطبيق».
                 (وقع فعلاً في فحصٍ بمتصفّح ٢٠٢٦-٠٨-٠٦.)

                 **وخسارةُ نجمةٍ في الشريط أهونُ من خسارة اللوحة.** */
              showRating &&
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
              )
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
