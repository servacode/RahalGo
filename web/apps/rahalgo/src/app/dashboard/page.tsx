"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **رئيسيّةُ مدير المنصّة — كلُّ ما يجري في صفحةٍ واحدة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-١٠-٠٤: «لازم أفهم كلّ شي عم يصير بالمنصّة لأصغر تفصيل
 *  بدون ما أتنقّل بين الصفحات» · «الرئيسيّة لمدير المنصّة وحدَه».)
 *
 * **من أعلى إلى أسفل**: بانتظار قرارك · هلق · حركةُ المال اليوم والرصيدُ هلق ·
 * اليوم مقابل أمس · أسبابُ الفشل · آخر الأحداث · المنصّة بالأرقام · زوّارُ الموقع.
 * **وشريطُ الطوارئ فوقها في كلّ صفحة** (`EmergencyBanner` في `layout.tsx`).
 *
 * # القواعد
 *
 *  - **كلُّ رقمٍ بابٌ** يفتح صفحةً فيها الرقمُ نفسُه — والمحرّكُ يعدّ بشرط
 *    صفحته (`admin_overview_handlers.go`)، **واختبارٌ يقارن الرقمين.**
 *    **وبابٌ لا يملكه صاحبُ الجلسة يُرسَم رقماً لا زرّاً** (تدقيقُ ٢٠٢٦-١٠-٢٧).
 *  - **الرقمُ الذي لم يُقرأ «غير معروف»** لا صفرٌ أخضر.
 *  - **الصفرُ رماديٌّ هادئ**، والأحمرُ للمستعجل وحدَه — **والأصفارُ تحت «عرض الكل».**
 *  - **فشلُ التحديث لا يمحو الصفحة** — تبقى آخرُ أرقامٍ وصلت ومعها سطرٌ يقول ذلك.
 *  - **«اليوم» يومُ المنصّة** — يبدأ بساعة فتح الدوام (`day_start_hour` من المحرّك)،
 *    **والروابطُ تحمل اليومَ نفسَه** (`today`) لا يومَ المتصفّح.
 */

import { useCallback, useEffect, useState, type ReactNode } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import {
  getMessages,
  defaultLocale,
  fmtNum,
  fmtMoney,
  fmtSpan,
  fmtTime,
  fmtDateTime,
  fmtLongDate,
  damascusDay,
  errorText,
} from "@rahalgo/i18n";
import {
  useLiveRefresh,
  Alert,
  Button,
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
  IconUsers,
  IconUser,
  IconStar,
  IconHourglass,
  IconPromos,
  IconLocation,
  IconTrendUp,
  IconTrendDown,
  IconTrendFlat,
  IconStatus,
  IconRoles,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useCanCall, useCanOpen } from "@/lib/policy";
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
const SETTING_KEYS = m.admin.settings.keys as Record<string, { label?: string } | undefined>;

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
  day_start_hour?: number;
  ticket_late_hours?: number;
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
  yesterday_same_time?: Day;
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

type IconT = typeof IconWarning;

/** **رقمٌ أو «غير معروف»** — ولا صفرَ مكانَ ما لم يُقرأ. */
const known = (n: N): n is number => typeof n === "number";
const show = (n: N, f: (x: number) => string = fmtNum) => (known(n) ? f(n) : H.unknown);

/**
 * **نبرةُ البطاقة**: غائبٌ عاديّ · صفرٌ رماديّ · وفوق الصفر بنبرة أهمّيّته.
 * **والأحمرُ للمستعجل وحدَه** — ما ينتظره إنسانٌ في الشارع.
 */
const toneOf = (n: N, urgent: boolean): TileTone =>
  !known(n) ? "default" : n === 0 ? "muted" : urgent ? "danger" : "warning";

/** **ترتيبُ «بانتظار قرارك»**: المستعجلُ الذي فيه رقمٌ · ثمّ العاديُّ الذي فيه رقم · ثمّ الأصفار. */
const rank = (c: { urgent: boolean; n: N }) => ((c.n ?? 0) > 0 ? (c.urgent ? 2 : 1) : 0);

/** **الصفرُ يهدأ** — رقمٌ صفريٌّ رماديّ، وما عداه عاديّ. */
const zeroMuted = (n: N): TileTone => (known(n) && n === 0 ? "muted" : "default");

/** **نبرةُ رقمٍ له إشارة** — الربحُ أخضر والخسارةُ حمراء والصفرُ رماديّ. */
const signTone = (n: N): TileTone => (!known(n) ? "default" : n > 0 ? "success" : n < 0 ? "danger" : "muted");

const HOUR = 3600_000;

/**
 * **أيُّ صفحةٍ يفتحها صاحبُ الجلسة** — بقدرات بنود القائمة نفسِها (`layout.tsx`). **و`platform.overview`
 * قد تُمنح لدورٍ مخصَّص** — فبطاقةٌ تفتح باباً لا يملكه تُرسم رقماً لا زرّاً.
 */
const PAGE_CAPS: [string, string[]][] = [
  ["/dashboard/emergencies", ["emergencies.manage"]],
  ["/dashboard/orders", ["orders.read"]],
  ["/dashboard/history", ["orders.read"]],
  ["/dashboard/opsmap", ["orders.read"]],
  ["/dashboard/expansion", ["orders.read"]],
  ["/dashboard/ops", ["orders.read", "observability.read"]],
  ["/dashboard/tickets", ["support.manage", "finance.read", "disputes.manage"]],
  ["/dashboard/treasury", ["finance.read"]],
  ["/dashboard/cash", ["finance.read"]],
  ["/dashboard/payouts", ["finance.read"]],
  ["/dashboard/obligations", ["finance.read"]],
  ["/dashboard/profits", ["finance.read"]],
  ["/dashboard/leads", ["merchants.verify"]],
  ["/dashboard/users", ["users.read"]],
  ["/dashboard/sections", ["market.manage"]],
  ["/dashboard/promos", ["offers.manage"]],
  ["/dashboard/settings", ["settings.general.manage", "settings.financial.manage", "settings.security.manage"]],
];

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
    // **النزاعُ إلى تبويبه** — كان يفتح الخسائر (تدقيقُ ٢٠٢٦-١٠-٢٧).
    case "dispute":
      return "/dashboard/tickets?tab=disputes";
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

/** **ماذا وقع** — الفعلُ واسمُ صاحبه، **والإعدادُ باسمه العربيّ** والنزاعُ بطرفه ورقم طلبه. */
function eventWhat(e: Overview["events"][number]): string {
  const name = e.entity === "setting" && !e.label ? (SETTING_KEYS[e.entity_id]?.label ?? "") : e.label;
  return [ACTIONS[e.action] ?? e.action, name, e.number != null ? `#${e.number}` : ""]
    .filter(Boolean)
    .join(m.common.nameSeparator);
}

const AWAITING: {
  key: keyof Overview["awaiting"];
  label: string;
  href: string;
  urgent: boolean;
  icon: IconT;
}[] = [
  { key: "emergencies_open", label: H.emergenciesOpen, href: "/dashboard/emergencies", urgent: true, icon: IconWarning },
  { key: "reports_waiting", label: H.reportsWaiting, href: "/dashboard/orders?awaiting=1", urgent: true, icon: IconWarning },
  { key: "orders_unassigned", label: H.ordersUnassigned, href: "/dashboard/orders?filter=no_driver", urgent: true, icon: IconOrder },
  { key: "tickets_late", label: "", href: "/dashboard/tickets?late=1", urgent: true, icon: IconSupport },
  { key: "drivers_over_cash", label: H.driversOverCash, href: "/dashboard/cash?filter=over", urgent: true, icon: IconBalance },
  // **نقدٌ لم يُسلَّم منذ أيّام** — بعد عدد الأيّام في الإعدادات (قرارُ المالك ٢٠٢٦-١٠-٠٤).
  { key: "drivers_cash_overdue", label: H.driversCashOverdue, href: "/dashboard/cash?filter=overdue", urgent: true, icon: IconBalance },
  { key: "tickets_open", label: H.ticketsOpen, href: "/dashboard/tickets?status=unresolved", urgent: false, icon: IconSupport },
  // **والتعويضاتُ تبويبٌ في «المشاكل والتعويضات»** (٢٠٢٦-١٠-٢٧) — لا صفحتُها القديمة.
  { key: "compensations_pending", label: H.compensationsPending, href: "/dashboard/tickets?tab=compensations", urgent: false, icon: IconWallet },
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
  items: "/dashboard/sections",
  promos: "/dashboard/promos",
};

const GROUP_ICON: Record<string, IconT> = {
  customer: IconUsers,
  driver: IconDriver,
  sales: IconUser,
  stores: IconStore,
  items: IconGrid,
  zones: IconLocation,
  promos: IconPromos,
};

/** **حالُ الصفحة قبل الأرقام** — الرأسُ نفسُه وهيكلُ بطاقاتٍ، أو دائرةٌ وجملةٌ وزرٌّ بعرضٍ كامل. */
function Shell({ children }: { children: ReactNode }) {
  return (
    <div className="space-y-4">
      <PageHero icon={IconGrid} title={H.title} actions={<PlatformSwitch compact />} />
      {children}
    </div>
  );
}

export default function DashboardPage() {
  const router = useRouter();
  const { can } = useAuth();
  const canCall = useCanCall();
  const canOpen = useCanOpen();
  const [data, setData] = useState<Overview | null>(null);
  const [error, setError] = useState("");
  const [loadedAt, setLoadedAt] = useState<number | null>(null);
  const [busy, setBusy] = useState(false);
  const [showZeros, setShowZeros] = useState(false);
  const [, tick] = useState(0);

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

  // **و«آخر تحديث قبل…» يمشي وحدَه** — كلَّ نصف دقيقة. **والأرقامُ نفسُها كلَّ دقيقة**: العالقُ والمتاجرُ
  // الفاتحةُ والشكاوى المتأخّرةُ تتبدّل مع الوقت ولو لم يقع حدث (تدقيقُ ٢٠٢٦-١٠-٢٧).
  useEffect(() => {
    const t = setInterval(() => tick((x) => x + 1), 30_000);
    const r = setInterval(() => {
      if (document.visibilityState === "visible") load();
    }, 60_000);
    return () => {
      clearInterval(t);
      clearInterval(r);
    };
  }, [load]);

  // **حيّةٌ على كلّ ما يغيّر أرقامَها** — والشكوى (`ticket`) كانت غائبة.
  useLiveRefresh(
    ["order", "orders", "account", "lead", "wallet", "alerts", "ticket", "emergency",
      "driver", "merchant", "dispute", "settings", "user"],
    load,
  );

  /** **أيفتح صاحبُ الجلسة هذا الرابط؟** — وإلّا رُسمت البطاقةُ رقماً. */
  const mayOpen = (href: string): boolean => {
    const [path = "", qs = ""] = href.split("?");
    if (/^\/dashboard\/users\/[^/]+$/.test(path)) return canOpen.user;
    if (/^\/dashboard\/merchants\/[^/]+$/.test(path)) return canOpen.merchant;
    if (path === "/dashboard/tickets") {
      const tab = new URLSearchParams(qs).get("tab");
      if (tab === "disputes") return canCall("GET", "/disputes");
      if (tab === "compensations") return canCall("GET", "/compensations");
      if (!tab && !new URLSearchParams(qs).get("t")) return canCall("GET", "/tickets");
    }
    const hit = PAGE_CAPS.find(([p]) => path === p);
    return hit ? hit[1].some((c) => can(c)) : false;
  };
  const go = (href: string) => (mayOpen(href) ? () => router.push(href) : undefined);

  if (!data) {
    return (
      <Shell>
        {error ? (
          <section className="surface flex flex-col items-center gap-3 p-8 text-center">
            <span className="inline-flex h-16 w-16 items-center justify-center rounded-full bg-danger-solid text-on-solid">
              <IconWarning size={28} />
            </span>
            <h2 className="heading-section">{H.loadFailedTitle}</h2>
            <p className="max-w-md text-sm text-ink-muted">{H.loadFailedHint}</p>
            <p className="max-w-md text-xs text-ink-muted">{error}</p>
            <Button className="w-full sm:max-w-xs" onClick={load} disabled={busy}>
              {H.retry}
            </Button>
          </section>
        ) : (
          <div className="space-y-4" aria-busy="true">
            {[8, 4, 6].map((n, i) => (
              <section key={i} className="surface p-4">
                <span className="mb-3 block h-4 w-32 animate-pulse rounded-control bg-field" />
                <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-4">
                  {Array.from({ length: n }, (_, j) => (
                    <span key={j} className="block h-16 animate-pulse rounded-card bg-field" />
                  ))}
                </div>
              </section>
            ))}
          </div>
        )}
      </Shell>
    );
  }

  const a = data.awaiting;
  const lv = data.live;
  const t = data.today_stats;
  const y = data.yesterday_stats;
  const ys = data.yesterday_same_time;
  const startHour = data.day_start_hour ?? 0;
  const lateHours = data.ticket_late_hours ?? 2;
  const today = data.today;
  const hourText = fmtNum(startHour);
  const ago =
    loadedAt === null
      ? ""
      : Date.now() - loadedAt < 60_000
        ? H.updatedNow
        : H.updatedAgo.replace("{t}", fmtSpan((Date.now() - loadedAt) / 1000));
  const dayLine = H.dayLine.replace("{date}", fmtLongDate(`${today}T12:00:00Z`)).replace("{h}", hourText);
  /** **يومُ المنصّة لحدثٍ** — بمنطقتها ومن ساعة بدئها، **كالمحرّك** (`overviewDays`). */
  const platformDayOf = (iso: string) => damascusDay(Date.parse(iso) - startHour * HOUR);
  const weekFrom = damascusDay(Date.parse(`${today}T12:00:00Z`) - 6 * 24 * HOUR);
  const profitsToday = `/dashboard/profits?from=${today}&to=${today}`;

  // ── بانتظار قرارك: ما فيه رقمٌ أوّلاً، والأصفارُ تحت «عرض الكل» ──
  const awaitingItems = AWAITING.map((c) => ({
    ...c,
    label: c.key === "tickets_late" ? H.ticketsLate.replace("{h}", fmtNum(lateHours)) : c.label,
    n: a[c.key],
  })).sort((x, z) => rank(z) - rank(x));
  const zeros = awaitingItems.filter((c) => known(c.n) && c.n === 0);
  const pending = awaitingItems.filter((c) => !(known(c.n) && c.n === 0));
  const allClear = pending.length === 0;
  const visibleAwaiting = showZeros ? awaitingItems : pending;

  // ── اليوم مقابل أمس ──
  const dayRows: { label: string; key: keyof Day; icon: IconT; bad?: boolean; f?: (x: number) => string }[] = [
    { label: H.rowOrders, key: "orders", icon: IconOrder },
    { label: H.rowDelivered, key: "delivered", icon: IconSuccess },
    { label: H.rowFailed, key: "failed", icon: IconWarning, bad: true },
    { label: H.rowAvgDelivery, key: "avg_delivery_min", icon: IconHourglass, bad: true, f: (x) => H.minutes.replace("{n}", fmtNum(x)) },
    { label: H.rowNewCustomers, key: "new_customers", icon: IconUsers },
    { label: H.rowPlatformRating, key: "avg_platform_rating", icon: IconStar, f: (x) => H.stars.replace("{n}", fmtNum(x)) },
    { label: H.rowDriverRating, key: "avg_driver_rating", icon: IconStar, f: (x) => H.stars.replace("{n}", fmtNum(x)) },
  ];
  // **ومتوسّطٌ بلا مادّةٍ ليس مجهولاً** — لا تسليمَ ولا تقييمَ اليوم: شرطةٌ هادئة.
  const dayCell = (d: Day | undefined, r: (typeof dayRows)[number]) => {
    if (!d) return H.unknown;
    const v = d[r.key];
    if (v === null && r.key.startsWith("avg_") && known(d.orders)) return "—";
    return show(v, r.f);
  };

  const health = (
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
  ));

  const stuckOn = known(lv.orders_stuck) && lv.orders_stuck > 0;
  const treasuryNeg = known(data.money.treasury_balance) && data.money.treasury_balance < 0;
  const netNeg = known(data.money.net) && data.money.net < 0;

  return (
    <div className="space-y-4">
      {/* **رأسٌ موحّد** (قرارُ المالك ٢٠٢٦-١٠-٢٦) — اليومُ وساعةُ بدئه وآخرُ تحديث، والدوامُ أوّلُ ما يُرى
          (طلبُ المالك ٢٠٢٦-١٠-١٨)، والتحديثُ بجانبه. */}
      <PageHero
        icon={IconGrid}
        title={H.title}
        subtitle={
          <>
            {dayLine}
            {ago && (
              <>
                {m.common.nameSeparator}
                {ago}
              </>
            )}
          </>
        }
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
          {health}
        </div>
      </PageHero>
      {error && <Alert tone="warning">{H.refreshFailed}</Alert>}
      {data.missing.length > 0 && <Alert tone="warning">{H.missing}</Alert>}
      {/* **بياناتُ التواصل التجريبيّة** — قرارُ المالك ٢٠٢٦-١٠-٠٤ (الإعدادات ١٥). */}
      <ContactTestAlert />

      {/* ══ ١ · بانتظار قرارك ══ — ما فيه رقمٌ وحدَه، المستعجلُ أحمرُ أوّلاً والعاديُّ برتقاليّ. */}
      <Panel
        title={H.awaiting}
        note={allClear ? undefined : H.awaitingHint}
        actions={
          zeros.length > 0 && (
            <Button variant="ghost" onClick={() => setShowZeros((v) => !v)}>
              {showZeros ? H.hideZeros : H.showZeros.replace("{n}", fmtNum(zeros.length))}
            </Button>
          )
        }
      >
        {allClear && (
          <div className={`flex items-center gap-3 rounded-card bg-success-tint p-3 ${showZeros ? "mb-3" : ""}`}>
            <span className="inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full bg-success-solid text-on-solid">
              <IconSuccess size={20} />
            </span>
            <div className="min-w-0">
              <p className="text-sm font-bold">{H.allClear}</p>
              <p className="text-xs text-ink-muted">{H.allClearHint}</p>
            </div>
          </div>
        )}
        {visibleAwaiting.length > 0 && (
          <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-4 xl:grid-cols-5">
            {visibleAwaiting.map((c) => (
              <Tile
                key={c.key}
                icon={c.icon}
                label={c.label}
                value={show(c.n)}
                tone={toneOf(c.n, c.urgent)}
                onClick={go(c.href)}
              />
            ))}
          </div>
        )}
      </Panel>

      {/* ══ ٢ · هلق ══ — مسارُ الطلب مراحلَ (عموديّةً على الجوّال)، والعالقُ تحته، والسائقون والمتاجرُ بعناوينهم. */}
      <Panel title={H.now}>
        <div className="grid grid-cols-1 gap-4 xl:grid-cols-3">
          <div className="xl:col-span-2">
            <p className="mb-2 text-xs font-bold text-ink-muted">
              {H.ordersTitle}
              {m.common.nameSeparator}
              {show(lv.orders_open)}
            </p>
            <div className="flex flex-col gap-1.5 sm:flex-row sm:items-stretch">
              {STAGES.map((s, i) => {
                const n = lv.stages[s];
                const on = known(n) && n > 0;
                const href = `/dashboard/orders?stage=${s}`;
                const open = go(href);
                return (
                  <div key={s} className="flex items-center gap-1.5 sm:min-w-0 sm:flex-1">
                    <button
                      type="button"
                      onClick={open}
                      disabled={!open}
                      className={`flex flex-1 items-center gap-3 rounded-card border p-3 text-start transition-colors enabled:hover:bg-row-hover sm:flex-col sm:gap-1 sm:text-center ${
                        on ? "border-accent" : "border-line-soft"
                      }`}
                    >
                      <span className={`figure leading-none ${on ? "" : "text-ink-muted"}`}>{show(n)}</span>
                      <span className="text-xs text-ink-muted">{H.stage[s]}</span>
                    </button>
                    {i < STAGES.length - 1 && <IconPrev size={14} className="hidden shrink-0 text-ink-muted sm:block" />}
                  </div>
                );
              })}
            </div>
            {/* **والعالقُ يفتح فلترَ «عالق» نفسَه** — كان يفتح كلَّ الطلبات (تدقيقُ ٢٠٢٦-١٠-٢٧). */}
            <div className="mt-2">
              <Tile
                icon={IconWarning}
                label={H.stuck}
                value={show(lv.orders_stuck)}
                sub={stuckOn ? H.stuckHint : undefined}
                tone={!known(lv.orders_stuck) ? "default" : stuckOn ? "danger" : "muted"}
                onClick={go("/dashboard/orders?filter=stuck")}
              />
            </div>
          </div>
          <div className="space-y-3">
            <div>
              <p className="mb-2 text-xs font-bold text-ink-muted">
                {H.drivers}
                <span className="font-normal">
                  {m.common.nameSeparator}
                  {H.driversNote}
                </span>
              </p>
              <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 xl:grid-cols-2">
                <Tile icon={IconDriver} label={H.driversOnShift} value={show(lv.drivers_on_shift)} tone={zeroMuted(lv.drivers_on_shift)} onClick={go("/dashboard/opsmap?on_shift=true&status=active")} />
                <Tile icon={IconDriver} label={H.driversFree} value={show(lv.drivers_free)} tone={known(lv.drivers_free) && lv.drivers_free > 0 ? "success" : zeroMuted(lv.drivers_free)} onClick={go("/dashboard/opsmap?on_shift=true&status=active&has_active=false")} />
                <Tile icon={IconDriver} label={H.driversBusy} value={show(lv.drivers_busy)} tone={zeroMuted(lv.drivers_busy)} onClick={go("/dashboard/opsmap?on_shift=true&status=active&has_active=true")} />
              </div>
            </div>
            <div>
              <p className="mb-2 text-xs font-bold text-ink-muted">{H.storesTitle}</p>
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
        </div>
      </Panel>

      {/* ══ ٣ · المال ══ — حركةُ اليوم (تفتح الأرباحَ على اليوم نفسِه) والرصيدُ هلق، كلٌّ بلوحته. */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <Panel title={H.money} note={H.moneyNote} className="lg:col-span-2">
          <div className="grid grid-cols-2 gap-2 sm:grid-cols-4">
            <Tile
              icon={IconWallet}
              label={H.net}
              value={show(data.money.net, fmtMoney)}
              sub={netNeg ? H.netNegative : H.netHint}
              tone={signTone(data.money.net)}
              onClick={go(profitsToday)}
            />
            <Tile icon={IconBalance} label={H.sales} value={show(data.money.sales, fmtMoney)} tone={zeroMuted(data.money.sales)} onClick={go(profitsToday)} />
            <Tile icon={IconTrendDown} label={H.losses} value={show(data.money.losses, fmtMoney)} tone={known(data.money.losses) && data.money.losses > 0 ? "warning" : zeroMuted(data.money.losses)} onClick={go(profitsToday)} />
            <Tile
              icon={IconWallet}
              label={H.compensations}
              value={show(data.money.compensations_amount, fmtMoney)}
              sub={known(data.money.compensations_count) ? H.compensationsSub.replace("{n}", fmtNum(data.money.compensations_count)) : undefined}
              tone={zeroMuted(data.money.compensations_amount)}
              onClick={go("/dashboard/tickets?tab=compensations")}
            />
          </div>
        </Panel>
        <Panel title={H.balanceNow} note={H.balanceNote}>
          <div className="grid grid-cols-2 gap-2 lg:grid-cols-1">
            <Tile
              icon={IconWallet}
              label={H.treasury}
              value={show(data.money.treasury_balance, fmtMoney)}
              sub={treasuryNeg ? H.treasuryNegative : undefined}
              tone={treasuryNeg ? "danger" : "accent"}
              onClick={go("/dashboard/treasury?tab=statement")}
            />
            <Tile icon={IconBalance} label={H.cashHeld} value={show(data.money.cash_held, fmtMoney)} tone={zeroMuted(data.money.cash_held)} onClick={go("/dashboard/cash")} />
          </div>
        </Panel>
      </div>

      {/* ══ ٤ · اليوم مقابل أمس ══ — رقمُ اليوم كبيرٌ، **ويُقارَن بأمس لنفس الساعة** (اليومُ لم ينتهِ بعد،
          ومقارنتُه بيومٍ كاملٍ تقول «تراجع» كلَّ صباح)، وأمسُ كلُّه سطرٌ رماديّ. */}
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-3">
        <Panel title={H.today} note={H.todayNote.replace("{h}", hourText)} className="lg:col-span-2">
          <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 xl:grid-cols-4">
            {dayRows.map((r) => {
              const tv = t[r.key];
              const cmp = ys ? ys[r.key] : undefined;
              const both = known(tv) && known(cmp);
              const up = both && tv > cmp;
              const down = both && tv < cmp;
              const good = (up && !r.bad) || (down && r.bad);
              const Trend = up ? IconTrendUp : down ? IconTrendDown : IconTrendFlat;
              return (
                <Tile
                  key={r.key}
                  icon={r.icon}
                  label={r.label}
                  value={dayCell(t, r)}
                  tone={known(tv) && tv === 0 ? "muted" : "default"}
                  sub={H.yesterdayFull.replace("{v}", dayCell(y, r))}
                  extra={
                    ys && (
                      <span
                        className={`inline-flex items-center gap-1 rounded-full px-2 py-0.5 text-2xs font-bold ${
                          !both || (!up && !down) ? "bg-field text-ink-muted" : good ? "bg-success-tint text-success" : "bg-danger-tint text-danger"
                        }`}
                      >
                        <Trend size={12} />
                        {H.yesterdaySameTime.replace("{v}", dayCell(ys, r))}
                      </span>
                    )
                  }
                />
              );
            })}
          </div>
        </Panel>

        {/* **أسبابُ الفشل تُضغط** — كلُّ سطرٍ يفتح السجلَّ على اليوم وبحالته، **والفشلُ بلا سببٍ برتقاليّ** يُفحص. */}
        <Panel title={H.failuresTitle}>
          {data.missing.includes("failures") ? (
            <p className="text-sm text-ink-muted">{H.unknown}</p>
          ) : data.failures_today.length === 0 ? (
            <div className="flex flex-col items-center gap-2 py-4 text-center">
              <span className="inline-flex h-12 w-12 items-center justify-center rounded-full bg-success-solid text-on-solid">
                <IconSuccess size={22} />
              </span>
              <p className="text-sm font-bold">{H.noFailures}</p>
              <p className="text-xs text-ink-muted">{H.noFailuresHint}</p>
            </div>
          ) : (
            <ul className="space-y-1.5 text-sm">
              {data.failures_today.map((f) => {
                const href = `/dashboard/history?status=${f.status}&from=${today}&to=${today}`;
                const open = go(href);
                const noReason = !f.reason;
                return (
                  <li key={f.status + f.reason}>
                    <button
                      type="button"
                      onClick={open}
                      disabled={!open}
                      className={`flex w-full items-center gap-2 rounded-card border px-3 py-2 text-start transition-colors enabled:hover:bg-row-hover ${
                        noReason ? "border-warning bg-warning-tint" : "border-line-soft"
                      }`}
                    >
                      <span className="min-w-0 flex-1">
                        {STATUS[f.status] ?? f.status}
                        {m.common.nameSeparator}
                        {noReason ? H.noReason : (FAIL[f.reason] ?? f.reason)}
                        {noReason && <span className="block text-2xs font-bold text-warning">{H.noReasonWarn}</span>}
                      </span>
                      <span className="rounded-full bg-danger-solid px-2 py-0.5 text-xs font-bold text-on-solid">{fmtNum(f.count)}</span>
                    </button>
                  </li>
                );
              })}
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
            <div className="flex flex-col items-center gap-2 py-6 text-center">
              <span className="inline-flex h-12 w-12 items-center justify-center rounded-full bg-accent text-on-bright">
                <IconStatus size={22} />
              </span>
              <p className="text-sm font-bold">{H.noEvents}</p>
              <p className="text-xs text-ink-muted">{H.noEventsHint}</p>
            </div>
          ) : (
            <ol className="relative space-y-3 border-s border-line-soft ps-4 text-sm">
              {data.events.map((e) => {
                const raw = eventHref(e);
                const href = raw && mayOpen(raw) ? raw : null;
                const what = eventWhat(e);
                // **والحدثُ القديمُ بتاريخه** — «2:05 م» وحدَها تبدو اليوم وهي من أسبوع.
                const when = platformDayOf(e.created_at) === today ? fmtTime(e.created_at) : fmtDateTime(e.created_at);
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
                      <span className="text-xs text-ink-muted">{when}</span>
                    </div>
                    {e.actor_name && <p className="text-xs text-ink-muted">{H.by.replace("{name}", e.actor_name)}</p>}
                  </li>
                );
              })}
            </ol>
          )}
        </Panel>

        {/* **بطاقةٌ لكلّ نوع** — المجموعُ كبيرٌ وتحته شارتا الجديد، بدل الجدولين. */}
        <Panel title={H.platform}>
          {data.missing.includes("platform") ? (
            <p className="text-sm text-ink-muted">{H.unknown}</p>
          ) : (
            <div className="space-y-4">
              <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
                {data.platform.groups.map((g) => (
                  <CountTile
                    key={g.key}
                    c={g}
                    icon={GROUP_ICON[g.key] ?? IconGrid}
                    label={(H.group as Record<string, string>)[g.key] ?? g.key}
                    onClick={GROUP_HREF[g.key] ? go(GROUP_HREF[g.key]!) : undefined}
                  />
                ))}
              </div>
              {data.platform.staff.length > 0 && (
                <div>
                  <p className="mb-2 text-xs font-bold text-ink-muted">{H.staffTitle}</p>
                  <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
                    {data.platform.staff.map((g) => (
                      <CountTile key={g.key} c={g} icon={IconRoles} label={roleLabelByCode(g.key)} onClick={go(`/dashboard/users?role=${g.key}`)} />
                    ))}
                  </div>
                </div>
              )}
              <div>
                <p className="mb-2 text-xs font-bold text-ink-muted">{H.accountsTitle}</p>
                <div className="grid grid-cols-2 gap-2 sm:grid-cols-3">
                  <Tile icon={IconOrder} label={H.customersOrderedWeek} value={show(data.platform.customers_ordered_week)} tone={zeroMuted(data.platform.customers_ordered_week)} onClick={go(`/dashboard/history?from=${weekFrom}&to=${today}`)} />
                  <Tile icon={IconDriver} label={H.driversActive} value={show(data.platform.drivers_active)} tone={zeroMuted(data.platform.drivers_active)} onClick={go("/dashboard/users?role=driver&status=active")} />
                  <Tile icon={IconDriver} label={H.driversSuspended} value={show(data.platform.drivers_suspended)} tone={known(data.platform.drivers_suspended) && data.platform.drivers_suspended > 0 ? "warning" : zeroMuted(data.platform.drivers_suspended)} onClick={go("/dashboard/users?role=driver&status=suspended")} />
                  <Tile icon={IconDriver} label={H.driversBlocked} value={show(data.platform.drivers_blocked)} tone={known(data.platform.drivers_blocked) && data.platform.drivers_blocked > 0 ? "danger" : zeroMuted(data.platform.drivers_blocked)} onClick={go("/dashboard/users?role=driver&status=blocked")} />
                  {/* **وأعدادُ المتاجر رقمٌ بلا رابط** — لا قائمةَ فيها العددُ نفسُه (كانت تفتح الخريطة). */}
                  <Tile icon={IconStore} label={H.storesActive} value={show(data.platform.stores_active)} tone={zeroMuted(data.platform.stores_active)} />
                  <Tile icon={IconStore} label={H.storesNotActive} value={show(data.platform.stores_not_active)} tone={zeroMuted(data.platform.stores_not_active)} />
                </div>
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

/** **بطاقةُ عدد** — المجموعُ كبيرٌ وتحته شارتان: جديدُ اليوم وجديدُ آخر ٧ أيّام. */
function CountTile({ c, icon, label, onClick }: { c: Count; icon: IconT; label: string; onClick?: () => void }) {
  const pill = (n: number, text: string) => (
    <span
      className={`inline-flex rounded-full px-2 py-0.5 text-2xs font-bold ${
        n > 0 ? "bg-success-tint text-success" : "bg-field text-ink-muted"
      }`}
    >
      {text.replace("{n}", fmtNum(n))}
    </span>
  );
  return (
    <Tile
      icon={icon}
      label={label}
      value={fmtNum(c.total)}
      tone={c.total === 0 ? "muted" : "default"}
      onClick={onClick}
      extra={
        <>
          {pill(c.today, H.newToday)}
          {pill(c.week, H.newWeek)}
        </>
      }
    />
  );
}
