"use client";

/**
 * **محفظتي — الرصيدُ والحركاتُ والكشف** (مرآةُ `WalletScreen` بلا سحب).
 *
 * **والزبونُ لا يسحب** — رصيدُه يُنفَق لا يُقبَض (قرارُ المالك ٢٠٢٦-٠٨-١٤).
 * **والشاشةُ شاشةُ العُدّة نفسُها** (`WalletPage`) — بطاقاتٌ لا جدول.
 */

import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Alert, LoadingState, WalletPage } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { NeedLogin } from "@/components/customer/NeedLogin";
import { ShopPageHeader } from "@/components/customer/ShopPageHeader";
import { ShamcashTopUp } from "@/components/customer/ShamcashTopUp";

const m = getMessages(defaultLocale);
const W = m.shop.wallet;

export default function WalletRoute() {
  const { user, loading } = useAuth();
  if (loading) return <LoadingState variant="list" rows={3} />;
  if (!user) return <NeedLogin />;
  return (
    <div className="mx-auto flex w-full flex-col gap-4 max-w-none">
      <ShopPageHeader title={W.title} />
      {/* **كيف تُشحن المحفظة** — نقداً بالمكتب، أو بتحويل شام كاش حين يفعّله المالك (زرٌّ يظهر بشرطه). */}
      <Alert tone="info" title={W.topUpTitle}>
        {W.topUpBody}
      </Alert>
      <ShamcashTopUp />
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
