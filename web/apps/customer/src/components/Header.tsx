"use client";

import Link from "next/link";
import { useCallback, useEffect, useRef, useState } from "react";
import { usePathname, useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  LiveNotifications,
  useLiveRefresh,
  IconOrder,
  IconWallet,
  IconUser,
  IconLogout,
  IconOverview,
  IconChevronDown,
  IconStar,
} from "@rahalgo/ui";
import { homeFor, portalFor, goTo } from "@rahalgo/auth";
import { api, mediaUrl, tokenStore } from "@/lib/api";
import { useAuth, isLoggedIn } from "@/lib/auth";
import { useCart } from "@/lib/cart";

const m = getMessages(defaultLocale);
const N = m.site.nav;
const fmt = new Intl.NumberFormat("ar-SY");
const WS_URL = (process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080").replace(/^http/, "ws") + "/api/v1/ws";

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

  // الرصيد يتحدّث لحظياً عند أي حركة على المحفظة — بلا إعادة تحميل
  useLiveRefresh(["wallet"], loadSummary);

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

  const avatar = mediaUrl(summary?.avatar_thumb_url);

  return (
    <header className="sticky top-0 z-40 border-b border-line bg-surface">
      <div className="mx-auto flex max-w-6xl items-center gap-3 px-4 py-3">
        <Link href="/" className="flex items-center gap-2">
          <span className="flex h-9 w-9 items-center justify-center rounded-control bg-primary font-bold text-white">
            {m.terms.brandInitial}
          </span>
          <span className="hidden font-bold sm:inline">{m.common.appName}</span>
        </Link>

        <div className="ms-auto flex items-center gap-2">
          {logged && (
            <>
              {/* الإشعارات والبث الحي — نفس المكوّن المركزي المستعمل في اللوحات */}
              <LiveNotifications
                api={api}
                wsUrl={WS_URL}
                token={tokenStore.access}
                Link={Link}
              />
              {/* رصيد المحفظة */}
              <Link
                href="/wallet"
                className="flex items-center gap-1.5 rounded-control bg-primary-light px-2.5 py-1.5 text-sm font-bold text-primary-dark hover:bg-primary-light/70"
                title={m.terms.wallet}
              >
                <IconWallet size={15} />
                <span dir="ltr">{fmt.format(summary?.balance ?? 0)}</span>
                <span className="hidden text-xs font-normal sm:inline">{m.common.currency}</span>
              </Link>
              {/* اختصار طلباتي */}
              <Link
                href="/orders"
                className={`flex items-center gap-1.5 rounded-control px-2.5 py-1.5 text-sm ${
                  pathname.startsWith("/orders")
                    ? "bg-primary-light font-medium text-primary-dark"
                    : "text-ink-muted hover:text-ink"
                }`}
                title={m.terms.orders}
              >
                <IconOrder size={16} />
                <span className="hidden md:inline">{m.terms.orders}</span>
              </Link>
            </>
          )}

          {/* اختصار السلة */}
          <Link
            href="/cart"
            className="relative flex items-center gap-1.5 rounded-control bg-primary px-3 py-1.5 text-sm font-medium text-white"
          >
            🛒 <span className="hidden sm:inline">{m.terms.cart}</span>
            {count > 0 && (
              <span className="absolute -top-2 -start-2 flex h-5 min-w-5 items-center justify-center rounded-badge bg-accent px-1 text-xs font-bold text-white">
                {fmt.format(count)}
              </span>
            )}
          </Link>

          {logged ? (
            /* قائمة البروفايل */
            <div className="relative" ref={menuRef}>
              <button
                onClick={() => setOpen((o) => !o)}
                className="flex items-center gap-1.5 rounded-control border border-line py-1 pe-2 ps-1 hover:bg-page"
              >
                <span className="flex h-7 w-7 shrink-0 items-center justify-center overflow-hidden rounded-full bg-primary-light text-sm font-bold text-primary-dark">
                  {avatar ? (
                    // eslint-disable-next-line @next/next/no-img-element
                    <img src={avatar} alt="" className="h-full w-full object-cover" />
                  ) : (
                    (summary?.full_name || user?.phone || m.terms.avatarFallback).slice(0, 1)
                  )}
                </span>
                <span className="hidden max-w-[8rem] truncate text-sm font-medium text-ink sm:inline">
                  {summary?.full_name || user?.phone}
                </span>
                <IconChevronDown size={15} className="text-ink-muted" />
              </button>

              {open && (
                <div className="absolute end-0 mt-1 w-52 overflow-hidden rounded-card border border-line bg-surface py-1 shadow-lg">
                  <MenuLink href="/account" icon={<IconUser size={16} />} label={m.terms.account} />
                  <MenuLink href="/wallet" icon={<IconWallet size={16} />} label={m.terms.wallet} />
                  <MenuLink href="/orders" icon={<IconOrder size={16} />} label={m.terms.orders} />
                  <MenuLink href="/ratings" icon={<IconStar size={16} />} label={m.terms.ratings} />
                  <MenuLink href="/cart" icon={<span className="text-base">🛒</span>} label={m.terms.cart} />
                  {portal && (
                    <button
                      onClick={backToDashboard}
                      className="flex w-full items-center gap-2.5 px-3 py-2 text-sm text-accent-dark hover:bg-accent/10"
                    >
                      <IconOverview size={16} />
                      {m.shared.backToDashboard}
                    </button>
                  )}
                  <div className="my-1 border-t border-line" />
                  <button
                    onClick={() => {
                      logout();
                      router.push("/");
                    }}
                    className="flex w-full items-center gap-2.5 px-3 py-2 text-sm text-danger hover:bg-danger/10"
                  >
                    <IconLogout size={16} />
                    {m.auth.logout}
                  </button>
                </div>
              )}
            </div>
          ) : (
            <Link
              href="/login"
              className="flex items-center gap-1.5 rounded-control border border-line px-3 py-1.5 text-sm"
            >
              <IconUser size={16} />
              {N.login}
            </Link>
          )}
        </div>
      </div>
    </header>
  );
}

function MenuLink({ href, icon, label }: { href: string; icon: React.ReactNode; label: string }) {
  return (
    <Link href={href} className="flex items-center gap-2.5 px-3 py-2 text-sm text-ink hover:bg-page">
      {icon}
      {label}
    </Link>
  );
}
