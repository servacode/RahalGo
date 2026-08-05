/** help — الهيكلُ المشترك، والنصُّ من المعجم. */

import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { LegalPage, type Block } from "../legal/LegalPage";
import { getContact } from "../legal/contact";

const m = getMessages(defaultLocale);
const L = m.site.legal;

export default async function Page() {
  return (
    <LegalPage
      kind="help"
      title={L.helpTitle}
      subtitle={L.helpSubtitle}
      blocks={L.help as Block[]}
      contact={await getContact()}
    />
  );
}
