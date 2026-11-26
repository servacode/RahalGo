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

import { useCallback, useEffect, useState } from "react";
import { useParams } from "next/navigation";
import { getMessages, defaultLocale, errorText, fmtClockTime, fmtNum } from "@rahalgo/i18n";
import { Alert, Badge, IconCheck, IconChat, LoadingState, OrderChat, Sheet, useLiveRefresh } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { CANCELLABLE, amountText, hasChat, useStoreDeliveries } from "@/lib/store/deliveries";
import type { Delivery } from "@/lib/store/types";
import { fill } from "@/lib/customer/texts";
import { CancelDeliveryButton, DeliveryDetails, StatusPill, statusText } from "@/components/store/DeliveryParts";
import { PageHeader } from "@/components/store/SetupParts";

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
  // **حديثُ الكابتن** (قرارُ المالك ٢٠٢٦-١١-٢٦) — اللوحُ نفسُه الذي عند الزبون (`OrderChat`).
  const [chat, setChat] = useState(false);

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

  const ended = at < 0;
  const progress = Math.min(100, Math.max(8, Math.round((Math.max(at, 0) / LAST) * 100)));
  const mode = d?.delivery_mode;

  return (
    <div className={`console-page ${d && hasChat(d) ? "pb-36" : ""}`} data-testid="store-delivery-track">
      <PageHeader title={fill(D.trackTitle, { n: d?.number ?? "" })} href="/store/delivery" />
      {error && <Alert>{error}</Alert>}
      {!d ? (
        !error && <LoadingState variant="list" rows={2} />
      ) : (
        <div className="grid grid-cols-1 gap-rg-lg md:grid-cols-2 md:items-start md:gap-5">
          <div className="flex min-w-0 flex-col gap-rg-md">
            {/* ── بطاقةُ الحالة الكبيرة وشريطُ التقدّم ── */}
            {ended ? (
              /* **المنتهيةُ رماديّة** (مراجعةُ التصميم ٢٠٢٦-١٠-٢٩) — الإلغاءُ ليس خطأً. */
              <section className="panel flex flex-col gap-rg-sm p-rg-lg">
                <StatusPill status={d.status} mode={mode} />
                <p className="text-body text-ink-muted">
                  {(d.status === "failed" ? ST.failed : d.status === "rejected" ? ST.rejected : ST.cancelled) +
                    (d.cancel_reason?.trim() ? ` — ${d.cancel_reason}` : "")}
                </p>
              </section>
            ) : (
              <section className="flex flex-col gap-rg-sm rounded-card bg-primary-solid p-rg-lg text-on-solid" data-testid="delivery-hero">
                <p className="text-caption opacity-90">{D.trackNow}</p>
                <p className="heading-card leading-snug">{statusText(d.status, mode)}</p>
                {d.status === "dispatching" && <p className="text-caption opacity-90">{D.searchingHint}</p>}
                <div
                  className="h-1.5 overflow-hidden rounded-badge bg-on-solid-tint"
                  role="progressbar"
                  aria-valuemin={0}
                  aria-valuemax={100}
                  aria-valuenow={progress}
                >
                  <div className="h-full rounded-badge bg-on-solid" style={{ width: `${progress}%` }} />
                </div>
              </section>
            )}

            {/* ── كودُ الاستلام في «إحضار» — بارزٌ ── */}
            {fetchMode && d.delivery_code?.trim() && !ended && (
              <section className="rounded-card border-2 border-dashed border-success-edge bg-success-tint p-rg-lg text-center" data-testid="delivery-code">
                <p className="text-caption text-ink-muted">{D.codeTitle}</p>
                <p className="my-rg-sm text-5xl font-bold tracking-[0.3em] text-success" dir="ltr">
                  {d.delivery_code}
                </p>
                <p className="text-caption text-ink-muted">{D.codeHint}</p>
              </section>
            )}

            {fetchMode && ["pending", "approved", "rejected"].includes(d.fetch_amount_status ?? "") && d.fetch_amount != null && (
              <Alert tone={d.fetch_amount_status === "approved" ? "success" : d.fetch_amount_status === "rejected" ? "error" : "warning"}>
                {fill(
                  d.fetch_amount_status === "approved"
                    ? D.amountRowApproved
                    : d.fetch_amount_status === "rejected"
                      ? D.amountRowRejected
                      : D.amountRowPending,
                  { amount: amountText(d.fetch_amount, d.amount_currency) },
                )}
              </Alert>
            )}

            {/* ── خطٌّ زمنيٌّ رأسيٌّ متّصل ── */}
            {!ended && (
              <ol className="panel flex flex-col p-rg-lg" data-testid="delivery-timeline">
                {steps.map((label, i) => {
                  const done = i < at || (i === at && at === LAST);
                  const now = i === at && at < LAST;
                  const last = i === steps.length - 1;
                  const time = i === 0 ? d.created_at : i === 4 ? d.picked_up_at : i === LAST ? d.delivered_at : null;
                  return (
                    <li key={label} className={`flex gap-rg-md ${last ? "" : "pb-rg-lg"}`}>
                      <span className="flex w-6 shrink-0 flex-col items-center">
                        <span
                          className={`flex h-6 w-6 shrink-0 items-center justify-center rounded-badge border-2 ${
                            done
                              ? "border-success bg-success text-on-bright"
                              : now
                                ? "border-primary bg-primary-tint motion-safe:animate-pulse"
                                : "border-line bg-paper"
                          }`}
                        >
                          {done ? <IconCheck size={14} /> : now ? <span className="h-2.5 w-2.5 rounded-badge bg-primary" /> : null}
                        </span>
                        {!last && <span className={`mt-rg-xs w-0.5 flex-1 rounded-badge ${done ? "bg-success" : "bg-line"}`} />}
                      </span>
                      <span className={`min-w-0 flex-1 pt-px text-body ${now ? "font-bold" : done ? "" : "text-ink-muted"}`}>{label}</span>
                      {(done || now) && time && <span className="pt-0.5 text-caption text-ink-muted">{fmtClockTime(time)}</span>}
                    </li>
                  );
                })}
              </ol>
            )}
          </div>

          <div className="flex min-w-0 flex-col gap-rg-lg">
            <section className="panel flex flex-col gap-rg-md p-rg-lg">
              <header className="flex items-center justify-between gap-rg-sm">
                <h2 className="console-h2">{D.trackDetails}</h2>
                <StatusPill status={d.status} mode={mode} />
              </header>
              <DeliveryDetails d={d} />
            </section>
            {/* ── الإلغاءُ ما دام الكابتنُ لم يصل — وإلّا سطرٌ يشرح ── */}
            {CANCELLABLE.has(d.status) ? (
              <div className="flex flex-col gap-rg-xs">
                <CancelDeliveryButton busy={busy} onConfirm={() => void cancel()} />
                <p className="text-center text-caption text-ink-muted">{D.canCancelUntil}</p>
              </div>
            ) : (
              OPEN.has(d.status) && <p className="text-center text-caption text-ink-muted">{fetchMode ? D.noCancelFetch : D.noCancelNow}</p>
            )}
          </div>
        </div>
      )}
      {/* **راسل الكابتن — قرصٌ طافٍ** (قرارُ المالك ٢٠٢٦-١١-٢٦: «عائم متل الزبون») فوق شريط التنقّل، وعليه عدّادُ الجديد. */}
      {d && hasChat(d) && !chat && (
        <button
          type="button"
          aria-label={D.chat}
          title={D.chat}
          onClick={() => setChat(true)}
          data-testid="delivery-chat-fab"
          className="fixed start-4 z-30 flex h-14 w-14 items-center justify-center rounded-full bg-primary-solid text-on-solid shadow-e3"
          style={{ bottom: "calc(5rem + env(safe-area-inset-bottom))" }}
        >
          <IconChat size={24} />
          {(d.chat_unread ?? 0) > 0 && (
            <span className="absolute -top-1 -end-1">
              <Badge variant="danger">{fmtNum(d.chat_unread ?? 0)}</Badge>
            </span>
          )}
        </button>
      )}
      {chat && d && (
        <Sheet open onClose={() => setChat(false)} title={D.chat}>
          {/* **ووصولُ الحديث وسمُ القراءة في المحرّك** — فتُقرأ التوصيلةُ بعده لتسقط شارتُها. */}
          <OrderChat api={api} orderId={d.id} bare streamClass="min-h-60 max-h-[55vh]" onLoaded={() => void load()} />
        </Sheet>
      )}
    </div>
  );
}
