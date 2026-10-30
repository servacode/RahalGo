"use client";

/**
 * **«ما استلمت العمولة»** — قرارُ المالك ٢٠٢٦-١٠-٢٦: تُمسح نهائيّاً بقيدٍ عكسيّ، وبسببٍ مكتوب.
 *
 * نافذةٌ واحدةٌ لصفحة «نقد بيد السائقين» ولكروت سجلّ الطلبات (قرارُ المالك ٢٠٢٦-١٠-٣٠) — الحكمُ
 * كلُّه في المحرّك (`/admin/orders/{id}/waive-commission`)، وخطؤه يُقال هنا كما ردّه.
 */
import { useState } from "react";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import { Alert, FormActions, Input, Modal, Money } from "@rahalgo/ui";
import { api } from "@/lib/api";

const C = getMessages(defaultLocale).admin.cashOutstanding;

/** **مَن يملك المحو** — بابُ المحرّك نفسُه. */
export const WAIVE_ROUTE = ["POST", "/orders/{id}/waive-commission"] as const;

export function WaiveCommissionDialog({
  orderId,
  orderNumber,
  commission,
  onClose,
  onDone,
}: {
  orderId: string;
  orderNumber: number | null;
  commission: number;
  onClose: () => void;
  onDone: (amount: number) => void;
}) {
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  return (
    <Modal open onClose={onClose} title={C.waiveTitle.replace("{o}", String(orderNumber ?? ""))}>
      <form
        className="space-y-3"
        onSubmit={(ev) => {
          ev.preventDefault();
          if (!reason.trim()) {
            setError(C.waiveNeedReason);
            return;
          }
          setBusy(true);
          setError("");
          void (async () => {
            try {
              await api(`/api/v1/admin/orders/${orderId}/waive-commission`, {
                method: "POST",
                body: JSON.stringify({ reason: reason.trim() }),
              });
              onDone(commission);
            } catch (err) {
              setError(errorText(err));
              setBusy(false);
            }
          })();
        }}
      >
        <p className="flex items-center justify-between rounded-control bg-field px-3 py-2 text-sm">
          <span className="text-ink-muted">{C.splitCommission}</span>
          <Money value={commission} small />
        </p>
        <p className="text-sm text-ink-muted">{C.waiveHint}</p>
        <Input
          id="waive-reason"
          label={C.waiveReason}
          placeholder={C.waiveReasonPh}
          required
          autoFocus
          maxLength={300}
          value={reason}
          onChange={(ev) => setReason(ev.target.value)}
        />
        {error && <Alert>{error}</Alert>}
        <FormActions submit onCancel={onClose} busy={busy} saveLabel={C.waiveConfirm} />
      </form>
    </Modal>
  );
}
