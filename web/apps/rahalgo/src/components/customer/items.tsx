"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **بطاقةُ الصنف ونافذتُه — مرآةُ `ItemCard` و`ItemOptionsSheet`**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **السعرُ بالدولار و«≈ بالليرة» على سطرٍ واحد** — الدولارُ ما كتبه صاحبُ
 * المتجر، **والليرةُ ما يدفعه الزبون** (يحسبها المحرّكُ من سعر الصرف).
 *
 * **واسمُ المتجر تحت الصورة** — «شاورما» من مطعمين اسمان متطابقان،
 * **والمتجرُ هو ما يفرّق بينهما.**
 *
 * **والمغلقُ والمنتهي يُحجبان بحجابٍ يقول السبب** — المتجرُ مغلقٌ الآن، أو
 * الصنفُ غيرُ متوفّر. **ولا زرَّ إضافةٍ عليهما.**
 */

import { useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { getMessages, defaultLocale, errorText, fmtNum } from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  Checkbox,
  Confirm,
  LoadingState,
  Money,
  Radio,
  Sheet,
  FavoriteButton,
  IconAdd,
  IconMinus,
  IconStore,
} from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import { useCart, unitPrice } from "@/lib/customer/cart";
import { useFav } from "@/lib/customer/favorites";
import type { Item, ItemDetail, ModifierGroup, ModifierOption } from "@/lib/customer/types";
import { dollars, fill, backAt } from "@/lib/customer/texts";
import { useShopToast } from "./toast";
import { useAddGate, AvailabilityNotice } from "./ServiceNotices";
import { AddressSheet } from "./AddressSheet";
import { useCustomer } from "@/lib/customer/context";
import { useAuth } from "@/lib/auth";

const m = getMessages(defaultLocale);
const I = m.shop.item;
const S = m.shop.stores;
const SV = m.shop.service;

/** **السعرُ على سطرٍ واحد** — دولارٌ ثمّ «≈ ليرة»، أو ليرةٌ وحدَها. */
export function Price({ syp, usd, className = "" }: { syp: number; usd?: number | null; className?: string }) {
  if (usd == null) return <Money value={syp} className={`font-bold text-accent-text ${className}`} />;
  return (
    <span className={`inline-flex flex-wrap items-baseline gap-x-1.5 ${className}`}>
      <span dir="ltr" className="font-bold text-accent-text tabular-nums">
        {dollars(usd)}
      </span>
      <span className="text-xs text-ink-muted">
        {"≈ "}
        <Money value={syp} small />
      </span>
    </span>
  );
}

function closedLabel(item: Item): string {
  if (item.source_closed) {
    const t = backAt(item.source_opens_at);
    return t ? `${I.storeClosed} — ${fill(I.opensAt, { time: t })}` : I.storeClosed;
  }
  return I.unavailable;
}

/**
 * **بطاقةُ الصنف — كبطاقة تطبيق الزبون حرفاً** (ملاحظةُ المالك ٢٠٢٦-١٠-١٧: «الشريط العلوي
 * تبع السعر لسا مختلف»): إطارٌ فيه **شريطٌ علويٌّ** (القلبُ ثمّ السعرُ بالدولار و«≈ ليرة» على
 * سطرٍ واحد) **ثمّ الصورة ثمّ شريطٌ سفليّ** (اسمُ المتجر و«+»)، **واسمُ الصنف تحت الإطار.**
 *
 * **والقلبُ زرٌّ مستقلٌّ في الشريط** — لا داخلَ زرِّ البطاقة: زرٌّ في زرٍّ لا يصحّ.
 */
export function ItemCard({
  item,
  onOpen,
  storeLink = true,
}: {
  item: Item;
  onOpen: (item: Item) => void;
  /** **اسمُ المتجر رابطٌ لصفحته** (٢٠٢٦-١٠-١٨) — ويُطفأ في صفحة المتجر نفسِها. */
  storeLink?: boolean;
}) {
  const closed = item.source_closed || !item.available;
  const img = mediaUrl(item.image_thumb_url ?? item.image_url);
  const fav = useFav();
  return (
    <div className="flex min-w-0 flex-col gap-1.5">
      <div className="surface flex min-w-0 flex-col overflow-hidden">
        {/* ── الشريطُ العلويّ: القلبُ والسعر ── */}
        <div className="flex min-w-0 items-center gap-1 px-1.5 py-1">
          <FavoriteButton
            itemID={item.id}
            on={fav.has(item.id)}
            onToggle={fav.toggle}
            onRequireLogin={fav.needLogin}
            size="sm"
          />
          <span className="ms-auto flex min-w-0 items-baseline gap-1 whitespace-nowrap">
            {item.price_usd != null ? (
              <>
                <span dir="ltr" className="text-sm font-bold text-accent-text tabular-nums">
                  {dollars(item.price_usd)}
                </span>
                <span className="truncate text-2xs text-ink-muted">
                  {"≈ "}
                  <Money value={item.price} small />
                </span>
              </>
            ) : (
              <Money value={item.price} className="text-sm font-bold text-accent-text" />
            )}
          </span>
        </div>
        <button
          type="button"
          onClick={() => onOpen(item)}
          className="flex min-w-0 flex-col text-start transition-transform active:scale-[0.98]"
        >
          {/* **مربّعٌ ثابتٌ مهما كانت الصورة** (ملاحظةُ المالك ٢٠٢٦-١٠-١٩) — صورةٌ طويلةٌ كانت تمطّ
              بطاقتَها، **فالصورةُ مطلقةٌ داخلَ المربّع وتُقصّ**، والبطاقاتُ بطولٍ واحد. */}
          <div className="relative aspect-square w-full overflow-hidden bg-field">
            {img && <img src={img} alt="" loading="lazy" className="absolute inset-0 h-full w-full object-cover" />}
            {closed && (
              <div className="scrim absolute inset-0 flex items-center justify-center p-2 text-center text-sm font-bold text-on-solid">
                {closedLabel(item)}
              </div>
            )}
            {item.discount_percent != null && item.discount_percent > 0 && (
              <span className="absolute start-2 top-2">
                <Badge variant="accent">{`-${fmtNum(item.discount_percent)}%`}</Badge>
              </span>
            )}
          </div>
        </button>
        {/* ── الشريطُ السفليّ: اسمُ المتجر (رابطٌ لصفحته) و«+» — خارجَ زرّ الصورة: رابطٌ في زرٍّ لا يصحّ ── */}
        <div className="flex min-h-10 w-full min-w-0 items-center gap-1 px-2 py-1">
          {storeLink && item.merchant_id && item.merchant_name ? (
            <Link
              href={`/stores/${item.merchant_id}`}
              aria-label={`${S.openStore} — ${item.merchant_name}`}
              className="flex-1 truncate text-2xs text-ink-muted hover:text-accent-text hover:underline"
            >
              {item.merchant_name}
            </Link>
          ) : (
            <span className="flex-1 truncate text-2xs text-ink-muted">{item.merchant_name ?? ""}</span>
          )}
          {!closed && (
            <button
              type="button"
              aria-label={I.add}
              onClick={() => onOpen(item)}
              className="flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-accent text-on-bright"
            >
              <IconAdd size={15} />
            </button>
          )}
        </div>
      </div>
      {/* ── اسمُ الصنف تحت الإطار، في الوسط ── */}
      <button type="button" tabIndex={-1} onClick={() => onOpen(item)} className="min-w-0 text-center">
        <p className="line-clamp-2 text-sm font-medium leading-snug">{item.name}</p>
        {item.price_before != null && (
          <p className="text-xs text-ink-muted line-through">
            {item.price_before_usd != null ? dollars(item.price_before_usd) : <Money value={item.price_before} small />}
          </p>
        )}
      </button>
    </div>
  );
}

/** **شبكةُ الأصناف** — عمودان على الجوّال وتتّسع مع الشاشة. */
export function ItemGrid({
  items,
  onOpen,
  storeLink = true,
}: {
  items: Item[];
  onOpen: (item: Item) => void;
  storeLink?: boolean;
}) {
  return (
    <div className="grid grid-cols-2 gap-3 sm:grid-cols-3 md:grid-cols-4 lg:grid-cols-5 xl:grid-cols-6">
      {items.map((it) => (
        <ItemCard key={it.id} item={it} onOpen={onOpen} storeLink={storeLink} />
      ))}
    </div>
  );
}

/** **أيُّ مجموعةٍ إلزاميّةٍ لم يُختر منها؟** — والمحرّكُ يردّ السلّةَ كلَّها إن نقصت. */
function missingGroup(groups: ModifierGroup[], picked: Record<string, string[]>): ModifierGroup | null {
  for (const g of groups) {
    if (g.min_select > 0 && (picked[g.id]?.length ?? 0) < g.min_select) return g;
  }
  return null;
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **نافذةُ الصنف — تفاصيلُه وخياراتُه وعددُه ثمّ «أضف»**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **وصنفٌ من متجرٍ غيرِ متجر السلّة يُسأل عنه** — الطلبُ الواحدُ من متجرٍ
 * واحد (`max_sources`)، **ومن أضاف ولم يعلم اكتشف عند الإرسال.**
 */
export function ItemSheet({
  item,
  onClose,
  storeLink = true,
}: {
  item: Item | null;
  onClose: () => void;
  /** **اسمُ المتجر رابطٌ لصفحته** — ويُطفأ في صفحة المتجر نفسِها. */
  storeLink?: boolean;
}) {
  const cart = useCart();
  const toast = useShopToast();
  const [detail, setDetail] = useState<ItemDetail | null>(null);
  const [loadError, setLoadError] = useState("");
  const [picked, setPicked] = useState<Record<string, string[]>>({});
  const [qty, setQty] = useState(1);
  const [warn, setWarn] = useState("");
  const [askOther, setAskOther] = useState(false);
  const fav = useFav();
  // **بوّابةُ الإضافة قبل السلّة** (قرارُ المالك ٢٠٢٦-١٠-٢٦) — كالتطبيق: متوقّفٌ أو عنوانٌ لا نصله
  // أو بلا عنوان يُقال هنا، **ولا يُضاف شيءٌ يُردّ عند الإرسال.**
  const gate = useAddGate();
  const { address } = useCustomer();
  const { user } = useAuth();
  const [pickAddr, setPickAddr] = useState(false);

  useEffect(() => {
    setDetail(null);
    setLoadError("");
    setPicked({});
    setQty(1);
    setWarn("");
    if (!item) return;
    let alive = true;
    api<ItemDetail>(`/api/v1/public/items/${item.id}`)
      .then((d) => {
        if (!alive) return;
        setDetail(d);
        /* **والمجموعةُ الإلزاميّةُ بخيارٍ واحدٍ تُختار وحدَها** — «الحجم:
           عادي» بلا ضغطةٍ إضافيّة. */
        const pre: Record<string, string[]> = {};
        for (const g of d.modifiers ?? []) {
          const first = g.options.find((o) => o.available);
          if (g.min_select > 0 && g.max_select === 1 && first) pre[g.id] = [first.id];
        }
        setPicked(pre);
      })
      .catch((err) => alive && setLoadError(errorText(err)));
    return () => {
      alive = false;
    };
  }, [item]);

  const view = detail?.item ?? item;
  const groups = useMemo(() => detail?.modifiers ?? [], [detail]);
  const options: ModifierOption[] = useMemo(() => {
    const out: ModifierOption[] = [];
    for (const g of groups) for (const o of g.options) if (picked[g.id]?.includes(o.id)) out.push(o);
    return out;
  }, [groups, picked]);

  if (!item || !view) return null;
  const closed = view.source_closed || !view.available;
  const unit = unitPrice({ item: view, options });
  const usdUnit =
    view.price_usd != null ? view.price_usd + options.reduce((s, o) => s + (o.price_delta_usd ?? 0), 0) : null;
  const img = mediaUrl(view.image_url ?? view.image_thumb_url);

  const doAdd = () => {
    if (gate.kind !== "ok") return;
    cart.add(view, qty, options);
    toast(I.added);
    onClose();
  };

  const tryAdd = () => {
    const miss = missingGroup(groups, picked);
    if (miss) {
      setWarn(fill(I.needOption, { group: miss.name }));
      return;
    }
    if (item.has_options && !detail) return;
    if (cart.merchantId && view.merchant_id && cart.merchantId !== view.merchant_id) {
      setAskOther(true);
      return;
    }
    doAdd();
  };

  const toggle = (g: ModifierGroup, o: ModifierOption) => {
    setWarn("");
    setPicked((cur) => {
      const have = cur[g.id] ?? [];
      if (g.max_select === 1) return { ...cur, [g.id]: [o.id] };
      if (have.includes(o.id)) return { ...cur, [g.id]: have.filter((x) => x !== o.id) };
      if (g.max_select > 0 && have.length >= g.max_select) return cur;
      return { ...cur, [g.id]: [...have, o.id] };
    });
  };

  return (
    <>
      <Sheet
        open
        onClose={onClose}
        title={view.name}
        footer={
          closed ? (
            <Alert tone="warning">{closedLabel(view)}</Alert>
          ) : gate.kind === "closed" ? (
            <Alert tone="warning" title={SV.closedTitle}>
              {gate.text}
            </Alert>
          ) : gate.kind === "blocked" ? (
            <AvailabilityNotice av={gate.av} address={address} />
          ) : gate.kind === "need_address" ? (
            <div className="space-y-2">
              <Alert tone="info">{SV.needAddress}</Alert>
              {user ? (
                <Button className="w-full py-3" onClick={() => setPickAddr(true)}>
                  {SV.addAddress}
                </Button>
              ) : (
                <Link
                  href="/login"
                  className="flex w-full items-center justify-center rounded-control bg-accent py-3 font-bold text-on-bright"
                >
                  {SV.loginFirst}
                </Link>
              )}
            </div>
          ) : (
            <div className="flex items-center gap-3">
              <div className="flex items-center gap-1 rounded-control bg-field p-1">
                <button
                  type="button"
                  aria-label={I.minus}
                  onClick={() => setQty((q) => Math.max(1, q - 1))}
                  className="taparea flex h-9 w-9 items-center justify-center rounded-control hover:bg-row-hover"
                >
                  <IconMinus size={16} />
                </button>
                <span className="min-w-8 text-center font-bold tabular-nums">{fmtNum(qty)}</span>
                <button
                  type="button"
                  aria-label={I.plus}
                  onClick={() => setQty((q) => q + 1)}
                  className="taparea flex h-9 w-9 items-center justify-center rounded-control hover:bg-row-hover"
                >
                  <IconAdd size={16} />
                </button>
              </div>
              <Button className="flex-1 py-3" onClick={tryAdd} disabled={item.has_options && !detail && !loadError}>
                <span className="flex items-center justify-center gap-2">
                  {I.add}
                  <span aria-hidden>·</span>
                  <Money value={unit * qty} />
                </span>
              </Button>
            </div>
          )
        }
      >
        <div className="space-y-4">
          {img && (
            <div className="overflow-hidden rounded-card bg-field">
              <img src={img} alt="" className="max-h-72 w-full object-cover" />
            </div>
          )}
          <div className="flex flex-wrap items-center justify-between gap-2">
            <Price syp={unit} usd={usdUnit} />
            <span className="flex items-center gap-2">
              <Badge variant={closed ? "danger" : "success"}>{closed ? I.unavailable : I.available}</Badge>
              <FavoriteButton itemID={view.id} on={fav.has(view.id)} onToggle={fav.toggle} onRequireLogin={fav.needLogin} />
            </span>
          </div>
          {view.merchant_name &&
            (storeLink && view.merchant_id ? (
              <Link
                href={`/stores/${view.merchant_id}`}
                onClick={onClose}
                className="flex w-fit items-center gap-1.5 text-sm text-accent-text hover:underline"
              >
                <IconStore size={14} />
                {view.merchant_name}
              </Link>
            ) : (
              <p className="flex items-center gap-1.5 text-sm text-ink-muted">
                <IconStore size={14} />
                {view.merchant_name}
              </p>
            ))}
          {view.description && <p className="text-sm leading-relaxed text-ink-muted">{view.description}</p>}
          {item.has_options && !detail && !loadError && <LoadingState variant="inline" />}
          {loadError && <Alert>{loadError}</Alert>}
          {groups.map((g) => (
            <fieldset key={g.id} className="surface-inset space-y-2 p-3">
              <legend className="flex w-full items-center justify-between gap-2 text-sm font-bold">
                <span>{g.name}</span>
                <span className="text-xs font-normal text-ink-muted">
                  {g.min_select > 0 ? I.required : g.max_select === 1 ? I.pickOne : fill(I.pickUpTo, { n: fmtNum(g.max_select) })}
                </span>
              </legend>
              {g.options.map((o) => {
                const on = picked[g.id]?.includes(o.id) ?? false;
                const label = (
                  <span className="flex w-full items-center justify-between gap-3">
                    <span className={o.available ? "" : "text-ink-dim line-through"}>{o.name}</span>
                    {o.price_delta !== 0 && (
                      <span className="text-xs text-ink-muted">
                        {"+ "}
                        <Money value={o.price_delta} small />
                      </span>
                    )}
                  </span>
                );
                return g.max_select === 1 ? (
                  <Radio
                    key={o.id}
                    id={`opt-${o.id}`}
                    name={`grp-${g.id}`}
                    checked={on}
                    disabled={!o.available}
                    onChange={() => toggle(g, o)}
                    label={label}
                  />
                ) : (
                  <Checkbox
                    key={o.id}
                    id={`opt-${o.id}`}
                    checked={on}
                    disabled={!o.available}
                    onChange={() => toggle(g, o)}
                    label={label}
                  />
                );
              })}
            </fieldset>
          ))}
          {warn && <Alert tone="warning">{warn}</Alert>}
        </div>
      </Sheet>
      <AddressSheet open={pickAddr} onClose={() => setPickAddr(false)} />
      <Confirm
        open={askOther}
        title={I.otherStoreTitle}
        body={I.otherStoreText}
        confirmLabel={I.otherStoreClear}
        onCancel={() => setAskOther(false)}
        onConfirm={() => {
          setAskOther(false);
          cart.clear();
          doAdd();
        }}
      />
    </>
  );
}
