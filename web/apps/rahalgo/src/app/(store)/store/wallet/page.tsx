"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **‏/store/wallet — المحفظةُ وطريقةُ استلام المستحقّات**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `WalletScreen(payouts = true)` وبطاقةِ «كيف تحب تستلم مستحقاتك؟»
 *  في `StoreScreen.kt`.)
 *
 * **والطريقةُ تُبدَّل بسؤال** — «ينطبق على الطلبات الجديدة فقط» — كما في
 * التطبيق. **وما لك نقداً من الطلبات السابقة** يُعرض تحتها ما دامت نقديّة.
 */

import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { WalletPage } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { SettlementCard } from "@/components/store/StoreSettings";
import { BackArrow } from "@/components/store/SetupParts";

const m = getMessages(defaultLocale);
const S = m.store;

export default function StoreWallet() {
  const { user } = useAuth();
  return (
    <WalletPage
      api={api}
      path="/api/v1/my/wallet"
      balanceLabel={S.wallet.balance}
      holderName={user?.full_name}
      holderPhone={user?.phone}
      payouts
      console
      leading={<BackArrow />}
      aside={<SettlementCard />}
    />
  );
}
