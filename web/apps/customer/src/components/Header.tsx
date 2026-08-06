"use client";

/**
 * شريط موقع الزبون — نفس الشريط العلوي المركزي المستعمل في اللوحات.
 *
 * **بلا قائمة منسدلة عن قصد**: كانت تُخفي خلف نقرةٍ ما هو أصلاً معروضٌ بجانبها
 * (السلة والطلبات والإشعارات)، وتُخفي خلفها ما ليس معروضاً (التقييمات ولوحة
 * التحكم) — فلا هي اختصار ولا هي ترتيب. الآن كل شيء ظاهر: الأيقونة وحدها على
 * الهاتف والاسمُ معها على الشاشات الأوسع، والصورة نفسها زرُّ الحساب.
 */

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { usePathname, useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  TopBar,
  TopBarLink,
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
  IconBell,
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
  /** شكاواه المفتوحة — **الأيقونةُ تظهر بها وتغيب بإغلاقها.** */
  open_tickets: number;
  /** عروضٌ ساريةٌ الآن — **وأيقونةُ العروض تظهر بها وتغيب.** */
  live_offers: number;
}

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
        {logo ? (
          // eslint-disable-next-line @next/next/no-img-element
          <img
            src={logo}
            alt={name || m.common.appName}
            className="h-9 w-9 rounded-control object-cover"
          />
        ) : (
          <span className="flex h-9 w-9 items-center justify-center rounded-control bg-primary font-bold text-on-solid">
            {m.terms.brandInitial}
          </span>
        )}
        <span className="hidden font-bold sm:inline">{name || m.common.appName}</span>
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
          extras={
            <>
              {/* **لا سلّة في الشريط**: صارت عائمةً أسفل الصفحة (`FloatingCart`)
                  لأن الشريط يمضي مع التمرير، فتغيب السلّة في اللحظة التي
                  تُستعمل فيها. وذهابُها يحسم كذلك التباسها بـ«الطلبات»
                  المجاورة — R-88. */}
              {/* **بابُ الشكاوى لا يُفتح إلّا حين يُحتاج.**

                  أيقونةٌ دائمةٌ في شريطٍ ضيّقٍ تزاحم ما يُستعمل كلَّ يوم،
                  **وشكوى تُفتح مرّةً في السنة لا تستحقّ مكاناً دائماً.** فمن
                  اشتكى ظهرت له **حتى تُغلق شكواه**، ومن لا شكوى له لا يراها.

                  (قرارُ المالك ٢٠٢٦-٠٨-٠٣.) */}
              {(summary?.open_tickets ?? 0) > 0 && (
                <TopBarLink
                  Link={Link}
                  href="/complaints"
                  title={m.site.complaint.mine}
                  aria-label={m.site.complaint.mine}
                  tone={pathname.startsWith("/complaints") ? "active" : "plain"}
                
                className="hidden md:flex">
                  <IconSupport size={TOPBAR_ICON} />
                </TopBarLink>
              )}
              {/* **أيقونةُ العروض — والعرضُ يُرى والكودُ يُكتب.**

                  ومن لم يسمع بكود الخصم لا يستفيد منه **ولا يعلم أنّه فاته.**

                  **ولا تظهر على فراغ**: من ضغطها مرّةً فوجد شاشةً خاليةً لم
                  يعد يضغطها، **فيفوته أوّلُ عرضٍ حقيقيّ.** (قرارُ المالك
                  ٢٠٢٦-٠٨-٠٥.) */}
              {(summary?.live_offers ?? 0) > 0 && (
              <TopBarLink
                Link={Link}
                href="/offers"
                title={m.customer.offers.title}
                aria-label={m.customer.offers.title}
                tone={pathname.startsWith("/offers") ? "active" : "plain"}
              
                className="hidden md:flex">
                <IconPromos size={TOPBAR_ICON} />
              </TopBarLink>
              )}
              {/* **المفضّلة — ومن يطلب من مطعمٍ كلَّ أسبوعٍ لا يبحث عنه كلَّ
                  مرّة.**

                  **وتُخفى على الجوّال**: نزلت إلى الشريط السفليّ حيث يصل
                  الإبهام، **وبندان لفعلٍ واحدٍ في شاشةٍ واحدةٍ يجعلان
                  المستخدمَ يسأل أيّهما الصحيح.**

                  **وتظهر دائماً لا عند الامتلاء وحدَه**: العروضُ تُخفى على
                  فراغٍ لأنّها خبرٌ يأتي من المنصة، **والمفضّلةُ بابٌ يملؤه
                  صاحبُه** — ومن لا يراها لا يعرف أنّها له. */}
              <TopBarLink
                Link={Link}
                href="/favorites"
                title={m.customer.favorites.title}
                aria-label={m.customer.favorites.title}
                tone={pathname.startsWith("/favorites") ? "active" : "plain"}
                className="hidden md:flex"
              >
                <IconHeart size={TOPBAR_ICON} />
              </TopBarLink>
              {/* **ادعُ صديقاً** — ومن جلب يُكافأ، والمنصةُ تدفع. */}
              <TopBarLink
                Link={Link}
                href="/invite"
                title={m.customer.invite.title}
                aria-label={m.customer.invite.title}
                tone={pathname.startsWith("/invite") ? "active" : "plain"}
              
                className="hidden md:flex">
                <IconLink size={TOPBAR_ICON} />
              </TopBarLink>
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
        </>
      )}
    </TopBar>
  );
}
