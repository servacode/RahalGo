"use client";

/**
 * شريط موقع الزبون — نفس الشريط العلوي المركزي المستعمل في اللوحات.
 *
 * # ما في الشريط وما في القائمة
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «نزيل هذه العناصر من التوب بار ليكون توب بار
 *  احترافيّاً بعناصر قليلة واضحة».)
 *
 *   الشريطُ  → ما يُفتح كلَّ يوم: التسوّقُ والجرسُ والمحفظةُ والطلبات
 *   القائمةُ → ما يُفتح مرّةً في الشهر: العروضُ والدعوةُ والمفضّلةُ والشكاوى
 *
 * **وكانت قائمةٌ منسدلةٌ قد رُفضت من قبل** — لأنّها كانت تُخفي خلف نقرةٍ ما
 * هو معروضٌ بجانبها أصلاً. **وهذه عكسُها**: ما فيها ليس في الشريط، وما في
 * الشريط ليس فيها. **فلا بابَ له بابان.**
 *
 * **وأربعُ أيقوناتٍ صامتةٍ في صفٍّ واحدٍ لا تُقرأ**: قلبٌ وسلسلةٌ وبوقٌ ودعمٌ
 * بلا أسماء — **والقائمةُ تكتب أسماءَها.**
 */

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { usePathname, useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  TopBar,
  BrandMark,
  TopBarLink,
  AppDownloadChip,
  TopBarChip,
  TopBarActions,
  TOPBAR_ICON,
  LiveNotifications,
  useLiveRefresh,
  IconOrder,
  IconPromos,
  IconHeart,
  IconLink,
  IconSupport,
  IconWallet,
  IconUser,
  IconStore,
  IconOverview,
} from "@rahalgo/ui";
import { homeFor, portalFor, goTo } from "@rahalgo/auth";
import { api, mediaUrl, tokenStore } from "@/lib/api";
import { useAuth, isLoggedIn } from "@/lib/auth";

const m = getMessages(defaultLocale);
const N = m.site.nav;
const WS_URL =
  (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080").replace(/^http/, "ws") +
  "/api/v1/ws";

interface Summary {
  full_name: string;
  avatar_thumb_url: string | null;
  balance: number;
  /* **وذهب `open_tickets` و`live_offers`** — كانا يُخفيان أيقونتين حين
     يفرغان، **وصارت العروضُ والشكاوى بنودَ قائمةٍ ثابتة.** والخادمُ ما زال
     يُرسلهما، **وحقلٌ يُستقبَل ولا يُقرأ يُغري بقراءته لاحقاً كأنّه محسوب.** */
}

/**
 * **بنودُ قائمة الحساب — ثابتةٌ لا تظهر وتغيب.**
 *
 * كانت العروضُ والشكاوى تُخفيان حين يفرغان (`live_offers` و`open_tickets`)
 * — **وكان ذلك صحيحاً في شريطٍ ضيّق**: أيقونةٌ تُفتح على فراغٍ تُعلّم صاحبَها
 * ألّا يضغطها.
 *
 * **وفي قائمةٍ ينقلب الحكم**: القائمةُ هي حيث يُبحث عمّا لا يُرى، **وبندٌ
 * يختفي يجعل صاحبَه يظنّ الميزةَ غيرَ موجودة** — فيسأل «أين شكواي؟» ولا
 * يجد باباً.
 */
const MENU = [
  { href: "/offers", label: m.customer.offers.title, icon: IconPromos },
  { href: "/favorites", label: m.customer.favorites.title, icon: IconHeart },
  { href: "/invite", label: m.customer.invite.title, icon: IconLink },
  /* **والاسمُ واحدٌ في الخمس** — كان «شكاواي» هنا و«الشكاوى والبلاغات» في
     اللوحات الأربع. **والصفحةُ تحمل الاثنين**: ما رفعتَه وما رُفع عليك.
     (قرارُ المالك ٢٠٢٦-٠٨-٠٧: «لازم بدل شكاوي تكون شكاوى وبلاغات، مثل باقي
     اللوحات».) */
  { href: "/complaints", label: m.terms.complaints, icon: IconSupport },
] as const;

export default function Header({
  /** **اسمُ المنصة من الإعدادات** — وفارغٌ يعني «خذ من المعجم». */
  name = "",
  /** **شعارُها** — وفارغٌ يعني أنّ حرفَ العلامة يبقى. */
  logo = null,
}: {
  name?: string;
  logo?: string | null;
} = {}) {
  const { user, logout } = useAuth();
  const router = useRouter();
  const pathname = usePathname();
  const logged = isLoggedIn(user);
  const [summary, setSummary] = useState<Summary | null>(null);

  const loadSummary = useCallback(() => {
    if (logged) api<Summary>("/api/v1/me/summary").then(setSummary).catch(() => undefined);
  }, [logged]);

  useEffect(() => {
    loadSummary();
  }, [loadSummary, pathname]);

  // الرصيد والصورة يتحدّثان لحظياً — بلا إعادة تحميل
  // ("profile" حدث محلي يبثّه AccountSettings عند تغيير الصورة أو الرقم)
  useLiveRefresh(["wallet", "profile"], loadSummary);

  const portal = user ? portalFor(user.roles) : null;

  async function backToDashboard() {
    if (!user || !portal) return;
    try {
      await goTo(homeFor(user.roles));
    } catch {
      /* يبقى في الموقع */
    }
  }

  /**
   * **العلامةُ ومعها بابُ التسوّق.**
   *
   * (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «التسوّق يجب أن يكون بجانب لوغو المنصة».)
   *
   * # ولماذا هنا لا في جهة الأدوات
   *
   * كان في صفّ الأدوات يساراً مع الإشعارات والمحفظة والخروج — **وتلك أدواتُ
   * حسابٍ يفتحها من يعرف ما يريد.** والتسوّقُ **تنقّلٌ لا أداة**: هو الطريقُ
   * الذي يمشي فيه الزائرُ أوّلَ مرّة.
   *
   * **والعينُ تبدأ من العلامة** — في صفحةٍ عربيّةٍ تقع في أقصى اليمين، فأوّلُ
   * ما بعدها أوّلُ ما يُقرأ. **وطرفُ الشريط الآخرُ آخرُ ما يُنظر إليه.**
   *
   * **ويُرى قبل الدخول وبعده**: بقيّةُ الروابط داخل فرع الداخلين، **والتسوّقُ
   * سببُ وجود الموقع** — ومن لم يدخل بعد هو أحوجُ الناس إليه.
   */
  const brand = (
    <>
      <Link href="/" className="flex items-center gap-2">
        {/* **والشعارُ يحلّ محلّ الحرف ولا يُلغيه.**

            (قرارُ المالك ٢٠٢٦-٠٨-٠٦: هويّةُ المنصة — الاسمُ واللوغو.)

            **ومنصّةٌ لم تَرفع شعاراً يجب أن تبقى تعمل**: مربّعٌ فارغٌ في
            الشريط العلويّ **أسوأُ من حرف.** */}
        {/* ══════════════════════════════════════════════════════════
            **اللوغو وحدَه — ولا اسمَ بجانبه**
            ══════════════════════════════════════════════════════════

            (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «اسم المنصة لا داعي له في التوب بار،
             فقط اترك لوغو المنصة».)

            **والاسمُ كان يقول ما يقوله الشعار** — والشعارُ يُقرأ بلمحةٍ
            والاسمُ يُقرأ بحرف. **وشريطٌ فيه علامةٌ واسمُها يكرّر نفسَه في
            أضيق مكانٍ في الشاشة.**

            **ويبقى للقارئ الصوتيّ**: `alt` الشعار اسمُ المنصة **فمن لا
            يرى يسمعه**، ومن لا شعارَ عنده يقرأ حرفَ الاسم. */}
        <BrandMark size={36} />
      </Link>

      <TopBarLink
        Link={Link}
        href="/shop"
        title={N.shop}
        aria-label={N.shop}
        tone={pathname.startsWith("/shop") ? "active" : "primary"}
        /* **ويُخفى على الجوّال** — الشريطُ السفليُّ يحمله بتسميته.
           **وبابان لشيءٍ واحدٍ في شاشةٍ بثلاثمئةٍ وستّين يزاحمان ما لا بديلَ
           له.** (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «الشكلُ كلُّه مو مفهوم».) */
        className="hidden md:flex"
      >
        <IconStore size={TOPBAR_ICON} />
        <span className="hidden sm:inline">{N.shop}</span>
      </TopBarLink>
    </>
  );

  return (
    <TopBar start={brand} sticky>
      {logged ? (
        <TopBarActions
          Link={Link}
          notifications={
            <LiveNotifications
              api={api}
              wsUrl={WS_URL}
              token={tokenStore.access}
              Link={Link}
              allHref="/notifications"
            />
          }
          walletHref="/wallet"
          balance={summary?.balance ?? 0}
          walletIcon={<IconWallet size={TOPBAR_ICON} />}
          accountHref="/account"
          accountLabel={m.terms.account}
          avatarUrl={mediaUrl(summary?.avatar_thumb_url)}
          name={summary?.full_name || user?.phone || ""}
          onLogout={() => {
            logout();
            router.push("/");
          }}
          logoutLabel={m.auth.logout}
          active={pathname}
          menu={MENU}
          extras={
            <>
              {/* **لا سلّة في الشريط**: صارت عائمةً أسفل الصفحة (`FloatingCart`)
                  لأن الشريط يمضي مع التمرير، فتغيب السلّة في اللحظة التي
                  تُستعمل فيها. وذهابُها يحسم كذلك التباسها بـ«الطلبات»
                  المجاورة — R-88. */}
              {/* ══════════════════════════════════════════════════════
                  **وأربعةٌ نزلت إلى القائمة**
                  ══════════════════════════════════════════════════════

                  (قرارُ المالك ٢٠٢٦-٠٨-٠٦.)

                  كانت هنا: الشكاوى · العروضُ · المفضّلةُ · الدعوة —
                  **أربعُ أيقوناتٍ صامتةٍ في صفٍّ واحد** قلبٌ وسلسلةٌ وبوقٌ
                  ودعمٌ، **يُقرأ الواحدُ منها بالتخمين لا بالنظر.**

                  **واثنتان منها كانتا تظهران وتغيبان** بحسب العدد — فيتبدّل
                  ترتيبُ الشريط بين زيارةٍ وأخرى، **ومن حفظ موضعَ زرٍّ وجده
                  مكانَ غيره.**

                  **والطلباتُ وحدَها بقيت**: تُفتح كلَّ يومٍ لا كلَّ شهر. */}
              <TopBarLink
                Link={Link}
                href="/orders"
                title={m.terms.orders}
                aria-label={m.terms.orders}
                tone={pathname.startsWith("/orders") ? "active" : "plain"}
                className="hidden md:flex"
              >
                <IconOrder size={TOPBAR_ICON} />
              </TopBarLink>
              {/* ══════════════════════════════════════════════════════
                  **وزرُّ التطبيق للداخل كما هو للزائر**
                  ══════════════════════════════════════════════════════

                  (كشفه فحصٌ يدويٌّ ٢٠٢٦-٠٨-٠٨: يظهر للزائر ويختفي فورَ
                   الدخول.)

                  وُضع بجانب «تسجيل الدخول» حرفيّاً — **ففُقد بفقدِ جاره.**
                  **والداخلُ أحوجُ إليه**: من طلب مرّةً هو من يريد التطبيق،
                  ومن لم يقرّر الدخولَ بعد قد لا يعود.

                  ولا يظهر إن لم يُضبط رابطٌ ولا رُفع ملفّ — يخفي نفسَه. */}
              <AppDownloadChip label={m.auth.getApp} />
              {/* لوحتي لمن له لوحة فقط — الزبون لا لوحة له وعناصره كلها هنا */}
              {portal && (
                <TopBarChip tone="accent" onClick={backToDashboard} title={m.shared.backToDashboard}>
                  <IconOverview size={TOPBAR_ICON} />
                  <span className="hidden md:inline">{m.shared.backToDashboard}</span>
                </TopBarChip>
              )}
            </>
          }
        />
      ) : (
        <>
          <TopBarLink Link={Link} href="/login" className="border border-line">
            <IconUser size={TOPBAR_ICON} />
            {N.login}
          </TopBarLink>
          {/* **وبجانبه زرُّ التطبيق** — لمن لم يقرّر الدخولَ بعد.
              (طلبُ المالك ٢٠٢٦-٠٨-٠٨: «لازم يكون بالتوب بار».) */}
          <AppDownloadChip label={m.auth.getApp} />
        </>
      )}
    </TopBar>
  );
}
