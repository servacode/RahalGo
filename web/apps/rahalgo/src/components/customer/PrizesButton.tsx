"use client";

/**
 * **هديّةٌ عائمةٌ مجسّمةٌ بزاوية السوق** (ملاحظتا المالك ٢٠٢٦-١١-٢٤: «عائمة ثري دي» و«بلا الدائرة
 * البيضا») — كالتطبيق (`FloatingGift.kt`)، وعليها نقاطُه هذا الشهر.
 * **والزرُّ يُرى قبل الإطلاق** (قرارُ المالك ٢٠٢٦-١١-٢٣) ليفتح «انتظرونا ١/١٢»؛ والشارةُ وحدَها تُخفى.
 * **والنبضةُ في CSS** (`.shop-prize-badge`) وتُعاد بتبديل `key` الشارة عند تغيّر العدد.
 */

import Link from "next/link";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { badgeText, prizesLive } from "@/lib/customer/prizes";
import { fill } from "@/lib/customer/texts";
import { usePrizesMe } from "@/lib/customer/usePrizes";

const P = getMessages(defaultLocale).shop.prizes;

export function PrizesButton() {
  const { me } = usePrizesMe();
  if (!me) return null;
  const live = prizesLive(me);
  const text = badgeText(me);
  return (
    <Link
      href="/shop/prizes"
      className="prize-fab"
      aria-label={live ? fill(P.badgeLabel, { n: text }) : P.title}
      title={P.title}
      data-testid="prizes-button"
    >
      <img src="/prizes/gift.svg" alt="" width={72} height={72} />
      {live && (
        <i key={text} className="shop-prize-badge" dir="ltr">
          {text}
        </i>
      )}
    </Link>
  );
}
