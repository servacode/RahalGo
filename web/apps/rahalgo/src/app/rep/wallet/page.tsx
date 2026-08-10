"use client";

/** المحفظة — الصفحة المركزية نفسها، وللمندوب فيها طلبُ سحب. */

import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { WalletPage } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";

const m = getMessages(defaultLocale);

export default function RepWalletPage() {
  const { user } = useAuth();
  return (
    <WalletPage
      api={api}
      path="/api/v1/rep/wallet"
      balanceLabel={m.shared.payout.available}
      hint={m.shared.payout.hint}
      payouts
      holderName={user?.full_name}
      holderPhone={user?.phone}
    />
  );
}
