/**
 * ═════════════════════════════════════════════════════════════════════
 * **‏/app — بابُ التطبيقات، وهو البابُ الوحيدُ لغير الإدارة**
 * ═════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٢٦: «الويب ما بدّي يظلّ لا تسجيل دخول مندوب ولا
 *  زبون ولا سائق ولا متجر — كلّ شخص عنده تطبيقه، والويب صار للإدارة
 *  وتصفّح الموقع فقط».)
 *
 * # ولماذا صفحةٌ لا تحويلٌ صامت
 *
 * **من فتح `‏/login` كان يجد نموذجاً** — فصار يجد سطراً يقول أين
 * يذهب. **والتحويلُ إلى المتجر مباشرةً يُقرأ عطباً.**
 *
 * # وهي أيضاً ما تسقط إليه بوّابةُ التحديث
 *
 * **`UpdateGate` في أندرويد يجرّب `market://` ثمّ يسقط إلى
 * `https://rahalgo.com/app`** — **وكان ذلك العنوانُ أربعمئةً وأربعة**
 * (قِيس ٢٠٢٦-٠٨-٢٦). فمن لا متجرَ على جهازه كان يُترك بلا مخرج.
 */

import type { Metadata } from "next";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { ButtonLink } from "@rahalgo/ui";

const m = getMessages(defaultLocale);
const A = m.site.appGate;

const PLAY = "https://play.google.com/store/apps/details?id=com.rahalgo.customer";

export const metadata: Metadata = {
  title: A.metaTitle,
  description: A.metaDescription,
};

export default function Page() {
  return (
    <main className="mx-auto flex max-w-xl flex-col gap-6 px-4 py-16 text-center">
      <h1 className="heading-page">{A.title}</h1>
      <p className="text-muted">{A.body}</p>
      <ButtonLink href={PLAY} className="mx-auto">
        {A.cta}
      </ButtonLink>
      <p className="text-muted text-sm">{A.note}</p>
    </main>
  );
}
