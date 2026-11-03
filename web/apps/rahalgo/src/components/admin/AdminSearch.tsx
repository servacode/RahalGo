"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **البحثُ السريع (Ctrl+K) — رقمُ طلب، تلفون، اسمُ متجرٍ أو شخص، أو صفحة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (إعادةُ تصميم اللوحة ٢٠٢٦-١١-٠٣.) الواجهةُ في العُدّة (`CommandPalette`)، وهنا **تجميعُ النتائج من
 * الواجهات الموجودة**، بلا نقطةٍ جديدةٍ في المحرّك:
 *
 *   طلبات   `GET /admin/orders?query=`        رقمٌ · هاتفُ الزبون · اسمُه          (orders.read)
 *   ناس     `GET /admin/users?query=`         اسمٌ · هاتف (لمن يملك contact.read)  (users.read)
 *   متاجر   `GET /admin/market/stores?q=`     اسمٌ · «M-1045»                      (market.manage)
 *   صفحات   قائمةُ اللوحة نفسُها بعد ترشيح الصلاحيات
 *
 * **كلُّ نداءٍ بقدرته** — من لا يملك القدرةَ لا يُنادى له (لا ضجيجَ ٤٠٣)، **والرابطُ لا يُرسم لمن لا يفتحه**
 * (`useCanOpen`). وفشلُ نداءٍ لا يُسقط البقيّة.
 */

import { useCallback, useState } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  CommandPalette,
  TopBarChip,
  useCtrlK,
  IconSearch,
  IconOrder,
  IconUsers,
  IconStore,
  type ChromeNavItem,
  type PaletteResult,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useCanOpen } from "@/lib/policy";
import { roleLabelByCode } from "@/lib/rolemeta";

const m = getMessages(defaultLocale);
const K = m.admin.kit;
const G = K.palette.groups;
const SEP = m.common.nameSeparator;

interface OrderHit {
  id: string;
  number: number;
  merchant_name?: string;
  customer_name?: string;
  customer_phone?: string;
  customer_phone_masked?: string;
  status?: string;
}
interface UserHit {
  id: string;
  full_name?: string;
  phone?: string;
  roles?: string[];
}
interface StoreHit {
  id: string;
  number?: number;
  name: string;
}

const STATUS = m.orders.status as Record<string, string>;
const join = (...parts: (string | undefined | false)[]) => parts.filter(Boolean).join(SEP);

export function AdminSearch({ nav }: { nav: ChromeNavItem[] }) {
  const router = useRouter();
  const { can } = useAuth();
  const canOpen = useCanOpen();
  const [open, setOpen] = useState(false);
  useCtrlK(() => setOpen(true));

  const search = useCallback(
    async (q: string): Promise<PaletteResult[]> => {
      const enc = encodeURIComponent(q);
      const needle = q.toLowerCase();

      const orders =
        can("orders.read") && canOpen.order
          ? api<{ orders?: OrderHit[] }>(`/api/v1/admin/orders?query=${enc}&per_page=5`)
              .then((r) =>
                (r.orders ?? []).map<PaletteResult>((o) => ({
                  key: `o-${o.id}`,
                  group: G.orders,
                  icon: IconOrder,
                  title: `#${o.number}${o.merchant_name ? `${SEP}${o.merchant_name}` : ""}`,
                  meta: join(o.customer_name, o.customer_phone ?? o.customer_phone_masked, o.status ? STATUS[o.status] ?? o.status : ""),
                  href: `/dashboard/history?order=${o.id}`,
                })),
              )
              // @empty-ok فشلُ نداءٍ لا يُسقط البقيّة — تُعرض نتائجُ غيره.
              .catch(() => [] as PaletteResult[])
          : Promise.resolve([] as PaletteResult[]);

      const people =
        can("users.read") && canOpen.user
          ? api<{ users?: UserHit[] }>(`/api/v1/admin/users?query=${enc}&per_page=5`)
              .then((r) =>
                (r.users ?? []).map<PaletteResult>((u) => ({
                  key: `u-${u.id}`,
                  group: G.people,
                  icon: IconUsers,
                  title: u.full_name || u.phone || "",
                  meta: join(u.phone && u.full_name ? u.phone : "", (u.roles ?? []).map((x) => roleLabelByCode(x)).join(K.listSep)),
                  href: `/dashboard/users/${u.id}`,
                })),
              )
              // @empty-ok كذلك.
              .catch(() => [] as PaletteResult[])
          : Promise.resolve([] as PaletteResult[]);

      const stores =
        can("market.manage") && canOpen.merchant
          ? api<{ stores?: StoreHit[] }>(`/api/v1/admin/market/stores?q=${enc}&per_page=5`)
              .then((r) =>
                (r.stores ?? []).map<PaletteResult>((s) => ({
                  key: `s-${s.id}`,
                  group: G.stores,
                  icon: IconStore,
                  title: s.name,
                  meta: s.number ? `M-${s.number}` : undefined,
                  href: `/dashboard/merchants/${s.id}`,
                })),
              )
              // @empty-ok كذلك.
              .catch(() => [] as PaletteResult[])
          : Promise.resolve([] as PaletteResult[]);

      // **الصفحاتُ من القائمة نفسِها** — فلا تظهر صفحةٌ لا يملكها صاحبُ الجلسة.
      const pages = nav
        .filter((i) => i.label.toLowerCase().includes(needle))
        .slice(0, 5)
        .map<PaletteResult>((i) => ({
          key: `p-${i.href}`,
          group: G.pages,
          icon: i.icon,
          title: i.label,
          href: i.href,
        }));

      const [o, p, s] = await Promise.all([orders, people, stores]);
      return [...o, ...p, ...s, ...pages];
    },
    [can, canOpen.order, canOpen.user, canOpen.merchant, nav],
  );

  return (
    <>
      <TopBarChip
        type="button"
        onClick={() => setOpen(true)}
        aria-label={K.palette.open}
        title={`${K.palette.open} (${K.palette.hintKeys})`}
        className="gap-2"
      >
        <IconSearch size={18} />
        <span className="hidden sm:inline">{K.search.button}</span>
        <kbd dir="ltr" className="hidden rounded-control border border-line-soft px-1.5 text-xs text-ink-muted lg:inline">
          {K.palette.hintKeys}
        </kbd>
      </TopBarChip>
      <CommandPalette open={open} onClose={() => setOpen(false)} search={search} onPick={(href) => router.push(href)} />
    </>
  );
}
