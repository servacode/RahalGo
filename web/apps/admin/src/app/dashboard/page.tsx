"use client";

import { useCallback, useEffect, useState } from "react";
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
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";

const m = getMessages(defaultLocale);

interface Stats {
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
  useLiveRefresh(["order", "account", "lead"], load);

  return (
    <div>
      <h1 className="text-2xl font-bold">
        {m.admin.dashboard.welcome}
        {user?.full_name ? `${m.common.listSeparator}${user.full_name}` : ""}
      </h1>

      {stats && (
        <div className="mt-6 space-y-4">
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
