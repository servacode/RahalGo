/** privacy — الهيكلُ المشترك، والنصُّ من المعجم. */

import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { pageMeta } from "@/lib/seo";
import { LegalPage, type Block } from "../legal/LegalPage";
import { getContact, parseBlocks } from "../legal/contact";

const m = getMessages(defaultLocale);
const L = m.site.legal;


/** **بطاقةُ الصفحة في البحث** — عنوانٌ ووصفٌ خاصّان (٢٠٢٦-٠٨-١٧). */
export async function generateMetadata() {
  return pageMeta("seoPrivacy", "seoPrivacyDesc", "/privacy");
}

export default async function Page() {
  const contact = await getContact();
  /* **والنصُّ المحرَّرُ من اللوحة يغلب نصَّ المعجم** — وفارغُه يعيده.
     (طلبُ المالك ٢٠٢٦-٠٨-٠٩: صفحاتٌ ديناميّة.) */
  const edited = contact?.privacy_text?.trim();
  return (
    <LegalPage
      kind="privacy"
      title={L.privacyTitle}
      subtitle={L.privacySubtitle}
      blocks={edited ? parseBlocks(edited) : (L.privacy as Block[])}
      contact={contact}
    />
  );
}
