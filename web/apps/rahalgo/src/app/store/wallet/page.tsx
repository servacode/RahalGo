"use client";

/** محفظتي — الصفحة المركزية نفسها، ولصاحب المتجر فيها طلبُ سحب. */

import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { WalletPage } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";

const m = getMessages(defaultLocale);

export default function MerchantWalletPage() {
  const { user } = useAuth();
  return (
    <WalletPage
      api={api}
      path="/api/v1/my/wallet"
      balanceLabel={m.merchant.walletBalance}
      hint={m.merchant.walletHint}
      payouts
      holderName={user?.full_name}
      holderPhone={user?.phone}
    />
  );
}
