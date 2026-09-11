/**
 * ══════════════════════════════════════════════════════════════════════
 * **مصدرُ الخريطة — موضعٌ واحدٌ يُقرأ منه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (إغلاقُ `TD-WEB-RASTER-SOURCE`، قرارُ المالك ٢٠٢٦-٠٨-٢١.)
 *
 * # العيبُ الذي أُغلق
 *
 * **كانت الخريطةُ ترتدّ إلى `tile.openstreetmap.org` تلقائيّاً** بعد
 * أربعةِ أخطاءِ بلاطة — **ومع ضبط المصدر الخاصّ**، فأوّلُ انقطاعٍ
 * يُخرج الإنتاجَ إلى بنيةٍ عموميّةٍ لا نملكها ولا نضمنها.
 *
 * **والارتدادُ لم يكن غفلةً بل تصميماً** — ولذلك لا يكفي حذفُ نصّ.
 *
 * # وما صار
 *
 * **مصدرٌ واحدٌ يملكه رحّال غو، ولا بديلَ عنه.** فإن غاب: **حالُ
 * «الخريطةُ غيرُ متاحة»** — والصفحةُ تعمل، ولا نداءَ إلى أحد.
 */

import { mapStyleUrl, mapTilesUrl } from "./runtimeconfig";

/** **بادئةُ PMTiles** — كما في أندرويد حرفيّاً. */
const PMTILES = "pmtiles://";

/**
 * **نمطُ الخريطة** — من المرحلة ٦.
 *
 * **ويُولَّد من `maps/style/rahalgo.style.json`** بربطٍ واحدٍ
 * (`maps/scripts/bind-style.mjs`) — **ولا نسخةَ ثانيةٌ للويب تنحرف
 * عن أندرويد** (البند ١٠).
 */
// **ودالّةٌ لا ثابت** — **ثابتٌ على مستوى الملفّ يُخبَز وقتَ البناء**
// (دورةُ ٧١و)، **فتصير صورةُ التجهيز عاجزةً عن خرائط الإنتاج.**
export const MAP_STYLE_URL = (): string => mapStyleUrl();

/**
 * **مصدرُ البلاطات المتّجهة** — PMTiles واحدٌ مُصدَرٌ بنسخة.
 *
 * **ويبقى فارغاً إن كان النمطُ يحمله بنفسه** — وهو الحالُ في
 * `style.online.json`.
 */
export const MAP_TILES_URL = (): string => mapTilesUrl();

/**
 * **ما لا يجوز أن يُطلَب وقتَ التشغيل أبداً** — البند ٢٦.
 *
 * **ويُفحص في وقت البناء وفي الاختبار** — فلا يعود أحدٌ يكتبها
 * ارتداداً.
 */
export const FORBIDDEN_MAP_HOSTS = [
  "tile.openstreetmap.org",
  "tile.openstreetmap.de",
  "basemaps.cartocdn.com",
  "demotiles.maplibre.org",
  "api.maptiler.com",
  "api.mapbox.com",
  "tiles.stadiamaps.com",
] as const;

/** **أهذا مصدرٌ عموميٌّ ممنوع؟** */
export function isForbiddenMapSource(url: string): boolean {
  const u = url.toLowerCase();
  if (FORBIDDEN_MAP_HOSTS.some((h) => u.includes(h))) return true;
  // **ومرايا OSM بأسماءِ نطاقاتٍ فرعيّة** — `a.tile.` و`{s}.tile.`.
  return /\btile\.openstreetmap\./.test(u) || /[{[]s[}\]]\.tile\./.test(u);
}

/** **حالُ المصدر** — يُقرأ في الواجهة وفي الاختبار. */
export type MapSourceState =
  | { readonly ok: true; readonly styleUrl: string }
  | { readonly ok: false; readonly reason: "missing" | "forbidden" };

/**
 * **يحسم المصدرَ مرّةً واحدة.**
 *
 * **ولا يرتدّ أبداً** (البند ١٩): غيابُ الإعداد **يعطي حالَ عطلٍ
 * مضبوطة**، لا مصدراً عموميّاً.
 */
export function resolveMapSource(
  // **والافتراضُ يُحسَب عند النداء لا عند تحميل الوحدة** — دورةُ ٧١و:
  // **تهيئةُ التشغيل قد تصل بعد التحميل.**
  styleUrl: string = MAP_STYLE_URL(),
): MapSourceState {
  const url = styleUrl.trim();
  if (!url) return { ok: false, reason: "missing" };
  if (isForbiddenMapSource(url)) return { ok: false, reason: "forbidden" };
  return { ok: true, styleUrl: url };
}

/**
 * **أيحتاج النمطُ بروتوكولَ PMTiles؟**
 *
 * **ويُسأل عن النمط لا عن الإعداد** — فالمصدرُ داخلَه.
 */
export function needsPmtiles(styleText: string): boolean {
  return styleText.includes(PMTILES);
}

/**
 * **النسبُ المركزيّ** — البند ١٧.
 *
 * **ولا يُكتب في صفحة**: النمطُ يحمله في `sources.base.attribution`،
 * **وMapLibre يعرضه بنفسه.** وهذا احتياطٌ لمن لم يُحمَّل نمطُه بعد.
 */
export const MAP_ATTRIBUTION_FALLBACK =
  '© <a href="https://www.openstreetmap.org/copyright" rel="noreferrer" target="_blank">مساهمو OpenStreetMap</a>';
