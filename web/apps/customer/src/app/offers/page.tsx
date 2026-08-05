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
import {
  EmptyState,
  LoadingState,
  BannerSlider,
  PageContainer,
  PageHeader,
  IconPromos,
} from "@rahalgo/ui";
import { api, mediaUrl } from "@/lib/api";

const m = getMessages(defaultLocale);
const C = m.customer.offers;

interface Banner {
  id: string;
  title: string;
  image_url: string | null;
  target: string | null;
}

interface Offer {
  id: string;
  title: string;
  body: string;
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
  const [banners, setBanners] = useState<Banner[]>([]);

  useEffect(() => {
    api<{ offers: Offer[] }>("/api/v1/public/offers")
      .then((r) => setRows(r.offers ?? []))
      .catch(() => setRows([]));
    // **واللافتاتُ من مصدر الرئيسية نفسِه.**
    //
    // كانت تُطلب من `/banners` — **وهو طريقُ الإدارة**، يردّ للزبون «غيرُ
    // موجود» **فيبقى السلايدرُ فارغاً أبداً** ولا خطأ يُقال.
    api<{ banners: Banner[] }>("/api/v1/public/home")
      .then((r) => setBanners(r.banners ?? []))
      .catch(() => setBanners([]));
  }, []);

  if (rows === null) return <LoadingState />;

  const discounts = rows;

  return (
    <PageContainer>
      <PageHeader icon={IconPromos} title={C.title} subtitle={C.subtitle} />

      {rows.length === 0 && banners.length === 0 && (
        <EmptyState icon={IconPromos} title={C.empty} />
      )}

      {/* **اللافتاتُ أوّلاً** — خبرٌ يُقرأ بنظرة، والخصومُ تحتها تُتصفَّح. */}
      <BannerSlider
        Link={Link}
        items={banners.map((b) => ({
          id: b.id,
          title: b.title,
          // **والمسارُ يُحوَّل إلى رابط** — كان يُمرَّر خاماً، **فالصورةُ لا
          // تُحمَّل ويبقى إطارٌ رماديٌّ بعنوان.**
          imageUrl: mediaUrl(b.image_url) ?? null,
          href: b.target || undefined,
        }))}
      />

      {discounts.length > 0 && (
        /* **وبطاقةٌ واحدةٌ بعرض الشاشة تُبعثر العين** — فتُقسم بما يتّسع. */
        <div className="grid gap-3 sm:grid-cols-2 xl:grid-cols-3 2xl:grid-cols-4">
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
                  alt="" loading="lazy"
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
    </PageContainer>
  );
}
