import type { Metadata } from "next";
import { getMessages, getDir, defaultLocale, withPlatform } from "@rahalgo/i18n";
import { PlatformProvider, fetchPlatform } from "@rahalgo/ui";
import { AuthProvider } from "@/lib/auth";
// خط المنصة — مصدر مركزي واحد (packages/ui/src/fonts.css)
import "@rahalgo/ui/fonts.css";
import "./globals.css";

const m = getMessages(defaultLocale);

/**
 * **وعنوانُ الصفحة من الإعدادات لا من المعجم.**
 *
 * (قاعدةُ المالك: «لا أريد أن تكتب اسمَ المنصة بأيّ مكانٍ أبداً».)
 *
 * **وكان مكتوباً في المعجم** — فمن بدّل الاسمَ من اللوحة بدّل الشريطَ
 * والشعار، **وبقي عنوانُ التبويب يقول الاسمَ القديم.**
 *
 * **و`generateMetadata` لا ثابتٌ**: الأوّلُ يُنادى لكلّ طلبٍ فيقرأ الإعدادَ
 * الحيّ، **والثابتُ يُحسب مرّةً عند البناء فيتجمّد.**
 */
export async function generateMetadata(): Promise<Metadata> {
  const brand = await fetchPlatform(API);
  const name = brand.name || m.common.appName;
  const title = withPlatform(m.merchant.appTitle, name);
  const description = withPlatform(m.merchant.appDescription, name);
  return {
    metadataBase: new URL(SITE),
    title: { default: title, template: `%s | ${name}` },
    description,
    openGraph: { title, description, siteName: name, locale: "ar_SY", type: "website" },
  };
}

/** أصلُ المحرّك — **هويّةُ المنصة تُقرأ منه لا من المعجم.** */
const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";
const SITE = process.env.NEXT_PUBLIC_SITE_URL ?? "http://localhost:3002";

/* **والهويّةُ تُقرأ في الخادم** — فترسم الخلفيّةُ والشعارُ مع أوّل
   رسمة. (والشرحُ في `platform-server.ts`.) */
export default async function RootLayout({ children }: { children: React.ReactNode }) {
  const brand = await fetchPlatform(API);
  return (
    <html lang={defaultLocale} dir={getDir(defaultLocale)}>
      <body className="flex min-h-screen flex-col">
        <PlatformProvider apiBase={API} initial={brand}>
          <AuthProvider>{children}</AuthProvider>
        </PlatformProvider>
      </body>
    </html>
  );
}
