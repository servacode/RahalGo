"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **النقدُ والصندوق — صندوقُ المكتب والإغلاقُ اليوميّ** (قرارُ المالك ٢٠٢٦-١٠-٠٤)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **ما دخل الدرجَ وما خرج منه منذ آخر إغلاق**، و«إغلاقُ اليوم»: الموظّفُ يكتب ما
 * عدّه، والمحرّكُ يقارنه بالمتوقَّع ويُظهر الفرق، **وموظّفٌ آخرُ يراجع.** والنقصُ
 * يُسجَّل، ويُحَلّ إن وُجد المال، وإن بقي يوماً صار خسارةً بموافقة مدير المنصّة.
 */

import { useState } from "react";
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
import { useAuth } from "@/lib/auth";
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
  const { user: me } = useAuth();
  const canCall = useCanCall();
  const canClose = canCall("POST", "/cashbox/closes");
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
  const [rejectClose, setRejectClose] = useState<CashClose | null>(null);
  const [resolve, setResolve] = useState<CashClose | null>(null);
  const [reason, setReason] = useState("");

  function reloadAll() {
    box.reload();
    closes.reload();
  }

  async function post(path: string, body: unknown) {
    setBusy(true);
    try {
      await api(path, { method: "POST", body: JSON.stringify(body) });
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

  return (
    <div className="space-y-4">
      {fail && <Alert onDismiss={() => setFail("")}>{fail}</Alert>}
      <StatGrid>
        <StatCard label={C.expected} value={fmtMoney(pos.expected)} icon={IconWallet} emphasis />
        <StatCard label={C.opening} value={fmtMoney(pos.opening)} sub={pos.opened_at ? fmtDateTime(pos.opened_at) : undefined} />
        <StatCard label={C.in} value={fmtMoney(pos.cash_in)} tone="success" />
        <StatCard label={C.out} value={fmtMoney(pos.cash_out)} tone="danger" />
      </StatGrid>

      <div className="flex flex-wrap items-center gap-2">
        {canClose && !pending && (
          <Button onClick={() => setClosing(true)} disabled={busy}>
            {C.closeDay}
          </Button>
        )}
        <Link href="/dashboard/cash" className="text-sm text-primary underline">
          {C.cashPage}
        </Link>
        <span className="text-xs text-ink-muted">
          {C.openLines}: <span dir="ltr">{pos.open_lines}</span>
        </span>
      </div>

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
              {me?.id !== pending.proposed_by ? (
                <Button disabled={busy} onClick={() => void post(`/api/v1/admin/cashbox/closes/${pending.id}/approve`, {})}>
                  {C.approveClose}
                </Button>
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

      <section className="space-y-2">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h3 className="font-bold">{C.entries}</h3>
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
          <div className="surface overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="text-2xs text-ink-muted">
                <tr className="border-b border-line-soft">
                  <th className="p-2 text-start">{T.statement.colDate}</th>
                  <th className="p-2 text-start">{C.colSource}</th>
                  <th className="p-2 text-start">{T.statement.colAmount}</th>
                  <th className="p-2 text-start">{C.colParty}</th>
                  <th className="p-2 text-start">{T.statement.colBy}</th>
                  <th className="p-2 text-start">{T.statement.colNote}</th>
                </tr>
              </thead>
              <tbody>
                {box.data.entries.map((e) => (
                  <tr key={e.id} className="border-b border-line-soft">
                    <td className="whitespace-nowrap p-2" dir="ltr">
                      {fmtDateTime(e.created_at)}
                    </td>
                    <td className="p-2">{sourceLabel(e.source)}</td>
                    <td className="p-2">
                      <Signed value={e.direction === "in" ? e.amount : -e.amount} />
                    </td>
                    <td className="p-2">{e.user_name}</td>
                    <td className="p-2">{e.recorded_by}</td>
                    <td className="p-2 text-ink-muted">{e.note}</td>
                  </tr>
                ))}
              </tbody>
            </table>
          </div>
        )}
        <Pagination page={page} total={box.data.total} perPage={box.data.per_page} onChange={setPage} />
      </section>

      <section className="space-y-2">
        <h3 className="font-bold">{C.closes}</h3>
        {!closes.data ? (
          <LoadingState variant="text" />
        ) : closes.data.closes.length === 0 ? (
          <EmptyState icon={IconWallet} title={C.noCloses} />
        ) : (
          <div className="surface overflow-x-auto">
            <table className="w-full text-sm">
              <thead className="text-2xs text-ink-muted">
                <tr className="border-b border-line-soft">
                  <th className="p-2 text-start">{C.colDay}</th>
                  <th className="p-2 text-start">{C.colExpected}</th>
                  <th className="p-2 text-start">{C.colCounted}</th>
                  <th className="p-2 text-start">{C.difference}</th>
                  <th className="p-2 text-start">{C.colStatus}</th>
                  <th className="p-2 text-start">{C.colBy}</th>
                  <th className="p-2 text-start">{C.colReviewer}</th>
                  <th className="p-2 text-start">{C.shortfall}</th>
                </tr>
              </thead>
              <tbody>
                {closes.data.closes.map((c) => (
                  <tr key={c.id} className="border-b border-line-soft">
                    <td className="whitespace-nowrap p-2" dir="ltr">
                      {c.day}
                    </td>
                    <td className="p-2">
                      <Amount value={c.expected} />
                    </td>
                    <td className="p-2">
                      <Amount value={c.counted} />
                    </td>
                    <td className="p-2">
                      <DiffBadge diff={c.difference} />
                    </td>
                    <td className="p-2">
                      <Badge variant={c.status === "approved" ? "success" : c.status === "rejected" ? "danger" : "warning"}>
                        {C.statuses[c.status]}
                      </Badge>
                      {c.self_approved && <span className="ms-1 text-2xs text-warning">{T.approvals.selfFlag}</span>}
                    </td>
                    <td className="p-2">{c.proposer_name}</td>
                    <td className="p-2">{c.decider_name ?? ""}</td>
                    <td className="p-2">
                      {c.shortfall_status && (
                        <div className="flex flex-wrap items-center gap-1">
                          <Badge variant={c.shortfall_status === "pending" ? "warning" : "neutral"}>
                            {C.shortfallStatuses[c.shortfall_status]}
                          </Badge>
                          {c.shortfall_status === "pending" && canClose && (
                            <Button variant="secondary" disabled={busy} onClick={() => setResolve(c)}>
                              {C.resolve}
                            </Button>
                          )}
                          {c.shortfall_status === "pending" && c.shortfall_due_at && (
                            <span className="text-2xs text-ink-muted">
                              {C.dueAt} <span dir="ltr">{fmtDateTime(c.shortfall_due_at)}</span>
                            </span>
                          )}
                        </div>
                      )}
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
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
