"use client";

/** محفظتي — الصفحة المركزية نفسها بترتيب تبويبات المتجر. */

import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { WalletPage } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";

const m = getMessages(defaultLocale);

/** الأهمّ لصاحب المتجر: مستحقّه أولاً ثم ما صُرف له. */
const KIND_ORDER = [
  "merchant_earning",
  "payout",
  "adjustment",
  "compensation",
  "topup",
  "order_payment",
  "refund",
];

export default function MerchantWalletPage() {
  const { user } = useAuth();
  return (
    <WalletPage
      api={api}
      path="/api/v1/my/wallet"
      kindOrder={KIND_ORDER}
      balanceLabel={m.merchant.walletBalance}
      hint={m.merchant.walletHint}
      payouts
      holderName={user?.full_name}
      holderPhone={user?.phone}
    />
  );
}
