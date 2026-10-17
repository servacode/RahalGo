"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«السائق دفع مبلغ بمكان الاستلام» — سؤالٌ لا يُغلق إلّا بجواب**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `FetchAmountApproveDialog` في تطبيق المتجر.)
 *
 * **في توصيلة «إحضار»** يدفع السائقُ ثمنَ الغرض عند المورّد ويكتب المبلغ.
 * **ويُسأل صاحبُ المتجر**: يوافق فيدفعه للسائق حين يصله الغرض، أو يرفض
 * فيكتبه السائقُ من جديد.
 *
 * **والجوابُ بنسخة** (`expected_version`) — مبلغٌ كُتب من جديدٍ بعد أن فُتحت
 * النافذةُ لا يُوافَق عليه بجوابٍ عن المبلغ القديم: **المحرّكُ يردّه، ويُعرض
 * الجديد.**
 *
 * **وأخضرُ للموافقة وأحمرُ للرفض** — كما في التطبيق.
 */

import { useState } from "react";
import { getMessages, defaultLocale, errorText, fmtNum } from "@rahalgo/i18n";
import { Alert, Modal } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { awaitsAmount, useStoreDeliveries } from "@/lib/store/deliveries";
import { fill } from "@/lib/customer/texts";

const m = getMessages(defaultLocale);
const D = m.store.delivery;

export default function FetchAmountWatcher() {
  const { list, reload } = useStoreDeliveries();
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const d = (list ?? []).find(awaitsAmount);
  if (!d) return null;

  async function decide(approve: boolean) {
    if (!d || busy) return;
    setBusy(true);
    setError("");
    try {
      await api(`/api/v1/merchant/deliveries/${d.id}/fetch-amount/${approve ? "approve" : "reject"}`, {
        method: "POST",
        body: JSON.stringify({ expected_version: d.fetch_amount_version ?? 0 }),
      });
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
      void reload();
    }
  }

  return (
    <Modal
      open
      dismissable={false}
      onClose={() => undefined}
      title={fill(D.amountTitle, { n: d.number })}
      top
    >
      <div className="flex flex-col items-center gap-2 text-center">
        <p className="text-sm text-ink-muted">{D.amountPaid}</p>
        <p className="figure text-accent-text">
          {fill(D.amountValue, { amount: fmtNum(d.fetch_amount ?? 0) })}
        </p>
        {d.collect_amount != null && d.collect_amount > 0 && (
          <p className="text-sm text-ink-muted">{fill(D.amountEstimate, { amount: fmtNum(d.collect_amount) })}</p>
        )}
        <p className="text-sm">{D.amountBody}</p>
        {error && <Alert className="w-full">{error}</Alert>}
        <button
          type="button"
          disabled={busy}
          onClick={() => void decide(true)}
          className="mt-2 w-full rounded-control bg-success px-4 py-3 text-base font-bold text-on-bright transition-opacity hover:opacity-90 disabled:opacity-60"
        >
          {D.amountApprove}
        </button>
        <button
          type="button"
          disabled={busy}
          onClick={() => void decide(false)}
          className="w-full rounded-control bg-danger-solid px-4 py-3 text-base font-bold text-on-solid transition-opacity hover:opacity-90 disabled:opacity-60"
        >
          {D.amountReject}
        </button>
      </div>
    </Modal>
  );
}
