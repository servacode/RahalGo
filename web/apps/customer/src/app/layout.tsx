import type { Metadata } from "next";
import { getMessages, getDir, defaultLocale } from "@rahalgo/i18n";
import { AuthProvider } from "@/lib/auth";
import { CartProvider } from "@/lib/cart";
import Header from "@/components/Header";
// الخط المعتمد (BRAND.md) مستضاف ذاتياً
import "@fontsource/ibm-plex-sans-arabic/400.css";
import "@fontsource/ibm-plex-sans-arabic/500.css";
import "@fontsource/ibm-plex-sans-arabic/700.css";
import "./globals.css";

const m = getMessages(defaultLocale);

export const metadata: Metadata = {
  title: m.site.appTitle,
  description: m.site.appDescription,
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
