"use client";

/** المحفظة — الصفحة المركزية بترتيب تبويبات المندوب وطلب السحب. */

import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { WalletPage } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";

const m = getMessages(defaultLocale);

/** الأهمّ للمندوب أولاً: عمولته ثم ما سحبه — لا ترتيب ورودها في القاعدة. */
const KIND_ORDER = [
  "commission",
  "payout",
  "compensation",
  "adjustment",
  "refund",
  "topup",
  "order_payment",
];

export default function RepWalletPage() {
  const { user } = useAuth();
  return (
    <WalletPage
      api={api}
      path="/api/v1/rep/wallet"
      kindOrder={KIND_ORDER}
      balanceLabel={m.shared.payout.available}
      hint={m.shared.payout.hint}
      payouts
      holderName={user?.full_name}
      holderPhone={user?.phone}
    />
  );
}
