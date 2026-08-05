/** privacy — الهيكلُ المشترك، والنصُّ من المعجم. */

import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { LegalPage, type Block } from "../legal/LegalPage";
import { getContact } from "../legal/contact";

const m = getMessages(defaultLocale);
const L = m.site.legal;

export default async function Page() {
  return (
    <LegalPage
      kind="privacy"
      title={L.privacyTitle}
      subtitle={L.privacySubtitle}
      blocks={L.privacy as Block[]}
      contact={await getContact()}
    />
  );
}
