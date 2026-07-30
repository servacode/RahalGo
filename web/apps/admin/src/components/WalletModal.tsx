"use client";

/** نافذة المحفظة المشتركة — تُستخدم في أقسام المستخدمين والزبائن والمندوبين. */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Button, Input, Select, Badge, Modal, IconWallet } from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";

const m = getMessages(defaultLocale);
const fmtNum = new Intl.NumberFormat("ar-SY");

function translateKey(key: string): string {
  let node: unknown = m;
  for (const part of key.split(".")) {
    if (typeof node !== "object" || node === null) return m.errors.internal;
    node = (node as Record<string, unknown>)[part];
  }
  return typeof node === "string" ? node : m.errors.internal;
}
function errText(err: unknown): string {
  return err instanceof ApiError ? translateKey(err.body.message_key) : m.errors.internal;
}

interface WalletTx {
  id: number;
  amount: number;
  kind: string;
  ref: string;
  note: string;
  created_at: string;
}

export default function WalletModal({
  user,
  onClose,
  isAdmin,
}: {
  user: { id: string; phone: string; full_name: string };
  onClose: () => void;
  isAdmin: boolean;
}) {
  const KINDS: Record<string, string> = m.admin.users.txKinds;
  const [balance, setBalance] = useState<number | null>(null);
  const [txs, setTxs] = useState<WalletTx[]>([]);
  const [amount, setAmount] = useState("");
  const [kind, setKind] = useState("topup");
  const [debit, setDebit] = useState(false);
  const [note, setNote] = useState("");
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const loadWallet = useCallback(async () => {
    try {
      const st = await api<{ balance: number; transactions: WalletTx[] }>(
        `/api/v1/admin/users/${user.id}/wallet`,
      );
      setBalance(st.balance);
      setTxs(st.transactions);
      setError("");
    } catch (err) {
      setError(errText(err));
    }
  }, [user.id]);

  useEffect(() => {
    void loadWallet();
  }, [loadWallet]);

  async function apply(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    const sign = kind === "payout" || (kind === "adjustment" && debit) ? -1 : 1;
    try {
      await api(`/api/v1/admin/users/${user.id}/wallet`, {
        method: "POST",
        body: JSON.stringify({ amount: sign * (Number(amount) || 0), kind, note }),
      });
      setAmount("");
      setNote("");
      await loadWallet();
    } catch (err) {
      setError(errText(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal
      open
      onClose={onClose}
      size="lg"
      title={`${m.admin.users.walletTitle}: ${user.full_name || user.phone}`}
    >
      <div className="mb-4 flex items-center justify-between rounded-card bg-primary-light p-4">
        <span className="flex items-center gap-2 font-medium text-primary-dark">
          <IconWallet size={18} />
          {m.admin.users.balance}
        </span>
        <span className="text-2xl font-bold text-primary-dark">
          {balance === null ? "…" : `${fmtNum.format(balance)} ${m.common.currency}`}
        </span>
      </div>

      {isAdmin && (
        <form onSubmit={apply} className="mb-4 space-y-3 rounded-card border border-line p-4">
          <div className="grid grid-cols-2 gap-3">
            <Input
              id="w-amount"
              label={`${m.admin.users.amount} (${m.common.currency})`}
              type="number"
              min="1"
              required
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
            />
            <Select
              id="w-kind"
              label={m.admin.users.movementKind}
              value={kind}
              onChange={(e) => setKind(e.target.value)}
            >
              <option value="topup">{KINDS.topup}</option>
              <option value="compensation">{KINDS.compensation}</option>
              <option value="adjustment">{KINDS.adjustment}</option>
              <option value="payout">{KINDS.payout}</option>
            </Select>
          </div>
          {kind === "adjustment" && (
            <label className="flex cursor-pointer items-center gap-2 text-sm">
              <input
                type="checkbox"
                checked={debit}
                onChange={(e) => setDebit(e.target.checked)}
                className="h-4 w-4 accent-danger"
              />
              {m.admin.users.isDebit}
            </label>
          )}
          <Input
            id="w-note"
            label={m.admin.users.noteField}
            value={note}
            onChange={(e) => setNote(e.target.value)}
          />
          {error && (
            <p className="rounded-control bg-danger/10 px-3 py-2 text-sm text-danger">{error}</p>
          )}
          <Button type="submit" disabled={busy} className="w-full">
            {m.admin.users.applyMovement}
          </Button>
        </form>
      )}

      <h3 className="mb-2 text-sm font-bold">{m.admin.users.txHistory}</h3>
      {txs.length === 0 ? (
        <p className="rounded-control bg-page p-4 text-center text-sm text-ink-muted">
          {m.admin.users.noTx}
        </p>
      ) : (
        <ul className="max-h-60 space-y-1.5 overflow-y-auto">
          {txs.map((t) => (
            <li
              key={t.id}
              className="flex items-center justify-between rounded-control border border-line px-3 py-2 text-sm"
            >
              <span className="flex items-center gap-2">
                <Badge variant={t.amount > 0 ? "success" : "danger"}>
                  {KINDS[t.kind] ?? t.kind}
                </Badge>
                {t.note && <span className="text-xs text-ink-muted">{t.note}</span>}
              </span>
              <span className={`font-bold ${t.amount > 0 ? "text-success" : "text-danger"}`}>
                {t.amount > 0 ? "+" : ""}
                {fmtNum.format(t.amount)}
              </span>
            </li>
          ))}
        </ul>
      )}
    </Modal>
  );
}
