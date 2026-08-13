/**
 * **تعليماتُ السائق** — دورتُه التشغيليّة خطوةً خطوة.
 *
 * (أمرُ المالك ٢٠٢٦-٠٨-١٣: «التطبيقُ والويبُ نفسُ النموذج والتسميات
 *  والشكل والأفعال والأسماء».)
 *
 * # ولماذا صفحةٌ ثانيةٌ غيرُ `/help`
 *
 * **تلك أسئلةُ الزبون**: «كيف أطلب؟ اختر متجراً وأضِف إلى السلّة» —
 * تُقال لمن يشتري، **لا لمن يقود.** ومن فتحها وهو سائقٌ لم يجد جواباً
 * عن سؤاله الأوّل: **كيف أبدأ ورديّتي؟**
 *
 * **ونصُّها من المحرّك** (`page.driver_help_text`) — هو النصُّ نفسُه
 * الذي يقرؤه في التطبيق، **فلا تعليمتان لعملٍ واحد.**
 */

import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { LegalPage } from "../../(site)/legal/LegalPage";
import { getContact, parseBlocks } from "../../(site)/legal/contact";

const m = getMessages(defaultLocale);
const L = m.site.legal;

export default async function Page() {
  const contact = await getContact();
  return (
    <LegalPage
      kind="help"
      title={L.helpTitle}
      subtitle={L.driverHelpSubtitle}
      blocks={parseBlocks(contact?.driver_help_text?.trim() ?? "")}
      contact={contact}
    />
  );
}
