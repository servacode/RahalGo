"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **‏/portal — بابٌ واحدٌ يعرف لوحةَ من طرقه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (كشفه جردُ الإشعارات ٢٠٢٦-٠٨-١٣، بأمر المالك: «اعمل جرداً للإشعارات».)
 *
 * # العطبُ الذي كشفه الجرد
 *
 * **أربعةَ عشرَ إشعاراً في المحرّك تشير إلى `/portal`** — «طلبٌ جديد»
 * للمتجر، و«حوالةٌ في محفظتك» للمندوب، و«شكوى عليك»، و«أُغلق دوامُك».
 * **ولا مسارَ بهذا الاسم في الموقع أصلاً**: جذورُه `driver` و`store`
 * و`rep` و`dashboard`.
 *
 * **فمن ضغط خبرَه وقع على أربعمئةٍ وأربعة** — وهو أسوأُ من إشعارٍ بلا
 * رابط: **يُقرأ أنّ المنصّةَ معطوبة، لا أنّ الخبرَ بلا وجهة.**
 *
 * # ولماذا يُصلَح هنا لا في المحرّك
 *
 * **المحرّكُ لا يعرف لوحةَ من يُرسل إليه** — «شكوى عليك» تذهب إلى سائقٍ
 * أو متجرٍ أو مندوب، **ولكلٍّ لوحةٌ باسمٍ آخر.** ولو حُلَّت هناك
 * **لَاحتاج كلُّ موضعٍ أن يسأل عن أدوار المتلقّي** قبل أن يكتب رابطاً —
 * أربعةَ عشرَ سؤالاً عن شيءٍ يعرفه المتصفّحُ بلا سؤال.
 *
 * **و`/portal` كان قصداً صحيحاً بلا تنفيذ**: «لوحتُك أنت أيّاً كانت».
 * **فيُنفَّذ هنا** — صفحةٌ واحدةٌ تقرأ الأدوارَ وتحوّل، **وما تحتها يُلحق
 * كما هو**: `/portal/wallet` تصير `/driver/wallet` لسائقٍ و`/rep/wallet`
 * لمندوب.
 */

import { useEffect } from "react";
import { useParams, useRouter } from "next/navigation";
import { useAuth, portalFor } from "@rahalgo/auth";
import { LoadingState } from "@rahalgo/ui";

export default function PortalRedirect() {
  const { user, loading } = useAuth();
  const router = useRouter();
  const params = useParams();

  useEffect(() => {
    if (loading) return;
    /* **ومن لم يدخل بعدُ يُساق إلى الدخول ثمّ يعود** — لا يُترك على
       شاشةٍ فارغةٍ ولا يُرمى إلى الجذر: **الإشعارُ فُتح لسبب.** */
    const rest = params?.rest;
    const tail = Array.isArray(rest) ? rest.join("/") : (rest ?? "");
    const suffix = tail ? `/${tail}` : "";
    if (!user) {
      router.replace(`/login?next=${encodeURIComponent(`/portal${suffix}`)}`);
      return;
    }
    /* **وزبونٌ لا لوحةَ له** — `portalFor` تردّ فارغاً، وبيتُه الجذر. */
    const base = portalFor(user.roles) ?? "/";
    router.replace(base === "/" ? `/${tail}` : `${base}${suffix}`);
  }, [loading, user, params, router]);

  return <LoadingState />;
}
