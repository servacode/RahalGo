import type { MetadataRoute } from "next";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { fetchPlatform } from "@rahalgo/ui";
import { readServerConfig } from "@/lib/config";
import pwa from "@/lib/pwa.gen.json";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **بيانُ التطبيق — ما يقرؤه المتصفّحُ حين يُضاف الموقعُ للشاشة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-١٠-١٧: زبونُ آيفون بلا تطبيق — يضيف الموقعَ للشاشة
 *  الرئيسيّة **فيفتح كأنّه تطبيق**: بلا شريط عنوانٍ ولا أزرارِ متصفّح.)
 *
 * - **يبدأ من السوق** (`/shop`) — لا من صفحة التعريف.
 * - **والاسمُ من الإعدادات** كعنوان التبويب — ومن بدّله من اللوحة بدّل
 *   اسمَ الأيقونة معه بلا نشر. **ويسقط إلى المعجم** إن لم تُقرأ.
 * - **والألوانُ والأيقوناتُ مولَّدة** من التوكنز وشعار المنظومة
 *   (`web/scripts/make-pwa.py`) — **ولا لونَ مكتوباً هنا.**
 */
export const dynamic = "force-dynamic";

export default async function manifest(): Promise<MetadataRoute.Manifest> {
  const m = getMessages(defaultLocale);
  const brand = await fetchPlatform(readServerConfig().apiUrl);
  const name = brand.name || m.common.appName;
  return {
    name,
    short_name: name,
    description: m.common.appTagline,
    lang: defaultLocale,
    dir: "rtl",
    start_url: "/shop",
    scope: "/",
    display: "standalone",
    orientation: "portrait",
    theme_color: pwa.theme,
    background_color: pwa.background,
    icons: pwa.icons
      .filter((i) => !i.src.includes("apple-touch"))
      .map((i) => ({
        src: i.src,
        sizes: `${i.size}x${i.size}`,
        type: "image/png",
        purpose: i.src.includes("maskable") ? "maskable" : "any",
      })),
  };
}
