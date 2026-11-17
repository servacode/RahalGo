"use client";

/**
 * **«تعديل الأجرة» — على كلّ طلبٍ لحتّى التسليم** (قرارُ المالك ٢٠٢٦-١١-١٨)
 *
 * عاديٌّ وخاصٌّ (مشتريات · أمانة · إحضار) وتوصيلةُ متجرٍ وطلبُ واتساب. **المكتبُ يكتب الأجرةَ
 * الجديدةَ وسبباً إجباريّاً**، و**المعاينةُ حيّةٌ من الخادم** (`dry_run`) كلّما تبدّل الرقم: الأجرةُ
 * القديمةُ ← الجديدة، وما يأخذه الكابتن، وحصّةُ المنصّة، والمجموعُ الجديدُ على الدافع، وما يُردّ
 * أو يُخصم من محفظته. **والحسبةُ كلُّها في المحرّك** (`orders/order_fee_edit.go`) — الشاشةُ تسأل
 * وتُظهر ما ردّه، ولا تحسب رقماً.
 */
import { useEffect, useState } from "react";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import { Alert, FormActions, Input, Modal, Money } from "@rahalgo/ui";
import { api } from "@/lib/api";

const F = getMessages(defaultLocale).admin.ordersPage.feeEdit;

/** **بابُ المحرّك** — بقدرة تعديل الطلب نفسِها (`orders.intervene`). */
export const FEE_EDIT_ROUTE = ["POST", "/orders/{id}/delivery-fee"] as const;

export interface FeeEditResult {
  kind: string;
  payer: string;
  old_fee: number;
  new_fee: number;
  old_driver_fee: number;
  driver_fee: number;
  old_platform_share: number;
  platform_share: number;
  waived: number;
  old_total: number;
  new_total: number;
  old_cash_due: number;
  cash_due: number;
  refunded: number;
  charged: number;
  reconfirm: boolean;
  after_pickup: boolean;
  reason: string;
}

/** **من يدفع** بكلمته — الزبونُ أو المتجرُ أو المستلِم. */
function payerText(p: string): string {
  switch (p) {
    case "merchant":
      return F.payerMerchant;
    case "merchant_cash":
      return F.payerMerchantCash;
    case "recipient":
      return F.payerRecipient;
  }
  return F.payerCustomer;
}

export function FeeEditor({
  orderId,
  orderNumber,
  currentFee,
  onClose,
  onDone,
}: {
  orderId: string;
  orderNumber: number;
  /** **الأجرةُ الحاليّةُ قبل الإعفاء** — قيمةٌ مبدئيّةٌ للحقل، والخادمُ يقول الحقيقة. */
  currentFee: number;
  onClose: () => void;
  onDone: (res: FeeEditResult) => void;
}) {
  const [fee, setFee] = useState(String(currentFee));
  const [reason, setReason] = useState("");
  const [preview, setPreview] = useState<FeeEditResult | null>(null);
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");

  const n = Number(fee);
  const valid = fee.trim() !== "" && Number.isInteger(n) && n > 0;

  // ── المعاينةُ الحيّة: كلّما تبدّل الرقمُ يُسأل الخادم (بعد سكونٍ قصير) ─────────────
  useEffect(() => {
    setPreview(null);
    if (!valid || n === currentFee) return;
    let live = true;
    const timer = setTimeout(() => {
      api<{ result: FeeEditResult }>(`/api/v1/admin/orders/${orderId}/delivery-fee`, {
        method: "POST",
        // **السببُ يُكتب عند التأكيد** — والمعاينةُ لا تُدقَّق ولا تُكتب.
        body: JSON.stringify({ fee: n, reason: reason.trim() || F.previewReason, dry_run: true }),
      })
        .then((r) => {
          if (!live) return;
          setPreview(r.result);
          setErr("");
        })
        .catch((e) => live && setErr(errorText(e)));
    }, 400);
    return () => {
      live = false;
      clearTimeout(timer);
    };
    // السببُ لا يغيّر الأرقام — فلا يُعاد السؤالُ مع كلّ حرف.
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [fee, orderId]);

  async function submit() {
    if (!valid || reason.trim() === "") return;
    setBusy(true);
    setErr("");
    try {
      const r = await api<{ result: FeeEditResult }>(`/api/v1/admin/orders/${orderId}/delivery-fee`, {
        method: "POST",
        body: JSON.stringify({ fee: n, reason: reason.trim(), dry_run: false }),
      });
      onDone(r.result);
    } catch (e) {
      setErr(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  const row = "flex items-center justify-between gap-2";
  const pair = (a: number, b: number) => (
    <span className="flex items-center gap-1.5">
      <Money value={a} small />
      <span aria-hidden>←</span>
      <Money value={b} small />
    </span>
  );

  return (
    <Modal open onClose={onClose} title={F.title.replace("{n}", String(orderNumber))}>
      <div className="space-y-3 text-sm">
        <Input
          type="number"
          inputMode="numeric"
          placeholder={F.newFee}
          aria-label={F.newFee}
          value={fee}
          onChange={(e) => {
            setFee(e.target.value);
            setErr("");
          }}
        />
        {preview && (
          <div className="space-y-1.5 rounded-control bg-field px-3 py-2">
            <div className={`${row} font-bold`}>
              <span>{F.fee}</span>
              {pair(preview.old_fee, preview.new_fee)}
            </div>
            <div className={row}>
              <span className="text-ink-muted">{F.driverGets}</span>
              {pair(preview.old_driver_fee, preview.driver_fee)}
            </div>
            <div className={row}>
              <span className="text-ink-muted">{F.platformGets}</span>
              {pair(preview.old_platform_share, preview.platform_share)}
            </div>
            {preview.waived > 0 && (
              <div className={row}>
                <span className="text-ink-muted">{F.waived}</span>
                <Money value={preview.waived} small />
              </div>
            )}
            <div className={`${row} border-t border-line-soft pt-1.5`}>
              <span className="text-ink-muted">{F.totalOn.replace("{who}", payerText(preview.payer))}</span>
              {pair(preview.old_total, preview.new_total)}
            </div>
            <div className={row}>
              <span className="text-ink-muted">{F.cashDue}</span>
              {pair(preview.old_cash_due, preview.cash_due)}
            </div>
            {preview.refunded > 0 && (
              <div className={row}>
                <span className="text-ink-muted">{F.refunded}</span>
                <Money value={preview.refunded} small />
              </div>
            )}
            {preview.charged > 0 && (
              <div className={row}>
                <span className="text-ink-muted">{F.charged}</span>
                <Money value={preview.charged} small />
              </div>
            )}
            {preview.reconfirm && <p className="text-xs text-ink-muted">{F.reconfirm}</p>}
          </div>
        )}
        <Input
          placeholder={F.reason}
          aria-label={F.reason}
          value={reason}
          onChange={(e) => {
            setReason(e.target.value);
            setErr("");
          }}
        />
        {err && <Alert tone="error">{err}</Alert>}
        <FormActions
          busy={busy || !preview || reason.trim() === ""}
          onSave={() => void submit()}
          onCancel={onClose}
          saveLabel={F.confirm}
        />
      </div>
    </Modal>
  );
}
