"use client";

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";
import {
  useLiveRefresh,
  StatCard,
  StatGrid,
  Card,
  IconUsers,
  IconDriver,
  IconUser,
  IconStore,
  IconOrder,
  IconZones,
  IconPromos,
  IconWarning,
  IconCheck,
  IconWallet,
  IconBalance,
  IconSupport,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";

const m = getMessages(defaultLocale);
const D = m.admin.dash;

interface Stats {
  // الآن
  orders_open: number;
  orders_stuck: number;
  drivers_on_shift: number;
  drivers_busy: number;
  merchants_open: number;
  payouts_pending: number;
  tickets_open: number;
  // اليوم
  orders_today: number;
  delivered_today: number;
  cancelled_today: number;
  sales_today: number;
  net_today: number;
  cash_held_total: number;
  // المخزون
  customers: number;
  drivers: number;
  sales_reps: number;
  merchants_active: number;
  merchants_total: number;
  menu_items: number;
  zones_active: number;
  promos_active: number;
}

interface WhatsAppStatus {
  provider: string;
  connected?: boolean;
  logged_in?: boolean;
  paired_as?: string;
}

export default function DashboardPage() {
  const { user } = useAuth();
  const [stats, setStats] = useState<Stats | null>(null);
  const [wa, setWa] = useState<WhatsAppStatus | null>(null);

  const load = useCallback(() => {
    api<Stats>("/api/v1/admin/stats").then(setStats).catch(() => setStats(null));
    api<WhatsAppStatus>("/api/v1/admin/whatsapp").then(setWa).catch(() => setWa(null));
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  // النظرة العامة حيّة — أي طلب أو حساب أو طلب انضمام جديد يحدّث الأرقام فوراً
  useLiveRefresh(["order", "account", "lead", "wallet", "alerts"], load);

  return (
    <div>
      <h1 className="heading-page">
        {m.admin.dashboard.welcome}
        {user?.full_name ? `${m.common.listSeparator}${user.full_name}` : ""}
      </h1>

      {stats && (
        <div className="mt-6 space-y-6">
          {/*
            ثلاث طبقات بهذا الترتيب: **الآن** ثم **اليوم** ثم المخزون.
            كانت ثمانية أرقام كلُّها `count(*)` — أرقامُ مخزون تقول ماذا يملك
            النظام لا ماذا يجري فيه. فمن يفتح اللوحة صباحاً لا يعرف منها كيف
            كان أمس. والترتيب مقصود: من يفتح لوحةً يقرأ أعلاها.
          */}
          <section>
            <h2 className="mb-2 font-bold">{D.now}</h2>
            <StatGrid>
              <StatCard
                icon={IconOrder}
                label={D.ordersOpen}
                value={fmtNum(stats.orders_open)}
              />
              <StatCard
                icon={stats.orders_stuck > 0 ? IconWarning : IconCheck}
                label={D.ordersStuck}
                value={stats.orders_stuck > 0 ? fmtNum(stats.orders_stuck) : m.common.zero}
                sub={stats.orders_stuck === 0 ? D.nothingStuck : undefined}
                tone={stats.orders_stuck > 0 ? "danger" : "success"}
              />
              <StatCard
                icon={IconDriver}
                label={D.driversOnShift}
                value={fmtNum(stats.drivers_on_shift)}
                sub={D.driversBusy + m.common.nameSeparator + fmtNum(stats.drivers_busy)}
                tone={stats.drivers_on_shift > 0 ? "success" : "default"}
              />
              <StatCard
                icon={IconStore}
                label={D.merchantsOpen}
                value={fmtNum(stats.merchants_open)}
              />
              <StatCard
                icon={IconWallet}
                label={D.payoutsPending}
                value={fmtNum(stats.payouts_pending)}
                tone={stats.payouts_pending > 0 ? "accent" : "default"}
              />
              <StatCard
                icon={IconSupport}
                label={D.ticketsOpen}
                value={fmtNum(stats.tickets_open)}
                tone={stats.tickets_open > 0 ? "accent" : "default"}
              />
            </StatGrid>
          </section>

          <section>
            <h2 className="mb-2 font-bold">{D.today}</h2>
            <StatGrid>
              <StatCard icon={IconOrder} label={D.ordersToday} value={fmtNum(stats.orders_today)} />
              <StatCard
                icon={IconCheck}
                label={D.deliveredToday}
                value={fmtNum(stats.delivered_today)}
                tone="success"
              />
              <StatCard
                icon={IconWarning}
                label={D.cancelledToday}
                value={fmtNum(stats.cancelled_today)}
                tone={stats.cancelled_today > 0 ? "danger" : "default"}
              />
              <StatCard
                icon={IconBalance}
                label={D.salesToday}
                value={`${fmtNum(stats.sales_today)} ${m.common.currency}`}
              />
              {/* صافي المنصة: لا محفظة لها تُجمَع منها — هي الدفتر لا طرفٌ فيه */}
              <StatCard
                icon={IconWallet}
                label={D.netToday}
                value={`${fmtNum(stats.net_today)} ${m.common.currency}`}
                sub={D.netHint}
                tone="accent"
              />
              <StatCard
                icon={IconBalance}
                label={D.cashHeld}
                value={`${fmtNum(stats.cash_held_total)} ${m.common.currency}`}
                sub={D.cashHeldHint}
              />
            </StatGrid>
          </section>

          <section>
            <h2 className="mb-2 font-bold">{D.stock}</h2>
            <StatGrid>
              <StatCard icon={IconUsers} label={m.admin.dashboard.stats.customers} value={stats.customers} />
              <StatCard icon={IconDriver} label={m.admin.dashboard.stats.drivers} value={stats.drivers} />
              <StatCard
                icon={IconStore}
                label={m.admin.dashboard.stats.merchantsActive}
                value={stats.merchants_active}
                sub={m.admin.dashboard.totalSuffix.replace("{n}", fmtNum(stats.merchants_total))}
              />
              <StatCard icon={IconUser} label={m.admin.dashboard.stats.salesReps} value={stats.sales_reps} />
              <StatCard icon={IconOrder} label={m.admin.dashboard.stats.menuItems} value={stats.menu_items} />
              <StatCard icon={IconZones} label={m.admin.dashboard.stats.zonesActive} value={stats.zones_active} />
              <StatCard icon={IconPromos} label={m.admin.dashboard.stats.promosActive} value={stats.promos_active} />
            </StatGrid>
          </section>

          {/* حالة واتساب — كرت مركزي لا مربّع مرتجل */}
          <Card>
            <div className="flex items-center gap-3">
              <span
                className={`h-3 w-3 shrink-0 rounded-badge ${
                  wa?.provider === "dev" || wa?.logged_in
                    ? "bg-success"
                    : wa?.connected
                      ? "bg-warning"
                      : "bg-danger"
                }`}
              />
              <div className="min-w-0 text-sm">
                <p className="font-bold">{m.admin.dashboard.whatsappStatus}</p>
                <p className="truncate text-ink-muted">
                  {wa === null
                    ? m.common.loading
                    : wa.provider === "dev"
                      ? m.admin.dashboard.waDevProvider
                      : wa.logged_in
                        ? `${m.admin.dashboard.waPaired} +${wa.paired_as}`
                        : wa.connected
                          ? m.admin.dashboard.waNotPaired
                          : m.admin.dashboard.waDisconnected}
                </p>
              </div>
            </div>
          </Card>
        </div>
      )}
    </div>
  );
}
