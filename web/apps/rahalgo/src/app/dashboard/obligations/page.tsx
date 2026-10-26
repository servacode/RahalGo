"use client";

/**
 * **الديون — تبويبُ «الديون» في الخزينة** (قراراتُ المالك ٢٠٢٦-١٠-٠٤).
 *
 * الدَّينُ على المتاجر والمناديب (`financial_obligations`): على من، وكم، ولماذا،
 * ومن أيّ طلب، وكم بقي. **وفعلان، كلٌّ منهما اقتراحٌ يوافق عليه غيرُ مقترِحه**:
 * «دفع نقدي بالمكتب» (يدخل صندوقَ المكتب) و«شطب الدين» (يوافق مديرُ المنصّة).
 *
 * **والحالُ أربعة**: مفتوح · انسدّ · انشطب لأنّ الطلب ما صار · شطبته الإدارة —
 * **والملغى لا يُعرض مسدَّداً.** **وفشلُ التحميل خطأٌ لا صفر.**
 */

import { useEffect, useState } from "react";
import {
  getMessages,
  defaultLocale,
  fmtNum,
  fmtRef,
  fmtDateTime,
  fmtMoney,
  errorText,
} from "@rahalgo/i18n";
import {
  Money,
  Badge,
  Button,
  ButtonLink,
  Input,
  Select,
  Modal,
  Alert,
  PageContainer,
  Pagination,
  LoadingState,
  ReloadState,
  useLiveData,
  IconBalance,
  IconWallet,
  IconStore,
  IconUser,
  IconOrder,
  IconStatus,
  IconDate,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useCanOpen } from "@/lib/policy";
import { OpenLink } from "@/components/admin/OpenLink";
import { PageHero } from "@/components/admin/PageHero";
import { Tile } from "@/components/admin/DashParts";

const m = getMessages(defaultLocale);
const O = m.admin.obligations;

type Kind = "" | "merchant" | "rep" | "driver";
type State = "" | "open" | "closed" | "paid" | "voided" | "written_off";
type RowState = "open" | "paid" | "voided" | "written_off";
type Method = keyof typeof O.methods;

interface Settlement {
  amount: number;
  remaining: number;
  order_number: number | null;
  method: Method;
  created_at: string;
}

interface Pending {
  id: string;
  kind: "office_cash" | "write_off";
  amount: number;
  note: string;
  /** مقترحُ الطلب — لا يرى زرَّ الموافقة عليه. */
  proposed_by?: string;
}

interface Obligation {
  id: string;
  party_kind: "merchant" | "rep" | "driver";
  party_id: string;
  party_user_id: string;
  party_name: string;
  amount: number;
  outstanding: number;
  cause: string;
  order_id: string | null;
  order_number: number | null;
  created_at: string;
  age_days: number;
  state: RowState;
  pending: Pending | null;
  settlements: Settlement[];
}

interface Summary {
  outstanding_total: number;
  merchants_total: number;
  reps_total: number;
  drivers_total: number;
  debtors: number;
  near_limit: number;
  overdue: number;
  pending_requests: number;
}

interface ObligationsPage {
  obligations: Obligation[];
  total: number;
  page: number;
  per_page: number;
  summary: Summary;
  alert_days: number;
  can_manage: boolean;
  can_approve_writeoff: boolean;
  can_export: boolean;
}

interface Filters {
  kind: Kind;
  state: State;
  cause: string;
  q: string;
  party: string;
  from: string;
  to: string;
  overdue: boolean;
  /** **ما عليه طلبُ دفعٍ أو شطبٍ ينتظر الموافقة** — تفتحه بطاقةُ الرئيسيّة (٢٠٢٦-١٠-١٥). */
  pending: boolean;
}

const EMPTY: Filters = {
  kind: "",
  state: "",
  cause: "",
  q: "",
  party: "",
  from: "",
  to: "",
  overdue: false,
  pending: false,
};

/** **سببٌ مصنَّفٌ يُترجَم، ومجهولُه يُعرض كما جاء.** */
const causeText = (c: string): string => (O.causes as Record<string, string>)[c] ?? c;

const STATE_VARIANT: Record<RowState, "warning" | "success" | "neutral" | "danger"> = {
  open: "warning",
  paid: "success",
  voided: "neutral",
  written_off: "danger",
};

/** **المرشّحاتُ في رابط الصفحة** — يُرسَل الرابطُ فيُفتح على الحال نفسِه. */
function readUrl(): Filters {
  if (typeof window === "undefined") return EMPTY;
  const p = new URLSearchParams(window.location.search);
  return {
    kind: (p.get("party_kind") ?? "") as Kind,
    state: (p.get("state") ?? "") as State,
    cause: p.get("cause") ?? "",
    q: p.get("q") ?? "",
    party: p.get("party_id") ?? "",
    from: p.get("from") ?? "",
    to: p.get("to") ?? "",
    overdue: p.get("overdue") === "1",
    pending: p.get("pending") === "1",
  };
}

function query(f: Filters): string {
  const p = new URLSearchParams();
  if (f.kind) p.set("party_kind", f.kind);
  if (f.state) p.set("state", f.state);
  if (f.cause) p.set("cause", f.cause);
  if (f.q.trim()) p.set("q", f.q.trim());
  if (f.party) p.set("party_id", f.party);
  if (f.from) p.set("from", f.from);
  if (f.to) p.set("to", f.to);
  if (f.overdue) p.set("overdue", "1");
  if (f.pending) p.set("pending", "1");
  return p.toString();
}

/** **خانةُ CSV لا تُقرأ صيغة** — كسجلّ الأحداث. */
const QUOTE = String.fromCharCode(34);
const NEEDS_QUOTES = new RegExp(`[${QUOTE},\\n\\r]`);
function csvCell(v: string): string {
  let s = v ?? "";
  if (/^[=+\-@\t\r]/.test(s)) s = `'${s}`;
  if (NEEDS_QUOTES.test(s)) s = QUOTE + s.split(QUOTE).join(QUOTE + QUOTE) + QUOTE;
  return s;
}

function buildCsv(rows: Obligation[]): string {
  const C = O.csv;
  const head = [C.party, C.kind, C.cause, C.order, C.amount, C.outstanding, C.state, C.createdAt, C.age];
  const lines = [head.map(csvCell).join(",")];
  for (const o of rows) {
    lines.push(
      [
        o.party_name,
        O.kinds[o.party_kind],
        causeText(o.cause),
        o.order_number != null ? String(o.order_number) : "",
        String(o.amount),
        String(o.outstanding),
        O.states[o.state],
        fmtDateTime(o.created_at),
        String(o.age_days),
      ]
        .map(csvCell)
        .join(","),
    );
  }
  return "﻿" + lines.join("\r\n");
}

const partyHref = (o: Obligation) =>
  o.party_kind === "merchant" ? `/dashboard/merchants/${o.party_id}` : `/dashboard/users/${o.party_id}`;

export default function ObligationsPage() {
  const { user: me } = useAuth();
  // **وملفُّ الطرف والطلبُ لمن يُفتحان له** (قرارُ المالك ٢٠٢٦-١٠-٠٥).
  const canOpen = useCanOpen();
  const [f, setF] = useState<Filters>(EMPTY);
  const [ready, setReady] = useState(false);
  const [page, setPage] = useState(1);
  /** **الدينُ المفتوحةُ سطورُه** — واحدٌ في المرّة، فلا تزدحم الشاشة. */
  const [open, setOpen] = useState<string | null>(null);
  /** **نافذةُ الفعل** — دفعٌ أو شطبٌ على دينٍ بعينه. */
  const [act, setAct] = useState<{ o: Obligation; kind: "office_cash" | "write_off" } | null>(null);
  const [notice, setNotice] = useState("");
  const [exporting, setExporting] = useState(false);
  /** **بحثٌ يُكتب ثمّ يُرسَل** — لا نداءَ مع كلّ حرف. */
  const [typed, setTyped] = useState("");

  useEffect(() => {
    const init = readUrl();
    setF(init);
    setTyped(init.q);
    setReady(true);
  }, []);

  useEffect(() => {
    if (!ready) return;
    const qs = query(f);
    window.history.replaceState(null, "", qs ? `?${qs}` : window.location.pathname);
  }, [f, ready]);

  const qs = query(f);
  const { data, loading, error, reload } = useLiveData<ObligationsPage>(
    () =>
      ready
        ? api(`/api/v1/admin/obligations?page=${page}${qs ? `&${qs}` : ""}`)
        : new Promise<ObligationsPage>(() => {}),
    ["wallet"],
    [qs, page, ready],
  );
  /** **التحميلُ عند كلّ تبديل** — لا تبقى القائمةُ القديمةُ ظاهرةً كأنّها الجديدة. */
  const [shownKey, setShownKey] = useState("");
  const key = `${qs}|${page}`;
  useEffect(() => {
    if (data) setShownKey(key);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [data]);
  const stale = shownKey !== key;

  function pick(next: Partial<Filters>) {
    setPage(1);
    setOpen(null);
    setShownKey("");
    setF((cur) => ({ ...cur, ...next }));
  }

  async function exportCsv() {
    setExporting(true);
    setNotice("");
    try {
      const r = await api<{ obligations: Obligation[]; truncated: boolean; cap: number }>(
        `/api/v1/admin/obligations/export${qs ? `?${qs}` : ""}`,
      );
      const blob = new Blob([buildCsv(r?.obligations ?? [])], { type: "text/csv;charset=utf-8" });
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = `debts-${new Date().toISOString().slice(0, 10)}.csv`;
      a.click();
      URL.revokeObjectURL(url);
      if (r?.truncated) setNotice(O.exportTruncated.replace("{n}", fmtNum(r.cap)));
    } catch (err) {
      setNotice(`${O.exportFailed}: ${errorText(err)}`);
    } finally {
      setExporting(false);
    }
  }

  async function decide(p: Pending, approve: boolean) {
    setNotice("");
    try {
      await api(`/api/v1/admin/obligation-requests/${p.id}/${approve ? "approve" : "reject"}`, {
        method: "POST",
        body: JSON.stringify({}),
      });
      setNotice(O.decided);
      reload();
    } catch (err) {
      setNotice(errorText(err));
    }
  }

  if (!ready || (loading && !data)) return <LoadingState />;

  const rows = data?.obligations ?? [];
  const sum = data?.summary;
  const alertDays = data?.alert_days ?? 30;

  // **تصميمٌ احترافيّ** (المالك ٢٠٢٦-١٠-٢٦: «صفحة الديون كمان تصميم احترافي كالسابق») — رأسٌ
  // موحّدٌ بتبويبات الأطراف، أرقامٌ تُضغط، فلاترُ في كرت، وكرتٌ لكلّ دينٍ بمبلغه وما بقي وأفعاله.
  return (
    <PageContainer>
      <PageHero<string>
        icon={IconBalance}
        title={O.title}
        subtitle={O.hint}
        actions={
          <>
            <ButtonLink variant="secondary" href="/dashboard/treasury">
              {O.treasuryLink}
            </ButtonLink>
            {data?.can_export && (
              <Button variant="secondary" onClick={exportCsv} disabled={exporting}>
                {exporting ? O.exporting : O.export}
              </Button>
            )}
          </>
        }
        tabs={[
          { key: "all", label: O.kinds.all, icon: IconBalance },
          { key: "merchant", label: O.merchantsTotal, icon: IconStore },
          { key: "rep", label: O.repsTotal, icon: IconUser },
          { key: "driver", label: O.driversTotal, icon: IconUser },
        ]}
        value={f.kind || "all"}
        onChange={(k) => pick({ kind: (k === "all" ? "" : k) as Kind })}
      />

      {error && !data ? (
        <ReloadState label={O.loadFailed} onRetry={reload} />
      ) : (
        <>
          {error && (
            <Alert tone="error" className="mb-3">
              {O.loadFailed}
            </Alert>
          )}
          {notice && (
            <Alert tone="info" className="mb-3" onDismiss={() => setNotice("")}>
              {notice}
            </Alert>
          )}

          {/* **كلُّ المفتوح بأنواعه** — لا الصفحةُ المعروضة. */}
          <div className="mb-2 grid grid-cols-2 gap-2 lg:grid-cols-4">
            <Tile
              icon={IconWallet}
              label={O.outstandingTotal}
              value={fmtMoney(sum?.outstanding_total ?? 0)}
              sub={O.debtorsSub.replace("{n}", fmtNum(sum?.debtors ?? 0))}
              tone={(sum?.outstanding_total ?? 0) > 0 ? "warning" : "muted"}
              onClick={() => pick({ kind: "", state: "open" })}
            />
            <Tile
              icon={IconStore}
              label={O.merchantsTotal}
              value={fmtMoney(sum?.merchants_total ?? 0)}
              sub={(sum?.near_limit ?? 0) > 0 ? O.nearLimitSub.replace("{n}", fmtNum(sum?.near_limit ?? 0)) : undefined}
              tone={(sum?.near_limit ?? 0) > 0 ? "danger" : (sum?.merchants_total ?? 0) > 0 ? "default" : "muted"}
              onClick={() => pick({ kind: "merchant" })}
            />
            <Tile
              icon={IconUser}
              label={O.repsTotal}
              value={fmtMoney(sum?.reps_total ?? 0)}
              tone={(sum?.reps_total ?? 0) > 0 ? "default" : "muted"}
              onClick={() => pick({ kind: "rep" })}
            />
            <Tile
              icon={IconUser}
              label={O.driversTotal}
              value={fmtMoney(sum?.drivers_total ?? 0)}
              tone={(sum?.drivers_total ?? 0) > 0 ? "default" : "muted"}
              onClick={() => pick({ kind: "driver" })}
            />
          </div>
          <div className="mb-4 grid grid-cols-1 gap-2 sm:grid-cols-2">
            <Tile
              icon={IconDate}
              label={O.overdue.replace("{n}", fmtNum(alertDays))}
              value={fmtNum(sum?.overdue ?? 0)}
              tone={(sum?.overdue ?? 0) > 0 ? "danger" : "muted"}
              onClick={() => pick({ overdue: !f.overdue })}
            />
            <Tile
              icon={IconStatus}
              label={O.pendingRequests}
              value={fmtNum(sum?.pending_requests ?? 0)}
              tone={(sum?.pending_requests ?? 0) > 0 ? "accent" : "muted"}
              onClick={() => pick({ pending: !f.pending })}
            />
          </div>

          {/* ── الفلاتر في كرتٍ واحد ── */}
          <form
            className="surface mb-4 space-y-3 p-4"
            onSubmit={(e) => {
              e.preventDefault();
              pick({ q: typed });
            }}
          >
            <div className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-4">
              <Input
                id="obl-search"
                icon={<IconUser size={15} />}
                value={typed}
                placeholder={O.search}
                onChange={(e) => setTyped(e.target.value)}
                onBlur={() => typed !== f.q && pick({ q: typed })}
              />
              <Select aria-label={O.states.all} value={f.state} onChange={(e) => pick({ state: e.target.value as State })}>
                <option value="">{O.states.all}</option>
                <option value="open">{O.states.open}</option>
                <option value="closed">{O.states.closed}</option>
                <option value="paid">{O.states.paid}</option>
                <option value="voided">{O.states.voided}</option>
                <option value="written_off">{O.states.written_off}</option>
              </Select>
              <Select aria-label={O.causeAll} value={f.cause} onChange={(e) => pick({ cause: e.target.value })}>
                <option value="">{O.causeAll}</option>
                {Object.entries(O.causes).map(([k, v]) => (
                  <option key={k} value={k}>
                    {v}
                  </option>
                ))}
              </Select>
              <div className="grid grid-cols-2 gap-2">
                <Input type="date" aria-label={O.from} title={O.from} value={f.from} onChange={(e) => pick({ from: e.target.value })} />
                <Input type="date" aria-label={O.to} title={O.to} value={f.to} onChange={(e) => pick({ to: e.target.value })} />
              </div>
            </div>
            <div className="flex flex-wrap items-center gap-2">
              <Button
                type="button"
                variant={f.overdue ? "primary" : "secondary"}
                onClick={() => pick({ overdue: !f.overdue })}
              >
                {O.overdueOnly}
              </Button>
              <Button
                type="button"
                variant={f.pending ? "primary" : "secondary"}
                onClick={() => pick({ pending: !f.pending })}
              >
                {O.pendingOnly}
              </Button>
              {f.party && <Badge variant="neutral">{O.partyFiltered}</Badge>}
              {qs && (
                <Button
                  type="button"
                  variant="ghost"
                  onClick={() => {
                    setTyped("");
                    pick(EMPTY);
                  }}
                >
                  {O.clear}
                </Button>
              )}
            </div>
          </form>

          {stale || loading ? (
            <LoadingState />
          ) : rows.length === 0 ? (
            <section className="surface flex flex-col items-center gap-3 p-8 text-center">
              <span className="inline-flex h-16 w-16 items-center justify-center rounded-full bg-success-solid text-on-solid">
                <IconBalance size={28} />
              </span>
              <h2 className="heading-section">{O.empty}</h2>
              <p className="max-w-md text-sm text-ink-muted">{qs ? O.emptyFiltered : O.emptyHint}</p>
            </section>
          ) : (
            <ul className="grid grid-cols-1 gap-3 lg:grid-cols-2">
              {rows.map((o) => {
                const paidPart = o.amount - o.outstanding;
                const pct = o.amount > 0 ? Math.round((paidPart / o.amount) * 100) : 0;
                return (
                  <li key={o.id} className="surface flex flex-col gap-3 p-4">
                    <div className="flex items-start gap-3">
                      <span
                        className={`inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-full ${
                          o.state === "open"
                            ? o.age_days >= alertDays
                              ? "bg-danger-solid text-on-solid"
                              : "bg-warning-solid text-on-solid"
                            : o.state === "paid"
                              ? "bg-success-solid text-on-solid"
                              : "bg-field text-ink-muted"
                        }`}
                      >
                        {o.party_kind === "merchant" ? <IconStore size={20} /> : <IconUser size={20} />}
                      </span>
                      <span className="min-w-0 flex-1">
                        <OpenLink
                          allowed={o.party_kind === "merchant" ? canOpen.merchant : canOpen.user}
                          href={partyHref(o)}
                          className="block truncate font-bold hover:underline"
                          title={O.openParty}
                        >
                          {o.party_name || "—"}
                        </OpenLink>
                        <span className="block truncate text-sm text-ink-muted">{causeText(o.cause)}</span>
                      </span>
                      <span className="flex shrink-0 flex-col items-end gap-1">
                        <Badge variant={STATE_VARIANT[o.state]}>{O.states[o.state]}</Badge>
                        <Badge variant="neutral">{O.kinds[o.party_kind]}</Badge>
                      </span>
                    </div>

                    <div className="space-y-2 rounded-control bg-field px-3 py-2.5">
                      <div className="flex items-end justify-between gap-2">
                        <span>
                          <span className="block text-2xs text-ink-muted">{O.outstanding}</span>
                          <span className={`figure block ${o.outstanding > 0 ? "text-warning" : "text-ink-muted"}`}>
                            {fmtMoney(o.outstanding)}
                          </span>
                        </span>
                        <span className="text-end text-sm">
                          <span className="block text-2xs text-ink-muted">{O.amount}</span>
                          <b className="tabular-nums">{fmtMoney(o.amount)}</b>
                        </span>
                      </div>
                      {o.amount > 0 && o.state === "open" && paidPart > 0 && (
                        <span className="block h-1.5 w-full overflow-hidden rounded-full bg-surface" aria-hidden>
                          <span className="block h-full rounded-full bg-success" style={{ width: `${pct}%` }} />
                        </span>
                      )}
                    </div>

                    <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-2xs text-ink-muted">
                      <span className="flex items-center gap-1">
                        <IconDate size={12} />
                        <span dir="ltr">{fmtDateTime(o.created_at)}</span>
                      </span>
                      <span className={o.state === "open" && o.age_days >= alertDays ? "font-bold text-danger" : ""}>
                        {O.ageDays.replace("{n}", fmtNum(o.age_days))}
                      </span>
                      {o.order_number != null && o.order_id && (
                        <OpenLink
                          allowed={canOpen.order}
                          href={`/dashboard/orders?id=${o.order_id}`}
                          dir="ltr"
                          title={O.openOrder}
                          className="flex items-center gap-1 hover:underline"
                        >
                          <IconOrder size={12} />#{fmtRef(o.order_number)}
                        </OpenLink>
                      )}
                      {!f.party && (
                        <button type="button" className="hover:underline" onClick={() => pick({ party: o.party_id })}>
                          {O.partyFilter}
                        </button>
                      )}
                    </div>

                    {/* **طلبٌ ينتظر** — ويوافق عليه غيرُ مقترِحه (والخادمُ يحكم). */}
                    {o.pending && (
                      <div className="space-y-2 rounded-control border border-accent px-3 py-2 text-sm">
                        <span className="block font-medium">
                          {o.pending.kind === "office_cash"
                            ? O.pendingCash.replace("{amount}", fmtMoney(o.pending.amount))
                            : O.pendingWriteOff}
                        </span>
                        {o.pending.note && <span className="block text-xs text-ink-muted">{o.pending.note}</span>}
                        <span className="grid grid-cols-2 gap-2">
                          {data?.can_manage &&
                            me?.id !== o.pending.proposed_by &&
                            (o.pending.kind === "office_cash" || data.can_approve_writeoff) && (
                              <Button className="w-full" onClick={() => decide(o.pending!, true)}>
                                {O.approve}
                              </Button>
                            )}
                          {data?.can_manage && (
                            <Button className="w-full" variant="danger" onClick={() => decide(o.pending!, false)}>
                              {O.reject}
                            </Button>
                          )}
                        </span>
                      </div>
                    )}

                    {/* **سطورُ التسوية بطريقتها** — انقطع من مستحق · دفع بالمكتب · انشطب. */}
                    {open === o.id && (
                      <ul className="space-y-1.5">
                        {o.settlements.map((st, i) => (
                          <li key={i} className="flex flex-wrap items-center justify-between gap-2 rounded-control bg-field px-3 py-2 text-sm">
                            <span className="min-w-0">
                              <span className="block font-medium">{O.methods[st.method] ?? st.method}</span>
                              <span className="block text-2xs text-ink-muted">
                                <span dir="ltr">{fmtDateTime(st.created_at)}</span>
                                {st.order_number != null && <span dir="ltr"> · #{fmtRef(st.order_number)}</span>}
                              </span>
                            </span>
                            <span className="text-end">
                              <Money
                                value={st.amount}
                                className={`font-bold tabular-nums ${
                                  st.method === "voided" || st.method === "written_off"
                                    ? "text-ink-muted line-through"
                                    : "text-success"
                                }`}
                              />
                              <span className="block text-2xs text-ink-muted">
                                {O.remaining} {fmtMoney(st.remaining)}
                              </span>
                            </span>
                          </li>
                        ))}
                      </ul>
                    )}

                    {(o.settlements.length > 0 || (o.state === "open" && !o.pending && data?.can_manage)) && (
                      <div className="mt-auto flex flex-wrap gap-2 border-t border-line-soft pt-3">
                        {o.state === "open" && !o.pending && data?.can_manage && (
                          <>
                            <Button className="flex-1" onClick={() => setAct({ o, kind: "office_cash" })}>
                              {O.officeCash}
                            </Button>
                            <Button className="flex-1" variant="secondary" onClick={() => setAct({ o, kind: "write_off" })}>
                              {O.writeOff}
                            </Button>
                          </>
                        )}
                        {o.settlements.length > 0 && (
                          <Button variant="ghost" onClick={() => setOpen((cur) => (cur === o.id ? null : o.id))}>
                            {O.history} ({fmtNum(o.settlements.length)})
                          </Button>
                        )}
                      </div>
                    )}
                  </li>
                );
              })}
            </ul>
          )}

          {data && data.total > data.per_page && (
            <div className="mt-4 flex justify-center">
              <Pagination
                page={page}
                total={data.total}
                perPage={data.per_page}
                onChange={(p) => {
                  setShownKey("");
                  setPage(p);
                }}
              />
            </div>
          )}
        </>
      )}

      {act && (
        <ActionModal
          o={act.o}
          kind={act.kind}
          onClose={() => setAct(null)}
          onDone={() => {
            setAct(null);
            setNotice(O.sent);
            reload();
          }}
        />
      )}
    </PageContainer>
  );
}

/** **نافذةُ الاقتراح** — مبلغٌ (للدفع) وسببٌ إلزاميّ. */
function ActionModal({
  o,
  kind,
  onClose,
  onDone,
}: {
  o: Obligation;
  kind: "office_cash" | "write_off";
  onClose: () => void;
  onDone: () => void;
}) {
  const [amount, setAmount] = useState(String(o.outstanding));
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");
  const cash = kind === "office_cash";

  async function send() {
    setBusy(true);
    setErr("");
    try {
      await api(`/api/v1/admin/obligations/${o.id}/${cash ? "office-cash" : "write-off"}`, {
        method: "POST",
        body: JSON.stringify(cash ? { amount: Number(amount), note } : { note }),
      });
      onDone();
    } catch (e) {
      setErr(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open onClose={onClose} title={cash ? O.officeCashTitle : O.writeOffTitle}>
      <div className="space-y-3">
        <p className="text-sm text-ink-muted">{cash ? O.officeCashHint : O.writeOffHint}</p>
        <p className="text-sm font-bold">
          {o.party_name} · <Money value={o.outstanding} className="tabular-nums" />
        </p>
        {cash && (
          <Input
            id="obl-amount"
            label={O.amountField}
            type="number"
            min={1}
            max={o.outstanding}
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
          />
        )}
        <Input id="obl-note" label={O.reasonField} value={note} onChange={(e) => setNote(e.target.value)} />
        {err && <Alert tone="error">{err}</Alert>}
        <div className="flex gap-2">
          <Button onClick={send} disabled={busy || !note.trim() || (cash && !(Number(amount) > 0))}>
            {O.send}
          </Button>
          <Button variant="secondary" onClick={onClose}>
            {O.cancel}
          </Button>
        </div>
      </div>
    </Modal>
  );
}
