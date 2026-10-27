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

import { useState } from "react";
import Link from "next/link";
import { fmtMoney, fmtNum, fmtDateTime, errorText } from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  Chips,
  FormActions,
  Input,
  LoadingState,
  Modal,
  Pagination,
  ReloadState,
  IconBalance,
  IconArrowIn,
  IconArrowOut,
  useLiveData,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useCanCall } from "@/lib/policy";
import { T, Signed, Amount, Equation, EmptyCard, fmtDay, dayStartsLine } from "./shared";

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
  /** **ساعةُ بدء يوم الحساب** (`platform.day_start_hour`). */
  day_start_hour?: number;
}

function newKey(): string {
  return typeof crypto !== "undefined" && "randomUUID" in crypto ? crypto.randomUUID() : String(Date.now());
}

/**
 * **تقريرُ الدرج لفترة** — ما دخل وما طلع بمصادره، وفروقُ الإغلاق، والرصيدُ أوّلاً وآخراً.
 *
 * (تدقيقُ الخزينة ٢٠٢٦-١٠-٢٧) **بمعادلةٍ تُقرأ**: البدايةُ + الداخل − الطالع ± فروقُ الإغلاق = النهاية —
 * وفروقُ الإغلاق فيها هي ما يُكمل المعادلةَ من أرقام المحرّك نفسِها (الإغلاقاتُ المعتمَدةُ في الفترة)،
 * والمصادرُ صفوفٌ بأيقونة داخلٍ أو طالع بدل الجدول.
 */
function CashReport() {
  const [period, setPeriod] = useState<Period>("day");
  const { data, error, reload } = useLiveData<Report>(
    () => api(`/api/v1/admin/cashbox/report?period=${period}`),
    ["wallet"],
    [period],
  );
  const periods: Period[] = ["day", "week", "month"];
  const settled = data ? data.end_balance - data.start_balance - data.total_in + data.total_out : 0;
  return (
    <section className="surface space-y-3 p-4">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <h3 className="heading-card">{C.reportTitle}</h3>
        <Chips items={periods.map((p) => ({ id: p, label: C.periods[p] }))} value={period} onChange={setPeriod} />
      </div>
      {error && !data ? (
        <ReloadState onRetry={reload} />
      ) : !data ? (
        <LoadingState variant="stats" />
      ) : (
        <>
          <p className="flex flex-wrap items-center gap-x-2 gap-y-1 text-xs text-ink-muted">
            <span className="font-bold text-ink">
              {data.from === data.to
                ? fmtDay(data.to)
                : C.reportRange.replace("{from}", fmtDay(data.from)).replace("{to}", fmtDay(data.to))}
            </span>
            <span>{dayStartsLine(data.day_start_hour)}</span>
          </p>
          <Equation
            terms={[
              { label: C.reportStart, value: data.start_balance, tone: data.start_balance < 0 ? "danger" : undefined },
              { label: C.reportIn, value: data.total_in, tone: "success", op: "+" },
              { label: C.reportOut, value: data.total_out, tone: "danger", op: "−" },
              {
                label: C.reportDiff,
                value: settled,
                op: "±",
                signed: true,
                tone: settled < 0 ? "danger" : settled > 0 ? "warning" : undefined,
                sub: C.reportDiffSub
                  .replace("{short}", fmtNum(data.shortage))
                  .replace("{over}", fmtNum(data.excess))
                  .replace("{n}", fmtNum(data.closes)),
              },
              {
                label: C.reportEnd,
                value: data.end_balance,
                op: "=",
                result: true,
                tone: data.end_balance < 0 ? "danger" : "success",
              },
            ]}
          />
          <p className="text-2xs text-ink-muted">{C.reportEq}</p>
          {data.by_source.length === 0 ? (
            <EmptyCard icon={IconBalance} title={C.reportEmpty} tone="muted" bare />
          ) : (
            <div className="space-y-1.5">
              <p className="text-xs font-bold text-ink-muted">{C.reportBySource}</p>
              <ul className="grid grid-cols-1 gap-2 lg:grid-cols-2">
                {data.by_source.map((x) => {
                  const inn = x.direction === "in";
                  return (
                    <li key={`${x.direction}-${x.source}`} className="flex items-center gap-3 rounded-card border border-line-soft p-3">
                      <span
                        className={`inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full text-on-solid ${
                          inn ? "bg-success-solid" : "bg-danger-solid"
                        }`}
                      >
                        {inn ? <IconArrowIn size={16} /> : <IconArrowOut size={16} />}
                      </span>
                      <span className="min-w-0 flex-1">
                        <span className="block truncate font-bold">{sourceLabel(x.source)}</span>
                        <span className="block text-2xs text-ink-muted">{C.reportCount.replace("{n}", fmtNum(x.count))}</span>
                      </span>
                      <span className="figure shrink-0 leading-none">
                        <Signed value={inn ? x.amount : -x.amount} />
                      </span>
                    </li>
                  );
                })}
              </ul>
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
  const [reasonFail, setReasonFail] = useState("");

  function reloadAll() {
    box.reload();
    closes.reload();
  }

  /** `sink` — **حيث يُكتب الخطأ**: داخل النافذة المفتوحة، أو أعلى الصفحة. */
  async function post(path: string, body: unknown, headers?: Record<string, string>, sink: (m: string) => void = setFail) {
    setBusy(true);
    try {
      await api(path, { method: "POST", body: JSON.stringify(body), headers });
      setFail("");
      reloadAll();
      return true;
    } catch (err) {
      sink(errorText(err));
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
        <Equation
          terms={[
            { label: C.opening, value: pos.opening, sub: pos.opened_at ? fmtDateTime(pos.opened_at) : undefined },
            { label: C.in, value: pos.cash_in, tone: "success", op: "+" },
            { label: C.out, value: pos.cash_out, tone: "danger", op: "−" },
            {
              label: C.expected,
              value: pos.expected,
              op: "=",
              result: true,
              tone: pos.expected < 0 ? "danger" : "success",
              sub: pos.expected < 0 ? C.negative : undefined,
            },
          ]}
        />
        <p className="text-xs text-ink-muted">{C.howTo}</p>
        <div className="flex flex-wrap items-center gap-2 border-t border-line-soft pt-3">
          {canClose && !pending && (
            <Button
              onClick={() => {
                setReasonFail("");
                setClosing(true);
              }}
              disabled={busy}
            >
              {C.closeDay}
            </Button>
          )}
          {showOpening && (
            <Button
              variant="secondary"
              onClick={() => {
                setReasonFail("");
                setOpening(true);
              }}
              disabled={busy}
            >
              {C.openingButton}
            </Button>
          )}
          {canDeposit && (
            <Button
              variant="secondary"
              onClick={() => {
                setReasonFail("");
                setDeposit(true);
              }}
              disabled={busy}
            >
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
              <Button
                variant="secondary"
                disabled={busy}
                onClick={() => {
                  setReason("");
                  setReasonFail("");
                  setRejectClose(pending);
                }}
              >
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
          {/* **حبّتان لا زرٌّ يقلب اسمَه** — المختارةُ ملوّنة (تدقيقُ الخزينة ٢٠٢٦-١٠-٢٧). */}
          <Chips
            items={[
              { id: "open", label: C.showOpen },
              { id: "all", label: C.showAll },
            ]}
            value={all ? "all" : "open"}
            onChange={(k) => {
              setAll(k === "all");
              setPage(1);
            }}
          />
        </div>
        {box.data.entries.length === 0 ? (
          <EmptyCard icon={IconBalance} title={C.noEntries} hint={C.noEntriesHint} tone="muted" bare />
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
                    {inn ? <IconArrowIn size={16} /> : <IconArrowOut size={16} />}
                  </span>
                  <span className="min-w-0 flex-1">
                    <span className="block font-bold">
                      {sourceLabel(e.source)}
                      {e.user_name && <span className="font-normal text-ink-muted"> · {e.user_name}</span>}
                    </span>
                    <span className="block truncate text-xs text-ink-muted">
                      <span dir="ltr">{fmtDateTime(e.created_at)}</span>
                      {e.recorded_by && ` · ${T.statement.colBy}: ${e.recorded_by}`}
                      {e.note && e.source !== "advance_return" && ` · ${e.note}`}
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
        {box.data.total > box.data.per_page && (
          <div className="flex justify-center">
            <Pagination page={page} total={box.data.total} perPage={box.data.per_page} onChange={setPage} />
          </div>
        )}
      </section>

      {/* **إغلاقاتُ الصندوق بطاقات** — اليومُ، والمفروضُ والمعدودُ والفرق، والحالُ ومن أغلق ومن راجع. */}
      <section className="surface space-y-3 p-4">
        <h3 className="heading-card">{C.closes}</h3>
        {closes.error && !closes.data ? (
          <ReloadState onRetry={closes.reload} />
        ) : !closes.data ? (
          <LoadingState variant="text" />
        ) : closes.data.closes.length === 0 ? (
          <EmptyCard icon={IconBalance} title={C.noCloses} hint={C.noClosesHint} tone="muted" bare />
        ) : (
          <div className="grid grid-cols-1 gap-2 lg:grid-cols-2">
            {closes.data.closes.map((c) => (
              <div key={c.id} className="space-y-2 rounded-card border border-line-soft p-3 text-sm">
                <div className="flex flex-wrap items-center justify-between gap-2">
                  <span className="font-bold">{fmtDay(c.day)}</span>
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
                      <Button
                        variant="secondary"
                        disabled={busy}
                        onClick={() => {
                          setReason("");
                          setReasonFail("");
                          setResolve(c);
                        }}
                        className="ms-auto"
                      >
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
          {reasonFail && <Alert>{reasonFail}</Alert>}
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
              void post(
                `/api/v1/admin/cashbox/closes`,
                { counted: Math.round(countedNum), note: note.trim() },
                undefined,
                setReasonFail,
              ).then(
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
          {reasonFail && <Alert>{reasonFail}</Alert>}
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
              void post(`/api/v1/admin/cashbox/opening`, { counted: Math.round(openingNum) }, undefined, setReasonFail).then((ok) => {
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
          {reasonFail && <Alert>{reasonFail}</Alert>}
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
                setReasonFail,
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

      {/* **رفضُ الإغلاق و«وُجد المال» بسببٍ لازم** (تدقيقُ الخزينة ٢٠٢٦-١٠-٢٧) — المحرّكُ يردّ السببَ
          الفارغ، فالزرُّ معطّلٌ حتّى يُكتب، والنافذةُ تبقى مفتوحةً والخطأُ داخلها حتّى ينجح. */}
      <Modal open={rejectClose !== null} onClose={() => setRejectClose(null)} title={C.rejectClose}>
        <div className="space-y-3">
          {reasonFail && <Alert>{reasonFail}</Alert>}
          <Input id="cb-reject" label={C.rejectReason} value={reason} onChange={(e) => setReason(e.target.value)} />
          {reason.trim() === "" && <p className="text-2xs text-ink-muted">{C.reasonNeeded}</p>}
          <FormActions
            busy={busy || reason.trim() === ""}
            tone="danger"
            saveLabel={C.rejectClose}
            onCancel={() => setRejectClose(null)}
            onSave={() => {
              const c = rejectClose;
              if (!c) return;
              void post(`/api/v1/admin/cashbox/closes/${c.id}/reject`, { note: reason.trim() }, undefined, setReasonFail).then(
                (ok) => {
                  if (!ok) return;
                  setRejectClose(null);
                  setReason("");
                },
              );
            }}
          />
        </div>
      </Modal>
      <Modal open={resolve !== null} onClose={() => setResolve(null)} title={C.resolve}>
        <div className="space-y-3">
          {reasonFail && <Alert>{reasonFail}</Alert>}
          <Input id="cb-resolve" label={C.resolveNote} value={reason} onChange={(e) => setReason(e.target.value)} />
          {reason.trim() === "" && <p className="text-2xs text-ink-muted">{C.reasonNeeded}</p>}
          <FormActions
            busy={busy || reason.trim() === ""}
            saveLabel={C.resolve}
            onCancel={() => setResolve(null)}
            onSave={() => {
              const c = resolve;
              if (!c?.shortfall_id) return;
              void post(
                `/api/v1/admin/cashbox/shortfalls/${c.shortfall_id}/resolve`,
                { note: reason.trim() },
                undefined,
                setReasonFail,
              ).then((ok) => {
                if (!ok) return;
                setResolve(null);
                setReason("");
              });
            }}
          />
        </div>
      </Modal>
    </div>
  );
}
