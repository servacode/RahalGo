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

import Link from "next/link";
import { useRouter } from "next/navigation";
import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, errorText, fmtMoney, fmtRelStamp } from "@rahalgo/i18n";
import { Alert, Button, ButtonLink, DataView, EmptyState, IconMoto, LoadingState, ReloadState, useLiveRefresh, useViewMode } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useStore } from "@/lib/store/context";
import type { Delivery } from "@/lib/store/types";
import { CancelDeliveryButton, DeliveryRow, StatusPill } from "@/components/store/DeliveryParts";
import { CANCELLABLE } from "@/lib/store/deliveries";
import { fill } from "@/lib/customer/texts";
import { PageHeader } from "@/components/store/SetupParts";

const m = getMessages(defaultLocale);
const D = m.store.delivery;
const X = m.store.deliveries;
const K = m.store.console;

/** **مرآةُ `PAGE = 50`** — وهو ما يردّه المحرّك في الصفحة (`ListMerchantDeliveries`). */
const PAGE = 50;

export default function DeliveriesHistoryPage() {
  const { store } = useStore();
  const router = useRouter();
  const [, , { narrow }] = useViewMode("store-deliveries", "table");
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
    <div className="console-page" data-testid="store-deliveries">
      <PageHeader
        title={X.title}
        actions={
          <ButtonLink Link={Link} href="/store/delivery" className="min-h-11">
            <IconMoto size={18} />
            {K.deliveryNew}
          </ButtonLink>
        }
      />
      {cancelError && <Alert>{cancelError}</Alert>}
      {failed && !rows && <ReloadState onRetry={() => void fetchPage(1, true)} />}
      {rows == null && !failed && <LoadingState variant="list" rows={4} />}
      {rows != null && rows.length === 0 && <EmptyState icon={IconMoto} title={D.none} />}

      {/* ── الحاسوب: جدولٌ كثيف، والصفُّ يفتح المتابعة ── */}
      {rows != null && rows.length > 0 && !narrow && (
        <DataView<Delivery>
          dense
          view="table"
          items={rows}
          getKey={(d) => d.id}
          empty={D.none}
          onRowClick={(d) => router.push(`/store/delivery/${d.id}`)}
          columns={[
            {
              id: "number",
              header: K.colOrder,
              widthClass: "w-40",
              cell: (d) => (
                <span className="flex flex-col" data-testid="store-delivery">
                  <Link
                    href={`/store/delivery/${d.id}`}
                    onClick={(e) => e.stopPropagation()}
                    className="font-bold text-ink hover:underline"
                  >
                    {fill(D.trackTitle, { n: d.number })}
                  </Link>
                  {d.created_at && <span className="text-xs text-ink-muted">{fmtRelStamp(d.created_at)}</span>}
                </span>
              ),
            },
            { id: "status", header: K.colStatus, widthClass: "w-40", cell: (d) => <StatusPill status={d.status} /> },
            {
              id: "customer",
              header: K.colCustomer,
              cell: (d) => (
                <span className="flex flex-col">
                  <span className="line-clamp-1 font-bold">{d.customer_name}</span>
                  <span className="text-xs text-ink-muted" dir="ltr">
                    {d.customer_phone}
                  </span>
                </span>
              ),
            },
            { id: "address", header: K.colAddress, cell: (d) => <span className="line-clamp-2 text-ink-muted">{d.address_text}</span> },
            {
              id: "fee",
              header: K.colFee,
              align: "end",
              widthClass: "w-32",
              cell: (d) => <span className="font-bold text-accent-text">{fmtMoney(d.delivery_fee)}</span>,
            },
          ]}
          actions={(d) =>
            CANCELLABLE.has(d.status) ? (
              <CancelDeliveryButton compact busy={cancelling === d.id} onConfirm={() => void cancel(d.id)} />
            ) : null
          }
        />
      )}

      {/* ── الجوّال: بطاقةٌ تنطوي وتنفتح ── */}
      {rows != null && rows.length > 0 && narrow && (
        <div className="flex flex-col gap-2 md:grid md:grid-cols-2 md:items-start">
          {rows.map((d) => (
            <DeliveryRow key={d.id} d={d} cancelling={cancelling === d.id} onCancel={() => void cancel(d.id)} />
          ))}
        </div>
      )}
      {more && (
        <Button variant="secondary" className="min-h-11 self-center" disabled={busy} onClick={() => void fetchPage(page + 1, false)} data-testid="deliveries-more">
          {D.more}
        </Button>
      )}
      {failed && rows && <ReloadState onRetry={() => void fetchPage(page + 1, false)} />}
    </div>
  );
}
