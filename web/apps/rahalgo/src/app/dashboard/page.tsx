"use client";

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale, fmtNum, fmtMoney, errorText } from "@rahalgo/i18n";
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
  Alert,
  LoadingState,
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
  // بانتظار قرارك — انظر `admin_stats_handlers.go`
  compensations_pending: number;
  reports_waiting: number;
  emergencies_open: number;
  orders_unassigned: number;
  leads_new: number;
  menu_pending: number;
  drivers_over_cash: number;
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **بانتظار قرارك — ما لا يتحرّك حتّى تتحرّك يد** (قرارُ المالك ٢٠٢٦-١٠-٠٣)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **كلُّ بطاقةٍ تفتح صفحتَها مرشَّحةً على ما عدّته** — فالرقمُ بابٌ لا خبر.
 *
 * **وكلٌّ بقدرة صفحتها** — القدرةُ نفسُها التي تُظهر بابَها في القائمة
 * (`layout.tsx`): **بطاقةٌ تفتح صفحةً تردّ «لا صلاحية» وعدٌ يُعتذر عنه.**
 */
const AWAITING: {
  key: keyof Stats;
  label: string;
  href: string;
  cap: string;
  icon: typeof IconWallet;
}[] = [
  { key: "reports_waiting", label: D.reportsWaiting, href: "/dashboard/orders?awaiting=1",
    cap: "orders.read", icon: IconWarning },
  { key: "emergencies_open", label: D.emergenciesOpen, href: "/dashboard/emergencies",
    cap: "support.manage", icon: IconWarning },
  { key: "orders_unassigned", label: D.ordersUnassigned, href: "/dashboard/orders?status=dispatching",
    cap: "orders.read", icon: IconOrder },
  { key: "compensations_pending", label: D.compensationsPending, href: "/dashboard/compensations",
    cap: "finance.read", icon: IconWallet },
  { key: "payouts_pending", label: D.payoutsPending, href: "/dashboard/payouts?status=pending",
    cap: "finance.read", icon: IconWallet },
  { key: "drivers_over_cash", label: D.driversOverCash, href: "/dashboard/cash",
    cap: "finance.read", icon: IconBalance },
  { key: "tickets_open", label: D.ticketsOpen, href: "/dashboard/tickets",
    cap: "support.manage", icon: IconSupport },
  { key: "leads_new", label: D.leadsNew, href: "/dashboard/leads",
    cap: "merchants.verify", icon: IconStore },
  { key: "menu_pending", label: D.menuPending, href: "/dashboard/sections",
    cap: "content.manage", icon: IconOrder },
];

interface WhatsAppStatus {
  provider: string;
  connected?: boolean;
  logged_in?: boolean;
  paired_as?: string;
}

export default function DashboardPage() {
  const { user, can } = useAuth();
  const router = useRouter();
  const [stats, setStats] = useState<Stats | null>(null);
  const [wa, setWa] = useState<WhatsAppStatus | null>(null);
  // **وخطأُ الأرقام يُقال** (تدقيقُ اللوحة ٢٠٢٦-١٠-٠٣): كانت الصفحةُ تبقى ترحيباً بلا شيءٍ تحته.
  const [statsError, setStatsError] = useState("");

  // ══════════════════════════════════════════════════════════════════
  // **وحالُ واتساب إعدادٌ أمنيّ** — `settings.security.manage`
  // ══════════════════════════════════════════════════════════════════
  //
  // **وكان النداءُ يقع لكلّ من فتح الرئيسيّة** — **فالعمليّاتُ
  // والماليّةُ تُردّان ٤٠٣ في كلّ فتحة.** (قِيس ٢٠٢٦-٠٩-١٣.)
  //
  // **وردٌّ مُنِعَ في كلّ فتحةِ شاشةٍ يُغرِق السجلَّ** — **فيُفقَد فيه
  // المنعُ الذي يعني شيئاً.**
  const load = useCallback(() => {
    api<Stats>("/api/v1/admin/stats")
      .then((s) => {
        setStats(s);
        setStatsError("");
      })
      .catch((e) => {
        setStats(null);
        setStatsError(errorText(e));
      });
    if (!can("settings.security.manage")) {
      setWa(null);
      return;
    }
    api<WhatsAppStatus>("/api/v1/admin/whatsapp").then(setWa).catch(() => setWa(null));
  }, [can]);

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

      {!stats && statsError && (
        <div className="mt-6">
          <Alert>{statsError}</Alert>
        </div>
      )}
      {!stats && !statsError && (
        <div className="mt-6">
          <LoadingState variant="stats" />
        </div>
      )}
      {stats && (
        <div className="mt-6 space-y-6">
          {/* **بانتظار قرارك أوّلاً** — (قرارُ المالك ٢٠٢٦-١٠-٠٣): من فتح
              اللوحةَ يقرأ أعلاها، **وما ينتظر يداً أعجلُ من أيّ رقم.** */}
          {AWAITING.some((a) => can(a.cap)) && (
            <section>
              <h2 className="mb-2 font-bold">{D.awaiting}</h2>
              <StatGrid>
                {AWAITING.filter((a) => can(a.cap)).map((a) => {
                  const n = stats[a.key];
                  return (
                    <StatCard
                      key={a.key}
                      icon={a.icon}
                      label={a.label}
                      value={n > 0 ? fmtNum(n) : m.common.zero}
                      tone={n > 0 ? "danger" : "success"}
                      onClick={() => router.push(a.href)}
                    />
                  );
                })}
              </StatGrid>
            </section>
          )}

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
                value={fmtMoney(stats.sales_today)}
              />
              {/* صافي المنصة: لا محفظة لها تُجمَع منها — هي الدفتر لا طرفٌ فيه */}
              <StatCard
                icon={IconWallet}
                label={D.netToday}
                value={fmtMoney(stats.net_today)}
                sub={D.netHint}
                tone="accent"
              />
              <StatCard
                icon={IconBalance}
                label={D.cashHeld}
                value={fmtMoney(stats.cash_held_total)}
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

          {/* حالة واتساب — كرت مركزي لا مربّع مرتجل. **ولمن يملك قدرتَه وحدَه** (تدقيقُ اللوحة
              ٢٠٢٦-١٠-٠٣): كان يُرسم لكلّ أحدٍ «جارٍ التحميل» إلى الأبد بنقطةٍ حمراء. */}
          {can("settings.security.manage") && (
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
          )}
        </div>
      )}
    </div>
  );
}
