"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **نموذجُ طلب الواتساب** — قرارُ المالك ٢٠٢٦-١١-٠٤ ثمّ ٢٠٢٦-١١-٠٦ (المبسَّط)
 * ══════════════════════════════════════════════════════════════════════
 *
 * «نكتب تفاصيل الطلب، اسم المتجر… وينزل طلب نظامي ع التطبيق». **وينزل للسائقين مباشرةً بلا موافقة.**
 * ثمّ بسّطه المالك (٢٠٢٦-١١-٠٦):
 *
 *	المتجر    من متاجرنا: الاسمُ من القائمة وحدَه · غيرُ تابع: الاسمُ مكتوباً وحدَه — لا رقمَ ولا خريطة
 *	الزبون    رقمُه وحدَه، واسمُه «زبون» — **والموقعُ والعنوانُ يطلبهما البوتُ منه ويملآنهما**
 *	ملاحظة    تفاصيلُ الطلب، اختياريّة
 *	الأجرة    يكتبها الموظّف (لا مسافةَ تُحسب قبل أن يصل الموقع) — والأساسُ والحدّان اقتراحاً
 *	البضاعة   **يكتب ثمنَها السائقُ لما يستلم**، والزبونُ تصله رسالةٌ بالمبالغ
 *
 * **و«أمانة من شخص»** (قرارُ المالك ٢٠٢٦-١١-١١): صاحبُ الأمانة مكانَ المتجر — رقمُه (واسمُه اختياريّ)،
 * والبوتُ يطلب الموقعَ من الطرفين، **والطلبُ لا ينزل للسائقين حتّى يصل موقعُ صاحب الأمانة** (أو يُلصق
 * رابطُ موقعه هنا). ولا ثمنَ بضاعة.
 *
 * **والحسبةُ والتحقّقُ في المحرّك** (`orders/whatsapp_order.go`) — الشاشةُ تعرض ما يقوله.
 * **والإرسالُ مرّةً واحدةً مهما أُعيد** — مفتاحُ المحاولة (`Idempotency-Key`).
 */

import Link from "next/link";
import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, errorText, fmtMoney, fmtNum } from "@rahalgo/i18n";
import { AdminCard, Alert, Button, Input, Textarea, IconSend } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { attemptKey, clearAttempt, isDecided } from "@/lib/customer/storage";
import { fill } from "@/lib/customer/texts";
import { StorePicker, type WaStore } from "./StorePicker";

const m = getMessages(defaultLocale);
const W = m.admin.whatsappOrders;
const SLOT = "whatsapp_order";

/** **أجرةُ الطلب كما يقولها المحرّك** — `QuoteWhatsAppOrder` بلا موقعٍ للزبون: أساسٌ وحدّان. */
interface WaQuote {
  fee: number;
  fee_min: number;
  fee_max: number;
  fee_required: boolean;
}

/** **ما يقوله الإنشاءُ للصفحة** — و`pickupPending` أمانةٌ تنتظر موقعَ صاحبها. */
export interface WaSent {
  id: string;
  number: number;
  noPoint: boolean;
  pickupPending: boolean;
}

/** **نقطةُ صاحب الأمانة من رابطٍ ملصق** — `resolve-location`. */
interface WaPoint {
  lat: number;
  lng: number;
}

export function WhatsAppOrderForm({ onSent }: { onSent: (o: WaSent) => void }) {
  const [source, setSource] = useState<"store" | "amanah">("store");
  const [senderName, setSenderName] = useState("");
  const [senderPhone, setSenderPhone] = useState("");
  const [senderLink, setSenderLink] = useState("");
  const [senderPoint, setSenderPoint] = useState<WaPoint | null>(null);
  const [linkState, setLinkState] = useState<"" | "busy" | "found" | "notFound">("");
  const [storeKind, setStoreKind] = useState<"ours" | "notOurs">("ours");
  const [store, setStore] = useState<WaStore | null>(null);
  const [storeName, setStoreName] = useState("");
  const [phone, setPhone] = useState("");
  const [details, setDetails] = useState("");
  const [quote, setQuote] = useState<WaQuote | null>(null);
  const [quoteError, setQuoteError] = useState("");
  const [feeText, setFeeText] = useState("");
  const [openFor, setOpenFor] = useState<number | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [tried, setTried] = useState(false);

  // ── الأساسُ والحدّان — من متجرنا إن اختير، أو نقطة صاحب الأمانة، وإلّا بلا نقطة ─────
  const amanah = source === "amanah";
  const storeId = !amanah && storeKind === "ours" ? (store?.id ?? "") : "";
  const pickLat = amanah ? senderPoint?.lat : undefined;
  const pickLng = amanah ? senderPoint?.lng : undefined;
  const fetchQuote = useCallback(async () => {
    setQuoteError("");
    try {
      const q = storeId
        ? `?store_id=${encodeURIComponent(storeId)}`
        : pickLat != null && pickLng != null
          ? `?pickup_lat=${pickLat}&pickup_lng=${pickLng}`
          : "";
      setQuote(await api<WaQuote>(`/api/v1/admin/whatsapp-orders/quote${q}`));
    } catch (e) {
      setQuote(null);
      setQuoteError(errorText(e));
    }
  }, [storeId, pickLat, pickLng]);
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

  // ── رابطُ موقع صاحب الأمانة ← نقطة (اختياريّ) — والمحرّكُ يقرؤه (`geolink`) ─────
  async function resolveSenderLink() {
    const url = senderLink.trim();
    setSenderPoint(null);
    if (!url) {
      setLinkState("");
      return;
    }
    setLinkState("busy");
    try {
      const r = await api<{ found: boolean; lat?: number; lng?: number }>(
        "/api/v1/admin/whatsapp-orders/resolve-location",
        { method: "POST", body: JSON.stringify({ url }) },
      );
      if (r.found && r.lat != null && r.lng != null) {
        setSenderPoint({ lat: r.lat, lng: r.lng });
        setLinkState("found");
      } else {
        setLinkState("notFound");
      }
    } catch {
      setLinkState("notFound");
    }
  }

  /** **رقمٌ بلا مفتاحٍ ولا صفر** — لمقارنة رقمي الطرفين (والمحرّكُ يحرسها أيضاً: `wa_same_phone`). */
  const digits = (v: string) => v.replace(/\D/g, "").replace(/^(00963|963|0)/, "");
  const feeValue = feeText.trim() === "" ? null : Number(feeText.replace(/\D/g, ""));
  const feeOk = feeValue != null && feeValue > 0;
  const hasStore = storeKind === "ours" ? !!store : !!storeName.trim();
  const samePhone = amanah && !!senderPhone.trim() && digits(senderPhone) === digits(phone);
  const missing = (
    amanah
      ? [!senderPhone.trim() && W.senderPhone, !phone.trim() && W.recipientPhone, !feeOk && W.fee]
      : [!hasStore && W.store, !phone.trim() && W.customerPhone, !feeOk && W.fee]
  ).filter(Boolean) as string[];

  function reset() {
    setSenderName("");
    setSenderPhone("");
    setSenderLink("");
    setSenderPoint(null);
    setLinkState("");
    setStore(null);
    setStoreName("");
    setPhone("");
    setDetails("");
    setFeeText("");
    setTried(false);
  }

  async function send() {
    if (busy) return;
    if (missing.length > 0 || samePhone || linkState === "busy") {
      setTried(true);
      return;
    }
    setBusy(true);
    setError("");
    try {
      const body: Record<string, unknown> = {
        recipient_phone: phone.trim(),
        details: details.trim(),
        fee: feeValue,
      };
      if (amanah) {
        body.amanah = true;
        body.sender_name = senderName.trim();
        body.sender_phone = senderPhone.trim();
        if (senderPoint) {
          body.sender_lat = senderPoint.lat;
          body.sender_lng = senderPoint.lng;
        }
      } else if (storeKind === "ours" && store) body.store_id = store.id;
      else body.store_name = storeName.trim();
      const r = await api<{ id: string; number: number; has_point?: boolean; pickup_pending?: boolean }>(
        "/api/v1/admin/whatsapp-orders",
        {
          method: "POST",
          headers: { "Idempotency-Key": attemptKey(SLOT) },
          body: JSON.stringify(body),
        },
      );
      clearAttempt(SLOT);
      reset();
      onSent({ id: r.id, number: r.number, noPoint: r.has_point !== true, pickupPending: r.pickup_pending === true });
    } catch (e) {
      if (isDecided(e)) clearAttempt(SLOT);
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="space-y-4">
      {/* ── ٠ · نوعُ الطلب — من متجر أو أمانة من شخص (قرارُ المالك ٢٠٢٦-١١-١١) ─────────── */}
      <div className="flex flex-wrap gap-2" role="radiogroup" aria-label={W.source}>
        {(["store", "amanah"] as const).map((k) => (
          <Button
            key={k}
            variant={source === k ? "primary" : "secondary"}
            role="radio"
            aria-checked={source === k}
            disabled={busy}
            onClick={() => {
              setSource(k);
              setTried(false);
            }}
          >
            {k === "store" ? W.fromStore : W.amanah}
          </Button>
        ))}
      </div>

      {/* ── ١أ · صاحبُ الأمانة — رقمُه، والبوتُ يطلب موقعَه ───────────────────── */}
      {amanah && (
        <AdminCard title={W.sender}>
          <div className="space-y-3">
            <Input
              label={W.senderName}
              value={senderName}
              maxLength={120}
              disabled={busy}
              onChange={(e) => setSenderName(e.target.value)}
            />
            <div className="space-y-1">
              <Input
                label={W.senderPhone}
                dir="ltr"
                inputMode="tel"
                value={senderPhone}
                disabled={busy}
                error={tried && !senderPhone.trim() ? W.required : tried && samePhone ? W.samePhone : undefined}
                onChange={(e) => setSenderPhone(e.target.value)}
              />
              <p className="text-sm text-ink-muted">{W.senderPhoneHint}</p>
            </div>
            <div className="space-y-1">
              <Input
                label={W.senderLink}
                dir="ltr"
                value={senderLink}
                disabled={busy}
                onChange={(e) => {
                  setSenderLink(e.target.value);
                  setSenderPoint(null);
                  setLinkState("");
                }}
                onBlur={() => void resolveSenderLink()}
              />
              <p className="text-sm text-ink-muted">{W.senderLinkHint}</p>
              {linkState === "busy" && <p className="text-sm text-ink-muted">{W.searching}</p>}
              {linkState === "found" && <p className="text-sm text-success">{W.senderLinkFound}</p>}
              {linkState === "notFound" && <p className="text-sm text-danger">{W.senderLinkNotFound}</p>}
            </div>
          </div>
        </AdminCard>
      )}

      {/* ── ١ · المتجر — اسمُه وحدَه ───────────────────────────────────── */}
      {!amanah && (
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
              <StorePicker value={store} disabled={busy} onPick={setStore} />
              {tried && !store && <p className="text-sm text-danger">{fill(W.missing, { fields: W.store })}</p>}
            </>
          ) : (
            <Input
              label={W.storeName}
              value={storeName}
              maxLength={120}
              disabled={busy}
              error={tried && !storeName.trim() ? W.required : undefined}
              onChange={(e) => setStoreName(e.target.value)}
            />
          )}
        </div>
        </AdminCard>
      )}

      {/* ── ٢ · الزبون — رقمُه وحدَه، والبوتُ يطلب موقعَه وعنوانَه ─────────── */}
      <AdminCard title={amanah ? W.recipient : W.customer}>
        <div className="space-y-3">
          <div className="space-y-1">
            <Input
              label={amanah ? W.recipientPhone : W.customerPhone}
              dir="ltr"
              inputMode="tel"
              value={phone}
              disabled={busy}
              error={tried && !phone.trim() ? W.required : undefined}
              onChange={(e) => setPhone(e.target.value)}
            />
            <p className="text-sm text-ink-muted">{amanah ? W.recipientPhoneHint : W.customerPhoneHint}</p>
            {openFor != null && <Alert tone="warning">{fill(W.openForPhone, { n: openFor })}</Alert>}
          </div>
          <Textarea
            label={W.note}
            placeholder={W.detailsHint}
            value={details}
            maxLength={500}
            autoGrow
            disabled={busy}
            onChange={(e) => setDetails(e.target.value)}
          />
        </div>
      </AdminCard>

      {/* ── ٣ · الأجرة والإرسال ─────────────────────────────────────── */}
      <AdminCard title={W.fee}>
        <div className="space-y-3">
          {quoteError && <Alert tone="error">{quoteError}</Alert>}
          <div className="flex flex-wrap items-end gap-2">
            <Input
              label={W.fee}
              inputMode="numeric"
              dir="ltr"
              value={feeText}
              disabled={busy}
              wrapperClassName="w-40"
              error={tried && !feeOk ? W.required : undefined}
              onChange={(e) => setFeeText(e.target.value)}
            />
            {quote && (
              <span className="pb-2 text-sm text-ink-muted">
                {fill(W.feeBase, { fee: fmtMoney(quote.fee) })} · {fill(W.feeRange, { min: fmtNum(quote.fee_min), max: fmtNum(quote.fee_max) })}
              </span>
            )}
          </div>
          <p className="text-sm text-ink-muted">{amanah ? W.amanahNoGoods : W.goodsByDriver}</p>
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
