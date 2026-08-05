"use client";

/**
 * **العروض** — ما تُنزله المنصةُ ويراه الزبونُ بلا أن يبحث.
 *
 * # الفرقُ عن كود الخصم
 *
 * **الكودُ يُكتب والعرضُ يُرى.** ومن لم يسمع بالكود لا يستفيد منه **ولا يعلم
 * أنّه فاته.**
 *
 * # والسعرُ من الخادم
 *
 * **حسبةٌ في المتصفّح تفترق عمّا يُقيَّد في الطلب** — فيرى سعراً ويُحاسَب
 * بآخر، **وهي أسرعُ طريقةٍ لكسر الثقة.**
 */

import { useEffect, useState } from "react";
import Link from "next/link";
import { getMessages, defaultLocale, fmtNum } from "@rahalgo/i18n";
import { EmptyState, LoadingState, IconPromos } from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const C = m.customer.offers;

interface Offer {
  id: string;
  kind: "banner" | "discount";
  title: string;
  body: string;
  image_url: string | null;
  href: string;
  menu_item_id: string | null;
  item_name: string;
  merchant_name: string;
  item_image_url: string | null;
  price_before: number;
  price_after: number;
  discount_percent: number | null;
}

export default function OffersPage() {
  const [rows, setRows] = useState<Offer[] | null>(null);

  useEffect(() => {
    api<{ offers: Offer[] }>("/api/v1/public/offers")
      .then((r) => setRows(r.offers ?? []))
      .catch(() => setRows([]));
  }, []);

  if (rows === null) return <LoadingState />;

  const banners = rows.filter((o) => o.kind === "banner");
  const discounts = rows.filter((o) => o.kind === "discount");

  return (
    <div className="mx-auto w-full max-w-3xl space-y-5 p-4">
      <div>
        <h1 className="flex items-center gap-2 text-xl font-bold">
          <IconPromos size={20} className="text-ink-muted" />
          {C.title}
        </h1>
        <p className="mt-1 text-sm text-ink-muted">{C.subtitle}</p>
      </div>

      {rows.length === 0 && <EmptyState icon={IconPromos} title={C.empty} />}

      {/* **اللافتاتُ أوّلاً** — خبرٌ يُقرأ بنظرة، والخصومُ تحتها تُتصفَّح. */}
      {banners.map((o) => {
        const card = (
          <div className="overflow-hidden rounded-card border border-line bg-surface">
            {o.image_url && (
              // eslint-disable-next-line @next/next/no-img-element
              <img src={o.image_url} alt="" className="h-36 w-full object-cover" />
            )}
            <div className="p-4">
              <p className="font-bold">{o.title}</p>
              {o.body && <p className="mt-1 text-sm text-ink-muted">{o.body}</p>}
            </div>
          </div>
        );
        /* **ولافتةٌ بلا وجهةٍ تُقرأ ولا تُفتح** — وهي حالٌ مشروعة، **ورابطٌ
           يُضغط ولا يذهب يُعلّم ألّا يُضغط ما بعده.** */
        return o.href ? (
          <Link key={o.id} href={o.href} className="block">
            {card}
          </Link>
        ) : (
          <div key={o.id}>{card}</div>
        );
      })}

      {discounts.length > 0 && (
        <div className="grid gap-3 sm:grid-cols-2">
          {discounts.map((o) => (
            <Link
              key={o.id}
              href={`/item/${o.menu_item_id}`}
              className="flex gap-3 rounded-card border border-line bg-surface p-3 transition-colors hover:border-accent"
            >
              {o.item_image_url && (
                // eslint-disable-next-line @next/next/no-img-element
                <img
                  src={o.item_image_url}
                  alt=""
                  className="h-20 w-20 shrink-0 rounded-control object-cover"
                />
              )}
              <div className="min-w-0 flex-1">
                <p className="truncate font-bold">{o.item_name || o.title}</p>
                <p className="truncate text-sm text-ink-muted">{o.merchant_name}</p>
                <div className="mt-1.5 flex items-baseline gap-2" dir="ltr">
                  <span className="font-bold text-success">
                    {fmtNum(o.price_after)} {m.common.currency}
                  </span>
                  <span className="text-xs text-ink-muted line-through">
                    {fmtNum(o.price_before)}
                  </span>
                </div>
                <span className="mt-1 inline-block rounded-badge bg-danger/10 px-2 py-0.5 text-2xs font-bold text-danger">
                  −{o.discount_percent}%
                </span>
              </div>
            </Link>
          ))}
        </div>
      )}
    </div>
  );
}
