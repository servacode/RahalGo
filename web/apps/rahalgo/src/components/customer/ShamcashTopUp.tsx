"use client";

/**
 * **شحنُ المحفظة بشام كاش** — زرٌّ يفتح لوحةً بثلاث خطواتٍ مرقَّمة (حوّل ← اكتب رقم العملية ← أرسل)،
 * وقائمةُ طلبات الزبون وحالاتِها. **والطلبُ يُنشأ مرّةً واحدةً مهما أُعيدت المحاولة** (`Idempotency-Key`).
 *
 * **والزرُّ والقائمةُ يُسلَّمان لمن يرسم الصفحة** (`children`) كي يقع الزرُّ بجانب «كشف حساب» في صفٍّ
 * واحد، وتقع القائمةُ تحت الصفّ — والمنطقُ كلُّه (الجلبُ والإرسالُ والمفتاح) هنا لم يتغيّر.
 */

import { useRef, useState, type ReactNode } from "react";
import { getMessages, defaultLocale, errorText, fmtMoney, fmtNum } from "@rahalgo/i18n";
import { Alert, Badge, Button, IconCheck, IconClock, IconClose, IconPhoto, IconWallet, Input, Modal } from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";
import { attemptKey, clearAttempt, isDecided } from "@/lib/customer/storage";
import { fill } from "@/lib/customer/texts";
import { useShamcash, type ShamcashInfo, type ShamcashRequest } from "@/lib/customer/shamcash";
import { ShamcashAccountCard } from "@/components/customer/ShamcashAccountCard";
import { shrink, type UploadedPhoto } from "@/components/CustomPhotos";
import { OUTLINE_CARD } from "@/components/customer/kit";

const m = getMessages(defaultLocale);
const S = m.shop.shamcash;
const SLOT = "shamcash_topup";

/** **رأسُ خطوةٍ مرقَّمة** — دائرةٌ بالرقم وعنوانٌ؛ ومحتواها تحتها. */
function Step({ n, title, children }: { n: number; title: string; children: ReactNode }) {
  return (
    <section className="flex flex-col gap-3">
      <div className="flex items-center gap-3">
        <span
          aria-hidden
          className="grid size-9 shrink-0 place-items-center rounded-full bg-primary text-sm font-bold text-on-bright"
        >
          {fmtNum(n)}
        </span>
        <h3 className="text-card-title">{title}</h3>
      </div>
      {children}
    </section>
  );
}

/** **رقاقةُ حالة الطلب** — لونٌ وأيقونةٌ ونصّ، فلا يُترك اللونُ وحدَه. */
function StatusBadge({ r }: { r: ShamcashRequest }) {
  if (r.status === "approved") {
    return (
      <Badge variant="success" className="gap-1">
        <IconCheck size={12} />
        {S.statusApproved}
      </Badge>
    );
  }
  if (r.status === "rejected") {
    return (
      <Badge variant="danger" className="gap-1">
        <IconClose size={12} />
        {S.statusRejectedNoReason}
      </Badge>
    );
  }
  return (
    <Badge variant="warning" className="gap-1">
      <IconClock size={12} />
      {S.statusPending}
    </Badge>
  );
}

function TopUpForm({ info, onSent }: { info: ShamcashInfo; onSent: () => void }) {
  const [amount, setAmount] = useState("");
  const [txn, setTxn] = useState("");
  const [photo, setPhoto] = useState<UploadedPhoto | null>(null);
  const [upBusy, setUpBusy] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [done, setDone] = useState(false);
  const ref = useRef<HTMLInputElement>(null);

  const value = /^\d+$/.test(amount.trim()) ? Number(amount.trim()) : 0;
  const amountOk = value > 0 && value >= info.min_topup;
  const canSend = amountOk && txn.trim() !== "" && !busy && !upBusy;

  const pick = async (e: React.ChangeEvent<HTMLInputElement>) => {
    const file = e.target.files?.[0];
    if (ref.current) ref.current.value = "";
    if (!file) return;
    setUpBusy(true);
    setError("");
    try {
      const body = new FormData();
      body.append("file", await shrink(file), "receipt.jpg");
      setPhoto(await api<UploadedPhoto>("/api/v1/my/shamcash/receipt", { method: "POST", body }));
    } catch (err) {
      setError(errorText(err) || S.receiptFailed);
    } finally {
      setUpBusy(false);
    }
  };

  const send = async () => {
    if (!canSend) return;
    setBusy(true);
    setError("");
    try {
      await api("/api/v1/my/shamcash/topups", {
        method: "POST",
        headers: { "Idempotency-Key": attemptKey(SLOT) },
        body: JSON.stringify({ amount: value, txn_ref: txn.trim(), ...(photo ? { receipt_id: photo.id } : {}) }),
      });
      clearAttempt(SLOT);
      setDone(true);
      setAmount("");
      setTxn("");
      setPhoto(null);
      onSent();
    } catch (err) {
      if (isDecided(err)) clearAttempt(SLOT);
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  };

  return (
    <div className="flex flex-col gap-6">
      <Step n={2} title={S.step2}>
        <div className={`${OUTLINE_CARD} flex flex-col gap-3`}>
          <Input
            id="sc-amount"
            label={S.amount}
            inputMode="numeric"
            dir="ltr"
            value={amount}
            onChange={(e) => setAmount(e.target.value.replace(/\D/g, ""))}
            placeholder={fill(S.amountHint, { min: fmtMoney(info.min_topup) })}
            error={amount !== "" && !amountOk ? fill(S.amountInvalid, { min: fmtMoney(info.min_topup) }) : undefined}
          />
          <Input id="sc-txn" label={S.txn} dir="ltr" value={txn} onChange={(e) => setTxn(e.target.value)} placeholder={S.txnHint} />
          <div className="flex flex-col gap-2">
            <p className="text-sm font-bold">{S.receipt}</p>
            <div className="flex flex-wrap items-center gap-2">
              {photo && <img src={mediaUrl(photo.thumb_url) ?? photo.thumb_url} alt="" className="h-16 w-16 rounded-control border border-line object-cover" />}
              <input ref={ref} type="file" accept="image/*" className="hidden" onChange={(e) => void pick(e)} />
              {photo ? (
                <Button type="button" variant="secondary" onClick={() => setPhoto(null)}>
                  {S.receiptRemove}
                </Button>
              ) : (
                <Button
                  type="button"
                  variant="secondary"
                  disabled={upBusy}
                  className="flex items-center gap-2"
                  onClick={() => ref.current?.click()}
                >
                  <IconPhoto size={16} />
                  {S.receiptAdd}
                </Button>
              )}
            </div>
          </div>
        </div>
      </Step>
      <Step n={3} title={S.step3}>
        <p className="text-sm text-ink-muted">{S.step3Hint}</p>
        {error && <Alert>{error}</Alert>}
        {done && <Alert tone="success">{S.sent}</Alert>}
        <Button type="button" size="lg" className="w-full" disabled={!canSend} onClick={() => void send()}>
          {S.send}
        </Button>
      </Step>
    </div>
  );
}

/**
 * **زرُّ الشحن + اللوحةُ + قائمةُ الطلبات** — لا يظهر شيءٌ حتّى يردّ الخادمُ ويكون الشحنُ مفعَّلاً
 * (والأبناءُ يُرسَمون دائماً، وزرُّهم وقائمتُهم `null` حتّى ذلك).
 */
export function ShamcashTopUp({ children }: { children: (parts: { action: ReactNode; requests: ReactNode }) => ReactNode }) {
  const { info, requests, reload } = useShamcash();
  const [open, setOpen] = useState(false);
  const enabled = !!info && info.topup_enabled;

  const action = enabled ? (
    <Button type="button" size="lg" className="flex min-h-12 flex-1 items-center justify-center gap-2" onClick={() => setOpen(true)}>
      <IconWallet size={16} />
      {S.topUpButton}
    </Button>
  ) : null;

  const list =
    enabled && requests.length > 0 ? (
      <div className={`${OUTLINE_CARD} flex flex-col gap-2`} data-testid="shamcash-requests">
        <p className="text-card-title">{S.listTitle}</p>
        <ul className="divide-y divide-line-soft">
          {requests.map((r) => (
            <li key={r.id} className="flex items-center justify-between gap-3 py-3">
              <span className="flex min-w-0 flex-col">
                <span className="font-bold tabular-nums">{fmtMoney(r.amount)}</span>
                <span dir="ltr" className="truncate text-start text-xs text-ink-muted">
                  {r.txn_ref}
                </span>
                {r.status === "rejected" && r.reject_reason && <span className="text-xs text-danger">{r.reject_reason}</span>}
              </span>
              <StatusBadge r={r} />
            </li>
          ))}
        </ul>
      </div>
    ) : null;

  return (
    <>
      {children({ action, requests: list })}
      {enabled && info && (
        <Modal open={open} onClose={() => setOpen(false)} title={S.panelTitle}>
          <div className="flex flex-col gap-6">
            <Step n={1} title={S.step1}>
              <p className="text-sm text-ink-muted">{S.panelHint}</p>
              <ShamcashAccountCard info={info} />
            </Step>
            <TopUpForm info={info} onSent={reload} />
          </div>
        </Modal>
      )}
    </>
  );
}
