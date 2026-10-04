"use client";

/**
 * **نافذةُ استلام النقد من السائق — واحدةٌ للمنصة كلّها.**
 *
 * قرارُ المالك (٢٠٢٦-١٠-٠٤): «طريقٌ واحدٌ لنفس العملية». كانت صفحةُ النقد
 * تفتح نافذةً، وملفُّ السائق نافذةً أخرى بسلوكٍ آخر. **والآن هذه وحدَها**،
 * تُستدعى من صفحة «النقد والصندوق» ومن تبويب الصندوق في ملفّ السائق.
 *
 * المبلغُ (افتراضُه كلُّ ما بذمّته) · ملاحظة · تأكيدٌ يقول ما سيُكتب وما يبقى ·
 * ثمّ كلمةُ السرّ (خطوةُ التحقّق العامّة) · والخطأُ يُعرض في مكانه.
 */

import { useState } from "react";
import { getMessages, defaultLocale, fmtMoney, errorText } from "@rahalgo/i18n";
import { Alert, Confirm, FormActions, Input, Modal, Money } from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const A = m.admin.acc;
const C = m.admin.cashOutstanding;

export function DriverCashReceive({
  driverID,
  driverName,
  held,
  onClose,
  onDone,
}: {
  driverID: string;
  driverName?: string;
  held: number;
  onClose: () => void;
  /** يُنادى بعد نجاح الاستلام بما بقي بذمّته. */
  onDone: (rest: number) => void;
}) {
  const [amount, setAmount] = useState(String(held));
  const [note, setNote] = useState("");
  const [confirming, setConfirming] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const value = Number(amount) || 0;

  async function submit() {
    // **التأكيدُ يُغلق قبل النداء** — كلمةُ السرّ تُطلب في نافذةٍ تحته.
    setConfirming(false);
    setBusy(true);
    setError("");
    try {
      const r = await api<{ held: number }>(`/api/v1/admin/drivers/${driverID}/settle`, {
        method: "POST",
        body: JSON.stringify({ amount: value, note: note.trim() }),
      });
      onDone(r?.held ?? Math.max(held - value, 0));
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  return (
    <>
      <Modal
        open
        onClose={onClose}
        title={driverName ? C.receiveFrom.replace("{n}", driverName) : A.settleTitle}
      >
        <form
          className="space-y-3"
          onSubmit={(e) => {
            e.preventDefault();
            if (value > 0 && value <= held) {
              setError("");
              setConfirming(true);
            } else {
              setError(C.badAmount);
            }
          }}
        >
          <p className="flex items-center justify-between rounded-control bg-field px-3 py-2 text-sm">
            <span className="text-ink-muted">{A.settleHeld}</span>
            <Money value={held} small />
          </p>
          <Input
            id="cash-receive-amount"
            label={`${A.settleAmount} (${m.common.currency})`}
            type="number"
            min="1"
            max={held}
            required
            autoFocus
            value={amount}
            onChange={(e) => setAmount(e.target.value)}
          />
          <p className="text-xs text-ink-muted">{A.settleHint}</p>
          <Input
            id="cash-receive-note"
            label={C.noteOptional}
            value={note}
            maxLength={300}
            onChange={(e) => setNote(e.target.value)}
          />
          {error && <Alert>{error}</Alert>}
          <FormActions submit onCancel={onClose} busy={busy} saveLabel={C.confirm} />
        </form>
      </Modal>
      <Confirm
        open={confirming}
        tone="primary"
        title={A.settleConfirm}
        body={A.settleConfirmBody
          .replace("{amount}", fmtMoney(value))
          .replace("{rest}", fmtMoney(Math.max(held - value, 0)))}
        confirmLabel={A.settleConfirm}
        busy={busy}
        onConfirm={() => void submit()}
        onCancel={() => setConfirming(false)}
      />
    </>
  );
}
