"use client";

/**
 * **نموذجُ التنقّل — مصدرٌ واحدٌ للسايدبار والفتات وشريط التبويب على الهاتف**
 *
 * (التصميمُ المعتمد ٢٠٢٦-١١-٠٨.) **بنودُه هي بنودُ القائمة الجانبيّة وبصلاحياتها** لا بابَ جديداً ولا اختلافَ في من يرى ماذا:
 * «الطلبات» حين يدير المتجرُ طلباتِه، و«الموظّفون» لا يراه الموظّف، و«مزامنة الأسعار» لمن يستعمل الأمين وحدَه.
 */

import type { ComponentType } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
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
import { ameenNeedsLook, useAmeenInfo } from "./AmeenStatus";

const m = getMessages(defaultLocale);
const N = m.store.nav;
const SH = m.store.shell;

type Icon = ComponentType<{ size?: number; className?: string; strokeWidth?: number }>;

export interface NavEntry {
  href: string;
  label: string;
  icon: Icon;
  count?: number;
  /** نقطةٌ حمراءُ بجانب البند (مزامنةُ الأمين متعطّلة). */
  pip?: boolean;
  /** مطابقةٌ حرفيّة — `/store` بدايةُ كلّ مسار. */
  exact?: boolean;
}
export interface NavGroup {
  key: string;
  title: string;
  items: NavEntry[];
}

/** أبوابٌ ليست في القائمة الجانبيّة لكنّ للفتات اسمَها. */
const EXTRA: { href: string; label: string }[] = [
  { href: "/store/settings", label: m.store.console.settingsNav },
  { href: "/store/notifications", label: SH.notifications },
  { href: "/store/order-phones", label: N.orderPhones },
  { href: "/store/alerts", label: N.alerts },
];

export const ADD_HREF = "/store/menu/new";

export function useStoreNav(path: string) {
  const { store, selfManage } = useStore();
  const { pending } = useStoreOrders();
  const role = useTeamRole();
  const ameen = useAmeenInfo();

  const groups: NavGroup[] = [
    {
      key: "ops",
      title: N.groupOps,
      items: [
        { href: "/store", label: N.store, icon: IconStore, exact: true },
        ...(selfManage ? [{ href: "/store/orders", label: N.orders, icon: IconOrder, count: pending.length }] : []),
        { href: "/store/menu", label: N.menu, icon: IconList },
        ...(store?.uses_ameen === true
          ? [{ href: "/store/price-sync", label: N.priceSync, icon: IconRotate, pip: ameenNeedsLook(ameen) }]
          : []),
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

  const all = [
    ...groups.flatMap((g) => g.items),
    { href: ADD_HREF, label: N.addItem, icon: IconStore } as NavEntry,
    ...EXTRA.map((e) => ({ ...e, icon: IconStore }) as NavEntry),
  ];

  /** **الأطولُ مطابقةً يفوز** — `/store/menu/new` تبدأ بـ`/store/menu`. */
  const hit = all
    .filter((i) => (i.exact ? path === i.href : path === i.href || path.startsWith(`${i.href}/`)))
    .sort((a, b) => b.href.length - a.href.length)[0];

  return { groups, activeHref: hit?.href ?? "", crumb: hit?.label ?? N.store, role };
}
