/**
 * ══════════════════════════════════════════════════════════════════════
 * **تهيئةُ البيئة — وقتَ التشغيل لا وقتَ البناء**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (دورةُ ٧١و.)
 *
 * # العطبُ الذي أُغلق
 *
 * **وكانت `NEXT_PUBLIC_*` تُخبَز في الحزمة وقتَ البناء.** **فصورتان
 * من التزامٍ واحدٍ ليستا أثراً واحداً**: قِيس أنّ حزمةَ التجهيز فيها
 * `staging` مرّةً و`api.rahalgo.com` صفراً، **وحزمةَ الإنتاج بالعكس.**
 *
 * **وما يُختبَر هنا لا يُرقَّى هناك** — وهو نقضُ قاعدةِ «ابنِ مرّةً
 * ورقِّ الأثرَ عينَه».
 *
 * # وموضعٌ واحدٌ يُقرأ منه
 *
 * **ومصدران للشيء الواحد يفترقان يوماً.** **فلا `process.env` في
 * شاشة، ولا `window` غفلاً، ولا عنوانٌ مكتوبٌ في مكوّن** — **هذا
 * العقدُ لا غير.**
 *
 * # والعميلُ والخادمُ يقرآن من منبعين مختلفين — وهو مقصود
 *
 * **والخادمُ يعمل في الحاوية، فيقرأ بيئتَها عند كلّ طلب.**
 * **والمتصفّحُ لا بيئةَ له**، فيقرأ ما كتبه `/config.js` في
 * `window` — **وهو مسارٌ يُنفَّذ عند كلّ طلبٍ ولا يُخزَّن.**
 *
 * # والفشلُ مغلق
 *
 * **ولا ارتدادَ صامتاً إلى `localhost` ولا إلى تجهيزٍ ولا إنتاج.**
 * **ومن ترك ارتداداً وجّه عميلَ إنتاجٍ إلى تجهيز** — **وذاك أسوأُ من
 * شاشةِ خطأ**: نداءاتٌ حقيقيّةٌ تذهب إلى قاعدةٍ أخرى ولا يعلم أحد.
 */

/** PublicConfig **ما يُسمَح بأن يراه المتصفّح** — عناوينُ واسمُ بيئة. */
export interface PublicConfig {
  /** **عنوانُ المحرّك** — يُشتقّ منه مقبسُ البثّ كذلك. */
  apiUrl: string;
  /** **عنوانُ الموقع** — للوصف والفهرسة والروابط المطلقة. */
  siteUrl: string;
  /** **اسمُ البيئة** — `production` · `staging` · `development`. */
  environment: string;
  /** **نمطُ الخريطة** — وفارغٌ يعني «الخريطةُ غيرُ متاحة». */
  mapStyleUrl: string;
  /** **بلاطاتُ الخريطة** — وفارغٌ إن كان النمطُ يحملها. */
  mapTilesUrl: string;
}

/**
 * CONFIG_GLOBAL **اسمُ المتغيّر العامّ** — مركزيٌّ فلا يُكتب حرفاً في
 * موضعين.
 */
export const CONFIG_GLOBAL = "__RAHALGO_CONFIG__";

/**
 * readServerConfig **يقرأ بيئةَ الحاوية** — للخادم وحدَه.
 *
 * **ولا يُنادى من مكوّن عميل** — `process.env` هناك فارغةٌ بعد البناء،
 * **وهي تماماً ما كنّا نهرب منه.**
 */
export function readServerConfig(): PublicConfig {
  const env = process.env;
  return {
    apiUrl: (env.RAHALGO_API_URL ?? "").trim(),
    siteUrl: (env.RAHALGO_SITE_URL ?? "").trim(),
    environment: (env.RAHALGO_ENVIRONMENT ?? "").trim(),
    mapStyleUrl: (env.RAHALGO_MAP_STYLE_URL ?? "").trim(),
    mapTilesUrl: (env.RAHALGO_MAP_PMTILES_URL ?? "").trim(),
  };
}

/**
 * REQUIRED **ما لا تعمل الواجهةُ بدونه** — ويُسقط الردَّ صراحةً.
 *
 * **والخريطةُ ليست منها**: غيابُها حالٌ معلَنةٌ («الخريطةُ غيرُ
 * متاحة») **والصفحةُ تعمل.** **وغيابُ عنوان المحرّك ليس كذلك** —
 * **لا شيءَ يعمل، ولا يُخمَّن.**
 */
export const REQUIRED_CONFIG_KEYS = ["apiUrl", "siteUrl", "environment"] as const;

/** missingConfig **ما نقص من الواجب** — فارغةٌ تعني سليماً. */
export function missingConfig(c: PublicConfig): string[] {
  return REQUIRED_CONFIG_KEYS.filter((k) => !c[k]);
}

/**
 * SECRET_PATTERNS **ما لا يدخل تهيئةً عامّةً أبداً.**
 *
 * **وتهيئةُ التشغيل العامّةُ عامّةٌ بالتعريف** — يقرؤها كلُّ من فتح
 * الصفحة. **ويُحرَس بفحصٍ دائمٍ لا بانتباهِ مراجع.**
 */
export const SECRET_PATTERNS = [
  "SECRET",
  "PASSWORD",
  "CREDENTIAL",
  "PRIVATE",
  "JWT",
  "TOKEN",
  "_KEY",
] as const;

/** clientConfig **ما كتبه `/config.js` في المتصفّح.** */
export function clientConfig(): PublicConfig | null {
  if (typeof window === "undefined") return null;
  const v = (window as unknown as Record<string, unknown>)[CONFIG_GLOBAL];
  if (!v || typeof v !== "object") return null;
  return v as PublicConfig;
}

/**
 * config **العقدُ الذي تقرؤه الشاشاتُ كلُّها.**
 *
 * **ويعمل في الجانبين**: المتصفّحُ من `window`، والخادمُ من بيئته.
 * **فلا تُكتب فروعٌ في كلّ شاشة.**
 */
export function config(): PublicConfig {
  return clientConfig() ?? readServerConfig();
}

/** apiUrl **عنوانُ المحرّك** — والمنادي لا يعرف من أين جاء. */
export function apiUrl(): string {
  return config().apiUrl;
}

/**
 * wsUrl **عنوانُ قناة البثّ** — مُشتَقٌّ من عنوان المحرّك.
 *
 * **ولا متغيّرٌ ثانٍ له** — **ومن ضبط أحدَهما ونسي الآخرَ بنى بثّاً
 * يشير إلى بيئةٍ غيرِ التي تخدم نداءاته.**
 */
export function wsUrl(path = "/api/v1/ws"): string {
  return apiUrl().replace(/^http/, "ws") + path;
}

/** siteUrl · environment · isStaging — بقيّةُ العقد. */
export function siteUrl(): string {
  return config().siteUrl;
}
export function environment(): string {
  return config().environment;
}
export function isStaging(): boolean {
  return environment() === "staging";
}
