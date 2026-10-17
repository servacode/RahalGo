/**
 * **تخزينُ المتصفّح — وكلُّ قراءةٍ وكتابةٍ محميّة.**
 *
 * **ونافذةٌ خاصّةٌ في سفاري آيفون تردّ استثناءً عند الكتابة** — ومن لم
 * يحمِ النداءَ أسقط الشاشةَ كلَّها لأجل سلّةٍ لم تُحفظ. **فالفشلُ هنا
 * صامتٌ عمداً**: السلّةُ تعمل في الذاكرة وتُنسى عند الإغلاق، والشاشةُ تعيش.
 */

export function readJSON<T>(key: string, fallback: T): T {
  try {
    const raw = window.localStorage.getItem(key);
    if (!raw) return fallback;
    return JSON.parse(raw) as T;
  } catch {
    return fallback;
  }
}

export function writeJSON(key: string, value: unknown): void {
  try {
    window.localStorage.setItem(key, JSON.stringify(value));
  } catch {
    /* **وتخزينٌ محجوبٌ لا يُسقط شيئاً** — انظر أعلاه. */
  }
}

export function removeKey(key: string): void {
  try {
    window.localStorage.removeItem(key);
  } catch {
    /* انظر أعلاه */
  }
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **مفتاحُ المحاولة — للطلب لا للضغطة** (مرآةُ `Attempt.kt`)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **يُولَّد حين يبدأ الإرسالُ ويبقى ما دام لم ينجح** — فإعادةُ المحاولة
 * بعد انقطاعٍ تحمله نفسَه، **فيردّ المحرّكُ الطلبَ الأوّلَ بعينه** ولا
 * يُنشئ ثانياً. **ومفتاحٌ جديدٌ لكلّ ضغطةٍ لا يحمي من شيء.**
 *
 * **ويُحفظ في التخزين** — فمن أغلق الصفحةَ قبل وصول الردّ وعاد يحمل
 * المفتاحَ نفسَه.
 */
const ATTEMPT_PREFIX = "rahalgo_attempt_";

export function attemptKey(slot: string): string {
  const k = ATTEMPT_PREFIX + slot;
  const have = readJSON<string>(k, "");
  if (have) return have;
  const fresh =
    typeof crypto !== "undefined" && "randomUUID" in crypto
      ? crypto.randomUUID()
      : `${Date.now()}-${Math.random().toString(36).slice(2)}`;
  writeJSON(k, fresh);
  return fresh;
}

export function clearAttempt(slot: string): void {
  removeKey(ATTEMPT_PREFIX + slot);
}

/**
 * **أحُسم الطلبُ في الخادم؟** — ردٌّ برمزٍ يعني «قيل فيه»، فالمفتاحُ يُمحى.
 *
 * **وانقطاعٌ بلا ردٍّ يعني «لا ندري»** — والمفتاحُ يبقى لإعادة المحاولة.
 * (مرآةُ `isDecided` في أندرويد.)
 */
export function isDecided(err: unknown): boolean {
  const status = (err as { status?: number })?.status;
  return typeof status === "number" && status >= 400 && status < 500;
}
