"use client";

/**
 * **«عدّل ثمن الطلب» لطلب الواتساب من الكارت** (قرارُ المالك ٢٠٢٦-١١-١٨)
 *
 * ثمنُ البضاعة اللي دفعها الكابتن للمحل — بيكتبه الكابتن من المحل، **والمكتب هون بيكتبه أو بيصحّحه**
 * من القبول لحد التسليم، بالليرة أو بالدولار، بسبب إجباري. **والزبون بتوصله رسالة المبالغ** متل ما
 * بتوصله لما يكتبه الكابتن. وصفر = أمانة بلا ثمن. الحسبة كلها بالمحرّك (`orders/fetch_amount.go`).
 */
import { useState } from "react";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import { Alert, FormActions, Input, Modal, Select } from "@rahalgo/ui";
import { api } from "@/lib/api";

const G = getMessages(defaultLocale).admin.ordersPage.goodsEdit;

/** **بابُ المحرّك** — بقدرة تعديل الطلب نفسِها (`orders.intervene`). */
export const GOODS_EDIT_ROUTE = ["POST", "/whatsapp-orders/{id}/goods"] as const;

export function GoodsEditor({
  orderId,
  orderNumber,
  current,
  currency,
  onClose,
  onDone,
}: {
  orderId: string;
  orderNumber: number;
  /** **الثمنُ الحاليّ بوحدته** (الدولارُ بالسنتات) — وnull: لسا ما انكتب. */
  current?: number | null;
  currency?: string;
  onClose: () => void;
  onDone: () => void;
}) {
  const startCur = currency === "USD" ? "USD" : "SYP";
  const [cur, setCur] = useState(startCur);
  const [amount, setAmount] = useState(
    current == null ? "" : String(startCur === "USD" ? current / 100 : current),
  );
  const [reason, setReason] = useState("");
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");

  const n = Number(amount);
  const valid = amount.trim() !== "" && Number.isFinite(n) && n >= 0 && (cur === "USD" || Number.isInteger(n));

  async function submit() {
    if (!valid || reason.trim() === "") return;
    setBusy(true);
    setErr("");
    try {
      await api(`/api/v1/admin/whatsapp-orders/${orderId}/goods`, {
        method: "POST",
        body: JSON.stringify({ amount: n, currency: cur, reason: reason.trim() }),
      });
      onDone();
    } catch (e) {
      setErr(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open onClose={onClose} title={G.title.replace("{n}", String(orderNumber))}>
      <div className="space-y-3 text-sm">
        <p className="text-ink-muted">{G.hint}</p>
        <div className="flex gap-2">
          <Input
            type="number"
            inputMode="decimal"
            placeholder={G.amount}
            aria-label={G.amount}
            value={amount}
            onChange={(e) => {
              setAmount(e.target.value);
              setErr("");
            }}
          />
          <Select aria-label={G.currency} value={cur} onChange={(e) => setCur(e.target.value)}>
            <option value="SYP">{G.syp}</option>
            <option value="USD">{G.usd}</option>
          </Select>
        </div>
        <Input
          placeholder={G.reason}
          aria-label={G.reason}
          value={reason}
          onChange={(e) => {
            setReason(e.target.value);
            setErr("");
          }}
        />
        {err && <Alert tone="error">{err}</Alert>}
        <FormActions
          busy={busy || !valid || reason.trim() === ""}
          onSave={() => void submit()}
          onCancel={onClose}
          saveLabel={G.confirm}
        />
      </div>
    </Modal>
  );
}
