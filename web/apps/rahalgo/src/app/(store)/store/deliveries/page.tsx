"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **‏/store/deliveries — «سجل التوصيلات»: كلُّ ما أُرسل، صفحةً صفحة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `DeliveryHistoryScreen` و`DeliveryHistoryViewModel` في
 *  `DeliveryScreen.kt`: خمسون في الصفحة، **و«عرض المزيد» ما دامت الصفحةُ
 *  ممتلئة**، والسطرُ يفتح متابعتَها، **وما يُلغى يُلغى من هنا.**)
 *
 * **والتفاصيلُ صفحةُ المتابعة نفسُها** (`/store/delivery/[id]`).
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import { Alert, Button, EmptyState, IconMoto, LoadingState, ReloadState, useLiveRefresh } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useStore } from "@/lib/store/context";
import type { Delivery } from "@/lib/store/types";
import { DeliveryRow } from "@/components/store/DeliveryParts";

const m = getMessages(defaultLocale);
const D = m.store.delivery;
const X = m.store.deliveries;

/** **مرآةُ `PAGE = 50`** — وهو ما يردّه المحرّك في الصفحة (`ListMerchantDeliveries`). */
const PAGE = 50;

export default function DeliveriesHistoryPage() {
  const { store } = useStore();
  const id = store?.id ?? "";
  const [rows, setRows] = useState<Delivery[] | null>(null);
  const [page, setPage] = useState(1);
  const [more, setMore] = useState(false);
  const [busy, setBusy] = useState(false);
  const [failed, setFailed] = useState(false);
  const [cancelling, setCancelling] = useState<string | null>(null);
  const [cancelError, setCancelError] = useState("");

  const fetchPage = useCallback(
    async (p: number, reset: boolean) => {
      if (!id) return;
      setBusy(true);
      setFailed(false);
      try {
        const r = await api<{ deliveries: Delivery[] }>(`/api/v1/merchant/stores/${id}/deliveries?page=${p}`);
        const got = r.deliveries ?? [];
        setRows((old) => (reset ? got : [...(old ?? []), ...got]));
        setMore(got.length >= PAGE);
        setPage(p);
      } catch {
        setFailed(true);
      } finally {
        setBusy(false);
      }
    },
    [id],
  );
  useEffect(() => {
    setRows(null);
    void fetchPage(1, true);
  }, [fetchPage]);
  useLiveRefresh(["delivery", "order"], () => void fetchPage(1, true));

  async function cancel(did: string) {
    if (cancelling) return;
    setCancelling(did);
    setCancelError("");
    try {
      await api(`/api/v1/merchant/deliveries/${did}/cancel`, { method: "POST", body: "{}" });
    } catch (e) {
      setCancelError(errorText(e));
    } finally {
      setCancelling(null);
      void fetchPage(1, true);
    }
  }

  return (
    <div className="flex flex-col gap-3">
      <header>
        <h2 className="heading-page">{X.title}</h2>
        <p className="text-sm text-ink-muted">{X.hint}</p>
      </header>
      {cancelError && <Alert>{cancelError}</Alert>}
      {failed && !rows && <ReloadState onRetry={() => void fetchPage(1, true)} />}
      {rows == null && !failed && <LoadingState variant="inline" label={D.loading} />}
      {rows != null && rows.length === 0 && (
        <EmptyState icon={IconMoto} title={D.none} action={<p className="text-xs text-ink-muted">{D.noneHint}</p>} />
      )}
      {rows?.map((d) => (
        <DeliveryRow key={d.id} d={d} cancelling={cancelling === d.id} onCancel={() => void cancel(d.id)} />
      ))}
      {more && (
        <Button variant="secondary" disabled={busy} onClick={() => void fetchPage(page + 1, false)} data-testid="deliveries-more">
          {D.more}
        </Button>
      )}
      {failed && rows && <ReloadState onRetry={() => void fetchPage(page + 1, false)} />}
    </div>
  );
}
