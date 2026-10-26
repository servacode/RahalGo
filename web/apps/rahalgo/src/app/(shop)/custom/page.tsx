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
import { useState } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale, errorText, fmtMoney, fmtRef } from "@rahalgo/i18n";
import { Alert, Button, LoadingState, Textarea } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { useCustomer } from "@/lib/customer/context";
import { attemptKey, clearAttempt, isDecided } from "@/lib/customer/storage";
import { customFee, fill } from "@/lib/customer/texts";
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
  const [pickAddr, setPickAddr] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [code, setCode] = useState("");

  if (loading) return <LoadingState variant="list" rows={3} />;
  if (!user) return <NeedLogin />;

  const cd = platform?.custom_delivery;
  const adminFee = cd?.source === "admin_defined";
  const fee = cd ? customFee(cd, mode) : 0;
  // **والتوصيلةُ المجانيّةُ تُقال قبل الطلب** (قرارُ المالك ٢٠٢٦-١٠-٢٦: «الزبون لسا يشوف السعر ١٠٠ —
  // رح يفكّر إنّا خدعناه»). والإعفاءُ يقع في الخادم عند تثبيت الأجرة، **بسقف العرض إن كان.**
  const freeOn = !!freeDelivery && (mode === "amanah" ? freeDelivery.amanah : freeDelivery.purchase);
  const waive = freeOn ? Math.min(fee, freeDelivery?.max_fee ?? fee) : 0;
  const closed = platform ? !platform.launch.customer_custom_orders : false;

  const send = async () => {
    if (!address || busy) return;
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
        }),
      });
      clearAttempt(SLOT);
      setRequest("");
      setNotes("");
      setMode("purchase");
      toast(fill(K.done, { n: fmtRef(ref.number) }));
      router.push(ref.id ? `/orders/${ref.id}` : "/orders");
    } catch (err) {
      if (isDecided(err)) clearAttempt(SLOT);
      setError(errorText(err));
      setCode((err as { body?: { code?: string } })?.body?.code ?? "");
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
      <div>
        <h1 className="heading-page">{K.title}</h1>
        <p className="text-sm text-ink-muted">{K.subtitle}</p>
      </div>

      {closed && <Alert tone="warning">{K.closed}</Alert>}

      <section className="surface flex flex-col gap-3 p-4">
        <p className="text-sm font-bold">{K.modeTitle}</p>
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
        <p className="text-xs text-ink-muted">{mode === "amanah" ? K.amanahHint : K.purchaseHint}</p>

        <div>
          <Textarea
            id="custom-request"
            label={K.what}
            value={request}
            onChange={(e) => setRequest(e.target.value.slice(0, MAX_REQUEST))}
            rows={4}
            maxLength={MAX_REQUEST}
          />
          <p className="mt-1 text-xs text-ink-muted">{K.whatHint}</p>
        </div>
      </section>

      <AddressChip onOpen={() => setPickAddr(true)} />

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
                  <p className="text-sm font-bold text-success">{m.shop.free.customFree}</p>
                </>
              ) : (
                <p className="text-sm font-bold">{fill(K.feeValue, { amount: fmtMoney(fee) })}</p>
              )}
              <p className="text-xs text-ink-muted">
                {mode === "amanah" ? fill(K.amanahTotal, { amount: fmtMoney(fee - waive) }) : K.totalPending}
              </p>
            </>
          ) : (
            <p className="text-xs text-ink-muted">{K.feeOnDeal}</p>
          )}
          <p className="text-xs text-ink-muted">{mode === "amanah" && adminFee ? K.amanahPay : K.payAfterCost}</p>
        </div>
      </section>

      {!ordering.open && <Alert tone="warning">{ordering.text}</Alert>}
      {gate.kind === "blocked" && <AvailabilityNotice av={gate.av} address={address} />}

      <Button
        className="w-full py-3 text-base"
        onClick={send}
        disabled={busy || !request.trim() || !address || !ordering.open || closed || gate.kind === "blocked"}
      >
        {busy ? K.sending : K.send}
      </Button>
      {!address && <p className="text-xs text-ink-muted">{K.needAddress}</p>}
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
