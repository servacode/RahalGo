"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الرئيسيّة — مركزُ القيادة، لكلّ دورٍ ما يملكه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (إعادةُ تصميم اللوحة ٢٠٢٦-١١-٠٣ — **وهي تنسخ «الرئيسيّةُ للمدير وحدَه» ٢٠٢٦-١٠-٠٤**.)
 *
 * تجيب على ثلاثة أسئلة من أوّل نظرة: **شو بدّه مني هلّق؟ · المال تمام؟ · المنصة عايشة؟**
 *
 *   ١ · بدّه منك انتباه   (`AttentionStrip`)  — عدّاداتٌ فوق الصفر، كلٌّ رابطٌ للحلّ.
 *   ٢ · نبضُ المنصة هلّق  (`SummaryStrip`)    — سائقون · متاجر · طلباتٌ جارية · متوسّط التوصيل.
 *   ٣ · المال اليوم       (`SummaryStrip`)    — لمن يملك المال.
 *   ثمّ ما تحتها: لكلّ دورٍ قائمتُه (عالقٌ · سحوبٌ ناطرة · شكاوى) ولمدير المنصّة تفاصيلُ اليوم.
 *
 * # القواعد
 *
 *  - **كلُّ كتلةٍ بقدرة بياناتها** — لا نداءَ ٤٠٣: الانتباهُ والنبضُ والرصيدُ من `/admin/menu-counts`
 *    (يردّ لكلّ موظّفٍ ما يملكه ويُنزل الباقيَ `null`)، وتفاصيلُ المدير من `/admin/overview`.
 *  - **كلُّ رقمٍ رابطٌ** يفتح قائمةً فيها الرقمُ نفسُه — وفتحُ الرابط بقدرته (`mayOpen`).
 *  - **الرقمُ الذي لم يُقرأ «غير معروف»** لا صفرٌ أخضر. **والصفرُ هادئ.**
 *  - **حيّةٌ**: بلا زرّ تحديث — تتحدّث مع الأحداث وكلَّ دقيقة، وفشلُ التحديث لا يمحو ما وصل.
 *  - **اليومُ يومُ المنصّة** — يبدأ بساعة فتح الدوام، والروابطُ تحمل اليومَ نفسَه.
 */

import { useEffect, useState, type ReactNode } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import {
  getMessages,
  defaultLocale,
  fmtNum,
  fmtMoney,
  fmtSpan,
  fmtDateTime,
  fmtLongDate,
  damascusDay,
} from "@rahalgo/i18n";
import {
  AdminAsync,
  AdminCard,
  AdminEntityCard,
  AdminEntityGrid,
  AdminLink,
  AdminPage,
  AttentionStrip,
  Alert,
  Button,
  ContactButtons,
  Row,
  StatusBadge,
  StoryTimeline,
  SummaryStrip,
  TONE_TEXT,
  agoText,
  useLiveData,
  IconGrid,
  IconOrder,
  IconDriver,
  IconStore,
  IconWallet,
  IconBalance,
  IconTrendDown,
  IconSuccess,
  IconWarning,
  IconHourglass,
  IconUsers,
  IconUser,
  IconStar,
  IconPromos,
  IconLocation,
  IconRoles,
  IconSupport,
  type AttentionItem,
  type StoryEvent,
  type SummaryItem,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useCanOpen } from "@/lib/policy";
import { roleLabelByCode } from "@/lib/rolemeta";
import ContactTestAlert from "@/components/admin/settings/contact-check";
import SiteStatsSection from "@/components/admin/SiteStatsSection";
import PlatformSwitch from "@/components/admin/settings/PlatformSwitch";
import { LIVE_KINDS, useMenuCounts, type MenuCountsData } from "@/components/admin/MenuCounts";

const m = getMessages(defaultLocale);
const H = m.admin.home;
const K = m.admin.kit;
const KA = K.attention;
const KH = K.home;
const ACTIONS = m.admin.audit.actions as Record<string, string>;
const STATUS = m.orders.status as Record<string, string>;
const FAIL = m.common.failReasons as Record<string, string>;
const SETTING_KEYS = m.admin.settings.keys as Record<string, { label?: string } | undefined>;
const ALERT_REASONS = m.admin.ordersPage.alertReasons as Record<string, string>;

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
  missing: string[];
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

interface StuckAlert {
  order_id: string;
  number: number;
  status: string;
  merchant_name: string;
  customer_phone: string;
  customer_phone_masked?: string;
  reason: string;
  minutes: number;
  since?: string;
}

interface PayoutRow {
  id: string;
  user_id: string;
  user_name: string;
  user_role: string;
  amount: number;
  available: number;
  created_at: string;
}

interface TicketRow {
  id: string;
  number: number;
  subject: string;
  customer_name: string;
  order_number: number | null;
  status: string;
  late?: boolean;
  created_at: string;
}

type IconT = typeof IconWarning;

/** **رقمٌ أو «غير معروف»** — ولا صفرَ مكانَ ما لم يُقرأ. */
const known = (n: N): n is number => typeof n === "number";
const show = (n: N, f: (x: number) => string = fmtNum) => (known(n) ? f(n) : H.unknown);
const num = (n: N) => (known(n) ? n : 0);

/** **الصفرُ يهدأ** — رقمٌ صفريٌّ رماديّ، وما عداه عاديّ. */
const zeroMuted = (n: N): SummaryItem["tone"] => (known(n) && n === 0 ? "muted" : "neutral");
/** **نبرةُ رقمٍ له إشارة** — الخسارةُ حمراء، والصفرُ هادئ، والباقي عاديّ. */
const lossTone = (n: N): SummaryItem["tone"] => (!known(n) ? "neutral" : n < 0 ? "danger" : n === 0 ? "muted" : "neutral");

const HOUR = 3600_000;

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

/** **قدراتُ فتح الصفحات** — كلُّ رابطٍ بقدرة بابه في المحرّك، وإلّا رُسم الرقمُ رقماً لا زرّاً. */
const PAGE_CAPS: [string, string[]][] = [
  ["/dashboard/emergencies", ["emergencies.manage"]],
  ["/dashboard/orders", ["orders.read"]],
  ["/dashboard/history", ["orders.read"]],
  ["/dashboard/opsmap", ["orders.read"]],
  ["/dashboard/expansion", ["orders.read"]],
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

/** **عنوانُ كتلة** — واحدٌ لكلّ كتلة، بلا شرحٍ تحته. */
function Block({ title, children }: { title: string; children: ReactNode }) {
  return (
    <section className="space-y-3" aria-label={title}>
      <h2 className="admin-h2">{title}</h2>
      {children}
    </section>
  );
}

/** **«بدّه منك انتباه» من عدّادات المحرّك** — كلُّ عدّادٍ بقدرته (`null` لمن لا يملكها فلا يُرسم). */
function attentionItems(a: MenuCountsData["attention"]): AttentionItem[] {
  return [
    { key: "emergencies", label: KA.emergencies, count: num(a.emergencies_open), href: "/dashboard/emergencies", tone: "danger" },
    { key: "stuck", label: KA.stuckOrders, count: num(a.stuck_orders), href: "/dashboard/orders?filter=stuck", tone: "danger" },
    { key: "unassigned", label: KA.unassigned, count: num(a.unassigned_orders), href: "/dashboard/orders?filter=no_driver", tone: "danger" },
    { key: "reports", label: KA.reports, count: num(a.reports_waiting), href: "/dashboard/orders?awaiting=1", tone: "danger" },
    {
      key: "silent",
      label: KA.storeSilent,
      count: num(a.stores_silent),
      href: "/dashboard/orders?filter=awaiting_accept",
      names: a.stores_silent_names ?? undefined,
    },
    {
      key: "cashOver",
      label: KA.cashOver,
      count: num(a.cash_over),
      href: "/dashboard/cash?filter=over",
      tone: "danger",
      names: a.cash_over_names ?? undefined,
    },
    { key: "cashOverdue", label: KA.cashOverdue, count: num(a.cash_overdue), href: "/dashboard/cash?filter=overdue", tone: "danger" },
    { key: "ticketsLate", label: KA.complaintsLate, count: num(a.tickets_late), href: "/dashboard/tickets?late=1", tone: "danger" },
    { key: "tickets", label: KA.complaints, count: num(a.tickets_open), href: "/dashboard/tickets?status=unresolved" },
    { key: "payouts", label: KA.payouts, count: num(a.payouts_pending), href: "/dashboard/payouts?status=pending" },
    { key: "compensations", label: KA.compensations, count: num(a.compensations_pending), href: "/dashboard/tickets?tab=compensations" },
    { key: "obligations", label: KA.obligations, count: num(a.obligations_overdue), href: "/dashboard/obligations?overdue=1" },
    { key: "obligationRequests", label: KA.obligationRequests, count: num(a.obligation_requests), href: "/dashboard/obligations?pending=1" },
    { key: "leads", label: KA.leads, count: num(a.leads_new), href: "/dashboard/leads" },
    { key: "expansion", label: KA.expansion, count: num(a.expansion_waiting), href: "/dashboard/expansion?filter=waiting" },
  ];
}

export default function DashboardPage() {
  const router = useRouter();
  const { can } = useAuth();
  const canOpen = useCanOpen();
  const hasOverview = can("platform.overview");
  const canOrders = can("orders.read");
  const canFinance = can("finance.read");
  const canSupport = can("support.manage");

  // **الانتباهُ والنبضُ والرصيدُ من نداء الهيكل الواحد** (القائمةُ تقرأ منه أيضاً).
  const { data: mc, loading: mcLoading, error: mcError, reload: reloadMc, loadedAt: mcAt } = useMenuCounts();

  // **وكلَّ دقيقة احتياطاً لحدثٍ فات** — والعالقُ والشكاوى المتأخرةُ تتبدّل مع الوقت ولو لم يقع حدث.
  const [tick, setTick] = useState(0);
  const [, clock] = useState(0);
  useEffect(() => {
    const r = setInterval(() => {
      if (document.visibilityState === "visible") setTick((x) => x + 1);
    }, 60_000);
    // **و«تحدّث قبل…» يمشي وحدَه** كلَّ نصف دقيقة.
    const c = setInterval(() => clock((x) => x + 1), 30_000);
    return () => {
      clearInterval(r);
      clearInterval(c);
    };
  }, []);

  const [ovAt, setOvAt] = useState<number | null>(null);
  const ov = useLiveData<Overview | null>(
    async () => {
      if (!hasOverview) return null;
      const d = await api<Overview>("/api/v1/admin/overview");
      setOvAt(Date.now());
      return d;
    },
    LIVE_KINDS,
    [hasOverview, tick],
    800,
  );

  // **قوائمُ الدور** — لمن لا يملك الرئيسيّةَ الكاملة: أقدمُ عالق · سحوبٌ ناطرة · شكاوى مفتوحة.
  const listRole = !hasOverview;
  const stuck = useLiveData<StuckAlert[]>(
    () => (listRole && canOrders ? api<StuckAlert[]>("/api/v1/admin/orders/alerts") : Promise.resolve([])),
    ["order", "orders", "alerts"],
    [listRole, canOrders, tick],
    800,
  );
  const payouts = useLiveData<{ payouts?: PayoutRow[] }>(
    () =>
      listRole && canFinance
        ? api<{ payouts?: PayoutRow[] }>("/api/v1/admin/payouts?status=pending&per_page=5")
        : Promise.resolve({ payouts: [] }),
    ["wallet", "payout"],
    [listRole, canFinance, tick],
    800,
  );
  const tickets = useLiveData<{ tickets?: TicketRow[] }>(
    () =>
      listRole && canSupport
        ? api<{ tickets?: TicketRow[] }>("/api/v1/admin/tickets?status=unresolved&per_page=5")
        : Promise.resolve({ tickets: [] }),
    ["ticket"],
    [listRole, canSupport, tick],
    800,
  );

  /** **أيفتح صاحبُ الجلسة هذا الرابط؟** — وإلّا رُسم الرقمُ رقماً. */
  const mayOpen = (href: string): boolean => {
    const path = href.split("?")[0] ?? "";
    if (/^\/dashboard\/users\/[^/]+$/.test(path)) return canOpen.user;
    if (/^\/dashboard\/merchants\/[^/]+$/.test(path)) return canOpen.merchant;
    const hit = PAGE_CAPS.find(([p]) => path === p);
    return hit ? hit[1].some((c) => can(c)) : false;
  };
  /** الرابطُ إن فُتح، وإلّا `undefined` (فيُرسم الرقمُ بلا رابط). */
  const link = (href: string) => (mayOpen(href) ? href : undefined);

  const o = ov.data;
  const a = mc?.attention;
  const pulse = mc?.pulse ?? null;
  const today = o?.today ?? mc?.today ?? "";
  const startHour = o?.day_start_hour ?? mc?.day_start_hour ?? 0;
  const platformDayOf = (iso: string) => damascusDay(Date.parse(iso) - startHour * HOUR);
  const weekFrom = today ? damascusDay(Date.parse(`${today}T12:00:00Z`) - 6 * 24 * HOUR) : "";
  const todayRange = `from=${today}&to=${today}`;

  // ── سطرُ الرأس الحيّ: اليومُ وساعةُ بدئه وآخرُ تحديث ──
  const at = hasOverview ? ovAt : mcAt;
  const ago =
    at === null
      ? ""
      : Date.now() - at < 60_000
        ? H.updatedNow
        : H.updatedAgo.replace("{t}", fmtSpan((Date.now() - at) / 1000));
  const dayLine = today
    ? H.dayLine.replace("{date}", fmtLongDate(`${today}T12:00:00Z`)).replace("{h}", fmtNum(startHour))
    : "";
  const meta = [dayLine, ago].filter(Boolean).join(m.common.nameSeparator);

  const refreshFailed = hasOverview ? ov.error && !!o : mcError && !!mc;

  // ══ ١ · بدّه منك انتباه ══
  const attention = a ? attentionItems(a) : [];

  // ══ ٢ · نبضُ المنصة ══
  const opsLink = link("/dashboard/opsmap");
  const ordersLink = link("/dashboard/orders");
  const pulseItems: SummaryItem[] = pulse
    ? [
        {
          key: "online",
          label: K.pulse.driversOnline,
          value: show(pulse.drivers_on_shift),
          icon: IconDriver,
          tone: zeroMuted(pulse.drivers_on_shift),
          href: opsLink && "/dashboard/opsmap?on_shift=true&status=active",
        },
        {
          key: "busy",
          label: K.pulse.driversBusy,
          value: show(pulse.drivers_busy),
          icon: IconDriver,
          tone: zeroMuted(pulse.drivers_busy),
          href: opsLink && "/dashboard/opsmap?on_shift=true&status=active&has_active=true",
        },
        {
          key: "free",
          label: K.pulse.driversFree,
          value: show(pulse.drivers_free),
          icon: IconDriver,
          tone: zeroMuted(pulse.drivers_free),
          href: opsLink && "/dashboard/opsmap?on_shift=true&status=active&has_active=false",
        },
        {
          key: "stores",
          label: K.pulse.storesOpen,
          value: show(pulse.stores_open_now),
          icon: IconStore,
          tone: zeroMuted(pulse.stores_open_now),
          meta: known(pulse.stores_active) ? H.storesOf.replace("{n}", fmtNum(pulse.stores_active)) : undefined,
          href: opsLink,
        },
        {
          key: "onWay",
          label: K.pulse.ordersOnWay,
          value: show(pulse.stages.to_customer),
          icon: IconOrder,
          tone: zeroMuted(pulse.stages.to_customer),
          href: ordersLink && "/dashboard/orders?stage=to_customer",
        },
        {
          key: "avg",
          label: K.pulse.avgDelivery,
          value: known(pulse.avg_delivery_min) ? H.minutes.replace("{n}", fmtNum(pulse.avg_delivery_min)) : "—",
          icon: IconHourglass,
          tone: known(pulse.avg_delivery_min) ? "neutral" : "muted",
          href: today ? link("/dashboard/history") && `/dashboard/history?status=delivered&${todayRange}` : undefined,
        },
      ]
    : [];

  const flowItems: SummaryItem[] = pulse
    ? [
        {
          key: "open",
          label: KH.flowOpen,
          value: show(pulse.orders_open),
          icon: IconOrder,
          tone: zeroMuted(pulse.orders_open),
          href: ordersLink,
        },
        ...STAGES.map<SummaryItem>((s) => ({
          key: s,
          label: H.stage[s],
          value: show(pulse.stages[s]),
          tone: zeroMuted(pulse.stages[s]),
          href: ordersLink && `/dashboard/orders?stage=${s}`,
        })),
      ]
    : [];

  // ══ ٣ · المال اليوم ══
  const profitsLink = today ? link("/dashboard/profits") && `/dashboard/profits?${todayRange}` : undefined;
  const cashLink = link("/dashboard/cash");
  const treasuryLink = link("/dashboard/treasury") && "/dashboard/treasury?tab=statement";
  const moneyItems: SummaryItem[] = o
    ? [
        {
          key: "sales",
          label: K.money.sales,
          value: show(o.money.sales, fmtMoney),
          icon: IconBalance,
          tone: zeroMuted(o.money.sales),
          href: profitsLink,
        },
        {
          key: "net",
          label: H.net,
          value: show(o.money.net, fmtMoney),
          icon: IconWallet,
          tone: lossTone(o.money.net),
          href: profitsLink,
        },
        {
          key: "losses",
          label: H.losses,
          value: show(o.money.losses, fmtMoney),
          icon: IconTrendDown,
          tone: known(o.money.losses) && o.money.losses > 0 ? "attention" : zeroMuted(o.money.losses),
          href: profitsLink,
        },
        {
          key: "comp",
          label: H.compensations,
          value: show(o.money.compensations_amount, fmtMoney),
          tone: zeroMuted(o.money.compensations_amount),
          meta: known(o.money.compensations_count) ? H.compensationsSub.replace("{n}", fmtNum(o.money.compensations_count)) : undefined,
          href: link("/dashboard/tickets") && "/dashboard/tickets?tab=compensations",
        },
        {
          key: "cash",
          label: K.money.cashWithDrivers,
          value: show(o.money.cash_held, fmtMoney),
          icon: IconDriver,
          tone: zeroMuted(o.money.cash_held),
          href: cashLink,
        },
        {
          key: "treasury",
          label: H.treasury,
          value: show(o.money.treasury_balance, fmtMoney),
          icon: IconWallet,
          tone: known(o.money.treasury_balance) && o.money.treasury_balance < 0 ? "danger" : "neutral",
          href: treasuryLink,
        },
      ]
    : mc?.money
      ? [
          {
            key: "cash",
            label: K.money.cashWithDrivers,
            value: show(mc.money.cash_held, fmtMoney),
            icon: IconDriver,
            tone: zeroMuted(mc.money.cash_held),
            href: cashLink,
          },
          {
            key: "treasury",
            label: H.treasury,
            value: show(mc.money.treasury_balance, fmtMoney),
            icon: IconWallet,
            tone: known(mc.money.treasury_balance) && mc.money.treasury_balance < 0 ? "danger" : "neutral",
            href: treasuryLink,
          },
        ]
      : [];

  // ── صحّةُ النظام: نقطةٌ بلونها لكلّ خدمة، والضغطُ يفتح المراقبة لمن يملكها ──
  const healthRow = o
    ? (
        [
          [H.healthApi, o.health.api],
          [H.healthDb, o.health.database],
          [H.healthWa, o.health.whatsapp],
          [H.healthRouting, o.health.routing],
        ] as const
      ).map(([label, st]) => {
        const tone = st === "ok" || st === "dev" ? "good" : st === "down" ? "bad" : "attention";
        const dot = tone === "good" ? "bg-success" : tone === "bad" ? "bg-danger" : "bg-warning";
        const body = (
          <>
            <span aria-hidden className={`h-2 w-2 shrink-0 rounded-full ${dot}`} />
            {label}
            <span className={TONE_TEXT[tone]}>{(H.healthState as Record<string, string>)[st] ?? H.unknown}</span>
          </>
        );
        const cls = "inline-flex items-center gap-1.5 rounded-full border border-line-soft px-3 py-1 text-sm";
        return can("observability.read") ? (
          <AdminLink key={label} href="/dashboard/ops" className={`${cls} hover:bg-row-hover`}>
            {body}
          </AdminLink>
        ) : (
          <span key={label} className={cls}>
            {body}
          </span>
        );
      })
    : [];

  const notices = (
    <>
      {refreshFailed && <Alert tone="warning">{H.refreshFailed}</Alert>}
      {o && o.missing.length > 0 && <Alert tone="warning">{H.missing}</Alert>}
      {/* **بياناتُ التواصل التجريبيّة** — قرارُ المالك ٢٠٢٦-١٠-٠٤ (الإعدادات ١٥). */}
      {hasOverview && <ContactTestAlert />}
    </>
  );

  const orderHref = (id: string) => (canOpen.order ? `/dashboard/history?order=${id}` : undefined);

  return (
    <AdminPage
      title={H.title}
      icon={IconGrid}
      meta={meta || undefined}
      actions={hasOverview ? <PlatformSwitch compact /> : undefined}
      notices={notices}
    >
      {/* ══ ١ · بدّه منك انتباه ══ */}
      {mcError && !mc ? (
        <AdminAsync loading={false} error onRetry={reloadMc}>
          {null}
        </AdminAsync>
      ) : (
        <AttentionStrip items={attention} loading={mcLoading && !mc} />
      )}

      {/* ══ ٢ · نبضُ المنصة هلّق ══ */}
      {pulse && (
        <Block title={K.pulse.title}>
          <SummaryStrip items={pulseItems} max={6} />
          <SummaryStrip items={flowItems} max={6} />
          {healthRow.length > 0 && (
            <div className="flex flex-wrap items-center gap-2">
              <span className="text-sm font-bold text-ink-muted">{KH.health}</span>
              {healthRow}
            </div>
          )}
        </Block>
      )}

      {/* ══ ٣ · المال اليوم ══ */}
      {moneyItems.length > 0 && (
        <Block title={K.money.title}>
          <SummaryStrip items={moneyItems} max={moneyItems.length > 4 ? 6 : 3} />
        </Block>
      )}

      {/* ══ قوائمُ الدور ══ */}
      {listRole && canOrders && (
        <Block title={KH.stuckList}>
          <AdminAsync
            loading={stuck.loading}
            error={stuck.error}
            onRetry={() => void stuck.reload()}
            hasData={!!stuck.data}
            empty={(stuck.data ?? []).length === 0}
            emptyTitle={KH.noStuck}
            emptyIcon={IconSuccess}
            rows={2}
          >
            <AdminEntityGrid>
              {(stuck.data ?? []).slice(0, 5).map((s) => {
                const tone = s.reason === "emergency" || s.reason === "no_driver" ? "danger" : "attention";
                const href = orderHref(s.order_id);
                return (
                  <AdminEntityCard
                    key={s.order_id}
                    tone={tone}
                    eyebrow={KH.orderNumber.replace("{n}", String(s.number))}
                    title={s.merchant_name}
                    href={href}
                    status={
                      <span className={`text-sm font-bold ${TONE_TEXT[tone === "danger" ? "bad" : "attention"]}`}>
                        {ALERT_REASONS[s.reason] ?? s.reason}
                      </span>
                    }
                    facts={[
                      { key: "since", label: KH.waiting, value: fmtSpan(s.minutes * 60) },
                      { key: "status", label: KH.status, value: STATUS[s.status] ?? s.status },
                    ]}
                    primary={
                      href && (
                        <Button size="sm" variant="secondary" onClick={() => router.push(href)}>
                          {KH.openOrder}
                        </Button>
                      )
                    }
                  >
                    {!s.customer_phone_masked && s.customer_phone && <ContactButtons phone={s.customer_phone} />}
                  </AdminEntityCard>
                );
              })}
            </AdminEntityGrid>
          </AdminAsync>
        </Block>
      )}

      {listRole && canFinance && (
        <Block title={KH.payoutsList}>
          <AdminAsync
            loading={payouts.loading}
            error={payouts.error}
            onRetry={() => void payouts.reload()}
            hasData={!!payouts.data}
            empty={(payouts.data?.payouts ?? []).length === 0}
            emptyTitle={KH.noPayouts}
            emptyIcon={IconSuccess}
            rows={2}
          >
            <AdminEntityGrid>
              {(payouts.data?.payouts ?? []).slice(0, 5).map((p) => (
                <AdminEntityCard
                  key={p.id}
                  tone="attention"
                  eyebrow={agoText(p.created_at)}
                  title={p.user_name}
                  href={canOpen.user ? `/dashboard/users/${p.user_id}` : undefined}
                  facts={[
                    { key: "amount", label: KH.amount, value: fmtMoney(p.amount) },
                    { key: "available", label: KH.available, value: fmtMoney(p.available) },
                    { key: "role", label: KH.role, value: roleLabelByCode(p.user_role), wide: true },
                  ]}
                  primary={
                    <Button size="sm" variant="secondary" onClick={() => router.push("/dashboard/payouts?status=pending")}>
                      {KH.review}
                    </Button>
                  }
                />
              ))}
            </AdminEntityGrid>
          </AdminAsync>
        </Block>
      )}

      {listRole && canSupport && (
        <Block title={KH.ticketsList}>
          <AdminAsync
            loading={tickets.loading}
            error={tickets.error}
            onRetry={() => void tickets.reload()}
            hasData={!!tickets.data}
            empty={(tickets.data?.tickets ?? []).length === 0}
            emptyTitle={KH.noTickets}
            emptyIcon={IconSupport}
            rows={2}
          >
            <AdminEntityGrid>
              {(tickets.data?.tickets ?? []).slice(0, 5).map((t) => (
                <AdminEntityCard
                  key={t.id}
                  tone={t.late ? "danger" : "attention"}
                  eyebrow={`${KH.ticketNumber.replace("{n}", String(t.number))}${m.common.nameSeparator}${agoText(t.created_at)}`}
                  title={t.subject || t.customer_name}
                  href={`/dashboard/tickets?t=${t.id}`}
                  status={<StatusBadge domain="complaint" status={t.status} />}
                  facts={[
                    { key: "customer", label: KH.customer, value: t.customer_name || "—" },
                    {
                      key: "order",
                      label: KH.order,
                      value: t.order_number != null ? `#${t.order_number}` : "—",
                      href: t.order_number != null && canOpen.order ? `/dashboard/history?q=${t.order_number}` : undefined,
                    },
                  ]}
                  primary={
                    <Button size="sm" variant="secondary" onClick={() => router.push(`/dashboard/tickets?t=${t.id}`)}>
                      {KH.review}
                    </Button>
                  }
                />
              ))}
            </AdminEntityGrid>
          </AdminAsync>
        </Block>
      )}

      {/* ══ تفاصيلُ اليوم — لمدير المنصّة ══ */}
      {hasOverview && (
        <AdminAsync loading={ov.loading} error={ov.error} onRetry={() => void ov.reload()} hasData={!!o} rows={4}>
          {o && (
            <OverviewDetails
              o={o}
              today={today}
              weekFrom={weekFrom}
              todayRange={todayRange}
              platformDayOf={platformDayOf}
              link={link}
              mayOpen={mayOpen}
              onGo={(href) => router.push(href)}
            />
          )}
        </AdminAsync>
      )}
    </AdminPage>
  );
}

// ═════════════════════════════════════════════════════════════════════
//  تفاصيلُ اليوم: اليوم مقابل أمس · أسبابُ الفشل · آخرُ الأحداث · المنصّة بالأرقام · الزوّار
// ═════════════════════════════════════════════════════════════════════

function OverviewDetails({
  o,
  today,
  weekFrom,
  todayRange,
  platformDayOf,
  link,
  mayOpen,
  onGo,
}: {
  o: Overview;
  today: string;
  weekFrom: string;
  todayRange: string;
  platformDayOf: (iso: string) => string;
  link: (href: string) => string | undefined;
  mayOpen: (href: string) => boolean;
  onGo: (href: string) => void;
}) {
  const t = o.today_stats;
  const y = o.yesterday_stats;
  const ys = o.yesterday_same_time;
  const historyLink = link("/dashboard/history");
  const usersLink = link("/dashboard/users");

  // ── اليوم مقابل أمس ──
  const dayRows: { key: keyof Day; label: string; icon: IconT; f?: (x: number) => string; href?: string }[] = [
    { key: "orders", label: H.rowOrders, icon: IconOrder, href: historyLink && `/dashboard/history?${todayRange}` },
    { key: "delivered", label: H.rowDelivered, icon: IconSuccess, href: historyLink && `/dashboard/history?status=delivered&${todayRange}` },
    { key: "failed", label: H.rowFailed, icon: IconWarning, href: historyLink && `/dashboard/history?${todayRange}` },
    { key: "avg_delivery_min", label: H.rowAvgDelivery, icon: IconHourglass, f: (x) => H.minutes.replace("{n}", fmtNum(x)) },
    { key: "new_customers", label: H.rowNewCustomers, icon: IconUsers, href: usersLink && "/dashboard/users?role=customer" },
    { key: "avg_platform_rating", label: H.rowPlatformRating, icon: IconStar, f: (x) => H.stars.replace("{n}", fmtNum(x)) },
    { key: "avg_driver_rating", label: H.rowDriverRating, icon: IconStar, f: (x) => H.stars.replace("{n}", fmtNum(x)) },
  ];
  // **ومتوسّطٌ بلا مادّةٍ ليس مجهولاً** — لا تسليمَ ولا تقييمَ اليوم: شرطةٌ هادئة.
  const dayCell = (d: Day | undefined, r: (typeof dayRows)[number]) => {
    if (!d) return H.unknown;
    const v = d[r.key];
    if (v === null && r.key.startsWith("avg_") && known(d.orders)) return "—";
    return show(v, r.f);
  };
  const dayItems: SummaryItem[] = dayRows.map((r) => {
    const tv = t[r.key];
    return {
      key: r.key,
      label: r.label,
      value: dayCell(t, r),
      icon: r.icon,
      tone: known(tv) && tv === 0 ? "muted" : "neutral",
      // **أمسِ لنفس الساعة** — اليومُ لم ينتهِ، ومقارنتُه بيومٍ كاملٍ تقول «تراجع» كلَّ صباح.
      meta: ys ? KH.trend.replace("{v}", dayCell(ys, r)) : H.yesterdayFull.replace("{v}", dayCell(y, r)),
      href: r.href,
    };
  });

  // ── المنصّة بالأرقام ──
  const pill = (c: Count) =>
    `${H.newToday.replace("{n}", fmtNum(c.today))}${m.common.nameSeparator}${H.newWeek.replace("{n}", fmtNum(c.week))}`;
  const groupItems: SummaryItem[] = o.platform.groups.map((g) => ({
    key: g.key,
    label: (H.group as Record<string, string>)[g.key] ?? g.key,
    value: fmtNum(g.total),
    icon: GROUP_ICON[g.key] ?? IconGrid,
    tone: g.total === 0 ? "muted" : "neutral",
    meta: pill(g),
    href: GROUP_HREF[g.key] ? link(GROUP_HREF[g.key]!) : undefined,
  }));
  const staffItems: SummaryItem[] = o.platform.staff.map((g) => ({
    key: g.key,
    label: roleLabelByCode(g.key),
    value: fmtNum(g.total),
    icon: IconRoles,
    tone: g.total === 0 ? "muted" : "neutral",
    meta: pill(g),
    href: usersLink && `/dashboard/users?role=${g.key}`,
  }));
  const p = o.platform;
  const accountItems: SummaryItem[] = [
    { key: "ordered", label: H.customersOrderedWeek, value: show(p.customers_ordered_week), icon: IconOrder, tone: zeroMuted(p.customers_ordered_week), href: weekFrom ? historyLink && `/dashboard/history?from=${weekFrom}&to=${today}` : undefined },
    { key: "dActive", label: H.driversActive, value: show(p.drivers_active), icon: IconDriver, tone: zeroMuted(p.drivers_active), href: usersLink && "/dashboard/users?role=driver&status=active" },
    { key: "dSusp", label: H.driversSuspended, value: show(p.drivers_suspended), icon: IconDriver, tone: known(p.drivers_suspended) && p.drivers_suspended > 0 ? "attention" : zeroMuted(p.drivers_suspended), href: usersLink && "/dashboard/users?role=driver&status=suspended" },
    { key: "dBlock", label: H.driversBlocked, value: show(p.drivers_blocked), icon: IconDriver, tone: known(p.drivers_blocked) && p.drivers_blocked > 0 ? "danger" : zeroMuted(p.drivers_blocked), href: usersLink && "/dashboard/users?role=driver&status=blocked" },
    // **وأعدادُ المتاجر رقمٌ بلا رابط** — لا قائمةَ فيها العددُ نفسُه.
    { key: "sActive", label: H.storesActive, value: show(p.stores_active), icon: IconStore, tone: zeroMuted(p.stores_active) },
    { key: "sNot", label: H.storesNotActive, value: show(p.stores_not_active), icon: IconStore, tone: zeroMuted(p.stores_not_active) },
  ];

  // ── آخرُ الأحداث ──
  const events: StoryEvent[] = o.events.map((e) => {
    const raw = eventHref(e);
    const href = raw && mayOpen(raw) ? raw : null;
    const what = eventWhat(e);
    return {
      id: String(e.id),
      at: e.created_at,
      title: href ? (
        <Link href={href} className="hover:underline">
          {what}
        </Link>
      ) : (
        what
      ),
      actor: e.actor_name || undefined,
      // **والحدثُ القديمُ بتاريخه** — «2:05 م» وحدَها تبدو اليوم وهي من أسبوع.
      note: platformDayOf(e.created_at) === today ? undefined : fmtDateTime(e.created_at),
    };
  });

  return (
    <>
      <Block title={H.today}>
        <SummaryStrip items={dayItems.slice(0, 4)} max={4} />
        <SummaryStrip items={dayItems.slice(4)} max={4} />
      </Block>

      <div className="grid grid-cols-1 gap-4 xl:grid-cols-2">
        {/* **أسبابُ الفشل تُضغط** — كلُّ سطرٍ يفتح السجلَّ على اليوم وبحالته، والفشلُ بلا سببٍ يُعلَّم. */}
        <AdminCard title={H.failuresTitle} padded={false}>
          {o.missing.includes("failures") ? (
            <p className="px-4 pb-4 text-sm text-ink-muted lg:px-5">{H.unknown}</p>
          ) : o.failures_today.length === 0 ? (
            <p className={`flex items-center gap-2 px-4 pb-4 text-sm font-bold lg:px-5 ${TONE_TEXT.good}`}>
              <IconSuccess size={16} />
              {H.noFailures}
            </p>
          ) : (
            <ul className="admin-rows">
              {o.failures_today.map((f) => {
                const href = historyLink && `/dashboard/history?status=${f.status}&${todayRange}`;
                const noReason = !f.reason;
                return (
                  <Row
                    key={f.status + f.reason}
                    title={
                      <>
                        {STATUS[f.status] ?? f.status}
                        {m.common.nameSeparator}
                        {noReason ? H.noReason : (FAIL[f.reason] ?? f.reason)}
                      </>
                    }
                    meta={noReason ? <span className={`font-bold ${TONE_TEXT.attention}`}>{H.noReasonWarn}</span> : undefined}
                    trailing={<span className={`font-bold tabular-nums ${TONE_TEXT.bad}`}>{fmtNum(f.count)}</span>}
                    onOpen={href ? () => onGo(href) : undefined}
                  />
                );
              })}
            </ul>
          )}
        </AdminCard>

        <AdminCard title={H.events}>
          {o.missing.includes("events") ? (
            <p className="text-sm text-ink-muted">{H.unknown}</p>
          ) : o.events.length === 0 ? (
            <p className="text-sm text-ink-muted">{H.noEvents}</p>
          ) : (
            <StoryTimeline events={events} />
          )}
        </AdminCard>
      </div>

      {!o.missing.includes("platform") && (
        <Block title={H.platform}>
          <SummaryStrip items={groupItems.slice(0, 4)} max={4} />
          <SummaryStrip items={groupItems.slice(4)} max={4} />
          {staffItems.length > 0 && (
            <>
              <h3 className="text-sm font-bold text-ink-muted">{H.staffTitle}</h3>
              <SummaryStrip items={staffItems.slice(0, 4)} max={4} />
              {staffItems.length > 4 && <SummaryStrip items={staffItems.slice(4, 8)} max={4} />}
            </>
          )}
          <h3 className="text-sm font-bold text-ink-muted">{H.accountsTitle}</h3>
          <SummaryStrip items={accountItems.slice(0, 3)} max={3} />
          <SummaryStrip items={accountItems.slice(3)} max={3} />
        </Block>
      )}

      {/* ── زوّارُ الموقع والتحميلات (طلبُ المالك ٢٠٢٦-١٠-٠٧) ── */}
      <SiteStatsSection />
    </>
  );
}
