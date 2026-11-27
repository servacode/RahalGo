"use client";

/**
 * **‏/store/cards — «بطاقاتي» للمتجر** (قرارُ المالك ٢٠٢٦-١١-٢٧): بطاقةُ رحّال غو وكرتُ اليوم كبيرتين، تُفتحان
 * من أيقونة الشريط العلويّ (مرآةُ `Overlay.Cards` في تطبيق المتجر). **والمطفأةُ لا تُرسم.**
 */

import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { useStore } from "@/lib/store/context";
import { LoyaltyCards, useStampCard } from "@/components/LoyaltyCards";
import { PageHeader } from "@/components/store/SetupParts";

const L = getMessages(defaultLocale).loyalty;

export default function StoreCardsRoute() {
  const { store } = useStore();
  const card = useStampCard(store ? `/api/v1/merchant/stores/${store.id}/stamp-card` : null);
  return (
    <div className="flex flex-col gap-rg-md">
      <PageHeader title={L.cardsTitle} />
      <LoyaltyCards card={card} seenKey={`store.${store?.id ?? ""}`} />
    </div>
  );
}
