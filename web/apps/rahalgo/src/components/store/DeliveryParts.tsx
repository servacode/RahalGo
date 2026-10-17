"use client";

/**
 * **قطعُ «لدي توصيلة» المشتركة** — صفُّ التوصيلة، وشارةُ حالها، وزرُّ
 * الإلغاء بسؤال. (مرآةُ `DeliveryUi.kt` وصفّ `DeliveryRow` في التطبيق.)
 */

import Link from "next/link";
import { useState } from "react";
import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";
import { Button, Confirm, IconClose, IconHourglass, IconLocation, IconMoto, IconNote, IconPhone, IconUser, IconWallet } from "@rahalgo/ui";
import type { Delivery } from "@/lib/store/types";
import { CANCELLABLE } from "@/lib/store/deliveries";
import { fill } from "@/lib/customer/texts";

const m = getMessages(defaultLocale);
const D = m.store.delivery;
const STATUS = D.status as Record<string, string>;

type Tone = "waiting" | "moving" | "done" | "ended";

/** **مطابقٌ لـ`toneOf` في `DeliveryUi.kt`.** */
export function toneOf(status: string): Tone {
  if (["pending", "accepted", "dispatching"].includes(status)) return "waiting";
  if (["assigned", "at_pickup", "picked_up", "on_the_way", "at_dropoff"].includes(status)) return "moving";
  if (status === "delivered") return "done";
  return "ended";
}

const PILL: Record<Tone, string> = {
  waiting: "bg-warning-tint text-warning",
  moving: "bg-accent-tint text-accent-text",
  done: "bg-success-tint text-success",
  ended: "bg-danger-tint text-danger",
};
const EDGE: Record<Tone, string> = {
  waiting: "border-warning-edge",
  moving: "border-accent-edge",
  done: "border-success-edge",
  ended: "border-danger-edge",
};

export function statusText(status: string): string {
  return STATUS[status] ?? D.searching;
}

export function StatusPill({ status }: { status: string }) {
  const tone = toneOf(status);
  return (
    <span className={`inline-flex items-center gap-1.5 rounded-badge px-2.5 py-1 text-xs font-bold ${PILL[tone]}`}>
      <span className={`h-2 w-2 rounded-badge bg-current ${tone === "waiting" ? "motion-safe:animate-pulse" : ""}`} />
      {statusText(status)}
    </span>
  );
}

export function SearchingBanner() {
  return (
    <div className="flex items-center gap-3 rounded-control border border-warning-edge bg-warning-tint p-3">
      <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-badge bg-warning text-on-bright motion-safe:animate-pulse">
        <IconMoto size={18} />
      </span>
      <span className="min-w-0">
        <span className="block text-sm font-bold text-warning">{D.searching}</span>
        <span className="block text-xs text-ink-muted">{D.searchingHint}</span>
      </span>
    </div>
  );
}

function feeRow(d: Delivery): string {
  const fetch = d.delivery_mode === "fetch";
  const fee = fmtNum(d.delivery_fee);
  if (d.fee_payer === "recipient") return fill(fetch ? D.rowCashFetch : D.rowCash, { fee });
  if (d.fee_payer === "merchant_cash") return fill(D.rowMeCash, { fee });
  return fill(D.rowWallet, { fee });
}

function Line({ icon: Icon, children, strong, muted }: { icon: typeof IconUser; children: React.ReactNode; strong?: boolean; muted?: boolean }) {
  return (
    <p className={`flex items-center gap-2 text-sm ${strong ? "font-bold" : ""} ${muted ? "text-ink-muted" : ""}`}>
      <Icon size={16} className="shrink-0 text-ink-muted" />
      <span className="min-w-0 break-words">{children}</span>
    </p>
  );
}

/** **تفاصيلُ التوصيلة** — في الصفّ وفي شاشة المتابعة نفسِها. */
export function DeliveryLines({ d }: { d: Delivery }) {
  const fetch = d.delivery_mode === "fetch";
  return (
    <div className="flex flex-col gap-1">
      {fetch && (
        <Line icon={IconMoto} strong>
          {D.rowFetch}
        </Line>
      )}
      <Line icon={IconUser} strong>
        {d.customer_name}
      </Line>
      <Line icon={IconPhone}>
        <span dir="ltr">{d.customer_phone}</span>
      </Line>
      <Line icon={IconLocation}>{d.address_text}</Line>
      {!d.dropoff_known && (
        <Line icon={IconLocation} muted>
          {D.rowNoPoint}
        </Line>
      )}
      {d.parcel_note.trim() && <Line icon={IconNote}>{d.parcel_note}</Line>}
      <Line icon={IconWallet}>{feeRow(d)}</Line>
      {d.collect_amount != null && d.collect_amount > 0 && (
        <Line icon={IconWallet}>{fill(fetch ? D.rowCollectFetch : D.rowCollect, { amount: fmtNum(d.collect_amount) })}</Line>
      )}
    </div>
  );
}

export function CancelDeliveryButton({ busy, onConfirm }: { busy: boolean; onConfirm: () => void }) {
  const [asking, setAsking] = useState(false);
  return (
    <>
      <Button variant="secondary" className="flex w-full items-center justify-center gap-2 py-3 text-danger" disabled={busy} onClick={() => setAsking(true)}>
        <IconClose size={16} />
        {busy ? D.cancelling : D.cancel}
      </Button>
      <Confirm
        open={asking}
        title={D.cancelQ}
        body={D.cancelBody}
        confirmLabel={D.cancelYes}
        busy={busy}
        onConfirm={() => {
          setAsking(false);
          onConfirm();
        }}
        onCancel={() => setAsking(false)}
      />
    </>
  );
}

export function DeliveryRow({ d, cancelling, onCancel }: { d: Delivery; cancelling: boolean; onCancel: () => void }) {
  const tone = toneOf(d.status);
  return (
    <article className={`surface flex flex-col gap-2 rounded-card border p-3 ${EDGE[tone]}`} data-testid="store-delivery">
      <Link href={`/store/delivery/${d.id}`} className="flex flex-col gap-2">
        <header className="flex items-center justify-between gap-2">
          <span className="font-bold">{fill(D.trackTitle, { n: d.number })}</span>
          <StatusPill status={d.status} />
        </header>
        {d.status === "dispatching" && <SearchingBanner />}
        <DeliveryLines d={d} />
        <span className="flex items-center gap-1 text-xs text-accent-text">
          <IconHourglass size={12} />
          {D.trackOpen}
        </span>
      </Link>
      {CANCELLABLE.has(d.status) && <CancelDeliveryButton busy={cancelling} onConfirm={onCancel} />}
    </article>
  );
}
