"use client";

/**
 * سجلّ الأحداث.
 *
 * كان `audit_log` يُكتب ولا يُقرأ إلا في نشاط مستخدمٍ بعينه. **وسجلٌّ لا يُقرأ
 * ليس سجلاً** — هو تكلفةُ كتابةٍ بلا فائدةِ قراءة.
 *
 * # بعد فحص ٢٠٢٦-١٠-٠٤ (قراراتُ المالك الستّة)
 *
 * - **يراه كلُّ من ملك `audit.read`**، والمبالغُ تُحذف في الخادم عمّن لا يملك
 *   قراءةَ المال — والشاشةُ تقول «المبلغ مخفي» لا «لا مبلغ».
 * - **المجموعةُ الأولى «الأفعال الحساسة»**، والدخولُ في مجموعته.
 * - **بحثٌ وفلترُ شخصٍ وفعلٍ وتاريخٍ بأزرار سريعة**، وتصديرٌ يُكتب في السجلّ.
 *
 * # التصميمُ الاحترافيّ (تدقيقُ ٢٠٢٦-١٠-٢٧)
 *
 * - رأسٌ موحّد (`PageHero`) وأربعُ خاناتٍ تُضغط: حسّاسُ اليوم · مالُ اليوم ·
 *   الإعداداتُ · كلُّ اليوم — **والعددُ من الخادم بيوم المنصّة** (من ساعة فتح الدوام).
 * - الفلاترُ في بطاقةٍ واحدة: حبّاتُ المجموعات المهمّة و«المزيد»، وحبّاتُ التاريخ
 *   و«تاريخ مخصص»، **وشريحةٌ ظاهرةٌ لأيّ تقييدٍ جاء من رابط** (إعدادٌ أو شخص) تُشال بضغطة.
 * - **بطاقةٌ لكلّ حدث** مقسومةٌ بعناوين الأيّام — لا جدولَ ولا مبدّلَ عرض.
 * - **النافذةُ عربيّةٌ كلُّها**: لا مفتاحَ إنكليزيّ ولا معرّفَ طويل ولا `true`؛ والمبلغُ
 *   مرّةً واحدةً باتّجاهه، والمعلوماتُ التقنيّةُ مطويّة.
 * - **الرابطُ لمن يُفتح له** — وغيرُه يرى الهدفَ نصّاً.
 */

import { useCallback, useEffect, useMemo, useState, type ComponentType, type ReactNode } from "react";
import {
  getMessages, defaultLocale, fmtDateTime, fmtNum, fmtMoney, fmtTime, fmtLongDate, errorText, damascusDay,
} from "@rahalgo/i18n";
import {
  Alert, Badge, Button, Input, Checkbox, Chips, LoadingState,
  Pagination, Select, Sheet,
  IconShieldCheck, IconSettings, IconUser, IconStatus, IconWallet, IconClose, IconSearch,
  IconDate, IconOverview, IconLock, IconWarning,
} from "@rahalgo/ui";
import Link from "next/link";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useCanCall, useCanOpen } from "@/lib/policy";
import { roleLabelByCode, classifyRole } from "@/lib/rolemeta";
import { PageHero } from "@/components/admin/PageHero";
import { Tile } from "@/components/admin/DashParts";

const m = getMessages(defaultLocale);
const A = m.admin.audit;
/** **واسما التاريخين من معجم الكشف** — لا يُترجَمان مرّتين. */
const ST = m.shared.statement;

interface Entry {
  id: number;
  actor_id: string | null;
  actor_name: string | null;
  actor_phone: string | null;
  actor_roles: string[] | null;
  action: string;
  entity: string;
  entity_id: string | null;
  target_label: string;
  target_number: number | null;
  details: Record<string, unknown> | null;
  ip: string | null;
  user_agent: string;
  created_at: string;
  redacted: boolean;
}

interface Page {
  entries: Entry[];
  total: number;
  total_capped: boolean;
  per_page: number;
  money_visible: boolean;
  /** **زرُّ التصدير لمدير المنصّة ومالكها وحدَهما** (`audit.export` — قرارُ المالك 2026-10-04). */
  can_export: boolean;
  /** **يومُ المنصّة من الخادم** — يبدأ بساعة فتح الدوام (`platform.day_start_hour`). */
  today?: string;
  yesterday?: string;
  week_from?: string;
  day_start_hour?: number;
  today_counts?: { sensitive: number; finance: number; settings: number; all: number };
}

/** **المجموعاتُ بترتيب الخادم** (`auditGroups`) — والأولى الافتراضيّة. */
const GROUPS = [
  "sensitive", "all", "login", "finance", "ops", "admin", "order", "driver",
  "merchant", "customer", "user", "menu", "catalog", "platform", "geo",
] as const;
type Group = (typeof GROUPS)[number];
/** **المهمّةُ ظاهرةٌ دائماً، والباقي تحت «المزيد».** */
const MAIN_GROUPS: Group[] = ["sensitive", "finance", "ops", "admin", "login", "all"];
const MORE_GROUPS: Group[] = GROUPS.filter((g) => !MAIN_GROUPS.includes(g));

/**
 * بادئةُ الفعل ← مجموعتُه — لتجميع قائمة «نوع الفعل».
 * **كتصنيف الخادم** (`audit_catalog.go`): السوقُ مع الكتالوج والتوسّعُ مع المناطق،
 * و`system` مع إدارة المنصّة — لا «الكل».
 */
const PREFIX_GROUP: Record<string, Group> = {
  auth: "login", finance: "finance", ops: "ops", admin: "admin", order: "order",
  driver: "driver", merchant: "merchant", customer: "customer", user: "user",
  menu: "menu", catalog: "catalog", market: "catalog", platform: "platform", coverage: "geo",
  coverage_request: "geo", operational_area: "geo", branch: "geo", expansion: "geo",
  system: "admin",
};

const ACTIONS = A.actions as Record<string, string>;
const GROUP_LABEL = A.groups as Record<Group, string>;
const KEY_LABEL = A.keys as Record<string, string>;
const PARTY = A.parties as Record<string, string>;
const DISPUTE_ACTION = A.disputeActions as Record<string, string>;
const DECISION = A.decisions as Record<string, string>;
const T = A.targets;

const actionText = (a: string) => ACTIONS[a] ?? a;

/**
 * **اسمُ الحالة بالعربيّة — من أيّ معجمٍ كانت.**
 *
 * السجلُّ يجمع أحداثَ الطلبات والسحوبات والشكاوى، **ولكلٍّ معجمُ حالاتٍ
 * خاصّ**. فيُسأل الأقربُ فالأقرب، **ويبقى الرمزُ آخرَ ملاذٍ لا أوّلَ عرض.**
 */
const statusText = (s: string) =>
  (m.orders.status as Record<string, string>)[s] ??
  (m.shared.payout.status as Record<string, string>)[s] ??
  (m.admin.tickets.status as Record<string, string>)[s] ??
  s;

/** نوعُ حركةِ المحفظة بالعربيّة — `topup` تُقرأ «شحن رصيد». */
const kindText = (k: string) =>
  (m.shared.txKinds as Record<string, string>)[k] ??
  // **ونوعُ الطارئ أيضاً** — `driver.emergency` يكتب `kind: accident` (فحصُ المتصفّح ٢٠٢٦-١٠-٠٥).
  (m.admin.emergencyRoom.kinds as Record<string, string>)[k] ??
  // **ونوعُ التعويض** — السجلُّ الموحّد يكتب `kind: complaint` لتعويض صاحب شكوى.
  (m.admin.compensations.kinds as Record<string, string>)[k] ??
  k;

/** **اسمٌ قد يكون مفتاحَ ترجمة** — حذفُ الأدوار (`0270`) يكتب `name: roles.ops`. */
const nameText = (n: string) => (n.startsWith("roles.") ? roleLabelByCode(n.slice(6)) : n);

/** **سببُ الشكوى رمزٌ** (`customer.complaint_opened` يكتب `reason: late`) — وغيرُه نصٌّ حرّ. */
const reasonText = (r: string) => (m.site.complaint.reasons as Record<string, string>)[r] ?? r;

/** **الطرفُ أو الوجهةُ بالعربيّة** — `to: platform` و`party: sales` (تدقيقُ ٢٠٢٦-١٠-٢٧). */
const partyText = (p: string) => {
  if (PARTY[p]) return PARTY[p];
  const role = roleLabelByCode(p);
  return role !== p ? role : statusText(p);
};

/** **رمزٌ لم يُترجَم** — حروفٌ لاتينيّةٌ صغيرةٌ وشَرطاتٌ (ومنها المعرّفاتُ الطويلة). لا يُعرض. */
const isCode = (s: string) => /^[a-z0-9][a-z0-9_.:/-]*$/.test(s);
const ISO_TIME = /^\d{4}-\d{2}-\d{2}T\d{2}:\d{2}/;
const ISO_DAY = /^\d{4}-\d{2}-\d{2}$/;

/** **مفاتيحُ مالُها مال** — تُعرض بالعملة لا رقماً عارياً. */
const MONEY_KEYS = new Set([
  "amount", "compensation", "compensation_proposed", "max_fee", "driver_share", "rep_share",
  "delta", "difference", "balance", "new_fee", "old_fee", "new_goods", "old_goods", "new_total",
  "old_total", "total", "paid", "rep_debt", "rep_reversed", "held_after", "driver_held_after",
  "held_forfeited", "expected", "counted", "absorbed", "waived", "threshold", "limit",
]);
const PARTY_KEYS = new Set(["to", "from", "party", "party_kind", "by", "actor_role", "fault", "paid_from"]);
const STATUS_KEYS = new Set(["status", "status_was", "from_status"]);
const PERCENT_KEYS = new Set(["percent", "commission_percent", "driver_percent"]);
/** **ما يُعرض في مكانه الخاصّ** — لا يتكرّر في «التفاصيل». */
const OWN_ROW = new Set(["before", "after", "note", "amount", "debit"]);

/**
 * **قيمةُ حقلٍ من التفاصيل بالعربيّة — أو لا شيء.**
 *
 * (تدقيقُ ٢٠٢٦-١٠-٢٧: كانت النافذةُ تكتب `request_id` و`true` و`platform` وتاريخاً بصيغة
 * الآلة.) **فالحقلُ بلا اسمٍ عربيّ لا يُعرض، والرمزُ الذي لم يُترجَم لا يُعرض** — والتصديرُ
 * يحمل التفاصيلَ كاملةً لمن احتاجها.
 */
function detailText(k: string, v: unknown): string | null {
  if (v === null || v === undefined || v === "") return null;
  if (typeof v === "boolean") return v ? A.yes : A.no;
  if (typeof v === "number") {
    if (MONEY_KEYS.has(k)) return fmtMoney(v);
    if (PERCENT_KEYS.has(k)) return `${fmtNum(v)}%`;
    return fmtNum(v);
  }
  if (typeof v !== "string") return null;
  if (ISO_TIME.test(v)) return fmtDateTime(v);
  if (ISO_DAY.test(v)) return fmtLongDate(`${v}T12:00:00Z`);
  let out: string = v;
  if (k === "kind") out = kindText(v);
  else if (k === "name") out = nameText(v);
  else if (k === "reason") out = reasonText(v);
  else if (k === "role") out = roleLabelByCode(v);
  else if (k === "action") out = DISPUTE_ACTION[v] ?? v;
  else if (k === "decision") out = DECISION[v] ?? v;
  else if (PARTY_KEYS.has(k)) out = partyText(v);
  else if (STATUS_KEYS.has(k)) out = statusText(v);
  return isCode(out) ? null : out;
}

/** **قيمةُ الإعداد كما يقرؤها إنسان** — لا `null` ولا علاماتُ اقتباس. */
const settingValue = (v: unknown): string => {
  if (v === null || v === undefined) return A.defaultValue;
  if (typeof v === "boolean") return v ? A.boolOn : A.boolOff;
  if (typeof v === "number") return fmtNum(v);
  if (typeof v === "string") return v === "" ? A.emptyValue : v;
  return JSON.stringify(v);
};

/** الأفعال المالية تُبرَز: هي ما يُبحث عنه حين يُبحث في هذا السجلّ. */
const toneOf = (action: string): "danger" | "warning" | "neutral" =>
  action.startsWith("finance.") ? "danger"
    : action.startsWith("ops.") || action.startsWith("admin.") ? "warning"
      : "neutral";
const CIRCLE: Record<ReturnType<typeof toneOf>, string> = {
  danger: "bg-danger-solid text-on-solid",
  warning: "bg-warning-solid text-on-solid",
  neutral: "bg-field text-ink-muted",
};

/** **الهدفُ مقروءاً ورابطاً** (المشكلةُ الرابعة: «تغيير حالة طلب» بلا طلب). */
function targetOf(e: Entry): { text: string; href?: string } | null {
  if (!e.entity && !e.entity_id) return null;
  const label = e.target_label;
  const id = e.entity_id ?? "";
  switch (e.entity) {
    case "order":
      return e.target_number
        ? { text: T.order.replace("{n}", String(e.target_number)), href: `/dashboard/history?q=${e.target_number}` }
        : { text: T.order.replace("{n}", "—") };
    case "ticket":
      // **التذكرةُ نفسُها لا القائمة** — `?t=` تفتحها مباشرة (تدقيقُ ٢٠٢٦-١٠-٢٧).
      return {
        text: T.ticket.replace("{n}", e.target_number ? String(e.target_number) : "—"),
        href: id ? `/dashboard/tickets?t=${encodeURIComponent(id)}` : undefined,
      };
    case "merchant":
      return { text: label || T.merchant, href: id ? `/dashboard/merchants/${id}` : undefined };
    case "user":
    case "wallet":
      return { text: label || (e.entity === "user" ? T.user : T.wallet), href: id && e.entity === "user" ? `/dashboard/users/${id}` : undefined };
    case "lead":
      return {
        text: label || T.lead,
        href: label ? `/dashboard/leads?status=all&q=${encodeURIComponent(label)}` : "/dashboard/leads?status=all",
      };
    case "setting":
      return {
        text: (m.admin.settings.keys as Record<string, { label: string }>)[id]?.label ?? T.setting,
        href: "/dashboard/settings",
      };
    case "settings":
    case "platform":
      return { text: T.platform, href: "/dashboard/settings" };
    case "role":
      return { text: `${T.role}: ${roleLabelByCode(id)}` };
    case "dispute": {
      const base = label ? T.disputeWith.replace("{name}", label) : T.dispute;
      return {
        text: e.target_number ? `${base}${m.common.nameSeparator}${T.order.replace("{n}", String(e.target_number))}` : base,
        href: "/dashboard/tickets?tab=disputes",
      };
    }
    case "menu_item":
      return { text: label || T.menuItem };
    case "menu_section":
      return { text: label || T.menuSection };
    case "free_delivery_campaign":
      return { text: T.freeDelivery, href: "/dashboard/promos" };
    case "offer":
      return { text: label || T.offer, href: "/dashboard/promos" };
    case "campaign":
      return { text: label || T.campaign, href: "/dashboard/promos" };
    case "banner":
      return { text: label || T.banner };
    case "cashbox":
    case "office_cash_close":
    case "office_cash_entry":
    case "office_cash_shortfall":
      return { text: T.cashbox, href: "/dashboard/treasury?tab=cashbox" };
    case "treasury_withdrawal":
      return { text: T.withdrawal, href: "/dashboard/treasury" };
    case "emergency":
      return { text: T.emergency, href: "/dashboard/emergencies" };
    case "zone":
    case "delivery_zone":
      return { text: label || T.zone, href: "/dashboard/opsmap" };
    case "coverage_request":
      return { text: T.coverage, href: "/dashboard/opsmap" };
    case "section":
      return { text: label || T.section, href: "/dashboard/sections" };
    case "category":
      return { text: label || T.category, href: "/dashboard/sections" };
    case "expense":
      return { text: T.expense, href: "/dashboard/expenses" };
    case "expense_category":
      return { text: label || T.expenseCategory, href: "/dashboard/expenses" };
    case "expense_request":
      return { text: T.expenseRequest, href: "/dashboard/expenses" };
    case "payout":
      return { text: T.payout, href: "/dashboard/payouts" };
    case "obligation":
      return { text: T.obligation, href: "/dashboard/obligations" };
    case "notification":
      return { text: T.notification };
    case "incentive":
    case "incentives":
      return { text: T.incentive, href: "/dashboard/incentives" };
    default:
      return label ? { text: label } : null;
  }
}

/** **أيُّ صفحةٍ يفتحها صاحبُ الجلسة** — بقدرات بنود القائمة الجانبيّة نفسِها (`layout.tsx`). */
const PAGE_CAPS: [string, string[]][] = [
  ["/dashboard/history", ["orders.read"]],
  ["/dashboard/opsmap", ["orders.read"]],
  ["/dashboard/emergencies", ["emergencies.manage"]],
  ["/dashboard/treasury", ["finance.read"]],
  ["/dashboard/payouts", ["finance.read"]],
  ["/dashboard/obligations", ["finance.read"]],
  ["/dashboard/expenses", ["finance.read"]],
  ["/dashboard/incentives", ["finance.read"]],
  ["/dashboard/leads", ["merchants.verify"]],
  ["/dashboard/sections", ["market.manage"]],
  ["/dashboard/promos", ["offers.manage"]],
];

/**
 * **أيُفتح الرابطُ لصاحب الجلسة؟** (تدقيقُ ٢٠٢٦-١٠-٢٧: «الثقة والسلامة» كانت ترى رابطَ
 * المتجر والتذكرة فتفتح صفحةً تُردّ ممنوعة.) **وما لا يُفتح يُرسم نصّاً.**
 */
function useMayOpen(): (href: string) => boolean {
  const { can } = useAuth();
  const canCall = useCanCall();
  const canOpen = useCanOpen();
  return useCallback(
    (href: string) => {
      const [path = "", qs = ""] = href.split("?");
      const sp = new URLSearchParams(qs);
      if (/^\/dashboard\/users\/[^/]+$/.test(path)) return canOpen.user;
      if (/^\/dashboard\/merchants\/[^/]+$/.test(path)) return canOpen.merchant;
      if (path === "/dashboard/settings") return canOpen.settings;
      if (path === "/dashboard/tickets") {
        if (sp.get("tab") === "disputes") return canCall("GET", "/disputes");
        return canCall("GET", "/tickets");
      }
      const hit = PAGE_CAPS.find(([p]) => p === path);
      return hit ? hit[1].some((c) => can(c)) : false;
    },
    [can, canCall, canOpen],
  );
}

/** **ملخّصُ التفاصيل بلغةٍ لا بـJSON** — «كم» و«لمن»، لا ترميزٌ يُفكّ. والمبلغُ في مكانه. */
function summaryOf(e: Entry): string {
  const d = e.details;
  const bits: string[] = [];
  if (d) {
    if (typeof d.compensation === "number" && d.compensation > 0) bits.push(fmtMoney(d.compensation));
    for (const k of ["status", "to", "party", "action", "kind", "name", "title", "reason", "resolution"]) {
      const t = detailText(k, d[k]);
      if (t) bits.push(t);
    }
    if (typeof d.note === "string" && d.note) bits.push(d.note);
    // تغيير الإعداد: الفرق لا النتيجة — «صار ٧٠» بلا «كان ٥٠» يُثبت الفعل ولا يُظهر أثره
    if (d.before !== undefined || d.after !== undefined)
      bits.push(`${settingValue(d.before)} ← ${settingValue(d.after)}`);
  }
  if (e.redacted) bits.push(A.redacted);
  return bits.join(m.common.listSeparator);
}

/** **المبلغُ واتّجاهُه** — مرّةً واحدة: خصمٌ أو إضافةٌ حين يقولها الحدث (`debit`). */
function amountOf(e: Entry): { value: number; dir: "debit" | "credit" | null } | null {
  const d = e.details;
  if (!d || typeof d.amount !== "number") return null;
  return { value: d.amount, dir: d.debit === true ? "debit" : d.debit === false ? "credit" : null };
}

/** اسمُ الفاعل — **و«النظام» حين لا أحد.** */
const actorText = (e: Entry) => (e.actor_id ? e.actor_name ?? e.actor_phone ?? A.unknown : A.system);

/**
 * **أعلى دورٍ وحدَه** (تدقيقُ ٢٠٢٦-١٠-٢٧: «مدير، زبون، مالك المنصة») — المحميُّ فالمرتفعُ
 * فالوظيفة، **وصفةُ الحساب (زبون…) لا تُذكر لمن له وظيفة.**
 */
const ROLE_RANK: Record<string, number> = { protected: 0, elevated: 1, staff: 2, custom: 3, legacy: 4, account_type: 5 };
function topRole(e: Entry): string {
  const roles = [...(e.actor_roles ?? [])].sort(
    (a, b) => (ROLE_RANK[classifyRole(a)] ?? 9) - (ROLE_RANK[classifyRole(b)] ?? 9),
  );
  return roles[0] ? roleLabelByCode(roles[0]) : "";
}
const rolesText = (e: Entry) =>
  (e.actor_roles ?? []).map((r) => roleLabelByCode(r)).join(m.common.listSeparator);

/** **وصفُ الجهاز مختصراً** — لا نصُّ المتصفّح الطويل بالإنكليزيّة. */
function deviceText(ua: string): string {
  const D = A.devices;
  if (!ua) return A.unknown;
  if (/okhttp|dart|ktor|rahalgo/i.test(ua)) return D.app;
  if (/iphone/i.test(ua)) return D.iphone;
  if (/ipad/i.test(ua)) return D.ipad;
  if (/android/i.test(ua)) return D.android;
  if (/windows/i.test(ua)) return D.windows;
  if (/mac os|macintosh/i.test(ua)) return D.mac;
  if (/linux/i.test(ua)) return D.linux;
  return D.other;
}

/** **خانةُ CSV لا تُقرأ صيغة** — `=` و`+` و`-` و`@` في أوّلها تُبطَل. */
const QUOTE = String.fromCharCode(34);
const NEEDS_QUOTES = new RegExp(`[${QUOTE},\\n\\r]`);
function csvCell(v: string): string {
  let s = v ?? "";
  if (/^[=+\-@\t\r]/.test(s)) s = `'${s}`;
  if (NEEDS_QUOTES.test(s)) s = QUOTE + s.split(QUOTE).join(QUOTE + QUOTE) + QUOTE;
  return s;
}

function buildCsv(rows: Entry[]): string {
  const C = A.csv;
  const head = [C.time, C.actor, C.phone, C.roles, C.action, C.actionKey, C.target, C.details, C.ip, C.device];
  const lines = [head.map(csvCell).join(",")];
  for (const e of rows) {
    const t = targetOf(e);
    const amt = amountOf(e);
    const extra = e.details && Object.keys(e.details).length > 0 ? JSON.stringify(e.details) : "";
    const summary = [amt ? fmtMoney(amt.value) : "", summaryOf(e), extra].filter(Boolean).join(" | ");
    lines.push([
      fmtDateTime(e.created_at), actorText(e), e.actor_phone ?? "", rolesText(e),
      actionText(e.action), e.action, t?.text ?? "", summary, e.ip ?? "", e.user_agent ?? "",
    ].map(csvCell).join(","));
  }
  return "﻿" + lines.join("\r\n");
}

/** **رابطُ الهدف لمن يُفتح له — ونصٌّ لغيره.** والضغطُ عليه لا يفتح البطاقة. */
function TargetLink({ t, mayOpen, className = "" }: {
  t: { text: string; href?: string };
  mayOpen: (href: string) => boolean;
  className?: string;
}) {
  if (t.href && mayOpen(t.href)) {
    return (
      <Link href={t.href} onClick={(ev) => ev.stopPropagation()} className={`text-primary hover:underline ${className}`}>
        {t.text}
      </Link>
    );
  }
  return <span className={className}>{t.text}</span>;
}

/** **أسماءُ المعرّفات المعروفة** — تصير رابطاً لصاحبها لا رقماً طويلاً. */
const ID_LINKS: Record<string, "user" | "merchant" | "order"> = {
  merchant_id: "merchant", previous_merchant_id: "merchant",
  customer_id: "user", driver_id: "user", rep_id: "user", beneficiary_id: "user",
  complainant_id: "user", from_rep: "user", to_rep: "user", owner: "user",
  order_id: "order",
};

/** **اللوحةُ الجانبيّة** — كلُّ ما في الحدث بالعربيّة: قبل وبعد، المبلغُ واتّجاهُه، الملاحظةُ كاملة. */
function DetailPanel({
  e, onClose, onActor, mayOpen,
}: {
  e: Entry | null;
  onClose: () => void;
  onActor: (id: string) => void;
  mayOpen: (href: string) => boolean;
}) {
  if (!e) return null;
  const d = e.details ?? {};
  const t = targetOf(e);
  const amt = amountOf(e);
  const rest = Object.entries(d)
    .filter(([k]) => !OWN_ROW.has(k) && KEY_LABEL[k])
    .map(([k, v]) => [k, detailText(k, v)] as const)
    .filter((x): x is readonly [string, string] => x[1] !== null);
  // **والمعرّفُ رابطٌ لصاحبه** — `party_id` بحسب نوع الطرف.
  const links: { label: string; href: string }[] = [];
  for (const [k, v] of Object.entries(d)) {
    if (typeof v !== "string" || !v) continue;
    let kind = ID_LINKS[k];
    if (k === "party_id") kind = d.party === "merchant" || d.party_kind === "merchant" ? "merchant" : "user";
    if (!kind) continue;
    const href = kind === "merchant" ? `/dashboard/merchants/${v}` : kind === "user" ? `/dashboard/users/${v}` : `/dashboard/orders?id=${v}`;
    const label = kind === "merchant" ? A.openStore : kind === "user" ? A.openAccount : A.openOrder;
    if (!links.some((l) => l.href === href)) links.push({ label: `${label}${KEY_LABEL[k] ? ` (${KEY_LABEL[k]})` : ""}`, href });
  }
  const canOrder = mayOpen("/dashboard/history");
  const shownLinks = links.filter((l) => (l.href.startsWith("/dashboard/orders") ? canOrder : mayOpen(l.href)));
  const row = (label: string, value: ReactNode) => (
    <div className="flex flex-col gap-0.5 border-b border-line-soft py-2 last:border-b-0">
      <span className="text-xs text-ink-muted">{label}</span>
      <span className="break-words text-sm">{value}</span>
    </div>
  );
  const hasNote = typeof d.note === "string" && d.note !== "";
  return (
    <Sheet open onClose={onClose} title={A.panelTitle}
      footer={e.actor_id ? (
        <Button className="w-full" variant="secondary" onClick={() => onActor(e.actor_id as string)}>
          {A.filterByActor}
        </Button>
      ) : undefined}>
      <div className="flex flex-col">
        <div className="mb-2 flex items-center gap-3 rounded-control bg-field px-3 py-2.5">
          <span className={`inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full ${CIRCLE[toneOf(e.action)]}`}>
            {e.actor_id ? <IconUser size={18} /> : <IconSettings size={18} />}
          </span>
          <span className="min-w-0">
            <span className="block font-bold">{actionText(e.action)}</span>
            <span className="block text-xs text-ink-muted" dir="ltr">{fmtDateTime(e.created_at)}</span>
          </span>
        </div>
        {amt && (
          <div className="mb-2 rounded-control border border-line-soft px-3 py-2.5">
            <span className="block text-2xs text-ink-muted">
              {A.amount}{amt.dir ? ` · ${amt.dir === "debit" ? A.debit : A.credit}` : ""}
            </span>
            <span className={`figure block ${amt.dir === "debit" ? "text-danger" : amt.dir === "credit" ? "text-success" : ""}`}>
              {amt.dir === "debit" ? "− " : amt.dir === "credit" ? "+ " : ""}{fmtMoney(amt.value)}
            </span>
          </div>
        )}
        {row(A.actor, (
          <span>
            {actorText(e)}
            {topRole(e) && <span className="text-ink-muted">{m.common.nameSeparator}{topRole(e)}</span>}
          </span>
        ))}
        {t && row(A.target, <TargetLink t={t} mayOpen={mayOpen} className="underline" />)}
        {(d.before !== undefined || d.after !== undefined) && (
          <>
            {row(A.before, settingValue(d.before))}
            {row(A.after, settingValue(d.after))}
          </>
        )}
        {hasNote && row(A.note, <span className="whitespace-pre-wrap">{String(d.note)}</span>)}
        {e.redacted && row(A.details, A.redacted)}
        {rest.map(([k, v]) => (
          <div key={k}>{row(KEY_LABEL[k] ?? "", <span dir="auto">{v}</span>)}</div>
        ))}
        {rest.length === 0 && !amt && !e.redacted && !hasNote && d.before === undefined && d.after === undefined
          && row(A.details, A.noDetails)}
        {shownLinks.length > 0 && (
          <div className="mt-2 grid grid-cols-1 gap-2 sm:grid-cols-2">
            {shownLinks.map((l) => (
              <Link key={l.href} href={l.href}
                className="rounded-control border border-line-soft px-3 py-2 text-center text-sm font-medium text-primary hover:bg-row-hover">
                {l.label}
              </Link>
            ))}
          </div>
        )}
        <details className="mt-3 rounded-control bg-field px-3 py-2">
          <summary className="cursor-pointer text-xs font-bold text-ink-muted">{A.tech}</summary>
          <div className="mt-1 flex flex-col">
            {row(A.deviceShort, deviceText(e.user_agent))}
            {row(A.ip, <span dir="ltr">{e.ip || A.unknown}</span>)}
            {e.user_agent && row(A.deviceFull, <span dir="ltr" className="break-all text-xs">{e.user_agent}</span>)}
          </div>
        </details>
      </div>
    </Sheet>
  );
}

type Range = "all" | "today" | "yesterday" | "last7" | "custom";

/** **يومُ المنصّة لحدث** — بطرح ساعة فتح الدوام، كما يحسبه الخادم. */
const platformDay = (iso: string, startHour: number) => damascusDay(new Date(iso).getTime() - startHour * 3_600_000);

export default function AuditPage() {
  const mayOpen = useMayOpen();
  const [group, setGroup] = useState<Group>("sensitive");
  const [showMore, setShowMore] = useState(false);
  const [q, setQ] = useState("");
  const [actor, setActor] = useState("");
  const [action, setAction] = useState("");
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  const [customRange, setCustomRange] = useState(false);
  const [showExtra, setShowExtra] = useState(false);
  /** **والتجديدُ مخفيٌّ افتراضاً** — تكتبه الساعةُ لا الإنسان. ويُعرض خيارُه في مجموعة الدخول. */
  const [withRefresh, setWithRefresh] = useState(false);
  const [page, setPage] = useState(1);
  /**
   * **رابطُ «السجل» من بطاقة الإعداد** (قرارُ المالك ٢٠٢٦-١٠-٠٤، الإعدادات البند ٩):
   * `?entity=setting&entity_id=<المفتاح>` يفتح تاريخَ ذلك الإعداد وحدَه —
   * **وشريحةٌ ظاهرةٌ تقوله وتُشال بضغطة** (تدقيقُ ٢٠٢٦-١٠-٢٧).
   */
  const [target, setTarget] = useState<{ entity: string; entityId: string }>({ entity: "", entityId: "" });
  useEffect(() => {
    const sp = new URLSearchParams(window.location.search);
    const entity = sp.get("entity") ?? "";
    const entityId = sp.get("entity_id") ?? "";
    if (entity || entityId) setTarget({ entity, entityId });
  }, []);
  const scoped = target.entity !== "" || target.entityId !== "";

  const [data, setData] = useState<Page | null>(null);
  const [loading, setLoading] = useState(true);
  /**
   * **وسجلٌّ فارغٌ على خطأٍ شهادةُ زور.** من بحث عن تعويضٍ صُرف ولم يجده يظنّ
   * أنّه لم يُصرف — فالخطأُ يُقال، **ولا دائرةَ تدور للأبد تحته** (المشكلةُ
   * الحادية عشرة).
   */
  const [error, setError] = useState("");
  const [open, setOpen] = useState<Entry | null>(null);
  const [actors, setActors] = useState<{ id: string; name: string }[]>([]);
  const [exporting, setExporting] = useState(false);
  const [exportNote, setExportNote] = useState("");

  /* **ملاحظةُ التصدير مؤقّتة** — تختفي وحدَها. */
  useEffect(() => {
    if (!exportNote) return;
    const t = setTimeout(() => setExportNote(""), 10_000);
    return () => clearTimeout(t);
  }, [exportNote]);

  const effGroup: Group = scoped ? "all" : group;
  const params = useCallback(() => {
    const qs = new URLSearchParams({ group: effGroup });
    if (target.entity) qs.set("entity", target.entity);
    if (target.entityId) qs.set("entity_id", target.entityId);
    if ((withRefresh && effGroup === "login") || action === "auth.refresh") qs.set("refresh", "true");
    if (from) qs.set("from", from);
    if (to) qs.set("to", to);
    if (actor) qs.set("actor", actor);
    if (action) qs.set("action", action);
    const text = q.trim();
    if (text) {
      qs.set("q", text);
      /* **والفعلُ يُبحث باسمه العربيّ** — الأسماءُ هنا لا في الخادم، فتُرسل
         الأفعالُ التي طابق اسمُها ما كُتب. */
      if (text.length >= 2) {
        const hits = Object.entries(ACTIONS)
          .filter(([, label]) => label.includes(text))
          .map(([k]) => k)
          .slice(0, 50);
        if (hits.length) qs.set("q_actions", hits.join(","));
      }
    }
    return qs;
  }, [effGroup, withRefresh, from, to, actor, action, q, target]);

  const load = useCallback(() => {
    setLoading(true);
    setError("");
    const qs = params();
    qs.set("limit", "50");
    qs.set("page", String(page));
    api<Page>(`/api/v1/admin/audit?${qs}`)
      .then((r) => setData(r))
      .catch((err) => setError(errorText(err)))
      .finally(() => setLoading(false));
  }, [params, page]);

  /* **والبحثُ يُرسَل بعد أن يكفّ الإصبع** — لا مع كلّ حرف. */
  useEffect(() => {
    const t = setTimeout(load, 250);
    return () => clearTimeout(t);
  }, [load]);

  /* **قائمةُ الأشخاص** — من فعل شيئاً في آخر تسعين يوماً. */
  useEffect(() => {
    api<{ actors: { id: string; name: string }[] }>("/api/v1/admin/audit/actors")
      .then((r) => setActors(r?.actors ?? []))
      // @empty-ok — **فلترُ الشخص اختياريّ**: إن تعذّرت قائمتُه بقي البحثُ بالاسم.
      .catch(() => setActors([]));
  }, []);

  const reset = <V,>(set: (v: V) => void) => (v: V) => {
    set(v);
    setPage(1);
  };

  /**
   * **مدىً جاهز — بيوم المنصّة من الخادم** (تدقيقُ ٢٠٢٦-١٠-٢٧): «اليوم» يبدأ بساعة فتح
   * الدوام كبقيّة اللوحة. وقبل وصول الجواب الأوّل يُقدَّر بيوم دمشق.
   */
  const startHour = data?.day_start_hour ?? 0;
  const today = data?.today ?? damascusDay(Date.now() - startHour * 3_600_000);
  const yesterday = data?.yesterday ?? damascusDay(Date.now() - startHour * 3_600_000 - 86_400_000);
  const weekFrom = data?.week_from ?? damascusDay(Date.now() - startHour * 3_600_000 - 6 * 86_400_000);
  const presets: Record<Exclude<Range, "all" | "custom">, [string, string]> = {
    today: [today, today],
    yesterday: [yesterday, yesterday],
    last7: [weekFrom, today],
  };
  const range: Range = customRange
    ? "custom"
    : !from && !to
      ? "all"
      : (Object.entries(presets).find(([, [f, t]]) => f === from && t === to)?.[0] as Range | undefined) ?? "custom";
  const pickRange = (r: Range) => {
    setPage(1);
    if (r === "custom") {
      setCustomRange(true);
      return;
    }
    setCustomRange(false);
    if (r === "all") {
      setFrom("");
      setTo("");
      return;
    }
    const [f, t] = presets[r];
    setFrom(f);
    setTo(t);
  };

  const filtered = q !== "" || actor !== "" || action !== "" || from !== "" || to !== "" || scoped;
  const clearAll = () => {
    setQ(""); setActor(""); setAction(""); setFrom(""); setTo(""); setCustomRange(false); setPage(1);
    clearTarget();
  };
  function clearTarget() {
    setTarget({ entity: "", entityId: "" });
    try {
      window.history.replaceState(null, "", window.location.pathname);
    } catch {
      // @empty-ok — **الرابطُ زينة**: التقييدُ انشال من الحالة على أيّ حال.
    }
    setPage(1);
  }

  /** **خانةُ رأس الصفحة تفتح قائمتَها** — فيطابق العددُ ما يُعرض. */
  type TileKey = "sensitive" | "finance" | "settings" | "all";
  const pickTile = (k: TileKey) => {
    clearTarget();
    setQ(""); setActor(""); setCustomRange(false);
    setFrom(today); setTo(today);
    setGroup(k === "sensitive" ? "sensitive" : k === "finance" ? "finance" : "all");
    setAction(k === "settings" ? "admin.setting_update" : "");
    setPage(1);
  };
  const tileOn = (k: TileKey) =>
    !scoped && from === today && to === today && !q && !actor &&
    (k === "settings"
      ? action === "admin.setting_update" && group === "all"
      : !action && group === (k === "sensitive" ? "sensitive" : k === "finance" ? "finance" : "all"));

  /** **قائمةُ الأفعال مجمّعةً بمجموعتها** — بأسمائها العربيّة. */
  const actionOptions = useMemo(() => {
    const by = new Map<Group, [string, string][]>();
    for (const [k, label] of Object.entries(ACTIONS)) {
      const g: Group = PREFIX_GROUP[k.split(".")[0] ?? ""] ?? "admin";
      if (!by.has(g)) by.set(g, []);
      by.get(g)!.push([k, label]);
    }
    return GROUPS.filter((g) => by.has(g)).map((g) => ({
      group: g,
      items: by.get(g)!.sort((a, b) => a[1].localeCompare(b[1], "ar")),
    }));
  }, []);

  async function exportCsv() {
    setExporting(true);
    setExportNote("");
    try {
      const r = await api<{ entries: Entry[]; truncated: boolean; cap: number }>(
        `/api/v1/admin/audit/export?${params()}`,
      );
      const blob = new Blob([buildCsv(r?.entries ?? [])], { type: "text/csv;charset=utf-8" });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `audit-${from || "all"}_${to || today}.csv`;
      a.click();
      URL.revokeObjectURL(url);
      if (r?.truncated) setExportNote(A.exportTruncated.replace("{n}", fmtNum(r.cap)));
    } catch (err) {
      setExportNote(`${A.exportFailed}: ${errorText(err)}`);
    }
    setExporting(false);
  }

  const list = data?.entries ?? [];
  /** **الأحداثُ بعناوين أيّامها** — اليوم · أمس · ثمّ التاريخ. */
  const byDay = useMemo(() => {
    const out: { day: string; items: Entry[] }[] = [];
    for (const e of list) {
      const day = platformDay(e.created_at, startHour);
      const last = out[out.length - 1];
      if (last && last.day === day) last.items.push(e);
      else out.push({ day, items: [e] });
    }
    return out;
  }, [list, startHour]);
  const dayTitle = (day: string) =>
    day === today ? A.today : day === yesterday ? A.yesterday : fmtLongDate(`${day}T12:00:00Z`);

  const scopeName = target.entity === "setting"
    ? (m.admin.settings.keys as Record<string, { label: string }>)[target.entityId]?.label ?? T.setting
    : (targetOf({
      id: 0, actor_id: null, actor_name: null, actor_phone: null, actor_roles: null, action: "",
      entity: target.entity, entity_id: target.entityId, target_label: "", target_number: null,
      details: null, ip: null, user_agent: "", created_at: "", redacted: false,
    })?.text ?? T.other);
  const actorName = actor === "system" ? A.personSystem : actors.find((a) => a.id === actor)?.name ?? A.unknown;
  const counts = data?.today_counts;
  const tiles: { k: TileKey; icon: ComponentType<{ size?: number; className?: string }>; tone: "danger" | "warning" | "accent" | "default" }[] = [
    { k: "sensitive", icon: IconShieldCheck, tone: "danger" },
    { k: "finance", icon: IconWallet, tone: "warning" },
    { k: "settings", icon: IconSettings, tone: "accent" },
    { k: "all", icon: IconOverview, tone: "default" },
  ];

  return (
    <div>
      <PageHero
        icon={IconShieldCheck}
        title={A.title}
        subtitle={A.subtitle}
        actions={data?.can_export ? (
          <Button variant="secondary" onClick={exportCsv} disabled={exporting}>
            {exporting ? A.exporting : A.export}
          </Button>
        ) : undefined}
      >
        <div className="grid grid-cols-2 gap-2 lg:grid-cols-4">
          {tiles.map((t) => {
            const n = counts?.[t.k] ?? 0;
            return (
              <div key={t.k} className={tileOn(t.k) ? "rounded-card ring-2 ring-accent" : ""}>
                <Tile
                  icon={t.icon}
                  label={A.tiles[t.k]}
                  value={counts ? fmtNum(n) : "—"}
                  tone={n > 0 ? t.tone : "muted"}
                  onClick={() => pickTile(t.k)}
                />
              </div>
            );
          })}
        </div>
        {data && !data.money_visible && (
          <p className="mt-2 flex items-center gap-1.5 text-xs text-ink-muted">
            <IconLock size={13} />
            {A.moneyHidden}
          </p>
        )}
        {exportNote && (
          <p className="mt-2 flex items-center gap-1.5 text-xs font-medium text-warning" role="status">
            <IconWarning size={13} />
            {exportNote}
          </p>
        )}
      </PageHero>

      {/* ── الفلاترُ في بطاقةٍ واحدة ── */}
      <section className="surface mb-4 space-y-3 p-4">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h2 className="heading-card">
            {A.filtersTitle}
            {data && (
              <span className="ms-2 text-sm font-normal text-ink-muted">
                {data.total_capped ? A.countCapped.replace("{n}", fmtNum(data.total)) : A.count.replace("{n}", fmtNum(data.total))}
              </span>
            )}
          </h2>
          {filtered && (
            <Button variant="ghost" onClick={clearAll}>{A.clear}</Button>
          )}
        </div>

        {/* **التقييدُ ظاهرٌ ويُشال** — رابطُ «السجل» من الإعدادات، أو شخصٌ بعينه. */}
        {(scoped || actor) && (
          <div className="flex flex-wrap gap-2">
            {scoped && (
              <button type="button" onClick={clearTarget} title={A.scopedClear}
                className="inline-flex items-center gap-2 rounded-full bg-accent-tint px-3 py-1.5 text-sm font-medium text-accent">
                {target.entity === "setting"
                  ? A.scopedSetting.replace("{name}", scopeName)
                  : A.scopedOther.replace("{name}", scopeName)}
                <IconClose size={14} />
              </button>
            )}
            {actor && (
              <button type="button" onClick={() => reset(setActor)("")} title={A.scopedClear}
                className="inline-flex items-center gap-2 rounded-full bg-primary-tint px-3 py-1.5 text-sm font-medium text-primary">
                {A.actorChip.replace("{name}", actorName)}
                <IconClose size={14} />
              </button>
            )}
          </div>
        )}

        {!scoped && (
          <div className="space-y-1.5">
            <span className="block text-xs font-bold text-ink-muted">{A.groupLabel}</span>
            <Chips
              wrap
              items={[
                ...MAIN_GROUPS.map((g) => ({ id: g, label: GROUP_LABEL[g] })),
                ...(showMore || MORE_GROUPS.includes(group) ? MORE_GROUPS.map((g) => ({ id: g, label: GROUP_LABEL[g] })) : []),
              ]}
              value={group}
              onChange={(g) => reset(setGroup)(g)}
            />
            {!MORE_GROUPS.includes(group) && (
              <Button variant="ghost" onClick={() => setShowMore((v) => !v)}>
                {showMore ? A.less : A.more}
              </Button>
            )}
          </div>
        )}

        <div className="space-y-1.5">
          <span className="block text-xs font-bold text-ink-muted">{A.dateLabel}</span>
          <Chips
            wrap
            items={[
              { id: "all" as Range, label: A.rangeAll },
              { id: "today" as Range, label: A.rangeToday },
              { id: "yesterday" as Range, label: A.rangeYesterday },
              { id: "last7" as Range, label: A.rangeLast7 },
              { id: "custom" as Range, label: A.rangeCustom },
            ]}
            value={range}
            onChange={pickRange}
          />
          {range === "custom" && (
            <div className="grid grid-cols-2 gap-2 sm:max-w-md">
              <Input id="aud-from" type="date" label={ST.from} value={from}
                onChange={(e) => reset(setFrom)(e.target.value)} />
              <Input id="aud-to" type="date" label={ST.to} value={to}
                onChange={(e) => reset(setTo)(e.target.value)} />
            </div>
          )}
          {range !== "all" && data?.day_start_hour ? (
            <p className="text-2xs text-ink-muted">{A.dayStartNote.replace("{h}", fmtNum(data.day_start_hour))}</p>
          ) : null}
        </div>

        <div className="grid grid-cols-1 gap-2 sm:grid-cols-2">
          <Input
            id="aud-q"
            type="search"
            icon={<IconSearch size={15} />}
            aria-label={A.search}
            placeholder={A.searchPlaceholder}
            value={q}
            onChange={(e) => reset(setQ)(e.target.value)}
          />
          <Select id="aud-actor" aria-label={A.person} value={actor}
            onChange={(e) => reset(setActor)(e.target.value)}>
            <option value="">{A.personAll}</option>
            <option value="system">{A.personSystem}</option>
            {actors.map((a) => <option key={a.id} value={a.id}>{a.name}</option>)}
            {actor && actor !== "system" && !actors.some((a) => a.id === actor) && (
              <option value={actor}>{A.unknown}</option>
            )}
          </Select>
        </div>

        <div>
          <Button variant="ghost" onClick={() => setShowExtra((v) => !v)} aria-expanded={showExtra || action !== ""}>
            {A.extraFilters}
          </Button>
          {(showExtra || action !== "" || (withRefresh && effGroup === "login")) && (
            <div className="mt-2 grid grid-cols-1 gap-2 sm:grid-cols-2">
              <Select id="aud-action" aria-label={A.actionType} value={action}
                onChange={(e) => reset(setAction)(e.target.value)}>
                <option value="">{A.actionAll}</option>
                {actionOptions.map((o) => (
                  <optgroup key={o.group} label={GROUP_LABEL[o.group]}>
                    {o.items.map(([k, label]) => <option key={k} value={k}>{label}</option>)}
                  </optgroup>
                ))}
              </Select>
              {effGroup === "login" && (
                <div className="flex items-center">
                  <Checkbox
                    id="aud-refresh"
                    label={A.showRefresh}
                    checked={withRefresh}
                    onChange={(e) => reset(setWithRefresh)(e.target.checked)}
                  />
                </div>
              )}
            </div>
          )}
        </div>
      </section>

      {error && (
        <Alert tone="error" title={A.loadFailed} className="mb-3">
          <span className="flex flex-wrap items-center gap-2">
            {error}
            <Button variant="secondary" onClick={load}>{m.common.retry}</Button>
          </span>
        </Alert>
      )}

      {error && !data ? null : !data && loading ? (
        <LoadingState />
      ) : !loading && list.length === 0 ? (
        <section className="surface flex flex-col items-center gap-3 p-8 text-center">
          <span className="inline-flex h-16 w-16 items-center justify-center rounded-full bg-success-solid text-on-solid">
            <IconStatus size={28} />
          </span>
          <h2 className="heading-section">{A.empty}</h2>
          <p className="max-w-md text-sm text-ink-muted">{filtered ? A.emptyFiltered : A.emptyGroup}</p>
          {filtered && <Button variant="secondary" onClick={clearAll}>{A.clear}</Button>}
        </section>
      ) : (
        <div className={loading ? "opacity-60 transition-opacity" : ""} aria-busy={loading}>
          {byDay.map(({ day, items }) => (
            <section key={day} className="mb-4">
              <h2 className="heading-card mb-2 flex items-center gap-2">
                <IconDate size={16} className="text-ink-muted" />
                {dayTitle(day)}
              </h2>
              <ul className="space-y-2">
                {items.map((e) => {
                  const t = targetOf(e);
                  const amt = amountOf(e);
                  const sum = summaryOf(e);
                  const role = topRole(e);
                  return (
                    <li key={e.id}>
                      <div
                        role="button"
                        tabIndex={0}
                        onClick={() => setOpen(e)}
                        onKeyDown={(ev) => {
                          if (ev.key === "Enter" || ev.key === " ") {
                            ev.preventDefault();
                            setOpen(e);
                          }
                        }}
                        className="surface flex cursor-pointer items-start gap-3 p-3 transition-colors hover:bg-row-hover"
                      >
                        <span className={`inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full ${CIRCLE[toneOf(e.action)]}`}>
                          {e.actor_id ? <IconUser size={18} /> : <IconSettings size={18} />}
                        </span>
                        <span className="min-w-0 flex-1">
                          <span className="block font-bold">{actionText(e.action)}</span>
                          <span className="block text-sm">
                            <span>{actorText(e)}</span>
                            {role && <span className="text-ink-muted">{m.common.nameSeparator}{role}</span>}
                            {t && (
                              <>
                                <span className="text-ink-muted"> ← </span>
                                <TargetLink t={t} mayOpen={mayOpen} />
                              </>
                            )}
                          </span>
                          {sum && <span className="mt-0.5 line-clamp-2 block text-xs text-ink-muted">{sum}</span>}
                        </span>
                        <span className="shrink-0 text-end">
                          {amt && (
                            <span className={`block font-bold tabular-nums ${amt.dir === "debit" ? "text-danger" : amt.dir === "credit" ? "text-success" : ""}`}>
                              {amt.dir === "debit" ? "− " : amt.dir === "credit" ? "+ " : ""}{fmtMoney(amt.value)}
                            </span>
                          )}
                          <time dir="ltr" className="block text-2xs text-ink-muted">{fmtTime(e.created_at)}</time>
                        </span>
                      </div>
                    </li>
                  );
                })}
              </ul>
            </section>
          ))}
        </div>
      )}

      {data && data.total > data.per_page && (
        <div className="mt-4 flex justify-center">
          <Pagination page={page} total={data.total} perPage={data.per_page} onChange={setPage} busy={loading} />
        </div>
      )}

      <DetailPanel
        e={open}
        mayOpen={mayOpen}
        onClose={() => setOpen(null)}
        onActor={(id) => {
          /* **«كل أفعال هذا الشخص» تبدأ نظيفة** — لا بحثٌ ولا فعلٌ ولا تاريخٌ باقٍ يُفرغ القائمة. */
          setOpen(null);
          clearTarget();
          setQ(""); setAction(""); setFrom(""); setTo(""); setCustomRange(false);
          setActor(id);
          setGroup("all");
          setPage(1);
        }}
      />
    </div>
  );
}
