"use client";

/**
 * **‏/cards — «بطاقاتي»** (قرارُ المالك ٢٠٢٦-١١-٢٧): بطاقةُ رحّال غو ثمّ كرتُ اليوم كبيرتين، تُفتحان من أيقونة
 * الشريط العلويّ (مرآةُ `Overlay.Cards` في تطبيق الزبون). **والمطفأةُ لا تُرسم**، والضيفُ يُدعى للدخول.
 */

import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { LoadingState } from "@rahalgo/ui";
import { useAuth } from "@/lib/auth";
import { LoyaltyCards, useStampCard } from "@/components/LoyaltyCards";
import { NeedLogin } from "@/components/customer/NeedLogin";
import { ShopPageHeader } from "@/components/customer/ShopPageHeader";

const L = getMessages(defaultLocale).loyalty;

export default function CardsRoute() {
  const { user, loading } = useAuth();
  const card = useStampCard(user ? "/api/v1/my/stamp-card" : null);
  if (loading) return <LoadingState variant="list" rows={3} />;
  if (!user) return <NeedLogin />;
  return (
    <div className="mx-auto flex w-full max-w-none flex-col gap-4">
      <ShopPageHeader title={L.cardsTitle} href="/shop" />
      <LoyaltyCards card={card} seenKey="customer" />
    </div>
  );
}
