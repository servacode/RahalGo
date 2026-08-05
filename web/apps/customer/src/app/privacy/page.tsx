"use client";

/** privacy — الهيكلُ المشترك، والنصُّ من المعجم. */

import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { IconLock } from "@rahalgo/ui";
import { LegalPage, type Block } from "../legal/LegalPage";

const m = getMessages(defaultLocale);
const L = m.site.legal;

export default function Page() {
  return (
    <LegalPage
      icon={IconLock}
      title={L.privacyTitle}
      subtitle={L.privacySubtitle}
      blocks={L.privacy as Block[]}
    />
  );
}
