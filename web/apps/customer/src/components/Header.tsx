"use client";

/** شريط موقع الزبون — نفس الشريط العلوي المركزي المستعمل في اللوحات. */

import Link from "next/link";
import { useCallback, useEffect, useRef, useState } from "react";
import { usePathname, useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  TopBar,
  TopBarLink,
  TopBarChip,
  WalletPill,
  Avatar,
  CountBadge,
  MenuPanel,
  MenuItem,
  LiveNotifications,
  useLiveRefresh,
  IconOrder,
  IconWallet,
  IconUser,
  IconLogout,
  IconOverview,
  IconChevronDown,
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
  const [open, setOpen] = useState(false);
  const menuRef = useRef<HTMLDivElement>(null);

  const loadSummary = useCallback(() => {
    if (logged) api<Summary>("/api/v1/me/summary").then(setSummary).catch(() => undefined);
  }, [logged]);

  useEffect(() => {
    loadSummary();
  }, [loadSummary, pathname]);

  // الرصيد والصورة يتحدّثان لحظياً — بلا إعادة تحميل
  // ("profile" حدث محلي يبثّه AccountSettings عند تغيير الصورة أو الرقم)
  useLiveRefresh(["wallet", "profile"], loadSummary);

  useEffect(() => setOpen(false), [pathname]);

  useEffect(() => {
    function onClick(e: MouseEvent) {
      if (menuRef.current && !menuRef.current.contains(e.target as Node)) setOpen(false);
    }
    document.addEventListener("mousedown", onClick);
    return () => document.removeEventListener("mousedown", onClick);
  }, []);

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
        <div className="relative" ref={menuRef}>
          <TopBarChip
            onClick={() => setOpen((o) => !o)}
            className="border border-line ps-1 hover:bg-page"
          >
            <Avatar url={mediaUrl(summary?.avatar_thumb_url)} name={summary?.full_name || user?.phone || ""} />
            <span className="hidden max-w-[8rem] truncate font-medium text-ink sm:inline">
              {summary?.full_name || user?.phone}
            </span>
            <IconChevronDown size={15} className="text-ink-muted" />
          </TopBarChip>

          {open && (
            <MenuPanel>
              <MenuItem Link={Link} href="/account" icon={<IconUser size={16} />} label={m.terms.account} />
              <MenuItem Link={Link} href="/wallet" icon={<IconWallet size={16} />} label={m.terms.wallet} />
              <MenuItem Link={Link} href="/orders" icon={<IconOrder size={16} />} label={m.terms.orders} />
              <MenuItem Link={Link} href="/ratings" icon={<IconStar size={16} />} label={m.terms.ratings} />
              <MenuItem Link={Link} href="/notifications" icon={<IconBell size={16} />} label={m.shared.notifications.title} />
              <MenuItem Link={Link} href="/cart" icon={<IconCart size={16} />} label={m.terms.cart} />
              {portal && (
                <MenuItem
                  icon={<IconOverview size={16} />}
                  label={m.shared.backToDashboard}
                  onClick={backToDashboard}
                  tone="accent"
                />
              )}
              <div className="my-1 border-t border-line" />
              <MenuItem
                icon={<IconLogout size={16} />}
                label={m.auth.logout}
                tone="danger"
                onClick={() => {
                  logout();
                  router.push("/");
                }}
              />
            </MenuPanel>
          )}
        </div>
      ) : (
        <TopBarLink Link={Link} href="/login" className="border border-line">
          <IconUser size={16} />
          {N.login}
        </TopBarLink>
      )}
    </TopBar>
  );
}
