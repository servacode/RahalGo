"use client";

/**
 * المحفظة — الصفحة المركزية نفسها، وللسائق فيها طلبُ سحبٍ وسجلُّ طلباته.
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

export default function DriverWalletPage() {
  const { user } = useAuth();
  return (
    <WalletPage
      api={api}
      path="/api/v1/my/wallet"
      balanceLabel={m.shared.payout.available}
      hint={m.shared.payout.hint}
      payouts
      holderName={user?.full_name}
      holderPhone={user?.phone}
      /* **بطاقاتٌ وحدَها** — لوحةُ السائق وتطبيقُه شيءٌ واحد. */
      cardsOnly
    />
  );
}
