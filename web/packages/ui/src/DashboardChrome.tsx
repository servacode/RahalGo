"use client";

/**
 * الهيكل العائم الموحّد (سايدبار + توب بار) — نسخة واحدة مركزية لكل بوابات اللوحة
 * (إدارة/مندوب/متجر). يُحقن لها api و Link و mediaUrl الخاصة بالتطبيق (حقن تبعية)
 * فتبقى @rahalgo/ui غير مقيّدة بإطار معيّن.
 */

import { useEffect, useState, type ComponentType, type ReactNode } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  IconWallet,
  IconStar,
  IconLogout,
  IconHamburger,
  IconClose,
  IconStore,
} from "./icons";

const m = getMessages(defaultLocale);
const fmt = new Intl.NumberFormat("ar-SY");

type ApiFn = <T>(path: string, init?: RequestInit) => Promise<T>;
type IconType = ComponentType<{ size?: number; strokeWidth?: number; className?: string }>;
type LinkType = ComponentType<{ href: string; className?: string; title?: string; children: ReactNode; onClick?: () => void }>;

export interface ChromeNavItem {
  href: string;
  label: string;
  icon: IconType;
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
  topbarStart,
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
  topbarStart?: ReactNode;
  children: ReactNode;
}) {
  const [menuOpen, setMenuOpen] = useState(false);
  const [summary, setSummary] = useState<Summary | null>(null);
  const [rep, setRep] = useState<Rep | null>(null);

  useEffect(() => {
    api<Summary>("/api/v1/me/summary").then(setSummary).catch(() => undefined);
    if (showRating) api<Rep>("/api/v1/me/reputation").then(setRep).catch(() => undefined);
  }, [api, pathname, showRating]);

  useEffect(() => setMenuOpen(false), [pathname]);

  const isActive = (href: string) => (href === homeHref ? pathname === href : pathname.startsWith(href));
  const activeLabel = nav.find((i) => isActive(i.href))?.label ?? brand;
  const avatar = mediaUrl(summary?.avatar_thumb_url);

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
          <div className="flex h-9 w-9 items-center justify-center rounded-control bg-primary font-bold text-white">
            ر
          </div>
          <span className="font-bold">{brand}</span>
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
            <Link
              key={item.href}
              href={item.href}
              className={`flex items-center gap-2.5 rounded-control px-3 py-2 text-sm transition-colors ${
                active
                  ? "bg-primary-light font-medium text-primary-dark"
                  : "text-ink-muted hover:bg-page hover:text-ink"
              }`}
            >
              <Icon size={17} strokeWidth={active ? 2.2 : 1.8} />
              {item.label}
            </Link>
          );
        })}
      </nav>
    </>
  );

  return (
    <div className="flex min-h-screen bg-page">
      <aside className="sticky top-3 m-3 me-0 hidden h-[calc(100vh-1.5rem)] w-60 shrink-0 flex-col overflow-hidden rounded-card border border-line bg-surface shadow-sm lg:flex">
        {sidebar}
      </aside>

      {menuOpen && (
        <div className="fixed inset-0 z-40 bg-ink/40 lg:hidden" onClick={() => setMenuOpen(false)} />
      )}
      <aside
        className={`fixed inset-y-0 start-0 z-50 flex w-64 flex-col bg-surface shadow-xl transition-transform duration-200 lg:hidden ${
          menuOpen ? "translate-x-0" : "translate-x-full rtl:translate-x-full ltr:-translate-x-full"
        }`}
      >
        {sidebar}
      </aside>

      <div className="flex min-w-0 flex-1 flex-col p-3">
        <header className="mb-3 flex items-center gap-3 rounded-card border border-line bg-surface px-4 py-2.5 shadow-sm">
          <button
            onClick={() => setMenuOpen(true)}
            className="text-ink-muted hover:text-ink lg:hidden"
            aria-label={brand}
          >
            <IconHamburger size={22} />
          </button>
          <h2 className="text-sm font-bold text-ink">{activeLabel}</h2>

          {topbarStart}

          <div className="ms-auto flex items-center gap-2.5">
            {showRating && rep && rep.rating.count > 0 && (
              <Link
                href={ratingHref ?? accountHref}
                className="flex items-center gap-1 rounded-control bg-amber-50 px-2.5 py-1.5 text-sm font-bold text-amber-600 hover:bg-amber-100"
                title={m.rep.reputation.myRating}
              >
                <span dir="ltr">{rep.rating.avg.toFixed(1)}</span>
                <span className="text-amber-400">★</span>
                {rep.rating.trend === "up" && <span className="text-success">▲</span>}
                {rep.rating.trend === "down" && <span className="text-danger">▼</span>}
              </Link>
            )}
            {walletHref && (
              <Link
                href={walletHref}
                className="flex items-center gap-1.5 rounded-control bg-primary-light px-2.5 py-1.5 text-sm font-bold text-primary-dark hover:bg-primary-light/70"
                title={m.site.wallet.title}
              >
                <IconWallet size={15} />
                <span dir="ltr">{fmt.format(summary?.balance ?? 0)}</span>
                <span className="hidden text-xs font-normal sm:inline">{m.common.currency}</span>
              </Link>
            )}
            <Link
              href={accountHref}
              title={summary?.full_name || phone || ""}
              className="flex items-center gap-1.5 rounded-control border border-line py-1 pe-2 ps-1 hover:bg-page"
            >
              <span className="flex h-7 w-7 shrink-0 items-center justify-center overflow-hidden rounded-full bg-primary-light text-sm font-bold text-primary-dark">
                {avatar ? (
                  // eslint-disable-next-line @next/next/no-img-element
                  <img src={avatar} alt="" className="h-full w-full object-cover" />
                ) : (
                  (summary?.full_name || phone || "؟").slice(0, 1)
                )}
              </span>
              <span className="hidden max-w-[7rem] truncate text-sm font-medium text-ink sm:inline">
                {summary?.full_name || phone}
              </span>
            </Link>
            <button
              onClick={onLogout}
              className="flex items-center gap-1.5 rounded-control px-2 py-1.5 text-sm text-danger hover:bg-danger/10"
            >
              <IconLogout size={16} />
              <span className="hidden sm:inline">{m.auth.logout}</span>
            </button>
          </div>
        </header>

        <main className="min-w-0 flex-1 rounded-card border border-line bg-surface p-4 shadow-sm">
          {children}
        </main>
      </div>
    </div>
  );
}
