"use client";

/**
 * **شام كاش في الخزينة** — حالُ الربط ورصيدُ الحساب، فواتيرُ الدفع بفلتر، ترجيعُ فاتورةٍ مدفوعة
 * (مدير المنصّة وحدَه: `treasury.manage`)، ومطابقةُ يومٍ مع قيود شام كاش.
 *
 * **والمفاتيحُ على السيرفر لا في اللوحة** — فإن لم تكن موجودةً قال الشريطُ ذلك ولم تُجرَّب المطابقة.
 */

import { useEffect, useState } from "react";
import { fmtDateTime, fmtMoney, fmtNum, errorText, damascusDay } from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  Confirm,
  Input,
  LoadingState,
  Pagination,
  ReloadState,
  Select,
  IconReceipt,
  useLiveData,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { T, Amount, EmptyCard } from "./shared";

const S = T.shamcash;
const STATUSES = ["pending", "paid", "expired", "failed", "refunded", "partly_refunded"] as const;
type BadgeVariant = "neutral" | "success" | "warning" | "danger" | "info";
function statusTone(st: string): BadgeVariant {
  switch (st) {
    case "paid":
      return "success";
    case "pending":
      return "warning";
    case "failed":
      return "danger";
    case "refunded":
    case "partly_refunded":
      return "info";
    default:
      return "neutral";
  }
}

interface AdminBill {
  id: string;
  number: string;
  purpose: "topup" | "order";
  amount: number;
  status: string;
  created_at: string;
  paid_at: string | null;
  order_id: string | null;
  order_number: number | null;
  order_error: string;
  user_id: string;
  user_name: string;
  user_phone: string;
  sham_tran_id: number | null;
  refunded: number;
}

interface BillsData {
  bills: AdminBill[];
  total: number;
  page: number;
  per_page: number;
  book_balance: number;
  enabled: boolean;
  configured: boolean;
}

interface ReconRow {
  bill_no: string;
  tran_id: string | number;
  type: "payment" | "refund";
  remote_amount: string | null;
  local_amount: number | null;
  status: "match" | "missing_local" | "missing_remote" | "amount_mismatch";
}

interface ReconData {
  date: string;
  rows: ReconRow[];
  summary: { remote_count: number; local_count: number; matched: number; mismatched: number };
}

const label = (map: Record<string, string>, k: string) => map[k] ?? k;

export function ShamCashTab() {
  const { can } = useAuth();
  const canRefund = can("treasury.manage");
  const [status, setStatus] = useState("");
  const [qInput, setQInput] = useState("");
  const [q, setQ] = useState("");
  const [page, setPage] = useState(1);
  const [refund, setRefund] = useState<AdminBill | null>(null);

  useEffect(() => {
    const t = window.setTimeout(() => {
      setQ(qInput.trim());
      setPage(1);
    }, 300);
    return () => window.clearTimeout(t);
  }, [qInput]);

  const qs = new URLSearchParams();
  if (status) qs.set("status", status);
  if (q) qs.set("q", q);
  qs.set("page", String(page));

  const { data, error, reload } = useLiveData<BillsData>(() => api(`/api/v1/admin/shamcash/bills?${qs.toString()}`), ["wallet"], [
    status,
    q,
    page,
  ]);

  return (
    <div className="space-y-4">
      {error && !data ? (
        <ReloadState onRetry={reload} />
      ) : !data ? (
        <LoadingState />
      ) : (
        <>
          <section className="surface flex flex-wrap items-center gap-2 p-4">
            <Badge variant={data.enabled ? "success" : "neutral"}>{data.enabled ? S.enabled : S.disabled}</Badge>
            <Badge variant={data.configured ? "success" : "warning"}>{data.configured ? S.configured : S.notConfigured}</Badge>
            <span className="ms-auto text-end">
              <span className="block text-xs text-ink-muted">{S.bookBalance}</span>
              <span className="figure block leading-none">{fmtMoney(data.book_balance)}</span>
              <span className="block text-xs text-ink-muted">{S.bookHint}</span>
            </span>
          </section>

          <section className="surface grid grid-cols-1 gap-3 p-4 sm:grid-cols-2">
            <Input id="sham-q" label={S.search} value={qInput} onChange={(e) => setQInput(e.target.value)} />
            <Select
              id="sham-status"
              label={S.statusLabel}
              value={status}
              onChange={(e) => {
                setStatus(e.target.value);
                setPage(1);
              }}
            >
              <option value="">{S.allStatuses}</option>
              {STATUSES.map((k) => (
                <option key={k} value={k}>
                  {label(S.status, k)}
                </option>
              ))}
            </Select>
          </section>

          {data.bills.length === 0 ? (
            <EmptyCard icon={IconReceipt} title={S.empty} hint={S.emptyHint} tone="muted" />
          ) : (
            <ul className="space-y-2">
              {data.bills.map((b) => {
                const left = b.amount - b.refunded;
                const refundable = canRefund && (b.status === "paid" || b.status === "partly_refunded") && left > 0;
                return (
                  <li key={b.id} className="surface flex flex-col gap-2 p-3">
                    <div className="flex flex-wrap items-center gap-2">
                      <span className="font-bold" dir="ltr">
                        {b.number}
                      </span>
                      <Badge variant={statusTone(b.status)}>{label(S.status, b.status)}</Badge>
                      <Badge variant="neutral">{label(S.purpose, b.purpose)}</Badge>
                      <span className="figure ms-auto leading-none">{fmtMoney(b.amount)}</span>
                    </div>
                    <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-ink-muted">
                      <span>
                        {b.user_name || "—"}
                        {b.user_phone && (
                          <>
                            {" "}
                            <span dir="ltr">{b.user_phone}</span>
                          </>
                        )}
                      </span>
                      {b.order_number != null && <span>{S.orderNo.replace("{n}", fmtNum(b.order_number))}</span>}
                      {b.sham_tran_id != null && (
                        <span>
                          {S.tranId}: <span dir="ltr">{b.sham_tran_id}</span>
                        </span>
                      )}
                      {b.refunded > 0 && (
                        <span>
                          {S.refundedAmount}: {fmtMoney(b.refunded)}
                        </span>
                      )}
                      <span dir="ltr">{fmtDateTime(b.created_at)}</span>
                    </div>
                    {b.order_error && <Alert tone="warning">{b.order_error}</Alert>}
                    {refundable && (
                      <div>
                        <Button variant="secondary" onClick={() => setRefund(b)}>
                          {S.refund}
                        </Button>
                      </div>
                    )}
                  </li>
                );
              })}
            </ul>
          )}
          {data.total > data.per_page && (
            <div className="flex justify-center">
              <Pagination page={page} total={data.total} perPage={data.per_page} onChange={setPage} />
            </div>
          )}
          <Reconcile configured={data.configured} />
        </>
      )}
      {refund && (
        <RefundDialog
          bill={refund}
          onClose={(done) => {
            setRefund(null);
            if (done) reload();
          }}
        />
      )}
    </div>
  );
}

function RefundDialog({ bill, onClose }: { bill: AdminBill; onClose: (done: boolean) => void }) {
  const left = bill.amount - bill.refunded;
  const [amount, setAmount] = useState(String(left));
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [fail, setFail] = useState("");
  // One key per dialog: a double click or a retry after a dropped response refunds once.
  const [idemKey] = useState(() =>
    typeof crypto !== "undefined" && "randomUUID" in crypto ? crypto.randomUUID() : String(Date.now()),
  );
  const value = Math.round(Number(amount));
  const valid = Number.isFinite(value) && value > 0 && value <= left;

  async function go() {
    if (!valid || busy) return;
    setBusy(true);
    setFail("");
    try {
      const r = await api<{ refund: { status: string } }>(`/api/v1/admin/shamcash/bills/${bill.id}/refund`, {
        method: "POST",
        headers: { "Idempotency-Key": idemKey },
        body: JSON.stringify({ amount: value, note: note.trim() }),
      });
      if (r.refund?.status === "failed") {
        setFail(S.refundFailed);
        setBusy(false);
        return;
      }
      onClose(true);
    } catch (err) {
      setFail(errorText(err));
      setBusy(false);
    }
  }

  return (
    <Confirm
      open
      tone="primary"
      title={S.refundTitle}
      body={
        <div className="space-y-3">
          <p>{S.refundBody.replace("{n}", bill.number)}</p>
          <Input
            id="sham-refund-amount"
            label={S.refundAmount}
            type="number"
            inputMode="numeric"
            dir="ltr"
            min={1}
            max={left}
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
          />
          <p className="text-xs text-ink-muted">{S.refundRemaining.replace("{amount}", fmtMoney(left))}</p>
          <Input id="sham-refund-note" label={S.refundNote} value={note} onChange={(e) => setNote(e.target.value)} />
          {fail && <Alert>{fail}</Alert>}
        </div>
      }
      confirmLabel={S.refundConfirm}
      busy={busy || !valid}
      onConfirm={() => void go()}
      onCancel={() => onClose(false)}
    />
  );
}

function Reconcile({ configured }: { configured: boolean }) {
  const [date, setDate] = useState(damascusDay());
  const [data, setData] = useState<ReconData | null>(null);
  const [busy, setBusy] = useState(false);
  const [fail, setFail] = useState("");

  async function run() {
    setBusy(true);
    setFail("");
    try {
      setData(await api<ReconData>(`/api/v1/admin/shamcash/reconcile?date=${encodeURIComponent(date)}`));
    } catch (err) {
      setData(null);
      setFail(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  const tone = (s: string): BadgeVariant => (s === "match" ? "success" : s === "amount_mismatch" ? "danger" : "warning");

  return (
    <section className="surface space-y-3 p-4">
      <h2 className="heading-card">{S.reconcile}</h2>
      <p className="text-sm text-ink-muted">{S.reconcileHint}</p>
      <div className="flex flex-wrap items-end gap-2">
        <Input id="sham-recon-date" type="date" label={S.date} value={date} onChange={(e) => setDate(e.target.value)} />
        <Button onClick={() => void run()} disabled={busy || !date || !configured}>
          {S.reconcileRun}
        </Button>
      </div>
      {fail && <Alert>{fail}</Alert>}
      {busy && <LoadingState />}
      {data && !busy && (
        <>
          <div className="grid grid-cols-2 gap-2 lg:grid-cols-4">
            {(
              [
                [S.summaryRemote, data.summary.remote_count],
                [S.summaryLocal, data.summary.local_count],
                [S.summaryMatched, data.summary.matched],
                [S.summaryMismatched, data.summary.mismatched],
              ] as const
            ).map(([k, v]) => (
              <div key={k} className="rounded-card border border-line-soft p-3">
                <span className="block text-xs text-ink-muted">{k}</span>
                <span className="figure block leading-tight">{fmtNum(v)}</span>
              </div>
            ))}
          </div>
          {data.rows.length === 0 ? (
            <p className="text-sm text-ink-muted">{S.reconcileEmpty}</p>
          ) : (
            <ul className="space-y-2">
              {data.rows.map((r, i) => (
                <li key={`${r.bill_no}-${r.tran_id}-${i}`} className="flex flex-wrap items-center gap-2 rounded-card border border-line-soft p-3 text-sm">
                  <span className="font-bold" dir="ltr">
                    {r.bill_no || "—"}
                  </span>
                  <Badge variant="neutral">{label(S.type, r.type)}</Badge>
                  <Badge variant={tone(r.status)}>{label(S.match, r.status)}</Badge>
                  <span className="text-xs text-ink-muted" dir="ltr">
                    #{r.tran_id}
                  </span>
                  <span className="ms-auto flex gap-4 text-xs text-ink-muted">
                    <span>
                      {S.rowRemote}: {r.remote_amount != null ? <span dir="ltr">{r.remote_amount}</span> : "—"}
                    </span>
                    <span>
                      {S.rowLocal}: {r.local_amount != null ? <Amount value={r.local_amount} /> : "—"}
                    </span>
                  </span>
                </li>
              ))}
            </ul>
          )}
        </>
      )}
    </section>
  );
}
