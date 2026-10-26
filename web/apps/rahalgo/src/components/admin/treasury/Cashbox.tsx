"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **النقدُ والصندوق — صندوقُ المكتب والإغلاقُ اليوميّ** (قرارُ المالك ٢٠٢٦-١٠-٠٤)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **ما دخل الدرجَ وما خرج منه منذ آخر إغلاق**، و«إغلاقُ اليوم»: الموظّفُ يكتب ما
 * عدّه، والمحرّكُ يقارنه بالمتوقَّع ويُظهر الفرق، **وموظّفٌ آخرُ يراجع.** والنقصُ
 * يُسجَّل، ويُحَلّ إن وُجد المال، وإن بقي يوماً صار خسارةً بموافقة مدير المنصّة.
 *
 * # قراراتُ المالك ٢٠٢٦-١٠-١٥
 *
 * **رصيدُ البداية** مرّةً قبل أيّ إغلاق · **إيداعٌ بالدرج** من جيب المالك (ليس ربحاً، بكلمة
 * السرّ) · **الدرجُ قد يصير سالباً** ويظهر بالأحمر · **تقريرُ يومٍ وأسبوعٍ وشهر** · وزرُّ
 * «تأكيد المراجعة» **بحكم المحرّك** (`can_approve`) — المالكُ وحدَه يراجع إغلاقَه حين لا غيرُه.
 */

import { useState, type ReactNode } from "react";
import Link from "next/link";
import { fmtMoney, fmtDateTime, errorText } from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  Confirm,
  EmptyState,
  FormActions,
  Input,
  LoadingState,
  Modal,
  Pagination,
  ReloadState,
  StatCard,
  StatGrid,
  IconWallet,
  useLiveData,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useCanCall } from "@/lib/policy";
import { T, Signed, Amount } from "./shared";

const C = T.cashbox;

interface Position {
  opening: number;
  opened_at: string | null;
  cash_in: number;
  cash_out: number;
  expected: number;
  book: number;
  open_lines: number;
}
interface Entry {
  id: string;
  direction: "in" | "out";
  amount: number;
  source: string;
  ref: string;
  user_name: string;
  recorded_by: string;
  note: string;
  closed: boolean;
  created_at: string;
}
export interface CashClose {
  id: string;
  day: string;
  /** `opening` رصيدُ البداية، و`daily` إغلاقُ يوم. */
  kind?: "daily" | "opening";
  opening: number;
  cash_in: number;
  cash_out: number;
  expected: number;
  counted: number;
  difference: number;
  note: string;
  status: "pending" | "approved" | "rejected";
  proposed_by: string;
  proposer_name: string;
  decider_name: string | null;
  decided_at: string | null;
  decision_note: string;
  self_approved: boolean;
  created_at: string;
  shortfall_id: string | null;
  shortfall_status: "pending" | "approved" | "rejected" | "resolved" | null;
  shortfall_due_at: string | null;
  /** **أيستطيع هذا الموظّفُ مراجعتَه؟** — من المحرّك، للمعلَّق وحدَه. */
  can_approve?: boolean;
  self_approval?: boolean;
}

type Period = "day" | "week" | "month";
interface Report {
  period: Period;
  from: string;
  to: string;
  total_in: number;
  total_out: number;
  by_source: { source: string; direction: "in" | "out"; amount: number; count: number }[];
  close_difference: number;
  shortage: number;
  excess: number;
  closes: number;
  start_balance: number;
  end_balance: number;
}

function newKey(): string {
  return typeof crypto !== "undefined" && "randomUUID" in crypto ? crypto.randomUUID() : String(Date.now());
}

/** **تقريرُ الدرج لفترة** — ما دخل وما خرج بمصادره، وفروقُ الإغلاق، والرصيدُ أوّلاً وآخراً. */
function CashReport() {
  const [period, setPeriod] = useState<Period>("day");
  const { data, error, reload } = useLiveData<Report>(
    () => api(`/api/v1/admin/cashbox/report?period=${period}`),
    ["wallet"],
    [period],
  );
  const periods: Period[] = ["day", "week", "month"];
  return (
    <section className="space-y-2">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h3 className="font-bold">{C.reportTitle}</h3>
        <div className="flex gap-1.5" role="group" aria-label={C.reportTitle}>
          {periods.map((p) => (
            <Button key={p} variant={p === period ? "primary" : "secondary"} className="!px-3" onClick={() => setPeriod(p)}>
              {C.periods[p]}
            </Button>
          ))}
        </div>
      </div>
      {error && !data ? (
        <ReloadState onRetry={reload} />
      ) : !data ? (
        <LoadingState variant="stats" />
      ) : (
        <>
          <p className="text-xs text-ink-muted">
            {C.reportRange.replace("{from}", data.from).replace("{to}", data.to)}
          </p>
          <StatGrid>
            <StatCard
              label={C.reportStart}
              value={fmtMoney(data.start_balance)}
              tone={data.start_balance < 0 ? "danger" : undefined}
            />
            <StatCard label={C.reportIn} value={fmtMoney(data.total_in)} tone="success" />
            <StatCard label={C.reportOut} value={fmtMoney(data.total_out)} tone="danger" />
            <StatCard
              label={C.reportEnd}
              value={fmtMoney(data.end_balance)}
              tone={data.end_balance < 0 ? "danger" : undefined}
              emphasis
            />
            <StatCard
              label={C.reportDiff}
              value={fmtMoney(data.close_difference)}
              sub={`${C.reportShortage} ${fmtMoney(data.shortage)} · ${C.reportExcess} ${fmtMoney(data.excess)} · ${C.reportCloses} ${data.closes}`}
              tone={data.close_difference < 0 ? "danger" : data.close_difference > 0 ? "warning" : "muted"}
            />
          </StatGrid>
          {data.by_source.length === 0 ? (
            <p className="text-sm text-ink-muted">{C.reportEmpty}</p>
          ) : (
            <div className="surface overflow-x-auto">
              <table className="w-full text-sm">
                <caption className="p-2 text-start text-2xs text-ink-muted">{C.reportBySource}</caption>
                <tbody>
                  {data.by_source.map((x) => (
                    <tr key={`${x.direction}-${x.source}`} className="border-b border-line-soft">
                      <td className="p-2">{sourceLabel(x.source)}</td>
                      <td className="p-2 text-ink-muted" dir="ltr">
                        {x.count}
                      </td>
                      <td className="p-2">
                        <Signed value={x.direction === "in" ? x.amount : -x.amount} />
                      </td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}
    </section>
  );
}

function sourceLabel(s: string): string {
  return (C.sources as Record<string, string>)[s] ?? s;
}

/** **الفرقُ بكلمته** — زيادةٌ أو نقصٌ أو مطابق. */
export function DiffBadge({ diff }: { diff: number }) {
  if (diff === 0) return <Badge variant="success">{C.balanced}</Badge>;
  return (
    <Badge variant={diff < 0 ? "danger" : "warning"}>
      {diff < 0 ? C.short : C.over} <span dir="ltr" className="ms-1">{fmtMoney(Math.abs(diff))}</span>
    </Badge>
  );
}

export function CashboxTab() {
  const canCall = useCanCall();
  const canClose = canCall("POST", "/cashbox/closes");
  const canOpening = canCall("POST", "/cashbox/opening");
  const canDeposit = canCall("POST", "/cashbox/deposits");
  const [all, setAll] = useState(false);
  const [page, setPage] = useState(1);
  const box = useLiveData<{
    position: Position;
    entries: Entry[];
    total: number;
    per_page: number;
    pending_close: CashClose | null;
  }>(() => api(`/api/v1/admin/cashbox?page=${page}${all ? "&all=1" : ""}`), ["wallet"], [page, all]);
  const closes = useLiveData<{ closes: CashClose[] }>(() => api(`/api/v1/admin/cashbox/closes`), ["wallet"]);

  const [closing, setClosing] = useState(false);
  const [counted, setCounted] = useState("");
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [fail, setFail] = useState("");
  const [opening, setOpening] = useState(false);
  const [openingCounted, setOpeningCounted] = useState("");
  const [deposit, setDeposit] = useState(false);
  const [depositAmount, setDepositAmount] = useState("");
  const [depositNote, setDepositNote] = useState("");
  const [depositKey, setDepositKey] = useState(newKey);
  const [done, setDone] = useState("");
  const [rejectClose, setRejectClose] = useState<CashClose | null>(null);
  const [resolve, setResolve] = useState<CashClose | null>(null);
  const [reason, setReason] = useState("");

  function reloadAll() {
    box.reload();
    closes.reload();
  }

  async function post(path: string, body: unknown, headers?: Record<string, string>) {
    setBusy(true);
    try {
      await api(path, { method: "POST", body: JSON.stringify(body), headers });
      setFail("");
      reloadAll();
      return true;
    } catch (err) {
      setFail(errorText(err));
      return false;
    } finally {
      setBusy(false);
    }
  }

  if (box.error && !box.data) return <ReloadState onRetry={reloadAll} />;
  if (!box.data) return <LoadingState variant="stats" />;
  const pos = box.data.position;
  const pending = box.data.pending_close;
  const countedNum = Number(counted);
  const preview = counted.trim() === "" || Number.isNaN(countedNum) ? null : countedNum - pos.expected;
  // **رصيدُ البداية قبل أيّ إغلاقٍ مُعتمَد** — والمحرّكُ يردّ غيرَه (`cash_opening_late`).
  const showOpening = canOpening && !pending && pos.opened_at === null;
  const openingNum = Number(openingCounted);
  const depositNum = Number(depositAmount);

  return (
    <div className="space-y-4">
      {fail && <Alert onDismiss={() => setFail("")}>{fail}</Alert>}
      {done && (
        <Alert tone="success" onDismiss={() => setDone("")}>
          {done}
        </Alert>
      )}
      {/* **الدرجُ بمعادلةٍ واحدة** (قرارُ المالك ٢٠٢٦-١٠-٢٦: «صلّح الأرقام لتكون واضحة ونصلّح التصميم»):
          رصيدُ الفتح + الداخل − الخارج = المفروض بالدرج هلّق، **والأزرارُ تحتها بسطرٍ واحد.** */}
      <section className="surface space-y-3 p-4">
        <div className="flex flex-wrap items-stretch gap-2">
          <BoxCell label={C.opening} value={pos.opening} sub={pos.opened_at ? fmtDateTime(pos.opened_at) : undefined} />
          <BoxOp>+</BoxOp>
          <BoxCell label={C.in} value={pos.cash_in} tone="success" />
          <BoxOp>−</BoxOp>
          <BoxCell label={C.out} value={pos.cash_out} tone="danger" />
          <BoxOp>=</BoxOp>
          <BoxCell label={C.expected} value={pos.expected} strong tone={pos.expected < 0 ? "danger" : "success"} sub={pos.expected < 0 ? C.negative : undefined} />
        </div>
        <p className="text-xs text-ink-muted">{C.howTo}</p>
        <div className="flex flex-wrap items-center gap-2 border-t border-line-soft pt-3">
          {canClose && !pending && (
            <Button onClick={() => setClosing(true)} disabled={busy}>
              {C.closeDay}
            </Button>
          )}
          {showOpening && (
            <Button variant="secondary" onClick={() => setOpening(true)} disabled={busy}>
              {C.openingButton}
            </Button>
          )}
          {canDeposit && (
            <Button variant="secondary" onClick={() => setDeposit(true)} disabled={busy}>
              {C.depositButton}
            </Button>
          )}
          <Link href="/dashboard/cash" className="ms-auto text-sm font-bold text-accent-text hover:underline">
            {C.cashPage} ↗
          </Link>
          <span className="text-xs text-ink-muted">
            {C.openLines}: <span dir="ltr">{pos.open_lines}</span>
          </span>
        </div>
      </section>

      {pending && (
        <div className="surface space-y-2 p-3">
          <div className="flex flex-wrap items-center gap-2">
            <Badge variant="warning">{C.pendingClose}</Badge>
            <span className="text-sm">
              {C.colExpected}: <Amount value={pending.expected} /> · {C.colCounted}: <Amount value={pending.counted} />
            </span>
            <DiffBadge diff={pending.difference} />
            <span className="text-xs text-ink-muted">
              {C.colBy}: {pending.proposer_name} · <span dir="ltr">{fmtDateTime(pending.created_at)}</span>
            </span>
          </div>
          {pending.note && <p className="text-sm text-ink-muted">{pending.note}</p>}
          {canClose && (
            <div className="flex flex-wrap gap-2">
              {pending.can_approve ? (
                <>
                  <Button disabled={busy} onClick={() => void post(`/api/v1/admin/cashbox/closes/${pending.id}/approve`, {})}>
                    {C.approveClose}
                  </Button>
                  {pending.self_approval && <span className="text-xs text-warning">{C.selfReview}</span>}
                </>
              ) : (
                <span className="text-xs text-ink-muted">{C.ownClose}</span>
              )}
              <Button variant="secondary" disabled={busy} onClick={() => setRejectClose(pending)}>
                {C.rejectClose}
              </Button>
            </div>
          )}
        </div>
      )}

      <CashReport />

      {/* **حركاتُ الدرج بطاقات** — داخلٌ بالأخضر وخارجٌ بالأحمر، ومن سلّم أو استلم وسببُه. */}
      <section className="surface space-y-3 p-4">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h3 className="heading-card">{C.entries}</h3>
          <Button
            variant="secondary"
            onClick={() => {
              setAll(!all);
              setPage(1);
            }}
          >
            {all ? C.showOpen : C.showAll}
          </Button>
        </div>
        {box.data.entries.length === 0 ? (
          <EmptyState icon={IconWallet} title={C.noEntries} />
        ) : (
          <ul className="space-y-2">
            {box.data.entries.map((e) => {
              const inn = e.direction === "in";
              return (
                <li key={e.id} className="flex items-center gap-3 rounded-card border border-line-soft p-3">
                  <span
                    className={`inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-on-solid ${
                      inn ? "bg-success-solid" : "bg-danger-solid"
                    }`}
                  >
                    {inn ? "▲" : "▼"}
                  </span>
                  <span className="min-w-0 flex-1">
                    <span className="block font-bold">
                      {sourceLabel(e.source)}
                      {e.user_name && <span className="font-normal text-ink-muted"> · {e.user_name}</span>}
                    </span>
                    <span className="block truncate text-xs text-ink-muted">
                      <span dir="ltr">{fmtDateTime(e.created_at)}</span>
                      {e.recorded_by && ` · ${T.statement.colBy}: ${e.recorded_by}`}
                      {e.note && ` · ${e.note}`}
                    </span>
                  </span>
                  <span className={`figure shrink-0 leading-none ${inn ? "text-success" : "text-danger"}`}>
                    <Signed value={inn ? e.amount : -e.amount} />
                  </span>
                </li>
              );
            })}
          </ul>
        )}
        <Pagination page={page} total={box.data.total} perPage={box.data.per_page} onChange={setPage} />
      </section>

      {/* **إغلاقاتُ الصندوق بطاقات** — اليومُ، والمفروضُ والمعدودُ والفرق، والحالُ ومن أغلق ومن راجع. */}
      <section className="surface space-y-3 p-4">
        <h3 className="heading-card">{C.closes}</h3>
        {!closes.data ? (
          <LoadingState variant="text" />
        ) : closes.data.closes.length === 0 ? (
          <EmptyState icon={IconWallet} title={C.noCloses} />
        ) : (
          <div className="grid grid-cols-1 gap-2 lg:grid-cols-2">
            {closes.data.closes.map((c) => (
              <div key={c.id} className="space-y-2 rounded-card border border-line-soft p-3 text-sm">
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <span className="font-bold" dir="ltr">{c.day}</span>
                  <span className="flex flex-wrap items-center gap-1">
                    {c.kind === "opening" ? (
                      <Badge variant="neutral">{C.openingBadge}</Badge>
                    ) : (
                      <DiffBadge diff={c.difference} />
                    )}
                    <Badge variant={c.status === "approved" ? "success" : c.status === "rejected" ? "danger" : "warning"}>
                      {C.statuses[c.status]}
                    </Badge>
                    {c.self_approved && <span className="text-2xs text-warning">{T.approvals.selfFlag}</span>}
                  </span>
                </div>
                <div className="grid grid-cols-2 gap-2">
                  <div className="rounded-control bg-field px-2.5 py-1.5">
                    <span className="block text-2xs text-ink-muted">{C.colExpected}</span>
                    <Amount value={c.expected} />
                  </div>
                  <div className="rounded-control bg-field px-2.5 py-1.5">
                    <span className="block text-2xs text-ink-muted">{C.colCounted}</span>
                    <Amount value={c.counted} />
                  </div>
                </div>
                <p className="text-xs text-ink-muted">
                  {C.colBy}: {c.proposer_name}
                  {c.decider_name && ` · ${C.colReviewer}: ${c.decider_name}`}
                </p>
                {c.shortfall_status && (
                  <div className="flex flex-wrap items-center gap-1 border-t border-line-soft pt-2">
                    <span className="text-xs text-ink-muted">{C.shortfall}:</span>
                    <Badge variant={c.shortfall_status === "pending" ? "warning" : "neutral"}>
                      {C.shortfallStatuses[c.shortfall_status]}
                    </Badge>
                    {c.shortfall_status === "pending" && c.shortfall_due_at && (
                      <span className="text-2xs text-ink-muted">
                        {C.dueAt} <span dir="ltr">{fmtDateTime(c.shortfall_due_at)}</span>
                      </span>
                    )}
                    {c.shortfall_status === "pending" && canClose && (
                      <Button variant="secondary" disabled={busy} onClick={() => setResolve(c)} className="ms-auto">
                        {C.resolve}
                      </Button>
                    )}
                  </div>
                )}
              </div>
            ))}
          </div>
        )}
      </section>

      <Modal open={closing} onClose={() => setClosing(false)} title={C.closeDay}>
        <div className="space-y-3">
          <p className="text-sm text-ink-muted">{C.closeHint}</p>
          <p className="text-sm">
            {C.expected}: <Amount value={pos.expected} />
          </p>
          <Input
            id="cb-counted"
            type="number"
            inputMode="numeric"
            dir="ltr"
            label={C.counted}
            value={counted}
            onChange={(e) => setCounted(e.target.value)}
          />
          {preview !== null && (
            <p className="text-sm">
              {C.difference}: <DiffBadge diff={preview} />
            </p>
          )}
          <Input id="cb-note" label={C.closeNote} value={note} onChange={(e) => setNote(e.target.value)} />
          <FormActions
            busy={busy || preview === null || countedNum < 0}
            saveLabel={C.closeDay}
            onCancel={() => setClosing(false)}
            onSave={() => {
              void post(`/api/v1/admin/cashbox/closes`, { counted: Math.round(countedNum), note: note.trim() }).then(
                (ok) => {
                  if (!ok) return;
                  setClosing(false);
                  setCounted("");
                  setNote("");
                },
              );
            }}
          />
        </div>
      </Modal>

      <Modal open={opening} onClose={() => setOpening(false)} title={C.openingTitle}>
        <div className="space-y-3">
          <p className="text-sm text-ink-muted">{C.openingHint}</p>
          <Input
            id="cb-opening"
            type="number"
            inputMode="numeric"
            dir="ltr"
            label={C.openingCounted}
            value={openingCounted}
            onChange={(e) => setOpeningCounted(e.target.value)}
          />
          <FormActions
            busy={busy || openingCounted.trim() === "" || Number.isNaN(openingNum) || openingNum < 0}
            saveLabel={C.openingButton}
            onCancel={() => setOpening(false)}
            onSave={() => {
              void post(`/api/v1/admin/cashbox/opening`, { counted: Math.round(openingNum) }).then((ok) => {
                if (!ok) return;
                setOpening(false);
                setOpeningCounted("");
              });
            }}
          />
        </div>
      </Modal>

      <Modal open={deposit} onClose={() => setDeposit(false)} title={C.depositTitle}>
        <div className="space-y-3">
          <p className="text-sm text-ink-muted">{C.depositHint}</p>
          <Input
            id="cb-deposit-amount"
            type="number"
            inputMode="numeric"
            dir="ltr"
            label={C.depositAmount}
            value={depositAmount}
            onChange={(e) => setDepositAmount(e.target.value)}
          />
          <Input id="cb-deposit-note" label={C.depositNote} value={depositNote} onChange={(e) => setDepositNote(e.target.value)} />
          <FormActions
            busy={busy || !(depositNum > 0) || depositNote.trim() === ""}
            saveLabel={C.depositButton}
            onCancel={() => setDeposit(false)}
            onSave={() => {
              void post(
                `/api/v1/admin/cashbox/deposits`,
                { amount: Math.round(depositNum), note: depositNote.trim() },
                { "Idempotency-Key": depositKey },
              ).then((ok) => {
                if (!ok) return;
                setDeposit(false);
                setDepositAmount("");
                setDepositNote("");
                setDepositKey(newKey());
                setDone(C.depositDone);
              });
            }}
          />
        </div>
      </Modal>

      <Confirm
        open={rejectClose !== null}
        title={C.rejectClose}
        body={<Input id="cb-reject" label={C.rejectReason} value={reason} onChange={(e) => setReason(e.target.value)} />}
        confirmLabel={C.rejectClose}
        busy={busy}
        onConfirm={() => {
          const c = rejectClose;
          setRejectClose(null);
          if (c) void post(`/api/v1/admin/cashbox/closes/${c.id}/reject`, { note: reason.trim() }).then(() => setReason(""));
        }}
        onCancel={() => setRejectClose(null)}
      />
      <Confirm
        open={resolve !== null}
        tone="primary"
        title={C.resolve}
        body={<Input id="cb-resolve" label={C.resolveNote} value={reason} onChange={(e) => setReason(e.target.value)} />}
        confirmLabel={C.resolve}
        busy={busy}
        onConfirm={() => {
          const c = resolve;
          setResolve(null);
          if (c?.shortfall_id)
            void post(`/api/v1/admin/cashbox/shortfalls/${c.shortfall_id}/resolve`, { note: reason.trim() }).then(() =>
              setReason(""),
            );
        }}
        onCancel={() => setResolve(null)}
      />
    </div>
  );
}

/** **خانةٌ في معادلة الدرج** — لفظٌ ورقمٌ وسطرٌ صغير. */
function BoxCell({
  label,
  value,
  sub,
  tone,
  strong,
}: {
  label: string;
  value: number;
  sub?: string;
  tone?: "success" | "danger";
  strong?: boolean;
}) {
  return (
    <div className={`min-w-[7rem] flex-1 rounded-card border p-3 ${strong ? "border-accent" : "border-line-soft"}`}>
      <span className="block text-xs text-ink-muted">{label}</span>
      <span className={`figure block leading-tight ${tone === "success" ? "text-success" : tone === "danger" ? "text-danger" : ""}`}>
        {fmtMoney(value)}
      </span>
      {sub && <span className="block text-2xs text-ink-muted">{sub}</span>}
    </div>
  );
}

function BoxOp({ children }: { children: ReactNode }) {
  return <span className="figure flex items-center px-1 text-ink-muted">{children}</span>;
}
