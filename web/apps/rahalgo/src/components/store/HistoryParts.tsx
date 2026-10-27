"use client";

/**
 * **قطعُ سجلّ الطلبات** — حالةُ الطلب، وأسطرُ المال، ونافذةُ «إبلاغ عن سائق».
 *
 * (مرآةُ `HistoryScreen.kt` و`ReportSheet` فيه.)
 *
 * **والبلاغُ يصل الإدارةَ لا السائق** — والأسبابُ من المحرّك
 * (`/merchant/report-reasons`) **والمعجمُ يقولها بالعربيّة.**
 */

import { useEffect, useState } from "react";
import { getMessages, defaultLocale, errorText, fmtMoney, fmtNum } from "@rahalgo/i18n";
import { Alert, FormActions, Modal, Radio, Textarea } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { noPayout, type MerchantOrder } from "@/lib/store/types";
import { fill } from "@/lib/customer/texts";
import { useShopToast } from "@/components/customer/toast";

const m = getMessages(defaultLocale);
const S = m.store;
const H = S.history;

export function orderStatus(status: string): string {
  return (S.orders.status as Record<string, string>)[status] ?? status;
}

export function reasonText(code: string): string {
  return (H.reasons as Record<string, string>)[code] ?? code;
}

/** **المجموعُ وخصمُ المنصّة والمستحقّ** — مرآةُ `SumLine` الثلاثة. */
export function MoneyLines({ o }: { o: MerchantOrder }) {
  // **وطلبٌ انتهى بلا بيعٍ لا حسبةَ فيه** — انظر `noPayout`.
  if (noPayout(o.status)) return <p className="text-sm text-ink-muted">{S.orders.noPayout}</p>;
  return (
    <dl className="flex flex-col gap-0.5 text-sm">
      <div className="flex justify-between gap-2">
        <dt className="text-ink-muted">{S.orders.sum}</dt>
        <dd>{fmtMoney(o.subtotal)}</dd>
      </div>
      <div className="flex justify-between gap-2">
        <dt className="text-ink-muted">{fill(S.orders.cut, { n: fmtNum(o.commission_percent_exact ?? o.commission_percent) })}</dt>
        <dd>{`− ${fmtMoney(o.platform_commission)}`}</dd>
      </div>
      <div className="flex justify-between gap-2 border-t border-line-soft pt-1 font-bold text-accent-text">
        <dt>{S.orders.due}</dt>
        <dd>{fmtMoney(o.merchant_net)}</dd>
      </div>
    </dl>
  );
}

export function ReportDriver({
  order,
  onClose,
  onSent,
}: {
  order: MerchantOrder;
  onClose: () => void;
  onSent: () => void;
}) {
  const toast = useShopToast();
  const [reasons, setReasons] = useState<string[]>([]);
  const [reason, setReason] = useState("");
  const [note, setNote] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  useEffect(() => {
    api<{ reasons: string[] }>("/api/v1/merchant/report-reasons")
      .then((r) => setReasons(r.reasons ?? []))
      .catch((e) => setError(errorText(e)));
  }, []);

  async function send() {
    if (!reason) return;
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/merchant/orders/${order.id}/report`, {
        method: "POST",
        body: JSON.stringify({ reason, note: note.trim() }),
      });
      toast(H.reportSent, "success");
      onSent();
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <Modal open onClose={onClose} title={fill(H.reportTitle, { n: order.number })}>
      <div className="flex flex-col gap-3">
        <p className="text-xs text-ink-muted">{H.reportHint}</p>
        <div className="flex flex-col gap-2">
          {reasons.map((r) => (
            <Radio key={r} id={`rep-${r}`} name="report-reason" checked={reason === r} onChange={() => setReason(r)} label={reasonText(r)} />
          ))}
        </div>
        <Textarea id="rep-note" label={H.reportNote} rows={2} value={note} onChange={(e) => setNote(e.target.value)} />
        {error && <Alert>{error}</Alert>}
        <FormActions onSave={() => void send()} onCancel={onClose} busy={busy || !reason} saveLabel={H.reportSend} />
      </div>
    </Modal>
  );
}
