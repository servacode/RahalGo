"use client";

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  useLiveRefresh,
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
const fmt = new Intl.NumberFormat("ar-SY");

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

function StatCard({
  icon: Icon,
  label,
  value,
  sub,
}: {
  icon: React.ComponentType<{ size?: number; className?: string }>;
  label: string;
  value: number;
  sub?: string;
}) {
  return (
    <div className="flex items-center gap-4 rounded-card border border-line bg-surface p-5">
      <div className="flex h-12 w-12 shrink-0 items-center justify-center rounded-card bg-primary-light">
        <Icon size={22} className="text-primary-dark" />
      </div>
      <div className="min-w-0">
        <p className="text-2xl font-bold">{fmt.format(value)}</p>
        <p className="truncate text-sm text-ink-muted">
          {label}
          {sub && <span className="text-xs"> · {sub}</span>}
        </p>
      </div>
    </div>
  );
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
        {user?.full_name ? `، ${user.full_name}` : ""} 👋
      </h1>

      {stats && (
        <div className="mt-6 grid gap-4 sm:grid-cols-2 lg:grid-cols-3 xl:grid-cols-4">
          <StatCard icon={IconUsers} label={m.admin.dashboard.stats.customers} value={stats.customers} />
          <StatCard icon={IconDriver} label={m.admin.dashboard.stats.drivers} value={stats.drivers} />
          <StatCard
            icon={IconStore}
            label={m.admin.dashboard.stats.merchantsActive}
            value={stats.merchants_active}
            sub={`${fmt.format(stats.merchants_total)} إجمالاً`}
          />
          <StatCard icon={IconUser} label={m.admin.dashboard.stats.salesReps} value={stats.sales_reps} />
          <StatCard icon={IconOrder} label={m.admin.dashboard.stats.menuItems} value={stats.menu_items} />
          <StatCard icon={IconZones} label={m.admin.dashboard.stats.zonesActive} value={stats.zones_active} />
          <StatCard icon={IconPromos} label={m.admin.dashboard.stats.promosActive} value={stats.promos_active} />

          {/* بطاقة حالة واتساب المختصرة */}
          <div className="flex items-center gap-4 rounded-card border border-line bg-surface p-5">
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
        </div>
      )}
    </div>
  );
}
