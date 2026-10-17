"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **‏/store/delivery/[id] — متابعةُ توصيلة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `DeliveryTrackScreen` في `DeliveryTrack.kt`.)
 *
 * **ستُّ مراحل** — والألفاظُ بحسب النوع («السائق في الطريق إليك» للتوصيل،
 * «في الطريق إلى مكان الاستلام» للإحضار). **وتتحدّث وحدَها كلَّ عشر ثوانٍ
 * ما دامت جارية**، وتقف حين تنتهي.
 *
 * **وكودُ الاستلام في «إحضار»** — يُعطى للسائق حين يصل بالغرض، **بس وقتها.**
 */

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { getMessages, defaultLocale, errorText, fmtClockTime, fmtNum } from "@rahalgo/i18n";
import { Alert, LoadingState, useLiveRefresh } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { CANCELLABLE, useStoreDeliveries } from "@/lib/store/deliveries";
import type { Delivery } from "@/lib/store/types";
import { fill } from "@/lib/customer/texts";
import { CancelDeliveryButton, DeliveryLines, SearchingBanner, StatusPill } from "@/components/store/DeliveryParts";

const m = getMessages(defaultLocale);
const D = m.store.delivery;
const LAST = 5;
const OPEN = new Set(["pending", "accepted", "dispatching", "assigned", "at_pickup", "picked_up", "on_the_way", "at_dropoff"]);

/** **مطابقٌ لـ`stepOf` في `DeliveryTrack.kt`.** */
function stepOf(status: string): number {
  switch (status) {
    case "pending":
      return 0;
    case "accepted":
    case "dispatching":
      return 1;
    case "assigned":
      return 2;
    case "at_pickup":
      return 3;
    case "picked_up":
    case "on_the_way":
    case "at_dropoff":
      return 4;
    case "delivered":
      return LAST;
    default:
      return -1;
  }
}

export default function DeliveryTrack() {
  const params = useParams<{ id: string }>();
  const id = params?.id ?? "";
  const { reload: reloadList } = useStoreDeliveries();
  const [d, setD] = useState<Delivery | null>(null);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    if (!id) return;
    try {
      setD(await api<Delivery>(`/api/v1/merchant/deliveries/${id}`));
      setError("");
    } catch (e) {
      setError(errorText(e));
    }
  }, [id]);

  useEffect(() => {
    void load();
  }, [load]);
  const status = d?.status ?? "";
  useEffect(() => {
    if (status && !OPEN.has(status)) return;
    const t = setInterval(() => void load(), 10_000);
    return () => clearInterval(t);
  }, [load, status]);
  useLiveRefresh(["order", "delivery"], () => void load());

  async function cancel() {
    if (busy) return;
    setBusy(true);
    try {
      setD(await api<Delivery>(`/api/v1/merchant/deliveries/${id}/cancel`, { method: "POST", body: "{}" }));
      void reloadList();
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  const fetchMode = d?.delivery_mode === "fetch";
  const steps = fetchMode
    ? [D.steps.platform, D.steps.waiting, D.steps.toPickupFetch, D.steps.atPickupFetch, D.steps.toYouFetch, D.steps.delivered]
    : [D.steps.platform, D.steps.waiting, D.steps.toYou, D.steps.atYou, D.steps.toRecipient, D.steps.delivered];
  const at = d ? stepOf(d.status) : -1;
  const ST = D.status as Record<string, string>;

  return (
    <div className="flex flex-col gap-3">
      <Link href="/store/delivery" className="text-sm text-accent-text">
        {D.back}
      </Link>
      <header>
        <h2 className="heading-page">{fill(D.trackTitle, { n: d?.number ?? "" })}</h2>
        <p className="text-sm text-ink-muted">{D.trackHint}</p>
      </header>
      {error && <Alert>{error}</Alert>}
      {!d ? (
        !error && <LoadingState variant="list" rows={2} />
      ) : (
        <>
          {fetchMode && d.delivery_code?.trim() && (
            <section className="rounded-card border border-success-edge bg-success-tint p-4 text-center">
              <p className="font-bold">{D.codeTitle}</p>
              <p className="my-2 text-4xl font-bold tracking-widest text-success" dir="ltr">
                {d.delivery_code}
              </p>
              <p className="text-xs text-ink-muted">{D.codeHint}</p>
            </section>
          )}
          {at < 0 ? (
            <Alert>
              {(d.status === "failed" ? ST.failed : d.status === "rejected" ? ST.rejected : ST.cancelled) +
                (d.cancel_reason?.trim() ? ` — ${d.cancel_reason}` : "")}
            </Alert>
          ) : (
            <>
              {d.status === "dispatching" && <SearchingBanner />}
              <ol className="surface flex flex-col gap-1 rounded-card p-4">
                {steps.map((label, i) => {
                  const done = i < at || (i === at && at === LAST);
                  const now = i === at && at < LAST;
                  const time = i === 0 ? d.created_at : i === 4 ? d.picked_up_at : i === LAST ? d.delivered_at : null;
                  return (
                    <li key={label} className="flex items-center gap-3 py-1.5">
                      <span
                        className={`h-3.5 w-3.5 shrink-0 rounded-badge ${
                          now
                            ? `${at === 0 ? "bg-warning" : "bg-accent"} motion-safe:animate-pulse`
                            : done
                              ? "bg-success"
                              : "bg-ink-faint"
                        }`}
                      />
                      <span className={`flex-1 text-sm ${now ? "font-bold" : done ? "" : "text-ink-muted"}`}>{label}</span>
                      {(done || now) && time && <span className="text-xs text-ink-muted">{fmtClockTime(time)}</span>}
                    </li>
                  );
                })}
              </ol>
            </>
          )}
          <section className="surface flex flex-col gap-2 rounded-card p-4">
            <header className="flex items-center justify-between gap-2">
              <h3 className="heading-card">{D.trackDetails}</h3>
              <StatusPill status={d.status} />
            </header>
            <DeliveryLines d={d} />
          </section>
          {fetchMode && ["pending", "approved", "rejected"].includes(d.fetch_amount_status ?? "") && d.fetch_amount != null && (
            <Alert tone={d.fetch_amount_status === "approved" ? "success" : d.fetch_amount_status === "rejected" ? "error" : "warning"}>
              {fill(
                d.fetch_amount_status === "approved"
                  ? D.amountRowApproved
                  : d.fetch_amount_status === "rejected"
                    ? D.amountRowRejected
                    : D.amountRowPending,
                { amount: fmtNum(d.fetch_amount) },
              )}
            </Alert>
          )}
          {CANCELLABLE.has(d.status) && <CancelDeliveryButton busy={busy} onConfirm={() => void cancel()} />}
        </>
      )}
    </div>
  );
}
