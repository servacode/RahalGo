/**
 * **جلبُ هويّة المنصة في الخادم — لِتُرسم مع أوّل بايت.**
 *
 * (عطبٌ شهده المالك ٢٠٢٦-٠٨-٠٦: «اللوحاتُ الأربعُ لا ترث نفس الخلفيّة التي
 *  اتّفقنا أنّها ستكون خلفيّةَ المشروع كامل».)
 *
 * # ولماذا كان الفرق
 *
 * **موقعُ الزبون يقرأ الهويّةَ في الخادم** ويمرّرها قيمةً مبدئيّة — **فترسم
 * الخلفيّةُ والشعارُ مع أوّل رسمة.**
 *
 * **واللوحاتُ الأربعُ كانت تقرؤها في المتصفّح**: نداءٌ بعد التحميل، ثمّ
 * تحميلُ الصورة، ثمّ الطلاء. **فتُفتح اللوحةُ بتدرّجٍ عارٍ ثانيةً أو ثانيتين
 * ثمّ تظهر الخلفيّة** — ومن يفتحها ويغلقها بسرعةٍ لا يراها أصلاً.
 *
 * **وذلك يُقرأ «لا ترث»** وإن كانت ترث متأخّرة.
 *
 * # وهذا ملفُّ خادمٍ لا عميل
 *
 * **`platform.tsx` تبدأ بـ`use client`** — ودالّةٌ تُنادى في الخادم لا تسكن
 * ملفَّ عميل. **فصار الجلبُ هنا وحدَه**، ويستورده كلُّ غلافٍ من الخمسة.
 *
 * **ولا يُكرَّر في خمسة أغلفة**: كان في غلاف الزبون وحدَه، **وهذا بعينه ما
 * جعل الشعارَ يعمل عنده وينكسر في الأربع.**
 */

import type { Platform } from "./platform";

/** ما يردّه المحرّك — أسماءُ الحقول بصيغة الـAPI. */
interface Wire {
  name?: string;
  logo?: string | null;
  otp_login?: boolean;
  app_url?: string;
  auth_bg?: string | null;
  auth_bg_dim?: number;
  site_bg?: string | null;
  site_bg_dim?: number;
}

/**
 * **وفشلُ النداء يُرجع فراغاً ولا يخترع شيئاً** — علامةٌ ناقصةٌ أهونُ من
 * علامةٍ كاذبة، **والتدرّجُ المرسومُ يبقى خلفيّةً في الحالين.**
 *
 * **و`no-store` لأنّ الهويّةَ تُبدَّل من اللوحة** — وصفحةٌ مخزَّنةٌ تعرض
 * شعاراً حُذف.
 */
export async function fetchPlatform(apiBase: string): Promise<Platform> {
  const empty: Platform = {
    name: "",
    logo: null,
    otpLogin: true,
    appUrl: "",
    authBg: null,
    authBgDim: 70,
    siteBg: null,
    siteBgDim: 55,
  };
  try {
    const res = await fetch(`${apiBase}/api/v1/public/platform`, { cache: "no-store" });
    if (!res.ok) return empty;
    const j = (await res.json()) as { data?: Wire };
    const d = j.data ?? {};
    return {
      // **والمسارُ يبقى كما ورد** — `PlatformProvider` هو من يُحوّله إلى رابطٍ
      // كاملٍ لِمن جاء من الخادم ومن جاء من الشبكة سواءً.
      name: d.name || "",
      logo: d.logo ?? null,
      otpLogin: d.otp_login !== false,
      appUrl: typeof d.app_url === "string" ? d.app_url : "",
      authBg: d.auth_bg ?? null,
      authBgDim: typeof d.auth_bg_dim === "number" ? d.auth_bg_dim : 70,
      siteBg: d.site_bg ?? null,
      siteBgDim: typeof d.site_bg_dim === "number" ? d.site_bg_dim : 55,
    };
  } catch {
    // @empty-ok — انظر أعلاه: الفراغُ قرارٌ لا صمت.
    return empty;
  }
}
