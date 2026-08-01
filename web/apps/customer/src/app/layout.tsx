import type { Metadata } from "next";
import { getMessages, getDir, defaultLocale } from "@rahalgo/i18n";
import { AuthProvider } from "@/lib/auth";
import { CartProvider } from "@/lib/cart";
import Header from "@/components/Header";
// خط المنصة — مصدر مركزي واحد (packages/ui/src/fonts.css)
import "@rahalgo/ui/fonts.css";
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
      <body className="flex min-h-screen flex-col">
        <AuthProvider>
          <CartProvider>
            <div className="flex min-h-screen flex-col p-3">
              <Header />
              {/*
                **شريطٌ وكرتُ محتوى — نفس تركيب اللوحات.**

                كان المحتوى عارياً على خلفية الصفحة بينما لوحاتُ الإدارة والمتجر
                والمندوب والسائق تلفّ محتواها بكرتٍ ذي حوافّ وحدّ وظلّ. فبدا
                الموقع صفحةً أخرى من منصّةٍ أخرى — **والمنصةُ الواحدة تُعرف من
                هيكلها قبل ألوانها**.

                والعرض يبقى كاملاً لا محصوراً: اللوحات تحصر عرضها لأن لها
                سايدباراً يقتطع جانباً، وهذا الموقع بلا سايدبار — فحصرُه يترك
                فراغين لا يملؤهما شيء.
              */}
              <main className="min-w-0 flex-1 rounded-card border border-line bg-surface p-5 shadow-sm">
                {children}
              </main>
            </div>
          </CartProvider>
        </AuthProvider>
      </body>
    </html>
  );
}
