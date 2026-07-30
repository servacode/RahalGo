"use client";

import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { IconOrder, IconWallet, IconUser, IconLogout } from "@rahalgo/ui";
import { useAuth, isLoggedIn } from "@/lib/auth";
import { useCart } from "@/lib/cart";

const m = getMessages(defaultLocale);
const fmt = new Intl.NumberFormat("ar-SY");

export default function Header() {
  const { user, logout } = useAuth();
  const { count } = useCart();
  const router = useRouter();
  const pathname = usePathname();
  const logged = isLoggedIn(user);

  const links = [
    { href: "/orders", label: m.site.nav.orders, icon: IconOrder, auth: true },
    { href: "/wallet", label: m.site.nav.wallet, icon: IconWallet, auth: true },
  ];

  return (
    <header className="sticky top-0 z-40 border-b border-line bg-surface">
      <div className="mx-auto flex max-w-5xl items-center gap-3 px-4 py-3">
        <Link href="/" className="flex items-center gap-2">
          <span className="flex h-9 w-9 items-center justify-center rounded-control bg-primary font-bold text-white">
            ر
          </span>
          <span className="hidden font-bold sm:inline">{m.common.appName}</span>
        </Link>

        <nav className="ms-2 flex items-center gap-1">
          {links.map(
            (l) =>
              (!l.auth || logged) && (
                <Link
                  key={l.href}
                  href={l.href}
                  className={`flex items-center gap-1.5 rounded-control px-2.5 py-1.5 text-sm ${
                    pathname.startsWith(l.href)
                      ? "bg-primary-light font-medium text-primary-dark"
                      : "text-ink-muted hover:text-ink"
                  }`}
                >
                  <l.icon size={16} />
                  <span className="hidden sm:inline">{l.label}</span>
                </Link>
              )
          )}
        </nav>

        <div className="ms-auto flex items-center gap-2">
          <Link
            href="/cart"
            className="relative flex items-center gap-1.5 rounded-control bg-primary px-3 py-1.5 text-sm font-medium text-white"
          >
            🛒 {m.site.nav.cart}
            {count > 0 && (
              <span className="absolute -top-2 -start-2 flex h-5 min-w-5 items-center justify-center rounded-badge bg-accent px-1 text-xs font-bold text-white">
                {fmt.format(count)}
              </span>
            )}
          </Link>
          {logged ? (
            <button
              onClick={() => {
                logout();
                router.push("/");
              }}
              className="flex items-center gap-1 rounded-control px-2 py-1.5 text-sm text-danger hover:bg-danger/10"
              title={m.auth.logout}
            >
              <IconLogout size={16} />
            </button>
          ) : (
            <Link
              href="/login"
              className="flex items-center gap-1.5 rounded-control border border-line px-3 py-1.5 text-sm"
            >
              <IconUser size={16} />
              {m.site.nav.login}
            </Link>
          )}
        </div>
      </div>
    </header>
  );
}
