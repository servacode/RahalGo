import type { Metadata } from "next";
import { getMessages, getDir, defaultLocale } from "@rahalgo/i18n";
import { AuthProvider } from "@/lib/auth";
import { CartProvider } from "@/lib/cart";
import Header from "@/components/Header";
// الخط المعتمد (BRAND.md) مستضاف ذاتياً
import "@fontsource/tajawal/400.css";
import "@fontsource/tajawal/500.css";
import "@fontsource/tajawal/700.css";
import "./globals.css";

const m = getMessages(defaultLocale);
const SITE = process.env.NEXT_PUBLIC_SITE_URL ?? "http://localhost:3003";

export const metadata: Metadata = {
  metadataBase: new URL(SITE),
  title: { default: m.site.appTitle, template: `%s | ${m.common.appName}` },
  description: m.site.appDescription,
  openGraph: {
    title: m.site.appTitle,
    description: m.site.appDescription,
    siteName: m.common.appName,
    locale: "ar_SY",
    type: "website",
  },
};

export default function RootLayout({ children }: { children: React.ReactNode }) {
  return (
    <html lang={defaultLocale} dir={getDir(defaultLocale)}>
      <body>
        <AuthProvider>
          <CartProvider>
            <Header />
            <main className="mx-auto w-full max-w-5xl flex-1 p-4">{children}</main>
          </CartProvider>
        </AuthProvider>
      </body>
    </html>
  );
}
