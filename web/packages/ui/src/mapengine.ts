/**
 * ══════════════════════════════════════════════════════════════════════
 * **تحميلُ محرّك الخريطة — مرّةً واحدةً في عمر الصفحة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (إغلاقُ `TD-WEB-RASTER-SOURCE`، ٢٠٢٦-٠٨-٢١.)
 *
 * **والمحرّكُ ثقيل** — فلا يُجَرّ إلّا عند أوّل خريطة.
 *
 * **وبروتوكولُ PMTiles يُسجَّل مرّةً واحدة**: تسجيلُه مرّتين يرمي،
 * **وصفحتان فيهما خريطتان تُحمَّلان معاً** — فالحارسُ لازم.
 */

let enginePromise: Promise<typeof import("maplibre-gl")> | null = null;
let protocolPromise: Promise<void> | null = null;
let workerSet = false;

/** **أيحتاج النمطُ بروتوكولَ PMTiles؟** */
function usesPmtiles(styleText: string): boolean {
  return styleText.includes("pmtiles://");
}

async function registerProtocol(
  maplibre: typeof import("maplibre-gl"),
  styleUrl: string,
): Promise<void> {
  // **ويُسأل النمطُ أوّلاً** — فلا تُحمَّل مكتبةٌ لا تلزم.
  const res = await fetch(styleUrl, { cache: "force-cache" });
  if (!res.ok) throw new Error(`style ${res.status}`);
  const text = await res.text();
  if (!usesPmtiles(text)) return;
  const { Protocol } = await import("pmtiles");
  // ══════════════════════════════════════════════════════════════════
  // **و`metadata` لازمة — وبلاها خريطةٌ فارغةٌ بلا خطأ**
  // ══════════════════════════════════════════════════════════════════
  //
  // **قسمُ البيانات الوصفيّة يحمل `vector_layers`** — وهي ما يربط
  // طبقاتِ النمط بطبقات المصدر.
  //
  // **وبلاها لا يكتمل المصدرُ ولا تُطلَب بلاطةٌ واحدة** — **ولا
  // يُرمى خطأ.** (قِيس ٢٠٢٦-٠٨-٢١: ترويسةٌ تُقرأ ثمّ صمت.)
  const protocol = new Protocol({ metadata: true });
  maplibre.addProtocol("pmtiles", protocol.tile);
}

/**
 * **يردّ المحرّكَ جاهزاً** — والبروتوكولُ مسجَّلٌ إن لزم.
 *
 * **ويرمي إن تعذّر النمط** — والمُنادي يعرض حالَ عطلٍ مضبوطة
 * **ولا يرتدّ إلى مصدرٍ عموميّ.**
 */
/**
 * ══════════════════════════════════════════════════════════════════════
 * **عاملُ MapLibre — يُعطى مساراً صريحاً**
 * ══════════════════════════════════════════════════════════════════════
 *
 * **السببُ الجذريُّ الذي أُغلق** (قِيس ٢٠٢٦-٠٨-٢١):
 *
 * **MapLibre يُقطّع البلاطاتِ في عاملٍ (Web Worker)**، ويترك بناءَه
 * للحُزَم. **وTurbopack لا يبنيه** — فلا يُنشَأ العاملُ ولا يُرمى خطأ.
 *
 * **والأثرُ خادعٌ تماماً**: النمطُ يصل، والأيقوناتُ تصل، وترويسةُ
 * الأرشيف وقسمُه الوصفيُّ يُقرآن — **ثمّ صمت.** لا بلاطةَ ولا حرفَ
 * ولا خطأ، **والخريطةُ فارغةٌ كأنّ المصدرَ عطبان.**
 *
 * **وأُثبت بالفصل**: مصدرُ `geojson` — الذي يحتاج العاملَ ولا يحتاج
 * بروتوكولاً — **لم يُحمَّل هو الآخر.** فالعطبُ في العامل لا في
 * PMTiles.
 *
 * # ولماذا ملفّان
 *
 * **`maplibre-gl-worker.mjs` شِبهٌ (١٨ كيلو) يستورد `maplibre-gl-shared.mjs`
 * (٤٧٦ كيلو) بمسارٍ نسبيّ** — فلا يكفي نسخُ الأوّل وحدَه.
 *
 * **وينسخهما `scripts/copy-map-worker.mjs` قبل البناء** — فلا يُنسى
 * ملفٌّ عند ترقية.
 */
const WORKER_URL = "/maplibre-gl-worker.mjs";

export async function loadMapEngine(styleUrl: string) {
  if (!enginePromise) enginePromise = import("maplibre-gl");
  const maplibre = await enginePromise;
  // **ويُضبط قبل إنشاء أيّ خريطة** — وبعدها لا أثرَ له.
  if (!workerSet) {
    maplibre.setWorkerUrl(WORKER_URL);
    workerSet = true;
  }
  if (!protocolPromise) protocolPromise = registerProtocol(maplibre, styleUrl);
  await protocolPromise;
  return maplibre;
}
