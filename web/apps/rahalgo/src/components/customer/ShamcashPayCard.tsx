"use client";

/**
 * **«ادفع للكابتن بشام كاش»** — تظهر في صفحة الطلب حين يصل الكابتنُ والطلبُ مختارٌ عليه شام كاش
 * ويبقى عليه مبلغٌ: المبلغُ وبطاقةُ حساب المنصّة (باركود ورقم ونسخ).
 */

import { getMessages, defaultLocale, fmtMoney } from "@rahalgo/i18n";
import { fill } from "@/lib/customer/texts";
import { useShamcash } from "@/lib/customer/shamcash";
import { ShamcashAccountCard } from "@/components/customer/ShamcashAccountCard";
import { OUTLINE_CARD } from "@/components/customer/kit";

const m = getMessages(defaultLocale);
const O = m.shop.orders;

export function ShamcashPayCard({ amount }: { amount: number }) {
  const { info } = useShamcash();
  if (!info || !info.account_number) return null;
  return (
    <div className={`${OUTLINE_CARD} flex flex-col gap-rg-sm`} data-testid="shamcash-pay-card">
      <p className="text-card-title">{O.shamcashPayTitle}</p>
      <p className="text-sm">{fill(O.shamcashPayBody, { amount: fmtMoney(amount) })}</p>
      <ShamcashAccountCard info={info} />
    </div>
  );
}
