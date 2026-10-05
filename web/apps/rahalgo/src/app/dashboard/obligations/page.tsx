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
import Link from "next/link";
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
  PageHeader,
  Pagination,
  EmptyState,
  LoadingState,
  ReloadState,
  StatGrid,
  StatCard,
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

const m = getMessages(defaultLocale);
const O = m.admin.obligations;

type Kind = "" | "merchant" | "rep";
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
  party_kind: "merchant" | "rep";
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
}

const EMPTY: Filters = { kind: "", state: "", cause: "", q: "", party: "", from: "", to: "", overdue: false };

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
  const unit = ` (${m.common.currency})`;

  return (
    <PageContainer width="full">
      <PageHeader
        icon={IconBalance}
        title={O.title}
        subtitle={O.hint}
        actions={
          <div className="flex flex-wrap gap-2">
            <ButtonLink variant="secondary" href="/dashboard/treasury">
              {O.treasuryLink}
            </ButtonLink>
            {data?.can_export && (
              <Button variant="secondary" onClick={exportCsv} disabled={exporting}>
                {exporting ? O.exporting : O.export}
              </Button>
            )}
          </div>
        }
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
          <StatGrid>
            <StatCard
              label={O.outstandingTotal + unit}
              value={fmtNum(sum?.outstanding_total ?? 0)}
              icon={IconWallet}
              tone={(sum?.outstanding_total ?? 0) > 0 ? "accent" : "default"}
            />
            <StatCard label={O.merchantsTotal + unit} value={fmtNum(sum?.merchants_total ?? 0)} icon={IconStore} />
            <StatCard label={O.repsTotal + unit} value={fmtNum(sum?.reps_total ?? 0)} icon={IconUser} />
            <StatCard label={O.debtors} value={fmtNum(sum?.debtors ?? 0)} icon={IconBalance} />
            <StatCard label={O.nearLimit} value={fmtNum(sum?.near_limit ?? 0)} icon={IconStore} />
            <StatCard
              label={O.overdue.replace("{n}", fmtNum(data?.alert_days ?? 30))}
              value={fmtNum(sum?.overdue ?? 0)}
              icon={IconDate}
              tone={(sum?.overdue ?? 0) > 0 ? "accent" : "default"}
            />
            <StatCard label={O.pendingRequests} value={fmtNum(sum?.pending_requests ?? 0)} icon={IconStatus} />
          </StatGrid>

          <form
            className="mb-4 mt-4 flex flex-wrap items-end gap-3"
            onSubmit={(e) => {
              e.preventDefault();
              pick({ q: typed });
            }}
          >
            <div className="w-56">
              <Input
                value={typed}
                placeholder={O.search}
                onChange={(e) => setTyped(e.target.value)}
                onBlur={() => typed !== f.q && pick({ q: typed })}
              />
            </div>
            <div className="w-40">
              <Select value={f.kind} onChange={(e) => pick({ kind: e.target.value as Kind })}>
                <option value="">{O.kinds.all}</option>
                <option value="merchant">{O.kinds.merchant}</option>
                <option value="rep">{O.kinds.rep}</option>
              </Select>
            </div>
            <div className="w-48">
              <Select value={f.state} onChange={(e) => pick({ state: e.target.value as State })}>
                <option value="">{O.states.all}</option>
                <option value="open">{O.states.open}</option>
                <option value="paid">{O.states.paid}</option>
                <option value="voided">{O.states.voided}</option>
                <option value="written_off">{O.states.written_off}</option>
              </Select>
            </div>
            <div className="w-52">
              <Select value={f.cause} onChange={(e) => pick({ cause: e.target.value })}>
                <option value="">{O.causeAll}</option>
                {Object.entries(O.causes).map(([k, v]) => (
                  <option key={k} value={k}>
                    {v}
                  </option>
                ))}
              </Select>
            </div>
            <div className="w-40">
              <Input type="date" label={O.from} value={f.from} onChange={(e) => pick({ from: e.target.value })} />
            </div>
            <div className="w-40">
              <Input type="date" label={O.to} value={f.to} onChange={(e) => pick({ to: e.target.value })} />
            </div>
            <Button
              type="button"
              variant={f.overdue ? "primary" : "secondary"}
              onClick={() => pick({ overdue: !f.overdue })}
            >
              {O.overdueOnly}
            </Button>
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
          </form>

          {stale || loading ? (
            <LoadingState />
          ) : rows.length === 0 ? (
            <EmptyState icon={IconBalance} title={O.empty} />
          ) : (
            <ul className="space-y-2">
              {rows.map((o) => (
                <li key={o.id} className="surface p-3">
                  <div className="flex flex-wrap items-center gap-3">
                    <span className="min-w-0 flex-1">
                      <span className="flex flex-wrap items-center gap-1.5 font-bold">
                        {o.party_kind === "merchant" ? <IconStore size={15} /> : <IconUser size={15} />}
                        <Link href={partyHref(o)} className="hover:underline" title={O.openParty}>
                          {o.party_name || "—"}
                        </Link>
                        <Badge variant="neutral">{O.kinds[o.party_kind]}</Badge>
                        {!f.party && (
                          <button
                            type="button"
                            className="text-2xs font-normal text-ink-muted hover:underline"
                            onClick={() => pick({ party: o.party_id })}
                          >
                            {O.partyFilter}
                          </button>
                        )}
                      </span>
                      <span className="block text-sm text-ink-muted">{causeText(o.cause)}</span>
                      <span className="flex items-center gap-1 text-xs text-ink-muted">
                        <IconDate size={13} />
                        <span dir="ltr">{fmtDateTime(o.created_at)}</span>
                        <span>· {O.ageDays.replace("{n}", fmtNum(o.age_days))}</span>
                      </span>
                    </span>

                    {o.order_number != null && o.order_id && (
                      <Link
                        href={`/dashboard/orders?id=${o.order_id}`}
                        dir="ltr"
                        title={O.openOrder}
                        className="flex shrink-0 items-center gap-1 text-sm text-ink-muted hover:underline"
                      >
                        <IconOrder size={14} />#{fmtRef(o.order_number)}
                      </Link>
                    )}

                    <span className="shrink-0 text-end">
                      <span className="block text-2xs text-ink-muted">{O.amount}</span>
                      <Money value={o.amount} className="text-ink-muted tabular-nums" />
                    </span>
                    <span className="shrink-0 text-end">
                      <span className="block text-2xs text-ink-muted">{O.outstanding}</span>
                      <Money
                        value={o.outstanding}
                        className={`font-bold tabular-nums ${o.outstanding > 0 ? "text-warning" : "text-ink-muted"}`}
                      />
                    </span>

                    <Badge variant={STATE_VARIANT[o.state]}>{O.states[o.state]}</Badge>

                    {o.settlements.length > 0 && (
                      <Button variant="secondary" onClick={() => setOpen((cur) => (cur === o.id ? null : o.id))}>
                        {O.history} ({fmtNum(o.settlements.length)})
                      </Button>
                    )}
                    {o.state === "open" && !o.pending && data?.can_manage && (
                      <>
                        <Button variant="secondary" onClick={() => setAct({ o, kind: "office_cash" })}>
                          {O.officeCash}
                        </Button>
                        <Button variant="ghost" onClick={() => setAct({ o, kind: "write_off" })}>
                          {O.writeOff}
                        </Button>
                      </>
                    )}
                  </div>

                  {/* **طلبٌ ينتظر** — ويوافق عليه غيرُ مقترِحه (والخادمُ يحكم). */}
                  {o.pending && (
                    <div className="surface-inset mt-3 flex flex-wrap items-center gap-3 px-3 py-2 text-sm">
                      <span className="flex-1">
                        {o.pending.kind === "office_cash"
                          ? O.pendingCash.replace("{amount}", fmtMoney(o.pending.amount))
                          : O.pendingWriteOff}
                        <span className="block text-xs text-ink-muted">{o.pending.note}</span>
                      </span>
                      {data?.can_manage &&
                        me?.id !== o.pending.proposed_by &&
                        (o.pending.kind === "office_cash" || data.can_approve_writeoff) && (
                          <Button onClick={() => decide(o.pending!, true)}>{O.approve}</Button>
                        )}
                      {data?.can_manage && (
                        <Button variant="secondary" onClick={() => decide(o.pending!, false)}>
                          {O.reject}
                        </Button>
                      )}
                    </div>
                  )}

                  {/* **سطورُ التسوية بطريقتها** — انقطع من مستحق · دفع بالمكتب · انشطب. */}
                  {open === o.id && (
                    <ul className="surface-inset mt-3 divide-y divide-line-soft">
                      {o.settlements.map((st, i) => (
                        <li key={i} className="flex flex-wrap items-center justify-between gap-2 px-3 py-2 text-sm">
                          <span className="flex items-center gap-2">
                            <IconStatus size={13} />
                            <span dir="ltr">{fmtDateTime(st.created_at)}</span>
                            <span className="text-ink-muted">{O.methods[st.method] ?? st.method}</span>
                            {st.order_number != null && (
                              <span dir="ltr" className="flex items-center gap-1 text-ink-muted">
                                <IconOrder size={13} />#{fmtRef(st.order_number)}
                              </span>
                            )}
                          </span>
                          <span className="flex items-center gap-4">
                            <span>
                              <Money
                                value={st.amount}
                                className={`tabular-nums ${
                                  st.method === "voided" || st.method === "written_off"
                                    ? "text-ink-muted line-through"
                                    : "text-success"
                                }`}
                              />
                            </span>
                            <span>
                              <span className="text-2xs text-ink-muted">{O.remaining} </span>
                              <Money value={st.remaining} className="tabular-nums text-ink-muted" />
                            </span>
                          </span>
                        </li>
                      ))}
                    </ul>
                  )}
                </li>
              ))}
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
