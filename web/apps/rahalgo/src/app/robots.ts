import type { MetadataRoute } from "next";
import { readServerConfig } from "@/lib/config";

const SITE = () => readServerConfig().siteUrl;

/**
 * **يُولَّد عند الطلب لا عند البناء** (دورةُ ٧١و-ر٢).
 *
 * **و`robots.ts` لا يخضع لـ`force-dynamic` التخطيطِ الجذريّ** — له
 * توليدُه الخاصّ. **فكان يُخبَز في الحزمة وهويّةُ البيئة خاليةٌ وقتَ
 * البناء**، فيخرج `Sitemap: /sitemap.xml` نسبيّاً.
 *
 * **ولا `fetch` فيه** — فلا إعادةَ توليدٍ تشفيه بعد النشر أبداً: يُخدَم
 * الجسمُ المخبوزُ إلى آخر عمر الأثر. (قِيس في ٧١و-ر١ مقابلَ الإنتاج
 * القائم الذي يعطي العنوانَ مطلقاً.)
 */
export const dynamic = "force-dynamic";

export default function robots(): MetadataRoute.Robots {
  return {
    rules: {
      userAgent: "*",
      allow: "/",
      // صفحات الحساب الشخصية لا تُفهرس
      // **و`‎/app` مسموحٌ عمداً** — هو البابُ الذي نريد أن يجده الناس.
      //
      // **وبابُ الإدارة لا يُذكر هنا إطلاقاً**: `robots.txt` ملفٌّ عامٌّ
      // يقرؤه كلُّ أحد، **ومن كتب `Disallow: /adminrahalgo` أعلن عنوانَه**
      // — وذاك نقيضُ سبب تغييره.
      // **ولا يُذكر بابُ الإدارة هنا** — `robots.txt` ملفٌّ عامٌّ
      // يقرؤه كلُّ أحد، **ومن كتب `Disallow: /adminrahalgo` أعلن
      // عنوانَه**، وذاك نقيضُ سبب تغييره.
      //
      // **وأقسامُ الزبون حُذفت ٢٠٢٦-٠٨-٢٦** فلم يبقَ ما يُمنع منها.
      disallow: ["/dashboard"],
    },
    sitemap: `${SITE()}/sitemap.xml`,
  };
}
