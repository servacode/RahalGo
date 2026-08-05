"use client";

/** help — الهيكلُ المشترك، والنصُّ من المعجم. */

import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { IconSupport } from "@rahalgo/ui";
import { LegalPage, type Block } from "../legal/LegalPage";

const m = getMessages(defaultLocale);
const L = m.site.legal;

export default function Page() {
  return (
    <LegalPage
      icon={IconSupport}
      title={L.helpTitle}
      subtitle={L.helpSubtitle}
      blocks={L.help as Block[]}
    />
  );
}
