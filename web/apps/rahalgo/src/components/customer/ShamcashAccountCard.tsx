"use client";

/**
 * **بطاقةُ حساب شام كاش** — الباركودُ واسمُ صاحب الحساب ورقمُه، وزرّا «نسخ رقم الحساب» و«حفظ صورة
 * الباركود» كبيران (الزبونُ يحوّل من تطبيق شام كاش فيحتاج الرقمَ أو الصورة بلمسة).
 * تُستعمل في لوحة الشحن وفي بطاقة «ادفع للكابتن بشام كاش».
 */

import { useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Button, IconCopy, IconDownload, IconWallet } from "@rahalgo/ui";
import { mediaUrl } from "@/lib/api";
import type { ShamcashInfo } from "@/lib/customer/shamcash";
import { useShopToast } from "@/components/customer/toast";

const m = getMessages(defaultLocale);
const S = m.shop.shamcash;

export function ShamcashAccountCard({ info, hero = false }: { info: ShamcashInfo; hero?: boolean }) {
  const toast = useShopToast();
  const [copied, setCopied] = useState(false);
  const qr = mediaUrl(info.qr_url);

  const copy = async () => {
    try {
      await navigator.clipboard.writeText(info.account_number);
      setCopied(true);
      toast(S.copied);
      setTimeout(() => setCopied(false), 2500);
    } catch {
      setCopied(false);
    }
  };

  /** **تنزيلُ الصورة** — جلبٌ كـ blob ثمّ رابطٌ للتنزيل؛ وإن منع المتصفّحُ (CORS) تُفتح في تبويبٍ جديد. */
  const saveQr = async () => {
    if (!qr) return;
    try {
      const res = await fetch(qr);
      if (!res.ok) throw new Error("fetch");
      const blob = await res.blob();
      const url = URL.createObjectURL(blob);
      const a = document.createElement("a");
      a.href = url;
      a.download = "shamcash-qr.png";
      document.body.appendChild(a);
      a.click();
      a.remove();
      setTimeout(() => URL.revokeObjectURL(url), 5000);
    } catch {
      window.open(qr, "_blank", "noopener");
    }
  };

  const ticket = qr ? (
    <div className="sham-ticket mx-auto w-full max-w-60 rounded-rg-lg p-2">
      <img src={qr} alt={S.qrAlt} className="aspect-square w-full object-contain" />
    </div>
  ) : null;

  const details = (
    <>
      <div className="flex flex-col items-center gap-0.5 text-center">
        {info.account_name && (
          <>
            <span className="text-xs text-ink-muted">{S.holder}</span>
            <span className="text-card-title font-bold">{info.account_name}</span>
          </>
        )}
      </div>
      <div className="flex flex-col gap-1">
        <span className="text-center text-xs text-ink-muted">{S.number}</span>
        <div className="flex items-center gap-2 rounded-control border border-line bg-ink-faint py-1.5 ps-3 pe-1.5">
          <span dir="ltr" className="sham-number min-w-0 flex-1 text-start text-sm font-bold tabular-nums">
            {info.account_number}
          </span>
          <button
            type="button"
            aria-label={S.copy}
            title={S.copy}
            onClick={() => void copy()}
            className="grid size-11 shrink-0 place-items-center rounded-full bg-primary text-on-bright"
          >
            <IconCopy size={18} />
          </button>
        </div>
      </div>
      {/* **النسخُ وحفظُ الصورة جنباً إلى جنب** — فعلان متكافئان. */}
      <div className="flex gap-3">
        <Button type="button" size="lg" className="flex min-h-12 flex-1 items-center justify-center gap-2" onClick={() => void copy()}>
          <IconCopy size={16} />
          {copied ? S.copied : S.copy}
        </Button>
        {qr && (
          <Button
            type="button"
            size="lg"
            variant="secondary"
            className="flex min-h-12 flex-1 items-center justify-center gap-2"
            onClick={() => void saveQr()}
          >
            <IconDownload size={16} />
            {S.saveQr}
          </Button>
        )}
      </div>
    </>
  );

  if (!hero) {
    return (
      <div className="flex flex-col gap-4 rounded-rg-lg border border-line bg-surface p-rg-lg" data-testid="shamcash-account">
        {ticket}
        {details}
      </div>
    );
  }

  return (
    <div className="overflow-hidden rounded-rg-lg border border-line bg-surface shadow-sm" data-testid="shamcash-account">
      <div className="prize-hero flex flex-col gap-4 p-rg-lg">
        <div className="flex items-center gap-3">
          <span aria-hidden className="prize-hero-soft grid size-10 shrink-0 place-items-center rounded-control">
            <IconWallet size={20} />
          </span>
          <div className="flex min-w-0 flex-col">
            <span className="text-card-title font-bold">{S.accountTitle}</span>
            <span className="text-xs opacity-90">{S.accountSub}</span>
          </div>
        </div>
        {ticket}
      </div>
      <div className="flex flex-col gap-4 p-rg-lg">{details}</div>
    </div>
  );
}
