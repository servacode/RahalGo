"use client";

/**
 * **شحنُ المحفظة بشام كاش** — زرٌّ كبيرٌ يفتح لوحةً فيها حسابُ المنصة ونموذجُ «أرسلت»، وتحتها قائمةُ
 * طلبات الزبون وحالاتِها. **والطلبُ يُنشأ مرّةً واحدةً مهما أُعيدت المحاولة** (`Idempotency-Key`).
 */

import { useRef, useState } from "react";
import { getMessages, defaultLocale, errorText, fmtMoney } from "@rahalgo/i18n";
import { Alert, Button, Input, Modal } from "@rahalgo/ui";
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

function statusText(r: ShamcashRequest): string {
  if (r.status === "approved") return S.statusApproved;
  if (r.status === "rejected") return r.reject_reason ? fill(S.statusRejected, { r: r.reject_reason }) : S.statusRejectedNoReason;
  return S.statusPending;
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
    <div className="flex flex-col gap-3">
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
            <Button type="button" variant="secondary" disabled={upBusy} onClick={() => ref.current?.click()}>
              {S.receiptAdd}
            </Button>
          )}
        </div>
      </div>
      {error && <Alert>{error}</Alert>}
      {done && <Alert tone="success">{S.sent}</Alert>}
      <Button type="button" size="lg" className="w-full" disabled={!canSend} onClick={() => void send()}>
        {S.send}
      </Button>
    </div>
  );
}

/** **زرُّ الشحن + اللوحةُ + قائمةُ الطلبات** — لا يظهر شيءٌ حتّى يردّ الخادمُ ويكون الشحنُ مفعَّلاً. */
export function ShamcashTopUp() {
  const { info, requests, reload } = useShamcash();
  const [open, setOpen] = useState(false);
  if (!info || !info.topup_enabled) return null;
  return (
    <>
      <Button type="button" size="lg" className="w-full" onClick={() => setOpen(true)}>
        {S.topUpButton}
      </Button>
      {requests.length > 0 && (
        <div className={`${OUTLINE_CARD} flex flex-col gap-2`}>
          <p className="text-card-title">{S.listTitle}</p>
          <ul className="divide-y divide-line-soft">
            {requests.map((r) => (
              <li key={r.id} className="flex items-center justify-between gap-3 py-2 text-sm">
                <span className="font-bold">{fmtMoney(r.amount)}</span>
                <span className={r.status === "rejected" ? "text-danger" : r.status === "approved" ? "text-success" : "text-ink-muted"}>
                  {statusText(r)}
                </span>
              </li>
            ))}
          </ul>
        </div>
      )}
      <Modal open={open} onClose={() => setOpen(false)} title={S.panelTitle}>
        <div className="flex flex-col gap-3">
          <p className="text-sm text-ink-muted">{S.panelHint}</p>
          <ShamcashAccountCard info={info} />
          <TopUpForm info={info} onSent={reload} />
        </div>
      </Modal>
    </>
  );
}
