"use client";

/**
 * محفظتي — الصفحة المركزية نفسها بترتيب تبويبات الزبون.
 *
 * الزبون لا لوحة له فالسجل هنا ضروري: يعرف أين صُرفت نقوده (دفع من المحفظة،
 * تعويض، استرجاع، أو تصحيح مالي). ولا يملك طلب سحب — رصيده يُنفَق لا يُقبَض.
 */

import { useEffect } from "react";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { WalletPage, LoadingState } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth, isLoggedIn } from "@/lib/auth";

const m = getMessages(defaultLocale);

/** من منظور الزبون: ماله أولاً، ثم ما صُرف منه. */
const KIND_ORDER = ["topup", "order_payment", "refund", "compensation", "adjustment"];

export default function CustomerWalletPage() {
  const { user, loading } = useAuth();
  const router = useRouter();

  useEffect(() => {
    if (!loading && !isLoggedIn(user)) router.replace("/login?next=/wallet");
  }, [user, loading, router]);

  if (loading || !isLoggedIn(user)) return <LoadingState />;

  return (
    <WalletPage
      api={api}
      path="/api/v1/my/wallet"
      kindOrder={KIND_ORDER}
      balanceLabel={m.site.wallet.balance}
      hint={m.site.wallet.hint}
      holderName={user?.full_name}
      holderPhone={user?.phone}
    />
  );
}
