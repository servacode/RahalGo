/** الرئيسية — تُقدَّم من الخادم (SEO): المحتوى في HTML الأولي، والترشيح تفاعلي. */

import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { Alert, EmptyState, IconStore } from "@rahalgo/ui";
import HomeClient, { type HomeData } from "./HomeClient";

const m = getMessages(defaultLocale);
const API = process.env.NEXT_PUBLIC_API_URL ?? "http://localhost:8080";

export const dynamic = "force-dynamic";

export default async function HomePage() {
  let data: HomeData = { banners: [], categories: [], sections: [] };
  /**
   * **«لم أصل» غيرُ «وصلتُ فلم أجد».**
   *
   * كانت الصفحةُ تقول «لا يوجد اتصال بالإنترنت» **في الحالين** — فمتجرٌ أُفرغت
   * أقسامُه يُقرأ انقطاعَ شبكة. **والزبونُ يُغلق الصفحة ويتّهم هاتفَه**، ونحن
   * نبحث عن عطبٍ في الشبكة والعلّةُ في القائمة.
   *
   * **ووقعت فعلاً** (٢٠٢٦-٠٨-٠٣): أُطفئت أقسامُ المنصة لتهيئة تجربةٍ نظيفة،
   * **فقال الموقعُ «لا إنترنت» والخادمُ يردّ مئتين** — وذهب المالكُ يُعيد
   * تشغيل المشروع كلَّه.
   *
   * وهي عائلةُ «تُعلن ما لا تعلمه» نفسُها التي أخرجت `G-04`.
   */
  let reached = false;
  try {
    const res = await fetch(`${API}/api/v1/public/home`, { cache: "no-store" });
    reached = res.ok;
    const json = (await res.json()) as { data?: HomeData };
    if (json.data) data = json.data;
  } catch {
    /* الخادمُ غير متاح — وهذه وحدَها رسالةُ الانقطاع */
  }

  if (!reached) {
    // **والانقطاعُ لافتةٌ لا سطرٌ رماديّ.**
    //
    // **سطرٌ باهتٌ في وسط صفحةٍ فارغةٍ يُقرأ «لا يوجد شيء»** لا «لم أصل» —
    // وهي عينُ العائلة التي أخرجت هذا الحارسَ أصلاً.
    return (
      <div className="py-10">
        <Alert tone="warning" title={m.errors.offline}>
          {m.errors.offlineHint}
        </Alert>
      </div>
    );
  }
  // **وفراغُ الأقسام لا فراغُ المتاجر** — الزبونُ يتصفّح أقساماً.
  if (data.sections.length === 0) {
    // **وفراغُ الأقسام حالُ فراغٍ لا نصٌّ مرتجَل** — بالشكل الذي يراه الزبونُ
    // في كلّ شاشةٍ فارغةٍ من المنصة.
    return <EmptyState icon={IconStore} title={m.site.home.emptyTitle} />;
  }
  return <HomeClient initial={data} />;
}
