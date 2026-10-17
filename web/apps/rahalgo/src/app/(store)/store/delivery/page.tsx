"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **‏/store/delivery — «لدي توصيلة»**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `DeliveryScreen.kt` و`DeliveryViewModel` في تطبيق المتجر.)
 *
 * **زرّان كبيران «توصيل / إحضار» ولا شيءَ مختارٌ سلفاً** — ولا يظهر
 * النموذجُ قبل الاختيار: الألفاظُ نفسُها تتبدّل بالنوع («اسم المستلم» أم
 * «اسم الشخص بمكان الاستلام»)، **ومن ملأ النموذجَ ثمّ بدّل النوع قرأ
 * ألفاظاً غيرَ التي كتب لها.**
 *
 * **والموقعُ رابطٌ من واتساب** — يُقرأ في المحرّك (`resolve-location`)
 * فتُحسب الأجرةُ بالمسافة. **وبلا رابطٍ تُحسب من منطقة المتجر.**
 *
 * **ومن يدفع الأجرة لا يُختار سلفاً** — و«من محفظتي» لا يُختار إن لم يكفِ
 * الرصيد (`merchant_can_pay` من المحرّك).
 *
 * **والإرسالُ مرّةً واحدةً مهما أُعيد** — مفتاحُ المحاولة (`Idempotency-Key`).
 */

import { useCallback, useEffect, useRef, useState } from "react";
import { getMessages, defaultLocale, errorText, fmtNum } from "@rahalgo/i18n";
import { Alert, Button, Checkbox, IconMoto, Input, LoadingState, Textarea } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useStore } from "@/lib/store/context";
import { useStoreDeliveries } from "@/lib/store/deliveries";
import type { DeliveryQuote } from "@/lib/store/types";
import { attemptKey, clearAttempt, isDecided } from "@/lib/customer/storage";
import { fill } from "@/lib/customer/texts";
import { DeliveryRow } from "@/components/store/DeliveryParts";

const m = getMessages(defaultLocale);
const D = m.store.delivery;
const SLOT = "merchant_delivery";

type Mode = "" | "send" | "fetch";
type Payer = "" | "merchant" | "merchant_cash" | "recipient";
type LinkState = "idle" | "checking" | "found" | "notFound";

function ModeButton({ on, title, sub, onClick, disabled }: { on: boolean; title: string; sub: string; onClick: () => void; disabled: boolean }) {
  return (
    <button
      type="button"
      onClick={onClick}
      disabled={disabled}
      aria-pressed={on}
      data-testid="delivery-mode"
      className={`flex flex-1 flex-col items-center gap-1 rounded-card border p-4 text-center transition-colors ${
        on ? "border-2 border-accent bg-accent-tint" : "border-line bg-surface hover:bg-row-hover"
      }`}
    >
      <span className="heading-card">{title}</span>
      <span className="text-xs text-ink-muted">{sub}</span>
    </button>
  );
}

function PayerOption({
  on,
  title,
  sub,
  disabled,
  onPick,
}: {
  on: boolean;
  title: string;
  sub: string;
  disabled: boolean;
  onPick: () => void;
}) {
  return (
    <button
      type="button"
      role="radio"
      aria-checked={on}
      disabled={disabled}
      onClick={onPick}
      className={`flex w-full items-center gap-3 rounded-control border p-3 text-start transition-colors disabled:opacity-60 ${
        on ? "border-2 border-accent bg-accent-tint" : "border-line bg-surface hover:bg-row-hover"
      }`}
    >
      <span
        className={`flex h-[18px] w-[18px] shrink-0 items-center justify-center rounded-full border ${on ? "border-accent" : "border-line"}`}
      >
        {on && <span className="h-2.5 w-2.5 rounded-full bg-accent" />}
      </span>
      <span className="min-w-0">
        <span className="block text-sm font-bold">{title}</span>
        <span className="block text-xs text-ink-muted">{sub}</span>
      </span>
    </button>
  );
}

export default function DeliveryPage() {
  const { store } = useStore();
  const storeId = store?.id ?? "";
  const { list, error: listError, reload } = useStoreDeliveries();

  const [mode, setMode] = useState<Mode>("");
  const [name, setName] = useState("");
  const [phone, setPhone] = useState("");
  const [address, setAddress] = useState("");
  const [link, setLink] = useState("");
  const [linkState, setLinkState] = useState<LinkState>("idle");
  const [point, setPoint] = useState<{ lat: number; lng: number } | null>(null);
  const [parcel, setParcel] = useState("");
  const [driverNote, setDriverNote] = useState("");
  const [collectOn, setCollectOn] = useState(false);
  const [collectAmount, setCollectAmount] = useState("");
  const [payer, setPayer] = useState<Payer>("");
  const [quote, setQuote] = useState<DeliveryQuote | null>(null);
  const [quoteError, setQuoteError] = useState("");
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [sent, setSent] = useState(false);
  const [cancelling, setCancelling] = useState<string | null>(null);
  const [cancelError, setCancelError] = useState("");
  const [shown, setShown] = useState(5);
  const linkTimer = useRef<ReturnType<typeof setTimeout> | null>(null);
  const linkSeq = useRef(0);

  const isFetch = mode === "fetch";
  const touched = () => {
    setSent(false);
    setError("");
  };

  const fetchQuote = useCallback(
    async (p: { lat: number; lng: number } | null, md: Mode) => {
      setQuote(null);
      setQuoteError("");
      if (!storeId) return;
      try {
        const q = await api<DeliveryQuote>(
          `/api/v1/merchant/stores/${storeId}/delivery-quote?mode=${md || "send"}` +
            (p ? `&lat=${p.lat}&lng=${p.lng}` : ""),
        );
        setQuote(q);
        setPayer((cur) => (cur === "merchant" && !q.merchant_can_pay ? "" : cur));
      } catch (e) {
        setQuoteError(errorText(e));
      }
    },
    [storeId],
  );

  useEffect(() => {
    void fetchQuote(null, "");
  }, [fetchQuote]);

  function pickMode(md: Mode) {
    if (md === mode) return;
    setMode(md);
    touched();
    void fetchQuote(point, md);
  }

  function onLink(text: string) {
    const v = text.slice(0, 2000);
    setLink(v);
    touched();
    if (linkTimer.current) clearTimeout(linkTimer.current);
    const seq = ++linkSeq.current;
    if (!v.includes("http")) {
      setLinkState("idle");
      if (point) {
        setPoint(null);
        void fetchQuote(null, mode);
      }
      return;
    }
    setLinkState("checking");
    linkTimer.current = setTimeout(async () => {
      try {
        const r = await api<{ found: boolean; lat: number; lng: number }>(
          `/api/v1/merchant/stores/${storeId}/resolve-location`,
          { method: "POST", body: JSON.stringify({ url: v }) },
        );
        if (seq !== linkSeq.current) return;
        if (r.found) {
          const p = { lat: r.lat, lng: r.lng };
          setPoint(p);
          setLinkState("found");
          void fetchQuote(p, mode);
        } else {
          setPoint(null);
          setLinkState("notFound");
          void fetchQuote(null, mode);
        }
      } catch {
        if (seq !== linkSeq.current) return;
        setPoint(null);
        setLinkState("notFound");
      }
    }, 400);
  }

  const collectValue = collectOn ? Number(collectAmount.replace(/\D/g, "")) || null : null;
  const missing = [
    !name.trim() && (isFetch ? D.nameFetch : D.name),
    !phone.trim() && (isFetch ? D.phoneFetch : D.phone),
    !address.trim() && (isFetch ? D.addressFetch : D.address),
    !payer && D.whoPays,
    collectOn && collectValue == null && (isFetch ? D.collectMissingFetch : D.collectMissing),
  ].filter(Boolean) as string[];
  const blocked = !!quoteError || (payer === "merchant" && quote != null && !quote.merchant_can_pay);

  async function send() {
    if (busy || !storeId) return;
    setBusy(true);
    setError("");
    setSent(false);
    try {
      await api(`/api/v1/merchant/stores/${storeId}/deliveries`, {
        method: "POST",
        headers: { "Idempotency-Key": attemptKey(SLOT) },
        body: JSON.stringify({
          recipient_name: name.trim(),
          recipient_phone: phone.trim(),
          address_text: address.trim(),
          lat: point?.lat ?? null,
          lng: point?.lng ?? null,
          parcel_note: parcel.trim(),
          driver_note: driverNote.trim(),
          fee_payer: payer,
          delivery_mode: mode,
          collect_amount: collectValue,
        }),
      });
      clearAttempt(SLOT);
      setSent(true);
      setName("");
      setPhone("");
      setAddress("");
      setParcel("");
      setDriverNote("");
      setPayer("");
      setMode("");
      setCollectOn(false);
      setCollectAmount("");
      setLink("");
      setLinkState("idle");
      setPoint(null);
      void fetchQuote(null, "");
      void reload();
    } catch (e) {
      if (isDecided(e)) clearAttempt(SLOT);
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  async function cancel(id: string) {
    if (cancelling) return;
    setCancelling(id);
    setCancelError("");
    try {
      await api(`/api/v1/merchant/deliveries/${id}/cancel`, { method: "POST", body: "{}" });
    } catch (e) {
      setCancelError(errorText(e));
    } finally {
      setCancelling(null);
      void reload();
    }
  }

  const linkNote =
    linkState === "checking"
      ? { text: D.linkChecking, cls: "text-ink-muted" }
      : linkState === "found"
        ? { text: D.linkFound, cls: "text-success" }
        : linkState === "notFound"
          ? { text: isFetch ? D.linkNotFoundFetch : D.linkNotFound, cls: "text-danger" }
          : { text: isFetch ? D.linkHowFetch : D.linkHow, cls: "text-ink-muted" };

  const payers: { key: Payer; title: string; sub: string }[] = [
    { key: "merchant", title: D.payMe, sub: D.payMeSub },
    { key: "merchant_cash", title: D.payMeCash, sub: isFetch ? D.payMeCashSubFetch : D.payMeCashSub },
    {
      key: "recipient",
      title: isFetch ? D.payRecipientFetch : D.payRecipient,
      sub: isFetch ? D.payRecipientSubFetch : D.payRecipientSub,
    },
  ];

  const feeSub =
    payer === "merchant"
      ? fill(D.feeFromWallet, { balance: fmtNum(quote?.available_balance ?? 0) })
      : payer === "merchant_cash"
        ? isFetch
          ? D.feeMeCashFetch
          : D.feeMeCash
        : payer === "recipient"
          ? isFetch
            ? D.feeCashFetch
            : D.feeCash
          : D.feeChoose;

  const rows = list;

  return (
    <div className="flex flex-col gap-3">
      <header>
        <h2 className="heading-page">{D.title}</h2>
        <p className="text-sm text-ink-muted">{D.hint}</p>
      </header>

      {quoteError && (
        <div className="flex items-center gap-3 rounded-card border border-danger-edge bg-danger-tint p-3">
          <IconMoto size={22} className="shrink-0 text-danger" />
          <div className="min-w-0 flex-1">
            <p className="text-sm font-bold text-danger">{D.blockedTitle}</p>
            <p className="text-xs">{quoteError}</p>
          </div>
          <Button variant="secondary" onClick={() => void fetchQuote(point, mode)}>
            {m.common.retry}
          </Button>
        </div>
      )}

      <div className="flex gap-3">
        <ModeButton on={mode === "send"} title={D.modeSend} sub={D.modeSendSub} disabled={busy} onClick={() => pickMode("send")} />
        <ModeButton on={mode === "fetch"} title={D.modeFetch} sub={D.modeFetchSub} disabled={busy} onClick={() => pickMode("fetch")} />
      </div>

      {sent && <Alert tone="success">{D.sent}</Alert>}

      {!mode ? (
        <p className="text-center text-sm text-ink-muted">{D.modePick}</p>
      ) : (
        <form
          className="surface flex flex-col gap-3 rounded-card p-4"
          onSubmit={(e) => {
            e.preventDefault();
            void send();
          }}
        >
          <Input
            label={isFetch ? D.nameFetch : D.name}
            value={name}
            disabled={busy}
            autoComplete="off"
            onChange={(e) => {
              setName(e.target.value);
              touched();
            }}
          />
          <Input
            label={isFetch ? D.phoneFetch : D.phone}
            value={phone}
            disabled={busy}
            type="tel"
            inputMode="tel"
            dir="ltr"
            autoComplete="off"
            onChange={(e) => {
              setPhone(e.target.value.replace(/[^\d+]/g, "").slice(0, 16));
              touched();
            }}
          />
          <Textarea
            id="delivery-address"
            label={isFetch ? D.addressFetch : D.address}
            value={address}
            disabled={busy}
            rows={2}
            onChange={(e) => {
              setAddress(e.target.value);
              touched();
            }}
          />
          <div className="flex flex-col gap-1">
            <Input
              label={isFetch ? D.linkFetch : D.link}
              placeholder={isFetch ? D.linkHintFetch : D.linkHint}
              value={link}
              disabled={busy}
              dir="ltr"
              inputMode="url"
              autoComplete="off"
              onChange={(e) => onLink(e.target.value)}
            />
            <p className={`text-xs ${linkNote.cls}`}>{linkNote.text}</p>
          </div>
          <Textarea
            id="delivery-parcel"
            maxLength={200}
            label={isFetch ? D.parcelFetch : D.parcel}
            placeholder={isFetch ? D.parcelHintFetch : D.parcelHint}
            value={parcel}
            disabled={busy}
            rows={2}
            onChange={(e) => {
              setParcel(e.target.value.slice(0, 200));
              touched();
            }}
          />
          <Textarea
            id="delivery-driver-note"
            maxLength={200}
            label={D.driverNote}
            value={driverNote}
            disabled={busy}
            rows={2}
            onChange={(e) => {
              setDriverNote(e.target.value.slice(0, 200));
              touched();
            }}
          />
          <Checkbox
            id="delivery-collect"
            label={isFetch ? D.collectQFetch : D.collectQ}
            checked={collectOn}
            disabled={busy}
            onChange={(e) => {
              setCollectOn(e.target.checked);
              if (!e.target.checked) setCollectAmount("");
              touched();
            }}
          />
          {collectOn && (
            <Input
              label={isFetch ? D.collectAmountFetch : D.collectAmount}
              value={collectAmount}
              disabled={busy}
              inputMode="numeric"
              dir="ltr"
              onChange={(e) => {
                setCollectAmount(e.target.value.replace(/\D/g, "").slice(0, 12));
                touched();
              }}
            />
          )}

          <div className="flex flex-col gap-2" role="radiogroup" aria-label={D.whoPays}>
            <p className="text-sm font-bold">{D.whoPaysQ}</p>
            {payers.map((p) => {
              const short = p.key === "merchant" && quote?.merchant_can_pay === false;
              return (
                <PayerOption
                  key={p.key}
                  on={payer === p.key}
                  title={p.title}
                  sub={short ? D.walletShort : p.sub}
                  disabled={busy || short}
                  onPick={() => {
                    setPayer(p.key);
                    touched();
                  }}
                />
              );
            })}
          </div>

          {!quoteError && quote && (
            <div className="rounded-control border border-success-edge bg-success-tint p-3">
              <p className="font-bold text-success">{fill(D.fee, { fee: fmtNum(quote.fee) })}</p>
              <p className="text-xs">{feeSub}</p>
              {quote.fee_basis === "no_point" ? (
                <p className="text-xs text-ink-muted">{isFetch ? D.feeNoPointFetch : D.feeNoPoint}</p>
              ) : quote.fee_basis === "distance" && quote.distance_m > 0 ? (
                <p className="text-xs text-ink-muted">{fill(D.feeDistance, { km: (quote.distance_m / 1000).toFixed(1) })}</p>
              ) : null}
            </div>
          )}
          {payer === "merchant" && quote && !quote.merchant_can_pay && <Alert>{D.cannotPay}</Alert>}
          {missing.length > 0 && (
            <p className="text-xs text-ink-muted">{fill(D.missing, { list: missing.join(m.common.listSep) })}</p>
          )}
          {error && <Alert>{error}</Alert>}
          <Button type="submit" size="lg" className="w-full" disabled={busy || missing.length > 0 || blocked || !storeId}>
            {busy ? D.sending : D.send}
          </Button>
        </form>
      )}

      <section className="flex flex-col gap-2" aria-label={D.mine}>
        <h3 className="heading-card">{D.mine}</h3>
        {(listError || cancelError) && <Alert>{cancelError || errorText(listError)}</Alert>}
        {rows == null && !listError && <LoadingState variant="inline" label={D.loading} />}
        {rows != null && rows.length === 0 && <p className="text-sm text-ink-muted">{D.none}</p>}
        {rows?.slice(0, shown).map((d) => (
          <DeliveryRow key={d.id} d={d} cancelling={cancelling === d.id} onCancel={() => void cancel(d.id)} />
        ))}
        {rows != null && rows.length > shown && (
          <Button variant="secondary" onClick={() => setShown((n) => n + 10)}>
            {D.more}
          </Button>
        )}
      </section>
    </div>
  );
}
