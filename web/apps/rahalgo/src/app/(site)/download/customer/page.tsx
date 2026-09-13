/**
 * **‏/download/customer — صفحةُ الزبون** (`DLC`، ٢٠٢٦-٠٩-١٣)
 *
 * **ولا منطقَ هنا**: **الحالُ من المحرّك والعرضُ من `AppPage`** —
 * **وأربعُ صفحاتٍ تكتب كلٌّ منها عرضَها تفترق في الرابعة.**
 */

import type { Metadata } from "next";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { readRelease, downloadHref } from "@/lib/releases";
import { AppPage } from "@/components/download/AppCard";

const m = getMessages(defaultLocale);
const D = m.site.download;

export const dynamic = "force-dynamic";

export const metadata: Metadata = {
  title: D.apps.customer.name,
  description: D.apps.customer.short,
};

export default async function Page() {
  const release = await readRelease("customer");
  return <AppPage release={release} downloadUrl={downloadHref(release)} />;
}
