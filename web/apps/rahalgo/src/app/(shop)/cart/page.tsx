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

import { useCallback, useEffect, useMemo, useState } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale, errorText, fmtMoney, fmtNum } from "@rahalgo/i18n";
import {
  Alert,
  Button,
  ButtonLink,
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
import { useCart, lineKey, linePayload, unitPrice } from "@/lib/customer/cart";
import { useCustomer } from "@/lib/customer/context";
import { attemptKey, clearAttempt, isDecided } from "@/lib/customer/storage";
import { changeText, fill, serviceText } from "@/lib/customer/texts";
import type { CartChange, PromoPreview, Quote } from "@/lib/customer/types";
import { AddressChip, AddressSheet } from "@/components/customer/AddressSheet";
import { AvailabilityNotice, useOrderingState } from "@/components/customer/ServiceNotices";
import { useShopToast } from "@/components/customer/toast";

const m = getMessages(defaultLocale);
const C = m.shop.cart;
const ORDER_SLOT = "order";

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

  const point = address ? `${address.lat},${address.lng}` : "";
  const linesKey = cart.lines.map((l) => `${lineKey(l)}:${l.qty}`).join(",");

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
  const extra = quote && !cut ? Math.min(Math.max(quote.sources_fee, 0), fee) : 0;
  const payable = quote ? (cut ? Math.max(0, quote.subtotal + fee - cut.discount) : quote.total) : null;
  const walletBlocked = payable != null && available < payable;

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
  const wa = (platform?.social.whatsapp ?? "").replace(/\D/g, "");

  return (
    <div className="grid grid-cols-1 gap-4 lg:grid-cols-[minmax(0,1fr)_minmax(0,24rem)] lg:items-start">
      <section className="flex flex-col gap-3">
        <div>
          <h1 className="heading-page">{C.title}</h1>
          <p className="text-sm text-ink-muted">{C.hint}</p>
        </div>
        <ul className="surface divide-y divide-line-soft">
          {cart.lines.map((l) => {
            const k = lineKey(l);
            const img = mediaUrl(l.item.image_thumb_url ?? l.item.image_url);
            return (
              <li key={k} className="flex items-center gap-3 p-3">
                <span className="h-14 w-14 shrink-0 overflow-hidden rounded-control bg-field">
                  {img && <img src={img} alt="" loading="lazy" className="h-full w-full object-cover" />}
                </span>
                <div className="min-w-0 flex-1">
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
                  <Money value={unitPrice(l) * l.qty} className="text-sm font-bold text-accent-text" />
                </div>
                <div className="flex shrink-0 items-center gap-0.5 rounded-control bg-field p-0.5">
                  <button
                    type="button"
                    aria-label={m.shop.item.minus}
                    onClick={() => cart.setQty(k, l.qty - 1)}
                    className="taparea flex h-8 w-8 items-center justify-center rounded-control hover:bg-row-hover"
                  >
                    <IconMinus size={14} />
                  </button>
                  <span className="min-w-6 text-center text-sm font-bold tabular-nums">{fmtNum(l.qty)}</span>
                  <button
                    type="button"
                    aria-label={m.shop.item.plus}
                    onClick={() => cart.setQty(k, l.qty + 1)}
                    className="taparea flex h-8 w-8 items-center justify-center rounded-control hover:bg-row-hover"
                  >
                    <IconAdd size={14} />
                  </button>
                </div>
                <button
                  type="button"
                  aria-label={C.remove}
                  onClick={() => cart.setQty(k, 0)}
                  className="taparea flex h-8 w-8 shrink-0 items-center justify-center rounded-control text-danger hover:bg-row-hover"
                >
                  <IconDelete size={16} />
                </button>
              </li>
            );
          })}
        </ul>
        <button type="button" onClick={cart.clear} className="w-fit text-sm text-danger hover:underline">
          {C.clear}
        </button>
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

              <fieldset className="flex flex-col gap-2">
                <legend className="mb-1 text-sm font-medium">{C.payment}</legend>
                <Radio id="pay-cash" name="pay" checked={!wallet} onChange={() => setWallet(false)} label={C.cash} />
                <Radio
                  id="pay-wallet"
                  name="pay"
                  checked={wallet}
                  disabled={walletBlocked}
                  onChange={() => setWallet(true)}
                  label={
                    <span>
                      {C.wallet}
                      <span className="ms-2 text-xs text-ink-muted">{fill(C.walletBalance, { amount: fmtMoney(available) })}</span>
                    </span>
                  }
                />
                {walletBlocked && (
                  <p className="text-xs text-ink-muted">
                    {fill(C.walletShort, { amount: fmtMoney(available) })} {C.codHint}
                  </p>
                )}
              </fieldset>

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
            <Money value={quote?.subtotal ?? cart.subtotal} />
          </div>
          {!quote ? (
            <p className="text-xs text-ink-muted">{quoting ? C.quoting : C.needAddress}</p>
          ) : (
            <>
              <div className="flex justify-between">
                <span className="text-ink-muted">{C.delivery}</span>
                <Money value={fee - extra} />
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
              <div className="flex justify-between border-t border-line-soft pt-2 font-bold">
                <span>{C.total}</span>
                <Money value={payable ?? quote.total} className="text-accent-text" />
              </div>
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

        {user && (
          <Button
            className="w-full py-3 text-base"
            onClick={send}
            disabled={busy || !address || !ordering.open || blocked || !canSubmit}
          >
            {busy ? C.sending : C.send}
          </Button>
        )}
      </aside>
      <AddressSheet open={pickAddr} onClose={() => setPickAddr(false)} />
    </div>
  );
}
