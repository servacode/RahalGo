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

export function WhatsAppOrderForm({ onSent }: { onSent: (o: { id: string; number: number; noPoint: boolean }) => void }) {
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

  // ── الأساسُ والحدّان — من متجرنا إن اختير، وإلّا بلا نقطة ─────────────
  const storeId = storeKind === "ours" ? (store?.id ?? "") : "";
  const fetchQuote = useCallback(async () => {
    setQuoteError("");
    try {
      const q = storeId ? `?store_id=${encodeURIComponent(storeId)}` : "";
      setQuote(await api<WaQuote>(`/api/v1/admin/whatsapp-orders/quote${q}`));
    } catch (e) {
      setQuote(null);
      setQuoteError(errorText(e));
    }
  }, [storeId]);
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

  const feeValue = feeText.trim() === "" ? null : Number(feeText.replace(/\D/g, ""));
  const feeOk = feeValue != null && feeValue > 0;
  const hasStore = storeKind === "ours" ? !!store : !!storeName.trim();
  const missing = [!hasStore && W.store, !phone.trim() && W.customerPhone, !feeOk && W.fee].filter(Boolean) as string[];

  function reset() {
    setStore(null);
    setStoreName("");
    setPhone("");
    setDetails("");
    setFeeText("");
    setTried(false);
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
        recipient_phone: phone.trim(),
        details: details.trim(),
        fee: feeValue,
      };
      if (storeKind === "ours" && store) body.store_id = store.id;
      else body.store_name = storeName.trim();
      const r = await api<{ id: string; number: number; has_point?: boolean }>("/api/v1/admin/whatsapp-orders", {
        method: "POST",
        headers: { "Idempotency-Key": attemptKey(SLOT) },
        body: JSON.stringify(body),
      });
      clearAttempt(SLOT);
      reset();
      onSent({ id: r.id, number: r.number, noPoint: r.has_point !== true });
    } catch (e) {
      if (isDecided(e)) clearAttempt(SLOT);
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  return (
    <div className="space-y-4">
      {/* ── ١ · المتجر — اسمُه وحدَه ───────────────────────────────────── */}
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

      {/* ── ٢ · الزبون — رقمُه وحدَه، والبوتُ يطلب موقعَه وعنوانَه ─────────── */}
      <AdminCard title={W.customer}>
        <div className="space-y-3">
          <div className="space-y-1">
            <Input
              label={W.customerPhone}
              dir="ltr"
              inputMode="tel"
              value={phone}
              disabled={busy}
              error={tried && !phone.trim() ? W.required : undefined}
              onChange={(e) => setPhone(e.target.value)}
            />
            <p className="text-sm text-ink-muted">{W.customerPhoneHint}</p>
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
          <p className="text-sm text-ink-muted">{W.goodsByDriver}</p>
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
