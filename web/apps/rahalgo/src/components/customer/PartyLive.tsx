"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **سؤالُ «تريد استقبال الأمانة؟» ولافتةُ المشاركة — للطرف الآخر في الطلب**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-١١-٢٨؛ الحالةُ في `lib/customer/party.tsx`.) **البطاقةُ لا تُظهر إلّا**: العنوانَ
 * (رقمَ المرسل أو اسمَ المتجر) وحالةَ الطلب وسطرَ الكابتن **وكودَ الاستلام إن وُجد** — لا مالَ ولا غيرُه.
 * **والكابتنُ بالتسمية الثابتةِ لا باسمه**: المحرّكُ لا يبعث اسماً.
 */

import { useState } from "react";
import { createPortal } from "react-dom";
import { getMessages, defaultLocale, withPlatform } from "@rahalgo/i18n";
import { Alert, Button, IconLocation, usePlatform } from "@rahalgo/ui";
import { usePartyOrders, type PartyView } from "@/lib/customer/party";
import { statusLabel } from "@/lib/customer/texts";

const m = getMessages(defaultLocale);
const P = m.shop.party;

function useTitle() {
  const { name } = usePlatform();
  return (o: PartyView): string => {
    if (o.kind === "amanah") {
      return o.sender_phone
        ? P.askAmanah.replace("{phone}", o.sender_phone)
        : withPlatform(P.askAmanahNoPhone, name);
    }
    if (o.kind === "delivery") return P.askDelivery.replace("{store}", o.store_name ?? "");
    return withPlatform(P.askOrder, name);
  };
}

/** **الحالةُ والكابتنُ والكود** — كلُّ ما تُظهره البطاقة بعد العنوان. */
function Details({ o }: { o: PartyView }) {
  const { name } = usePlatform();
  return (
    <ul className="mt-2 space-y-1 text-xs text-ink-muted">
      <li>{P.status.replace("{status}", statusLabel(o.status))}</li>
      {o.captain_assigned && <li>{withPlatform(P.captain, name)}</li>}
      {o.code ? <li className="font-bold text-ink">{P.code.replace("{code}", o.code)}</li> : null}
    </ul>
  );
}

/** **نافذةُ السؤال** — تُركَّب في القشرة، وتُفتح من اللافتة أو من `?party=`. */
export function PartyAsk() {
  const party = usePartyOrders();
  const title = useTitle();
  const [busy, setBusy] = useState(false);
  const [err, setErr] = useState("");
  const o = party.shownId ? (party.orders ?? []).find((x) => x.order_id === party.shownId) : undefined;
  if (!o || o.answer !== "" || typeof document === "undefined") return null;

  const send = async (a: "yes" | "no") => {
    setBusy(true);
    setErr("");
    const e = await party.answer(o.order_id, a);
    setErr(e);
    setBusy(false);
  };

  return createPortal(
    <div className="fixed inset-0 z-[80] flex items-end justify-center scrim p-0 sm:items-center sm:p-4">
      <div role="alertdialog" aria-modal="true" aria-labelledby="party-t" className="w-full surface-modal p-5 sm:max-w-md">
        <h2 id="party-t" className="heading-card">
          {title(o)}
        </h2>
        <Details o={o} />
        {err && <Alert>{err}</Alert>}
        <div className="mt-4 flex flex-col gap-2">
          <button
            type="button"
            disabled={busy}
            onClick={() => void send("yes")}
            className="w-full rounded-control bg-success px-4 py-3 text-sm font-bold text-on-bright transition-opacity hover:opacity-90 disabled:opacity-60"
          >
            {P.yes}
          </button>
          <Button variant="danger" className="w-full py-3" disabled={busy} onClick={() => void send("no")}>
            {P.no}
          </Button>
        </div>
      </div>
    </div>,
    document.body,
  );
}

/** **لافتةُ «بانتظار ردّك»** للرئيسيّة — النقرُ يفتح البطاقة. */
export function PartyWaiting() {
  const party = usePartyOrders();
  const waiting = (party.orders ?? []).filter((o) => o.open && o.answer === "");
  if (waiting.length === 0) return null;
  return (
    <div className="flex flex-col gap-2">
      {waiting.map((o) => (
        <button
          key={o.order_id}
          type="button"
          onClick={() => party.show(o.order_id)}
          className="surface-inset flex w-full items-center gap-3 p-3 text-start"
        >
          <IconLocation size={20} className="shrink-0 text-success" />
          <span className="text-sm font-bold">{o.kind === "amanah" ? P.waitingAmanah : P.waitingOrder}</span>
        </button>
      ))}
    </div>
  );
}

/** **لافتةُ «عم نشارك موقعك»** مع «إيقاف» — لكلّ طلبٍ يشارك. */
export function PartySharing() {
  const party = usePartyOrders();
  const { name } = usePlatform();
  const rows = (party.orders ?? []).filter((o) => party.sharingIds.includes(o.order_id));
  if (rows.length === 0) return null;
  return (
    <>
      {rows.map((o) => (
        <div key={o.order_id} className="surface-inset space-y-1 p-3">
          <div className="flex flex-wrap items-center gap-3">
            <IconLocation size={20} className="shrink-0 text-success motion-safe:animate-pulse" />
            <p className="min-w-0 flex-1 text-sm font-bold">
              {withPlatform(o.kind === "amanah" ? P.sharingAmanah : P.sharingOrder, name)}
            </p>
            <Button variant="secondary" onClick={() => void party.stop(o.order_id)}>
              {P.stop}
            </Button>
          </div>
          <Details o={o} />
        </div>
      ))}
    </>
  );
}
