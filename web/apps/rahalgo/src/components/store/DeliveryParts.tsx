"use client";

/**
 * **قطعُ «اطلب توصيلة» المشتركة** — صفُّ التوصيلة، وشارةُ حالها، وزرُّ
 * الإلغاء بسؤال. (مرآةُ `DeliveryUi.kt` وصفّ `DeliveryRow` في التطبيق.)
 */

import Link from "next/link";
import { useState } from "react";
import { getMessages, defaultLocale, fmtMoney, fmtNum, fmtRelStamp } from "@rahalgo/i18n";
import { Button, Confirm, IconChevronDown, IconClose, IconLocation, IconMoto, IconNote, IconPhone, IconUser, IconWallet } from "@rahalgo/ui";
import type { Delivery } from "@/lib/store/types";
import { CANCELLABLE, amountText } from "@/lib/store/deliveries";
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
  // **المنتهيةُ والملغاةُ رماديّة** (مراجعةُ التصميم ٢٠٢٦-١٠-٢٩) — أُلغيت بيده غالباً، فليست خطأً.
  ended: "bg-field text-ink-muted",
};
const EDGE: Record<Tone, string> = {
  waiting: "border-warning-edge",
  moving: "border-accent-edge",
  done: "border-success-edge",
  ended: "border-line",
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
      <span className="min-w-0 text-sm font-bold text-warning">{D.searching}</span>
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
        <Line icon={IconWallet}>{fill(fetch ? D.rowCollectFetch : D.rowCollect, { amount: amountText(d.collect_amount, d.amount_currency) })}</Line>
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

/**
 * **صفُّ التوصيلة مطويّاً** — رقمُها وحالُها، ثمّ المستلمُ والأجرة في سطر
 * (مراجعةُ التصميم ٢٠٢٦-١٠-٢٩: كانت كلُّ بطاقةٍ تحمل كلَّ شيء). **والضغطُ يفتحها**:
 * التفاصيلُ والإلغاءُ ورابطُ المتابعة.
 */
export function DeliveryRow({ d, cancelling, onCancel }: { d: Delivery; cancelling: boolean; onCancel: () => void }) {
  const tone = toneOf(d.status);
  const [open, setOpen] = useState(false);
  return (
    <article className={`surface flex flex-col gap-2 rounded-card border p-3 ${EDGE[tone]}`} data-testid="store-delivery">
      <button
        type="button"
        aria-expanded={open}
        aria-label={`${fill(D.trackTitle, { n: d.number })} · ${D.trackDetails}`}
        onClick={() => setOpen((v) => !v)}
        className="flex flex-col gap-1 text-start"
        data-testid="store-delivery-toggle"
      >
        <span className="flex w-full items-center justify-between gap-2">
          <span className="flex min-w-0 items-baseline gap-2">
            <span className="font-bold">{fill(D.trackTitle, { n: d.number })}</span>
            {/* **ويومُها وساعتُها** (مرآةُ `Dates.stamp`، تدقيقُ ٢٠٢٦-١٠-٢٧). */}
            {d.created_at && <span className="truncate text-xs text-ink-muted">{fmtRelStamp(d.created_at)}</span>}
          </span>
          <StatusPill status={d.status} />
        </span>
        <span className="flex w-full items-center justify-between gap-2 text-sm">
          <span className="min-w-0 truncate">{d.customer_name}</span>
          <span className="flex shrink-0 items-center gap-1 text-ink-muted">
            <span className="tabular-nums">{fmtMoney(d.delivery_fee)}</span>
            <IconChevronDown size={16} className={`transition-transform ${open ? "rotate-180" : ""}`} />
          </span>
        </span>
      </button>
      {open && (
        <>
          {d.status === "dispatching" && <SearchingBanner />}
          <DeliveryLines d={d} />
          <Link href={`/store/delivery/${d.id}`} className="text-sm font-bold text-primary hover:underline">
            {tone === "waiting" || tone === "moving" ? D.follow : D.trackDetails}
          </Link>
          {CANCELLABLE.has(d.status) && <CancelDeliveryButton busy={cancelling} onConfirm={onCancel} />}
        </>
      )}
    </article>
  );
}
