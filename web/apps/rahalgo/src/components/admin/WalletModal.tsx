"use client";

/**
 * نافذة المحفظة المشتركة — الرصيدُ، **وطلبُ حركةٍ تقرّره الماليّة**، والطلباتُ المعلّقة.
 *
 * ══════════════════════════════════════════════════════════════════════
 * **ولا قيدَ من هنا بعد اليوم** (قرارُ المالك ٢٠٢٦-١٠-٠٤)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **كانت «شحن» و«تعويض» تزيد الرصيدَ بكبسة موظّفٍ واحدٍ بلا طرفٍ آخر** — مالٌ من
 * عدم. **والآن طلبٌ** بسببٍ إلزاميٍّ وسقفٍ ومفتاحِ عدمِ تكرار، **ويوافق عليه غيرُه**.
 * **و«سحب» نُزع** — بابُه صفحةُ طلبات السحب وحدَها.
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtMoney, errorText } from "@rahalgo/i18n";
import { Alert, Button, Input, Select, Modal, IconWallet, Checkbox, FormSection } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { WalletRequestsList } from "@/components/admin/WalletRequests";

const m = getMessages(defaultLocale);
const A = m.admin.acc;

function newKey(): string {
  return typeof crypto !== "undefined" && "randomUUID" in crypto
    ? crypto.randomUUID()
    : `w-${Date.now()}-${Math.random().toString(36).slice(2)}`;
}

export default function WalletModal({
  user,
  onClose,
  isAdmin,
}: {
  user: { id: string; phone: string; full_name: string };
  onClose: () => void;
  /** **يقترح حركة** — `finance.manage`. */
  isAdmin: boolean;
}) {
  const [balance, setBalance] = useState<number | null>(null);
  const [balanceError, setBalanceError] = useState("");
  const [amount, setAmount] = useState("");
  const [kind, setKind] = useState<"topup" | "compensation" | "adjustment">("topup");
  const [debit, setDebit] = useState(false);
  const [note, setNote] = useState("");
  const [error, setError] = useState("");
  const [sent, setSent] = useState(false);
  const [busy, setBusy] = useState(false);
  const [cap, setCap] = useState<number | null>(null);
  const [refresh, setRefresh] = useState(0);
  // **مفتاحٌ واحدٌ للطلب الواحد** — إعادةُ المحاولة بعد انقطاعٍ لا تكتب طلباً ثانياً.
  const [idemKey, setIdemKey] = useState(newKey);

  const loadWallet = useCallback(async () => {
    try {
      const st = await api<{ balance: number }>(`/api/v1/admin/users/${user.id}/wallet`);
      setBalance(st.balance);
      setBalanceError("");
    } catch (err) {
      setBalanceError(errorText(err));
    }
  }, [user.id]);

  useEffect(() => {
    void loadWallet();
    api<{ cap: number }>(`/api/v1/admin/wallet-requests?user_id=${user.id}`)
      .then((r) => setCap(r?.cap ?? null))
      // @empty-ok **والسقفُ تلميحٌ** — المحرّكُ يحرسه على كلّ حال.
      .catch(() => setCap(null));
  }, [loadWallet, user.id]);

  async function submit(e: React.FormEvent) {
    e.preventDefault();
    setBusy(true);
    setError("");
    setSent(false);
    try {
      await api(`/api/v1/admin/users/${user.id}/wallet`, {
        method: "POST",
        headers: { "Idempotency-Key": idemKey },
        body: JSON.stringify({ amount: Number(amount) || 0, kind, debit: kind === "adjustment" && debit, note: note.trim() }),
      });
      setAmount("");
      setNote("");
      setDebit(false);
      setSent(true);
      setIdemKey(newKey());
      setRefresh((n) => n + 1);
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open onClose={onClose} size="lg" title={`${m.admin.users.walletTitle}: ${user.full_name || user.phone}`}>
      <div className="mb-4 flex items-center justify-between rounded-card bg-primary-tint p-4">
        <span className="flex items-center gap-2 font-medium text-primary-dark">
          <IconWallet size={18} />
          {m.admin.users.balance}
        </span>
        <span className="figure text-primary-dark">{balance === null ? "…" : fmtMoney(balance)}</span>
      </div>
      {balanceError && <Alert className="mb-3">{balanceError}</Alert>}

      {isAdmin && (
        <form onSubmit={submit} className="mb-4 space-y-3 rounded-card border border-line p-4">
          <p className="heading-card">{A.walletRequestTitle}</p>
          <p className="text-xs leading-relaxed text-ink-muted">{A.walletRequestHint}</p>
          <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
            <Input
              id="w-amount"
              label={`${m.admin.users.amount} (${m.common.currency})`}
              type="number"
              min="1"
              max={cap ?? undefined}
              required
              value={amount}
              onChange={(e) => setAmount(e.target.value)}
            />
            <Select
              id="w-kind"
              label={m.admin.users.movementKind}
              value={kind}
              onChange={(e) => setKind(e.target.value as typeof kind)}
            >
              <option value="topup">{A.kinds.topup}</option>
              <option value="compensation">{A.kinds.compensation}</option>
              <option value="adjustment">{A.kinds.adjustment}</option>
            </Select>
          </div>
          {kind === "adjustment" && (
            <Checkbox id="wallet-debit" checked={debit} onChange={(e) => setDebit(e.target.checked)} label={m.admin.users.isDebit} />
          )}
          <Input id="w-note" label={A.walletNote} required value={note} onChange={(e) => setNote(e.target.value)} />
          {cap !== null && <p className="text-xs text-ink-muted">{A.walletCap.replace("{cap}", fmtMoney(cap))}</p>}
          {error && <Alert>{error}</Alert>}
          {sent && <Alert tone="success">{A.walletSent}</Alert>}
          <Button type="submit" disabled={busy || !note.trim()} className="w-full">
            {A.walletSubmit}
          </Button>
        </form>
      )}

      <FormSection title={A.walletPending} icon={<IconWallet />}>
        <WalletRequestsList
          userID={user.id}
          refreshKey={refresh}
          onDecided={() => void loadWallet()}
        />
      </FormSection>
    </Modal>
  );
}
