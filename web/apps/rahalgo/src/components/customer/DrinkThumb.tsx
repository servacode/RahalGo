"use client";

/**
 * **صورةُ المشروب الصغيرة** (قرارُ المالك ٢٠٢٦-١٠-٣٠) — مربّعٌ مدوّرٌ ٣٦ بكسل جنب كلّ مشروب،
 * في نافذة الصنف وفي السلّة. **وبلا صورةٍ أيقونةُ الكأس** — فلا فراغَ يُقرأ عطباً في التحميل.
 */

import { IconCatDrinks } from "@rahalgo/ui";
import { mediaUrl } from "@/lib/api";

export function DrinkThumb({ url }: { url?: string | null }) {
  const src = mediaUrl(url ?? null);
  return (
    <span className="flex h-9 w-9 shrink-0 items-center justify-center overflow-hidden rounded-control bg-field text-ink-muted">
      {src ? (
        // eslint-disable-next-line @next/next/no-img-element
        <img src={src} alt="" loading="lazy" className="h-full w-full object-cover" />
      ) : (
        <IconCatDrinks size={18} />
      )}
    </span>
  );
}
