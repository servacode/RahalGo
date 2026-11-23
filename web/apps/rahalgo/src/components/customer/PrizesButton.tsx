"use client";

/**
 * **زرُّ الجوائز في رأس السوق** — هديّةٌ بجانب البحث وشارةُ «٣/٥»، تُخفى قبل الإطلاق.
 * **والنبضةُ في CSS** (`.shop-prize-badge`) وتُعاد بتبديل `key` الشارة عند تغيّر العدد.
 */

import Link from "next/link";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { IconCatGifts } from "@rahalgo/ui";
import { badgeText, prizesLive, progressInEntry } from "@/lib/customer/prizes";
import { fill } from "@/lib/customer/texts";
import { usePrizesMe } from "@/lib/customer/usePrizes";

const P = getMessages(defaultLocale).shop.prizes;

export function PrizesButton() {
  const { me } = usePrizesMe();
  if (!me || !prizesLive(me)) return null;
  const text = badgeText(me);
  const { done, total } = progressInEntry(me);
  return (
    <Link
      href="/shop/prizes"
      className="shop-iconbtn"
      aria-label={fill(P.badgeLabel, { n: String(done), t: String(total) })}
      title={P.title}
      data-testid="prizes-button"
    >
      <IconCatGifts size={20} />
      <i key={text} className="shop-prize-badge" dir="ltr">
        {text}
      </i>
    </Link>
  );
}
