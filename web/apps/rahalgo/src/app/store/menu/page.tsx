"use client";

/**
 * الأصناف — المحرّر المركزي نفسه بمسارات بوابة المتجر.
 *
 * # وصورةُ الصنف
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-٠٧: «كلُّ منتجٍ بالإضافة إلى تفاصيله يكون له صورة».)
 *
 * **وكان كلُّ ما يلزمها مبنيّاً ولا يد توصله**: العمودُ في قاعدة البيانات،
 * والحقلُ في المحرّك، و`imageUpload` و`thumb` في `MenuManager` — **والصفحةُ
 * لا تمرّرهما.** فصاحبُ المطعم يملك صنفاً بلا وجه.
 *
 * **والرفعُ كان للإدارة وحدَها** (`‎/admin/media`)، فوُلدت نقطةٌ في بوّابة
 * المتجر بأنواعٍ محصورة — **ولا يرفع تاجرٌ شعارَ المنصة.**
 */

import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  MenuManager,
  PageContainer,
  PageHeader,
  LoadingState,
  ImageUpload,
  MediaThumb,
  IconStore,
  type MenuPaths,
} from "@rahalgo/ui";
import { api, mediaUrl, ApiError } from "@/lib/api";
import { useStore } from "@/lib/store";

const m = getMessages(defaultLocale);

/** مسارات بوابة المتجر — الحارس في الخادم يتحقق من الملكية. */
const PATHS: MenuPaths = {
  menu: (id) => `/api/v1/merchant/stores/${id}/menu`,
  items: (id) => `/api/v1/merchant/stores/${id}/menu/items`,
  item: (id) => `/api/v1/merchant/menu/items/${id}`,
  platformSections: () => `/api/v1/merchant/platform-sections`,
};

/**
 * **نقطةُ رفع بوّابة المتجر** — وحارسُ أنواعها في المحرّك أضيقُ من الإدارة.
 *
 * **ونوعُ صورةِ القسم غيرُ نوع الصنف** (`menu_section` لا `menu_item`):
 * حارسُ المحرّك يفحص النوع، **وصورةُ قسمٍ تُرفع باسم صنفٍ تُحسب صنفاً حين
 * تُنظَّف الوسائطُ غيرُ المستعملة.**
 */
const MEDIA = "/api/v1/merchant/media";

/** **وترجمةُ خطأ الخادم تبقى في التطبيق** — `ApiError` نسخةُ كلٍّ من نفسِه. */
function errText(err: unknown): string {
  if (err instanceof ApiError) {
    if (err.body.message_key === "errors.image_too_large") return m.errors.image_too_large;
    if (err.body.message_key === "errors.invalid_image") return m.errors.invalid_image;
  }
  return m.errors.internal;
}

export default function MerchantMenuPage() {
  const { store } = useStore();
  if (!store) return <LoadingState />;
  return (
    <PageContainer>
      <PageHeader icon={IconStore} title={m.terms.menu} subtitle={store.name} />
      <MenuManager
        api={api}
        paths={PATHS}
        merchantID={store.id}
        imageUpload={(initialUrl, onChange) => (
          <ImageUpload
            kind="menu_item"
            label={m.common.media.image}
            initialUrl={initialUrl}
            /* **والفراغُ يعني «أزِلها»** — `MenuManager` يميّز `null` من
               `undefined`، فالسلسلةُ الفارغةُ تُترجَم إزالةً صريحة. */
            onChange={(id) => onChange(id || null)}
            api={api}
            mediaUrl={mediaUrl}
            path={MEDIA}
            errorText={errText}
          />
        )}
        thumb={(url, alt) => <MediaThumb url={url} alt={alt} fallback={alt} size={44} mediaUrl={mediaUrl} />}
        mediaUrl={mediaUrl}
      />
    </PageContainer>
  );
}
