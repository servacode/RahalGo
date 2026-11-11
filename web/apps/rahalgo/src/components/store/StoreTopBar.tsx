"use client";

/**
 * **الشريطُ العلويّ للوحة المتجر على الحاسوب — ٥٢ بكسلاً**
 *
 * (التصميمُ المعتمد ٢٠٢٦-١١-٠٨.) فتاتٌ بمبدّل المتجر والصفحةِ الحاليّة · بحثٌ (⌘K) · وضعُ الليل/النهار ·
 * مساعدة · إشعارات · «تسوّق كزبون» صغيراً · الصورةُ الرمزيّة (تفتح القائمة القديمة بكلّ أبوابها وخروجها).
 * **والبحثُ يفتح «الأصناف» مصفّاةً بما كُتب** — الاستعلامُ نفسُه الذي تعرفه الصفحة (`?q=`). وتحت `lg` لا يُرسم شيء.
 */

import Link from "next/link";
import { useEffect, useRef, useState, type FormEvent } from "react";
import { usePathname, useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  ActionMenu,
  ButtonLink,
  IconBell,
  IconButton,
  IconCart,
  IconHelp,
  IconMoon,
  IconSearch,
  IconSun,
  IconUpDown,
  useConsoleTheme,
  type ActionMenuItem,
} from "@rahalgo/ui";
import { mediaUrl } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useStore } from "@/lib/store/context";
import { initials } from "./StoreSidebar";
import { MENU_SEARCH_EVENT } from "@/lib/store/menuSearch";

const m = getMessages(defaultLocale);
const N = m.store.nav;
const SH = m.store.shell;

function StoreMark({ name, logo }: { name: string; logo: string | null }) {
  return logo ? (
    <img src={logo} alt="" className="hero-logo object-cover" data-size="sm" />
  ) : (
    <span className="hero-logo" data-size="sm">
      {name.slice(0, 1) || SH.noName}
    </span>
  );
}

export default function StoreTopBar({
  crumb,
  unread,
  onMenu,
}: {
  crumb: string;
  unread: number;
  onMenu: () => void;
}) {
  const router = useRouter();
  const path = usePathname() ?? "";
  const { user } = useAuth();
  const { store, stores, select } = useStore();
  const { theme, toggle } = useConsoleTheme();
  const [q, setQ] = useState("");
  const box = useRef<HTMLInputElement>(null);

  // **⌘K / Ctrl+K** تُركّز البحث.
  useEffect(() => {
    const onKey = (e: KeyboardEvent) => {
      if ((e.metaKey || e.ctrlKey) && e.key.toLowerCase() === "k") {
        e.preventDefault();
        box.current?.focus();
      }
    };
    window.addEventListener("keydown", onKey);
    return () => window.removeEventListener("keydown", onKey);
  }, []);

  const submit = (e: FormEvent) => {
    e.preventDefault();
    const t = q.trim();
    if (!t) return;
    // **وهو في «الأصناف» لا يُعاد تحميلُها** — حدثٌ تلتقطه الصفحةُ فتبحث بما كُتب.
    if (path === "/store/menu") window.dispatchEvent(new CustomEvent(MENU_SEARCH_EVENT, { detail: t }));
    else router.push(`/store/menu?q=${encodeURIComponent(t)}`);
  };

  const name = store?.name ?? "";
  const mark = <StoreMark name={name} logo={store ? mediaUrl(store.logo_thumb_url) : null} />;
  const switcher: ActionMenuItem[] = stores.map((s) => ({
    key: s.id,
    label: s.name,
    onSelect: () => select(s.id),
    disabled: s.id === store?.id,
  }));

  return (
    <header className="console-top" data-testid="store-topbar">
      <div className="flex items-center gap-rg-sm text-body text-ink-muted">
        {stores.length > 1 ? (
          <ActionMenu
            variant="ghost"
            size="sm"
            chevron={false}
            ariaLabel={SH.switchStore}
            label={
              <span className="flex items-center gap-rg-sm font-bold text-ink">
                {mark}
                {name}
                <IconUpDown size={13} />
              </span>
            }
            items={switcher}
          />
        ) : (
          <span className="crumb-switch cursor-default">
            {mark}
            {name}
          </span>
        )}
        <span className="text-line" aria-hidden>
          /
        </span>
        <span className="font-bold text-ink" data-testid="store-crumb">
          {crumb}
        </span>
      </div>

      <form onSubmit={submit} role="search" className="relative mx-auto w-[min(24rem,40vw)]">
        <span className="pointer-events-none absolute inset-y-0 start-2.5 flex items-center text-ink-muted" aria-hidden>
          <IconSearch size={16} />
        </span>
        <input
          ref={box}
          type="search"
          value={q}
          onChange={(e) => setQ(e.target.value.slice(0, 60))}
          placeholder={SH.search}
          aria-label={SH.searchLabel}
          data-testid="store-topsearch"
          className="search-field !bg-page !ps-9 !pe-12"
        />
        <kbd className="kbd pointer-events-none absolute end-2 top-1/2 -translate-y-1/2" aria-hidden>
          ⌘K
        </kbd>
      </form>

      <div className="flex items-center gap-rg-xs">
        <IconButton
          icon={theme === "dark" ? IconSun : IconMoon}
          label={theme === "dark" ? SH.themeLight : SH.themeDark}
          onClick={toggle}
          testId="store-theme-toggle"
        />
        <IconButton icon={IconHelp} label={SH.help} href="/help" Link={Link} />
        <IconButton icon={IconBell} label={SH.notifications} href="/store/notifications" Link={Link} dot={unread > 0} testId="store-bell" />
        <span className="mx-rg-sm h-5 w-px bg-line-soft" aria-hidden />
        <ButtonLink Link={Link} href="/shop" variant="ghost" size="sm">
          <IconCart size={15} />
          {N.shopAsCustomer}
        </ButtonLink>
        <button type="button" onClick={onMenu} aria-label={N.more} className="ms-rg-xs rounded-full" data-testid="store-topbar-avatar">
          <span className="avatar">{initials(user?.full_name || name)}</span>
        </button>
      </div>
    </header>
  );
}
