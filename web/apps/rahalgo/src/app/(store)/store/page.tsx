"use client";

/**
 * **‏/store — «متجري»**: ما جرى اليوم بأرقامٍ أربعة (مرآةُ بطاقة التقرير في
 * `StoreScreen.kt`)، **والاسمُ والحالُ ومفتاحُ الفتح في الشريط فوق.**
 *
 * **وما بقي من إعدادات المتجر** (الساعات · الأقسام · العنوان · سعر الصرف)
 * **في التطبيق وفي المرحلة الثانية** — هنا ما يحتاجه يومُ العمل.
 */

import Link from "next/link";
import { useCallback } from "react";
import { getMessages, defaultLocale, damascusDay, fmtMoney, fmtNum } from "@rahalgo/i18n";
import { Card, IconMoto, IconOrder, IconList, IconWallet, LoadingState, StatCard, StatGrid, useLiveData } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useStore } from "@/lib/store/context";
import type { StoreReport } from "@/lib/store/types";

const m = getMessages(defaultLocale);
const S = m.store;

export default function StoreHome() {
  const { store, selfManage } = useStore();
  const id = store?.id ?? "";
  const load = useCallback(() => {
    const today = damascusDay();
    return api<StoreReport>(`/api/v1/merchant/stores/${id}/reports?from=${today}&to=${today}`);
  }, [id]);
  const { data: report, loading } = useLiveData(load, ["order"], [id]);
  if (!store) return null;

  const links = [
    ...(selfManage ? [{ href: "/store/orders", label: S.nav.orders, icon: IconOrder }] : []),
    { href: "/store/menu", label: S.nav.menu, icon: IconList },
    { href: "/store/delivery", label: S.delivery.title, icon: IconMoto },
    { href: "/store/wallet", label: S.nav.wallet, icon: IconWallet },
  ];

  return (
    <div className="flex flex-col gap-3">
      {loading && !report ? (
        <LoadingState variant="stats" rows={4} />
      ) : (
        report && (
          <StatGrid>
            <StatCard label={S.today.orders} value={fmtNum(report.summary.orders)} />
            <StatCard label={S.today.delivered} value={fmtNum(report.summary.delivered)} tone="success" />
            <StatCard label={S.today.cancelled} value={fmtNum(report.summary.cancelled)} tone="muted" />
            <StatCard
              label={S.today.due}
              value={fmtMoney(report.summary.sales - report.summary.platform_commission)}
              tone="accent"
              emphasis
            />
          </StatGrid>
        )
      )}
      <Card padding="sm">
        <ul className="grid grid-cols-2 gap-2">
          {links.map((l) => {
            const Icon = l.icon;
            return (
              <li key={l.href}>
                <Link
                  href={l.href}
                  className="flex items-center gap-2 rounded-control border border-line bg-field px-3 py-3 text-sm font-bold hover:bg-row-hover"
                >
                  <Icon size={18} className="shrink-0 text-accent-text" />
                  {l.label}
                </Link>
              </li>
            );
          })}
        </ul>
      </Card>
    </div>
  );
}
