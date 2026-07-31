"use client";

/**
 * عمولاتي — الرصيد وحركاته، **وطلب سحبه**.
 *
 * كانت العمولات تدخل المحفظة وتقف بلا طريق للصرف: رقم يكبر بلا معنى عملي.
 * الآن تُغلق الدورة هنا — يطلب المندوب سحب رصيده، وتراه المالية، ويصله قرارها.
 */

import { useCallback, useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtDate } from "@rahalgo/i18n";
import {
  Badge,
  Button,
  Input,
  Modal,
  PageContainer,
  PageHeader,
  EmptyState,
  LoadingState,
  ListRow,
  Card,
  useLiveData,
  IconWallet,
  IconWarning,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";

const m = getMessages(defaultLocale);
const P = m.shared.payout;
const KIND_LABELS: Record<string, string> = m.shared.txKinds;

interface Tx {
  id: string;
  kind: string;
  amount: number;
  note: string;
  created_at: string;
}

interface Payout {
  id: string;
  amount: number;
  status: "pending" | "paid" | "rejected";
  note: string;
  decision: string;
  created_at: string;
}

const STATUS_VARIANT: Record<Payout["status"], "warning" | "success" | "danger"> = {
  pending: "warning",
  paid: "success",
  rejected: "danger",
};

function errText(err: unknown): string {
  if (!(err instanceof ApiError)) return m.errors.internal;
  const key = err.body.message_key.split(".").pop() ?? "";
  return (m.errors as Record<string, string>)[key] ?? m.errors.internal;
}

export default function WalletPage() {
  const [asking, setAsking] = useState(false);

  const {
    data: statement,
    loading,
    reload,
  } = useLiveData<{ balance: number; transactions: Tx[] }>(() => api("/api/v1/rep/wallet"), [
    "wallet",
  ]);
  const { data: payouts, reload: reloadPayouts } = useLiveData<Payout[]>(
    () => api("/api/v1/me/payouts"),
    ["wallet"],
  );

  const refreshAll = useCallback(() => {
    reload();
    reloadPayouts();
  }, [reload, reloadPayouts]);

  if (loading) return <LoadingState />;

  const balance = statement?.balance ?? 0;
  const txs = statement?.transactions ?? [];
  const requests = payouts ?? [];
  const hasPending = requests.some((p) => p.status === "pending");

  return (
    <PageContainer>
      <PageHeader
        icon={IconWallet}
        title={m.terms.wallet}
        actions={
          <Button onClick={() => setAsking(true)} disabled={balance <= 0 || hasPending}>
            {P.request}
          </Button>
        }
      />

      {/* الرصيد — الرقم الذي يهمّ المندوب أولاً */}
      <div className="rounded-card bg-primary p-6 text-center text-white">
        <p className="text-sm opacity-80">{P.available}</p>
        <p className="mt-1 text-3xl font-bold" dir="ltr">
          {fmtNum(balance)} <span className="text-base font-normal">{m.common.currency}</span>
        </p>
        <p className="mt-2 text-xs opacity-70">{P.hint}</p>
      </div>

      {requests.length > 0 && (
        <Card title={P.myRequests} icon={IconWallet}>
          <ul className="space-y-2">
            {requests.map((p) => (
              <ListRow
                key={p.id}
                title={
                  <span dir="ltr">
                    {fmtNum(p.amount)} {m.common.currency}
                  </span>
                }
                subtitle={p.decision || p.note || undefined}
                trailing={
                  <>
                    <Badge variant={STATUS_VARIANT[p.status]}>{P.status[p.status]}</Badge>
                    <span className="text-xs text-ink-muted" dir="ltr">
                      {fmtDate(p.created_at)}
                    </span>
                  </>
                }
              />
            ))}
          </ul>
        </Card>
      )}

      <Card title={m.terms.transactions} icon={IconWallet}>
        {txs.length === 0 ? (
          <EmptyState icon={IconWallet} title={m.terms.noTransactions} />
        ) : (
          <ul className="space-y-2">
            {txs.map((tx) => (
              <ListRow
                key={tx.id}
                title={KIND_LABELS[tx.kind] ?? tx.kind}
                subtitle={tx.note || undefined}
                trailing={
                  <>
                    <span
                      className={`font-bold ${tx.amount >= 0 ? "text-success" : "text-danger"}`}
                      dir="ltr"
                    >
                      {tx.amount >= 0 ? "+" : ""}
                      {fmtNum(tx.amount)}
                    </span>
                    <span className="text-xs text-ink-muted" dir="ltr">
                      {fmtDate(tx.created_at)}
                    </span>
                  </>
                }
              />
            ))}
          </ul>
        )}
      </Card>

      {asking && (
        <PayoutModal
          balance={balance}
          onClose={() => setAsking(false)}
          onDone={() => {
            setAsking(false);
            refreshAll();
          }}
        />
      )}
    </PageContainer>
  );
}

function PayoutModal({
  balance,
  onClose,
  onDone,
}: {
  balance: number;
  onClose: () => void;
  onDone: () => void;
}) {
  const [amount, setAmount] = useState(String(balance));
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setError("");
    const value = Number(amount);
    if (!Number.isFinite(value) || value <= 0 || value > balance) {
      return setError(m.errors.insufficient_balance);
    }
    setBusy(true);
    try {
      await api("/api/v1/me/payouts", {
        method: "POST",
        body: JSON.stringify({ amount: value, note }),
      });
      onDone();
    } catch (err) {
      setError(errText(err));
      setBusy(false);
    }
  }

  return (
    <Modal open onClose={onClose} title={P.title}>
      <form onSubmit={submit} className="space-y-4">
        <div>
          <Input
            id="payout-amount"
            label={P.amount}
            type="number"
            dir="ltr"
            min="1"
            max={balance}
            required
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
            className="text-center text-lg font-bold"
          />
          <button
            type="button"
            onClick={() => setAmount(String(balance))}
            className="mt-1.5 text-sm font-medium text-primary hover:underline"
          >
            {P.all} ({fmtNum(balance)} {m.common.currency})
          </button>
        </div>

        <Input
          id="payout-note"
          label={P.note}
          value={note}
          onChange={(e) => setNote(e.target.value)}
          placeholder={P.notePlaceholder}
        />

        {error && (
          <p
            role="alert"
            className="flex items-start gap-2 rounded-control border border-danger/25 bg-danger/10 px-3 py-2 text-sm text-danger"
          >
            <IconWarning size={16} className="mt-0.5 shrink-0" />
            {error}
          </p>
        )}

        <div className="flex justify-end gap-2">
          <Button type="button" variant="secondary" onClick={onClose}>
            {m.common.cancel}
          </Button>
          <Button type="submit" disabled={busy}>
            {busy ? m.common.loading : P.submit}
          </Button>
        </div>
      </form>
    </Modal>
  );
}
