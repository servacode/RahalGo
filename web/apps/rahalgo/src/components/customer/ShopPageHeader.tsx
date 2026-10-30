"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **رأسُ صفحات «حسابي» — سهمُ رجوعٍ وعنوان** (قرارُ المالك ٢٠٢٦-١٠-٣٠)
 * ══════════════════════════════════════════════════════════════════════
 *
 * مرآةُ `OverlayPage` في تطبيق الزبون: كلُّ صفحةٍ تُفتح من القائمة (السجلّ والمفضّلة
 * والعروض والدردشات والدعوة والشكاوى والمحفظة) **لها سهمُ رجوعٍ واحدُ الشكل**،
 * **ولا سطرَ شرحٍ تحت العنوان.**
 */

import Link from "next/link";
import type { ReactNode } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { IconPrev } from "@rahalgo/ui";

const m = getMessages(defaultLocale);

export function ShopPageHeader({
  title,
  href = "/account",
  actions,
}: {
  title: string;
  /** **إلى أين يرجع السهم** — «حسابي» افتراضاً، ومنها تُفتح هذه الصفحات. */
  href?: string;
  actions?: ReactNode;
}) {
  return (
    <header className="flex items-center gap-2">
      <Link
        href={href}
        aria-label={m.common.back}
        title={m.common.back}
        className="flex h-11 w-11 shrink-0 items-center justify-center rounded-control text-ink transition-colors hover:bg-row-hover"
      >
        <IconPrev size={22} />
      </Link>
      <h1 className="heading-page min-w-0 flex-1 truncate">{title}</h1>
      {actions}
    </header>
  );
}
