import type { Metadata } from "next";
import { getMessages, getDir, defaultLocale } from "@rahalgo/i18n";
import { AuthProvider } from "@/lib/auth";
// الخط المعتمد (BRAND.md) مستضاف ذاتياً — لا اعتماد على خوادم خارجية وقت البناء أو التشغيل
import "@fontsource/ibm-plex-sans-arabic/400.css";
import "@fontsource/ibm-plex-sans-arabic/500.css";
import "@fontsource/ibm-plex-sans-arabic/700.css";
import "./globals.css";

const m = getMessages(defaultLocale);

export const metadata: Metadata = {
  title: m.admin.appTitle,
  description: m.admin.appDescription,
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang={defaultLocale} dir={getDir(defaultLocale)}>
      <body>
        <AuthProvider>{children}</AuthProvider>
      </body>
    </html>
  );
}
