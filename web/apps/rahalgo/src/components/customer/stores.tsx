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
import { ShopPic } from "./ShopPic";

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
    // **والشعارُ على أبيض** (ملاحظةُ المالك ٢٠٢٦-١٠-١٩) — شعارٌ أخضرُ على الأزرق كان يذوب،
    // **والأبيضُ يُبقي كلَّ شعارٍ واضحاً مهما كانت الخلفيّة**، والشعارُ كاملٌ لا مقصوص.
    <div
      className={`flex shrink-0 items-center justify-center overflow-hidden rounded-card border border-line ${src ? "bg-on-solid p-1" : "bg-accent-tint"} ${className}`}
    >
      {src ? (
        // eslint-disable-next-line @next/next/no-img-element
        <img src={src} alt="" loading="lazy" className="h-full w-full object-contain" />
      ) : (
        <span aria-hidden className="heading-section text-accent-text">
          {storeInitial(name)}
        </span>
      )}
    </div>
  );
}

/** **الشريطُ نفسُه** — بطاقاتٌ بالشعار والاسم، **ولا يُرسم إن لم يصل متجر.** */
export function FeaturedStrip({ stores, title = ST.featured }: { stores: FeaturedStore[]; title?: string }) {
  if (stores.length === 0) return null;
  return (
    <section aria-label={title} className="flex flex-col gap-3">
      <div className="shop-sec-head">
        <h2>{title}</h2>
      </div>
      <ul className="shop-featured">
        {stores.map((s) => (
          <li key={s.id} className="contents">
            <Link href={`/stores/${s.id}`} className={`shop-store-card ${s.open_now ? "" : "opacity-70"}`}>
              <ShopPic src={s.logo_thumb_url ?? s.logo_url} name={s.name} fit="contain" initial />
              <b>{s.name}</b>
            </Link>
          </li>
        ))}
      </ul>
    </section>
  );
}
