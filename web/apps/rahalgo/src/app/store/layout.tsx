"use client";

/** بوابة المتجر — تستخدم الهيكل العائم المشترك (موحّد مع الإدارة والمندوب). */

import { useEffect } from "react";
import Link from "next/link";
import { usePathname, useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  Badge,
  Button,
  CategoryIcon,
  Select,
  DashboardChrome,
  type ChromeNavItem,
  IconOrder,
  IconCheck,
  IconStore,
  IconStatus,
  IconSupport,
  IconUser,
  IconSettings,
  IconWallet,
  IconWarning,
  BootScreen,
} from "@rahalgo/ui";
import { PasswordGate, FIELD_ROLES_ARE_CUSTOMERS, PANEL_PATHS } from "@rahalgo/auth";
import { useAuth, canAccessPortal } from "@/lib/auth";
import { StoreProvider, useStore } from "@/lib/store";
import { api, mediaUrl, tokenStore } from "@/lib/api";

const m = getMessages(defaultLocale);

/**
 * **قائمةُ المتجر — والطلباتُ الجاريةُ وحدَها تتبع الوضع.**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٠: «عندما تكون المنصّةُ هي من تدير، أخفِ قسمَ
 *  الطلبات من المتجر لأنّه لا يهمّه بهذا الوضع. وأضِف سجلَّ الطلبات
 *  بالحالتين».)
 *
 * **وكان القسمُ يبقى ويُفرَّغ**: في وضع «المنصّة تدير» يفتحه صاحبُه فيجده
 * خالياً — **وقسمٌ فارغٌ يُقرأ عطباً لا قراراً**، فيسأل «أين طلباتي؟»
 * ويظنّ المنصّةَ لا تحوّل إليه.
 *
 * **وسجلُّه يبقى في الوضعين**: «ماذا بعتُ وماذا ضاع منّي؟» سؤالٌ لا علاقةَ
 * له بمن يضغط الأزرار.
 */
function navFor(selfManage: boolean): ChromeNavItem[] {
  return [
    ...(selfManage ? [{ href: "/store", label: m.terms.orders, icon: IconOrder }] : []),
    { href: "/store/history", label: m.terms.ordersHistory, icon: IconCheck },
    { href: "/store/menu", label: m.terms.menu, icon: IconStore },
    { href: "/store/reports", label: m.terms.reports, icon: IconStatus },
    // **الصفحةُ صارت إنذارات لا تقييمات** — والمتجرُ لم يعد له نجوم.
    { href: "/store/reviews", label: m.terms.warnings, icon: IconWarning },
    { href: "/store/complaints", label: m.terms.complaints, icon: IconSupport },
    { href: "/store/wallet", label: m.terms.wallet, icon: IconWallet },
    // **وضبطُ المتجر بيد صاحبه** — الدوامُ ومدّةُ التحضير والحدُّ الأدنى.
    { href: "/store/settings", label: m.terms.settings, icon: IconSettings },
    { href: "/store/account", label: m.terms.account, icon: IconUser },
  ];
}

function PortalChrome({ children }: { children: React.ReactNode }) {
  const { user, loading, logout } = useAuth();
  const { stores, store, loading: storesLoading, select, refresh, selfManage } = useStore();
  const router = useRouter();
  const pathname = usePathname();

  useEffect(() => {
    if (!loading && !canAccessPortal(user)) router.replace("/login");
  }, [user, loading, router]);

  if (loading || storesLoading || !canAccessPortal(user)) {
    return (
      <BootScreen />
    );
  }

  if (!store) {
    /* ══════════════════════════════════════════════════════════════════
       **وشاشةٌ بلا مخرجٍ ليست شاشة**
       ══════════════════════════════════════════════════════════════════

       (كشفه جردُ لوحة المتجر ٢٠٢٦-٠٨-١٠.)

       **كانت رسالةً وحدَها في فراغ** — بلا شريطٍ علويٍّ ولا زرِّ خروج.
       **فمن وقع فيها لا يخرج ولا يذهب إلى حسابه** — يمسح بياناتِ المتصفّح
       أو يترك.

       **والحالُ صارت ممنوعةً عند المنح** (`merchant_needs_store`) — **لكنّها
       تبقى ممكنةً بغيره**: متجرٌ يُعطَّل أو يُحذف، وصاحبُه داخلٌ الآن.
       **وبابٌ يُغلق لا يُغني عن مخرجٍ في الغرفة.** */
    return (
      <main className="flex flex-1 flex-col items-center justify-center gap-4 p-6 text-center">
        <p className="text-ink-muted">{m.merchant.noStores}</p>
        <Button
          variant="secondary"
          onClick={() => {
            logout();
            router.replace("/login");
          }}
        >
          {m.auth.logout}
        </Button>
      </main>
    );
  }

  async function toggleEmergency() {
    if (!store) return;
    await api(`/api/v1/merchant/stores/${store.id}/emergency`, {
      method: "POST",
      body: JSON.stringify({ closed: !store.emergency_closed }),
    });
    await refresh();
  }

  // خاص بالمتجر: اختيار المتجر وحالته وزر الإغلاق الطارئ — يظهر في التوب بار.
  const storeControls = (
    <div className="flex flex-wrap items-center gap-2">
      {stores.length > 1 ? (
        /* **ومن العُدّة لا بيدٍ** — كانت تبني حقلَها بحشوةٍ ولونٍ خاصّين
           بها، **فتفترق عن كلّ قائمةٍ في المنصة.** (٢٠٢٦-٠٨-٠٨.) */
        <Select
          value={store.id}
          onChange={(e) => select(e.target.value)}
          aria-label={m.merchant.header.pickStore}
          className="max-w-40 !py-1 font-bold"
        >
          {stores.map((s) => (
            <option key={s.id} value={s.id}>
              {s.name}
            </option>
          ))}
        </Select>
      ) : (
        <span className="flex min-w-0 items-center gap-1.5 truncate text-sm font-bold">
          {/* **والأيقونةُ تُرسم لا تُطبع.**

              (شهده المالك ٢٠٢٦-٠٨-٠٧: «نصوصٌ إنكليزيّة بكلّ اللوحات».)

              **`category_icon` مفتاحٌ لا نصّ** — قيمتُه `food` و`grocery`،
              **فكانت تُطبع حرفيّاً بجانب اسم المتجر** في كلّ صفحةٍ من
              بوّابته. والمكوّنُ المركزيُّ يحوّلها إلى رسمٍ منذ زمن. */}
          <CategoryIcon name={store.category_icon} size={15} />
          <span className="truncate">{store.name}</span>
        </span>
      )}
      <Badge variant={store.emergency_closed ? "danger" : "success"}>
        {store.emergency_closed ? m.merchant.header.emergencyClosed : m.merchant.header.open}
      </Badge>
      <Button
        variant={store.emergency_closed ? "primary" : "danger"}
        onClick={toggleEmergency}
        className="flex items-center gap-1.5"
      >
        <IconWarning size={15} />
        <span className="hidden sm:inline">
          {store.emergency_closed ? m.merchant.header.reopen : m.merchant.header.closeNow}
        </span>
      </Button>
    </div>
  );

  return (
    <DashboardChrome
      brand={m.merchant.brand}
      nav={navFor(selfManage)}
      pathname={pathname}
      homeHref="/store"
      accountHref="/store/account"
      walletHref="/store/wallet"
      ratingHref="/store/reviews"
      showRating
      api={api}
      mediaUrl={mediaUrl}
      Link={Link}
      wsUrl={`${(process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080").replace(/^http/, "ws")}/api/v1/ws`}
      token={tokenStore.access}
      notificationsHref="/store/notifications"
      phone={user?.phone}
      // زرّ «تسوّق» يتبع دورَ الزبون: بلا الدور لا يستطيع صاحبه أن يطلب
      shopUrl={
        /* **والسوقُ في البيت نفسِه** — مسارٌ لا عنوان. */
        FIELD_ROLES_ARE_CUSTOMERS ? PANEL_PATHS.customer : undefined
      }
      shopLabel={m.shared.shopAsCustomer}
      topbarStart={storeControls}
      onLogout={() => {
        logout();
        router.replace("/login");
      }}
    >
      {children}
    </DashboardChrome>
  );
}

export default function PortalLayout({ children }: { children: React.ReactNode }) {
  // كلمة المرور المؤقتة تُبدَّل قبل أي شاشة — البوابة تحجب اللوحة حتى ذلك
  return (
    <PasswordGate>
      <StoreProvider>
        <PortalChrome>{children}</PortalChrome>
      </StoreProvider>
    </PasswordGate>
  );
}
