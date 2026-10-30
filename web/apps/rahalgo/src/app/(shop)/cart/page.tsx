"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **سلّتي — مراجعةٌ ثمّ إرسال** (مرآةُ `CartScreen.kt`)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **والأرقامُ كلُّها من المحرّك** (`/public/quote`): المجموعُ وأجرةُ
 * التوصيل والإجماليّ — **ولا حسبةَ في المتصفّح**: رسمُ المنطقة يتبدّل من
 * اللوحة، وحسبةٌ محلّيّةٌ تفترق عمّا يُقيَّد.
 *
 * **ويُرسَل معها ما كان معروضاً** (`expected`) — **فإن تبدّل سعرٌ أو أجرةٌ
 * قيل للزبون قبل أن يُرسل** («تغيّرت بعض القيم منذ أن راجعتها»)، **ولا
 * يُرسَل حتّى يقرّ.**
 *
 * **والطلبُ يُنشأ مرّةً واحدةً مهما أُعيدت المحاولة** — مفتاحُ المحاولة
 * (`Idempotency-Key`) يبقى ما دام لم ينجح.
 */

import Link from "next/link";
import { useCallback, useEffect, useMemo, useState, type ReactNode } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale, errorText, fmtMoney, fmtNum } from "@rahalgo/i18n";
import {
  Alert,
  Button,
  ButtonLink,
  Confirm,
  EmptyState,
  Input,
  LoadingState,
  Money,
  Radio,
  Textarea,
  IconCart,
  IconDelete,
  IconAdd,
  IconMinus,
  IconStore,
} from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useCart, drinkRows, lineKey, linePayload, unitPrice, MAX_DRINK } from "@/lib/customer/cart";
import { useCustomer } from "@/lib/customer/context";
import { attemptKey, clearAttempt, isDecided } from "@/lib/customer/storage";
import { changeText, fill, serviceText } from "@/lib/customer/texts";
import type { CartChange, PromoPreview, Quote } from "@/lib/customer/types";
import { AddressChip, AddressSheet } from "@/components/customer/AddressSheet";
import { DrinkThumb } from "@/components/customer/DrinkThumb";
import { AvailabilityNotice, useOrderingState } from "@/components/customer/ServiceNotices";
import { useShopToast } from "@/components/customer/toast";

const m = getMessages(defaultLocale);
const C = m.shop.cart;
const ORDER_SLOT = "order";

/** **أمتّصلٌ بالشبكة؟** — مرآةُ `Net.online`: لا يُرسَل طلبٌ بلا شبكة، ويُقال السببُ تحت. */
function useOnline(): boolean {
  const [online, setOnline] = useState(true);
  useEffect(() => {
    const sync = () => setOnline(navigator.onLine);
    sync();
    window.addEventListener("online", sync);
    window.addEventListener("offline", sync);
    return () => {
      window.removeEventListener("online", sync);
      window.removeEventListener("offline", sync);
    };
  }, []);
  return online;
}

/** **بصمةُ ما عُرض** — ومنها يُعرف أنّ الزبونَ أقرّ القيمَ الحاليّة لا غيرَها. */
function fingerprint(q: Quote | null, lines: string, point: string, discount: number): string {
  return [point, q?.subtotal, q?.delivery_fee, q?.total, discount, q?.availability?.available ?? true, lines].join("|");
}

export default function CartPage() {
  const { user, loading } = useAuth();
  const cart = useCart();
  const { address, available, platform, reloadWallet } = useCustomer();
  const ordering = useOrderingState();
  const router = useRouter();
  const toast = useShopToast();

  const [pickAddr, setPickAddr] = useState(false);
  const [quote, setQuote] = useState<Quote | null>(null);
  const [quoteError, setQuoteError] = useState("");
  const [quoting, setQuoting] = useState(false);
  const [changes, setChanges] = useState<CartChange[]>([]);
  const [acked, setAcked] = useState("");
  const [promo, setPromo] = useState("");
  const [promoRes, setPromoRes] = useState<PromoPreview | null>(null);
  const [promoBusy, setPromoBusy] = useState(false);
  const [wallet, setWallet] = useState(false);
  const [notes, setNotes] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [errorCode, setErrorCode] = useState("");
  const [uncertain, setUncertain] = useState(false);
  const [askClear, setAskClear] = useState(false);
  const online = useOnline();

  const point = address ? `${address.lat},${address.lng}` : "";
  const linesKey = cart.lines.map((l) => `${lineKey(l)}:${l.qty}`).join(",");
  const drinks = useMemo(() => drinkRows(cart.lines), [cart.lines]);

  /* ── التسعيرة: تُطلب عند كلّ تبدّلٍ في السلّة أو العنوان ── */
  const requote = useCallback(
    (prev: Quote | null) => {
      if (!address || cart.lines.length === 0) {
        setQuote(null);
        return;
      }
      const expected: { lines: Record<string, number>; delivery_fee?: number } = { lines: {} };
      for (const l of cart.lines) expected.lines[l.item.id] = unitPrice(l);
      if (prev) expected.delivery_fee = prev.delivery_fee;
      setQuoting(true);
      api<Quote>("/api/v1/public/quote", {
        method: "POST",
        body: JSON.stringify({
          items: cart.lines.map(linePayload),
          lat: address.lat,
          lng: address.lng,
          expected,
        }),
      })
        .then((q) => {
          setQuote(q);
          setChanges(q.changes ?? []);
          setQuoteError("");
        })
        .catch((err) => setQuoteError(errorText(err)))
        .finally(() => setQuoting(false));
    },
    // eslint-disable-next-line react-hooks/exhaustive-deps
    [point, linesKey],
  );

  useEffect(() => {
    requote(quote);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [requote]);

  const cut = promoRes?.valid ? promoRes : null;
  const fee = quote ? (cut ? cut.delivery_fee : quote.delivery_fee) : 0;
  // **وسطرُ المتجر الثاني يبقى ولو انطبق كود** (مرآةُ `CartScreen.kt` ٢٠٢٦-١٠-٢٧).
  const extra = quote ? Math.min(Math.max(quote.sources_fee, 0), fee) : 0;
  // **وكودُ التوصيل المجاني وحدَه يُغني عن العرض** — كودُ حسمٍ عاديٌّ لا يمحو «مجاني».
  const freeCode = cut?.kind === "free_delivery";
  const waived = freeCode ? 0 : (quote?.free_waived ?? 0);
  // **وسلّةٌ موقوفةٌ (`blocked`) لا مجموعَ لها** — المحرّكُ يصفّرها عمداً، و«0 ل.س» يُقرأ سعراً (مرآةُ `CartScreen.kt`).
  const priced = quote && !quote.blocked ? quote : null;
  const payable = priced ? (cut ? Math.max(0, priced.subtotal + fee - cut.discount) : priced.total) : null;
  // **والمحفظةُ لا تُعرض حتّى يغطّي رصيدُها الإجماليَّ** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — مرآةُ `CartScreen.kt`.
  const walletBlocked = payable == null || available < payable;

  useEffect(() => {
    if (walletBlocked && wallet) setWallet(false);
  }, [walletBlocked, wallet]);

  const fp = useMemo(() => fingerprint(quote, linesKey, point, cut?.discount ?? 0), [quote, linesKey, point, cut]);
  const canSubmit = changes.length === 0 || acked === fp;

  const applyPromo = async () => {
    const code = promo.trim();
    if (!code || !quote) return;
    setPromoBusy(true);
    try {
      const r = await api<PromoPreview>("/api/v1/promo/preview", {
        method: "POST",
        body: JSON.stringify({
          code,
          subtotal: quote.subtotal,
          delivery_fee: quote.delivery_fee,
          expected_discount: promoRes?.discount ?? null,
        }),
      });
      setPromoRes(r);
      if (r.changes?.length) setChanges((c) => [...c.filter((x) => x.type !== "promo_or_discount_changed"), ...(r.changes ?? [])]);
    } catch (err) {
      setPromoRes(null);
      setError(errorText(err));
    } finally {
      setPromoBusy(false);
    }
  };

  const send = async () => {
    if (!address || busy) return;
    setBusy(true);
    setError("");
    setErrorCode("");
    try {
      const created = await api<{ id: string; number: number }>("/api/v1/orders", {
        method: "POST",
        headers: { "Idempotency-Key": attemptKey(ORDER_SLOT) },
        body: JSON.stringify({
          items: cart.lines.map(linePayload),
          address_text: address.address_text,
          lat: address.lat,
          lng: address.lng,
          payment_method: wallet ? "wallet" : "cash",
          promo_code: cut ? promo.trim() : "",
          notes: notes.trim(),
        }),
      });
      clearAttempt(ORDER_SLOT);
      setUncertain(false);
      cart.clear();
      reloadWallet();
      toast(C.sent);
      router.push(created?.id ? `/orders/${created.id}` : "/orders");
    } catch (err) {
      if (isDecided(err)) {
        clearAttempt(ORDER_SLOT);
        setUncertain(false);
      } else {
        setUncertain(true);
      }
      setError(errorText(err));
      setErrorCode((err as { body?: { code?: string } })?.body?.code ?? "");
      requote(quote);
    } finally {
      setBusy(false);
    }
  };

  if (!cart.ready || loading) return <LoadingState variant="list" rows={3} />;

  if (cart.lines.length === 0) {
    return (
      <div className="flex flex-col gap-3">
        <h1 className="heading-page">{C.title}</h1>
        <EmptyState
          icon={IconCart}
          title={C.empty}
          action={
            <ButtonLink href="/shop" variant="secondary">
              {C.browse}
            </ButtonLink>
          }
        />
      </div>
    );
  }

  const av = quote?.availability;
  const blocked =
    !quote ||
    quote.out_of_zone ||
    quote.zone_closed ||
    quote.blocked === true ||
    quote.too_many_sources ||
    av?.available === false;
  // **ورقمٌ واحدٌ للحساب وعليه واتساب** (قرارُ المالك) — فإن لم يُضبط واتسابٌ منفصلٌ فرقمُ الدعم.
  const wa = ((platform?.social.whatsapp ?? "").trim() || (platform?.support_phone ?? "")).replace(/\D/g, "");
  const closedNow =
    !ordering.open ||
    quote?.out_of_zone === true ||
    quote?.zone_closed === true ||
    quote?.blocked === true ||
    av?.available === false;
  // ══ **والصنفُ الذي يمنع الإرسالَ يُعلَّم في سطره** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — مرآةُ `badLines`
  //    في `CartScreen.kt`: كانت «انظر السبب في الأعلى» ولا سببَ ظاهر. والمحرّكُ يسمّي الصنفَ في `changes`. ══
  const badLines = new Map<string, CartChange>();
  for (const c of changes) {
    if (c.type === "product_unavailable" || c.type === "product_removed" || c.type === "quantity_invalid") {
      badLines.set(c.menu_item_id, c);
    }
  }
  const lineProblem = (c: CartChange): string =>
    c.type === "quantity_invalid" ? C.lineQtyInvalid : c.option ? fill(C.lineOptionUnavailable, { option: c.option }) : C.lineUnavailable;
  // **سببُ تعطيل الإرسال يُسمّى في الشريط نفسِه** — مرآةُ `why` في `CartScreen.kt`، لا «انظر في الأعلى».
  const closedWhy = (): string => {
    if (badLines.size > 0) return badLines.size === 1 ? C.whyUnavailable : fill(C.whyUnavailableMany, { n: fmtNum(badLines.size) });
    if (!ordering.open) return ordering.text;
    if (av && !av.available) return serviceText(av.reason, av.message, av.next_available_at);
    if (quote?.out_of_zone) return C.outOfZone;
    if (quote?.zone_closed) return serviceText("zone_closed_now", "", quote.next_available_at);
    return C.whyBlocked;
  };
  const why = busy
    ? ""
    : !address
      ? C.whyAddress
      : !online
        ? C.whyOffline
        : closedNow || badLines.size > 0
          ? closedWhy()
          : quote?.too_many_sources
            ? quote.max_sources <= 1
              ? C.tooManyOne
              : fill(C.tooMany, { n: fmtNum(quote.max_sources) })
            : !quote
              ? C.whyPricing
              : !canSubmit
                ? C.whyReview
                : "";

  return (
    <div className="flex flex-col gap-4">
      <div className="grid grid-cols-1 gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,24rem)] lg:items-start">
        <section className="flex flex-col gap-3">
          <h1 className="heading-page">{C.title}</h1>
          {/* ══ **الأصنافُ كلُّها أوّلاً، ثمّ المشروباتُ مجموعةً تحتها** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — المشروبُ
              نفسُه من أكثر من صنفٍ سطرٌ واحدٌ بعدده المجموع: «+» و«−» لأوّل صنفٍ هو معه، والحذفُ من كلّها.
              **والسعرُ في عمودٍ واحدٍ ثابتٍ آخرَ السطر** لكلّ الأسطر، والعدّادُ والحذفُ قبله. ══ */}
          <ul className="surface divide-y divide-line-soft">
            {cart.lines.map((l) => {
              const k = lineKey(l);
              const img = mediaUrl(l.item.image_thumb_url ?? l.item.image_url);
              const bad = badLines.get(l.item.id);
              return (
                <li key={k} className={`p-3 ${bad ? "bg-danger-tint" : ""}`}>
                  <CartRow
                    media={
                      <span className="h-14 w-14 shrink-0 overflow-hidden rounded-control bg-field">
                        {img && <img src={img} alt="" loading="lazy" className="h-full w-full object-cover" />}
                      </span>
                    }
                    price={unitPrice(l) * l.qty}
                    qty={l.qty}
                    onLess={() => cart.setQty(k, l.qty - 1)}
                    onMore={() => cart.setQty(k, l.qty + 1)}
                    onRemove={() => cart.setQty(k, 0)}
                  >
                    <p className="truncate text-sm font-bold">{l.item.name}</p>
                    {l.options.length > 0 && (
                      <p className="truncate text-xs text-ink-muted">{l.options.map((o) => o.name).join(m.common.listSep)}</p>
                    )}
                    {l.item.merchant_name && (
                      <p className="flex items-center gap-1 truncate text-xs text-ink-muted">
                        <IconStore size={11} />
                        {l.item.merchant_name}
                      </p>
                    )}
                    {bad && <p className="text-xs font-bold text-danger">{lineProblem(bad)}</p>}
                  </CartRow>
                </li>
              );
            })}
            {drinks.map((d) => {
              const room = d.refs.find((r) => r.qty < MAX_DRINK);
              const first = d.refs[0];
              return (
                <li key={`drink-${d.name}`} className="p-3" data-testid="cart-drink">
                  <CartRow
                    media={
                      <span className="flex h-14 w-14 shrink-0 items-center justify-center">
                        <DrinkThumb url={d.thumb} />
                      </span>
                    }
                    price={d.total}
                    qty={d.qty}
                    onLess={() => first && cart.setDrinkQty(first.key, first.optionId, first.qty - 1)}
                    onMore={() => room && cart.setDrinkQty(room.key, room.optionId, room.qty + 1)}
                    moreDisabled={!room}
                    onRemove={() => cart.removeDrink(d.name)}
                  >
                    <p className="truncate text-sm font-bold">{d.name}</p>
                  </CartRow>
                </li>
              );
            })}
          </ul>
          {/* **زرٌّ واضحٌ بأيقونة سلّة المهملات تحت الأصناف مباشرةً** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — لا رابطٌ نصّيّ. */}
          <button
            type="button"
            onClick={() => setAskClear(true)}
            className="flex w-fit items-center gap-2 rounded-control border border-danger px-4 py-2 text-sm font-bold text-danger transition-colors hover:bg-danger-tint"
          >
            <IconDelete size={16} />
            {C.clear}
          </button>
          {/* **وإفراغُ السلّة يُسأل قبله** — ضغطةٌ غلطٌ تطيّر سلّةً كاملة. */}
          <Confirm
            open={askClear}
            title={C.clearTitle}
            body={C.clearBody}
            confirmLabel={C.clearYes}
            onCancel={() => setAskClear(false)}
            onConfirm={() => {
              setAskClear(false);
              cart.clear();
            }}
          />
        </section>

        <aside className="flex flex-col gap-3">
          {!user ? (
            <div className="surface flex flex-col gap-3 p-4">
              <p className="text-sm">{C.needLogin}</p>
              <ButtonLink href="/login?next=/cart" className="w-full py-3">
                {C.login}
              </ButtonLink>
            </div>
          ) : (
            <>
              <AddressChip onOpen={() => setPickAddr(true)} />

              <div className="surface flex flex-col gap-3 p-4">
                <div className="flex items-end gap-2">
                  <Input
                    id="promo"
                    label={C.promo}
                    dir="ltr"
                    value={promo}
                    onChange={(e) => {
                      setPromo(e.target.value);
                      setPromoRes(null);
                    }}
                    wrapperClassName="flex-1"
                  />
                  <Button variant="secondary" onClick={applyPromo} disabled={promoBusy || !promo.trim() || !quote}>
                    {C.promoApply}
                  </Button>
                </div>
                {promoRes && (
                  <p className={`text-xs ${promoRes.valid ? "text-success" : "text-danger"}`}>
                    {promoRes.valid
                      ? promoRes.kind === "free_delivery" && promoRes.discount === 0
                        ? C.promoFree
                        : fill(C.promoOk, { amount: fmtMoney(promoRes.discount) })
                      : promoRes.reason
                        ? serviceText(promoRes.reason)
                        : C.promoBad}
                  </p>
                )}

                {/* **سطرٌ واحدٌ حين لا تكفي المحفظة — لا خيارٌ مطفأٌ ولا «رصيدك لا يكفي»** (قرارُ المالك ٢٠٢٦-١٠-٣٠). */}
                {walletBlocked ? (
                  <p className="text-sm">{m.shop.orders.payCashOnly}</p>
                ) : (
                  <fieldset className="flex flex-col gap-2">
                    <legend className="mb-1 text-sm font-medium">{C.payment}</legend>
                    <Radio id="pay-cash" name="pay" checked={!wallet} onChange={() => setWallet(false)} label={C.cash} />
                    <Radio
                      id="pay-wallet"
                      name="pay"
                      checked={wallet}
                      onChange={() => setWallet(true)}
                      label={
                        <span>
                          {C.wallet}
                          <span className="ms-2 text-xs text-ink-muted">{fill(C.walletBalance, { amount: fmtMoney(available) })}</span>
                        </span>
                      }
                    />
                  </fieldset>
                )}

                <Textarea
                  id="notes"
                  label={C.notes}
                  placeholder={C.notesHint}
                  value={notes}
                  onChange={(e) => setNotes(e.target.value)}
                  rows={2}
                />
              </div>
            </>
          )}

          <div className="surface flex flex-col gap-2 p-4 text-sm">
            <h2 className="heading-card">{C.summary}</h2>
            <div className="flex justify-between">
              <span className="text-ink-muted">{C.subtotal}</span>
              {quote?.blocked ? <span>—</span> : <Money value={quote?.subtotal ?? cart.subtotal} />}
            </div>
            {!quote ? (
              <p className="text-xs text-ink-muted">{quoting ? C.quoting : C.needAddress}</p>
            ) : quote.blocked ? null : (
              <>
                <div className="flex justify-between">
                  <span className="text-ink-muted">{C.delivery}</span>
                  {/* **والتوصيلُ المجانيُّ يُقال لا يُصفَّر بصمت** (قرارُ المالك ٢٠٢٦-١٠-٢٦) — الأصلُ مشطوبٌ وبعده «مجاني». */}
                  {waived > 0 ? (
                    <span className="flex items-center gap-2">
                      <s className="text-xs text-ink-muted">
                        <Money value={fee - extra + waived} />
                      </s>
                      {fee - extra > 0 ? (
                        <Money value={fee - extra} />
                      ) : (
                        <span className="font-bold text-success">{m.shop.free.badge}</span>
                      )}
                    </span>
                  ) : (
                    <Money value={fee - extra} />
                  )}
                </div>
                {extra > 0 && (
                  <div className="flex justify-between">
                    <span className="text-ink-muted">{C.sourcesFee}</span>
                    <Money value={extra} />
                  </div>
                )}
                {cut && cut.discount > 0 && (
                  <div className="flex justify-between text-success">
                    <span>{C.discount}</span>
                    <span>
                      {"− "}
                      <Money value={cut.discount} />
                    </span>
                  </div>
                )}
                {/* **والإجماليُّ مرّةً واحدة — في الشريط الثابت جنب «أرسل الطلب»** (قرارُ المالك ٢٠٢٦-١٠-٣٠). */}
              </>
            )}
          </div>

          {quoteError && <Alert>{quoteError}</Alert>}
          {av && !av.available && <AvailabilityNotice av={av} address={address} />}
          {!av && quote?.out_of_zone && <Alert>{C.outOfZone}</Alert>}
          {!av && quote?.zone_closed && <Alert tone="warning">{serviceText("zone_closed_now", "", quote.next_available_at)}</Alert>}
          {quote?.too_many_sources && (
            <Alert>{quote.max_sources <= 1 ? C.tooManyOne : fill(C.tooMany, { n: fmtNum(quote.max_sources) })}</Alert>
          )}

          {changes.length > 0 && (
            <Alert tone="warning" title={C.reviewTitle}>
              <ul className="list-disc space-y-1 ps-4">
                {changes.map((c, i) => (
                  <li key={`${c.type}-${c.menu_item_id}-${i}`}>{changeText(c)}</li>
                ))}
              </ul>
              {!canSubmit && (
                <button type="button" onClick={() => setAcked(fp)} className="mt-2 font-bold underline">
                  {C.reviewCta}
                </button>
              )}
            </Alert>
          )}

          {error && (
            <Alert>
              {error}
              {errorCode === "cod_limit_exceeded" && wa && (
                <a
                  href={`https://wa.me/${wa}?text=${encodeURIComponent(C.codPrefill)}`}
                  target="_blank"
                  rel="noreferrer"
                  className="mt-1 block font-bold underline"
                >
                  {C.contactWhatsapp}
                </a>
              )}
              {errorCode === "whatsapp_required" && (
                <Link href="/account#whatsapp" className="mt-1 block font-bold underline">
                  {m.shop.custom.verifyNow}
                </Link>
              )}
            </Alert>
          )}
          {uncertain && (
            <Alert tone="warning">
              {C.uncertain}
              <button
                type="button"
                onClick={() => {
                  clearAttempt(ORDER_SLOT);
                  setUncertain(false);
                  setError("");
                }}
                className="mt-1 block font-bold underline"
              >
                {C.uncertainAck}
              </button>
            </Alert>
          )}
        </aside>
      </div>

      {/* ══ **الشريطُ الثابت — الإجماليُّ وزرُّ الإرسال وسببُ تعطيله** (مرآةُ `CartScreen.kt`) ══
          **كان الزرُّ آخرَ صفحةٍ طويلة، وإن كان معطّلاً لم يُعرف لماذا.** والإجماليُّ بمعادلة الملخّص حرفاً. */}
      {user && (
        <div className="surface sticky bottom-[calc(3.75rem+env(safe-area-inset-bottom))] z-30 flex flex-col gap-2 p-3 md:bottom-0">
          {why && <p className="line-clamp-3 text-xs text-danger">{why}</p>}
          <div className="flex items-center gap-3">
            <div className="min-w-0 flex-1">
              <p className="text-xs text-ink-muted">{C.total}</p>
              {payable != null ? <Money value={payable} className="figure" /> : <p className="figure">—</p>}
            </div>
            <Button
              className="flex-1 py-3 text-base"
              onClick={send}
              disabled={busy || !address || !online || !ordering.open || blocked || !canSubmit}
            >
              {busy ? C.sending : C.send}
            </Button>
          </div>
        </div>
      )}
      <AddressSheet open={pickAddr} onClose={() => setPickAddr(false)} />
    </div>
  );
}

/**
 * **سطرُ السلّة — للصنف وللمشروب بالشكل نفسِه** (قرارُ المالك ٢٠٢٦-١٠-٣٠).
 *
 * صورةٌ، ثمّ النصُّ، ثمّ «− n +» والحذف (تحت النصّ على الشاشة الضيّقة وقبل السعر على العريضة)،
 * **ثمّ السعرُ في صندوقٍ ثابتِ العرض آخرَ السطر** — فتصطفّ الأسعارُ كلُّها عموداً واحداً
 * لا تنزاح من سطرٍ لسطر.
 */
function CartRow({
  media,
  children,
  price,
  qty,
  onLess,
  onMore,
  onRemove,
  moreDisabled = false,
}: {
  media: ReactNode;
  children: ReactNode;
  price: number;
  qty: number;
  onLess: () => void;
  onMore: () => void;
  onRemove: () => void;
  moreDisabled?: boolean;
}) {
  return (
    <div className="flex items-center gap-3">
      {media}
      <div className="flex min-w-0 flex-1 flex-col gap-1.5 sm:flex-row sm:items-center sm:gap-3">
        <div className="min-w-0 flex-1">{children}</div>
        <div className="flex shrink-0 items-center gap-1">
          <div className="flex items-center gap-0.5 rounded-control bg-field p-0.5">
            <button
              type="button"
              aria-label={m.shop.item.minus}
              onClick={onLess}
              className="taparea flex h-8 w-8 items-center justify-center rounded-control hover:bg-row-hover"
            >
              <IconMinus size={14} />
            </button>
            <span className="min-w-6 text-center text-sm font-bold tabular-nums">{fmtNum(qty)}</span>
            <button
              type="button"
              aria-label={m.shop.item.plus}
              onClick={onMore}
              disabled={moreDisabled}
              className="taparea flex h-8 w-8 items-center justify-center rounded-control hover:bg-row-hover disabled:opacity-40"
            >
              <IconAdd size={14} />
            </button>
          </div>
          <button
            type="button"
            aria-label={C.remove}
            onClick={onRemove}
            className="taparea flex h-8 w-8 shrink-0 items-center justify-center rounded-control text-danger hover:bg-row-hover"
          >
            <IconDelete size={16} />
          </button>
        </div>
      </div>
      <Money value={price} className="block w-24 shrink-0 whitespace-nowrap text-end text-sm font-bold tabular-nums text-accent-text" />
    </div>
  );
}
