/** terms — الهيكلُ المشترك، والنصُّ من المعجم. */

import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { LegalPage, type Block } from "../legal/LegalPage";
import { getContact, parseBlocks } from "../legal/contact";

const m = getMessages(defaultLocale);
const L = m.site.legal;

export default async function Page() {
  const contact = await getContact();
  /* **والنصُّ المحرَّرُ من اللوحة يغلب نصَّ المعجم** — وفارغُه يعيده.
     (طلبُ المالك ٢٠٢٦-٠٨-٠٩: صفحاتٌ ديناميّة.) */
  const edited = contact?.terms_text?.trim();
  return (
    <LegalPage
      kind="terms"
      title={L.termsTitle}
      subtitle={L.termsSubtitle}
      blocks={edited ? parseBlocks(edited) : (L.terms as Block[])}
      contact={contact}
    />
  );
}
