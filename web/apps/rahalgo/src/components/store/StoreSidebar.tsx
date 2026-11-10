"use client";

/**
 * **سايدبارُ لوحة المتجر على الشاشة العريضة (`lg` فما فوق) — ٢٣٢ بكسلاً**
 *
 * (التصميمُ المعتمد ٢٠٢٦-١١-٠٨: «Stripe/Shopify».) علامةُ المنصّة فوق، ثمّ ثلاثُ مجموعاتٍ بعناوين
 * (التشغيل · المال والتقارير · المتجر والحساب) بأيقوناتٍ ١٧ ونشِطٍ بصبغةٍ هادئة،
 * وفي أسفلها **بطاقةُ حالة الأمين** وصاحبُ الحساب.
 * **وبنودُه بصلاحيات القائمة القديمة نفسِها** (انظر `navModel.ts`). وتحت `lg` لا يُرسم شيء.
 */

import Link from "next/link";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { BrandMark, phoneText, usePlatform } from "@rahalgo/ui";
import { useAuth } from "@/lib/auth";
import { useStore } from "@/lib/store/context";
import { AmeenSideCard } from "./AmeenStatus";
import { useStoreNav } from "./navModel";

const m = getMessages(defaultLocale);
const N = m.store.nav;
const SH = m.store.shell;

/** حرفان من اسم الحساب للصورة الرمزيّة. */
export function initials(name: string): string {
  const parts = name.trim().split(/\s+/).filter(Boolean);
  if (parts.length === 0) return SH.noName;
  const a = parts[0] ?? "";
  const b = parts[1] ?? "";
  return b ? a.slice(0, 1) + b.slice(0, 1) : a.slice(0, 1);
}

export default function StoreSidebar({ path, onMenu }: { path: string; onMenu: () => void }) {
  const { store } = useStore();
  const { user } = useAuth();
  const platform = usePlatform();
  const { groups, activeHref, role } = useStoreNav(path);
  const name = user?.full_name || store?.name || "";

  return (
    <aside className="console-side" data-testid="store-sidebar">
      <Link href="/store" aria-label={N.panel} className="flex h-13 shrink-0 items-center gap-rg-md px-rg-sm">
        <BrandMark size={26} rounded="control" />
        <b className="text-body-lg font-bold">{platform.name || m.common.appName}</b>
        <span className="ms-auto rounded-control border border-line-soft px-rg-sm text-meta text-ink-muted">{SH.brandTag}</span>
      </Link>

      <nav aria-label={N.sidebar} className="mt-rg-sm flex min-h-0 flex-1 flex-col gap-rg-xxs overflow-y-auto">
        {groups.map((g, gi) => (
          <div key={g.key} className="flex flex-col gap-rg-xxs">
            <p className="nav-group" style={gi === 0 ? { paddingTop: "0.25rem" } : undefined}>
              {g.title}
            </p>
            {g.items.map((it) => {
              const Ico = it.icon;
              return (
                <Link
                  key={it.href}
                  href={it.href}
                  aria-current={it.href === activeHref ? "page" : undefined}
                  className="nav-link"
                  data-testid={`nav-${it.href.replace(/^\/store\/?/, "") || "home"}`}
                >
                  <Ico size={17} className="shrink-0" />
                  <span className="min-w-0 flex-1 truncate">{it.label}</span>
                  {!!it.count && <span className="nav-count">{it.count > 99 ? "99+" : it.count}</span>}
                  {it.pip && <span className="dot" data-tone="bad" aria-label={SH.needsLook} />}
                </Link>
              );
            })}
          </div>
        ))}
      </nav>

      <div className="flex flex-col gap-rg-sm border-t border-line-soft pt-rg-md">
        <AmeenSideCard />
        <button
          type="button"
          onClick={onMenu}
          data-testid="store-menu-open"
          aria-label={N.more}
          className="flex items-center gap-rg-md rounded-control px-rg-sm py-rg-xs text-start transition-colors hover:bg-row-hover"
        >
          <span className="avatar">{initials(name)}</span>
          <span className="min-w-0 text-caption leading-tight">
            <span className="block truncate">{name}</span>
            {user?.phone && <span className="block truncate text-meta text-ink-muted">{phoneText(user.phone)}</span>}
            <span className="block text-meta text-ink-muted">{role === "staff" ? SH.roleStaff : SH.roleOwner}</span>
          </span>
        </button>
      </div>
    </aside>
  );
}
