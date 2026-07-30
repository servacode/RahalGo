import type { MetadataRoute } from "next";

const SITE = process.env.NEXT_PUBLIC_SITE_URL ?? "http://localhost:3003";
const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

// خريطة الموقع: الرئيسية + صفحة كل متجر فعال (تُبنى عند كل طلب للملف).
export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const entries: MetadataRoute.Sitemap = [
    { url: SITE, changeFrequency: "daily", priority: 1 },
  ];
  try {
    const res = await fetch(`${API}/api/v1/public/home`, { next: { revalidate: 3600 } });
    const json = (await res.json()) as { data?: { merchants?: { id: string }[] } };
    for (const m of json.data?.merchants ?? []) {
      entries.push({
        url: `${SITE}/m/${m.id}`,
        changeFrequency: "daily",
        priority: 0.8,
      });
    }
  } catch {
    /* الخادم غير متاح وقت البناء — الرئيسية تكفي */
  }
  return entries;
}
