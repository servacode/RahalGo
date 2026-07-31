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
  WalletPill,
  Avatar,
  CountBadge,
  LiveNotifications,
  useLiveRefresh,
  IconOrder,
  IconWallet,
  IconUser,
  IconLogout,
  IconOverview,
  IconStar,
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
      {logged && (
        <>
          {/* الإشعارات والبث الحي — نفس مكوّن اللوحات */}
          <LiveNotifications api={api} wsUrl={WS_URL} token={tokenStore.access} Link={Link} allHref="/notifications" />
          <WalletPill
            Link={Link}
            href="/wallet"
            balance={summary?.balance ?? 0}
            icon={<IconWallet size={15} />}
          />
          <TopBarLink
            Link={Link}
            href="/orders"
            title={m.terms.orders}
            tone={pathname.startsWith("/orders") ? "active" : "plain"}
          >
            <IconOrder size={16} />
            <span className="hidden md:inline">{m.terms.orders}</span>
          </TopBarLink>
        </>
      )}

      {logged && (
        <TopBarLink
          Link={Link}
          href="/ratings"
          title={m.terms.ratings}
          tone={pathname.startsWith("/ratings") ? "active" : "plain"}
        >
          <IconStar size={16} />
          <span className="hidden md:inline">{m.terms.ratings}</span>
        </TopBarLink>
      )}

      {/* لوحة التحكم تظهر لمن له لوحة فقط — الزبون لا لوحة له وعناصره كلها هنا */}
      {logged && portal && (
        <TopBarChip tone="accent" onClick={backToDashboard} title={m.shared.backToDashboard}>
          <IconOverview size={16} />
          <span className="hidden md:inline">{m.shared.backToDashboard}</span>
        </TopBarChip>
      )}

      <TopBarLink
        Link={Link}
        href="/cart"
        tone="primary"
        title={m.terms.cart}
        aria-label={m.terms.cart}
        className="relative !px-2.5"
      >
        <IconCart size={18} />
        <CountBadge count={count} />
      </TopBarLink>

      {logged ? (
        <>
          {/* الصورة نفسها زرُّ الحساب: أقصر طريق إلى ما يخصّ صاحبها */}
          <TopBarLink
            Link={Link}
            href="/account"
            title={m.terms.account}
            tone={pathname.startsWith("/account") ? "active" : "plain"}
            className="border border-line ps-1"
          >
            <Avatar
              url={mediaUrl(summary?.avatar_thumb_url)}
              name={summary?.full_name || user?.phone || ""}
            />
            <span className="hidden max-w-[8rem] truncate font-medium sm:inline">
              {summary?.full_name || user?.phone}
            </span>
          </TopBarLink>

          <TopBarChip
            tone="danger"
            title={m.auth.logout}
            aria-label={m.auth.logout}
            onClick={() => {
              logout();
              router.push("/");
            }}
          >
            <IconLogout size={16} />
            <span className="hidden lg:inline">{m.auth.logout}</span>
          </TopBarChip>
        </>
      ) : (
        <TopBarLink Link={Link} href="/login" className="border border-line">
          <IconUser size={16} />
          {N.login}
        </TopBarLink>
      )}

    </TopBar>
  );
}
