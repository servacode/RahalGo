"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **طلبٌ خاصّ — ما ليس في المنصّة** (مرآةُ `CustomScreen.kt`)
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-١٠-١٧: المرحلةُ الثانية من زبون الويب.)
 *
 * **أمانةٌ أم مشتريات** — الأمانةُ بلا ثمنِ بضاعة. **ولا سعرَ ولا متجرَ
 * يُسأل عنه**: السائقُ يشتري ويُحدَّد المبلغُ بعد ذلك، **ثمّ يؤكّده الزبونُ
 * من نافذة السعر** (القشرة) ويختار الدفعَ هناك.
 *
 * **ولا تُختار طريقةُ الدفع هنا** — المحفظةُ لا تُقبَل قبل معرفة المبلغ،
 * **فيُرسَل نقداً افتراضاً آمناً** كما يفعل التطبيق.
 *
 * **والأجرةُ من اللوحة إن حدّدتها** (`custom_delivery`) — وإلّا قيل إنّ
 * السائقَ يحدّدها. **والطلبُ يُنشأ مرّةً واحدةً** بمفتاح المحاولة.
 */

import Link from "next/link";
import { useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale, errorText, fmtMoney, fmtRef } from "@rahalgo/i18n";
import { Alert, Button, IconLocation, Input, LoadingState, Textarea } from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useCustomer } from "@/lib/customer/context";
import { attemptKey, clearAttempt, isDecided } from "@/lib/customer/storage";
import { customFee, fill, freeCustomOn, freeUntilText } from "@/lib/customer/texts";
import { AddressChip, AddressSheet } from "@/components/customer/AddressSheet";
import { NeedLogin } from "@/components/customer/NeedLogin";
import { useOrderingState, useAddGate, AvailabilityNotice } from "@/components/customer/ServiceNotices";
import { useShopToast } from "@/components/customer/toast";

const m = getMessages(defaultLocale);
const K = m.shop.custom;
const SLOT = "custom";
/** **حدودُ المحرّك** — مرآةُ `TextLimits` في أندرويد. */
const MAX_REQUEST = 1000;
const MAX_NOTES = 500;

type Mode = "purchase" | "amanah";

/** **«اطلب الموقع منه»** — ردُّ الخادم وحالُه (مرآةُ «إحضار» المتجر). */
type AskReq = { id: string; whatsapp_sent: boolean; link: string; wa_url: string };
type AskState = "idle" | "sending" | "waiting" | "received" | "expired";
/** **حدُّ اسم الشخص في المحرّك** (`maxPersonName`). */
const MAX_NAME = 80;

export default function CustomOrderPage() {
  const { user, loading } = useAuth();
  const { address, platform, freeDelivery } = useCustomer();
  const ordering = useOrderingState();
  // **وعنوانٌ لا نصله يُقال قبل الإرسال** (قرارُ المالك ٢٠٢٦-١٠-٢٦) — كالسلّة.
  const gate = useAddGate();
  const router = useRouter();
  const toast = useShopToast();
  const [mode, setMode] = useState<Mode>("purchase");
  const [request, setRequest] = useState("");
  const [notes, setNotes] = useState("");
  // **صاحبُ الأمانة ورقمُه، و«اطلب الموقع منه»** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — يلغي خريطةَ الصباح:
  // الزبونُ لا يعرف أين الأمانة. **كـ«إحضار» المتجر**: يصل صاحبَها رابطُ «أرسل موقعي»، ونقطتُه
  // تصير نقطةَ الاستلام قبل الإرسال أو بعده. مرآةُ `CustomScreen.kt`.
  const [holderName, setHolderName] = useState("");
  const [holderPhone, setHolderPhone] = useState("");
  const [ask, setAsk] = useState<AskState>("idle");
  const [askReq, setAskReq] = useState<AskReq | null>(null);
  const [askPhone, setAskPhone] = useState("");
  const [askError, setAskError] = useState("");
  const [pickAddr, setPickAddr] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [code, setCode] = useState("");

  // **ويُسأل كلَّ أربع ثوانٍ** ما دام ينتظر — **ويتوقّف بعد نصف ساعة** (الربطُ في الخادم باقٍ).
  useEffect(() => {
    if (ask !== "waiting" || !askReq) return;
    const id = askReq.id;
    const started = Date.now();
    const timer = setInterval(async () => {
      if (Date.now() - started > 30 * 60 * 1000) {
        clearInterval(timer);
        return;
      }
      try {
        const st = await api<{ status: string }>(`/api/v1/my/location-requests/${id}`);
        if (st.status === "received") setAsk("received");
        else if (st.status === "expired") setAsk("expired");
      } catch {
        /* **وتعثّرُ سؤالٍ واحدٍ لا يوقف الانتظار.** */
      }
    }, 4000);
    return () => clearInterval(timer);
  }, [ask, askReq]);

  if (loading) return <LoadingState variant="list" rows={3} />;
  if (!user) return <NeedLogin />;

  const cd = platform?.custom_delivery;
  const adminFee = cd?.source === "admin_defined";
  const fee = cd ? customFee(cd, mode) : 0;
  // **والتوصيلةُ المجانيّةُ تُقال قبل الطلب** (قرارُ المالك ٢٠٢٦-١٠-٢٦: «الزبون لسا يشوف السعر ١٠٠ —
  // رح يفكّر إنّا خدعناه»). والإعفاءُ يقع في الخادم عند تثبيت الأجرة، **بسقف العرض إن كان.**
  const freeOn = freeCustomOn(freeDelivery, mode);
  const waive = freeOn ? Math.max(0, Math.min(fee, freeDelivery?.max_fee ?? fee)) : 0;
  const freeUntil = freeOn ? freeUntilText(freeDelivery?.ends_at) : "";
  const closed = platform ? !platform.launch.customer_custom_orders : false;
  // **ونصُّ المالك يغلب** (مرآةُ `CustomScreen.kt` ٢٠٢٦-١٠-٢٧) — من كتب في اللوحة سببَه قُرئ كما كتبه.
  const closedText = closed ? platform?.launch.notice.trim() || K.closed : "";

  // **والأمانةُ بلا صاحبها ورقمه لا تُرسَل** — السائقُ يتّصل به. **والموقعُ ليس شرطاً.**
  const phoneDigits = holderPhone.replace(/\D/g, "").length;
  const missing =
    mode === "amanah"
      ? [!holderName.trim() ? K.holderName : "", phoneDigits < 9 ? K.holderPhone : ""].filter(Boolean)
      : [];
  const needHolder = missing.length > 0;

  const resetAsk = () => {
    setAsk("idle");
    setAskReq(null);
    setAskPhone("");
    setAskError("");
  };

  const askLocation = async () => {
    if (ask === "sending") return;
    setAskError("");
    if (phoneDigits < 9) {
      setAskError(K.askNeedPhone);
      return;
    }
    setAsk("sending");
    try {
      const r = await api<AskReq>("/api/v1/my/location-requests", {
        method: "POST",
        body: JSON.stringify({ phone: holderPhone.trim() }),
      });
      setAskReq(r);
      setAskPhone(holderPhone.trim());
      setAsk("waiting");
    } catch (e) {
      setAsk("idle");
      setAskError(errorText(e));
    }
  };

  const send = async () => {
    if (!address || busy || needHolder) return;
    setBusy(true);
    setError("");
    setCode("");
    try {
      const ref = await api<{ id: string; number: number }>("/api/v1/orders/custom", {
        method: "POST",
        headers: { "Idempotency-Key": attemptKey(SLOT) },
        body: JSON.stringify({
          request: request.trim(),
          address_text: address.address_text,
          lat: address.lat,
          lng: address.lng,
          notes: notes.trim(),
          payment_method: "cash",
          custom_mode: mode,
          ...(mode === "amanah"
            ? {
                holder_name: holderName.trim(),
                holder_phone: holderPhone.trim(),
                ...(askReq && (ask === "waiting" || ask === "received") ? { location_request_id: askReq.id } : {}),
              }
            : {}),
        }),
      });
      clearAttempt(SLOT);
      setRequest("");
      setNotes("");
      setMode("purchase");
      setHolderName("");
      setHolderPhone("");
      resetAsk();
      toast(fill(K.done, { n: fmtRef(ref.number) }));
      router.push(ref.id ? `/orders/${ref.id}` : "/orders");
    } catch (err) {
      if (isDecided(err)) clearAttempt(SLOT);
      setError(errorText(err));
      const c = (err as { body?: { code?: string } })?.body?.code ?? "";
      setCode(c);
      // **طلبُ موقعٍ رُبط بأمانةٍ قبلها** — يُطلب من جديد.
      if (err instanceof ApiError && c === "location_request_invalid") resetAsk();
    } finally {
      setBusy(false);
    }
  };

  const modes: { id: Mode; label: string }[] = [
    { id: "purchase", label: K.purchase },
    { id: "amanah", label: K.amanah },
  ];

  return (
    <div className="mx-auto flex w-full max-w-2xl flex-col gap-4">
      {/* **«اطلب أي شيء…» هي العنوان، ولا «طلب خاص» فوقها** (قرارُ المالك ٢٠٢٦-١٠-٣٠). */}
      <h1 className="heading-page">{K.subtitle}</h1>

      {closed && <Alert tone="warning">{closedText}</Alert>}

      <section className="surface flex flex-col gap-3 p-4">
        {/* **ولا «نوع الطلب» ولا سطرَ شرحٍ** — الخياران يشرحان نفسَيهما (قرارُ المالك ٢٠٢٦-١٠-٣٠). */}
        <div className="grid grid-cols-2 gap-2" role="radiogroup" aria-label={K.modeTitle}>
          {modes.map((it) => (
            <Button
              key={it.id}
              role="radio"
              aria-checked={mode === it.id}
              variant={mode === it.id ? "primary" : "secondary"}
              className="w-full py-3"
              onClick={() => setMode(it.id)}
            >
              {it.label}
            </Button>
          ))}
        </div>

        {/* **والمثالُ نصٌّ رماديٌّ داخلَ الحقل يزول بالكتابة** (قرارُ المالك ٢٠٢٦-١٠-٣٠). */}
        <Textarea
          id="custom-request"
          label={K.what}
          placeholder={K.whatHint}
          value={request}
          onChange={(e) => setRequest(e.target.value.slice(0, MAX_REQUEST))}
          rows={4}
          maxLength={MAX_REQUEST}
        />
      </section>

      <AddressChip onOpen={() => setPickAddr(true)} />

      {/* **صاحبُ الأمانة ورقمُه، و«اطلب الموقع منه»** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — كـ«إحضار» المتجر. */}
      {mode === "amanah" && (
        <section className="surface flex flex-col gap-3 p-4">
          <Input
            id="holder-name"
            label={K.holderName}
            value={holderName}
            maxLength={MAX_NAME}
            disabled={busy}
            onChange={(e) => setHolderName(e.target.value.slice(0, MAX_NAME))}
          />
          <Input
            id="holder-phone"
            label={K.holderPhone}
            value={holderPhone}
            dir="ltr"
            inputMode="tel"
            autoComplete="off"
            disabled={busy}
            onChange={(e) => {
              const v = e.target.value.replace(/[^\d+ ]/g, "").slice(0, 20);
              setHolderPhone(v);
              // **ورقمٌ تبدّل يُسقط طلبَ الموقع القديم** — كان لرقمٍ آخر.
              if (askReq && v.trim() !== askPhone) resetAsk();
            }}
          />
          {ask !== "received" && (
            <Button
              type="button"
              variant="secondary"
              className="flex w-full items-center justify-center gap-2"
              disabled={busy || ask === "sending" || ask === "waiting"}
              onClick={() => void askLocation()}
            >
              <IconLocation size={18} />
              <span>{ask === "sending" ? K.askSending : K.askLoc}</span>
            </Button>
          )}
          {ask === "idle" && !askError && <p className="text-xs text-ink-muted">{K.askHint}</p>}
          {ask === "waiting" && askReq && (
            <div className="flex flex-col gap-2 rounded-control border border-line bg-field p-3">
              <p className="text-xs text-ink-muted">{askReq.whatsapp_sent ? K.askWaiting : K.askNotSent}</p>
              {!askReq.whatsapp_sent && (
                <a className="text-sm font-bold text-accent-text" href={askReq.wa_url} target="_blank" rel="noreferrer">
                  {K.askShare}
                </a>
              )}
            </div>
          )}
          {ask === "received" && (
            <div className="rounded-control border border-success-edge bg-success-tint p-3">
              <p className="text-sm font-bold text-success">{K.askReceived}</p>
            </div>
          )}
          {ask === "expired" && <p className="text-xs text-danger">{K.askExpired}</p>}
          {askError && <p className="text-xs text-danger">{askError}</p>}
        </section>
      )}

      <section className="surface flex flex-col gap-3 p-4">
        <Textarea
          id="custom-notes"
          label={K.notes}
          value={notes}
          onChange={(e) => setNotes(e.target.value.slice(0, MAX_NOTES))}
          rows={2}
          maxLength={MAX_NOTES}
        />
        <div className="flex flex-col gap-1 border-t border-line-soft pt-3">
          {adminFee ? (
            <>
              {waive > 0 ? (
                <>
                  <p className="text-sm">
                    <s className="text-ink-muted">{fill(K.feeValue, { amount: fmtMoney(fee) })}</s>
                  </p>
                  {/* **والإعفاءُ الجزئيُّ لا يُقال «مجانية»** — سقفُ العرض أقلُّ من الأجرة، فيُقال ما بقي. */}
                  <p className="text-sm font-bold text-success">
                    {waive >= fee ? m.shop.free.customFree : fill(K.feeValue, { amount: fmtMoney(fee - waive) })}
                  </p>
                  {freeUntil && <p className="text-xs text-ink-muted">{freeUntil}</p>}
                </>
              ) : (
                <p className="text-sm font-bold">{fill(K.feeValue, { amount: fmtMoney(fee) })}</p>
              )}
              {/* **ولا سطرَ تحت الأجرة** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — «أجرة التوصيل: X» تكفي. مرآةُ `CustomScreen.kt`. */}
            </>
          ) : (
            <>
              <p className="text-xs text-ink-muted">{K.feeOnDeal}</p>
              {/* **والعرضُ يُقال ولو كانت الأجرةُ عند السائق** — بصيغةٍ صادقة: يُحجز وقتَ تحديد السعر. */}
              {freeOn && <p className="text-xs font-bold text-success">{m.shop.free.customPending}</p>}
            </>
          )}
        </div>
      </section>

      {!ordering.open && <Alert tone="warning">{ordering.text}</Alert>}
      {gate.kind === "blocked" && <AvailabilityNotice av={gate.av} address={address} />}

      <Button
        className="w-full py-3 text-base"
        onClick={send}
        disabled={busy || !request.trim() || !address || needHolder || !ordering.open || closed || gate.kind === "blocked"}
      >
        {busy ? K.sending : K.send}
      </Button>
      {!address && <p className="text-xs text-ink-muted">{K.needAddress}</p>}
      {needHolder && <p className="text-xs text-ink-muted">{fill(K.missing, { list: missing.join(K.listSep) })}</p>}
      {error && (
        <Alert>
          {error}
          {code === "whatsapp_required" && (
            <Link href="/account#whatsapp" className="mt-1 block font-bold underline">
              {K.verifyNow}
            </Link>
          )}
        </Alert>
      )}
      <AddressSheet open={pickAddr} onClose={() => setPickAddr(false)} />
    </div>
  );
}
