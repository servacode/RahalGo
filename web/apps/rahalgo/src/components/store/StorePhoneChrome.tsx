"use client";

/**
 * **قشرةُ الهاتف — شريطُ التنقّل الشفّاف وشريطُ التبويب (دون `lg`)**
 *
 * (التصميمُ المعتمد ٢٠٢٦-١١-٠٨: Apple HIG.) **شريطُ التنقّل** ٤٤ بكسلاً بلا خلفيّةٍ في أعلى الصفحة (العنوانُ الكبيرُ تحته)،
 * فإذا مرّ العنوانُ الكبيرُ تحته تدرّج الزجاجُ ظهوراً وظهر عنوانُ الصفحة في الوسط.
 * **وشريطُ التبويب** مضبَّبٌ شفّاف بخمسة تبويبات (متجري · الطلبات · أضف · الأصناف · حسابي)، و«أضف» دائرةٌ صغيرة.
 * **التبويباتُ هي تبويباتُ التطبيق وبصلاحيّاتها** (الطلباتُ حين يدير المتجرُ طلباتِه). وما سواها في «حسابي».
 */

import Link from "next/link";
import { useEffect, useRef, useState, type ReactNode } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { IconAdd, IconBell, IconButton, IconList, IconOrder, IconStore, IconUser } from "@rahalgo/ui";
import { useStore } from "@/lib/store/context";
import { useStoreOrders } from "@/lib/store/orders";

const m = getMessages(defaultLocale);
const S = m.store;

/** **شريطُ التنقّل**: يُوفّر فتحتين تملؤهما الصفحةُ (بداية/نهاية) وعنواناً يظهر عند التمرير. */
export function PhoneNavBar({
  title,
  unread,
  setStartEl,
  setEndEl,
}: {
  title: string;
  unread: number;
  setStartEl: (el: HTMLElement | null) => void;
  setEndEl: (el: HTMLElement | null) => void;
}) {
  const bar = useRef<HTMLDivElement>(null);
  useEffect(() => {
    const f = () => bar.current?.style.setProperty("--p", String(Math.max(0, Math.min(1, (window.scrollY - 24) / 30))));
    f();
    window.addEventListener("scroll", f, { passive: true });
    return () => window.removeEventListener("scroll", f);
  }, [title]);
  return (
    <div ref={bar} className="nav-bar" data-testid="phone-navbar">
      <div className="nav-bar-glass" aria-hidden />
      <span className="nav-bar-title" aria-hidden>
        {title}
      </span>
      <span ref={setStartEl} className="flex items-center gap-4 text-base" />
      <span className="flex items-center gap-3">
        <span ref={setEndEl} className="contents" />
        <IconButton icon={IconBell} label={S.shell.notifications} href="/store/notifications" Link={Link} dot={unread > 0} size={24} testId="phone-bell" />
      </span>
    </div>
  );
}

/** **شريطُ التبويب السفليّ.** */
export function PhoneTabBar({ path }: { path: string }) {
  const { selfManage } = useStore();
  const { pending } = useStoreOrders();
  const tabs: { key: string; href: string; label: string; icon: typeof IconStore; badge?: number }[] = [
    { key: "home", href: "/store", label: S.nav.store, icon: IconStore },
    ...(selfManage ? [{ key: "orders", href: "/store/orders", label: S.nav.orders, icon: IconOrder, badge: pending.length }] : []),
    { key: "add", href: "/store/menu/new", label: S.shell.tabAdd, icon: IconAdd },
    { key: "menu", href: "/store/menu", label: S.nav.menu, icon: IconList },
    { key: "account", href: "/store/account", label: S.nav.account, icon: IconUser },
  ];
  const is = (k: string) =>
    k === "home"
      ? path === "/store"
      : k === "orders"
        ? path === "/store/orders" || path.startsWith("/store/orders/")
        : k === "add"
          ? path === "/store/menu/new"
          : k === "menu"
            ? path === "/store/menu" || (path.startsWith("/store/menu/") && path !== "/store/menu/new")
            : false;
  const current = tabs.find((t) => t.key !== "account" && is(t.key))?.key ?? "account";
  return (
    <nav className="tab-bar" aria-label={S.nav.sidebar} data-testid="phone-tabbar" data-mobile-nav>
      {tabs.map((t) => {
        const Ico = t.icon;
        const on = t.key === current;
        let icon: ReactNode = <Ico size={26} strokeWidth={on ? 2.1 : 1.75} />;
        if (t.key === "add")
          icon = (
            <span className="tab-add">
              <IconAdd size={18} strokeWidth={2.6} />
            </span>
          );
        return (
          <Link
            key={t.key}
            href={t.href}
            aria-current={on && t.key !== "add" ? "page" : undefined}
            className="tab-item"
            style={t.key === "add" ? { color: "var(--color-primary)" } : undefined}
            data-testid={`tab-${t.key}`}
          >
            {!!t.badge && (
              <span
                className="absolute top-0.5 grid h-4.5 min-w-4.5 place-items-center rounded-full bg-danger-solid px-1 text-2xs font-bold text-on-solid"
                style={{ insetInlineStart: "calc(50% + 6px)" }}
              >
                {t.badge > 9 ? "9+" : t.badge}
              </span>
            )}
            {icon}
            <span>{t.label}</span>
          </Link>
        );
      })}
    </nav>
  );
}

/** حالةٌ مشتركةٌ بين الشريطين والصفحة: عنوانُ الصفحة وفتحتا الأزرار. */
export function usePhoneNav() {
  const [title, setTitle] = useState("");
  const [startEl, setStartEl] = useState<HTMLElement | null>(null);
  const [endEl, setEndEl] = useState<HTMLElement | null>(null);
  return { title, setTitle, startEl, setStartEl, endEl, setEndEl };
}
