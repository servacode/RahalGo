"use client";

/** terms — الهيكلُ المشترك، والنصُّ من المعجم. */

import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { IconNote } from "@rahalgo/ui";
import { LegalPage, type Block } from "../legal/LegalPage";

const m = getMessages(defaultLocale);
const L = m.site.legal;

export default function Page() {
  return (
    <LegalPage
      icon={IconNote}
      title={L.termsTitle}
      subtitle={L.termsSubtitle}
      blocks={L.terms as Block[]}
    />
  );
}
