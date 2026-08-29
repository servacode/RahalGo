import type { MetadataRoute } from "next";

const SITE = process.env.NEXT_PUBLIC_SITE_URL ?? "http://localhost:3003";

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
    sitemap: `${SITE}/sitemap.xml`,
  };
}
