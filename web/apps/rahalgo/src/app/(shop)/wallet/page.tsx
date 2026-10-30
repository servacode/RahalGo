"use client";

/**
 * **محفظتي — الرصيدُ والحركاتُ والكشف** (مرآةُ `WalletScreen` بلا سحب).
 *
 * **والزبونُ لا يسحب** — رصيدُه يُنفَق لا يُقبَض (قرارُ المالك ٢٠٢٦-٠٨-١٤).
 * **والشاشةُ شاشةُ العُدّة نفسُها** (`WalletPage`) — بطاقاتٌ لا جدول.
 */

import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { LoadingState, WalletPage } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { NeedLogin } from "@/components/customer/NeedLogin";
import { ShopPageHeader } from "@/components/customer/ShopPageHeader";

const m = getMessages(defaultLocale);
const W = m.shop.wallet;

export default function WalletRoute() {
  const { user, loading } = useAuth();
  if (loading) return <LoadingState variant="list" rows={3} />;
  if (!user) return <NeedLogin />;
  return (
    <div className="flex flex-col gap-4">
      <ShopPageHeader title={W.title} />
      <WalletPage
        api={api}
        path="/api/v1/my/wallet"
        balanceLabel={W.balance}
        holderName={user.full_name}
        holderPhone={user.phone}
        cardsOnly
      />
    </div>
  );
}
