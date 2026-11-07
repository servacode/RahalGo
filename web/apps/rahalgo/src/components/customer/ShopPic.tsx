"use client";

/**
 * **صورةُ سوق الزبون — الحقيقيّةُ إن وُجدت، وإلّا مربّعٌ مصبوغٌ بأيقونة القسم**
 *
 * **الصبغةُ تُحسب من الاسم** (ثابتةٌ لكلّ اسم) فلا تتبدّل بين زيارتين، **والتدرّجُ في `theme.css`**
 * (`.shop-pic`) يتبدّل مع الثيم. **ولا تُرسم صورةٌ مرسومةٌ باليد**: الصورُ كلُّها من المحرّك.
 */

import type { CSSProperties, ReactNode } from "react";
import { CategoryIcon } from "@rahalgo/ui";
import { mediaUrl } from "@/lib/api";

/** **صبغةٌ ثابتةٌ من النصّ** — ٠ إلى ٣٥٩. */
export function hueOf(text: string): number {
  let h = 7;
  for (const ch of Array.from(text)) h = (h * 31 + (ch.codePointAt(0) ?? 0)) % 360;
  return h;
}

export function ShopPic({
  src,
  name,
  icon,
  iconSize = 32,
  fit = "cover",
  initial = false,
  className = "",
  children,
}: {
  /** **مسارٌ من المحرّك** (يمرّ على `mediaUrl`) أو `null`. */
  src: string | null | undefined;
  name: string;
  /** **مفتاحُ أيقونة القسم** — وغيابُه أيقونةُ «أخرى». */
  icon?: string | null;
  iconSize?: number;
  /** **الشعاراتُ `contain`** (تُرى كاملةً على أبيض) والصورُ `cover`. */
  fit?: "cover" | "contain";
  /** **الحرفُ الأوّلُ من الاسم بدل الأيقونة** — للمتجر بلا شعار. */
  initial?: boolean;
  className?: string;
  children?: ReactNode;
}) {
  const url = mediaUrl(src ?? null);
  return (
    <div className={`shop-pic ${className}`} data-fit={fit} style={{ "--h": hueOf(name) } as CSSProperties}>
      {url ? (
        <img src={url} alt="" loading="lazy" />
      ) : initial ? (
        <span aria-hidden className="shop-ini">
          {Array.from(name.trim())[0] ?? ""}
        </span>
      ) : (
        <CategoryIcon name={icon} size={iconSize} />
      )}
      {children}
    </div>
  );
}
