"use client";

/**
 * **بطاقةُ حساب شام كاش** — الباركودُ واسمُ صاحب الحساب ورقمُه، وزرّا «نسخ رقم الحساب» و«حفظ صورة
 * الباركود» كبيران (الزبونُ يحوّل من تطبيق شام كاش فيحتاج الرقمَ أو الصورة بلمسة).
 * تُستعمل في لوحة الشحن وفي بطاقة «ادفع للكابتن بشام كاش».
 */

import { useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Button } from "@rahalgo/ui";
import { mediaUrl } from "@/lib/api";
import type { ShamcashInfo } from "@/lib/customer/shamcash";
import { useShopToast } from "@/components/customer/toast";

const m = getMessages(defaultLocale);
const S = m.shop.shamcash;

export function ShamcashAccountCard({ info }: { info: ShamcashInfo }) {
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

  return (
    <div className="flex flex-col gap-3 rounded-control border border-line p-3" data-testid="shamcash-account">
      {qr && <img src={qr} alt={S.qrAlt} className="mx-auto h-52 w-52 rounded-control border border-line object-contain" />}
      <dl className="flex flex-col gap-1 text-sm">
        {info.account_name && (
          <div className="flex justify-between gap-3">
            <dt className="text-ink-muted">{S.holder}</dt>
            <dd className="font-bold">{info.account_name}</dd>
          </div>
        )}
        <div className="flex justify-between gap-3">
          <dt className="text-ink-muted">{S.number}</dt>
          <dd dir="ltr" className="font-bold">
            {info.account_number}
          </dd>
        </div>
      </dl>
      <Button type="button" size="lg" className="w-full" onClick={() => void copy()}>
        {copied ? S.copied : S.copy}
      </Button>
      {qr && (
        <Button type="button" size="lg" variant="secondary" className="w-full" onClick={() => void saveQr()}>
          {S.saveQr}
        </Button>
      )}
    </div>
  );
}
