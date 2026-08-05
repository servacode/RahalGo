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
            {/* **ولا حشوةَ يميناً ويساراً** — (قاعدةُ المالك، قالها أربعَ
                مرّات): الصفحةُ تأخذ العرضَ كاملاً. **والعموديُّ في `main`
                وحدَه** لأنّ الشريطَ والفوترَ شريطان يبلغان الحافّة. */}
            <div className="flex min-h-screen flex-col">
              <Header />
              {/* ══════════════════════════════════════════════════════════
                  **المحتوى يقف على الصفحة — لا داخلَ صندوق**
                  ══════════════════════════════════════════════════════════

                  كان `<main className="surface-lit rounded-card bg-surface p-5">`
                  **يلفّ محتوى كلّ صفحةٍ في المنصة**، و`flex-1` تجعله يملأ
                  الشاشةَ طولاً — **ففراغُ الصفحة كان يُرى مستطيلاً أزرقَ
                  عملاقاً.** (شهده المالك ٢٠٢٦-٠٨-٠٦.)

                  # وأثرُه أنّ البطاقاتِ اختفت

                  **بطاقةُ الصنف `bg-surface` وأمُّها `bg-surface`** — لونٌ على
                  لونه، **فلا تُرى البطاقةُ جسماً إلّا بحدِّها.** ومن هنا جاء
                  الطابعُ الصندوقيُّ كلُّه: مستطيلاتٌ مرسومةٌ بخطوطٍ داخل
                  مستطيلٍ أكبر.

                  # ولا صفحةَ تعرّت بحذفه

                  **جُردت الثمانيَ عشرةَ صفحةً**: سبعٌ منها لا كرتَ فيها ظاهراً،
                  **وكلُّها تُفوّض إلى مكوّناتٍ مشتركةٍ تحمل بطاقاتِها**
                  (`AccountSettings` · `NotificationsPage` · `WalletPage` ·
                  `LegalPage`)، و`sso` صفحةُ تحويلٍ بلا محتوى.
                  ══════════════════════════════════════════════════════════ */}
              <main className="min-w-0 flex-1 py-4">{children}</main>
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
