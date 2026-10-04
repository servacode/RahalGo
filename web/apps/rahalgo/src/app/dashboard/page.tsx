"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **رئيسيّةُ مدير المنصّة — كلُّ ما يجري في صفحةٍ واحدة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-١٠-٠٤: «لازم أفهم كلّ شي عم يصير بالمنصّة لأصغر تفصيل
 *  بدون ما أتنقّل بين الصفحات» · «الرئيسيّة لمدير المنصّة وحدَه».)
 *
 * **من أعلى إلى أسفل**: بانتظار قرارك · الآن · اليوم مقابل أمس · المال اليوم ·
 * آخر الأحداث · المنصّة بالأرقام. **وشريطُ الطوارئ فوقها في كلّ صفحة**
 * (`EmergencyBanner` في `layout.tsx`).
 *
 * # القواعد
 *
 *  - **كلُّ رقمٍ بابٌ** يفتح صفحةً فيها الرقمُ نفسُه — والمحرّكُ يعدّ بشرط
 *    صفحته (`admin_overview_handlers.go`)، **واختبارٌ يقارن الرقمين.**
 *  - **الرقمُ الذي لم يُقرأ «غير معروف»** لا صفرٌ أخضر.
 *  - **الصفرُ رماديٌّ هادئ**، والأحمرُ للمستعجل وحدَه.
 *  - **فشلُ التحديث لا يمحو الصفحة** — تبقى آخرُ أرقامٍ وصلت ومعها سطرٌ يقول ذلك.
 */

import { useCallback, useEffect, useRef, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import {
  getMessages,
  defaultLocale,
  fmtNum,
  fmtMoney,
  fmtSpan,
  fmtTime,
  errorText,
} from "@rahalgo/i18n";
import {
  useLiveRefresh,
  StatCard,
  StatGrid,
  Card,
  Alert,
  Button,
  LoadingState,
  IconWarning,
  IconOrder,
  IconDriver,
  IconStore,
  IconWallet,
  IconBalance,
  IconSupport,
  IconLink,
  IconZones,
  IconStatus,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { roleLabelByCode } from "@/lib/rolemeta";

const m = getMessages(defaultLocale);
const H = m.admin.home;
const ACTIONS = m.admin.audit.actions as Record<string, string>;
const STATUS = m.orders.status as Record<string, string>;
const FAIL = m.common.failReasons as Record<string, string>;

type N = number | null | undefined;

interface Day {
  orders: N;
  delivered: N;
  failed: N;
  avg_delivery_min: N;
  new_customers: N;
  avg_platform_rating: N;
  avg_driver_rating: N;
  ratings: N;
}

interface Count {
  key: string;
  total: number;
  today: number;
  week: number;
}

interface Overview {
  generated_at: string;
  today: string;
  missing: string[];
  awaiting: Record<
    | "reports_waiting"
    | "emergencies_open"
    | "orders_unassigned"
    | "compensations_pending"
    | "payouts_pending"
    | "drivers_over_cash"
    | "tickets_open"
    | "tickets_late"
    | "leads_new"
    | "expansion_waiting",
    N
  >;
  live: {
    stages: Record<string, N>;
    orders_open: N;
    orders_stuck: N;
    drivers_on_shift: N;
    drivers_free: N;
    drivers_busy: N;
    stores_open_now: N;
    stores_active: N;
  };
  health: { api: string; database: string; whatsapp: string; routing: string };
  today_stats: Day;
  yesterday_stats: Day;
  failures_today: { status: string; reason: string; count: number }[];
  money: {
    sales: N;
    net: N;
    losses: N;
    treasury_balance: N;
    cash_held: N;
    compensations_count: N;
    compensations_amount: N;
  };
  events: {
    id: number;
    action: string;
    entity: string;
    entity_id: string;
    label: string;
    number: number | null;
    actor_name: string;
    created_at: string;
  }[];
  platform: {
    groups: Count[];
    staff: Count[];
    customers_ordered_week: N;
    drivers_active: N;
    drivers_suspended: N;
    drivers_blocked: N;
    stores_active: N;
    stores_not_active: N;
  };
}

/** **رقمٌ أو «غير معروف»** — ولا صفرَ مكانَ ما لم يُقرأ. */
const known = (n: N): n is number => typeof n === "number";
const show = (n: N, f: (x: number) => string = fmtNum) => (known(n) ? f(n) : H.unknown);

/**
 * **نبرةُ البطاقة**: غائبٌ عاديّ · صفرٌ رماديّ · وفوق الصفر بنبرة أهمّيّته.
 * **والأحمرُ للمستعجل وحدَه** — ما ينتظره إنسانٌ في الشارع.
 */
const toneOf = (n: N, urgent: boolean): "default" | "muted" | "danger" | "warning" =>
  !known(n) ? "default" : n === 0 ? "muted" : urgent ? "danger" : "warning";

/** **رابطُ الحدث إلى كيانه.** */
function eventHref(e: Overview["events"][number]): string | null {
  switch (e.entity) {
    case "user":
      return e.entity_id ? `/dashboard/users/${e.entity_id}` : null;
    case "merchant":
      return e.entity_id ? `/dashboard/merchants/${e.entity_id}` : null;
    case "order":
      return e.number != null ? `/dashboard/history?q=${e.number}` : null;
    case "ticket":
      return e.entity_id ? `/dashboard/tickets?t=${e.entity_id}` : null;
    case "emergency":
      return "/dashboard/emergencies";
    case "payout":
      return "/dashboard/payouts";
    case "dispute":
      return "/dashboard/losses";
    case "promo":
      return "/dashboard/promos";
    case "setting":
    case "platform":
      return "/dashboard/settings";
    case "zone":
    case "delivery_zone":
      return "/dashboard/opsmap";
    default:
      return null;
  }
}

const AWAITING: {
  key: keyof Overview["awaiting"];
  label: string;
  href: string;
  urgent: boolean;
  icon: typeof IconWarning;
}[] = [
  { key: "emergencies_open", label: H.emergenciesOpen, href: "/dashboard/emergencies", urgent: true, icon: IconWarning },
  { key: "reports_waiting", label: H.reportsWaiting, href: "/dashboard/orders?awaiting=1", urgent: true, icon: IconWarning },
  { key: "orders_unassigned", label: H.ordersUnassigned, href: "/dashboard/orders?filter=no_driver", urgent: true, icon: IconOrder },
  { key: "tickets_late", label: "", href: "/dashboard/tickets?late=1", urgent: true, icon: IconSupport },
  { key: "drivers_over_cash", label: H.driversOverCash, href: "/dashboard/cash?over=1", urgent: true, icon: IconBalance },
  { key: "tickets_open", label: H.ticketsOpen, href: "/dashboard/tickets?status=unresolved", urgent: false, icon: IconSupport },
  { key: "compensations_pending", label: H.compensationsPending, href: "/dashboard/compensations", urgent: false, icon: IconWallet },
  { key: "payouts_pending", label: H.payoutsPending, href: "/dashboard/payouts?status=pending", urgent: false, icon: IconWallet },
  { key: "leads_new", label: H.leadsNew, href: "/dashboard/leads", urgent: false, icon: IconLink },
  { key: "expansion_waiting", label: H.expansionWaiting, href: "/dashboard/expansion?filter=waiting", urgent: false, icon: IconZones },
];

const STAGES = ["at_store_prep", "dispatching", "to_store", "at_store", "to_customer"] as const;

const GROUP_HREF: Record<string, string> = {
  customer: "/dashboard/users?role=customer",
  driver: "/dashboard/users?role=driver",
  sales: "/dashboard/users?role=sales",
  stores: "/dashboard/opsmap",
  items: "/dashboard/sections",
  zones: "/dashboard/opsmap",
  promos: "/dashboard/promos",
};

function Section({ title, note, children }: { title: string; note?: string; children: React.ReactNode }) {
  return (
    <section>
      <h2 className="mb-2 flex flex-wrap items-baseline gap-2 font-bold">
        {title}
        {note && <span className="text-xs font-normal text-ink-muted">{note}</span>}
      </h2>
      {children}
    </section>
  );
}

export default function DashboardPage() {
  const router = useRouter();
  const [data, setData] = useState<Overview | null>(null);
  const [error, setError] = useState("");
  const [loadedAt, setLoadedAt] = useState<number | null>(null);
  const [busy, setBusy] = useState(false);
  const [, tick] = useState(0);
  const lateHours = useRef(2);

  // ══════════════════════════════════════════════════════════════════
  // **والفشلُ لا يمحو ما وصل** — تبقى الأرقامُ ومعها سطرٌ يقول إنّها قديمة.
  // ══════════════════════════════════════════════════════════════════
  const load = useCallback(() => {
    setBusy(true);
    api<Overview>("/api/v1/admin/overview")
      .then((d) => {
        setData(d);
        setError("");
        setLoadedAt(Date.now());
      })
      .catch((e) => setError(errorText(e)))
      .finally(() => setBusy(false));
  }, []);

  useEffect(() => {
    load();
  }, [load]);

  // **ومهلةُ الشكوى المتأخّرة من الإعدادات** — تُقال في عنوان بطاقتها.
  useEffect(() => {
    api<{ settings: { key: string; value: unknown }[] }>("/api/v1/admin/settings")
      .then(({ settings }) => {
        const v = settings.find((x) => x.key === "support.late_reply_hours")?.value;
        if (typeof v === "number") lateHours.current = v;
        tick((x) => x + 1);
      })
      // @empty-ok — **العنوانُ يبقى بافتراضه** (ساعتان) والرقمُ من المحرّك.
      .catch(() => undefined);
  }, []);

  // **و«آخر تحديث قبل…» يمشي وحدَه** — كلَّ نصف دقيقة.
  useEffect(() => {
    const t = setInterval(() => tick((x) => x + 1), 30_000);
    return () => clearInterval(t);
  }, []);

  // **حيّةٌ على كلّ ما يغيّر أرقامَها** — والشكوى (`ticket`) كانت غائبة.
  useLiveRefresh(
    ["order", "orders", "account", "lead", "wallet", "alerts", "ticket", "emergency",
      "driver", "merchant", "dispute", "settings", "user"],
    load,
  );

  const go = (href: string) => () => router.push(href);

  if (!data) {
    return (
      <div>
        <h1 className="heading-page">{H.title}</h1>
        <div className="mt-6">
          {error ? (
            <Alert>
              {error}{" "}
              <Button variant="secondary" onClick={load} disabled={busy}>
                {H.refresh}
              </Button>
            </Alert>
          ) : (
            <LoadingState variant="stats" />
          )}
        </div>
      </div>
    );
  }

  const a = data.awaiting;
  const lv = data.live;
  const t = data.today_stats;
  const y = data.yesterday_stats;
  const ago =
    loadedAt === null
      ? ""
      : Date.now() - loadedAt < 60_000
        ? H.updatedNow
        : H.updatedAgo.replace("{t}", fmtSpan((Date.now() - loadedAt) / 1000));

  const dayRows: { label: string; key: keyof Day; f?: (x: number) => string }[] = [
    { label: H.rowOrders, key: "orders" },
    { label: H.rowDelivered, key: "delivered" },
    { label: H.rowFailed, key: "failed" },
    { label: H.rowAvgDelivery, key: "avg_delivery_min", f: (x) => H.minutes.replace("{n}", fmtNum(x)) },
    { label: H.rowNewCustomers, key: "new_customers" },
    { label: H.rowPlatformRating, key: "avg_platform_rating", f: (x) => H.stars.replace("{n}", fmtNum(x)) },
    { label: H.rowDriverRating, key: "avg_driver_rating", f: (x) => H.stars.replace("{n}", fmtNum(x)) },
  ];
  // **ومتوسّطٌ بلا مادّةٍ ليس مجهولاً** — لا تسليمَ ولا تقييمَ اليوم: شرطةٌ هادئة.
  const dayCell = (d: Day, r: (typeof dayRows)[number]) => {
    const v = d[r.key];
    if (v === null && (r.key.startsWith("avg_")) && known(d.orders)) return "—";
    return show(v, r.f);
  };

  return (
    <div className="space-y-6">
      <div className="flex flex-wrap items-center gap-3">
        <h1 className="heading-page flex-1">{H.title}</h1>
        <span className="text-xs text-ink-muted">{ago}</span>
        <Button variant="secondary" onClick={load} disabled={busy}>
          {H.refresh}
        </Button>
      </div>
      {error && <Alert tone="warning">{H.refreshFailed}</Alert>}
      {data.missing.length > 0 && <Alert tone="warning">{H.missing}</Alert>}

      {/* ── ١ · بانتظار قرارك ── */}
      <Section title={H.awaiting} note={H.awaitingHint}>
        <StatGrid>
          {AWAITING.map((c) => {
            const n = a[c.key];
            const tone = toneOf(n, c.urgent);
            return (
              <StatCard
                key={c.key}
                icon={c.icon}
                label={c.key === "tickets_late" ? H.ticketsLate.replace("{h}", fmtNum(lateHours.current)) : c.label}
                value={show(n)}
                tone={tone}
                emphasis={tone === "danger" || tone === "warning"}
                onClick={go(c.href)}
              />
            );
          })}
        </StatGrid>
      </Section>

      {/* ── ٢ · الآن ── */}
      <Section title={H.now}>
        <StatGrid>
          {STAGES.map((s) => (
            <StatCard
              key={s}
              icon={IconOrder}
              label={H.stage[s]}
              value={show(lv.stages[s])}
              tone={known(lv.stages[s]) && lv.stages[s] === 0 ? "muted" : "default"}
              onClick={go(`/dashboard/orders?stage=${s}`)}
            />
          ))}
          <StatCard
            icon={IconWarning}
            label={H.stuck}
            value={show(lv.orders_stuck)}
            sub={known(lv.orders_open) ? `${H.liveOrders}${m.common.nameSeparator}${fmtNum(lv.orders_open)}` : undefined}
            tone={toneOf(lv.orders_stuck, true)}
            emphasis={known(lv.orders_stuck) && lv.orders_stuck > 0}
            onClick={go("/dashboard/orders")}
          />
          <StatCard
            icon={IconDriver}
            label={H.driversOnShift}
            value={show(lv.drivers_on_shift)}
            sub={H.driversNote}
            tone={known(lv.drivers_on_shift) && lv.drivers_on_shift === 0 ? "muted" : "default"}
            onClick={go("/dashboard/opsmap?on_shift=true&status=active")}
          />
          <StatCard
            icon={IconDriver}
            label={H.driversFree}
            value={show(lv.drivers_free)}
            tone={known(lv.drivers_free) && lv.drivers_free === 0 ? "muted" : "default"}
            onClick={go("/dashboard/opsmap?on_shift=true&status=active&has_active=false")}
          />
          <StatCard
            icon={IconDriver}
            label={H.driversBusy}
            value={show(lv.drivers_busy)}
            tone={known(lv.drivers_busy) && lv.drivers_busy === 0 ? "muted" : "default"}
            onClick={go("/dashboard/opsmap?on_shift=true&status=active&has_active=true")}
          />
          <StatCard
            icon={IconStore}
            label={H.storesOpen}
            value={show(lv.stores_open_now)}
            sub={known(lv.stores_active) ? H.storesOf.replace("{n}", fmtNum(lv.stores_active)) : undefined}
            tone={known(lv.stores_open_now) && lv.stores_open_now === 0 ? "muted" : "default"}
            onClick={go("/dashboard/opsmap")}
          />
        </StatGrid>
        <Card className="mt-3">
          <p className="mb-2 text-sm font-bold">{H.health}</p>
          <div className="grid grid-cols-2 gap-2 text-sm sm:grid-cols-4">
            {(
              [
                [H.healthApi, data.health.api],
                [H.healthDb, data.health.database],
                [H.healthWa, data.health.whatsapp],
                [H.healthRouting, data.health.routing],
              ] as const
            ).map(([label, st]) => (
              <Link key={label} href="/dashboard/ops" className="flex items-center gap-2 rounded-control p-1 hover:bg-row-hover">
                <span
                  className={`h-2.5 w-2.5 shrink-0 rounded-badge ${
                    st === "ok" || st === "dev" ? "bg-success" : st === "down" ? "bg-danger" : "bg-warning"
                  }`}
                />
                <span className="min-w-0">
                  <span className="block text-xs text-ink-muted">{label}</span>
                  {(H.healthState as Record<string, string>)[st] ?? H.unknown}
                </span>
              </Link>
            ))}
          </div>
        </Card>
      </Section>

      {/* ── ٣ · اليوم مقابل أمس ── */}
      <Section title={H.today} note={H.todayNote}>
        <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
          <Card>
            <div className="space-y-2 text-sm">
              <div className="flex gap-2 text-xs text-ink-muted">
                <span className="flex-1" />
                <span className="w-24 shrink-0">{H.colToday}</span>
                <span className="w-24 shrink-0">{H.colYesterday}</span>
              </div>
              {dayRows.map((r) => (
                <div key={r.key} className="flex items-baseline gap-2">
                  <span className="min-w-0 flex-1 text-ink-muted">{r.label}</span>
                  <span className="w-24 shrink-0 font-bold">{dayCell(t, r)}</span>
                  <span className="w-24 shrink-0">{dayCell(y, r)}</span>
                </div>
              ))}
            </div>
          </Card>
          <Card>
            <p className="mb-2 text-sm font-bold">{H.failuresTitle}</p>
            {data.missing.includes("failures") ? (
              <p className="text-sm text-ink-muted">{H.unknown}</p>
            ) : data.failures_today.length === 0 ? (
              <p className="text-sm text-ink-muted">{H.noFailures}</p>
            ) : (
              <ul className="space-y-1 text-sm">
                {data.failures_today.map((f) => (
                  <li key={f.status + f.reason} className="flex gap-2">
                    <span className="min-w-0 flex-1">
                      {STATUS[f.status] ?? f.status}
                      {m.common.nameSeparator}
                      {f.reason ? (FAIL[f.reason] ?? f.reason) : H.noReason}
                    </span>
                    <span className="font-bold">{fmtNum(f.count)}</span>
                  </li>
                ))}
              </ul>
            )}
          </Card>
        </div>
      </Section>

      {/* ── ٤ · المال اليوم ── */}
      <Section title={H.money} note={H.todayNote}>
        <StatGrid>
          <StatCard icon={IconBalance} label={H.sales} value={show(data.money.sales, fmtMoney)} onClick={go("/dashboard/profits")} />
          <StatCard
            icon={IconWallet}
            label={H.net}
            value={show(data.money.net, fmtMoney)}
            sub={H.netHint}
            tone="accent"
            onClick={go("/dashboard/profits")}
          />
          <StatCard icon={IconWallet} label={H.treasury} value={show(data.money.treasury_balance, fmtMoney)} onClick={go("/dashboard/wallet")} />
          <StatCard icon={IconBalance} label={H.cashHeld} value={show(data.money.cash_held, fmtMoney)} onClick={go("/dashboard/cash")} />
          <StatCard
            icon={IconWallet}
            label={H.compensations}
            value={show(data.money.compensations_amount, fmtMoney)}
            sub={known(data.money.compensations_count) ? H.compensationsSub.replace("{n}", fmtNum(data.money.compensations_count)) : undefined}
            tone={known(data.money.compensations_amount) && data.money.compensations_amount === 0 ? "muted" : "default"}
            onClick={go("/dashboard/losses")}
          />
          <StatCard
            icon={IconBalance}
            label={H.losses}
            value={show(data.money.losses, fmtMoney)}
            tone={known(data.money.losses) && data.money.losses === 0 ? "muted" : "default"}
            onClick={go("/dashboard/losses")}
          />
        </StatGrid>
      </Section>

      {/* ── ٥ · آخر الأحداث المهمّة ── */}
      <Section title={H.events}>
        <Card>
          {data.missing.includes("events") ? (
            <p className="text-sm text-ink-muted">{H.unknown}</p>
          ) : data.events.length === 0 ? (
            <p className="text-sm text-ink-muted">{H.noEvents}</p>
          ) : (
            <ul className="divide-y divide-line-soft text-sm">
              {data.events.map((e) => {
                const href = eventHref(e);
                const what = [
                  ACTIONS[e.action] ?? e.action,
                  e.label || (e.number != null ? fmtNum(e.number) : ""),
                ]
                  .filter(Boolean)
                  .join(m.common.nameSeparator);
                return (
                  <li key={e.id} className="flex flex-wrap items-baseline gap-2 py-2">
                    <IconStatus size={14} className="shrink-0 text-ink-muted" />
                    {href ? (
                      <Link href={href} className="min-w-0 flex-1 hover:underline">
                        {what}
                      </Link>
                    ) : (
                      <span className="min-w-0 flex-1">{what}</span>
                    )}
                    {e.actor_name && (
                      <span className="text-xs text-ink-muted">{H.by.replace("{name}", e.actor_name)}</span>
                    )}
                    <span className="text-xs text-ink-muted">{fmtTime(e.created_at)}</span>
                  </li>
                );
              })}
            </ul>
          )}
        </Card>
      </Section>

      {/* ── ٦ · المنصّة بالأرقام ── */}
      <Section title={H.platform}>
        <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
          <Card>
            {data.missing.includes("platform") ? (
              <p className="text-sm text-ink-muted">{H.unknown}</p>
            ) : (
              <div className="space-y-2 text-sm">
                <div className="flex gap-2 text-xs text-ink-muted">
                  <span className="flex-1" />
                  <span className="w-16 shrink-0">{H.colTotal}</span>
                  <span className="w-16 shrink-0">{H.colNewToday}</span>
                  <span className="w-16 shrink-0">{H.colNewWeek}</span>
                </div>
                {data.platform.groups.map((g) => (
                  <div key={g.key} className="flex items-baseline gap-2">
                    <Link href={GROUP_HREF[g.key] ?? "/dashboard"} className="min-w-0 flex-1 text-ink hover:underline">
                      {(H.group as Record<string, string>)[g.key] ?? g.key}
                    </Link>
                    <span className="w-16 shrink-0 font-bold">{fmtNum(g.total)}</span>
                    <span className={`w-16 shrink-0 ${g.today ? "" : "text-ink-muted"}`}>{fmtNum(g.today)}</span>
                    <span className={`w-16 shrink-0 ${g.week ? "" : "text-ink-muted"}`}>{fmtNum(g.week)}</span>
                  </div>
                ))}
              </div>
            )}
          </Card>
          <Card>
            <p className="mb-2 text-sm font-bold">{H.staffTitle}</p>
            {data.missing.includes("platform") ? (
              <p className="text-sm text-ink-muted">{H.unknown}</p>
            ) : (
              <div className="space-y-2 text-sm">
                <div className="flex gap-2 text-xs text-ink-muted">
                  <span className="flex-1" />
                  <span className="w-16 shrink-0">{H.colTotal}</span>
                  <span className="w-16 shrink-0">{H.colNewToday}</span>
                  <span className="w-16 shrink-0">{H.colNewWeek}</span>
                </div>
                {data.platform.staff.map((g) => (
                  <div key={g.key} className="flex items-baseline gap-2">
                    <Link href={`/dashboard/users?role=${g.key}`} className="min-w-0 flex-1 text-ink hover:underline">
                      {roleLabelByCode(g.key)}
                    </Link>
                    <span className="w-16 shrink-0 font-bold">{fmtNum(g.total)}</span>
                    <span className={`w-16 shrink-0 ${g.today ? "" : "text-ink-muted"}`}>{fmtNum(g.today)}</span>
                    <span className={`w-16 shrink-0 ${g.week ? "" : "text-ink-muted"}`}>{fmtNum(g.week)}</span>
                  </div>
                ))}
              </div>
            )}
          </Card>
        </div>
        <div className="mt-3">
          <StatGrid>
            <StatCard icon={IconOrder} label={H.customersOrderedWeek} value={show(data.platform.customers_ordered_week)} onClick={go("/dashboard/history")} />
            <StatCard icon={IconDriver} label={H.driversActive} value={show(data.platform.drivers_active)} onClick={go("/dashboard/users?role=driver&status=active")} />
            <StatCard
              icon={IconDriver}
              label={H.driversSuspended}
              value={show(data.platform.drivers_suspended)}
              tone={known(data.platform.drivers_suspended) && data.platform.drivers_suspended === 0 ? "muted" : "default"}
              onClick={go("/dashboard/users?role=driver&status=suspended")}
            />
            <StatCard
              icon={IconDriver}
              label={H.driversBlocked}
              value={show(data.platform.drivers_blocked)}
              tone={known(data.platform.drivers_blocked) && data.platform.drivers_blocked === 0 ? "muted" : "default"}
              onClick={go("/dashboard/users?role=driver&status=blocked")}
            />
            <StatCard icon={IconStore} label={H.storesActive} value={show(data.platform.stores_active)} onClick={go("/dashboard/opsmap")} />
            <StatCard
              icon={IconStore}
              label={H.storesNotActive}
              value={show(data.platform.stores_not_active)}
              tone={known(data.platform.stores_not_active) && data.platform.stores_not_active === 0 ? "muted" : "default"}
              onClick={go("/dashboard/opsmap")}
            />
          </StatGrid>
        </div>
      </Section>
    </div>
  );
}
