import type { MetadataRoute } from "next";

const SITE = process.env.NEXT_PUBLIC_SITE_URL ?? "http://localhost:3003";

export default function robots(): MetadataRoute.Robots {
  return {
    rules: {
      userAgent: "*",
      allow: "/",
      // صفحات الحساب الشخصية لا تُفهرس
      disallow: ["/cart", "/orders", "/wallet", "/login"],
    },
    sitemap: `${SITE}/sitemap.xml`,
  };
}
