import type { MetadataRoute } from "next";

const SITE = process.env.NEXT_PUBLIC_SITE_URL ?? "http://localhost:3003";
const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

/**
 * **خريطةُ الموقع** — الرئيسيةُ وأقسامُ السوق.
 *
 * # ولماذا لا متاجرَ فيها
 *
 * **كانت تنشر `‎/m/{id}` لكلّ متجرٍ فعّال** — أي أنّها تُسلّم لغوغل قائمةَ
 * مطاعمِنا بأسمائها وصفحاتِها. **والمتاجرُ مخفيّةٌ عن الزبون بالكامل**:
 * المنصةُ سوقٌ يجلب منها، **وهو يشتري «من رحّال» لا «من مطعم فلان».**
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٥.)
 *
 * **وحجبٌ في الشاشة تنقضه خريطةُ الموقع حجبٌ لم يقع**: الصفحاتُ كانت
 * مهجورةً لا يربط إليها شيء، **والخريطةُ وحدَها كانت تفتحها للعالم** —
 * ثمّ تُفهرَس فتُقرأ من البحث مباشرةً.
 *
 * # وما يُنشر بدلاً منها
 *
 * **أقسامُ السوق** (`‎/s/{id}`) — وهي ما يتصفّحه الزبونُ فعلاً في الرئيسية،
 * **فالمنشورُ يطابق المعروض.**
 */
export default async function sitemap(): Promise<MetadataRoute.Sitemap> {
  const entries: MetadataRoute.Sitemap = [
    { url: SITE, changeFrequency: "daily", priority: 1 },
    // ══════════════════════════════════════════════════════════════════
    // **وصفحاتُ المنصّة الثابتة — تُفهرَس وتُقرأ قبل التسجيل**
    // ══════════════════════════════════════════════════════════════════
    //
    // **ووثيقةٌ قانونيةٌ لا يجدها محرّكُ البحث كأنّها غيرُ منشورة** — ومن
    // بحث عن اسم المنصّة ليطمئنّ لم يجد شروطَها.
    //
    // **وغوغل بلاي يطلب رابطاً عامّاً لسياسة الخصوصيّة** — ورابطٌ لا
    // يُفهرَس يُقبل، **لكنّ المراجعَ البشريَّ يفتحه.**
    ...["terms", "privacy", "help", "about", "contact", "delete-account"].map((path) => ({
      url: `${SITE}/${path}`,
      changeFrequency: "monthly" as const,
      priority: 0.5,
    })),
  ];
  try {
    const res = await fetch(`${API}/api/v1/public/home`, { next: { revalidate: 3600 } });
    const json = (await res.json()) as { data?: { sections?: { id: string }[] } };
    for (const s of json.data?.sections ?? []) {
      entries.push({
        url: `${SITE}/s/${s.id}`,
        changeFrequency: "daily",
        priority: 0.8,
      });
    }
  } catch {
    /* الخادم غير متاح وقت البناء — الرئيسية تكفي */
  }
  return entries;
}
