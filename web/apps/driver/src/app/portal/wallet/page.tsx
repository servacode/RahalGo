"use client";

/**
 * المحفظة — الصفحة المركزية نفسها، بترتيب تبويبات السائق.
 *
 * ولا نقطة خادمٍ خاصّة به: `/my/wallet` مفتوحة لكل موثَّق وتعيد كشفه هو. وقد
 * كان في المشروع مسارٌ لكل دور يفعل الشيء نفسه حرفاً بحرف — والنقطة الثالثة
 * المتطابقة اعترافٌ بأن الأولى لم تكن للدور أصلاً.
 */

import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { WalletPage } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";

const m = getMessages(defaultLocale);

/** الأهمّ للسائق أولاً: أجرُه ثم ما سحبه — لا ترتيب ورودها في القاعدة. */
const KIND_ORDER = [
  "driver_earning",
  "payout",
  "compensation",
  "adjustment",
  "refund",
  "topup",
  "order_payment",
];

export default function DriverWalletPage() {
  const { user } = useAuth();
  return (
    <WalletPage
      api={api}
      path="/api/v1/my/wallet"
      kindOrder={KIND_ORDER}
      balanceLabel={m.shared.payout.available}
      hint={m.shared.payout.hint}
      payouts
      holderName={user?.full_name}
      holderPhone={user?.phone}
    />
  );
}
