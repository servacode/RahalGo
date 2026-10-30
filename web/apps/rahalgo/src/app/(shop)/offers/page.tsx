"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **العروض — الحسومُ الساريةُ وحدَها** (مرآةُ `Offers` في `MineScreens.kt`)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **عامّةٌ كالسوق** (`/public/offers` بلا حساب) **وتحمل موضعَ التوصيل** —
 * فلا يُعرض حسمٌ من مدينةٍ لا نوصل منها. **والسريانُ يقوله الخادم.**
 *
 * **والعرضُ على صنفٍ يُضغط فتُفتح نافذةُ الصنف** — بخياراته إن كانت له،
 * **والسعرُ فيها سعرُ المحرّك لا سعرُ البطاقة.**
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, errorText, fmtMoney, fmtNum, fmtSpan } from "@rahalgo/i18n";
import { Badge, EmptyState, LoadingState, Money, ReloadState, IconPromos, IconStore } from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import { useBrowseQuery, useCustomer } from "@/lib/customer/context";
import { fill, freeOfferLines } from "@/lib/customer/texts";
import type { FreeDelivery, Item, Offer } from "@/lib/customer/types";
import { ItemSheet } from "@/components/customer/items";
import { ShopPageHeader } from "@/components/customer/ShopPageHeader";

const m = getMessages(defaultLocale);
const F = m.shop.offers;
const FR = m.shop.free;

/**
 * **بطاقةُ عرض التوصيل المجاني — أوّلَ العروض** (مرآةُ `Offers` في `MineScreens.kt`).
 *
 * **كانت الصفحةُ تقول «لا عروض سارية» ولزبونها توصيلةٌ مجانيّة.** والأهليّةُ من الخادم.
 */
function FreeOfferCard({ fd }: { fd: FreeDelivery }) {
  const kinds = [
    fd.standard ? FR.kindStandard : "",
    fd.purchase ? FR.kindPurchase : "",
    fd.amanah ? FR.kindAmanah : "",
  ]
    .filter(Boolean)
    .join(m.common.listSep);
  return (
    <section className="surface flex flex-col gap-1 border border-success-edge p-4">
      <h2 className="text-base font-bold text-success">{FR.offerTitle}</h2>
      {freeOfferLines(fd, fill(FR.offerKinds, { kinds })).map((line) => (
        <p key={line} className="text-sm">
          {line}
        </p>
      ))}
    </section>
  );
}

/** **صنفُ العرض** — هيكلٌ يكفي النافذةَ لتجلب تفاصيلَه من المحرّك. */
function asItem(o: Offer): Item | null {
  if (!o.menu_item_id) return null;
  return {
    id: o.menu_item_id,
    name: o.item_name || o.title,
    description: "",
    price: o.price_after,
    merchant_name: o.merchant_name ?? null,
    image_url: o.item_image_url ?? o.image_url,
    image_thumb_url: o.item_image_url ?? o.image_url,
    available: true,
    source_closed: false,
    source_opens_at: null,
    section_id: "",
    section_name: "",
    price_before: o.price_before || null,
    discount_percent: o.discount_percent,
    has_options: o.has_options,
  };
}

function left(endsAt: string | null): string {
  if (!endsAt) return "";
  const end = new Date(endsAt).getTime();
  if (Number.isNaN(end)) return "";
  const sec = (end - Date.now()) / 1000;
  return sec > 0 ? fill(F.left, { t: fmtSpan(sec) }) : "";
}

export default function OffersPage() {
  const scope = useBrowseQuery("?");
  const { freeDelivery } = useCustomer();
  const free = freeDelivery && (freeDelivery.standard || freeDelivery.purchase || freeDelivery.amanah) ? freeDelivery : null;
  const [offers, setOffers] = useState<Offer[] | null>(null);
  const [error, setError] = useState("");
  const [open, setOpen] = useState<Item | null>(null);

  const load = useCallback(() => {
    setError("");
    api<{ offers: Offer[] }>(`/api/v1/public/offers${scope}`)
      .then((r) => setOffers((r.offers ?? []).filter((o) => o.live !== false)))
      .catch((err) => setError(errorText(err)));
  }, [scope]);

  useEffect(load, [load]);

  return (
    <div className="flex flex-col gap-4">
      <ShopPageHeader title={F.title} />
      {free && <FreeOfferCard fd={free} />}
      {error && !offers ? (
        <ReloadState label={error} onRetry={load} />
      ) : offers === null ? (
        <LoadingState variant="list" rows={3} />
      ) : offers.length === 0 ? (
        free ? null : <EmptyState icon={IconPromos} title={F.empty} action={<span className="text-xs text-ink-muted">{F.emptyHint}</span>} />
      ) : (
        <ul className="grid grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3">
          {offers.map((o) => {
            const item = asItem(o);
            const img = mediaUrl(o.item_image_url ?? o.image_url);
            const saved = Math.max(0, o.price_before - o.price_after);
            const rest = left(o.ends_at);
            const body = (
              <>
                <span className="h-16 w-16 shrink-0 overflow-hidden rounded-control bg-field">
                  {img && <img src={img} alt="" loading="lazy" className="h-full w-full object-cover" />}
                </span>
                <span className="flex min-w-0 flex-1 flex-col gap-1">
                  <span className="text-sm font-bold">{o.title || o.item_name}</span>
                  {o.merchant_name && (
                    <span className="flex items-center gap-1 truncate text-xs text-ink-muted">
                      <IconStore size={11} />
                      {o.merchant_name}
                    </span>
                  )}
                  {o.body && <span className="line-clamp-2 text-xs text-ink-muted">{o.body}</span>}
                  <span className="flex flex-wrap items-center gap-2">
                    {o.price_after > 0 && <Money value={o.price_after} className="font-bold text-accent-text" />}
                    {o.price_before > o.price_after && (
                      <span className="text-xs text-ink-muted line-through">
                        <Money value={o.price_before} small />
                      </span>
                    )}
                    {o.discount_percent != null && o.discount_percent > 0 && (
                      <Badge variant="accent">{`-${fmtNum(o.discount_percent)}%`}</Badge>
                    )}
                  </span>
                  {saved > 0 && <span className="text-xs text-success">{fill(F.saves, { amount: fmtMoney(saved) })}</span>}
                  {rest && <span className="text-xs text-ink-muted">{rest}</span>}
                </span>
              </>
            );
            return (
              <li key={o.id}>
                {item ? (
                  <button
                    type="button"
                    onClick={() => setOpen(item)}
                    aria-label={`${F.order} — ${o.title || o.item_name}`}
                    className="surface flex w-full items-start gap-3 p-3 text-start transition-transform active:scale-[0.99]"
                  >
                    {body}
                  </button>
                ) : (
                  <div className="surface flex items-start gap-3 p-3">{body}</div>
                )}
              </li>
            );
          })}
        </ul>
      )}
      <ItemSheet item={open} onClose={() => setOpen(null)} />
    </div>
  );
}
