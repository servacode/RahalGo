import type { Metadata } from "next";
import { getMessages, getDir, defaultLocale, withPlatform } from "@rahalgo/i18n";
import { PlatformProvider, fetchPlatform } from "@rahalgo/ui";
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
const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

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
  const title = withPlatform(m.site.appTitle, name);
  const description = withPlatform(m.site.appDescription, name);
  return {
    metadataBase: new URL(SITE),
    title: { default: title, template: `%s | ${name}` },
    description,
    openGraph: { title, description, siteName: name, locale: "ar_SY", type: "website" },
  };
}

/**
 * **هويّةُ المنصة تُقرأ في الخادم مرّةً لكلّ صفحة.**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «لا تنسَ إضافة هوية المنصة — الاسم واللوغو».)
 *
 * # ولماذا في الغلاف لا في الشريط
 *
 * **الشريطُ يظهر في كلّ صفحة** — فلو جلب الهويّةَ بنفسه لَنادى الخادمَ عند
 * كلّ تنقّل. **والغلافُ يُرسم في الخادم**، فتصل الهويّةُ مع أوّل بايت.
 *
 * # وفشلُ الجلب لا يُسقط الموقع
 *
 * **الاسمُ يسقط إلى المعجم والشعارُ إلى الحرف** — ومنصّةٌ لا تصل إعداداتُها
 * **يجب أن تبقى تعمل باسمها المكتوب**، لا أن تعرض شريطاً فارغاً.
 */

export default async function RootLayout({ children }: { children: React.ReactNode }) {
  const brand = await fetchPlatform(API);
  return (
    <html lang={defaultLocale} dir={getDir(defaultLocale)}>
      <body className="flex min-h-screen flex-col">
        {/* **الهويّةُ تُقرأ في الخادم وتُمرَّر مبدئيّةً** — فلا تومض
            العلامةُ فارغةً ثمّ تمتلئ. */}
        <PlatformProvider apiBase={API} initial={brand}>
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
              {/* ══════════════════════════════════════════════════════
                  **وحشوةٌ خفيفةٌ مركزيّةٌ — لا تُكتب في كلّ صفحة**
                  ══════════════════════════════════════════════════════

                  (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «البادينك هذا لازم يكون مركزي مشان
                  ما نعدّله بكلّ صفحة بشكلٍ منفصل».)

                  **وكنتُ كتبتُها في صفحتين منفصلتين** — التسوّق والإشعارات —
                  **وهي عينُ التكرار الذي يُحذّر منه**: قيمةٌ تُكتب في موضعين
                  تفترق يوماً، **ومن أراد تبديلَها يفتح ما لا يُحصى.**

                  **ولا في `PageContainer`**: نصفُ صفحات الموقع لا تستعملها
                  (السلّة والتسوّق والدخول والمساعدة والشروط) — **فالحشوةُ
                  هناك تترك النصفَ الآخر ملتصقاً بالحافّة.**

                  **و`<main>` يلفّ كلَّ صفحةٍ بلا استثناء.**

                  # وهي غيرُ الحشوة التي حُذفت

                  **المحذوفةُ كانت `p-3 sm:p-4 lg:p-6` مع حصرِ عرضٍ** —
                  تُضيّق الصفحةَ وتمنع المحتوى من ملء الشاشة. **وهذه اثنا عشرَ
                  بكسلاً تُبعد الحرفَ عن الحافّة** ولا تحصر شيئاً.
                  ══════════════════════════════════════════════════════ */}
              <main className="min-w-0 flex-1 px-3 py-4 sm:px-4">{children}</main>
              {/* **والشروطُ والمساعدةُ أسفلَ الصفحة** — حيث يُبحث عنها،
                  والشريطُ العلويُّ لما يُضغط كلَّ يوم. */}
              <Footer name={brand.name} />
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
        </PlatformProvider>
      </body>
    </html>
  );
}
