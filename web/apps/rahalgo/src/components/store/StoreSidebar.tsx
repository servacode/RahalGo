"use client";

/**
 * **سايدبارُ لوحة المتجر على الشاشة العريضة (`lg` فما فوق)**
 *
 * (قرارُ المالك ٢٠٢٦-١١-٠٧: «على اللابتوب اللوحة تبدو جوّالاً ممطوطاً».)
 *
 * **بنودُه هي بنودُ الشريط العلويّ والقائمة الجانبيّة نفسُها وبصلاحياتها** —
 * لا بابَ جديداً ولا اختلافَ في من يرى ماذا: «الطلبات» حين يدير المتجرُ طلباتِه،
 * و«الموظّفون» لا يراه الموظّف، و«مزامنة الأسعار» لمن يستعمل الأمين وحدَه.
 * **وتحت `lg` لا يُرسم شيء** — الجوّالُ والتابلت كما كانا.
 *
 * **ووجهتُه كوجهة سايدبار الإدارة** (`DashboardChrome`): لوحٌ زجاجيٌّ لاصق،
 * وأمارةُ «أنت هنا» شريطٌ في جهة البداية + صبغة + خطٌّ عريض.
 */

import Link from "next/link";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  BrandMark,
  CountBadge,
  IconAdd,
  IconChat,
  IconList,
  IconMoto,
  IconOrder,
  IconPromos,
  IconReceipt,
  IconRotate,
  IconStore,
  IconTrendUp,
  IconUser,
  IconUsers,
  IconWallet,
} from "@rahalgo/ui";
import { useStore } from "@/lib/store/context";
import { useStoreOrders } from "@/lib/store/orders";
import { useTeamRole } from "@/lib/store/team";

const m = getMessages(defaultLocale);
const N = m.store.nav;

type Icon = typeof IconStore;
interface Item {
  href: string;
  label: string;
  icon: Icon;
  count?: number;
  /** مطابقةٌ حرفيّة — `/store` بدايةُ كلّ مسار. */
  exact?: boolean;
}

export default function StoreSidebar({ path, selfManage }: { path: string; selfManage: boolean }) {
  const { store } = useStore();
  const { pending } = useStoreOrders();
  const role = useTeamRole();

  const items: Item[] = [
    ...(selfManage ? [{ href: "/store/orders", label: N.orders, icon: IconOrder, count: pending.length }] : []),
    { href: "/store", label: N.store, icon: IconStore, exact: true },
    { href: "/store/menu", label: N.menu, icon: IconList },
    { href: "/store/menu/new", label: N.addItem, icon: IconAdd },
    ...(store?.uses_ameen === true ? [{ href: "/store/price-sync", label: N.priceSync, icon: IconRotate }] : []),
    { href: "/store/offers", label: N.offers, icon: IconPromos },
    { href: "/store/delivery", label: N.delivery, icon: IconMoto },
    { href: "/store/wallet", label: N.wallet, icon: IconWallet },
    { href: "/store/reports", label: N.reports, icon: IconTrendUp },
    { href: "/store/history", label: N.history, icon: IconReceipt },
    { href: "/store/deliveries", label: N.deliveries, icon: IconMoto },
    { href: "/store/complaints", label: N.complaints, icon: IconChat },
    ...(role === "staff" ? [] : [{ href: "/store/team", label: N.staff, icon: IconUsers }]),
    { href: "/store/account", label: N.account, icon: IconUser },
  ];

  /* **الأطولُ مطابقةً يفوز** — `/store/menu/new` تبدأ بـ`/store/menu`، وغيرُه لا يُضاء معه. */
  const activeHref =
    items
      .filter((i) => (i.exact ? path === i.href : path === i.href || path.startsWith(`${i.href}/`)))
      .sort((a, b) => b.href.length - a.href.length)[0]?.href ?? "";

  const suspended = store?.status === "suspended";
  const state = !store
    ? ""
    : suspended
      ? m.store.state.suspended
      : store.emergency_closed
        ? m.store.state.tempClosed
        : !store.open_now
          ? m.store.state.closedHours
          : m.store.home.openNow;
  const tone = !store
    ? ""
    : suspended
      ? "bg-danger-tint text-danger"
      : store.emergency_closed
        ? "bg-warning-tint text-warning"
        : !store.open_now
          ? "bg-field text-ink-muted"
          : "bg-success-tint text-success";

  return (
    <aside
      className="surface-lit chrome sticky top-3 m-3 me-0 hidden h-[calc(100vh-1.5rem)] w-60 shrink-0 flex-col overflow-hidden rounded-card border border-line lg:flex"
      data-testid="store-sidebar"
    >
      <div className="flex flex-col gap-2 p-3">
        <Link href="/store" aria-label={N.panel} className="flex items-center gap-2">
          {store?.logo_thumb_url ? (
            // eslint-disable-next-line @next/next/no-img-element
            <img src={store.logo_thumb_url} alt="" className="size-10 shrink-0 rounded-control object-cover" />
          ) : (
            <BrandMark size={40} disc />
          )}
          <span className="min-w-0 flex-1 truncate text-sm font-bold">{store?.name ?? N.panel}</span>
        </Link>
        {store && (
          <Link href="/store" className={`w-fit rounded-badge px-3 py-1 text-xs font-bold ${tone}`} data-testid="store-sidebar-state">
            {state}
          </Link>
        )}
      </div>
      <nav aria-label={N.sidebar} className="flex min-h-0 flex-1 flex-col gap-0.5 overflow-y-auto p-2">
        {items.map((it) => {
          const Ico = it.icon;
          const active = it.href === activeHref;
          return (
            <Link
              key={it.href}
              href={it.href}
              aria-current={active ? "page" : undefined}
              className={`flex items-center gap-2.5 rounded-control border-s-2 px-3 py-2 text-sm transition-colors ${
                active
                  ? "border-primary bg-primary-tint font-bold text-ink"
                  : "border-transparent text-ink-muted hover:bg-row-hover hover:text-ink"
              }`}
            >
              <Ico size={17} strokeWidth={active ? 2.4 : 1.8} />
              <span className="min-w-0 flex-1 truncate">{it.label}</span>
              {!!it.count && <CountBadge count={it.count} tone="danger" />}
            </Link>
          );
        })}
      </nav>
    </aside>
  );
}
