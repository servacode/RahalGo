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
  IconGrid,
  IconSuccess,
  IconPrev,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { roleLabelByCode } from "@/lib/rolemeta";
import ContactTestAlert from "@/components/admin/settings/contact-check";
import SiteStatsSection from "@/components/admin/SiteStatsSection";
import PlatformSwitch from "@/components/admin/settings/PlatformSwitch";
import { PageHero } from "@/components/admin/PageHero";
import { Panel, Tile, type TileTone } from "@/components/admin/DashParts";

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
    | "drivers_cash_overdue"
    | "tickets_open"
    | "tickets_late"
    | "leads_new"
    | "expansion_waiting"
    | "obligations_overdue"
    | "obligation_requests",
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
  { key: "drivers_over_cash", label: H.driversOverCash, href: "/dashboard/cash?filter=over", urgent: true, icon: IconBalance },
  // **نقدٌ لم يُسلَّم منذ أيّام** — بعد عدد الأيّام في الإعدادات (قرارُ المالك ٢٠٢٦-١٠-٠٤).
  { key: "drivers_cash_overdue", label: H.driversCashOverdue, href: "/dashboard/cash?filter=overdue", urgent: true, icon: IconBalance },
  { key: "tickets_open", label: H.ticketsOpen, href: "/dashboard/tickets?status=unresolved", urgent: false, icon: IconSupport },
  { key: "compensations_pending", label: H.compensationsPending, href: "/dashboard/compensations", urgent: false, icon: IconWallet },
  { key: "payouts_pending", label: H.payoutsPending, href: "/dashboard/payouts?status=pending", urgent: false, icon: IconWallet },
  { key: "leads_new", label: H.leadsNew, href: "/dashboard/leads", urgent: false, icon: IconLink },
  { key: "expansion_waiting", label: H.expansionWaiting, href: "/dashboard/expansion?filter=waiting", urgent: false, icon: IconZones },
  { key: "obligations_overdue", label: H.obligationsOverdue, href: "/dashboard/obligations?overdue=1", urgent: false, icon: IconBalance },
  { key: "obligation_requests", label: H.obligationRequests, href: "/dashboard/obligations?pending=1", urgent: false, icon: IconBalance },
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
        <div className="mb-4">
          <PlatformSwitch />
        </div>
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
      {/* **رأسٌ موحّد** (قرارُ المالك ٢٠٢٦-١٠-٢٦) — والدوامُ أوّلُ ما يُرى: مفتوح · حسب الجدول · مسكّر
          بضغطة (طلبُ المالك ٢٠٢٦-١٠-١٨)، والتحديثُ بجانبه. */}
      <PageHero
        icon={IconGrid}
        title={H.title}
        subtitle={ago}
        actions={
          <>
            <PlatformSwitch compact />
            <Button variant="secondary" onClick={load} disabled={busy}>
              {H.refresh}
            </Button>
          </>
        }
      >
        {/* **صحّةُ النظام سطرٌ صغيرٌ في الرأس** — نقطةٌ بلونها لكلّ خدمة، والضغطُ يفتح المراقبة. */}
        <div className="flex flex-wrap items-center gap-2 text-xs">
          <span className="font-bold text-ink-muted">{H.health}:</span>
          {(
            [
              [H.healthApi, data.health.api],
              [H.healthDb, data.health.database],
              [H.healthWa, data.health.whatsapp],
              [H.healthRouting, data.health.routing],
            ] as const
          ).map(([label, st]) => (
            <Link
              key={label}
              href="/dashboard/ops"
              className="inline-flex items-center gap-1.5 rounded-full border border-line-soft px-2.5 py-1 hover:bg-row-hover"
            >
              <span
                className={`h-2 w-2 shrink-0 rounded-full ${
                  st === "ok" || st === "dev" ? "bg-success" : st === "down" ? "bg-danger" : "bg-warning"
                }`}
              />
              {label}
              <span className="text-ink-muted">{(H.healthState as Record<string, string>)[st] ?? H.unknown}</span>
            </Link>
          ))}
        </div>
      </PageHero>
      {error && <Alert tone="warning">{H.refreshFailed}</Alert>}
      {data.missing.length > 0 && <Alert tone="warning">{H.missing}</Alert>}
      {/* **بياناتُ التواصل التجريبيّة** — قرارُ المالك ٢٠٢٦-١٠-٠٤ (الإعدادات ١٥). */}
      <ContactTestAlert />

      {/* ══ ١ · بانتظار قرارك ══ — المستعجلُ أوّلاً، **وما فيه رقمٌ يُلوَّن والصفرُ يهدأ** (قرارُ المالك
          ٢٠٢٦-١٠-٢٦: «تصميم احترافي»). */}
      <Panel title={H.awaiting} note={H.awaitingHint}>
        {AWAITING.every((c) => known(a[c.key]) && a[c.key] === 0) && (
          <p className="mb-3 flex items-center gap-1.5 text-sm font-bold text-success">
            <IconSuccess size={16} />
            {H.allClear}
          </p>
        )}
        <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
          {[...AWAITING]
            .sort((x, y) => Number((a[y.key] ?? 0) > 0) - Number((a[x.key] ?? 0) > 0))
            .map((c) => (
              <Tile
                key={c.key}
                icon={c.icon}
                label={c.key === "tickets_late" ? H.ticketsLate.replace("{h}", fmtNum(lateHours.current)) : c.label}
                value={show(a[c.key])}
                tone={toneOf(a[c.key], c.urgent)}
                onClick={go(c.href)}
              />
            ))}
        </div>
      </Panel>

      {/* ══ ٢ · الآن ══ — مسارُ الطلب مراحلَ متتابعة، والسائقون والمتاجر بجانبه. */}
      <Panel title={H.now}>
        <div className="grid grid-cols-1 gap-3 xl:grid-cols-3">
          <div className="xl:col-span-2">
            <div className="flex flex-wrap items-stretch gap-1.5">
              {STAGES.map((s, i) => {
                const n = lv.stages[s];
                const on = known(n) && n > 0;
                return (
                  <div key={s} className="flex min-w-[8rem] flex-1 items-center gap-1.5">
                    <button
                      type="button"
                      onClick={go(`/dashboard/orders?stage=${s}`)}
                      className={`flex flex-1 flex-col items-center rounded-card border p-3 text-center transition-colors hover:bg-row-hover ${
                        on ? "border-accent" : "border-line-soft"
                      }`}
                    >
                      <span className={`figure leading-none ${on ? "" : "text-ink-muted"}`}>{show(n)}</span>
                      <span className="mt-1 text-xs text-ink-muted">{H.stage[s]}</span>
                    </button>
                    {i < STAGES.length - 1 && <IconPrev size={14} className="shrink-0 text-ink-muted" />}
                  </div>
                );
              })}
            </div>
            <button
              type="button"
              onClick={go("/dashboard/orders")}
              className={`mt-2 flex w-full items-center justify-between gap-2 rounded-card border p-3 text-sm transition-colors hover:bg-row-hover ${
                known(lv.orders_stuck) && lv.orders_stuck > 0 ? "border-danger bg-danger-tint" : "border-line-soft"
              }`}
            >
              <span className="flex items-center gap-2">
                <IconWarning size={16} className={known(lv.orders_stuck) && lv.orders_stuck > 0 ? "text-danger" : "text-ink-muted"} />
                <b>{H.stuck}</b>
                <span className="figure leading-none">{show(lv.orders_stuck)}</span>
              </span>
              <span className="text-xs text-ink-muted">
                {H.liveOrders}
                {m.common.nameSeparator}
                {show(lv.orders_open)}
              </span>
            </button>
          </div>
          <div className="grid grid-cols-2 gap-2">
            <Tile icon={IconDriver} label={H.driversOnShift} value={show(lv.drivers_on_shift)} tone={zeroMuted(lv.drivers_on_shift)} onClick={go("/dashboard/opsmap?on_shift=true&status=active")} />
            <Tile icon={IconDriver} label={H.driversFree} value={show(lv.drivers_free)} tone={zeroMuted(lv.drivers_free)} onClick={go("/dashboard/opsmap?on_shift=true&status=active&has_active=false")} />
            <Tile icon={IconDriver} label={H.driversBusy} value={show(lv.drivers_busy)} tone={zeroMuted(lv.drivers_busy)} onClick={go("/dashboard/opsmap?on_shift=true&status=active&has_active=true")} />
            <Tile
              icon={IconStore}
              label={H.storesOpen}
              value={show(lv.stores_open_now)}
              sub={known(lv.stores_active) ? H.storesOf.replace("{n}", fmtNum(lv.stores_active)) : undefined}
              tone={zeroMuted(lv.stores_open_now)}
              onClick={go("/dashboard/opsmap")}
            />
          </div>
        </div>
      </Panel>

      {/* ══ ٣ · المال اليوم ══ — الصافي أوّلاً وبلونه. */}
      <Panel title={H.money} note={H.todayNote}>
        <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 xl:grid-cols-6">
          <Tile icon={IconWallet} label={H.net} value={show(data.money.net, fmtMoney)} sub={H.netHint} tone="accent" onClick={go("/dashboard/profits")} />
          <Tile icon={IconBalance} label={H.sales} value={show(data.money.sales, fmtMoney)} onClick={go("/dashboard/profits")} />
          <Tile icon={IconWallet} label={H.treasury} value={show(data.money.treasury_balance, fmtMoney)} onClick={go("/dashboard/treasury?tab=statement")} />
          <Tile icon={IconBalance} label={H.cashHeld} value={show(data.money.cash_held, fmtMoney)} onClick={go("/dashboard/cash")} />
          <Tile
            icon={IconWallet}
            label={H.compensations}
            value={show(data.money.compensations_amount, fmtMoney)}
            sub={known(data.money.compensations_count) ? H.compensationsSub.replace("{n}", fmtNum(data.money.compensations_count)) : undefined}
            tone={zeroMuted(data.money.compensations_amount)}
            onClick={go("/dashboard/compensations")}
          />
          <Tile icon={IconBalance} label={H.losses} value={show(data.money.losses, fmtMoney)} tone={zeroMuted(data.money.losses)} onClick={go("/dashboard/profits")} />
        </div>
      </Panel>

      {/* ══ ٤ · اليوم مقابل أمس ══ — والسهمُ يقول الاتّجاه بلا حساب. */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <Panel title={H.today} note={H.todayNote} className="lg:col-span-2">
          <div className="overflow-x-auto">
            <table className="w-full text-sm tabular-nums">
              <thead>
                <tr className="text-xs text-ink-muted">
                  <th className="pb-2 text-start font-normal" />
                  <th className="pb-2 text-start font-normal">{H.colToday}</th>
                  <th className="pb-2 text-start font-normal">{H.colYesterday}</th>
                </tr>
              </thead>
              <tbody>
                {dayRows.map((r) => {
                  const tv = t[r.key];
                  const yv = y[r.key];
                  const up = known(tv) && known(yv) && tv > yv;
                  const down = known(tv) && known(yv) && tv < yv;
                  const bad = r.key === "failed" || r.key === "avg_delivery_min";
                  return (
                    <tr key={r.key} className="border-t border-line-soft">
                      <td className="py-2 text-ink-muted">{r.label}</td>
                      <td className="py-2 font-bold">
                        <span className="inline-flex items-center gap-1.5">
                          {dayCell(t, r)}
                          {(up || down) && (
                            <span className={`text-xs ${(up && !bad) || (down && bad) ? "text-success" : "text-danger"}`}>
                              {up ? "▲" : "▼"}
                            </span>
                          )}
                        </span>
                      </td>
                      <td className="py-2 text-ink-muted">{dayCell(y, r)}</td>
                    </tr>
                  );
                })}
              </tbody>
            </table>
          </div>
        </Panel>
        <Panel title={H.failuresTitle}>
          {data.missing.includes("failures") ? (
            <p className="text-sm text-ink-muted">{H.unknown}</p>
          ) : data.failures_today.length === 0 ? (
            <p className="flex items-center gap-1.5 text-sm font-bold text-success">
              <IconSuccess size={16} />
              {H.noFailures}
            </p>
          ) : (
            <ul className="space-y-1.5 text-sm">
              {data.failures_today.map((f) => (
                <li key={f.status + f.reason} className="flex items-center gap-2 rounded-control bg-field px-2.5 py-1.5">
                  <span className="min-w-0 flex-1">
                    {STATUS[f.status] ?? f.status}
                    {m.common.nameSeparator}
                    {f.reason ? (FAIL[f.reason] ?? f.reason) : H.noReason}
                  </span>
                  <span className="rounded-full bg-danger-solid px-2 py-0.5 text-xs font-bold text-on-solid">{fmtNum(f.count)}</span>
                </li>
              ))}
            </ul>
          )}
        </Panel>
      </div>

      {/* ══ ٥ · آخر الأحداث + المنصّة بالأرقام ══ */}
      <div className="grid grid-cols-1 gap-4 xl:grid-cols-2">
        <Panel title={H.events}>
          {data.missing.includes("events") ? (
            <p className="text-sm text-ink-muted">{H.unknown}</p>
          ) : data.events.length === 0 ? (
            <p className="text-sm text-ink-muted">{H.noEvents}</p>
          ) : (
            <ol className="relative space-y-3 border-s border-line-soft ps-4 text-sm">
              {data.events.map((e) => {
                const href = eventHref(e);
                const what = [ACTIONS[e.action] ?? e.action, e.label || (e.number != null ? `#${e.number}` : "")]
                  .filter(Boolean)
                  .join(m.common.nameSeparator);
                return (
                  <li key={e.id} className="relative">
                    <span className="absolute -start-[1.3rem] top-1.5 h-2 w-2 rounded-full bg-accent" />
                    <div className="flex flex-wrap items-baseline gap-x-2">
                      {href ? (
                        <Link href={href} className="min-w-0 flex-1 font-medium hover:underline">
                          {what}
                        </Link>
                      ) : (
                        <span className="min-w-0 flex-1 font-medium">{what}</span>
                      )}
                      <span className="text-xs text-ink-muted">{fmtTime(e.created_at)}</span>
                    </div>
                    {e.actor_name && <p className="text-xs text-ink-muted">{H.by.replace("{name}", e.actor_name)}</p>}
                  </li>
                );
              })}
            </ol>
          )}
        </Panel>

        <Panel title={H.platform}>
          {data.missing.includes("platform") ? (
            <p className="text-sm text-ink-muted">{H.unknown}</p>
          ) : (
            <div className="space-y-4">
              <CountTable
                rows={data.platform.groups.map((g) => ({
                  ...g,
                  label: (H.group as Record<string, string>)[g.key] ?? g.key,
                  href: GROUP_HREF[g.key] ?? "/dashboard",
                }))}
              />
              <div>
                <p className="mb-1 text-xs font-bold text-ink-muted">{H.staffTitle}</p>
                <CountTable
                  rows={data.platform.staff.map((g) => ({
                    ...g,
                    label: roleLabelByCode(g.key),
                    href: `/dashboard/users?role=${g.key}`,
                  }))}
                />
              </div>
              <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
                <Tile icon={IconOrder} label={H.customersOrderedWeek} value={show(data.platform.customers_ordered_week)} onClick={go("/dashboard/history")} />
                <Tile icon={IconDriver} label={H.driversActive} value={show(data.platform.drivers_active)} onClick={go("/dashboard/users?role=driver&status=active")} />
                <Tile icon={IconDriver} label={H.driversSuspended} value={show(data.platform.drivers_suspended)} tone={zeroMuted(data.platform.drivers_suspended)} onClick={go("/dashboard/users?role=driver&status=suspended")} />
                <Tile icon={IconDriver} label={H.driversBlocked} value={show(data.platform.drivers_blocked)} tone={zeroMuted(data.platform.drivers_blocked)} onClick={go("/dashboard/users?role=driver&status=blocked")} />
                <Tile icon={IconStore} label={H.storesActive} value={show(data.platform.stores_active)} onClick={go("/dashboard/opsmap")} />
                <Tile icon={IconStore} label={H.storesNotActive} value={show(data.platform.stores_not_active)} tone={zeroMuted(data.platform.stores_not_active)} onClick={go("/dashboard/opsmap")} />
              </div>
            </div>
          )}
        </Panel>
      </div>

      {/* ── زوّارُ الموقع والتحميلات (طلبُ المالك ٢٠٢٦-١٠-٠٧) ── */}
      <SiteStatsSection />
    </div>
  );
}

/** **الصفرُ يهدأ** — رقمٌ صفريٌّ رماديّ، وما عداه عاديّ. */
const zeroMuted = (n: N): TileTone => (known(n) && n === 0 ? "muted" : "default");

/** **جدولُ الأعداد** — الكلُّ وجديدُ اليوم وجديدُ الأسبوع، وكلُّ سطرٍ يفتح صفحته. */
function CountTable({ rows }: { rows: (Count & { label: string; href: string })[] }) {
  return (
    <div className="overflow-x-auto">
      <table className="w-full text-sm tabular-nums">
        <thead>
          <tr className="text-xs text-ink-muted">
            <th className="pb-1.5 text-start font-normal" />
            <th className="pb-1.5 text-start font-normal">{H.colTotal}</th>
            <th className="pb-1.5 text-start font-normal">{H.colNewToday}</th>
            <th className="pb-1.5 text-start font-normal">{H.colNewWeek}</th>
          </tr>
        </thead>
        <tbody>
          {rows.map((g) => (
            <tr key={g.key} className="border-t border-line-soft">
              <td className="py-1.5">
                <Link href={g.href} className="hover:underline">
                  {g.label}
                </Link>
              </td>
              <td className="py-1.5 font-bold">{fmtNum(g.total)}</td>
              <td className={`py-1.5 ${g.today ? "text-success" : "text-ink-muted"}`}>{g.today ? `+${fmtNum(g.today)}` : "0"}</td>
              <td className={`py-1.5 ${g.week ? "text-success" : "text-ink-muted"}`}>{g.week ? `+${fmtNum(g.week)}` : "0"}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
