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
