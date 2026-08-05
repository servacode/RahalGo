import type { Metadata } from "next";
import { getMessages, getDir, defaultLocale } from "@rahalgo/i18n";
import { AuthProvider } from "@/lib/auth";
import { CartProvider } from "@/lib/cart";
import Header from "@/components/Header";
import Footer from "@/components/Footer";
import { FloatingCart } from "@/components/FloatingCart";
import { BottomNav, BottomNavSpacer } from "@/components/BottomNav";
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
              <main className="surface-lit min-w-0 flex-1 rounded-card bg-surface p-5">
                {children}
              </main>
              {/* **والشروطُ والمساعدةُ أسفلَ الصفحة** — حيث يُبحث عنها،
                  والشريطُ العلويُّ لما يُضغط كلَّ يوم. */}
              <Footer />
              {/* السلّة العائمة خارج الكرت: تُرافق التصفّح ولا تختفي بالتمرير */}
              <FloatingCart />
              {/* **وفراغٌ بارتفاع الشريط السفليّ** — وبلاه يختفي آخرُ سطرٍ
                  خلفه، وهو غالباً زرُّ الحسم. */}
              <BottomNavSpacer />
            </div>
            {/* **أقسامُ الزبون حيث يصل الإبهام** — على الجوّال وحدَه.
                (خارجَ غلاف الحشوة لأنّه يلتصق بحافّة الشاشة.) */}
            <BottomNav />
          </CartProvider>
        </AuthProvider>
      </body>
    </html>
  );
}
