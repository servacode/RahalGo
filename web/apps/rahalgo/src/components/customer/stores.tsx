"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **شريطُ «متاجر مميّزة» — مربّعاتٌ لا دوائر** (قرارُ المالك ٢٠٢٦-١٠-١٨)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **مربّعٌ بزوايا مدوّرة** فيه شعارُ المتجر، **واسمُه تحته.** ومتجرٌ بلا
 * شعارٍ يُرسم **حرفَه الأوّلَ على مربّعٍ ملوّن** — لا مربّعاً فارغاً.
 *
 * **ويمرّ أفقيّاً من اليمين** (RTL) — والضغطةُ تفتح `/stores/{id}`.
 *
 * **والدوائرُ لشريط الأقسام** — **وشكلان مختلفان يقولان إنّهما بابان مختلفان.**
 */

import Link from "next/link";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { mediaUrl } from "@/lib/api";
import type { FeaturedStore } from "@/lib/customer/types";

const m = getMessages(defaultLocale);
const ST = m.shop.stores;

/** **الحرفُ الأوّل من الاسم** — بديلُ الشعار الغائب. */
export function storeInitial(name: string): string {
  const ch = Array.from(name.trim())[0];
  return ch ?? "";
}

/** **شعارُ المتجر في مربّع** — أو حرفُه الأوّلُ على لونٍ هادئ. */
export function StoreLogo({
  name,
  logo,
  className = "h-16 w-16",
}: {
  name: string;
  logo: string | null | undefined;
  className?: string;
}) {
  const src = mediaUrl(logo ?? null);
  return (
    <div className={`flex shrink-0 items-center justify-center overflow-hidden rounded-card border border-line bg-accent-tint ${className}`}>
      {src ? (
        // eslint-disable-next-line @next/next/no-img-element
        <img src={src} alt="" loading="lazy" className="h-full w-full object-cover" />
      ) : (
        <span aria-hidden className="heading-section text-accent-text">
          {storeInitial(name)}
        </span>
      )}
    </div>
  );
}

/** **الشريطُ نفسُه** — ولا يُرسم إن لم يصل متجر. */
export function FeaturedStrip({ stores }: { stores: FeaturedStore[] }) {
  if (stores.length === 0) return null;
  return (
    <section aria-label={ST.featured} className="flex flex-col gap-2">
      <h2 className="heading-section">{ST.featured}</h2>
      <ul className="-mx-3 flex snap-x gap-3 overflow-x-auto px-3 pb-1 sm:-mx-4 sm:px-4">
        {stores.map((s) => (
          <li key={s.id} className="w-20 shrink-0 snap-start">
            <Link href={`/stores/${s.id}`} className="flex flex-col items-center gap-1.5 text-center">
              <StoreLogo name={s.name} logo={s.logo_thumb_url ?? s.logo_url} className={`h-20 w-20 ${s.open_now ? "" : "opacity-70"}`} />
              <span className="line-clamp-2 w-full text-xs font-medium leading-snug">{s.name}</span>
            </Link>
          </li>
        ))}
      </ul>
    </section>
  );
}
