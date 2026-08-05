/** terms — الهيكلُ المشترك، والنصُّ من المعجم. */

import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { LegalPage, type Block } from "../legal/LegalPage";
import { getContact } from "../legal/contact";

const m = getMessages(defaultLocale);
const L = m.site.legal;

export default async function Page() {
  return (
    <LegalPage
      kind="terms"
      title={L.termsTitle}
      subtitle={L.termsSubtitle}
      blocks={L.terms as Block[]}
      contact={await getContact()}
    />
  );
}
