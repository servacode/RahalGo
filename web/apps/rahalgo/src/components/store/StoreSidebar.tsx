"use client";

/**
 * **سايدبارُ لوحة المتجر على الشاشة العريضة (`lg` فما فوق)**
 *
 * (قرارُ المالك ٢٠٢٦-١١-٠٧: «على اللابتوب اللوحة تبدو جوّالاً ممطوطاً» ثمّ «لسّا التصميم
 * ضعيف جداً — لازم يكون أقوى واحترافي أكثر».)
 *
 * **بنودُه هي بنودُ الشريط العلويّ والقائمة الجانبيّة نفسُها وبصلاحياتها** —
 * لا بابَ جديداً ولا اختلافَ في من يرى ماذا: «الطلبات» حين يدير المتجرُ طلباتِه،
 * و«الموظّفون» لا يراه الموظّف، و«مزامنة الأسعار» لمن يستعمل الأمين وحدَه.
 * **وتحت `lg` لا يُرسم شيء** — الجوّالُ والتابلت كما كانا.
 *
 * **التصميم**: هويّةُ المتجر فوق (شعارٌ واسمٌ وحالٌ بنقطةٍ ونصّ)، ثمّ زرُّ الفعل الأوّل
 * «أضف صنفاً»، ثمّ البنودُ في ثلاث مجموعاتٍ بعناوين (التشغيل · المال والتقارير ·
 * المتجر والحساب) — فخمسةَ عشرَ بنداً تُمسح بالعين لا تُقرأ سطراً سطراً.
 * وأمارةُ «أنت هنا» شريطٌ في جهة البداية + صبغة + خطٌّ عريض (لا لونٌ وحدَه).
 */

import Link from "next/link";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  BrandMark,
  ButtonLink,
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
import { mediaUrl } from "@/lib/api";
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
interface Group {
  key: string;
  title: string;
  items: Item[];
}

export default function StoreSidebar({ path, selfManage }: { path: string; selfManage: boolean }) {
  const { store } = useStore();
  const { pending } = useStoreOrders();
  const role = useTeamRole();

  const groups: Group[] = [
    {
      key: "ops",
      title: N.groupOps,
      items: [
        ...(selfManage ? [{ href: "/store/orders", label: N.orders, icon: IconOrder, count: pending.length }] : []),
        { href: "/store", label: N.store, icon: IconStore, exact: true },
        { href: "/store/menu", label: N.menu, icon: IconList },
        ...(store?.uses_ameen === true ? [{ href: "/store/price-sync", label: N.priceSync, icon: IconRotate }] : []),
        { href: "/store/offers", label: N.offers, icon: IconPromos },
        { href: "/store/delivery", label: N.delivery, icon: IconMoto },
      ],
    },
    {
      key: "money",
      title: N.groupMoney,
      items: [
        { href: "/store/wallet", label: N.wallet, icon: IconWallet },
        { href: "/store/reports", label: N.reports, icon: IconTrendUp },
        { href: "/store/history", label: N.history, icon: IconReceipt },
        { href: "/store/deliveries", label: N.deliveries, icon: IconMoto },
      ],
    },
    {
      key: "account",
      title: N.groupAccount,
      items: [
        { href: "/store/complaints", label: N.complaints, icon: IconChat },
        ...(role === "staff" ? [] : [{ href: "/store/team", label: N.staff, icon: IconUsers }]),
        { href: "/store/account", label: N.account, icon: IconUser },
      ],
    },
  ];
  /* «أضف صنفاً» زرُّ الفعل فوق القائمة — وبقي بابُه في المطابقة كي لا يُضاء «الأصناف» وهو فيه. */
  const addHref = "/store/menu/new";
  const all: Item[] = [...groups.flatMap((g) => g.items), { href: addHref, label: N.addItem, icon: IconAdd }];

  /* **الأطولُ مطابقةً يفوز** — `/store/menu/new` تبدأ بـ`/store/menu`، وغيرُه لا يُضاء معه. */
  const activeHref =
    all
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
  const dot = !store
    ? ""
    : suspended
      ? "bg-danger-solid"
      : store.emergency_closed
        ? "bg-warning-solid"
        : !store.open_now
          ? "bg-ink-dim"
          : "bg-success-solid";
  const logo = store ? mediaUrl(store.logo_thumb_url) : null;

  return (
    <aside
      className="panel sticky top-3 m-3 me-0 hidden h-[calc(100vh-1.5rem)] w-64 shrink-0 flex-col overflow-hidden lg:flex"
      data-testid="store-sidebar"
    >
      <Link
        href="/store"
        aria-label={N.panel}
        className="flex items-center gap-3 border-b border-line-soft p-4 transition-colors hover:bg-row-hover"
      >
        {logo ? (
          <img src={logo} alt="" className="size-11 shrink-0 rounded-control object-cover" />
        ) : (
          <BrandMark size={44} disc />
        )}
        <span className="flex min-w-0 flex-1 flex-col">
          <span className="truncate text-sm font-bold">{store?.name ?? N.panel}</span>
          {store && (
            <span className="mt-0.5 flex items-center gap-1.5 text-xs text-ink-muted" data-testid="store-sidebar-state">
              <span className={`size-2 shrink-0 rounded-badge ${dot}`} aria-hidden />
              <span className="truncate">{state}</span>
            </span>
          )}
        </span>
      </Link>

      {!suspended && (
        <div className="px-3 pt-3">
          <ButtonLink Link={Link} href={addHref} className="w-full" data-testid="store-sidebar-add">
            <IconAdd size={18} />
            {N.addItem}
          </ButtonLink>
        </div>
      )}

      <nav aria-label={N.sidebar} className="flex min-h-0 flex-1 flex-col gap-4 overflow-y-auto p-3">
        {groups.map((g) => (
          <div key={g.key} className="flex flex-col gap-0.5">
            <p className="px-3 pb-1 text-xs font-bold text-ink-dim">{g.title}</p>
            {g.items.map((it) => {
              const Ico = it.icon;
              const active = it.href === activeHref;
              return (
                <Link
                  key={it.href}
                  href={it.href}
                  aria-current={active ? "page" : undefined}
                  className={`flex min-h-10 items-center gap-3 rounded-control border-s-2 px-3 py-2 text-sm transition-colors ${
                    active
                      ? "border-accent bg-primary-tint font-bold text-ink"
                      : "border-transparent text-ink-muted hover:bg-row-hover hover:text-ink"
                  }`}
                >
                  <Ico size={18} strokeWidth={active ? 2.4 : 1.8} />
                  <span className="min-w-0 flex-1 truncate">{it.label}</span>
                  {!!it.count && <CountBadge count={it.count} tone="danger" />}
                </Link>
              );
            })}
          </div>
        ))}
      </nav>
    </aside>
  );
}
