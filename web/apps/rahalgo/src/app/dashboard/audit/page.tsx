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

import { Fragment, useCallback, useEffect, useMemo, useState, type ReactNode } from "react";
import {
  getMessages, defaultLocale, fmtDateTime, fmtNum, fmtMoney, fmtTime, fmtLongDate, errorText, damascusDay,
} from "@rahalgo/i18n";
import {
  Alert, Button, Checkbox, Select, Sheet,
  IconShieldCheck, IconSettings, IconUser, IconStatus, IconWallet, IconDate, IconOverview,
  AdminPage, AdminEmpty, AdminLoading, AdminReload, FilterBar, FilterField, FilterSelect, PeriodPicker,
  Row, RowGroupTitle, RowIcon, RowList, SummaryStrip, type AdminTab, type SummaryItem,
} from "@rahalgo/ui";
import Link from "next/link";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useCanCall, useCanOpen } from "@/lib/policy";
import { roleLabelByCode, classifyRole } from "@/lib/rolemeta";

const m = getMessages(defaultLocale);
const A = m.admin.audit;

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

/** **الأفعالُ الماليّةُ تُبرَز** — هي ما يُبحث عنه حين يُبحث في هذا السجلّ. وتُقال بأيقونة المحفظة
    مع اللون (دفعةُ التصميم ١) — لا دائرةٌ حمراءُ مصمتةٌ لكلّ حركة مال. */
const isFinance = (action: string) => action.startsWith("finance.");

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
        <div className="mb-2 flex items-center gap-3 border-b border-line-soft pb-3">
          <RowIcon
            icon={isFinance(e.action) ? IconWallet : e.actor_id ? IconUser : IconSettings}
            tone={isFinance(e.action) ? "attention" : "neutral"}
          />
          <span className="min-w-0">
            <span className="block font-bold">{actionText(e.action)}</span>
            <span className="block text-xs text-ink-muted" dir="ltr">{fmtDateTime(e.created_at)}</span>
          </span>
        </div>
        {amt && (
          <div className="mb-2 border-b border-line-soft pb-3">
            <span className="block text-xs text-ink-muted">
              {A.amount}{amt.dir ? ` · ${amt.dir === "debit" ? A.debit : A.credit}` : ""}
            </span>
            <span className={`admin-figure block ${amt.dir === "debit" ? "text-danger" : amt.dir === "credit" ? "text-success" : ""}`}>
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
        <details className="mt-3 border-t border-line-soft pt-3">
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

/** **يومُ المنصّة لحدث** — بطرح ساعة فتح الدوام، كما يحسبه الخادم. */
const platformDay = (iso: string, startHour: number) => damascusDay(new Date(iso).getTime() - startHour * 3_600_000);

export default function AuditPage() {
  const mayOpen = useMayOpen();
  const [group, setGroup] = useState<Group>("sensitive");
  const [q, setQ] = useState("");
  const [actor, setActor] = useState("");
  const [action, setAction] = useState("");
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
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
   * **يومُ المنصّة من الخادم** (تدقيقُ ٢٠٢٦-١٠-٢٧): «اليوم» يبدأ بساعة فتح الدوام كبقيّة اللوحة —
   * ومختارُ الفترة الواحد (`PeriodPicker`) يبني عليه. وقبل وصول الجواب الأوّل يُقدَّر بيوم دمشق.
   */
  const startHour = data?.day_start_hour ?? 0;
  const today = data?.today ?? damascusDay(Date.now() - startHour * 3_600_000);
  const yesterday = data?.yesterday ?? damascusDay(Date.now() - startHour * 3_600_000 - 86_400_000);

  const filtered = q !== "" || actor !== "" || action !== "" || from !== "" || to !== "" || scoped;
  const clearAll = () => {
    setQ(""); setActor(""); setAction(""); setFrom(""); setTo(""); setPage(1);
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

  const list = useMemo(() => data?.entries ?? [], [data]);
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
  // ══════════════════════════════════════════════════════════════════
  // **دفعةُ التصميم ١ (٢٠٢٦-١٠-٣١):** كانت الصفحةُ خارجَ الحاوية، بستّة أسطرِ فلاتر وخمسين بطاقةً منفصلة
  // وخاناتٍ داخل الرأس. **صارت**: رأسٌ بسطرٍ والمجموعاتُ المهمّةُ تبويباتٌ خطّاً · شريطٌ واحد (بحث ·
  // الفترة · الشخص · «فلاتر» للباقي · «امسح») · أربعةُ أرقامٍ لليوم **لا تُفلتر** · وصفوفٌ في لوحٍ واحدٍ
  // بعناوين الأيّام.
  // ══════════════════════════════════════════════════════════════════
  const n = (k: "sensitive" | "finance" | "settings" | "all") => counts?.[k] ?? 0;
  const summary: SummaryItem[] = [
    { key: "sensitive", icon: IconShieldCheck, label: A.tiles.sensitive, value: counts ? fmtNum(n("sensitive")) : "—", tone: n("sensitive") > 0 ? "attention" : "muted" },
    { key: "finance", icon: IconWallet, label: A.tiles.finance, value: counts ? fmtNum(n("finance")) : "—", tone: n("finance") > 0 ? "neutral" : "muted" },
    { key: "settings", icon: IconSettings, label: A.tiles.settings, value: counts ? fmtNum(n("settings")) : "—", tone: n("settings") > 0 ? "neutral" : "muted" },
    { key: "all", icon: IconOverview, label: A.tiles.all, value: counts ? fmtNum(n("all")) : "—", tone: n("all") > 0 ? "neutral" : "muted" },
  ];
  const moreGroup = MORE_GROUPS.includes(group);
  const tabs: AdminTab<Group>[] = [
    ...MAIN_GROUPS.map((g) => ({ key: g, label: GROUP_LABEL[g] })),
    // **ومجموعةٌ من «المزيد» تظهر تبويباً حين تُختار** — كي لا يبقى الرأسُ بلا تبويبٍ مختار.
    ...(moreGroup ? [{ key: group, label: GROUP_LABEL[group] }] : []),
  ];
  const moreActive = (action ? 1 : 0) + (moreGroup ? 1 : 0) + (withRefresh && effGroup === "login" ? 1 : 0);
  const countText = data
    ? data.total_capped
      ? A.countCapped.replace("{n}", fmtNum(data.total))
      : A.count.replace("{n}", fmtNum(data.total))
    : undefined;

  return (
    <AdminPage<Group>
      icon={IconShieldCheck}
      title={A.title}
      actions={
        data?.can_export ? (
          <Button variant="secondary" size="sm" onClick={exportCsv} disabled={exporting}>
            {exporting ? A.exporting : A.export}
          </Button>
        ) : undefined
      }
      tabs={scoped ? undefined : tabs}
      tab={group}
      onTab={(g) => reset(setGroup)(g)}
      notices={
        (scoped || actor || exportNote || error || (data && !data.money_visible)) ? (
          <div className="space-y-2">
            {/* **التقييدُ ظاهرٌ ويُشال** — رابطُ «السجل» من الإعدادات، أو شخصٌ بعينه. */}
            {scoped && (
              <Alert tone="info" onDismiss={clearTarget}>
                {target.entity === "setting"
                  ? A.scopedSetting.replace("{name}", scopeName)
                  : A.scopedOther.replace("{name}", scopeName)}
              </Alert>
            )}
            {actor && (
              <Alert tone="info" onDismiss={() => reset(setActor)("")}>
                {A.actorChip.replace("{name}", actorName)}
              </Alert>
            )}
            {data && !data.money_visible && <Alert tone="info">{A.moneyHidden}</Alert>}
            {exportNote && (
              <Alert tone="warning" onDismiss={() => setExportNote("")}>
                {exportNote}
              </Alert>
            )}
            {error && data && (
              <Alert tone="error" title={A.loadFailed}>
                <span className="flex flex-wrap items-center gap-2">
                  {error}
                  <Button variant="secondary" size="sm" onClick={load}>
                    {m.admin.kit.retry}
                  </Button>
                </span>
              </Alert>
            )}
          </div>
        ) : undefined
      }
      toolbar={
        <FilterBar
          search={{ id: "aud-q", value: q, onChange: (v) => reset(setQ)(v), placeholder: A.searchPlaceholder }}
          filtered={filtered}
          onClear={clearAll}
          moreActive={moreActive}
          end={countText}
          more={
            <>
              {!scoped && (
                <FilterField id="aud-group" label={A.more}>
                  <Select
                    id="aud-group"
                    value={moreGroup ? group : ""}
                    onChange={(e) => reset(setGroup)((e.target.value || "sensitive") as Group)}
                  >
                    <option value="">{A.all}</option>
                    {MORE_GROUPS.map((g) => (
                      <option key={g} value={g}>
                        {GROUP_LABEL[g]}
                      </option>
                    ))}
                  </Select>
                </FilterField>
              )}
              <FilterField id="aud-action" label={A.actionType}>
                <Select id="aud-action" value={action} onChange={(e) => reset(setAction)(e.target.value)}>
                  <option value="">{A.actionAll}</option>
                  {actionOptions.map((o) => (
                    <optgroup key={o.group} label={GROUP_LABEL[o.group]}>
                      {o.items.map(([k, label]) => (
                        <option key={k} value={k}>
                          {label}
                        </option>
                      ))}
                    </optgroup>
                  ))}
                </Select>
              </FilterField>
              {effGroup === "login" && (
                <Checkbox
                  id="aud-refresh"
                  label={A.showRefresh}
                  checked={withRefresh}
                  onChange={(e) => reset(setWithRefresh)(e.target.checked)}
                />
              )}
            </>
          }
        >
          <PeriodPicker
            id="aud-period"
            from={from}
            to={to}
            today={data?.today ?? ""}
            onChange={(f, t) => {
              setFrom(f);
              setTo(t);
              setPage(1);
            }}
          />
          <FilterSelect id="aud-actor" label={A.person} value={actor} onChange={(v) => reset(setActor)(v)}>
            <option value="">{A.personAll}</option>
            <option value="system">{A.personSystem}</option>
            {actors.map((a) => (
              <option key={a.id} value={a.id}>
                {a.name}
              </option>
            ))}
            {actor && actor !== "system" && !actors.some((a) => a.id === actor) && (
              <option value={actor}>{A.unknown}</option>
            )}
          </FilterSelect>
        </FilterBar>
      }
      summary={<SummaryStrip items={summary} />}
      pagination={
        data ? { page, total: data.total, perPage: data.per_page, onChange: setPage, busy: loading } : undefined
      }
    >
      {error && !data ? (
        <AdminReload onRetry={load} label={A.loadFailed} />
      ) : !data && loading ? (
        <AdminLoading />
      ) : !loading && list.length === 0 ? (
        <AdminEmpty
          icon={IconStatus}
          title={filtered ? A.empty : A.emptyGroup}
          action={
            filtered ? (
              <Button variant="secondary" size="sm" onClick={clearAll}>
                {A.clear}
              </Button>
            ) : undefined
          }
        />
      ) : (
        <div className={loading ? "opacity-60 transition-opacity" : ""} aria-busy={loading}>
          <RowList>
            {byDay.map(({ day, items }) => (
              <Fragment key={day}>
                <RowGroupTitle icon={IconDate}>{dayTitle(day)}</RowGroupTitle>
                {items.map((e) => {
                  const t = targetOf(e);
                  const amt = amountOf(e);
                  const sum = summaryOf(e);
                  const role = topRole(e);
                  const fin = e.action.startsWith("finance.");
                  return (
                    <Row
                      key={e.id}
                      label={actionText(e.action)}
                      onOpen={() => setOpen(e)}
                      leading={
                        <RowIcon
                          icon={fin ? IconWallet : e.actor_id ? IconUser : IconSettings}
                          tone={fin ? "attention" : "neutral"}
                        />
                      }
                      title={actionText(e.action)}
                      meta={
                        <>
                          <span className="block">
                            <span className="text-ink">{actorText(e)}</span>
                            {role && <span>{m.common.nameSeparator}{role}</span>}
                            {t && (
                              <>
                                <span> ← </span>
                                <TargetLink t={t} mayOpen={mayOpen} />
                              </>
                            )}
                          </span>
                          {sum && <span className="mt-0.5 line-clamp-2 block text-xs">{sum}</span>}
                        </>
                      }
                      trailing={
                        <>
                          {amt && (
                            <span
                              className={`block font-bold tabular-nums ${
                                amt.dir === "debit" ? "text-danger" : amt.dir === "credit" ? "text-success" : ""
                              }`}
                            >
                              {amt.dir === "debit" ? "− " : amt.dir === "credit" ? "+ " : ""}
                              {fmtMoney(amt.value)}
                            </span>
                          )}
                          <time dir="ltr" className="block text-xs text-ink-muted">
                            {fmtTime(e.created_at)}
                          </time>
                        </>
                      }
                    />
                  );
                })}
              </Fragment>
            ))}
          </RowList>
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
          setQ(""); setAction(""); setFrom(""); setTo("");
          setActor(id);
          setGroup("all");
          setPage(1);
        }}
      />
    </AdminPage>
  );
}
