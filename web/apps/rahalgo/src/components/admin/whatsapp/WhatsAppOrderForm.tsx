"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **نموذجُ طلب الواتساب** — قرارُ المالك ٢٠٢٦-١١-٠٤ ثمّ ٢٠٢٦-١١-٠٥
 * ══════════════════════════════════════════════════════════════════════
 *
 * «نكتب تفاصيل الطلب، اسم المتجر، اسم المتسلم وموقعه على الخريطة، وينزل طلب نظامي ع
 * التطبيق». **وينزل للسائقين مباشرةً بلا موافقة** (البند ٥).
 *
 *	المتجر    من متاجرنا (بحث) أو متجرٌ غيرُ تابع (اسمٌ ونقطةٌ على الخريطة)
 *	الطلب     تفاصيلُه، وثمنُ البضاعة إن عُرف — يدفعه السائقُ ويقبضه، خارجَ الحسابات
 *	المستلم   اسمُه ورقمُه وعنوانُه ونقطتُه على الخريطة
 *	الأجرة    تُحسب كالتوصيلات بالمسافة ويعدّلها الموظّف بحدّيها (البند ٣)
 *
 * **والحسبةُ والتحقّقُ في المحرّك** (`orders/whatsapp_order.go`) — الشاشةُ تعرض ما يقوله.
 * **والإرسالُ مرّةً واحدةً مهما أُعيد** — مفتاحُ المحاولة (`Idempotency-Key`).
 */

import Link from "next/link";
import dynamic from "next/dynamic";
import { useCallback, useEffect, useRef, useState } from "react";
import { getMessages, defaultLocale, errorText, fmtMoney, fmtNum } from "@rahalgo/i18n";
import { AdminCard, Alert, Button, Input, Textarea, IconLink, IconSend } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { attemptKey, clearAttempt, isDecided } from "@/lib/customer/storage";
import { dollars, fill } from "@/lib/customer/texts";
import { StorePicker, type WaStore } from "./StorePicker";

const PickMap = dynamic(() => import("@rahalgo/ui/map").then((mod) => mod.PickMap), { ssr: false });

const m = getMessages(defaultLocale);
const W = m.admin.whatsappOrders;
const SLOT = "whatsapp_order";

type Point = { lat: number; lng: number };
type LinkState = "idle" | "checking" | "found" | "notFound";

/** **أجرةُ الطلب كما يقولها المحرّك** — `QuoteWhatsAppOrder`. */
interface WaQuote {
  fee: number;
  fee_basis: string;
  distance_m: number;
  zone_name: string;
  share_percent: number;
  share_fixed: number;
  driver_fee: number;
  platform_share: number;
  fee_min: number;
  fee_max: number;
}

/** **موقعٌ على الخريطة أو برابطٍ يُلصق** — للمتجر غير التابع وللمستلِم. */
function LocationField({
  label,
  point,
  onPoint,
  disabled,
  error,
}: {
  label: string;
  point: Point | null;
  onPoint: (p: Point | null) => void;
  disabled: boolean;
  error?: boolean;
}) {
  const [link, setLink] = useState("");
  const [state, setState] = useState<LinkState>("idle");
  const seq = useRef(0);
  const timer = useRef<ReturnType<typeof setTimeout> | null>(null);

  useEffect(() => () => {
    if (timer.current) clearTimeout(timer.current);
  }, []);

  function onLink(text: string) {
    const v = text.slice(0, 2000);
    setLink(v);
    if (timer.current) clearTimeout(timer.current);
    const my = ++seq.current;
    if (!v.includes("http")) {
      setState("idle");
      return;
    }
    setState("checking");
    timer.current = setTimeout(async () => {
      try {
        const r = await api<{ found: boolean; lat: number; lng: number }>(
          "/api/v1/admin/whatsapp-orders/resolve-location",
          { method: "POST", body: JSON.stringify({ url: v }) },
        );
        if (my !== seq.current) return;
        if (r.found) {
          onPoint({ lat: r.lat, lng: r.lng });
          setState("found");
        } else {
          setState("notFound");
        }
      } catch {
        if (my === seq.current) setState("notFound");
      }
    }, 400);
  }

  return (
    <div className="space-y-2">
      <p className={`text-sm font-medium ${error ? "text-danger" : ""}`}>{label}</p>
      <PickMap api={api} lat={point?.lat ?? null} lng={point?.lng ?? null} height="h-56" onPick={(la, ln) => onPoint({ lat: la, lng: ln })} />
      <Input
        label={W.pasteLink}
        value={link}
        dir="ltr"
        disabled={disabled}
        icon={<IconLink size={16} />}
        onChange={(e) => onLink(e.target.value)}
      />
      {state === "checking" && <p className="text-sm text-ink-muted">{W.linkChecking}</p>}
      {state === "found" && <p className="text-sm text-success">{W.linkFound}</p>}
      {state === "notFound" && <p className="text-sm text-warning">{W.linkNotFound}</p>}
    </div>
  );
}

/** **دائرتا العملة** — الليرةُ أو الدولار لثمن البضاعة. */
function CurrencyPick({ value, onPick, disabled }: { value: "SYP" | "USD"; onPick: (c: "SYP" | "USD") => void; disabled: boolean }) {
  return (
    <div className="flex shrink-0 gap-2" role="radiogroup" aria-label={W.goods}>
      {(["SYP", "USD"] as const).map((c) => (
        <Button
          key={c}
          size="md"
          variant={value === c ? "primary" : "secondary"}
          role="radio"
          aria-checked={value === c}
          aria-label={c === "SYP" ? W.currencySyp : W.currencyUsd}
          disabled={disabled}
          onClick={() => onPick(c)}
        >
          {c === "SYP" ? m.common.currency : "$"}
        </Button>
      ))}
    </div>
  );
}

export function WhatsAppOrderForm({ onSent }: { onSent: (o: { id: string; number: number }) => void }) {
  const [storeKind, setStoreKind] = useState<"ours" | "notOurs">("ours");
  const [store, setStore] = useState<WaStore | null>(null);
  const [storePhone, setStorePhone] = useState("");
  const [storeName, setStoreName] = useState("");
  const [storeAddress, setStoreAddress] = useState("");
  const [storePoint, setStorePoint] = useState<Point | null>(null);
  const [details, setDetails] = useState("");
  const [goods, setGoods] = useState("");
  const [currency, setCurrency] = useState<"SYP" | "USD">("SYP");
  const [name, setName] = useState("");
  const [phone, setPhone] = useState("");
  const [address, setAddress] = useState("");
  const [point, setPoint] = useState<Point | null>(null);
  const [driverNote, setDriverNote] = useState("");
  const [quote, setQuote] = useState<WaQuote | null>(null);
  const [quoteError, setQuoteError] = useState("");
  const [feeEdit, setFeeEdit] = useState(false);
  const [feeText, setFeeText] = useState("");
  const [openFor, setOpenFor] = useState<number | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [tried, setTried] = useState(false);
  // **ومفتاحُ الخريطة يُبدَّل بعد الإرسال** — فتُرسم الخريطتان فارغتين للطلب التالي.
  const [round, setRound] = useState(0);

  // ── الأجرة: كلّما اكتملت النقطتان ─────────────────────────────────
  const pickup: { storeId?: string; at?: Point } | null =
    storeKind === "ours" ? (store ? { storeId: store.id } : null) : storePoint ? { at: storePoint } : null;
  const quoteKey = pickup && point ? `${pickup.storeId ?? ""}|${pickup.at?.lat ?? ""},${pickup.at?.lng ?? ""}|${point.lat},${point.lng}` : "";
  const fetchQuote = useCallback(async () => {
    setQuote(null);
    setQuoteError("");
    if (!pickup || !point) return;
    const from = pickup.storeId
      ? `store_id=${encodeURIComponent(pickup.storeId)}`
      : `pickup_lat=${pickup.at!.lat}&pickup_lng=${pickup.at!.lng}`;
    try {
      setQuote(await api<WaQuote>(`/api/v1/admin/whatsapp-orders/quote?${from}&lat=${point.lat}&lng=${point.lng}`));
    } catch (e) {
      setQuoteError(errorText(e));
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps -- **المفتاحُ يجمع النقطتين**
  }, [quoteKey]);
  useEffect(() => {
    void fetchQuote();
  }, [fetchQuote]);

  // ── طلبٌ مفتوحٌ للرقم نفسِه — كي لا يُرسَل سائقان إلى بابٍ واحد ─────
  useEffect(() => {
    const p = phone.trim();
    setOpenFor(null);
    if (p.replace(/\D/g, "").length < 7) return;
    const t = setTimeout(async () => {
      try {
        const r = await api<{ orders: { number: number }[] }>(
          `/api/v1/admin/whatsapp-orders?open=1&q=${encodeURIComponent(p)}`,
        );
        setOpenFor(r.orders?.[0]?.number ?? null);
      } catch {
        /* **تلميحٌ لا شرط** — تعذّرُه لا يمنع الإرسال. */
      }
    }, 500);
    return () => clearTimeout(t);
  }, [phone]);

  const goodsValue =
    goods.trim() === ""
      ? null
      : currency === "USD"
        ? Number(goods.replace(",", ".")) > 0
          ? Math.round(Number(goods.replace(",", ".")) * 100) / 100
          : NaN
        : Number(goods.replace(/\D/g, "")) || NaN;
  const feeValue = feeEdit && feeText.trim() !== "" ? Number(feeText.replace(/\D/g, "")) : null;
  const fee = feeValue ?? quote?.fee ?? null;
  const goodsText = goodsValue != null && !Number.isNaN(goodsValue) ? (currency === "USD" ? dollars(goodsValue) : fmtMoney(goodsValue)) : "";

  const hasStore = storeKind === "ours" ? !!store : !!storeName.trim() && !!storePoint;
  const missing = [
    !hasStore && W.store,
    !details.trim() && W.details,
    goodsValue != null && Number.isNaN(goodsValue) && W.goods,
    !name.trim() && W.recipientName,
    !phone.trim() && W.recipientPhone,
    !address.trim() && W.address,
    !point && W.recipientMap,
  ].filter(Boolean) as string[];

  function reset() {
    setStore(null);
    setStorePhone("");
    setStoreName("");
    setStoreAddress("");
    setStorePoint(null);
    setDetails("");
    setGoods("");
    setCurrency("SYP");
    setName("");
    setPhone("");
    setAddress("");
    setPoint(null);
    setDriverNote("");
    setFeeEdit(false);
    setFeeText("");
    setTried(false);
    setRound((r) => r + 1);
  }

  async function send() {
    if (busy) return;
    if (missing.length > 0) {
      setTried(true);
      return;
    }
    setBusy(true);
    setError("");
    try {
      const body: Record<string, unknown> = {
        details: details.trim(),
        goods_amount: goodsValue,
        amount_currency: currency,
        recipient_name: name.trim(),
        recipient_phone: phone.trim(),
        address_text: address.trim(),
        lat: point!.lat,
        lng: point!.lng,
        driver_note: driverNote.trim(),
      };
      if (storeKind === "ours" && store) {
        body.store_id = store.id;
        if (storePhone.trim() && storePhone.trim() !== store.phone) body.store_phone = storePhone.trim();
      } else {
        body.store_name = storeName.trim();
        body.store_phone = storePhone.trim();
        body.store_address = storeAddress.trim();
        body.store_lat = storePoint!.lat;
        body.store_lng = storePoint!.lng;
      }
      if (feeValue != null && quote && feeValue !== quote.fee) body.fee = feeValue;
      const r = await api<{ id: string; number: number }>("/api/v1/admin/whatsapp-orders", {
        method: "POST",
        headers: { "Idempotency-Key": attemptKey(SLOT) },
        body: JSON.stringify(body),
      });
      clearAttempt(SLOT);
      reset();
      onSent(r);
    } catch (e) {
      if (isDecided(e)) clearAttempt(SLOT);
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="space-y-4">
      {/* ── ١ · المتجر ─────────────────────────────────────────────── */}
      <AdminCard title={W.store}>
        <div className="space-y-3">
          <div className="flex flex-wrap gap-2" role="radiogroup" aria-label={W.store}>
            {(["ours", "notOurs"] as const).map((k) => (
              <Button
                key={k}
                variant={storeKind === k ? "primary" : "secondary"}
                role="radio"
                aria-checked={storeKind === k}
                disabled={busy}
                onClick={() => setStoreKind(k)}
              >
                {k === "ours" ? W.ours : W.notOurs}
              </Button>
            ))}
          </div>
          {storeKind === "ours" ? (
            <>
              <StorePicker
                value={store}
                disabled={busy}
                onPick={(s) => {
                  setStore(s);
                  setStorePhone(s?.phone ?? "");
                }}
              />
              {store && (
                <Input label={W.storePhone} dir="ltr" value={storePhone} disabled={busy} onChange={(e) => setStorePhone(e.target.value)} />
              )}
              {tried && !store && <p className="text-sm text-danger">{fill(W.missing, { fields: W.store })}</p>}
            </>
          ) : (
            <div className="space-y-3">
              <Input
                label={W.storeName}
                value={storeName}
                maxLength={120}
                disabled={busy}
                error={tried && !storeName.trim() ? W.required : undefined}
                onChange={(e) => setStoreName(e.target.value)}
              />
              <Input label={W.storePhoneOptional} dir="ltr" value={storePhone} disabled={busy} onChange={(e) => setStorePhone(e.target.value)} />
              <Input label={W.storeAddress} value={storeAddress} maxLength={300} disabled={busy} onChange={(e) => setStoreAddress(e.target.value)} />
              <LocationField
                key={`store-${round}`}
                label={W.storeMap}
                point={storePoint}
                onPoint={setStorePoint}
                disabled={busy}
                error={tried && !storePoint}
              />
            </div>
          )}
        </div>
      </AdminCard>

      {/* ── ٢ · الطلب ──────────────────────────────────────────────── */}
      <AdminCard title={W.details}>
        <div className="space-y-3">
          <Textarea
            label={W.details}
            placeholder={W.detailsHint}
            value={details}
            maxLength={500}
            autoGrow
            disabled={busy}
            error={tried && !details.trim() ? W.required : undefined}
            onChange={(e) => setDetails(e.target.value)}
          />
          <div className="flex items-end gap-2">
            <Input
              label={W.goods}
              inputMode="decimal"
              dir="ltr"
              value={goods}
              disabled={busy}
              wrapperClassName="flex-1"
              error={goodsValue != null && Number.isNaN(goodsValue) ? W.badAmount : undefined}
              onChange={(e) => setGoods(e.target.value)}
            />
            <CurrencyPick value={currency} onPick={setCurrency} disabled={busy} />
          </div>
          <p className="text-sm text-ink-muted">{W.goodsHint}</p>
        </div>
      </AdminCard>

      {/* ── ٣ · المستلم ────────────────────────────────────────────── */}
      <AdminCard title={W.recipient}>
        <div className="space-y-3">
          <Input
            label={W.recipientName}
            value={name}
            maxLength={120}
            disabled={busy}
            error={tried && !name.trim() ? W.required : undefined}
            onChange={(e) => setName(e.target.value)}
          />
          <div className="space-y-1">
            <Input
              label={W.recipientPhone}
              dir="ltr"
              inputMode="tel"
              value={phone}
              disabled={busy}
              error={tried && !phone.trim() ? W.required : undefined}
              onChange={(e) => setPhone(e.target.value)}
            />
            <p className="text-sm text-ink-muted">{W.recipientPhoneHint}</p>
            {openFor != null && <Alert tone="warning">{fill(W.openForPhone, { n: openFor })}</Alert>}
          </div>
          <Textarea
            label={W.address}
            value={address}
            maxLength={300}
            autoGrow
            disabled={busy}
            error={tried && !address.trim() ? W.required : undefined}
            onChange={(e) => setAddress(e.target.value)}
          />
          <LocationField key={`drop-${round}`} label={W.recipientMap} point={point} onPoint={setPoint} disabled={busy} error={tried && !point} />
          <Textarea label={W.driverNote} value={driverNote} maxLength={500} autoGrow disabled={busy} onChange={(e) => setDriverNote(e.target.value)} />
        </div>
      </AdminCard>

      {/* ── ٤ · الأجرة والإرسال ─────────────────────────────────────── */}
      <AdminCard title={W.fee}>
        <div className="space-y-3">
          {!quote && !quoteError && <p className="text-sm text-ink-muted">{W.feeNeedsPoints}</p>}
          {quoteError && <Alert tone="error">{quoteError}</Alert>}
          {quote && (
            <div className="space-y-1 text-sm">
              <p className="font-bold">
                {fill(W.feeLine, { fee: fmtMoney(quote.fee), km: fmtNum(Math.round(quote.distance_m / 100) / 10) })}
              </p>
              <p className="text-ink-muted">
                {fill(W.shareLine, { share: fmtMoney(quote.platform_share), driver: fmtMoney(quote.driver_fee) })}
              </p>
            </div>
          )}
          {quote && (
            <div className="flex flex-wrap items-end gap-2">
              {feeEdit ? (
                <>
                  <Input
                    label={W.feeEdit}
                    inputMode="numeric"
                    dir="ltr"
                    value={feeText}
                    disabled={busy}
                    wrapperClassName="w-40"
                    onChange={(e) => setFeeText(e.target.value)}
                  />
                  <span className="pb-2 text-sm text-ink-muted">
                    {fill(W.feeRange, { min: fmtNum(quote.fee_min), max: fmtNum(quote.fee_max) })}
                  </span>
                  <Button size="sm" variant="ghost" disabled={busy} onClick={() => { setFeeEdit(false); setFeeText(""); }}>
                    {W.feeReset}
                  </Button>
                </>
              ) : (
                <Button size="sm" variant="secondary" disabled={busy} onClick={() => { setFeeEdit(true); setFeeText(String(quote.fee)); }}>
                  {W.feeEdit}
                </Button>
              )}
            </div>
          )}
          {fee != null && !Number.isNaN(fee) && (
            <p className="text-sm font-bold">
              {goodsText
                ? fill(W.total, { goods: goodsText, fee: fmtMoney(fee) })
                : fill(W.totalNoGoods, { fee: fmtMoney(fee) })}
            </p>
          )}
          {tried && missing.length > 0 && (
            <Alert tone="warning">{fill(W.missing, { fields: missing.join(m.common.listSep) })}</Alert>
          )}
          {error && <Alert tone="error">{error}</Alert>}
          <div className="flex justify-end">
            <Button size="lg" disabled={busy} onClick={() => void send()}>
              <span className="inline-flex items-center gap-2">
                <IconSend size={18} />
                {W.send}
              </span>
            </Button>
          </div>
        </div>
      </AdminCard>
    </div>
  );
}

/** **رابطُ الطلب في لوحة الطلبات** — بعد الإرسال. */
export function OpenOrderLink({ id }: { id: string }) {
  return (
    <Link className="font-bold underline" href={`/dashboard/orders?id=${encodeURIComponent(id)}`}>
      {W.openOrder}
    </Link>
  );
}
