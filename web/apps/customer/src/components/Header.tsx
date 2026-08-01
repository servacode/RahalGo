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
import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";
import {
  TopBar,
  TopBarLink,
  TopBarChip,
  TopBarActions,
  TOPBAR_ICON,
  CountBadge,
  LiveNotifications,
  useLiveRefresh,
  IconOrder,
  IconWallet,
  IconUser,
  IconOverview,
  IconCart,
  IconBell,
} from "@rahalgo/ui";
import { homeFor, portalFor, goTo } from "@rahalgo/auth";
import { api, mediaUrl, tokenStore } from "@/lib/api";
import { useAuth, isLoggedIn } from "@/lib/auth";
import { useCart } from "@/lib/cart";

const m = getMessages(defaultLocale);
const N = m.site.nav;
const WS_URL =
  (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080").replace(/^http/, "ws") +
  "/api/v1/ws";

interface Summary {
  full_name: string;
  avatar_thumb_url: string | null;
  balance: number;
}

export default function Header() {
  const { user, logout } = useAuth();
  const { count } = useCart();

  /**
   * تسميةُ السلّة تقول **ما يعنيه الرقم**.
   *
   * كان الرقم عارياً بجانب أيقونةٍ تشبه أيقونة «الطلبات» المجاورة، فيُقرأ
   * «أربعة طلبات» وهو عدد أصناف طلبٍ واحد. وقد قرأه صاحب المنصة هكذا في أوّل
   * تجربةٍ بشرية — ومن قرأه هكذا مرّة يقرؤه كذلك كلَّ مرّة.
   */
  const cartTitle =
    count > 0
      ? m.site.cart.badgeTitle.replace(
          "{n}",
          m.site.cart.itemsCount.replace("{n}", fmtNum(count)),
        )
      : m.terms.cart;
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

  const brand = (
    <Link href="/" className="flex items-center gap-2">
      <span className="flex h-9 w-9 items-center justify-center rounded-control bg-primary font-bold text-white">
        {m.terms.brandInitial}
      </span>
      <span className="hidden font-bold sm:inline">{m.common.appName}</span>
    </Link>
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
              {/* الطلبات فاتورةٌ والسلّة عربة — كانتا سلّتين متجاورتين لا
                  يفرّق بينهما ناظر، فيُقرأ رقمُ السلّة «طلبات». */}
              <TopBarLink
                Link={Link}
                href="/orders"
                title={m.terms.orders}
                aria-label={m.terms.orders}
                tone={pathname.startsWith("/orders") ? "active" : "plain"}
                className="!px-2.5"
              >
                <IconOrder size={TOPBAR_ICON} />
              </TopBarLink>
              <TopBarLink
                Link={Link}
                href="/cart"
                tone="primary"
                title={cartTitle}
                aria-label={cartTitle}
                className="relative !px-2.5"
              >
                <IconCart size={TOPBAR_ICON} />
                <CountBadge count={count} />
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
          <TopBarLink
            Link={Link}
            href="/cart"
            tone="primary"
            title={cartTitle}
            aria-label={cartTitle}
            className="relative !px-2.5"
          >
            <IconCart size={TOPBAR_ICON} />
            <CountBadge count={count} />
          </TopBarLink>
          <TopBarLink Link={Link} href="/login" className="border border-line">
            <IconUser size={TOPBAR_ICON} />
            {N.login}
          </TopBarLink>
        </>
      )}
    </TopBar>
  );
}
