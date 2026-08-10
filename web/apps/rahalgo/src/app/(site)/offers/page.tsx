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
  Alert,
  EmptyState,
  LoadingState,
  BannerSlider,
  PageContainer,
  PageHeader,
  IconPromos,
} from "@rahalgo/ui";
import ItemGrid from "@/components/ItemGrid";
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
  item_image_url: string | null;
  price_before: number;
  price_after: number;
  discount_percent: number | null;
}

export default function OffersPage() {
  const [rows, setRows] = useState<Offer[] | null>(null);
  /** **وفشلُ الجلب لا يُعرض «لا عروض»** — والزبونُ يقرؤها منصّةً بلا عروضٍ
   *  فلا يعود يفتح الصفحة. **واللافتاتُ استثناء**: زينةٌ تُخفى بلا ضرر. */
  const [failed, setFailed] = useState(false);
  const [banners, setBanners] = useState<Banner[]>([]);

  useEffect(() => {
    api<{ offers: Offer[] }>("/api/v1/public/offers")
      .then((r) => setRows(r.offers ?? []))
      .catch(() => setFailed(true));
    // **واللافتاتُ من مصدر الرئيسية نفسِه.**
    //
    // كانت تُطلب من `/banners` — **وهو طريقُ الإدارة**، يردّ للزبون «غيرُ
    // موجود» **فيبقى السلايدرُ فارغاً أبداً** ولا خطأ يُقال.
    api<{ banners: Banner[] }>("/api/v1/public/home")
      .then((r) => setBanners(r.banners ?? []))
      // @empty-ok — **اللافتةُ زينةٌ تُخفى بلا ضرر**: الخصومُ هي المحتوى،
      // وسلايدرٌ غائبٌ لا يُقرأ نقصاً.
      .catch(() => setBanners([]));
  }, []);

  if (failed) {
    return (
      <PageContainer>
        <PageHeader icon={IconPromos} title={C.title} subtitle={C.subtitle} />
        <Alert tone="warning" title={m.errors.offline}>
          {m.errors.offlineHint}
        </Alert>
      </PageContainer>
    );
  }
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

      {/* **والخصومُ بشكل الأصناف نفسِه.**

          كانت بطاقةً أفقيّةً بمصغَّرةٍ ٨٠ بكسل — **والصنفُ نفسُه في السوق
          والقسم والبحث بطاقةٌ بصورةٍ تملأ عرضَها.** فيُقرأ الصنفُ في العروض
          **سطرَ جدولٍ وفي الأقسام سلعة**، وهو مقلوب: **العرضُ هو ما يُغري.**

          (شهده المالك ٢٠٢٦-٠٨-٠٥: «العروضُ يجب أن تُعرض بنفس طريقة الأصناف
          بالموقع الأساسيّ».)

          **والسعرُ قبل الخصم يُقرأ من البطاقة نفسِها** — فيها `price` وهو
          سعرُ البيع بعد الخصم، **وشارةُ النسبة تقول كم وُفِّر.** */}
      {discounts.length > 0 && (
        <ItemGrid
          next="/offers"
          items={discounts.map((o) => ({
            id: o.menu_item_id ?? o.id,
            name: o.item_name || o.title,
            // **ولا اسمَ متجرٍ هنا.**
            //
            // **الزبونُ يشتري «من رحّال» لا «من مطعم فلان»** — والمتاجرُ
            // مخفيّةٌ عنه بالكامل: المنصةُ سوقٌ يجلب منها. **وقد كتبتُه
            // وصفاً في أوّل صياغةٍ فخالفتُ قاعدةً مكتوبةً في `ItemCard`
            // نفسِها.** (قرارُ المالك ٢٠٢٦-٠٨-٠٥.)
            description: o.body,
            price: o.price_after,
            price_before: o.price_before,
            discount_percent: o.discount_percent,
            image_url: o.item_image_url,
            image_thumb_url: o.item_image_url,
            // **والعرضُ لا يُنشر على صنفٍ موقوف** — فما وصل هنا متاح.
            available: true,
            source_closed: false,
            source_opens_at: null,
            section_id: "",
            section_name: "",
          }))}
        />
      )}
    </PageContainer>
  );
}
