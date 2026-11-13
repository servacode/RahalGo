"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **هيكلُ سوق الزبون — شريطٌ علويٌّ وتبويباتٌ سفليّةٌ وشريطان عائمان**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (موافقةُ المالك ٢٠٢٦-١١-٠٨ على المخطّط.) **قطعٌ عرضٌ لا منطق**: كلُّ ما فيها يقرأ من السياقات
 * الموجودة (`useCart` · `useAuth` · `useActiveOrders` · `useCustomer`) ولا يكتب إلّا ما كان يكتبه الشريطُ
 * القديم. **وألوانُها كلُّها من `theme.css ٨ب`** — لا لونَ ولا قياسَ هنا.
 *
 * - **جوّال**: شريطٌ علويٌّ يتّخذ خلفيّتَه حين يُمرَّر · تبويباتٌ سفليّة · «طلبك في الطريق» عائمٌ فوقها ·
 *   وفي المتجر «عرض السلة».
 * - **حاسوب**: شعارٌ وعنوانٌ وبحثٌ وروابطُ ومفتاحُ ثيمٍ وجرسٌ وسلّةٌ وحساب، في شريطٍ لاصق.
 */

import Link from "next/link";
import { useEffect, useState, type ReactNode } from "react";
import { usePathname, useRouter } from "next/navigation";
import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";
import {
  BrandMark,
  IconBag,
  IconCart,
  IconChevronDown,
  IconLocation,
  IconMoto,
  IconOrder,
  IconNext,
  IconSearch,
  IconStore,
  IconUser,
  Money,
  usePlatform,
} from "@rahalgo/ui";
import { useAuth } from "@/lib/auth";
import { useActiveOrders } from "@/lib/customer/active";
import { useCart } from "@/lib/customer/cart";
import { useCustomer } from "@/lib/customer/context";
import { statusLabel } from "@/lib/customer/texts";
import { AddressSheet } from "./AddressSheet";
import { ThemeToggle } from "./ShopTheme";

const m = getMessages(defaultLocale);
const N = m.shop.nav;
const SK = m.shop.skin;
const H = m.shop.home;
const A = m.shop.address;

/** **حدثُ البحث من الشريط إلى السوق** — الحمولةُ النصُّ. */
export const SHOP_SEARCH_EVENT = "shop:search";

/** **مساراتُ التصفّح** — عليها يظهر العنوانُ في الشريط العلويّ والشريطُ العائم. */
const BROWSE = ["/shop", "/stores/", "/s/", "/offers", "/favorites"];
const isBrowse = (path: string) => BROWSE.some((p) => path.startsWith(p));

export function ShopTopBar({
  theme,
  onTheme,
  bell,
}: {
  theme: "light" | "dark";
  onTheme: () => void;
  /** **الجرسُ وقناةُ البثّ** — من الغلاف، لصاحب حسابٍ وحدَه. */
  bell: ReactNode;
}) {
  const { user } = useAuth();
  const cart = useCart();
  const { address } = useCustomer();
  const { name } = usePlatform();
  const path = usePathname() ?? "/shop";
  const router = useRouter();
  const [scrolled, setScrolled] = useState(false);
  const [pickAddr, setPickAddr] = useState(false);

  useEffect(() => {
    const on = () => setScrolled(window.scrollY > 8);
    on();
    window.addEventListener("scroll", on, { passive: true });
    return () => window.removeEventListener("scroll", on);
  }, []);

  // **بحثُ الحاسوب في الشريط** — يُبلَّغ به السوقُ بحدثٍ (`SHOP_SEARCH_EVENT`)، ومن كان في صفحةٍ أخرى
  // يُساق إلى السوق بالنصّ في الرابط (`?q=`). **والبحثُ نفسُه في السوق** — لا نداءَ جديداً هنا.
  const [q, setQ] = useState("");
  const typeQ = (v: string) => {
    setQ(v);
    if (path === "/shop") window.dispatchEvent(new CustomEvent(SHOP_SEARCH_EVENT, { detail: v }));
  };
  const submitQ = () => {
    if (path !== "/shop" && q.trim()) router.push(`/shop?q=${encodeURIComponent(q.trim())}`);
  };

  const showAddr = !!user && isBrowse(path);
  const merchant = !!user?.roles?.includes("merchant");
  const initial = Array.from((user?.full_name ?? "").trim())[0] ?? "";

  return (
    <header className="shop-top" data-scrolled={scrolled}>
      <div className="shop-wrap shop-top-in">
        <Link href="/shop" aria-label={N.shop} className="shop-brand">
          <BrandMark size={36} />
          <span className="hidden lg:inline">{name}</span>
        </Link>

        {showAddr && (
          <button type="button" onClick={() => setPickAddr(true)} className="shop-addr min-w-0 flex-1 lg:flex-none">
            <span className="shop-addr-ic">
              <IconLocation size={18} />
            </span>
            <span className="flex min-w-0 items-center gap-1">
              <small>{H.deliverTo}</small>
              <b className="flex items-center gap-1">
                <span className={address ? "min-w-0 truncate" : "min-w-0 line-clamp-2 text-xs leading-snug"}>{address ? address.address_text : A.none}</span>
                <IconChevronDown size={14} className="shrink-0 text-ink-muted" />
              </b>
            </span>
          </button>
        )}
        {!showAddr && <span className="flex-1 lg:hidden" />}

        <form
          className="shop-search"
          role="search"
          onSubmit={(e) => {
            e.preventDefault();
            submitQ();
          }}
        >
          <IconSearch size={18} />
          <input
            type="search"
            value={q}
            onChange={(e) => typeQ(e.target.value)}
            placeholder={H.search}
            aria-label={H.search}
            data-focus="self"
          />
        </form>

        {user && (
          <>
            <Link href="/orders" className="shop-navlink" aria-current={path.startsWith("/orders") ? "page" : undefined}>
              {N.orders}
            </Link>
            <Link href="/custom" className="shop-navlink" aria-current={path.startsWith("/custom") ? "page" : undefined}>
              {N.custom}
            </Link>
          </>
        )}
        {merchant && (
          <Link href="/store" className="shop-navlink">
            <IconStore size={16} />
            {N.myStore}
          </Link>
        )}

        {/* عالجوّال: أيقونةُ «متجري» بالأعلى للتبديل بلمسة (طلبُ المالك ٢٠٢٦-١١-٠٩) */}
        {merchant && (
          <span className="contents lg:hidden">
            <Link href="/store" className="shop-iconbtn" aria-label={N.myStore} title={N.myStore} data-testid="phone-my-store">
              <IconStore size={20} />
            </Link>
          </span>
        )}
        <ThemeToggle theme={theme} onToggle={onTheme} />
        {user && bell}

        {user && (
          <Link href="/cart" className="shop-cartbtn" aria-label={N.cart}>
            <IconCart size={18} />
            {N.cart}
            {cart.count > 0 && (
              <>
                <i>{fmtNum(cart.count)}</i>
                <Money value={cart.subtotal} />
              </>
            )}
          </Link>
        )}
        <Link href="/account" className="shop-acct" aria-label={N.account}>
          <i>{initial ? initial : <IconUser size={18} />}</i>
          {N.account}
        </Link>
      </div>
      {pickAddr && <AddressSheet open={pickAddr} onClose={() => setPickAddr(false)} />}
    </header>
  );
}

/** **التبويباتُ السفليّة** — حيث يصل الإبهام، وتختفي على الحاسوب. */
export function ShopTabBar() {
  const { user } = useAuth();
  const cart = useCart();
  const path = usePathname() ?? "/shop";
  /* **«تسوق» بالنص أكبرَ من الباقي، واثنان يميناً واثنان يساراً** (قرارُ المالك ٢٠٢٦-١١-١٣، مرآةُ التطبيق) —
     يميناً «سلتي» و«طلباتي»، ويساراً «طلب خاص» و«حسابي». */
  const items = user
    ? [
        { href: "/cart", label: N.cart, Icon: IconCart, count: cart.count },
        { href: "/orders", label: N.orders, Icon: IconOrder, count: 0 },
        { href: "/shop", label: N.shop, Icon: IconStore, count: 0, center: true },
        { href: "/custom", label: N.custom, Icon: IconBag, count: 0 },
        { href: "/account", label: N.account, Icon: IconUser, count: 0 },
      ]
    : [
        { href: "/shop", label: N.shop, Icon: IconStore, count: 0 },
        { href: "/account", label: N.account, Icon: IconUser, count: 0 },
      ];
  /* **و`/s/…` وصفحةُ المتجر والعروضُ والمفضّلةُ أقسامٌ من التسوّق** — فيُضاء «تسوّق» وهو فيها. */
  const active = isBrowse(path) ? "/shop" : path;
  return (
    <nav className="shop-tabbar" aria-label={N.shop}>
      {items.map(({ href, label, Icon, count, ...rest }) => (
        <Link
          key={href}
          href={href}
          className={"center" in rest && rest.center ? "shop-tab shop-tab-center" : "shop-tab"}
          aria-current={active.startsWith(href) ? "page" : undefined}
        >
          <span className="shop-tab-icon relative">
            {"center" in rest && rest.center ? (
              /* **شعارُ رحّال غو بدل أيقونة المتجر** (قرارُ المالك ٢٠٢٦-١١-١٣، مرآةُ التطبيق). */
              // eslint-disable-next-line @next/next/no-img-element
              <img src="/pwa/mark-color.png" alt="" width={32} height={32} />
            ) : (
              <Icon size={23} />
            )}
          </span>
          {label}
          {count > 0 && <b>{fmtNum(Math.min(count, 99))}</b>}
        </Link>
      ))}
    </nav>
  );
}

const IN_TRANSIT = ["picked_up", "on_the_way", "at_dropoff"];

/** **الشريطان العائمان فوق التبويبات** — «طلبك في الطريق»، وفي صفحة المتجر «عرض السلة». */
export function ShopFloat() {
  const { user } = useAuth();
  const cart = useCart();
  const { open } = useActiveOrders();
  const path = usePathname() ?? "/shop";
  const order = user && isBrowse(path) ? (open ?? [])[0] : undefined;
  const cartBar = !!user && cart.count > 0 && path.startsWith("/stores/");
  if (!order && !cartBar) return null;
  const total = Math.max(order?.stages?.length ?? 0, 1);
  return (
    <div className="shop-float">
      {order && (
        <Link href={`/orders/${order.id}`} className="shop-trackbar">
          <span className="ic">
            <IconMoto size={22} />
          </span>
          <span className="min-w-0 flex-1">
            <b className="block text-sm">{IN_TRANSIT.includes(order.status) ? SK.onTheWay : statusLabel(order.status)}</b>
            <span className="block truncate text-xs text-ink-muted">
              {`#${fmtNum(order.number)}`}
              {order.items_preview ? ` · ${order.items_preview}` : ""}
            </span>
            <span className="shop-steps" aria-hidden>
              {Array.from({ length: total }, (_, i) => (
                <i key={i} data-on={i <= (order.stage_at ?? 0)} />
              ))}
            </span>
          </span>
          <IconNext size={18} className="shrink-0 text-accent-text" />
        </Link>
      )}
      {cartBar && (
        <Link href="/cart" className="shop-cartbar">
          <span className="flex items-center">
            {SK.viewCart}
            <i>{fmtNum(cart.count)}</i>
          </span>
          <Money value={cart.subtotal} />
        </Link>
      )}
    </div>
  );
}
