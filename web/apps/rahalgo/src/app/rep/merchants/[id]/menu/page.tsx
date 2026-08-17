"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **أصنافُ العميل — يبنيها المندوبُ نيابةً عنه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-١٨: «يقوم المندوبُ بإضافة أصناف المنتجات الموجودة
 *  لدى المتجر بدلاً عنه… ويستطيع تعديلَ السعر وجعلَ المنتج متاحاً أو غيرَ
 *  متاحٍ ومتوفّراً وغيرَ متوفّر — نفس الفورم الموجود عند مدير المنصّة
 *  والموجود عند المتجر».)
 *
 * # ولا محرّرَ ثالث
 *
 * **`MenuManager` هو محرّرُ الإدارة والمتجر نفسُه** — يأخذ مساراتِ بوّابته
 * ورافعَ صورها. **ونسخةٌ ثالثةٌ من نموذجٍ فيه ثمانيةُ حقولٍ تعني ثلاثةَ
 * أماكنَ يُصلَح فيها العيبُ ويُنسى ثالثُها.**
 *
 * # وسعرُ البيع يُعرض له
 *
 * **بخلاف صاحب المتجر** — **المندوبُ يبني القائمةَ نيابةً فيحتاج الرقمين**:
 * ما يقبضه المتجرُ وما تبيع به المنصّة. **وصاحبُ المتجر يضع سعرَه ويقبض
 * عليه، وما تبيع به المنصّةُ ليس شأنَه.**
 *
 * # والحارسُ في المحرّك لا هنا
 *
 * **الشاشةُ تُفتح بمعرّفٍ في العنوان** — ومن بدّله بيده يصل إلى متجرٍ ليس
 * عميلَه. **فالمحرّكُ يسأل في كلّ نداء: أهذا المتجرُ عميلُه؟** — وهذه
 * الصفحةُ تعرض ما يردّه لا أكثر.
 */

import { use } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import {
  MenuManager,
  PageContainer,
  PageHeader,
  ImageUpload,
  MediaThumb,
  IconStore,
  type MenuPaths,
} from "@rahalgo/ui";
import { api, mediaUrl, ApiError } from "@/lib/api";

const m = getMessages(defaultLocale);

/** **مساراتُ بوّابة المندوب** — والحارسُ في الخادم يتحقّق من العميل. */
const PATHS: MenuPaths = {
  menu: (id) => `/api/v1/rep/stores/${id}/menu`,
  sections: (id) => `/api/v1/rep/stores/${id}/menu/sections`,
  section: (id) => `/api/v1/rep/menu/sections/${id}`,
  items: (id) => `/api/v1/rep/stores/${id}/menu/items`,
  item: (id) => `/api/v1/rep/menu/items/${id}`,
  platformSections: () => `/api/v1/rep/platform-sections`,
};

const MEDIA = "/api/v1/rep/media";

/** **وترجمةُ خطأ الخادم تبقى في التطبيق** — `ApiError` نسخةُ كلٍّ من نفسِه. */
function errText(err: unknown): string {
  if (err instanceof ApiError) {
    if (err.body.message_key === "errors.image_too_large") return m.errors.image_too_large;
    if (err.body.message_key === "errors.invalid_image") return m.errors.invalid_image;
  }
  return m.errors.internal;
}

export default function RepMenuPage({ params }: { params: Promise<{ id: string }> }) {
  const { id } = use(params);
  return (
    <PageContainer>
      <PageHeader icon={IconStore} title={m.terms.menu} subtitle={m.rep.menuHint} />
      <MenuManager
        api={api}
        paths={PATHS}
        merchantID={id}
        /* **ويرى سعرَ البيع** — انظر أعلى الملفّ. */
        showSalePrice
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
