"use client";

/**
 * **هويّةُ المنصة — مصدرٌ واحدٌ حيٌّ لا نصٌّ في شيفرة.**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «اسم المنصة لا داعي له في التوب بار، فقط اترك
 *  لوغو المنصة — وطبعاً يرث اللوغو في كلّ مكان: السايدبار الخاصّ بالإدارة
 *  والمندوب والسائق والمتجر، وأعلى لوحة تسجيل الدخول والخروج واستعادة كلمة
 *  المرور، وفي الانتقال بتسجيل الدخول والخروج، وعلى كروت الطلبات، وبنماذج
 *  الطباعة. لا أريد أيَّ مكانٍ يُكتب فيه اسم المنصة بشكلٍ جامد — يجب أن
 *  يأتي من الإعدادات فقط».)
 *
 * # ما كان
 *
 * **الاسمُ كان في المعجم** (`common.appName`) — يُقرأ في اثنَي عشرَ موضعاً.
 * **والحرفُ كذلك** (`terms.brandInitial`) في ستّة. **واللوحاتُ الأربعُ تكتب
 * اسمَها بيدها**، وأربعتُها مختلفة: «رحّال غو» و«بوّابة المتجر» وعنوانا دخولِ
 * السائق والمندوب.
 *
 * **فمن بدّل الاسمَ من الإعدادات بدّله في الموقع وحدَه** — وبقيت اللوحاتُ
 * والفواتيرُ وشاشاتُ الدخول تقول الاسمَ القديم. **وهويّةٌ تُبدَّل في مكانٍ
 * وتبقى في اثني عشر ليست هويّة.**
 *
 * # والحرفُ يُشتقّ ولا يُكتب
 *
 * **`brandInitial` كان «ر» في المعجم** — فمنصّةٌ تسمّي نفسَها باسمٍ آخر تبقى
 * تحمل حرفَ الاسم القديم. **والحرفُ أوّلُ الاسم المضبوط، يُحسب ولا يُخزَّن.**
 *
 * # ونداءٌ واحدٌ للتطبيق لا لكلّ مكوّن
 *
 * **الشعارُ يُطلب في السايدبار والشريط وشاشة الدخول والطبقة الانتقاليّة
 * وبطاقة الطلب والفاتورة** — ستّةُ مواضعَ في الشاشة الواحدة. **ونداءٌ لكلّ
 * واحدٍ منها ستّةُ نداءاتٍ لِما لا يتغيّر.**
 */

import {
  createContext,
  useContext,
  useEffect,
  useState,
  type ReactNode,
} from "react";

export interface Platform {
  /** اسمُ المنصة من الإعدادات — **وفارغٌ يعني «لم يُضبط بعد»، لا اسماً بديلاً.** */
  name: string;
  /** رابطُ الشعار الجاهز — **وفارغٌ يُبقي الحرف.** */
  logo: string | null;
  /**
   * **هل بابُ رمز التحقّق مفتوح؟** (`auth.otp_login` في الإعدادات.)
   *
   * (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «جهّز بالإعدادات بلوحة الادمن خيار لإطفاء أو
   *  تشغيل تسجيل الدخول برمز التحقّق».)
   *
   * **ومكانُه هنا لا في نداءٍ ثانٍ**: شاشةُ الدخول تحتاجه قبل أن يكون هناك
   * حساب، **وهي تنادي الهويّةَ أصلاً** — ونداءان لسطرين رحلةٌ زائدةٌ في أوّل
   * ما يُفتح.
   *
   * **والافتراضُ `true`** — فلو تأخّر الردُّ أو سقط **يُعرض البابان ثمّ يُخفى
   * ما يجب**، ولا يُحرَم أحدٌ باباً بسبب شبكةٍ بطيئة.
   */
  otpLogin: boolean;
  /**
   * **خلفيّةُ شاشات الدخول** (`auth.background` في الإعدادات) — **وفارغٌ
   * يُبقي اللونَ وحدَه.**
   *
   * (قرارُ المالك ٢٠٢٦-٠٨-٠٦.)
   */
  authBg: string | null;
  /**
   * **شفافيّةُ الطبقة فوق الخلفيّة** (٠..١٠٠) — **وصفرٌ يعني بلا طبقة.**
   *
   * (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «ضع خياراً للتحكّم بها… شريط من ٠ إلى ١٠٠».)
   */
  authBgDim: number;
}

const EMPTY: Platform = { name: "", logo: null, otpLogin: true, authBg: null, authBgDim: 70 };
const PlatformContext = createContext<Platform>(EMPTY);

export function PlatformProvider({
  children,
  /**
   * **قيمةٌ مبدئيّةٌ من الخادم** — للتطبيقات التي تقرأ الهويّةَ في التصيير
   * الخادميّ (موقعُ الزبون). **فلا تومض العلامةُ فارغةً ثمّ تمتلئ.**
   */
  initial,
  /** أصلُ المحرّك — يختلف بين التطبيقات فيُحقن. */
  apiBase,
}: {
  children: ReactNode;
  initial?: Platform;
  apiBase: string;
}) {
  const [platform, setPlatform] = useState<Platform>(initial ?? EMPTY);

  useEffect(() => {
    // **ومن جاءته قيمةٌ مبدئيّةٌ لا يُنادي**: الخادمُ قرأها قبله.
    if (initial?.name || initial?.logo) return;
    let alive = true;
    fetch(`${apiBase}/api/v1/public/platform`)
      .then((r) => (r.ok ? r.json() : null))
      .then((j) => {
        // **وفشلُ النداء يُبقي الفراغَ ولا يخترع اسماً** — علامةٌ ناقصةٌ
        // أهونُ من علامةٍ كاذبة.
        if (alive && j?.data)
          setPlatform({
            name: j.data.name ?? "",
            logo: j.data.logo ?? null,
            otpLogin: j.data.otp_login !== false,
            authBg: j.data.auth_bg ?? null,
            authBgDim: typeof j.data.auth_bg_dim === "number" ? j.data.auth_bg_dim : 70,
          });
      })
      .catch(() => undefined);
    return () => {
      alive = false;
    };
  }, [apiBase, initial]);

  return <PlatformContext.Provider value={platform}>{children}</PlatformContext.Provider>;
}

export function usePlatform(): Platform {
  return useContext(PlatformContext);
}

/** أوّلُ حرفٍ من الاسم — **يُشتقّ ولا يُكتب في معجم.** */
export function brandLetter(name: string): string {
  return name.trim().charAt(0);
}

/**
 * **علامةُ المنصة — صورةٌ إن رُفعت، وإلّا أوّلُ حرفٍ من اسمها.**
 *
 * **ولا تُحذف حين لا شعارَ ولا اسم**: مربّعٌ فارغٌ في الشريط أسوأُ من حرف،
 * **وشريطٌ بلا علامةٍ يُقرأ صفحةً لم تُحمَّل.** فيبقى المربّعُ بلونه ويُملأ
 * حين يُضبط الاسم.
 */
export function BrandMark({
  size = 36,
  rounded = "control",
  className = "",
}: {
  size?: number;
  /** `control` للشريط والسايدبار · `card` للطبقات الكبيرة */
  rounded?: "control" | "card" | "badge";
  className?: string;
}) {
  const { name, logo } = usePlatform();
  const box = { width: size, height: size };
  const shape = rounded === "card" ? "rounded-card" : rounded === "badge" ? "rounded-badge" : "rounded-control";

  if (logo) {
    return (
      // eslint-disable-next-line @next/next/no-img-element
      <img
        src={logo}
        alt={name}
        style={box}
        className={`shrink-0 object-cover ${shape} ${className}`}
      />
    );
  }
  // **ولا مربّعَ ملوّنٌ فارغ**: منصّةٌ لم تَرفع شعاراً ولم تُسمِّ نفسَها
  // بعدُ **يبقى مكانُ علامتها محجوزاً بهدوء** — فلا ينزلق ما بجانبه حين
  // تصل، **ولا يُقرأ مربّعٌ صارخٌ فارغٌ عطباً.**
  const letter = brandLetter(name);
  return (
    <span
      style={{ ...box, fontSize: Math.round(size * 0.44) }}
      aria-hidden
      className={`flex shrink-0 items-center justify-center font-bold ${
        letter ? "bg-primary text-on-bright" : "bg-page"
      } ${shape} ${className}`}
    >
      {letter}
    </span>
  );
}
