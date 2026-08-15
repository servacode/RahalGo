import type { Metadata } from "next";
import { getMessages, getDir, defaultLocale, withPlatform } from "@rahalgo/i18n";
import { PlatformProvider, fetchPlatform } from "@rahalgo/ui";
import { PasswordGate } from "@rahalgo/auth";
import { AuthProvider } from "@/lib/auth";
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
    // ══════════════════════════════════════════════════════════════
    // **وعنوانُ التبويب اسمُ المنصّة في كلّ صفحة**
    // ══════════════════════════════════════════════════════════════
    //
    // (قرارُ المالك ٢٠٢٦-٠٨-١٥: «اتركها فقط رحّال غو بكلّ الصفحات».)
    //
    // **وكان قالباً يُلحق الاسمَ بعنوان الصفحة** (`تواصل معنا | رحّال
    // غو`) — **وعنوانُ التبويب يُقرأ في شريطٍ ضيّقٍ فيه عشرُ تبويبات**:
    // ما يزيد على كلمتين يُقصّ، **فيبقى الجزءُ الذي لا يدلّ.**
    //
    // **والاسمُ وحدَه يُعرف من حرفه الأوّل.**
    title: { default: name, template: `${name}` },
    description,
    openGraph: { title, description, siteName: name, locale: "ar_SY", type: "website" },
  };
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الغلافُ الجذر — ما يشترك فيه كلُّ من يفتح المنصّة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-١٠: «يجب أن يكون بابٌ واحدٌ للجميع وشاشةُ تسجيلٍ
 *  واحدةٌ للجميع بدون استثناء».)
 *
 * # ولماذا لا يعرف هذا الغلافُ الزبونَ
 *
 * **كان يحمل السلّةَ والشريطَ والفوترَ وفقّاعةَ المحادثة** — لأنّه كان
 * غلافَ تطبيق الزبون وحدَه. **والآن تسكن اللوحاتُ الخمسُ تحته**، وشريطُ
 * سوقٍ فوق لوحة الإدارة **عبثٌ يراه صاحبُه عطباً.**
 *
 * **فما يبقى هنا ثلاثةٌ لا رابعَ لها**:
 *
 *   - **هويّةُ المنصّة** — تُقرأ في الخادم مرّةً لكلّ صفحة، فتصل مع أوّل
 *     بايت ولا تومض العلامةُ فارغةً ثمّ تمتلئ. **وفشلُ جلبها لا يُسقط
 *     الموقع**: الاسمُ يسقط إلى المعجم والشعارُ إلى الحرف.
 *   - **الجلسة** (`AuthProvider`) — يقرؤها كلُّ دور.
 *   - **بوّابةُ الكلمة المؤقّتة** — من أعادت الإدارةُ تعيينَ كلمته يُجبَر
 *     على تبديلها **أيَّ دورٍ كان**. وكانت في أربع لوحاتٍ وتُنسى الخامسة —
 *     **وهنا لا تُنسى واحدة.**
 *
 * **وما عداه ينزل إلى غلاف قسمِه**: `(site)` للسوق، ولكلّ لوحةٍ غلافُها.
 */
export default async function RootLayout({ children }: { children: React.ReactNode }) {
  const brand = await fetchPlatform(API);
  return (
    <html lang={defaultLocale} dir={getDir(defaultLocale)}>
      <body className="flex min-h-screen flex-col">
        <PlatformProvider apiBase={API} initial={brand}>
          <AuthProvider>
            {/* **ولا تمسّ زائراً**: من لا حسابَ له يمرّ، ومن لا علَمَ عليه
                يمرّ — **والمحرّكُ يُقنّع العلَمَ حين يكون الخيارُ مُطفأً**
                (`security.force_password_change`، مُطفأٌ افتراضاً بقرار
                المالك)، فلا سطرَ هنا يقرأ إعداداً. */}
            <PasswordGate>{children}</PasswordGate>
          </AuthProvider>
        </PlatformProvider>
      </body>
    </html>
  );
}
