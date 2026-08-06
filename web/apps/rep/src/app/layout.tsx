import type { Metadata } from "next";
import { getMessages, getDir, defaultLocale } from "@rahalgo/i18n";
import { PlatformProvider, fetchPlatform } from "@rahalgo/ui";
import { AuthProvider } from "@/lib/auth";
// خط المنصة — مصدر مركزي واحد (packages/ui/src/fonts.css)
import "@rahalgo/ui/fonts.css";
import "./globals.css";

const m = getMessages(defaultLocale);

export const metadata: Metadata = {
  title: m.rep.appTitle,
  description: m.rep.appDescription,
};

/** أصلُ المحرّك — **هويّةُ المنصة تُقرأ منه لا من المعجم.** */
const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

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
