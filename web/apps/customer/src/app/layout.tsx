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
            {/* **والحشوةُ تتنفّس بحجم الشاشة** — ثلاثةٌ على الجوّال حيث كلُّ بكسلٍ
                محسوب، **وستّةٌ على الحاسب** حيث الفراغُ هو ما يفصل الشاشةَ
                المصمَّمةَ عن الصفحة المحشوّة. */}
            {/* ══════════════════════════════════════════════════════════
                **والشريطُ خارجَ الحشوة** — (قرارُ المالك ٢٠٢٦-٠٨-٠٦.)
                ══════════════════════════════════════════════════════════

                حذفتُ حشوتَه وزواياه فصغُر، **وبقي محبوساً داخل غلافٍ يحشو
                الصفحةَ من جوانبها الثلاثة** — فيبقى شريطٌ منحسرٌ عن الحوافّ
                بزواياَ مربّعة، **وهو أسوأُ من اللوح المدوَّر الذي كان.**

                **والحابسُ هو البنيةُ لا الشريط**: عنصرٌ داخل أبٍ محشوٍّ لا
                يبلغ الحافّةَ مهما حُذف من حشوته هو. فخرج من الأب.

                **ويبقى المحتوى محشوّاً**: الحشوةُ انتقلت إلى ما تحته وحدَه،
                **فلا يلتصق النصُّ بحافّة الشاشة.**
                ══════════════════════════════════════════════════════════ */}
            <div className="flex min-h-screen flex-col">
              <Header />
              <div className="flex flex-1 flex-col p-3 sm:p-4 lg:p-6">
              {/* ══════════════════════════════════════════════════════════
                  **الموقعُ لم يعد بطاقةً واحدةً عملاقة**
                  ══════════════════════════════════════════════════════════

                  # ما كان

                  `<main className="surface-lit rounded-card bg-surface p-5">`
                  **يلفّ محتوى كلّ صفحةٍ في المنصة.** ومنطقُه المكتوبُ كان:
                  «اللوحاتُ تلفّ محتواها ببطاقة، فليفعل الموقعُ مثلَها كي
                  تُعرف المنصةُ من هيكلها».

                  **والقياسُ كان على شيءٍ لا يشبهه.** لوحةُ الإدارة جدولٌ في
                  ورقة: بطاقةٌ واحدةٌ فيها صفوف. **والسوقُ رفٌّ عليه بضاعة** —
                  والبضاعةُ هي الأجسام، والرفُّ لا يكون صندوقاً حولها.

                  # وأثرُه أنّ البطاقاتِ اختفت

                  **بطاقةُ الصنف `bg-surface` وأمُّها `bg-surface`** — لونٌ
                  واحدٌ على لونه. فلا تُرى البطاقةُ جسماً **إلّا بحدِّها**،
                  ومن هنا جاء الطابعُ الصندوقيُّ كلُّه: **مستطيلاتٌ مرسومةٌ
                  بخطوطٍ داخل مستطيلٍ أكبر.**

                  **والرفعُ والظلُّ اللذان أضفتُهما أمسِ لم يُريا أصلاً** —
                  لأنّ الجسمَ لم يكن يطفو على أرضٍ بل على لونِ نفسِه.

                  # وما صار

                  **المحتوى يقف على الأرض مباشرةً**، والبطاقةُ ترتفع عنها
                  فتُرى. **والحدُّ صار زينةً لا ضرورة.**

                  # والعرضُ محصور

                  كان كاملاً بحجّة «لا سايدبار فيه فحصرُه يترك فراغين».
                  **وهي حجّةُ لوحةِ تحكّمٍ لا حجّةُ سوق**: على شاشةٍ بألفٍ
                  وأربعمئة **تتمدّد خمسُ بطاقاتٍ حتّى تصير أشرطةً**، ويمشي
                  العنوانُ مئةً وأربعين حرفاً في السطر — **والعينُ تفقد أوّلَ
                  السطر قبل أن تبلغ آخرَه.**
                  ══════════════════════════════════════════════════════════ */}
              <main className="mx-auto w-full min-w-0 max-w-7xl flex-1">{children}</main>
              {/* **والشروطُ والمساعدةُ أسفلَ الصفحة** — حيث يُبحث عنها،
                  والشريطُ العلويُّ لما يُضغط كلَّ يوم. */}
              <Footer />
              {/* السلّة العائمة خارج الكرت: تُرافق التصفّح ولا تختفي بالتمرير */}
              <FloatingCart />
              {/* **وفراغٌ بارتفاع الشريط السفليّ** — وبلاه يختفي آخرُ سطرٍ
                  خلفه، وهو غالباً زرُّ الحسم. */}
              <BottomNavSpacer />
              </div>
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
