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
  password_min_length?: number;
  app_url?: string;
  auth_bg?: string | null;
  auth_bg_mobile?: string | null;
  auth_bg_dim?: number;
  site_bg?: string | null;
  site_bg_mobile?: string | null;
  site_bg_dim?: number;
  show_login?: boolean;
  show_shop?: boolean;
  support_phone?: string;
  social?: Record<string, unknown>;
  address?: string;
  location?: string;
}

/**
 * **وفشلُ النداء يُرجع فراغاً ولا يخترع شيئاً** — علامةٌ ناقصةٌ أهونُ من
 * علامةٍ كاذبة، **والتدرّجُ المرسومُ يبقى خلفيّةً في الحالين.**
 *
 * **و`no-store` لأنّ الهويّةَ تُبدَّل من اللوحة** — وصفحةٌ مخزَّنةٌ تعرض
 * شعاراً حُذف.
 */
/**
 * **قارئُ حسابات التواصل** — واحدٌ للخادم والمتصفّح.
 *
 * **وما ليس نصّاً يُقرأ فراغاً**: إعدادٌ حُذف أو خادمٌ قديمٌ لا يرسله
 * **يجب ألّا يكسر التذييل** — يُخفي أيقونةً لا غير.
 */
export function readSocial(v: unknown): {
  facebook: string;
  instagram: string;
  telegram: string;
  whatsapp: string;
} {
  const o = (v ?? {}) as Record<string, unknown>;
  const at = (k: string) => (typeof o[k] === "string" ? (o[k] as string) : "");
  return {
    facebook: at("facebook"),
    instagram: at("instagram"),
    telegram: at("telegram"),
    whatsapp: at("whatsapp"),
  };
}

export async function fetchPlatform(apiBase: string): Promise<Platform> {
  const empty: Platform = {
    name: "",
    logo: null,
    otpLogin: true,
    passwordMinLength: 8,
    appUrl: "",
    authBg: null,
    authBgMobile: null,
    authBgDim: 70,
    siteBg: null,
    siteBgMobile: null,
    siteBgDim: 55,
    // **والافتراضُ الظهور حتّى في الفراغ** — **بابٌ اختفى لأنّ نداءً سقط
    // عطبٌ يُقرأ في وجه أوّل زائر.**
    showLogin: true,
    showShop: true,
    supportPhone: "",
    social: { facebook: "", instagram: "", telegram: "", whatsapp: "" },
    address: "",
    location: "",
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
      passwordMinLength:
        typeof d.password_min_length === "number" && d.password_min_length > 0
          ? d.password_min_length
          : 8,
      appUrl: typeof d.app_url === "string" ? d.app_url : "",
      authBg: d.auth_bg ?? null,
      authBgMobile: d.auth_bg_mobile ?? null,
      authBgDim: typeof d.auth_bg_dim === "number" ? d.auth_bg_dim : 70,
      siteBg: d.site_bg ?? null,
      siteBgMobile: d.site_bg_mobile ?? null,
      siteBgDim: typeof d.site_bg_dim === "number" ? d.site_bg_dim : 55,
      showLogin: d.show_login !== false,
      showShop: d.show_shop !== false,
      supportPhone: typeof d.support_phone === "string" ? d.support_phone : "",
      social: readSocial(d.social),
      address: typeof d.address === "string" ? d.address : "",
      location: typeof d.location === "string" ? d.location : "",
    };
  } catch {
    // @empty-ok — انظر أعلاه: الفراغُ قرارٌ لا صمت.
    return empty;
  }
}
