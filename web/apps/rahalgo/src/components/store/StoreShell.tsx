"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **قشرةُ لوحة المتجر على الويب — لصاحب متجرٍ معه آيفون**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-١٠-١٧: «صاحبُ المتجر اللي معه آيفون يفتح الموقع
 *  نفسَه ويستقبل طلباته — وصاحبُ أندرويد يبقى على تطبيقه».)
 *
 * **الأقسامُ كما في تطبيق المتجر** (`MainActivity.kt`): الطلبات (حين يدير
 * المتجرُ طلباتِه) · متجري · الأصناف · توصيلة · المحفظة.
 *
 * **وما يُسأل من أيّ شاشةٍ يُركَّب هنا مرّةً**: رنّةُ الطلب الجديد، وزرُّ
 * «فعّل صوت الطلبات» (آيفون يمنع الصوتَ قبل أوّل لمسة)، وقفلُ الشاشة،
 * وعدّادُ الطلبات في عنوان التبويب، **ونافذةُ «السائق دفع مبلغ»** التي لا
 * تُغلق إلّا بجواب.
 *
 * **والدخولُ من `/login` نفسِه** — والتوجيهُ بالدور (`homeFor`).
 */

import Link from "next/link";
import { useEffect, useRef, type ReactNode } from "react";
import { usePathname, useRouter } from "next/navigation";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import {
  BrandMark,
  ButtonLink,
  EmptyState,
  IconMoto,
  IconOrder,
  IconStore,
  IconList,
  IconWallet,
  LoadingState,
  MobileNav,
  MobileNavSpacer,
  NotificationBell,
  NotificationToast,
  ReloadState,
  TopBar,
  TopBarLink,
  useLiveNotifications,
  wsBase,
  type NavItem,
} from "@rahalgo/ui";
import { api, tokenStore } from "@/lib/api";
import { hasRole, useAuth } from "@/lib/auth";
import { StoreProvider, useStore } from "@/lib/store/context";
import { StoreOrdersProvider, useStoreOrders } from "@/lib/store/orders";
import { StoreDeliveriesProvider } from "@/lib/store/deliveries";
import { ShopToastProvider } from "@/components/customer/toast";
import PwaHints from "@/components/customer/PwaHints";
import { fill } from "@/lib/customer/texts";
import StoreBar from "./StoreBar";
import OrderAlert from "./OrderAlert";
import FetchAmountWatcher from "./FetchAmountWatcher";

const m = getMessages(defaultLocale);
const S = m.store;

const ANDROID_HINT = {
  href: "/get/merchant",
  title: S.pwa.androidTitle,
  body: S.pwa.androidBody,
  cta: S.pwa.androidCta,
  offKey: "rahalgo_pwa_android_merchant_off",
};

/** **الجرسُ وقناةُ البثّ** — والقناةُ نفسُها توقظ الطلباتِ والتوصيلات (`useLiveRefresh`). */
function Bell() {
  const notif = useLiveNotifications(api, wsBase(), tokenStore.access);
  return (
    <>
      <NotificationBell unread={notif.unread} Link={Link} allHref="/store/notifications" />
      <NotificationToast notification={notif.toast} onDismiss={notif.dismissToast} />
    </>
  );
}

/**
 * **عدّادُ الطلبات الجديدة في عنوان التبويب** — «(٢) طلب جديد».
 *
 * **ومن ترك الصفحةَ في تبويبٍ ثانٍ يراه** — وهو كلُّ ما يُرى من صفحةٍ غائبة.
 */
function TitleBadge() {
  const { pending } = useStoreOrders();
  const base = useRef("");
  useEffect(() => {
    if (!base.current) base.current = document.title.replace(/^\(\d+\)\s.*?·\s/, "");
    const n = pending.length;
    document.title = n > 0 ? `${fill(S.alert.titleBadge, { n })} · ${base.current}` : base.current;
  }, [pending.length]);
  useEffect(
    () => () => {
      if (base.current) document.title = base.current;
    },
    [],
  );
  return null;
}

function Shell({ children }: { children: ReactNode }) {
  const path = usePathname() ?? "/store";
  const { loading, error, store, selfManage, reload } = useStore();
  const { pending } = useStoreOrders();

  const nav: NavItem[] = [
    ...(selfManage
      ? [{ href: "/store/orders", label: S.nav.orders, icon: IconOrder, count: pending.length }]
      : []),
    { href: "/store", label: S.nav.store, icon: IconStore },
    { href: "/store/menu", label: S.nav.menu, icon: IconList },
    { href: "/store/delivery", label: S.nav.delivery, icon: IconMoto },
    { href: "/store/wallet", label: S.nav.wallet, icon: IconWallet },
  ];
  /* **و`/store` بدايةُ كلّ مسار** — فيُطابَق حرفيّاً وإلّا أُضيء مع كلّ قسم. */
  const active =
    nav
      .map((n) => n.href)
      .filter((h) => h !== "/store")
      .find((h) => path.startsWith(h)) ?? (path === "/store" ? "/store" : "");

  let body: ReactNode = children;
  if (loading) body = <LoadingState variant="list" rows={3} />;
  else if (error) body = <ReloadState label={errorText(error)} onRetry={() => void reload()} />;
  else if (!store) body = <EmptyState icon={IconStore} title={S.noStore} />;

  return (
    <div className="flex min-h-screen flex-col">
      <TopBar
        sticky
        start={
          <Link href="/store" aria-label={S.nav.panel} className="flex items-center">
            <BrandMark size={40} />
          </Link>
        }
      >
        {nav.map((it) => {
          const Icon = it.icon;
          return (
            <TopBarLink
              key={it.href}
              Link={Link}
              href={it.href}
              tone={active === it.href ? "active" : "plain"}
              className="hidden md:inline-flex"
            >
              <Icon size={18} />
              {it.label}
            </TopBarLink>
          );
        })}
        <TopBarLink Link={Link} href="/shop" tone="plain">
          {S.nav.shop}
        </TopBarLink>
        <Bell />
      </TopBar>
      <main className="flex min-w-0 flex-1 flex-col gap-3 px-3 pb-4 pt-3 sm:px-4">
        <PwaHints android={ANDROID_HINT} />
        {store && <StoreBar />}
        {store && selfManage && <OrderAlert />}
        {body}
      </main>
      <MobileNavSpacer />
      <MobileNav items={nav} active={active} Link={Link} />
      {store && <FetchAmountWatcher />}
      <TitleBadge />
    </div>
  );
}

/**
 * **بابُ اللوحة** — الزائرُ إلى الدخول (ويعود إلى حيث كان)، **ومن ليس صاحبَ
 * متجرٍ يُدلّ على السوق** لا يقع على صفحةٍ فارغة. **والمحرّكُ يحرس كلَّ
 * باب** (`RequireRoles("merchant")`) — وهذا تهذيبٌ لا حماية.
 */
function Gate({ children }: { children: ReactNode }) {
  const { user, loading } = useAuth();
  const router = useRouter();
  const path = usePathname() ?? "/store";
  useEffect(() => {
    if (!loading && !user) router.replace(`/login?next=${encodeURIComponent(path)}`);
  }, [loading, user, router, path]);
  if (loading || !user) return <LoadingState variant="inline" />;
  if (!hasRole(user, "merchant")) {
    return (
      <main className="flex min-h-screen flex-1 flex-col justify-center p-4">
        <EmptyState
          icon={IconStore}
          title={S.notMerchant}
          action={<ButtonLink href="/shop">{S.goShop}</ButtonLink>}
        />
      </main>
    );
  }
  return (
    <StoreProvider key={user.id}>
      <StoreOrdersProvider>
        <StoreDeliveriesProvider>{children}</StoreDeliveriesProvider>
      </StoreOrdersProvider>
    </StoreProvider>
  );
}

export default function StoreShell({ children }: { children: ReactNode }) {
  return (
    <ShopToastProvider>
      <Gate>
        <Shell>{children}</Shell>
      </Gate>
    </ShopToastProvider>
  );
}
