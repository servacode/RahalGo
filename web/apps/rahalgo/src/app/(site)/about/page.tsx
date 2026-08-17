/**
 * **من نحن** — الهيكلُ المشترك، والنصُّ من المحرّك.
 *
 * (طلبُ المالك ٢٠٢٦-٠٨-١٣: «ولا تنسَ من نحن، لأنّ بدنا نرفعه على غوغل
 *  بلاي» — ثمّ: «التطبيقُ والويبُ نفسُ النموذج».)
 *
 * **وكانت في التطبيق ولا صفحةَ لها في الموقع** — فيقرأ السائقُ من نحن في
 * هاتفه ولا يجدها في الموقع، **ويُقرآن منصّتين.**
 *
 * **ونصُّها من `public/contact` كسائر الصفحات** — يُبدَّل من اللوحة بلا
 * نشر، **ويقرؤه التطبيقُ والموقعُ من مصدرٍ واحد** فلا يفترقان.
 */

import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { pageMeta } from "@/lib/seo";
import { LegalPage } from "../legal/LegalPage";
import { getContact, parseBlocks } from "../legal/contact";

const m = getMessages(defaultLocale);
const L = m.site.legal;


/** **بطاقةُ الصفحة في البحث** — عنوانٌ ووصفٌ خاصّان (٢٠٢٦-٠٨-١٧). */
export async function generateMetadata() {
  return pageMeta("seoAbout", "seoAboutDesc", "/about");
}

export default async function Page() {
  const contact = await getContact();
  return (
    <LegalPage
      kind="about"
      title={L.aboutTitle}
      subtitle={L.aboutSubtitle}
      blocks={parseBlocks(contact?.about_text?.trim() ?? "")}
      contact={contact}
    />
  );
}
