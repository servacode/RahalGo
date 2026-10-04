"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **طلباتُ حركة المحفظة — يقترحها موظّفٌ ويوافق غيرُه** (قرارُ المالك ٢٠٢٦-١٠-٠٤)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **ولا قيدَ قبل الموافقة**، وعندها قيدٌ بطرفين: الشحنُ النقديُّ يدخل صندوقَ المكتب،
 * والتعويضُ والتسويةُ من الخزينة. **وصاحبُ الاقتراح لا يرى زرَّ «موافقة» على طلبه** —
 * والمحرّكُ يردّه على كلّ حال (`self_approve`).
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtMoney, fmtDateTime, errorText } from "@rahalgo/i18n";
import { Alert, Badge, Button, Confirm, Input, LoadingState, EmptyState, IconWallet } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useCanCall } from "@/lib/policy";

const m = getMessages(defaultLocale);
const A = m.admin.acc;

export interface WalletRequestRow {
  id: string;
  user_id: string;
  user_name: string;
  kind: "topup" | "compensation" | "adjustment";
  debit: boolean;
  amount: number;
  note: string;
  status: string;
  proposed_by: string;
  proposer_name: string;
  self_approved: boolean;
  created_at: string;
}

export function kindLabel(r: Pick<WalletRequestRow, "kind" | "debit">): string {
  const base = A.kinds[r.kind] ?? A.txUnknown;
  if (r.kind !== "adjustment") return base;
  return `${base} · ${r.debit ? A.walletKindDebit : A.walletKindCredit}`;
}

export function WalletRequestsList({
  userID,
  onDecided,
  refreshKey = 0,
}: {
  /** **طلباتُ حسابٍ بعينه** — وفارغُه: كلُّ المعلّق. */
  userID?: string;
  onDecided?: () => void;
  refreshKey?: number;
}) {
  const { user: me } = useAuth();
  const canCall = useCanCall();
  const canDecide = canCall("POST", "/wallet-requests/{id}/approve");
  const [rows, setRows] = useState<WalletRequestRow[] | null>(null);
  const [error, setError] = useState("");
  const [approve, setApprove] = useState<WalletRequestRow | null>(null);
  const [reject, setReject] = useState<WalletRequestRow | null>(null);
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    try {
      const q = new URLSearchParams({ status: "pending" });
      if (userID) q.set("user_id", userID);
      const r = await api<{ requests: WalletRequestRow[] }>(`/api/v1/admin/wallet-requests?${q}`);
      setRows(r?.requests ?? []);
      setError("");
    } catch (err) {
      setRows([]);
      setError(errorText(err));
    }
  }, [userID]);

  useEffect(() => {
    void load();
  }, [load, refreshKey]);

  async function decide(r: WalletRequestRow, ok: boolean) {
    // **ونافذةُ التأكيد تُغلق قبل النداء** — الموافقةُ فعلٌ يطلب كلمةَ السرّ، ونافذتُها
    // (`z-50`) تحت التأكيد (`z-[70]`) فلا تُرى إن بقي مفتوحاً. (قِيس في متصفّح.)
    setApprove(null);
    setReject(null);
    setBusy(true);
    try {
      await api(`/api/v1/admin/wallet-requests/${r.id}/${ok ? "approve" : "reject"}`, {
        method: "POST",
        body: JSON.stringify({ note: ok ? "" : note.trim() }),
      });
      setApprove(null);
      setReject(null);
      setNote("");
      await load();
      onDecided?.();
    } catch (err) {
      setApprove(null);
      setReject(null);
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  if (rows === null) return <LoadingState variant="text" />;

  return (
    <div className="space-y-2">
      {error && <Alert>{error}</Alert>}
      {rows.length === 0 ? (
        <EmptyState icon={IconWallet} title={userID ? A.walletNoPending : A.requestsEmpty} />
      ) : (
        <ul className="divide-y divide-line">
          {rows.map((r) => (
            <li key={r.id} className="flex flex-wrap items-center gap-x-3 gap-y-1 py-2 text-sm">
              <Badge variant={r.kind === "adjustment" && r.debit ? "danger" : "success"}>{kindLabel(r)}</Badge>
              <span className="font-bold" dir="ltr">
                {fmtMoney(r.amount)}
              </span>
              {!userID && <span className="font-medium">{r.user_name}</span>}
              <span className="min-w-0 flex-1 truncate text-ink-muted">{r.note}</span>
              <span className="shrink-0 text-xs text-ink-muted">
                {A.proposedBy}: {r.proposer_name} · <span dir="ltr">{fmtDateTime(r.created_at)}</span>
              </span>
              {canDecide && (
                <span className="flex shrink-0 gap-1.5">
                  {me?.id !== r.proposed_by && (
                    <Button className="!px-2.5" disabled={busy} onClick={() => setApprove(r)}>
                      {A.approve}
                    </Button>
                  )}
                  <Button variant="secondary" className="!px-2.5" disabled={busy} onClick={() => setReject(r)}>
                    {A.reject}
                  </Button>
                </span>
              )}
            </li>
          ))}
        </ul>
      )}

      <Confirm
        open={approve !== null}
        tone="primary"
        title={A.approveTitle}
        body={
          approve &&
          A.approveBody
            .replace("{amount}", fmtMoney(approve.amount))
            .replace("{kind}", kindLabel(approve))
            .replace("{user}", approve.user_name)
            .replace("{side}", approve.kind === "topup" ? A.sideCashbox : A.sideTreasury)
        }
        confirmLabel={A.approve}
        busy={busy}
        onConfirm={() => approve && void decide(approve, true)}
        onCancel={() => setApprove(null)}
      />
      <Confirm
        open={reject !== null}
        title={A.rejectTitle}
        body={
          <Input id="wr-reject-note" label={A.rejectNote} value={note} onChange={(e) => setNote(e.target.value)} />
        }
        confirmLabel={A.reject}
        busy={busy}
        onConfirm={() => reject && void decide(reject, false)}
        onCancel={() => setReject(null)}
      />
    </div>
  );
}
