"use client";

/**
 * **قطعُ «اطلب توصيلة» المشتركة** — بطاقةُ التوصيلة، وشارةُ حالها، وتفاصيلُها،
 * وزرُّ الإلغاء بسؤال. (مرآةُ `DeliveryUi.kt` وصفّ `DeliveryRow` في التطبيق.)
 *
 * **التصميمُ المعتمد ٢٠٢٦-١١-٢٧** (نموذجُ المالك): بطاقةٌ لكلّ توصيلة، والحالةُ شارةٌ ملوّنة،
 * والجاريةُ بشريطٍ جانبيّ، و«إحضار» وسمٌ برتقاليّ.
 */

import Link from "next/link";
import { useState, type ComponentType } from "react";
import { getMessages, defaultLocale, fmtNum, fmtRelStamp } from "@rahalgo/i18n";
import { Badge, Button, Confirm, IconBag, IconChat, IconClose, IconLocation, IconMoto, IconPhone, IconUser, IconWallet } from "@rahalgo/ui";
import type { Delivery } from "@/lib/store/types";
import { CANCELLABLE, amountText } from "@/lib/store/deliveries";
import { fill } from "@/lib/customer/texts";

const m = getMessages(defaultLocale);
const D = m.store.delivery;
const STATUS = D.status as Record<string, string>;

type Tone = "waiting" | "moving" | "done" | "ended";
type BadgeVariant = "warning" | "primary" | "success" | "danger" | "neutral";

/** **مطابقٌ لـ`toneOf` في `DeliveryUi.kt`.** */
export function toneOf(status: string): Tone {
  if (["pending", "accepted", "dispatching"].includes(status)) return "waiting";
  if (["assigned", "at_pickup", "picked_up", "on_the_way", "at_dropoff"].includes(status)) return "moving";
  if (status === "delivered") return "done";
  return "ended";
}

/** **لونُ الحالة** — انتظارٌ برتقاليّ، وجاريةٌ بلون العلامة، وتمّت خضراء، والملغاةُ حمراء. */
export function badgeOf(status: string): BadgeVariant {
  if (["cancelled", "failed", "rejected"].includes(status)) return "danger";
  const tone = toneOf(status);
  return tone === "waiting" ? "warning" : tone === "moving" ? "primary" : tone === "done" ? "success" : "neutral";
}

/** **لفظُ الحالة** — وللإحضار ألفاظٌ خاصّةٌ بعد الإسناد («في الطريق إلى مكان الاستلام»). */
export function statusText(status: string, mode?: string): string {
  if (mode === "fetch") {
    const f = (D.statusFetch as Record<string, string>)[status];
    if (f) return f;
  }
  return STATUS[status] ?? D.searching;
}

export function StatusPill({ status, mode }: { status: string; mode?: string }) {
  const tone = toneOf(status);
  return (
    <Badge variant={badgeOf(status)} className="gap-rg-sm py-1 font-bold">
      <span className={`h-2 w-2 rounded-badge bg-current ${tone === "waiting" ? "motion-safe:animate-pulse" : ""}`} aria-hidden />
      {statusText(status, mode)}
    </Badge>
  );
}

export function SearchingBanner() {
  return (
    <div className="flex items-center gap-rg-md rounded-control border border-warning-edge bg-warning-tint p-rg-md">
      <span className="flex h-10 w-10 shrink-0 items-center justify-center rounded-badge bg-warning text-on-bright motion-safe:animate-pulse">
        <IconMoto size={18} />
      </span>
      <span className="min-w-0 text-body font-bold text-warning">{D.searching}</span>
    </div>
  );
}

/** **الأجرةُ ومن يدفعها** — بلا كلمة «الأجرة» (تسبقها في تسميةٍ أو في البطاقة). */
export function feeValue(d: Delivery): string {
  const fetch = d.delivery_mode === "fetch";
  const fee = fmtNum(d.delivery_fee);
  if (d.fee_payer === "recipient") return fill(fetch ? D.feeValCashFetch : D.feeValCash, { fee });
  if (d.fee_payer === "merchant_cash") return fill(D.feeValMeCash, { fee });
  return fill(D.feeValWallet, { fee });
}

function DetailRow({ icon: Icon, label, children, ltr }: { icon: ComponentType<{ size?: number; className?: string }>; label: string; children: React.ReactNode; ltr?: boolean }) {
  return (
    <li className="flex items-start gap-rg-md border-t border-line-soft py-rg-sm first:border-t-0 first:pt-0 last:pb-0">
      <span className="flex h-8 w-8 shrink-0 items-center justify-center rounded-control bg-field text-ink-muted" aria-hidden>
        <Icon size={16} />
      </span>
      <span className="min-w-0 flex-1">
        <span className="block text-caption text-ink-muted">{label}</span>
        <span className="block break-words text-body" dir={ltr ? "ltr" : undefined} style={ltr ? { textAlign: "start" } : undefined}>
          {children}
        </span>
      </span>
    </li>
  );
}

/** **بطاقةُ تفاصيل التوصيلة** — أيقونةٌ وتسميةٌ وقيمةٌ لكلّ سطر (شاشةُ المتابعة). */
export function DeliveryDetails({ d }: { d: Delivery }) {
  const fetch = d.delivery_mode === "fetch";
  return (
    <ul className="flex flex-col">
      {fetch && (
        <DetailRow icon={IconMoto} label={D.rowType}>
          {D.rowFetch}
        </DetailRow>
      )}
      <DetailRow icon={IconUser} label={fetch ? D.rowWhoFetch : D.rowWho}>
        {d.customer_name}
      </DetailRow>
      <DetailRow icon={IconPhone} label={D.rowPhone} ltr>
        {d.customer_phone}
      </DetailRow>
      <DetailRow icon={IconLocation} label={fetch ? D.rowAddrFetch : D.rowAddr}>
        {d.address_text}
        {!d.dropoff_known && <span className="block text-caption text-ink-muted">{D.rowNoPoint}</span>}
      </DetailRow>
      {d.parcel_note.trim() && (
        <DetailRow icon={IconBag} label={D.rowParcel}>
          {d.parcel_note}
        </DetailRow>
      )}
      <DetailRow icon={IconWallet} label={D.rowFee}>
        {feeValue(d)}
      </DetailRow>
      {d.collect_amount != null && d.collect_amount > 0 && (
        <DetailRow icon={IconWallet} label={fetch ? D.rowCollectLabelFetch : D.rowCollectLabel}>
          {fill(fetch ? D.rowCollectValFetch : D.rowCollectVal, { amount: amountText(d.collect_amount, d.amount_currency) })}
        </DetailRow>
      )}
    </ul>
  );
}

export function CancelDeliveryButton({ busy, onConfirm, compact = false }: { busy: boolean; onConfirm: () => void; compact?: boolean }) {
  const [asking, setAsking] = useState(false);
  return (
    <>
      <Button
        variant={compact ? "ghost" : "secondary"}
        className={
          compact
            ? "flex min-h-9 items-center gap-rg-xs text-danger"
            : "flex min-h-12 w-full items-center justify-center gap-rg-sm border border-danger-edge py-rg-md text-danger"
        }
        disabled={busy}
        onClick={() => setAsking(true)}
      >
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
 * **بطاقةُ التوصيلة** — رقمُها وحالُها شارةً ملوّنة، ثمّ المستلمُ والعنوانُ والغرضُ والأجرة.
 * **واللمسةُ تفتح المتابعة**، والإلغاءُ زرٌّ تحتها إن جاز.
 */
export function DeliveryRow({ d, cancelling, onCancel }: { d: Delivery; cancelling: boolean; onCancel: () => void }) {
  const tone = toneOf(d.status);
  const fetch = d.delivery_mode === "fetch";
  const ended = tone === "ended";
  return (
    <article
      className={`surface flex flex-col gap-rg-sm rounded-card border border-line p-rg-md ${
        tone === "moving" ? "border-s-4 border-s-primary" : ""
      } ${ended ? "opacity-75" : ""}`}
      data-testid="store-delivery"
    >
      <Link href={`/store/delivery/${d.id}`} className="flex flex-col gap-rg-xs text-start" aria-label={`${fill(D.trackTitle, { n: d.number })} · ${D.trackDetails}`}>
        <span className="flex w-full items-center justify-between gap-rg-sm">
          <span className="flex min-w-0 items-baseline gap-rg-sm">
            <span className="font-bold">{fill(D.trackTitle, { n: d.number })}</span>
          </span>
          <span className="flex shrink-0 items-center gap-rg-xs">
            {/* **رسائلُ الكابتن التي لم تُقرأ** (قرارُ المالك ٢٠٢٦-١١-٢٦). */}
            {(d.chat_unread ?? 0) > 0 && (
              <Badge variant="primary">
                <span className="flex items-center gap-1" aria-label={D.chatUnread}>
                  <IconChat size={14} />
                  {fmtNum(d.chat_unread ?? 0)}
                </span>
              </Badge>
            )}
            <StatusPill status={d.status} mode={d.delivery_mode} />
          </span>
        </span>
        <span className="flex items-center gap-rg-sm font-bold">
          {fetch && <Badge variant="warning">{D.tagFetch}</Badge>}
          <span className="min-w-0 truncate">{d.customer_name}</span>
        </span>
        <span className="line-clamp-2 text-caption text-ink-muted">{d.address_text}</span>
        <span className="mt-rg-xs flex flex-wrap items-center gap-rg-xs text-caption text-ink-muted">
          {d.parcel_note.trim() && <span className="rounded-badge bg-field px-rg-sm py-0.5">{d.parcel_note}</span>}
          <span className="rounded-badge bg-field px-rg-sm py-0.5">{feeValue(d)}</span>
          {/* **ويومُها وساعتُها** (مرآةُ `Dates.stamp`، تدقيقُ ٢٠٢٦-١٠-٢٧). */}
          {d.created_at && <span>{fmtRelStamp(d.created_at)}</span>}
        </span>
      </Link>
      {CANCELLABLE.has(d.status) && <CancelDeliveryButton compact busy={cancelling} onConfirm={onCancel} />}
    </article>
  );
}
