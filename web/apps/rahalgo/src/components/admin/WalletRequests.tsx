"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **طلباتُ حركة المحفظة — قائمةٌ واحدةٌ بكروت** (قرارُ المالك ٢٠٢٦-١٠-٢٧: «نوحد قسم حركة المحفظة»)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **كانت نسختان**: كروتٌ في صفحة «التعويضات» وقائمةٌ قديمةٌ في نافذة محفظة الحساب.
 * **والآن مكوّنٌ واحد** يناديه الموضعان — طلباتُ حسابٍ بعينه (`userID`) أو كلُّ المعلَّق.
 *
 * **ولا قيدَ قبل الموافقة**، وعندها قيدٌ بطرفين: الشحنُ النقديُّ يدخل صندوقَ المكتب،
 * والتعويضُ والتسويةُ من الخزينة. **وطلبُ الموظّف ينتظر، وحركةُ المالك أو المدير تنفذ
 * فوراً** (قرارُ المالك ٢٠٢٦-١٠-٢٧)، **وزرُّ «موافقة» ظاهرٌ لصاحب الاقتراح أيضاً** —
 * المقترحُ يوافق على ما اقترحه (قرارُ ٢٠٢٦-١٠-٢٢)، والمحرّكُ وحدَه يحكم.
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtMoney, fmtDateTime, errorText } from "@rahalgo/i18n";
import { Alert, Confirm, Input, LoadingState, IconWallet, IconCheck, IconBlock } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useCanCall } from "@/lib/policy";

const m = getMessages(defaultLocale);
const A = m.admin.acc;
const C = m.admin.compensations;

export interface WalletRequestRow {
  id: string;
  user_id: string;
  user_name: string;
  kind: "topup" | "compensation" | "adjustment" | "advance";
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

/** **زرُّ الموافقة** — بشكل بطاقات الموافقات. */
export function ApproveButton({ label, onClick, disabled }: { label: string; onClick: () => void; disabled?: boolean }) {
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={onClick}
      className="inline-flex items-center justify-center gap-1.5 rounded-control bg-success-solid py-2.5 text-sm font-bold text-on-solid transition-opacity hover:opacity-90 disabled:opacity-50"
    >
      <IconCheck size={16} />
      {label}
    </button>
  );
}

/** **زرُّ الرفض.** */
export function RejectButton({ label, onClick, disabled }: { label: string; onClick: () => void; disabled?: boolean }) {
  return (
    <button
      type="button"
      disabled={disabled}
      onClick={onClick}
      className="inline-flex items-center justify-center gap-1.5 rounded-control border border-danger py-2.5 text-sm font-bold text-danger transition-colors hover:bg-danger-tint disabled:opacity-50"
    >
      <IconBlock size={16} />
      {label}
    </button>
  );
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
  const canDecide = useCanCall()("POST", "/wallet-requests/{id}/approve");
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
      setNote("");
      await load();
      onDecided?.();
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  if (rows === null) return <LoadingState variant="text" />;

  return (
    <div className="space-y-3">
      {error && <Alert>{error}</Alert>}
      {rows.length === 0 ? (
        <section className="surface flex flex-col items-center gap-3 p-8 text-center">
          <span className="inline-flex h-16 w-16 items-center justify-center rounded-full bg-success-solid text-on-solid">
            <IconWallet size={28} />
          </span>
          <h2 className="heading-section">{userID ? A.walletNoPending : A.requestsEmpty}</h2>
          {!userID && <p className="max-w-md text-sm text-ink-muted">{C.walletEmptyHint}</p>}
        </section>
      ) : (
        <div className={`grid grid-cols-1 gap-4 ${userID ? "" : "lg:grid-cols-2 2xl:grid-cols-3"}`}>
          {rows.map((r) => (
            <article key={r.id} className="surface flex flex-col overflow-hidden transition-shadow hover:elev-2">
              <header className="flex items-center gap-2.5 border-b border-line-soft bg-field px-4 py-2.5">
                <span
                  className={`inline-flex h-8 w-8 shrink-0 items-center justify-center rounded-full ${
                    r.debit ? "bg-danger-solid text-on-solid" : "bg-success-solid text-on-solid"
                  }`}
                >
                  <IconWallet size={15} />
                </span>
                <span className="min-w-0 flex-1 truncate text-sm font-bold">{kindLabel(r)}</span>
                <span className="shrink-0 text-2xs text-ink-muted" dir="ltr">
                  {fmtDateTime(r.created_at)}
                </span>
              </header>
              <div className="flex flex-1 flex-col gap-3 p-4">
                <div className="flex items-center gap-3">
                  <span className="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-field text-base font-bold text-ink-muted">
                    {(r.user_name || "?").trim().charAt(0)}
                  </span>
                  <div className="min-w-0 flex-1">
                    <p className="truncate font-bold">{r.user_name}</p>
                    <p className="truncate text-xs text-ink-muted">
                      {A.proposedBy}: {r.proposer_name}
                    </p>
                  </div>
                  <span className={`figure shrink-0 leading-none ${r.debit ? "text-danger" : ""}`} dir="ltr">
                    {fmtMoney(r.amount)}
                  </span>
                </div>
                {r.note && <p className="border-s-2 border-accent ps-3 text-sm leading-relaxed">{r.note}</p>}
              </div>
              {canDecide && (
                <footer className="grid grid-cols-2 gap-2 border-t border-line-soft p-3">
                  <ApproveButton label={A.approve} disabled={busy} onClick={() => setApprove(r)} />
                  <RejectButton label={A.reject} disabled={busy} onClick={() => setReject(r)} />
                </footer>
              )}
            </article>
          ))}
        </div>
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
        body={<Input id="wr-reject-note" label={A.rejectNote} value={note} onChange={(e) => setNote(e.target.value)} />}
        confirmLabel={A.reject}
        busy={busy}
        onConfirm={() => reject && void decide(reject, false)}
        onCancel={() => setReject(null)}
      />
    </div>
  );
}
