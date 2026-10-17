import type { Metadata, Viewport } from "next";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { fetchPlatform } from "@rahalgo/ui";
import StoreShell from "@/components/store/StoreShell";
import { readServerConfig } from "@/lib/config";
import pwa from "@/lib/pwa.gen.json";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **غلافُ لوحة المتجر — `/store` وما تحته**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-١٠-١٧: صاحبُ متجرٍ معه آيفون يستقبل طلباته من الموقع.)
 *
 * **مجموعةُ مسارٍ لا مسار** — `(store)` لا تظهر في العنوان.
 *
 * **ووسومُ آيفون هنا كما في `(shop)`** — سفاري يقرأ وسومَ الصفحة التي ضُغط
 * فيها «إضافة إلى الشاشة الرئيسيّة»، **ومن أضافها من `/store` يفتحها كتطبيق.**
 */
export async function generateMetadata(): Promise<Metadata> {
  const m = getMessages(defaultLocale);
  const brand = await fetchPlatform(readServerConfig().apiUrl);
  const name = brand.name || m.common.appName;
  return {
    title: `${m.store.nav.panel} · ${name}`,
    appleWebApp: { capable: true, title: name, statusBarStyle: "black" },
    icons: { apple: "/pwa/apple-touch-icon.png" },
    other: { "apple-mobile-web-app-capable": "yes" },
    robots: { index: false, follow: false },
  };
}

export const viewport: Viewport = {
  themeColor: pwa.theme,
  viewportFit: "cover",
  width: "device-width",
  initialScale: 1,
};

export default function StoreLayout({ children }: { children: React.ReactNode }) {
  return <StoreShell>{children}</StoreShell>;
}
