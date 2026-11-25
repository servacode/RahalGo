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
 *
 * **والشحنُ بشام كاش** (قرارُ المالك ٢٠٢٦-١١-٢٦) — بلوحة الزبون نفسِها (`ShamcashTopUp`) بصفة المتجر:
 * المشحونُ «رصيد شحن» للتوصيل والديون، **وما بينسحب** (حدُّ السحب `withdrawable` من الخادم).
 */

import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { WalletPage } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { SettlementCard } from "@/components/store/StoreSettings";
import { BackArrow } from "@/components/store/SetupParts";
import { ShamcashTopUp } from "@/components/customer/ShamcashTopUp";

const m = getMessages(defaultLocale);
const S = m.store;

export default function StoreWallet() {
  const { user } = useAuth();
  return (
    <ShamcashTopUp as="merchant" panelHint={S.wallet.panelHint}>
      {({ action, requests }) => (
        <WalletPage
          api={api}
          path="/api/v1/my/wallet"
          balanceLabel={S.wallet.balance}
          holderName={user?.full_name}
          holderPhone={user?.phone}
          payouts
          console
          leading={<BackArrow />}
          heroActions={action}
          belowHero={
            action ? (
              <>
                <p className="text-sm text-ink-muted" data-testid="store-shamcash-hint">
                  {S.wallet.shamcashHint}
                </p>
                {requests}
              </>
            ) : null
          }
          aside={<SettlementCard />}
        />
      )}
    </ShamcashTopUp>
  );
}
