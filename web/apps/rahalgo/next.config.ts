import type { NextConfig } from "next";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **تطبيعُ العناوين — موضعٌ واحدٌ قبل أن تُخبَز**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٠: «نرفعه على Render للتجربة».)
 *
 * **ومنصّةُ الاستضافة تعطي اسمَ مضيفٍ لا رابطاً كاملاً**: مواصفةُ Render
 * تسمح بـ`host` (`rahalgo-api.onrender.com`) **ولا تعرف `url` أصلاً.**
 *
 * **وعنوانٌ بلا مخطّطٍ يُنادى `/rahalgo-api.onrender.com/api/v1/...`** —
 * مساراً نسبيّاً في الموقع نفسِه، **فيردّ ٤٠٤ ولا شيءَ يقول لماذا.**
 *
 * **فيُكمَّل هنا مرّةً واحدة.** ولا يُكتب هذا الشرطُ في كلّ قارئ:
 * `NEXT_PUBLIC_*` تُستبدَل نصّاً وقتَ البناء، **فما يُكتب هنا هو ما يصل
 * كلَّ ملفّ.**
 */
function asURL(raw: string | undefined, fallback: string): string {
  const v = (raw ?? "").trim();
  if (!v) return fallback;
  return /^https?:\/\//i.test(v) ? v.replace(/\/+$/, "") : `https://${v}`;
}

/**
 * **وعنوانُ الموقع من المنصّة نفسِها** — `RENDER_EXTERNAL_URL` تضعه
 * الاستضافةُ في الخدمة، **فلا يُسأل عنه أحدٌ ولا يُكتب بيد.**
 *
 * **ومن رفع على غير Render يضبط `NEXT_PUBLIC_SITE_URL`** — وهي الأسبق.
 */
const SITE = asURL(
  process.env.NEXT_PUBLIC_SITE_URL || process.env.RENDER_EXTERNAL_URL,
  "http://localhost:3003",
);

const nextConfig: NextConfig = {
  transpilePackages: ["@rahalgo/ui", "@rahalgo/i18n"],
  env: {
    NEXT_PUBLIC_API_URL: asURL(process.env.NEXT_PUBLIC_API_URL, "http://localhost:8080"),
    NEXT_PUBLIC_SITE_URL: SITE,
  },
};

export default nextConfig;
