"use client";

/**
 * شاشةُ المصادقة للزبون — **جسدٌ واحدٌ لثلاثة روابط.**
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٠٦: «لا يوجد رابطُ تسجيلٍ يبقى ضمن تسجيل الدخول —
 *  هل هذا شيءٌ طبيعيّ أم يجب أن يكون هناك رابطٌ خاصّ لتسجيل حسابٍ جديد وصفحة
 *  استعادة؟» والجواب: لا، ويجب.)
 *
 * # لماذا لا يكفي وضعٌ داخل صفحةٍ واحدة
 *
 * **ورابطُ الدعوة كان يقع على النموذج الخطأ**: من دُعي ليُنشئ حساباً يصل
 * `‎/login?ref=CODE` **فيرى شاشةً تطلب كلمةَ مرورٍ لا يملكها** — وعليه أن
 * يجد «إنشاء حساب» بنفسه. **وهذا ضررٌ مقيسٌ لا احتمال.**
 *
 * **وزرُّ الرجوع كان يخرج من الصفحة** بدل أن يعود خطوةً واحدة. **والتحديثُ
 * يُضيّع الوضع.** **ولا يُقاس كم وصل التسجيلَ ممّن فتح الدخول.**
 *
 * # وثلاثةُ ملفّاتٍ لا ثلاثةُ نُسخ
 *
 * **الصفحاتُ الثلاثُ ستّةُ أسطرٍ كلٌّ منها** — تفتح هذا الجسدَ بوضعٍ مختلف.
 * **والمنطقُ كلُّه هنا مرّةً واحدة**: التحويلُ بالدور، ورمزُ الدعوة، والوجهةُ
 * بعد الدخول.
 *
 * # والتبديلُ داخل البطاقة يُبدّل الرابط
 *
 * **من ضغط «إنشاء حساب» لا تُعاد الصفحة** — تتبدّل البطاقةُ كما كانت، **إنّما
 * يتبعها المسار** (`replace` لا `push`): **فزرُّ الرجوع يعود إلى ما قبل
 * الشاشة لا يتنقّل بين أوضاعها.**
 */

import { Suspense, useCallback, useEffect, useRef } from "react";
import { useRouter, useSearchParams } from "next/navigation";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import { BootScreen } from "@rahalgo/ui";
import { LoginCard, routeByRole, safeNext, useAuth } from "@rahalgo/auth";

const m = getMessages(defaultLocale);

export type AuthMode = "password" | "otp" | "reset" | "signup";

/** المسارُ لكلّ وضعٍ — **مصدرٌ واحدٌ يمنع أن يفترق التوجيهُ عن الروابط.** */
const PATH: Record<AuthMode, string> = {
  password: "/login",
  otp: "/login",
  reset: "/forgot",
  signup: "/signup",
};

export default function AuthScreen({ mode }: { mode: AuthMode }) {
  return (
    <Suspense>
      <Screen mode={mode} />
    </Suspense>
  );
}

function Screen({ mode }: { mode: AuthMode }) {
  const { user, loading, enter } = useAuth();
  const router = useRouter();
  const params = useSearchParams();
  const next = safeNext(params.get("next"));
  // **ورمزُ من دعاه يأتي في الرابط** — لا يُكتب باليد ولا يُطلب منه.
  const referral = params.get("ref") ?? "";
  const sent = useRef(false); // التحويل مرة واحدة — لا يتكرر مع كل إعادة رسم

  // من هو داخل أصلاً لا يرى شاشة الدخول إطلاقاً: يُنقل فوراً إلى مكانه حسب دوره.
  // (لتبديل الحساب: الخروج من الشريط العلوي ثم العودة إلى /login)
  useEffect(() => {
    if (loading || !user || sent.current) return;
    sent.current = true;
    void routeByRole(user, next ?? undefined);
  }, [user, loading, next]);

  const follow = useCallback(
    (m2: AuthMode) => {
      const to = PATH[m2];
      if (to === PATH[mode]) return;
      // **ويُحفظ ما في الرابط**: رمزُ الدعوة والوجهةُ بعد الدخول.
      const q = params.toString();
      router.replace(q ? `${to}?${q}` : to);
    },
    [mode, params, router],
  );

  if (loading || user) {
    return (
      <BootScreen />
    );
  }

  return (
    <LoginCard
      methods="both"
      /* **والزبونُ وحدَه يُنشئ حسابَه بنفسه** — واللوحاتُ حساباتُها من
         المنصة. (أمرُ المالك ٢٠٢٦-٠٨-٠٦: «اختلافُ الروابط حسب كلّ واجهة».) */
      signup
      referral={referral}
      initialMode={mode}
      onModeChange={follow}
      onSuccess={async (u) => {
        // **`enter` لا `setUser`** — تضع الهويّةَ وترفع الطبقةَ الانتقاليّة
        // معاً. **و`routeByRole` قد تعبر إلى بوّابةٍ أخرى** (تحميلُ تطبيقٍ
        // كامل)، **وبينهما ثانيتان كانت البطاقةُ تقضيهما ساكنة.**
        enter(u);
        await routeByRole(u, next ?? undefined);
      }}
    />
  );
}
