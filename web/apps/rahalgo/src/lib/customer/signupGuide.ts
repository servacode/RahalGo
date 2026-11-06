/**
 * ══════════════════════════════════════════════════════════════════════
 * **شرحُ إنشاء الحساب أوّلَ مرّة — مرآةُ `SignupGuide.kt`** (قرارُ المالك ٢٠٢٦-١١-٠٧)
 * ══════════════════════════════════════════════════════════════════════
 *
 * سهمٌ على «إنشاء حساب جديد» في الدخول، ثمّ فقاعةٌ فوق الحقل الحاليّ وحدَه
 * في التسجيل. **ومرّةً لكلّ متصفّح** — والتخزينُ محميّ: إن حُجب ظهر الشرحُ
 * ثانيةً ولا ضرر.
 *
 * **والمنطقُ صافٍ هنا** — يفحصه `scripts/check-signup-guide.mjs` بلا متصفّح.
 */

export type GuideSpot = "phone" | "code" | "name" | "password";
export type SignupGuideStep = "phone" | "code" | "details" | "taken";

/**
 * **أرقامُ رمز التسجيل** — كما يولّده المحرّك (`randomDigits(6)`)، وهو طولُ
 * `OtpInput` الافتراضيّ. **ويحرسه الفحصُ** مع المحرّك والتطبيق.
 */
export const SIGNUP_CODE_DIGITS = 6;

const KEY = "rahalgo_signup_guide_seen";

/**
 * **الفقاعةُ تتبع الخطوةَ الحقيقيّة** — الرقم ثمّ الرمز (إن طلبته المنصّة) ثمّ
 * البيانات: الحقلُ الملموسُ يغلب، وإلّا فالاسم ثمّ كلمة السرّ. **ولا حقلَ لرمز
 * الدعوة في الموقع** (يأتي من الرابط `?ref=`)، فلا فقاعةَ له.
 */
export function guideSpot(
  step: SignupGuideStep,
  nameFilled: boolean,
  passwordDone: boolean,
  focused: GuideSpot | null,
): GuideSpot | null {
  if (step === "phone") return "phone";
  if (step === "code") return "code";
  if (step !== "details") return null;
  if (focused === "name" || focused === "password") return focused;
  if (!nameFilled) return "name";
  if (!passwordDone) return "password";
  return null;
}

/** **كلمةُ السرّ تمّت** — بطولها الأدنى، وتأكيدُها مطابق. */
export function guidePasswordDone(password: string, password2: string, min: number): boolean {
  return password.length >= min && password2 === password;
}

/** **أرآه هذا المتصفّح؟** — وتخزينٌ محجوبٌ يقول «لا» فيظهر ثانية. */
export function guideSeen(storage: Pick<Storage, "getItem"> | undefined): boolean {
  try {
    return storage?.getItem(KEY) === "1";
  } catch {
    return false;
  }
}

/** **رآه — لا يعود.** والفشلُ صامت. */
export function markGuideSeen(storage: Pick<Storage, "setItem"> | undefined): void {
  try {
    storage?.setItem(KEY, "1");
  } catch {
    /* **تخزينٌ محجوب** — يظهر في الزيارة التالية، ولا ضرر. */
  }
}

/** **التخزينُ إن وُجد** — والوصولُ إليه نفسُه قد يرمي في سفاري الخاصّ. */
export function browserStorage(): Storage | undefined {
  try {
    return typeof window === "undefined" ? undefined : window.localStorage;
  } catch {
    return undefined;
  }
}
