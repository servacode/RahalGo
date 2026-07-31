/**
 * التنسيق المركزي للأرقام والتواريخ — مصدر واحد لكل التطبيقات.
 *
 * قرار: **الأرقام إنجليزية (0-9) في كل المشروع** لا عربية-هندية (٠-٩). نستعمل
 * محلّية "ar-SY-u-nu-latn": أرقام لاتينية بفواصل قياسية (,/.) مع الإبقاء على
 * أسماء الأشهر العربية (تموز…). هكذا يبقى النص عربياً والأرقام إنجليزية.
 *
 * ممنوع أن يكتب أي ملف `new Intl.NumberFormat("ar-SY")` — كان يولّد أرقاماً
 * عربية-هندية، وكان مكرّراً في 29 ملفاً. استعمل الدوال أدناه.
 */

const LOCALE = "ar-SY-u-nu-latn";

const numFmt = new Intl.NumberFormat(LOCALE);
const dateFmt = new Intl.DateTimeFormat(LOCALE, { dateStyle: "short" });
const dateTimeFmt = new Intl.DateTimeFormat(LOCALE, { dateStyle: "short", timeStyle: "short" });
const timeFmt = new Intl.DateTimeFormat(LOCALE, { timeStyle: "short" });
const longDateFmt = new Intl.DateTimeFormat(LOCALE, {
  day: "numeric",
  month: "long",
  year: "numeric",
});

/** رقم بفواصل آلاف إنجليزية: 211,234 */
export function fmtNum(n: number): string {
  return numFmt.format(n);
}

/** يحوّل أي مدخل تاريخ إلى Date (يقبل نصاً أو رقماً أو Date). */
function toDate(v: string | number | Date): Date {
  return v instanceof Date ? v : new Date(v);
}

/** تاريخ قصير: 31/7/2026 */
export function fmtDate(v: string | number | Date): string {
  return dateFmt.format(toDate(v));
}

/** تاريخ ووقت: 31/7/2026، 2:05 م */
export function fmtDateTime(v: string | number | Date): string {
  return dateTimeFmt.format(toDate(v));
}

/** وقت فقط: 2:05 م */
export function fmtTime(v: string | number | Date): string {
  return timeFmt.format(toDate(v));
}

/** تاريخ طويل بأشهر عربية وأرقام إنجليزية: 31 تموز 2026 */
export function fmtLongDate(v: string | number | Date): string {
  return longDateFmt.format(toDate(v));
}
