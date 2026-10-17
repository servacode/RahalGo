"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **قشرةُ الزبون على الويب — شريطٌ علويٌّ وأقسامٌ حيث يصل الإبهام**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-١٠-١٧: زبونُ آيفون يتسوّق ويطلب من المتصفّح، ويضيفه
 *  للشاشة الرئيسيّة فيفتح كأنّه تطبيق.)
 *
 * **الأقسامُ الأربعةُ كما في تطبيق أندرويد**: تسوّق · سلّتي · طلباتي ·
 * حسابي — **والزائرُ تسوّق · حسابي** (السلّةُ والطلباتُ لصاحب حساب).
 *
 * **على الجوّال شريطٌ سفليّ** (`MobileNav`، يختفي فوق ٧٦٨)، **وعلى
 * الحاسوب الأقسامُ في الشريط العلويّ** — فلا يضيع قسمٌ في أيّ مقاس.
 *
 * **ونافذةُ تأكيد السعر تُسأل من أيّ شاشة** — التسليمُ موقوفٌ على جوابه.
 */

import Link from "next/link";
import type { ReactNode } from "react";
import { usePathname } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  BrandMark,
  CountBadge,
  MobileNav,
  MobileNavSpacer,
  TopBar,
  TopBarLink,
  IconCart,
  IconOrder,
  IconStore,
  IconUser,
  type NavItem,
} from "@rahalgo/ui";
import { useAuth } from "@/lib/auth";
import { CartProvider, useCart } from "@/lib/customer/cart";
import { CustomerProvider } from "@/lib/customer/context";
import { ShopToastProvider } from "./toast";
import { OrderingBanner } from "./ServiceNotices";
import PwaHints from "./PwaHints";
import QuoteWatcher from "./QuoteWatcher";

const m = getMessages(defaultLocale);
const N = m.shop.nav;

/** **شاشاتُ الدخول بلا قشرة** — من جاء ليدخل لا يحتاج سلّةً ولا أقساماً. */
const BARE = ["/login", "/signup", "/forgot"];

function Shell({ children }: { children: ReactNode }) {
  const { user } = useAuth();
  const cart = useCart();
  const path = usePathname() ?? "/shop";

  const nav: NavItem[] = user
    ? [
        { href: "/shop", label: N.shop, icon: IconStore },
        { href: "/cart", label: N.cart, icon: IconCart, count: cart.count },
        { href: "/orders", label: N.orders, icon: IconOrder },
        { href: "/account", label: N.account, icon: IconUser },
      ]
    : [
        { href: "/shop", label: N.shop, icon: IconStore },
        { href: "/account", label: N.account, icon: IconUser },
      ];
  /* **و`/s/…` قسمٌ من التسوّق** — فيُضاء «تسوّق» وهو فيه. */
  const active = path.startsWith("/s/") ? "/shop" : path;

  if (BARE.some((p) => path.startsWith(p))) {
    return <main className="flex min-h-screen flex-1 flex-col">{children}</main>;
  }

  return (
    <div className="flex min-h-screen flex-col">
      <TopBar
        sticky
        start={
          <Link href="/shop" aria-label={N.shop} className="flex items-center">
            <BrandMark size={40} />
          </Link>
        }
      >
        {nav.map((it) => {
          const Icon = it.icon;
          const on = active.startsWith(it.href);
          return (
            <TopBarLink
              key={it.href}
              Link={Link}
              href={it.href}
              tone={on ? "active" : "plain"}
              className="hidden md:inline-flex"
            >
              <span className="relative">
                <Icon size={18} />
                {it.count != null && it.count > 0 && <CountBadge count={it.count} tone="danger" float max={9} />}
              </span>
              {it.label}
            </TopBarLink>
          );
        })}
        {user && (
          <Link
            href="/cart"
            aria-label={N.cart}
            className="taparea relative flex h-10 w-10 items-center justify-center rounded-control text-ink md:hidden"
          >
            <IconCart size={22} />
            {cart.count > 0 && <CountBadge count={cart.count} tone="danger" float max={9} />}
          </Link>
        )}
      </TopBar>
      <main className="flex min-w-0 flex-1 flex-col gap-3 px-3 pb-4 pt-3 sm:px-4">
        <PwaHints />
        <OrderingBanner />
        {children}
      </main>
      <MobileNavSpacer />
      <MobileNav items={nav} active={active} Link={Link} />
      {user && <QuoteWatcher />}
    </div>
  );
}

export default function CustomerShell({ children }: { children: ReactNode }) {
  return (
    <CustomerProvider>
      <CartProvider>
        <ShopToastProvider>
          <Shell>{children}</Shell>
        </ShopToastProvider>
      </CartProvider>
    </CustomerProvider>
  );
}
