"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **قشرةُ الزبون على الويب — شريطٌ علويٌّ وأقسامٌ حيث يصل الإبهام**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-١٠-١٧: زبونُ آيفون يتسوّق ويطلب من المتصفّح، ويضيفه
 *  للشاشة الرئيسيّة فيفتح كأنّه تطبيق.)
 *
 * **الأقسامُ الخمسةُ كما في تطبيق أندرويد**: تسوّق · سلّتي · طلباتي ·
 * طلب خاص · حسابي — **والزائرُ تسوّق · حسابي** (السلّةُ والطلباتُ لصاحب حساب).
 *
 * **على الجوّال شريطٌ سفليّ** (`MobileNav`، يختفي فوق ٧٦٨)، **وعلى
 * الحاسوب الأقسامُ في الشريط العلويّ** — فلا يضيع قسمٌ في أيّ مقاس.
 *
 * **وما يُسأل من أيّ شاشة يُركَّب هنا مرّةً**: نافذةُ تأكيد السعر، وسؤالُ
 * مشاركة الموقع، والتقييمُ بعد التسليم، وفقّاعةُ المحادثة مع السائق،
 * **وجرسُ الإشعارات بقناة البثّ** — كلُّها تقرأ الطلباتِ الجاريةَ من
 * قراءةٍ واحدة (`ActiveOrdersProvider`).
 */

import Link from "next/link";
import { useEffect, useMemo, type ReactNode } from "react";
import { usePathname, useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  BrandMark,
  ChatBubble,
  CountBadge,
  MobileNav,
  MobileNavSpacer,
  NotificationBell,
  NotificationToast,
  TopBar,
  TopBarLink,
  IconBag,
  IconCart,
  IconOrder,
  IconStore,
  IconUser,
  useLiveNotifications,
  wsBase,
  type NavItem,
} from "@rahalgo/ui";
import { api, onLogout, tokenStore } from "@/lib/api";
import { dropWebPushOnLogout } from "@/lib/webpush";
import { hasRole, useAuth } from "@/lib/auth";
import { ActiveOrdersProvider, useActiveOrders } from "@/lib/customer/active";
import { CartProvider, useCart } from "@/lib/customer/cart";
import { CustomerProvider } from "@/lib/customer/context";
import { FavoritesProvider } from "@/lib/customer/favorites";
import { LiveShareProvider } from "@/lib/customer/live";
import { ShopToastProvider } from "./toast";
import { OrderingBanner, ServiceBlockBanner, FreeDeliveryBanner } from "./ServiceNotices";
import PwaHints from "./PwaHints";
import QuoteWatcher from "./QuoteWatcher";
import WebPushCard from "@/components/WebPushCard";
import { LiveAsk, LiveBanner } from "./LiveShare";
import { RatingWatcher } from "./feedback";

const m = getMessages(defaultLocale);
const N = m.shop.nav;

/** **شاشاتُ الدخول بلا قشرة** — من جاء ليدخل لا يحتاج سلّةً ولا أقساماً. */
const BARE = ["/login", "/signup", "/forgot"];

/**
 * **الجرسُ وقناةُ البثّ** — لصاحب حسابٍ وحدَه.
 *
 * **والجرسُ والتنبيهُ العابرُ ما دامت الصفحةُ مفتوحة** — وقناةُ البثّ نفسُها
 * توقظ الطلباتِ والمحادثة (`useLiveRefresh`). **وما يصل والصفحةُ مغلقة
 * فإشعارُ المتصفّح** لمن فعّله (`WebPushCard`، ٢٠٢٦-١٠-١٧).
 */
function Bell() {
  // **والخروجُ يُلغي اشتراكَ إشعارات المتصفّح** — من دخل بعده لا تصله إشعاراتُ غيره.
  useEffect(() => onLogout(() => void dropWebPushOnLogout()), []);
  const notif = useLiveNotifications(api, wsBase(), tokenStore.access);
  return (
    <>
      <NotificationBell unread={notif.unread} Link={Link} allHref="/notifications" />
      <NotificationToast notification={notif.toast} onDismiss={notif.dismissToast} />
    </>
  );
}

/** **فقّاعةُ المحادثة** — لكلّ طلبٍ جارٍ أُسند له سائق، **وعدّادُه من المحرّك.** */
function Chats() {
  const { open, chats, reload } = useActiveOrders();
  const threads = useMemo(() => {
    const unread = new Map(chats.map((c) => [c.order_id, c.unread]));
    return (open ?? [])
      .filter((o) => !!o.driver_name)
      .map((o) => ({ id: o.id, number: o.number, peer: o.driver_name ?? "", unread: unread.get(o.id) ?? 0 }));
  }, [open, chats]);
  return <ChatBubble api={api} threads={threads} onRead={reload} />;
}

/**
 * **صاحبُ متجرٍ فتح الأيقونةَ من شاشته الرئيسيّة يصل لوحتَه لا السوق.**
 *
 * (قرارُ المالك ٢٠٢٦-١٠-١٧.) **بيانُ التطبيق يبدأ من `/shop`** — واحدٌ للكلّ،
 * **فمن أضاف الموقعَ من لوحة متجره يفتحه على السوق.** فيُساق إلى `/store`
 * **مرّةً في الجلسة** — ومن عاد إلى السوق بيده يبقى فيه.
 */
const MERCHANT_HOME_ONCE = "rahalgo_merchant_home_once";
function MerchantHome() {
  const { user } = useAuth();
  const path = usePathname();
  const router = useRouter();
  useEffect(() => {
    if (!user || !hasRole(user, "merchant") || path !== "/shop") return;
    const nav = window.navigator as Navigator & { standalone?: boolean };
    const standalone = nav.standalone === true || window.matchMedia?.("(display-mode: standalone)").matches;
    if (!standalone) return;
    try {
      if (window.sessionStorage.getItem(MERCHANT_HOME_ONCE)) return;
      window.sessionStorage.setItem(MERCHANT_HOME_ONCE, "1");
    } catch {
      return;
    }
    router.replace("/store");
  }, [user, path, router]);
  return null;
}

function Live({ children }: { children: ReactNode }) {
  const { open } = useActiveOrders();
  return <LiveShareProvider open={open}>{children}</LiveShareProvider>;
}

function Shell({ children }: { children: ReactNode }) {
  const { user } = useAuth();
  const cart = useCart();
  const path = usePathname() ?? "/shop";

  const nav: NavItem[] = user
    ? [
        { href: "/shop", label: N.shop, icon: IconStore },
        { href: "/cart", label: N.cart, icon: IconCart, count: cart.count },
        { href: "/orders", label: N.orders, icon: IconOrder },
        { href: "/custom", label: N.custom, icon: IconBag },
        { href: "/account", label: N.account, icon: IconUser },
      ]
    : [
        { href: "/shop", label: N.shop, icon: IconStore },
        { href: "/account", label: N.account, icon: IconUser },
      ];
  /* **و`/s/…` قسمٌ من التسوّق** — فيُضاء «تسوّق» وهو فيه. */
  const active = path.startsWith("/s/") || path.startsWith("/stores/") || path.startsWith("/offers") || path.startsWith("/favorites") ? "/shop" : path;

  if (BARE.some((p) => path.startsWith(p))) {
    return <main className="flex min-h-screen flex-1 flex-col">{children}</main>;
  }

  return (
    <div className="flex min-h-screen flex-col">
      <TopBar
        sticky
        start={
          <Link href="/shop" aria-label={N.shop} className="flex items-center">
            {/* **دائريٌّ على أبيض** (ملاحظةُ المالك ٢٠٢٦-١٠-١٩) — كان يذوب في الأزرق. */}
            <BrandMark size={40} disc />
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
        {/* **«لوحة متجري» لصاحب المتجر** (قرارُ المالك ٢٠٢٦-١٠-١٩) — يتسوّق زبوناً ويعود بضغطة. */}
        {user?.roles?.includes("merchant") && (
          <TopBarLink Link={Link} href="/store" tone="plain" className="rounded-full bg-primary px-3 text-on-bright hover:opacity-90">
            <IconStore size={16} />
            {N.myStore}
          </TopBarLink>
        )}
        {user && <Bell />}
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
        <ServiceBlockBanner />
        <FreeDeliveryBanner />
        {user && <LiveBanner />}
        {/* **«فعّل الإشعارات» قدّامه لا في «حسابي»** (ملاحظةُ المالك ٢٠٢٦-١٠-١٩) — **وصاحبُ
            الآيفون يُقال له كيف** (أضفه للشاشة الرئيسيّة)، فلا يدور على الإذن. */}
        {user && <WebPushCard app="customer" variant="banner" notes={path === "/shop"} />}
        {children}
      </main>
      <MobileNavSpacer />
      <MobileNav items={nav} active={active} Link={Link} />
      {user && (
        <>
          <MerchantHome />
          <QuoteWatcher />
          <LiveAsk />
          <RatingWatcher />
          <Chats />
        </>
      )}
    </div>
  );
}

/** **وطبقةُ الطلبات الجارية لصاحب حسابٍ وحدَه** — والضيفُ لا يُنادى عنه شيء. */
function Signed({ children }: { children: ReactNode }) {
  const { user } = useAuth();
  if (!user) return <LiveShareProvider open={null}>{children}</LiveShareProvider>;
  return (
    <ActiveOrdersProvider key={user.id}>
      <Live>{children}</Live>
    </ActiveOrdersProvider>
  );
}

export default function CustomerShell({ children }: { children: ReactNode }) {
  return (
    <CustomerProvider>
      <CartProvider>
        <ShopToastProvider>
          <FavoritesProvider>
            <Signed>
              <Shell>{children}</Shell>
            </Signed>
          </FavoritesProvider>
        </ShopToastProvider>
      </CartProvider>
    </CustomerProvider>
  );
}
