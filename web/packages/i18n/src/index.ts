/**
 * نظام اللغة المركزي — كل النصوص الظاهرة للمستخدم تعيش هنا حصراً.
 * العربية هي الأساس؛ إضافة لغة جديدة = ملف JSON واحد بنفس البنية + سطر في locales.
 * (GROUND-RULES §1.1)
 */
import ar from "./locales/ar.json";

/** بنية الرسائل مشتقة من العربية — أي لغة أخرى يجب أن تطابقها بالكامل */
export type Messages = typeof ar;
export type Locale = keyof typeof locales;

export const locales = {
  ar: { messages: ar, dir: "rtl", label: "العربية" },
} as const satisfies Record<string, { messages: Messages; dir: "rtl" | "ltr"; label: string }>;

export const defaultLocale: Locale = "ar";

export function getMessages(locale: Locale): Messages {
  return locales[locale].messages;
}

export function getDir(locale: Locale): "rtl" | "ltr" {
  return locales[locale].dir;
}

export { fmtNum, fmtRef, fmtDate, fmtDateTime, fmtTime, fmtLongDate } from "./format";

/**
 * ثوانٍ إلى «م:ث» — لعدّادٍ تنازليّ.
 *
 * **بخانتين للثواني دائماً**: «١:٥» تُقرأ دقيقةً وخمسَ ثوانٍ أو خمسين، **ورقمٌ
 * يحتمل قراءتين في عدّادٍ ينقضي أسوأُ من لا عدّاد.**
 */
export function fmtClock(totalSeconds: number): string {
  const s = Math.max(0, Math.floor(totalSeconds));
  return `${Math.floor(s / 60)}:${String(s % 60).padStart(2, "0")}`;
}

/**
 * **حاقنُ اسم المنصة في نصٍّ من المعجم.**
 *
 * (قاعدةُ المالك: «لا أريد أن تكتب اسمَ المنصة بأيّ مكانٍ أبداً».)
 *
 * **كان الاسمُ مكتوباً في واحدٍ وعشرين مفتاحاً**: «فاتورة صادرة آلياً عن
 * منصة رحّال» · «من رحّال غو» · وعناوينُ الخمسة. **ومن بدّل الاسمَ من
 * الإعدادات بدّل الشريطَ والشعارَ وبقيت فاتورتُه تحمل اسماً آخر.**
 *
 * # ولماذا هنا لا في `platform.tsx`
 *
 * **ذاك ملفُّ عميلٍ** (`"use client"`) — **وعنوانُ الصفحة يُحسب في الخادم.**
 * وأوّلُ موضعٍ وضعتُها فيه رمى: «Attempted to call withPlatform() from the
 * server but withPlatform is on the client».
 *
 * **وهي دالّةُ نصٍّ لا حالةَ لها** — فموطنُها حزمةُ النصوص، **يقرؤها
 * الطرفان.** (وهو الدرسُ نفسُه الذي وُلد منه `platform-server.ts`.)
 */
export function withPlatform(text: string, name: string): string {
  return text.replace(/\{platform\}/g, name || ar.common.appName);
}

/**
 * ══════════════════════════════════════════════════════════════════════
 * **ترجمةُ خطأِ الخادم — بيتُها المعجم**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (شهده المالك ٢٠٢٦-٠٨-١١: شاشةُ توثيق الواتساب تعرض رسالةً عامّةً وتُخفي
 *  سببَ الخادم الحقيقيّ.)
 *
 * **كانت في `@rahalgo/auth`** — وهي تعتمد `@rahalgo/ui`. **فشاشةٌ في `ui`
 * لا تستطيع استعمالَها** إلّا بدورةِ اعتماد. فكانت تكتب رسالةً عامّةً بيدها،
 * **وتُسكت الخادمَ وهو يقول السبب.**
 *
 * **وموضعُها الصحيحُ هنا**: تحويلُ مفتاحٍ إلى نصٍّ عربيٍّ شأنُ المعجم،
 * **والحزمتان كلتاهما تعتمدانه** فلا دورة.
 *
 * # ولا تعرف نوعَ الخطأ
 *
 * **تقرأ الشكلَ لا الصنف** (`body.message_key`) — **ولو اشترطت `ApiError`
 * لَاحتاجت حزمةَ العميل**، وعادت الدورةُ من بابٍ آخر.
 */
export function errorText(err: unknown, messages: Messages = locales[defaultLocale].messages): string {
  const errs = messages.errors as unknown as Record<string, string>;
  const key = (err as { body?: { message_key?: string } })?.body?.message_key;
  if (typeof key !== "string" || !key) return errs.internal ?? "";
  // **والمفتاحُ يُقرأ كاملاً ثمّ بآخر جزئه** — الخادمُ يرسل `errors.x`
  // و`auth.otpInvalid`، **والمعجمُ يعرف الاثنين في موضعين.**
  const direct = key.split(".").reduce<unknown>(
    (node, part) => (node && typeof node === "object" ? (node as Record<string, unknown>)[part] : undefined),
    messages as unknown,
  );
  if (typeof direct === "string" && direct) return direct;
  const last = key.split(".").pop() ?? "";
  return errs[last] || errs.internal || "";
}
