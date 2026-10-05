"use client";

/**
 * **نظرةُ الخزينة العامّة** — رصيدُها، وما في درج المكتب، والنقدُ في الشارع، وما
 * ينتظر قراراً، **وسحبُ الأدمن من رصيدها** (قرارُ المالك ٢٠٢٦-١٠-٠٤: «حتّى لو دفع
 * من المحفظة رح يكون واضح إنّ الأدمن سحب من رصيد الخزينة»).
 */

import { useState } from "react";
import { fmtMoney, fmtDateTime, errorText } from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  Checkbox,
  EmptyState,
  FormActions,
  Input,
  LoadingState,
  Modal,
  ReloadState,
  StatCard,
  StatGrid,
  IconWallet,
  IconBalance,
  IconDriver,
  IconStore,
  IconCheck,
  IconWarning,
  useLiveData,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useCanCall } from "@/lib/policy";
import { T, sectionLabel } from "./shared";
import type { ApprovalItem } from "./Approvals";
import { DiffBadge, type CashClose } from "./Cashbox";

const O = T.overview;
const W = T.withdraw;

interface Overview {
  treasury_balance: number;
  has_treasury: boolean;
  cashbox: { expected: number };
  today_in: number;
  today_out: number;
  cash_in_street: number;
  merchant_cash_due: number;
  pending_payouts: number;
  withdrawals_month: number;
  open_shortfalls: number;
  pending_count: number;
  pending_total: number;
  last_close: CashClose | null;
  approvals: ApprovalItem[];
}
interface Withdrawal {
  id: string;
  amount: number;
  note: string;
  from_cashbox: boolean;
  by_name: string;
  created_at: string;
}

function newKey(): string {
  return typeof crypto !== "undefined" && "randomUUID" in crypto ? crypto.randomUUID() : String(Date.now());
}

export function OverviewTab({ onTab }: { onTab: (tab: "approvals" | "cashbox") => void }) {
  const canCall = useCanCall();
  const canWithdraw = canCall("POST", "/treasury/withdrawals");
  const ov = useLiveData<Overview>(() => api(`/api/v1/admin/treasury/overview`), ["wallet"]);
  const wd = useLiveData<{ withdrawals: Withdrawal[] }>(() => api(`/api/v1/admin/treasury/withdrawals`), ["wallet"]);
  const [open, setOpen] = useState(false);
  const [amount, setAmount] = useState("");
  const [note, setNote] = useState("");
  const [fromCashbox, setFromCashbox] = useState(true);
  const [busy, setBusy] = useState(false);
  const [fail, setFail] = useState("");
  const [done, setDone] = useState("");
  const [idemKey, setIdemKey] = useState(newKey);

  async function withdraw() {
    setBusy(true);
    try {
      await api(`/api/v1/admin/treasury/withdrawals`, {
        method: "POST",
        headers: { "Idempotency-Key": idemKey },
        body: JSON.stringify({ amount: Math.round(Number(amount)), note: note.trim(), from_cashbox: fromCashbox }),
      });
      setOpen(false);
      setAmount("");
      setNote("");
      setIdemKey(newKey());
      setFail("");
      setDone(W.done);
      ov.reload();
      wd.reload();
    } catch (err) {
      setFail(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  if (ov.error && !ov.data) return <ReloadState onRetry={ov.reload} />;
  if (!ov.data) return <LoadingState variant="stats" />;
  const d = ov.data;
  const amountNum = Number(amount);

  return (
    <div className="space-y-4">
      {done && (
        <Alert tone="success" onDismiss={() => setDone("")}>
          {done}
        </Alert>
      )}
      {!d.has_treasury && <Alert tone="warning">{O.noTreasury}</Alert>}
      <StatGrid>
        <StatCard
          label={O.treasuryBalance}
          value={fmtMoney(d.treasury_balance)}
          icon={IconWallet}
          tone={d.treasury_balance >= 0 ? "success" : "danger"}
          emphasis
        />
        <StatCard label={O.cashboxExpected} value={fmtMoney(d.cashbox.expected)} icon={IconBalance} onClick={() => onTab("cashbox")} />
        <StatCard label={O.todayIn} value={fmtMoney(d.today_in)} tone="success" />
        <StatCard label={O.todayOut} value={fmtMoney(d.today_out)} tone="danger" />
        <StatCard label={O.cashInStreet} value={fmtMoney(d.cash_in_street)} icon={IconDriver} />
        <StatCard label={O.merchantCashDue} value={fmtMoney(d.merchant_cash_due)} icon={IconStore} />
        <StatCard label={O.pendingPayouts} value={fmtMoney(d.pending_payouts)} />
        <StatCard label={O.withdrawalsMonth} value={fmtMoney(d.withdrawals_month)} />
        <StatCard
          label={O.openShortfalls}
          value={fmtMoney(d.open_shortfalls)}
          icon={IconWarning}
          tone={d.open_shortfalls > 0 ? "danger" : "muted"}
        />
        <StatCard
          label={O.pendingApprovals}
          value={String(d.pending_count)}
          sub={fmtMoney(d.pending_total)}
          icon={IconCheck}
          tone={d.pending_count > 0 ? "warning" : "muted"}
          onClick={() => onTab("approvals")}
        />
      </StatGrid>

      <section className="surface space-y-1 p-3 text-sm">
        <h3 className="font-bold">{O.lastClose}</h3>
        {d.last_close ? (
          <p className="flex flex-wrap items-center gap-2">
            <span dir="ltr">{d.last_close.day}</span>
            <Badge variant={d.last_close.status === "approved" ? "success" : "warning"}>
              {T.cashbox.statuses[d.last_close.status]}
            </Badge>
            <DiffBadge diff={d.last_close.difference} />
            <span className="text-ink-muted">{d.last_close.proposer_name}</span>
          </p>
        ) : (
          <p className="text-ink-muted">{O.noClose}</p>
        )}
      </section>

      {d.approvals.length > 0 && (
        <section className="space-y-2">
          <div className="flex items-center justify-between">
            <h3 className="font-bold">{O.pendingApprovals}</h3>
            <Button variant="secondary" onClick={() => onTab("approvals")}>
              {O.seeAll}
            </Button>
          </div>
          <ul className="divide-y divide-line surface">
            {d.approvals.map((a) => (
              <li key={`${a.key}-${a.id}`} className="flex flex-wrap items-center gap-2 px-3 py-2 text-sm">
                <Badge variant="neutral">{sectionLabel(a.section)}</Badge>
                <span dir="ltr" className="font-bold">
                  {fmtMoney(a.amount)}
                </span>
                {a.party_name && <span className="font-medium">{T.approvals.party.replace("{name}", a.party_name)}</span>}
                <span className="min-w-0 flex-1 truncate text-ink-muted">{a.note}</span>
              </li>
            ))}
          </ul>
        </section>
      )}

      <section className="space-y-2">
        <div className="flex flex-wrap items-center justify-between gap-2">
          <h3 className="font-bold">{W.list}</h3>
          {canWithdraw && d.has_treasury && <Button onClick={() => setOpen(true)}>{W.button}</Button>}
        </div>
        {!wd.data ? (
          <LoadingState variant="text" />
        ) : wd.data.withdrawals.length === 0 ? (
          <EmptyState icon={IconWallet} title={W.empty} />
        ) : (
          <ul className="divide-y divide-line surface">
            {wd.data.withdrawals.map((x) => (
              <li key={x.id} className="flex flex-wrap items-center gap-2 px-3 py-2 text-sm">
                <span dir="ltr" className="font-bold text-danger">
                  −{fmtMoney(x.amount)}
                </span>
                <span className="font-medium">{x.by_name}</span>
                <span className="min-w-0 flex-1 truncate text-ink-muted">{x.note}</span>
                <Badge variant="neutral">{x.from_cashbox ? W.cashbox : W.notCashbox}</Badge>
                <span className="text-xs text-ink-muted" dir="ltr">
                  {fmtDateTime(x.created_at)}
                </span>
              </li>
            ))}
          </ul>
        )}
      </section>

      <Modal open={open} onClose={() => setOpen(false)} title={W.title}>
        <div className="space-y-3">
          <p className="text-sm text-ink-muted">{W.hint}</p>
          {fail && <Alert>{fail}</Alert>}
          <p className="text-sm">
            {O.treasuryBalance}:{" "}
            <span dir="ltr" className="font-bold">
              {fmtMoney(d.treasury_balance)}
            </span>
          </p>
          <Input
            id="tw-amount"
            type="number"
            inputMode="numeric"
            dir="ltr"
            label={W.amount}
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
          />
          <Input id="tw-note" label={W.note} value={note} onChange={(e) => setNote(e.target.value)} />
          <Checkbox id="tw-cash" label={W.fromCashbox} checked={fromCashbox} onChange={(e) => setFromCashbox(e.target.checked)} />
          <FormActions
            busy={busy || !(amountNum > 0) || note.trim() === ""}
            saveLabel={W.confirm}
            onCancel={() => setOpen(false)}
            onSave={() => void withdraw()}
          />
        </div>
      </Modal>
    </div>
  );
}
