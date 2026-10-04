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
 * - **التبويبُ الأوّلُ «الأفعال الحساسة»**، والدخولُ في تبويبه، ولكلّ مجموعةٍ
 *   تبويب.
 * - **جدولٌ من التصميم المركزيّ**: الوقت · الشخص ودوره · الفعل · الهدف رابطاً
 *   · ملخّص. **والضغطُ على السطر يفتح كلَّ التفاصيل.**
 * - **بحثٌ وفلترُ شخصٍ وفعلٍ وتاريخٍ بأزرار سريعة**، وتصديرٌ يُكتب في السجلّ.
 */

import { useCallback, useEffect, useMemo, useState } from "react";
import {
  getMessages, defaultLocale, fmtDateTime, fmtNum, fmtMoney, errorText, damascusDay,
} from "@rahalgo/i18n";
import {
  Alert, Badge, Button, DataView, EmptyState, Input, Checkbox, PageHeader,
  Pagination, Select, Sheet, Tabs, ViewToggle, useViewMode, type DataColumn,
  IconShieldCheck, IconSettings, IconUser, IconDate, IconStatus,
} from "@rahalgo/ui";
import Link from "next/link";
import { api } from "@/lib/api";
import { roleLabelByCode } from "@/lib/rbac";


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
}

/** **التبويباتُ بترتيب الخادم** (`auditGroups`) — والأوّلُ الافتراضيّ. */
const GROUPS = [
  "sensitive", "all", "login", "finance", "ops", "admin", "order", "driver",
  "merchant", "customer", "user", "menu", "catalog", "platform", "geo",
] as const;
type Group = (typeof GROUPS)[number];

/** بادئةُ الفعل ← تبويبُه — لتجميع قائمة «نوع الفعل». */
const PREFIX_GROUP: Record<string, Group> = {
  auth: "login", finance: "finance", ops: "ops", admin: "admin", order: "order",
  driver: "driver", merchant: "merchant", customer: "customer", user: "user",
  menu: "menu", catalog: "catalog", platform: "platform", coverage: "geo",
  coverage_request: "geo", operational_area: "geo", branch: "geo",
};

const ACTIONS = A.actions as Record<string, string>;
const GROUP_LABEL = A.groups as Record<Group, string>;

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
  (m.shared.txKinds as Record<string, string>)[k] ?? k;

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

/** **الهدفُ مقروءاً ورابطاً** (المشكلةُ الرابعة: «تغيير حالة طلب» بلا طلب). */
function targetOf(e: Entry): { text: string; href?: string } | null {
  const T = A.targets;
  if (!e.entity && !e.entity_id) return null;
  switch (e.entity) {
    case "order":
      return e.target_number
        ? { text: T.order.replace("{n}", String(e.target_number)),
            href: `/dashboard/history?q=${e.target_number}` }
        : { text: T.order.replace("{n}", "—") };
    case "ticket":
      return e.target_number
        ? { text: T.ticket.replace("{n}", String(e.target_number)), href: "/dashboard/tickets" }
        : { text: T.ticket.replace("{n}", "—") };
    case "merchant":
      return {
        text: e.target_label || T.merchant,
        href: e.entity_id ? `/dashboard/merchants/${e.entity_id}` : undefined,
      };
    case "user":
      return {
        text: e.target_label || T.user,
        href: e.entity_id ? `/dashboard/users/${e.entity_id}` : undefined,
      };
    case "lead":
      return { text: T.lead, href: "/dashboard/leads" };
    case "setting":
      return {
        text: (m.admin.settings.keys as Record<string, { label: string }>)[e.entity_id ?? ""]?.label
          ?? T.setting,
      };
    case "role":
      return { text: `${T.role}: ${roleLabelByCode(e.entity_id ?? "")}` };
    default:
      return e.target_label ? { text: e.target_label } : null;
  }
}

/** **ملخّصُ التفاصيل بلغةٍ لا بـJSON** — «كم» و«لمن»، لا ترميزٌ يُفكّ. */
function summaryOf(e: Entry): string {
  const d = e.details;
  const bits: string[] = [];
  if (d) {
    if (typeof d.amount === "number") bits.push(fmtMoney(d.amount));
    if (typeof d.compensation === "number" && d.compensation > 0) bits.push(fmtMoney(d.compensation));
    if (typeof d.status === "string") bits.push(statusText(d.status));
    if (typeof d.to === "string") bits.push(statusText(d.to));
    if (typeof d.kind === "string") bits.push(kindText(d.kind));
    if (typeof d.name === "string" && d.name) bits.push(d.name);
    if (typeof d.title === "string" && d.title) bits.push(d.title);
    if (typeof d.note === "string" && d.note) bits.push(d.note);
    if (typeof d.reason === "string" && d.reason) bits.push(d.reason);
    if (typeof d.resolution === "string" && d.resolution) bits.push(d.resolution);
    // تغيير الإعداد: الفرق لا النتيجة — «صار ٧٠» بلا «كان ٥٠» يُثبت الفعل ولا يُظهر أثره
    if (d.before !== undefined || d.after !== undefined)
      bits.push(`${settingValue(d.before)} ← ${settingValue(d.after)}`);
  }
  if (e.redacted) bits.push(A.redacted);
  return bits.join(m.common.listSeparator);
}

/** اسمُ الفاعل — **و«النظام» حين لا أحد.** */
const actorText = (e: Entry) => (e.actor_id ? e.actor_name ?? e.actor_phone ?? A.unknown : A.system);
const rolesText = (e: Entry) =>
  (e.actor_roles ?? []).map((r) => roleLabelByCode(r)).join(m.common.listSeparator);

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
    const extra = e.details && Object.keys(e.details).length > 0 ? JSON.stringify(e.details) : "";
    const summary = [summaryOf(e), extra].filter(Boolean).join(" | ");
    lines.push([
      fmtDateTime(e.created_at), actorText(e), e.actor_phone ?? "", rolesText(e),
      actionText(e.action), e.action, t?.text ?? "", summary, e.ip ?? "", e.user_agent ?? "",
    ].map(csvCell).join(","));
  }
  return "﻿" + lines.join("\r\n");
}

/** **اللوحةُ الجانبيّة** — كلُّ ما في السطر: قبل وبعد، الملاحظةُ كاملة، العنوانُ والجهاز. */
function DetailPanel({
  e, onClose, onActor,
}: {
  e: Entry | null;
  onClose: () => void;
  onActor: (id: string) => void;
}) {
  if (!e) return null;
  const d = e.details ?? {};
  const t = targetOf(e);
  const rest = Object.entries(d).filter(([k]) => !["before", "after", "note"].includes(k));
  const row = (label: string, value: React.ReactNode) => (
    <div className="flex flex-col gap-0.5 border-b border-line-soft py-2 last:border-b-0">
      <span className="text-xs text-ink-muted">{label}</span>
      <span className="break-words text-sm">{value}</span>
    </div>
  );
  return (
    <Sheet open onClose={onClose} title={A.panelTitle}
      footer={e.actor_id ? (
        <Button variant="secondary" onClick={() => onActor(e.actor_id as string)}>
          {A.filterByActor}
        </Button>
      ) : undefined}>
      <div className="flex flex-col">
        {row(A.action, <Badge variant={toneOf(e.action)}>{actionText(e.action)}</Badge>)}
        {row(A.when, <time dir="ltr">{fmtDateTime(e.created_at)}</time>)}
        {row(A.actor, actorText(e))}
        {rolesText(e) && row(A.role, rolesText(e))}
        {t && row(A.target, t.href ? <Link href={t.href} className="text-primary underline">{t.text}</Link> : t.text)}
        {(d.before !== undefined || d.after !== undefined) && (
          <>
            {row(A.before, settingValue(d.before))}
            {row(A.after, settingValue(d.after))}
          </>
        )}
        {typeof d.note === "string" && d.note && row(A.note, <span className="whitespace-pre-wrap">{d.note}</span>)}
        {e.redacted && row(A.details, A.redacted)}
        {rest.length > 0
          ? row(A.rawDetails, (
            <span className="flex flex-col gap-1">
              {rest.map(([k, v]) => (
                <span key={k} className="flex flex-wrap gap-2">
                  <span className="text-ink-muted" dir="ltr">{k}</span>
                  <span className="break-all">
                    {typeof v === "string" ? statusText(v) : typeof v === "number" ? fmtNum(v) : JSON.stringify(v)}
                  </span>
                </span>
              ))}
            </span>
          ))
          : !e.redacted && !(typeof d.note === "string" && d.note) && row(A.details, A.noDetails)}
        {row(A.ip, <span dir="ltr">{e.ip || A.unknown}</span>)}
        {row(A.device, <span dir="ltr" className="break-all">{e.user_agent || A.unknown}</span>)}
      </div>
    </Sheet>
  );
}

export default function AuditPage() {
  const [group, setGroup] = useState<Group>("sensitive");
  const [q, setQ] = useState("");
  const [actor, setActor] = useState("");
  const [action, setAction] = useState("");
  const [from, setFrom] = useState("");
  const [to, setTo] = useState("");
  /** **والتجديدُ مخفيٌّ افتراضاً** — تكتبه الساعةُ لا الإنسان. */
  const [withRefresh, setWithRefresh] = useState(false);
  const [page, setPage] = useState(1);

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
  const [view, setView] = useViewMode("audit");

  const params = useCallback(() => {
    const qs = new URLSearchParams({ group });
    if (withRefresh) qs.set("refresh", "true");
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
  }, [group, withRefresh, from, to, actor, action, q]);

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

  const reset = <T,>(set: (v: T) => void) => (v: T) => {
    set(v);
    setPage(1);
  };

  /** **مدىً جاهز** — بيوم دمشق (المشكلةُ التاسعة). */
  const today = damascusDay();
  const yesterday = damascusDay(Date.now() - 86_400_000);
  const presets = [
    { id: "today", label: A.rangeToday, from: today, to: today },
    { id: "yesterday", label: A.rangeYesterday, from: yesterday, to: yesterday },
    { id: "last7", label: A.rangeLast7, from: damascusDay(Date.now() - 6 * 86_400_000), to: today },
  ];
  const filtered = q !== "" || actor !== "" || action !== "" || from !== "" || to !== "";

  /** **قائمةُ الأفعال مجمّعةً بتبويبها** — بأسمائها العربيّة. */
  const actionOptions = useMemo(() => {
    const by = new Map<Group, [string, string][]>();
    for (const [k, label] of Object.entries(ACTIONS)) {
      const g: Group = PREFIX_GROUP[k.split(".")[0] ?? ""] ?? "all";
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
    } finally {
      setExporting(false);
    }
  }

  const list = data?.entries ?? [];

  const columns: DataColumn<Entry>[] = [
    {
      id: "when", header: A.when, icon: <IconDate />,
      cell: (e) => <time dir="ltr" className="whitespace-nowrap text-xs text-ink-muted">{fmtDateTime(e.created_at)}</time>,
    },
    {
      id: "actor", header: A.actor, primary: true, noLabel: true,
      cell: (e) => (
        <span className="inline-flex min-w-0 items-center gap-2">
          <span className="flex h-7 w-7 shrink-0 items-center justify-center rounded-control bg-field">
            {e.actor_id
              ? <IconUser size={14} className="text-ink-muted" />
              : <IconSettings size={14} className="text-primary" />}
          </span>
          <span className="min-w-0">
            <span className="block truncate text-sm font-medium">{actorText(e)}</span>
            {rolesText(e) && <span className="block truncate text-xs text-ink-muted">{rolesText(e)}</span>}
          </span>
        </span>
      ),
    },
    {
      id: "action", header: A.action,
      cell: (e) => <Badge variant={toneOf(e.action)}>{actionText(e.action)}</Badge>,
    },
    {
      id: "target", header: A.target,
      cell: (e) => {
        const t = targetOf(e);
        if (!t) return <span className="text-ink-muted">—</span>;
        return t.href ? (
          <Link href={t.href} onClick={(ev) => ev.stopPropagation()} className="text-primary underline">
            {t.text}
          </Link>
        ) : <span>{t.text}</span>;
      },
    },
    {
      id: "summary", header: A.details, block: true,
      cell: (e) => <span className="line-clamp-2 text-xs text-ink-muted">{summaryOf(e) || "—"}</span>,
    },
  ];

  return (
    <div>
      <PageHeader
        icon={IconShieldCheck}
        title={A.title}
        actions={data?.can_export ? (
          <Button variant="secondary" onClick={exportCsv} disabled={exporting}>
            {exporting ? A.exporting : A.export}
          </Button>
        ) : undefined}
      />
      <p className="mb-4 text-sm text-ink-muted">{A.hint}</p>
      {exportNote && (
        <Alert tone="info" className="mb-3" onDismiss={() => setExportNote("")}>{exportNote}</Alert>
      )}
      {data && !data.money_visible && (
        <Alert tone="info" className="mb-3">{A.moneyHidden}</Alert>
      )}

      <Tabs
        items={GROUPS.map((g) => ({ key: g, label: GROUP_LABEL[g] }))}
        value={group}
        onChange={(k) => reset(setGroup)(k)}
        className="mb-4"
      />

      {/* **المرشّحات** — بحثٌ وشخصٌ وفعلٌ ومدىً جاهزٌ بضغطة، وتلتفّ على الجوال. */}
      <div className="mb-3 grid grid-cols-1 gap-3 sm:grid-cols-2 lg:grid-cols-4">
        <Input
          id="aud-q"
          type="search"
          label={A.search}
          placeholder={A.searchPlaceholder}
          value={q}
          onChange={(e) => reset(setQ)(e.target.value)}
        />
        <Select id="aud-actor" label={A.person} value={actor}
          onChange={(e) => reset(setActor)(e.target.value)}>
          <option value="">{A.personAll}</option>
          <option value="system">{A.personSystem}</option>
          {actors.map((a) => <option key={a.id} value={a.id}>{a.name}</option>)}
          {actor && actor !== "system" && !actors.some((a) => a.id === actor) && (
            <option value={actor}>{A.unknown}</option>
          )}
        </Select>
        <Select id="aud-action" label={A.actionType} value={action}
          onChange={(e) => reset(setAction)(e.target.value)}>
          <option value="">{A.actionAll}</option>
          {actionOptions.map((o) => (
            <optgroup key={o.group} label={GROUP_LABEL[o.group]}>
              {o.items.map(([k, label]) => <option key={k} value={k}>{label}</option>)}
            </optgroup>
          ))}
        </Select>
        <div className="flex items-end">
          <Checkbox
            id="aud-refresh"
            label={A.showRefresh}
            checked={withRefresh}
            onChange={(e) => reset(setWithRefresh)(e.target.checked)}
          />
        </div>
      </div>
      <div className="mb-4 flex flex-wrap items-end gap-2">
        {presets.map((p) => {
          const on = from === p.from && to === p.to;
          return (
            <Button
              key={p.id}
              variant={on ? "primary" : "secondary"}
              aria-pressed={on}
              onClick={() => {
                setFrom(on ? "" : p.from);
                setTo(on ? "" : p.to);
                setPage(1);
              }}
            >
              {p.label}
            </Button>
          );
        })}
        <div className="min-w-[8.75rem] flex-1 sm:w-40 sm:flex-none">
          <Input id="aud-from" type="date" label={ST.from} value={from}
            onChange={(e) => reset(setFrom)(e.target.value)} />
        </div>
        <div className="min-w-[8.75rem] flex-1 sm:w-40 sm:flex-none">
          <Input id="aud-to" type="date" label={ST.to} value={to}
            onChange={(e) => reset(setTo)(e.target.value)} />
        </div>
        {filtered && (
          <Button variant="ghost" onClick={() => {
            setQ(""); setActor(""); setAction(""); setFrom(""); setTo(""); setPage(1);
          }}>
            {A.clear}
          </Button>
        )}
      </div>

      {error && (
        <Alert tone="error" title={A.loadFailed} className="mb-3">
          <span className="flex flex-wrap items-center gap-2">
            {error}
            <Button variant="secondary" onClick={load}>{m.common.retry}</Button>
          </span>
        </Alert>
      )}

      {data && (
        <div className="mb-2 flex items-center justify-between gap-2">
          <span className="text-sm text-ink-muted">
            {data.total_capped
              ? A.countCapped.replace("{n}", fmtNum(data.total))
              : A.count.replace("{n}", fmtNum(data.total))}
          </span>
          <ViewToggle view={view} onChange={setView}
            tableLabel={m.common.viewTable} cardsLabel={m.common.viewCards} />
        </div>
      )}

      {error && !data ? null : !loading && data && list.length === 0 ? (
        <EmptyState icon={IconStatus} title={A.empty} />
      ) : (
        <div className={loading && list.length > 0 ? "opacity-60 transition-opacity" : ""} aria-busy={loading}>
          <DataView
            items={list}
            getKey={(e) => String(e.id)}
            columns={columns}
            empty={A.empty}
            view={view}
            loading={loading}
            onRowClick={setOpen}
          />
        </div>
      )}

      {data && data.total > data.per_page && (
        <div className="mt-4 flex justify-center">
          <Pagination page={page} total={data.total} perPage={data.per_page} onChange={setPage} busy={loading} />
        </div>
      )}

      <DetailPanel
        e={open}
        onClose={() => setOpen(null)}
        onActor={(id) => {
          setOpen(null);
          setActor(id);
          setGroup("all");
          setPage(1);
        }}
      />
    </div>
  );
}
