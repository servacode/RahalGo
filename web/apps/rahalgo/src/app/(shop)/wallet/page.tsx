"use client";

/**
 * **محفظتي — الرصيدُ والحركاتُ والكشف** (مرآةُ `WalletScreen` بلا سحب).
 *
 * **والزبونُ لا يسحب** — رصيدُه يُنفَق لا يُقبَض (قرارُ المالك ٢٠٢٦-٠٨-١٤).
 * **والشاشةُ شاشةُ العُدّة نفسُها** (`WalletPage`) — بطاقاتٌ لا جدول.
 */

import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { IconHelp, LoadingState, WalletPage } from "@rahalgo/ui";
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
      {/* **شحنُ المحفظة** — نقداً بالمكتب، أو بتحويل شام كاش حين يفعّله المالك (الزرُّ يظهر بشرطه بجانب «كشف حساب»). */}
      <ShamcashTopUp>
        {({ action, requests }) => (
          <WalletPage
            api={api}
            path="/api/v1/my/wallet"
            balanceLabel={W.balance}
            holderName={user.full_name}
            holderPhone={user.phone}
            cardsOnly
            hero
            heroActions={action}
            belowHero={
              <>
                {requests}
                <div className="flex items-start gap-3 rounded-rg-lg bg-info-tint p-3" data-testid="wallet-topup-info">
                  <span aria-hidden className="grid size-9 shrink-0 place-items-center rounded-control bg-surface text-info">
                    <IconHelp size={18} />
                  </span>
                  <div className="flex min-w-0 flex-col gap-0.5">
                    <p className="text-card-title">{W.topUpTitle}</p>
                    <p className="text-xs text-ink-muted">{W.topUpBody}</p>
                  </div>
                </div>
              </>
            }
          />
        )}
      </ShamcashTopUp>
    </div>
  );
}
