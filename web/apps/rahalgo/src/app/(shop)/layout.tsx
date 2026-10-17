import type { Metadata, Viewport } from "next";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { fetchPlatform } from "@rahalgo/ui";
import CustomerShell from "@/components/customer/CustomerShell";
import { readServerConfig } from "@/lib/config";
import pwa from "@/lib/pwa.gen.json";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **غلافُ الزبون — سوقٌ وسلّةٌ وطلباتٌ على الويب**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-١٠-١٧: «زبونُ آيفون ما عنده تطبيق — يدخل ويتصفّح
 *  ويطلب من المتصفّح، ويضيفه للشاشة الرئيسيّة فيفتح كأنّه تطبيق».)
 *
 * **مجموعةُ مسارٍ لا مسار** — `(shop)` لا تظهر في العنوان: `/shop` و`/cart`
 * و`/orders` و`/account`، **وصفحاتُ الدخول** `/login` و`/signup` و`/forgot`.
 *
 * **ووسومُ آيفون هنا لا في الجذر** — سفاري يقرأ وسومَ الصفحة التي ضُغط
 * فيها «إضافة إلى الشاشة الرئيسيّة»، **وهي صفحاتُ السوق.** وشريطُ الحالة
 * `black` لا `black-translucent`: **الشفّافُ يمدّ الصفحةَ تحته** فيقع
 * الشريطُ العلويُّ تحت الساعة والبطّاريّة.
 */
export async function generateMetadata(): Promise<Metadata> {
  const m = getMessages(defaultLocale);
  const brand = await fetchPlatform(readServerConfig().apiUrl);
  const name = brand.name || m.common.appName;
  return {
    appleWebApp: { capable: true, title: name, statusBarStyle: "black" },
    icons: { apple: "/pwa/apple-touch-icon.png" },
    /* **ونكست يكتب `mobile-web-app-capable` وحدَه** لـ`capable` — **وسفاري
       قبل ١٦٫٤ لا يعرف إلّا اسمَه القديم**، فيُفتح في المتصفّح لا كتطبيق. */
    other: { "apple-mobile-web-app-capable": "yes" },
  };
}

export const viewport: Viewport = {
  themeColor: pwa.theme,
  viewportFit: "cover",
  width: "device-width",
  initialScale: 1,
};

export default function ShopLayout({ children }: { children: React.ReactNode }) {
  return <CustomerShell>{children}</CustomerShell>;
}
