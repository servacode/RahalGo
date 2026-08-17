/**
 * ══════════════════════════════════════════════════════════════════════
 * **عنوانٌ ووصفٌ لكلّ صفحة — لا عنوانٌ واحدٌ للموقع كلّه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-١٧: «أريد SEO قويّاً لظهورها بمحرّكات البحث
 *  وتصدّرها صفحاتِ النتائج الأولى».)
 *
 * # وما كان
 *
 * **كلُّ صفحاتِ الموقع تحمل العنوانَ نفسَه** — اسمَ المنصّة وحدَه. **وهو
 * قرارٌ اتّخذه للتبويب** (٢٠٢٦-٠٨-١٥: «اتركها فقط رحّال غو بكلّ
 * الصفحات») **وقتَ لم يكن الظهورُ في البحث مطلوباً.**
 *
 * **وعنوانُ الصفحة هو أوّلُ ما يقرؤه محرّكُ البحث وأكبرُ ما يُرتّب به** —
 * **وسبعُ صفحاتٍ بعنوانٍ واحدٍ تتنافس على الكلمة نفسِها وتُقرأ نسخاً
 * مكرّرة.** ومن بحث عن «شروط الاستخدام» لا يجد صفحتَه.
 *
 * **والوصفُ هو السطرُ تحت الرابط في نتيجة البحث** — ومن لا وصفَ له
 * يكتب غوغلُ له سطراً من صفحته، **وقد يكون سطرَ تذييل.**
 *
 * # وقالبُ العنوان في الغلاف الجذر
 *
 * **يفرض الاسمَ على كلّ عنوان** — فيُلغى هنا بـ`title.absolute`،
 * **والاسمُ يُلحق بالعنوان** كما تفعل المواقعُ التي تتصدّر.
 */

import type { Metadata } from "next";
import { getMessages, defaultLocale, withPlatform } from "@rahalgo/i18n";
import { fetchPlatform } from "@rahalgo/ui";

const m = getMessages(defaultLocale);
const SITE = process.env.NEXT_PUBLIC_SITE_URL ?? "http://localhost:3003";
const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

/** **يبني بطاقةَ صفحةٍ كاملةً** — عنواناً ووصفاً ورابطاً معياريّاً وصورة. */
export async function pageMeta(
  titleKey: keyof typeof m.site.homePage,
  descKey: keyof typeof m.site.homePage,
  path: string,
): Promise<Metadata> {
  const brand = await fetchPlatform(API);
  const name = brand.name || m.common.appName;
  const H = m.site.homePage as unknown as Record<string, string>;
  const title = withPlatform(H[titleKey] ?? "", name);
  const description = withPlatform(H[descKey] ?? "", name);
  /* **والصورةُ شعارُه** — **ورابطٌ يُشارَك بلا صورةٍ يظهر بمربّعٍ رماديّ**
     في واتساب وفيسبوك، وهما بابا الانتشار في الرقّة. */
  const image = brand.logo ? API + brand.logo : undefined;
  return {
    // **مطلقٌ لا يمرّ بالقالب** — انظر أعلى الملفّ.
    title: { absolute: `${title} | ${name}` },
    description,
    /* **والرابطُ المعياريُّ يمنع تكرارَ الصفحة** — بـ`www` وبلاها،
       وبمعاملاتِ حملةٍ تُلحَق بها: **وثلاثةُ عناوينَ لصفحةٍ واحدةٍ
       تقتسم ترتيبَها.** */
    alternates: { canonical: `${SITE}${path}` },
    openGraph: {
      title,
      description,
      url: `${SITE}${path}`,
      siteName: name,
      locale: "ar_SY",
      type: "website",
      ...(image ? { images: [image] } : {}),
    },
    twitter: {
      card: "summary_large_image",
      title,
      description,
      ...(image ? { images: [image] } : {}),
    },
  };
}
