"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **بطاقةُ الصنف ونافذتُه — مرآةُ `ItemCard` و`ItemOptionsSheet`**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **السعرُ بالدولار و«· بالليرة» على سطرٍ واحد** — الدولارُ ما كتبه صاحبُ
 * المتجر، **والليرةُ ما يدفعه الزبون** (يحسبها المحرّكُ من سعر الصرف).
 * و«·» لا «≈» (طلبُ المالك ٢٠٢٦-١١-٠٧): الليرةُ ليست تقريباً، هي ما يُدفع.
 *
 * **واسمُ المتجر تحت الصورة** — «شاورما» من مطعمين اسمان متطابقان،
 * **والمتجرُ هو ما يفرّق بينهما.**
 *
 * **والمغلقُ والمنتهي يُحجبان بحجابٍ يقول السبب** — المتجرُ مغلقٌ الآن، أو
 * الصنفُ غيرُ متوفّر. **ولا زرَّ إضافةٍ عليهما.**
 */

import { useEffect, useMemo, useState, type ReactNode } from "react";
import Link from "next/link";
import { getMessages, defaultLocale, errorText, fmtNum } from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  Checkbox,
  Confirm,
  LoadingState,
  DualPrice,
  Money,
  Radio,
  Sheet,
  FavoriteButton,
  IconAdd,
  IconMinus,
  IconStore,
  ImageViewer,
  IconZoomIn,
} from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import {
  useCart,
  unitPrice,
  MAX_DRINK,
  MAX_PER_OPTION,
  splitGroup,
  splitPicks,
  picksTotal,
  picksTotalUsd,
  type DrinkPick,
  type Pick,
} from "@/lib/customer/cart";
import { useFav } from "@/lib/customer/favorites";
import { isDrinksGroup, type Item, type ItemDetail, type ModifierGroup, type ModifierOption } from "@/lib/customer/types";
import { dollars, fill, backAt } from "@/lib/customer/texts";
import { useShopToast } from "./toast";
import { useAddGate, AvailabilityNotice } from "./ServiceNotices";
import { AddressSheet } from "./AddressSheet";
import { DrinkThumb } from "./DrinkThumb";
import { ShopPic } from "./ShopPic";
import { useCustomer } from "@/lib/customer/context";
import { useAuth } from "@/lib/auth";

const m = getMessages(defaultLocale);
const I = m.shop.item;
const S = m.shop.stores;
const SV = m.shop.service;
const SK = m.shop.skin;
/** **سقفُ العدد في النافذة** — مرآةُ `ItemOptionsSheet.kt` (`qty < 99`). */
const MAX_QTY = 99;

/**
 * **السعرُ على سطرٍ واحد** — دولارٌ ثمّ «· ليرة»، أو ليرةٌ وحدَها. مرآةُ `dualPrice` في
 * التطبيق (`Money.kt`). و`muted` لسطور الخيارات والمشروبات: بلا لونٍ ولا ثِقَل.
 */
export function Price({
  syp,
  usd,
  muted = false,
  className = "",
}: {
  syp: number;
  usd?: number | null;
  muted?: boolean;
  className?: string;
}) {
  return <DualPrice usd={usd} syp={syp} muted={muted} className={className} />;
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
  layout = "tile",
}: {
  item: Item;
  onOpen: (item: Item) => void;
  storeLink?: boolean;
  /** **`row`** صورةٌ وبجانبها النصُّ (صفحةُ المتجر) · **`tile`** بلاطةٌ رأسيّة (السوقُ والبحثُ والمفضّلة). */
  layout?: "row" | "tile";
}) {
  const closed = item.source_closed || !item.available;
  const img = item.image_thumb_url ?? item.image_url;
  const full = mediaUrl(item.image_url ?? item.image_thumb_url);
  const [viewing, setViewing] = useState(false);
  const fav = useFav();
  const cart = useCart();
  // **كم منه في السلّة** — شارةٌ على الصورة كما في المخطّط، وللعرض وحدَه.
  const inCart = cart.lines.reduce((n, l) => (l.item.id === item.id ? n + l.qty : n), 0);
  return (
    <div className="shop-item" data-layout={layout}>
      {/* **والصورةُ وعدسةُ التكبير أخوان لا متداخلان** — زرٌّ في زرٍّ لا يصحّ. */}
      <div className="shop-item-img">
        <ShopPic src={img} name={item.name} icon="food" iconSize={28} />
        {/* **الضغطُ على الصورة يكبّرها** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — و«اطلب» والاسمُ يفتحان الصنف. */}
        <button
          type="button"
          onClick={() => (full ? setViewing(true) : onOpen(item))}
          aria-label={full ? `${item.name} — ${I.zoomBtn}` : item.name}
          className="transition-transform active:scale-[0.98]"
        />
        {/* **غشاءٌ فاتحٌ وشارةٌ في الزاوية** (تدقيق D6، ٢٠٢٦-١١-١١) — كان حجاباً داكناً على الصورة كلِّها.
            **الصورةُ تبقى مفهومةً في الثيمين، والشارةُ تقول السبب**؛ ونصُّ «يفتح الساعة…» لنافذة الصنف. */}
        {closed && (
          <>
            <div aria-hidden className="shop-item-veil" />
            <span title={closedLabel(item)} className="shop-item-badge">
              {item.source_closed ? I.closedBadge : I.unavailable}
            </span>
          </>
        )}
        {item.discount_percent != null && item.discount_percent > 0 && (
          <span className="pointer-events-none absolute start-2 top-2">
            <Badge variant="accent">{`-${fmtNum(item.discount_percent)}%`}</Badge>
          </span>
        )}
        {inCart > 0 && (
          <span className="shop-qty-badge" aria-label={fill(SK.inCart, { n: fmtNum(inCart) })}>
            {fmtNum(inCart)}
          </span>
        )}
      </div>
      {full && viewing && <ImageViewer src={full} alt={item.name} onClose={() => setViewing(false)} />}
      <div className="shop-item-body">
        <button type="button" onClick={() => onOpen(item)} className="min-w-0 text-start">
          <h3 className="line-clamp-2">{item.name}</h3>
          {item.description && <p className="d">{item.description}</p>}
        </button>
        {/* **وداخل صفحة المتجر نفسِها لا اسمَ تحت كلّ صنف** (٢٠٢٦-١١-٠٤). */}
        {storeLink &&
          (item.merchant_id && item.merchant_name ? (
            <Link
              href={`/stores/${item.merchant_id}`}
              aria-label={`${S.openStore} — ${item.merchant_name}`}
              className="truncate text-xs text-ink-muted hover:text-accent-text hover:underline"
            >
              {item.merchant_name}
            </Link>
          ) : (
            <span className="truncate text-xs text-ink-muted">{item.merchant_name ?? ""}</span>
          ))}
        <div className="shop-item-foot">
          {/* **الدولارُ ثمّ الليرةُ** («$2.5 · 300 ل.س») — والليرةُ تنزل تحته إن ضاق ولا تُقصّ. */}
          <span className="flex min-w-0 flex-wrap items-baseline gap-x-1">
            <DualPrice usd={item.price_usd} syp={item.price} stacked />
            {item.price_before != null && (
              <span className="text-xs text-ink-muted line-through">
                {item.price_before_usd != null ? dollars(item.price_before_usd) : <Money value={item.price_before} small />}
              </span>
            )}
          </span>
          <span className="flex items-center gap-1.5">
            <FavoriteButton
              itemID={item.id}
              on={fav.has(item.id)}
              onToggle={fav.toggle}
              onRequireLogin={fav.needLogin}
              size="sm"
              className="shrink-0 rounded-full! border border-line-soft"
            />
            {!closed && (
              <button type="button" onClick={() => onOpen(item)} className="shop-add">
                <IconAdd size={14} />
                {I.orderBtn}
              </button>
            )}
          </span>
        </div>
      </div>
    </div>
  );
}

export function ItemGrid({
  items,
  onOpen,
  storeLink = true,
  layout = "tile",
}: {
  items: Item[];
  onOpen: (item: Item) => void;
  storeLink?: boolean;
  layout?: "row" | "tile";
}) {
  return (
    <div className="shop-items" data-layout={layout}>
      {items.map((it) => (
        <ItemCard key={it.id} item={it} onOpen={onOpen} storeLink={storeLink} layout={layout} />
      ))}
    </div>
  );
}

/** **أيُّ مجموعةٍ إلزاميّةٍ لم يُختر منها؟** — والمحرّكُ يردّ السلّةَ كلَّها إن نقصت. */
function missingGroup(
  groups: ModifierGroup[],
  picked: Record<string, string[]>,
  drinkQty: Record<string, number>,
  split: ModifierGroup | null,
  splitQty: Record<string, number>,
): ModifierGroup | null {
  for (const g of groups) {
    const n = isDrinksGroup(g)
      ? g.options.filter((o) => (drinkQty[o.id] ?? 0) > 0).length
      : g.id === split?.id
        ? Object.values(splitQty).some((x) => x > 0)
          ? 1
          : 0
        : (picked[g.id]?.length ?? 0);
    if (g.min_select > 0 && n < g.min_select) return g;
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
  // **وعددُ كلِّ مشروب** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — يبدأ صفراً ولا يتبع عددَ الصنف.
  const [drinkQty, setDrinkQty] = useState<Record<string, number>>({});
  // **وعددُ كلِّ خيارٍ في مجموعة العدّادات** (`splitGroup`) — يبدأ صفراً.
  const [splitQty, setSplitQty] = useState<Record<string, number>>({});
  const [warn, setWarn] = useState("");
  const [askOther, setAskOther] = useState(false);
  const fav = useFav();
  // **بوّابةُ الإضافة قبل السلّة** (قرارُ المالك ٢٠٢٦-١٠-٢٦) — كالتطبيق: متوقّفٌ أو عنوانٌ لا نصله
  // أو بلا عنوان يُقال هنا، **ولا يُضاف شيءٌ يُردّ عند الإرسال.**
  const gate = useAddGate();
  const { address } = useCustomer();
  const { user } = useAuth();
  const [pickAddr, setPickAddr] = useState(false);
  const [viewing, setViewing] = useState(false);

  useEffect(() => {
    setDetail(null);
    setLoadError("");
    setPicked({});
    setQty(1);
    setDrinkQty({});
    setSplitQty({});
    setWarn("");
    setViewing(false);
    if (!item) return;
    let alive = true;
    api<ItemDetail>(`/api/v1/public/items/${item.id}`)
      .then((d) => {
        if (!alive) return;
        setDetail(d);
        /* **والمجموعةُ الإلزاميّةُ بخيارٍ واحدٍ تُختار وحدَها** — «الحجم:
           عادي» بلا ضغطةٍ إضافيّة. */
        const pre: Record<string, string[]> = {};
        const sg = splitGroup(d.modifiers ?? []);
        for (const g of d.modifiers ?? []) {
          const first = g.options.find((o) => o.available);
          if (g.id !== sg?.id && !isDrinksGroup(g) && g.min_select > 0 && g.max_select === 1 && first) pre[g.id] = [first.id];
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
  // **مجموعةُ العدّادات** — وفي وجودها يختفي العددُ العامّ، ويصير كلُّ خيارٍ سطراً.
  const split = useMemo(() => splitGroup(groups), [groups]);
  const options: ModifierOption[] = useMemo(() => {
    const out: ModifierOption[] = [];
    for (const g of groups) {
      if (isDrinksGroup(g) || g.id === split?.id) continue;
      for (const o of g.options) if (picked[g.id]?.includes(o.id)) out.push(o);
    }
    return out;
  }, [groups, picked, split]);
  const drinks: DrinkPick[] = useMemo(() => {
    const out: DrinkPick[] = [];
    for (const g of groups) {
      if (!isDrinksGroup(g)) continue;
      for (const o of g.options) {
        const n = drinkQty[o.id] ?? 0;
        if (n > 0) out.push({ option: o, qty: n });
      }
    }
    return out;
  }, [groups, drinkQty]);

  if (!item || !view) return null;
  const closed = view.source_closed || !view.available;
  const unit = unitPrice({ item: view, options });
  const usdUnit =
    view.price_usd != null ? view.price_usd + options.reduce((s, o) => s + (o.price_delta_usd ?? 0), 0) : null;
  const img = mediaUrl(view.image_url ?? view.image_thumb_url);
  // **السطور**: سطرٌ لكلّ خيارٍ بعدّاده، أو سطرٌ واحدٌ بالعدد العامّ — والإجماليُّ جمعُها.
  const picks: Pick[] = split ? splitPicks(split, splitQty, options, drinks) : [{ options, qty, drinks }];
  const total = picksTotal(view, picks);
  const totalUsd = view.price_usd != null ? picksTotalUsd(view.price_usd, picks) : null;
  const nothing = !picks.some((p) => p.qty > 0);

  const doAdd = () => {
    if (gate.kind !== "ok") return;
    for (const p of picks) if (p.qty > 0) cart.add(view, p.qty, p.options, p.drinks);
    toast(I.added);
    onClose();
  };

  const tryAdd = () => {
    const miss = missingGroup(groups, picked, drinkQty, split, splitQty);
    if (miss) {
      /* **ومجموعةٌ إلزاميّةٌ كلُّ خياراتها نافدةٌ لا يُقال لها «اختر»** — لا شيءَ يُختار. */
      const out = !miss.options.some((o) => o.available);
      setWarn(fill(out ? I.allOptionsOut : I.needOption, { group: miss.name }));
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
              {/* **والعددُ العامُّ لا يُرسم مع العدّادات** — كلُّ خيارٍ بعدّاده. */}
              {!split && (
              <div className="flex items-center gap-1 rounded-control bg-field p-1">
                <button
                  type="button"
                  aria-label={I.minus}
                  onClick={() => setQty((q) => Math.max(1, q - 1))}
                  disabled={qty <= 1}
                  className="taparea flex h-9 w-9 items-center justify-center rounded-control hover:bg-row-hover disabled:opacity-40"
                >
                  <IconMinus size={16} />
                </button>
                <span className="min-w-8 text-center font-bold tabular-nums">{fmtNum(qty)}</span>
                <button
                  type="button"
                  aria-label={I.plus}
                  onClick={() => setQty((q) => Math.min(MAX_QTY, q + 1))}
                  disabled={qty >= MAX_QTY}
                  className="taparea flex h-9 w-9 items-center justify-center rounded-control hover:bg-row-hover disabled:opacity-40"
                >
                  <IconAdd size={16} />
                </button>
              </div>
              )}
              <Button
                className="flex-1 py-3"
                onClick={tryAdd}
                disabled={(item.has_options && !detail && !loadError) || (!!split && nothing)}
              >
                {/* **عبارةٌ فوق وسعرٌ تحت** — على عرض الهاتف كان السطرُ الواحدُ يلتفّ فيقطع «أضف إلى السلة» والسعرَ. */}
                <span className="flex flex-col items-center justify-center gap-0.5 leading-tight">
                  <span className="whitespace-nowrap">{I.add}</span>
                  {/* **والدولارُ يتبع العدد كالليرة** — كان سعرَ الواحد مهما زاد العدد. */}
                  {totalUsd != null ? (
                    <DualPrice usd={totalUsd} syp={total} inherit />
                  ) : (
                    <Money value={total} className="[&_span]:text-current!" />
                  )}
                </span>
              </Button>
            </div>
          )
        }
      >
        <div className="space-y-4">
          {/* **الصورةُ كاملةً بأبعادها، والنقرُ يفتحها بملء الشاشة** (قرارُ المالك ٢٠٢٦-١٠-٢٩). */}
          {img && (
            <button
              type="button"
              onClick={() => setViewing(true)}
              aria-label={`${view.name} — ${m.common.media.zoomHint}`}
              className="relative flex w-full items-center justify-center overflow-hidden rounded-card bg-field"
            >
              {/* **صورةٌ لا تأكل النافذة** (تدقيق D5): سقفُها نحو ١٦:١٠ — كانت ٣٢٠ بكسلاً. */}
              <img src={img} alt="" className="max-h-44 w-auto max-w-full object-contain" />
              {/* **عدسةٌ في دائرةٍ بيضاء تقول «تُكبَّر»** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — مرآةُ `ZoomBadge` في التطبيق. */}
              <span
                aria-hidden
                className="absolute bottom-2 end-2 flex h-9 w-9 items-center justify-center rounded-full border border-line bg-paper text-primary-solid shadow-e2"
              >
                <IconZoomIn size={20} />
              </span>
            </button>
          )}
          {img && viewing && <ImageViewer src={img} alt={view.name} onClose={() => setViewing(false)} />}
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
          {item.has_options && !detail && !loadError && <LoadingState variant="inline" />}
          {loadError && <Alert>{loadError}</Alert>}
          {groups.map((g) =>
            isDrinksGroup(g) ? (
              <CounterBlock
                key={g.id}
                group={g}
                counts={drinkQty}
                max={MAX_DRINK}
                thumbs
                price={(o) => <Price muted syp={o.price_delta} usd={o.price_delta_usd} />}
                onCount={(id, n) => {
                  setWarn("");
                  setDrinkQty((cur) => ({ ...cur, [id]: Math.max(0, Math.min(MAX_DRINK, n)) }));
                }}
              />
            ) : g.id === split?.id ? (
              /* **عدّادٌ جنب كلّ خيارٍ بسعره كاملاً** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — «عادي ٢٠ ل.س (− ٢ +)». */
              <CounterBlock
                key={g.id}
                group={g}
                counts={splitQty}
                max={MAX_PER_OPTION}
                price={(o) => (
                  <Price
                    muted
                    syp={view.price + o.price_delta}
                    usd={view.price_usd != null ? view.price_usd + (o.price_delta_usd ?? 0) : null}
                  />
                )}
                onCount={(id, n) => {
                  setWarn("");
                  setSplitQty((cur) => ({ ...cur, [id]: Math.max(0, Math.min(MAX_PER_OPTION, n)) }));
                }}
              />
            ) : (
            <fieldset key={g.id} className="surface-inset space-y-2 p-3">
              <legend className="float-start mb-2 flex w-full items-center justify-between gap-2 text-sm font-bold">
                <span>{g.name}</span>
                {/* **«إلزامي» شارةٌ مقروءة** (تدقيق D5) — كانت نصّاً رمادياً صغيراً. */}
                {g.min_select > 0 ? (
                  <Badge variant="accent">{I.required}</Badge>
                ) : (
                  <span className="text-xs font-normal text-ink-muted">
                    {g.max_select === 1 ? I.pickOne : fill(I.pickUpTo, { n: fmtNum(g.max_select) })}
                  </span>
                )}
              </legend>
              {g.min_select > 0 && !g.options.some((o) => o.available) && (
                <p className="text-xs text-danger">{fill(I.allOptionsOut, { group: g.name })}</p>
              )}
              {g.options.map((o) => {
                const on = picked[g.id]?.includes(o.id) ?? false;
                const label = (
                  <span className="flex w-full items-center justify-between gap-3">
                    <span className="flex items-center gap-2">
                      <span className={o.available ? "" : "text-ink-dim line-through"}>{o.name}</span>
                      {!o.available && <span className="text-xs text-danger">{I.unavailable}</span>}
                    </span>
                    {g.price_mode === "weight" ? (
                      // **«حسب الوزن» بسعره كاملاً** لا «+فرق» (قرارُ المالك ٢٠٢٦-١٠-٢٨) —
                      // مرآةُ `ItemOptionsSheet.kt`.
                      <span className="text-xs text-ink-muted">
                        <Price
                          muted
                          syp={view.price + o.price_delta}
                          usd={view.price_usd != null && o.price_delta_usd != null ? view.price_usd + o.price_delta_usd : null}
                        />
                      </span>
                    ) : (
                      o.price_delta !== 0 && (
                        <span className="text-xs text-ink-muted">
                          {"+ "}
                          <Price muted syp={o.price_delta} usd={o.price_delta_usd} />
                        </span>
                      )
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
            ),
          )}
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

/**
 * **مجموعةٌ بعدّادٍ لكلّ خيار** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — مرآةُ `CounterBlock` في التطبيق.
 *
 * المشروبات: «− ٠ +» لكلّ مشروب، يبدأ صفراً وسقفُه ٢٠، **ولا يُضرب بعدد الصنف.**
 * ومجموعةُ العدّادات (`splitGroup`): كلُّ خيارٍ بسعره كاملاً وعدّاده، وكلُّ خيارٍ سطرٌ في السلّة.
 */
function CounterBlock({
  group,
  counts,
  max,
  price,
  onCount,
  thumbs = false,
}: {
  group: ModifierGroup;
  counts: Record<string, number>;
  max: number;
  /** **صورةُ كلّ مشروبٍ جنبه** (٢٠٢٦-١٠-٣٠) — أو أيقونةُ الكأس. */
  thumbs?: boolean;
  price: (o: ModifierOption) => ReactNode;
  onCount: (id: string, n: number) => void;
}) {
  /* **كلُّ المشروبات ظاهرةٌ دائماً، والنافذةُ تتمرّر** (قرارُ المالك ٢٠٢٦-١١-٠٧: «سكرول مو إخفاء
     وإظهار») — و«أضف» والعددُ في تذييل `Sheet` ثابتان تحتها. */
  return (
    <fieldset className="surface-inset space-y-2 p-3">
      {/* **العنوانُ داخل الصندوق لا على حدّه** — عنصرُ legend يقف فوق الإطار فيلامس خطَّه؛ فيبقى للقارئ الصوتيّ ويُرسَم العنوانُ صفّاً عادياً. */}
      <legend className="sr-only">{group.name}</legend>
      <div className="flex w-full items-center justify-between gap-2 pb-1 text-sm font-bold" aria-hidden>
        <span>{group.name}</span>
        {group.min_select > 0 ? (
          <Badge variant="accent">{I.required}</Badge>
        ) : (
          <span className="text-xs font-normal text-ink-muted">{I.optional}</span>
        )}
      </div>
      {group.options.map((o) => {
        const n = counts[o.id] ?? 0;
        return (
          /* **العدّادُ بجانب اسمه لا في طرف السطر** (تدقيق D5). */
          <div key={o.id} className="flex items-center gap-3">
            <span className="flex min-w-0 items-center gap-2">
            {thumbs && <DrinkThumb url={o.image_thumb_url} />}
            <span className="flex min-w-0 flex-col">
              <span className={`${o.available ? "" : "text-ink-dim line-through"} ${n > 0 ? "font-bold" : ""}`}>{o.name}</span>
              {o.available ? (
                <span className="text-xs text-ink-muted">{price(o)}</span>
              ) : (
                <span className="text-xs text-danger">{I.unavailable}</span>
              )}
            </span>
            </span>
            <span className="flex shrink-0 items-center gap-0.5 rounded-control bg-field p-0.5">
              <button
                type="button"
                aria-label={I.minus}
                onClick={() => onCount(o.id, n - 1)}
                disabled={n <= 0}
                className="taparea flex h-8 w-8 items-center justify-center rounded-control hover:bg-row-hover disabled:opacity-40"
              >
                <IconMinus size={14} />
              </button>
              <span className={`min-w-6 text-center text-sm font-bold tabular-nums ${n > 0 ? "text-accent-text" : "text-ink-muted"}`}>
                {fmtNum(n)}
              </span>
              <button
                type="button"
                aria-label={I.plus}
                onClick={() => onCount(o.id, n + 1)}
                disabled={!o.available || n >= max}
                className="taparea flex h-8 w-8 items-center justify-center rounded-control hover:bg-row-hover disabled:opacity-40"
              >
                <IconAdd size={14} />
              </button>
            </span>
          </div>
        );
      })}
    </fieldset>
  );
}
