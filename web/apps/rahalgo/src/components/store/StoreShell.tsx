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

import { useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { usePathname, useRouter } from "next/navigation";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import {
  ButtonLink,
  ConsoleNavContext,
  EmptyState,
  IconStore,
  LoadingState,
  NotificationToast,
  ReloadState,
  useConsoleTheme,
  useLiveNotifications,
  wsBase,
} from "@rahalgo/ui";
import { api, onLogout, tokenStore } from "@/lib/api";
import { dropWebPushOnLogout } from "@/lib/webpush";
import WebPushCard from "@/components/WebPushCard";
import { hasRole, useAuth } from "@/lib/auth";
import { StoreProvider, useStore } from "@/lib/store/context";
import { StoreOrdersProvider, useStoreOrders } from "@/lib/store/orders";
import { StoreDeliveriesProvider } from "@/lib/store/deliveries";
import { ShopToastProvider } from "@/components/customer/toast";
import PwaHints from "@/components/customer/PwaHints";
import { fill } from "@/lib/customer/texts";
import OrderAlert from "./OrderAlert";
import FetchAmountWatcher from "./FetchAmountWatcher";
import StoreMenu from "./StoreMenu";
import StoreSidebar from "./StoreSidebar";
import StoreTopBar from "./StoreTopBar";
import { PhoneNavBar, PhoneTabBar, usePhoneNav } from "./StorePhoneChrome";
import { useStoreNav } from "./navModel";
import SetupWizard from "./SetupWizard";
import { useSetupWizard, useStoreSections } from "@/lib/store/setup";

const m = getMessages(defaultLocale);
const S = m.store;

const ANDROID_HINT = {
  href: "/get/merchant",
  title: S.pwa.androidTitle,
  body: S.pwa.androidBody,
  cta: S.pwa.androidCta,
  offKey: "rahalgo_pwa_android_merchant_off",
};

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

  const [menuOpen, setMenuOpen] = useState(false);
  const router = useRouter();
  /* **معالجُ الإعداد الأوّل** (مرآةُ `maybeStartWizard`، ٢٠٢٦-١٠-٢٨) — متجرٌ ينقصه
     إعدادٌ إلزاميٌّ لا يرى الأقسامَ حتّى يُكمله. **و«حسابي» يبقى مفتوحاً** — للخروج. */
  const sec = useStoreSections(store?.id ?? "");
  const wizard = useSetupWizard(store, sec);

  /* **الجرسُ وقناةُ البثّ** — والقناةُ نفسُها توقظ الطلباتِ والتوصيلات (`useLiveRefresh`).
     **والخروجُ يُلغي اشتراكَ إشعارات المتصفّح** — كما في قشرة الزبون. */
  useEffect(() => onLogout(() => void dropWebPushOnLogout()), []);
  const notif = useLiveNotifications(api, wsBase(), tokenStore.access, "merchant");

  const nav = useStoreNav(path);
  const phone = usePhoneNav();
  const navCtx = useMemo(
    () => ({ setTitle: phone.setTitle, startEl: phone.startEl, endEl: phone.endEl }),
    [phone.setTitle, phone.startEl, phone.endEl],
  );

  let body: ReactNode = children;
  if (loading) body = <LoadingState variant="list" rows={3} />;
  else if (error) body = <ReloadState label={errorText(error)} onRetry={() => void reload()} />;
  else if (!store) body = <EmptyState icon={IconStore} title={S.noStore} />;

  if (store && wizard.steps && !path.startsWith("/store/account")) {
    return (
      <SetupWizard
        steps={wizard.steps}
        index={wizard.index}
        setIndex={wizard.setIndex}
        sec={sec}
        onFinish={() => {
          wizard.finish();
          router.push("/store");
        }}
      />
    );
  }

  return (
    <ConsoleNavContext.Provider value={navCtx}>
      <div className="flex min-h-screen">
        {/* **على `lg` فما فوق: سايدبارٌ ثابتٌ في جهة البداية** — وتحتها شريطُ التبويب والعنوانُ الكبير. */}
        <StoreSidebar path={path} onMenu={() => setMenuOpen(true)} />
        <div className="flex min-w-0 flex-1 flex-col">
          <StoreTopBar crumb={nav.crumb} unread={notif.unread} onMenu={() => setMenuOpen(true)} />
          <PhoneNavBar title={phone.title} unread={notif.unread} setStartEl={phone.setStartEl} setEndEl={phone.setEndEl} />
          <main className="console-main" data-testid="store-main">
            <div className="flex flex-col gap-3 empty:hidden">
              <PwaHints android={ANDROID_HINT} />
              {store && selfManage && <OrderAlert />}
              {/* **وبجانب زرّ الصوت «فعّل إشعارات الطلبات»** — يرنّ والموقعُ مسكّر. */}
              {store && selfManage && <WebPushCard app="merchant" variant="banner" notes={path === "/store"} />}
            </div>
            {body}
          </main>
        </div>
      </div>
      <StoreMenu open={menuOpen} onClose={() => setMenuOpen(false)} />
      <PhoneTabBar path={path} />
      <NotificationToast notification={notif.toast} onDismiss={notif.dismissToast} />
      {store && <FetchAmountWatcher />}
      <TitleBadge />
    </ConsoleNavContext.Provider>
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
  // **ثيمُ اللوحة (فاتحٌ افتراضاً)** — يُقرأ هنا مرّةً إن لم يسبقه سكربتُ الـ`<head>`.
  useConsoleTheme();
  return (
    <div className="console-root">
      <ShopToastProvider>
        <Gate>
          <Shell>{children}</Shell>
        </Gate>
      </ShopToastProvider>
    </div>
  );
}
