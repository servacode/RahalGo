/**
 * ══════════════════════════════════════════════════════════════════════
 * **تهيئةُ التشغيل كما تراها الحِزَمُ المشتركة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (دورةُ ٧١و.)
 *
 * # ولماذا نسخةٌ هنا
 *
 * **والحِزَمُ لا تعرف `@/lib` في تطبيقٍ بعينه** — **ولا يجوز أن تعتمد
 * حزمةٌ مشتركةٌ على مسارٍ داخليٍّ لتطبيقٍ واحد.**
 *
 * **ولا حقيقةَ ثانيةً هنا**: **الاسمُ العامُّ واحدٌ** (`__RAHALGO_CONFIG__`)
 * **يكتبه `/config.js` في التطبيق** — **وهذه قراءةٌ منه لا مصدرٌ
 * بجانبه.** **ومن كتب قيمةً هنا بنى مصدراً ثانياً.**
 *
 * # ولا `process.env` — وهو بيتُ القصيد
 *
 * **و`NEXT_PUBLIC_*` تُخبَز في الحزمة وقتَ البناء** — **فصورةُ التجهيز
 * لا تعرف عنوانَ الإنتاج.** **فالقراءةُ من `window` وقتَ التشغيل.**
 */

/** **اسمٌ واحدٌ لا يُكتب حرفاً في موضعين.** */
export const CONFIG_GLOBAL = "__RAHALGO_CONFIG__";

/** RuntimeConfig **ما يُسمَح بأن يراه المتصفّح.** */
export interface RuntimeConfig {
  apiUrl: string;
  siteUrl: string;
  environment: string;
  mapStyleUrl: string;
  mapTilesUrl: string;
}

const EMPTY: RuntimeConfig = {
  apiUrl: "",
  siteUrl: "",
  environment: "",
  mapStyleUrl: "",
  mapTilesUrl: "",
};

/**
 * runtimeConfig **ما كتبه `/config.js`** — وفارغٌ على الخادم.
 *
 * **ولا ارتدادَ إلى عنوانٍ مخترَع**: **من ارتدّ صامتاً وجّه عميلَ
 * إنتاجٍ إلى تجهيز**، **والفراغُ حالٌ تُعلَن ولا تُخفى.**
 */
export function runtimeConfig(): RuntimeConfig {
  if (typeof window === "undefined") return EMPTY;
  const v = (window as unknown as Record<string, unknown>)[CONFIG_GLOBAL];
  if (!v || typeof v !== "object") return EMPTY;
  return { ...EMPTY, ...(v as Partial<RuntimeConfig>) };
}

/** apiBase **عنوانُ المحرّك** كما يراه المتصفّح. */
export function apiBase(): string {
  return runtimeConfig().apiUrl;
}

/** wsBase **قناةُ البثّ** — مُشتقّةٌ ولا متغيّرٌ ثانٍ لها. */
export function wsBase(path = "/api/v1/ws"): string {
  const base = apiBase();
  return base ? base.replace(/^http/, "ws") + path : "";
}

/** isStagingEnv **رايةُ التجهيز** — `P-0` البند ٤٥. */
export function isStagingEnv(): boolean {
  return runtimeConfig().environment === "staging";
}

/** mapStyleUrl · mapTilesUrl — وفارغٌ يعني «الخريطةُ غيرُ متاحة». */
export function mapStyleUrl(): string {
  return runtimeConfig().mapStyleUrl;
}
export function mapTilesUrl(): string {
  return runtimeConfig().mapTilesUrl;
}
