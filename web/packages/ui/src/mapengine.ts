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

import { NAME_AR_OVERRIDES } from "./mapNameAr";

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

/**
 * ══════════════════════════════════════════════════════════════════════
 * **العربيّةُ على الخريطة موصولةً من اليمين** (قرارُ المالك ٢٠٢٦-١٠-٠٤)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **وكانت أسماءُ الأماكن حروفاً مقطّعةً معكوسة** — MapLibre لا يشكّل
 * العربيّةَ ولا يقلب اتّجاهها بنفسه، **ويحتاج إضافةَ الاتّجاه.**
 *
 * **والإضافةُ من خادمنا لا من خارجه** — نسخةٌ محفوظةٌ في الأصول العامّة
 * (`@mapbox/mapbox-gl-rtl-text` 0.3.0، رخصة BSD-2، بصمتُها
 * `d1c69035…3f589e`). **ولا شبكةَ خارجيّة**: خريطةٌ تنتظر موقعاً لا نملكه
 * تسقط معه.
 *
 * **وتُضبط هنا وحدَه** — كلُّ خريطةٍ في اللوحة تمرّ بهذا الملفّ، فتُصلَح
 * كلُّها بسطرٍ واحد. **وتُضبط مرّةً في عمر الصفحة** — ضبطُها مرّتين يرمي.
 */
export const RTL_PLUGIN_URL = "/vendor/mapbox-gl-rtl-text-0.3.0.js";
let rtlSet = false;

/**
 * ══════════════════════════════════════════════════════════════════════
 * **كلُّ اسمٍ على الخريطة عربيّ** (قرارُ المالك ٢٠٢٦-١١-٠٣: «كلشي بالعربي لازم»)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **كانت بعضُ الشوارع تظهر لاتينيّةً** («Tell Abiad Street») لأنّ بلاطاتها بلا
 * `name:ar` والنمطُ كان يرتدّ إلى `name`.
 *
 * **فيُعاد كتابُ كلّ طبقةِ أسماءٍ عند التحميل**: `name:ar` ← `name_ar` ← جدولُ
 * التعريب (`mapNameAr.ts` المولَّد من `maps/config/name-ar-overrides.csv`) ←
 * `name:nonlatin` (اسمٌ غيرُ لاتينيّ) ← لا شيء. **ولا يُقرأ `name` الخامُ أبداً.**
 *
 * **ويُضبط الأحياءُ والشوارعُ الصغرى**: الأحياءُ طبقةٌ فوق القرى (أولويةُ وضعٍ أعلى)،
 * والشوارعُ الصغرى من التكبير ١٥. **وإن كان النمطُ المنشورُ قد حمل هذا كلَّه
 * (`name:nonlatin` في الحقل) فلا يُلمس.**
 */
type StyleLayer = {
  id: string;
  type: string;
  filter?: unknown;
  minzoom?: number;
  layout?: Record<string, unknown>;
  paint?: Record<string, unknown>;
  [k: string]: unknown;
};
type StyleJson = { layers?: StyleLayer[]; [k: string]: unknown };

const MAJOR_ROADS = ["motorway", "trunk", "primary", "secondary", "tertiary"];

function arabicNameExpr(): unknown[] {
  const match: unknown[] = ["match", ["to-string", ["get", "name"]]];
  for (const [latin, ar] of Object.entries(NAME_AR_OVERRIDES)) match.push(latin, ar);
  match.push(["get", "name:nonlatin"]);
  return ["coalesce", ["get", "name:ar"], ["get", "name_ar"], match];
}

const clsIn = (...v: string[]) => ["in", ["get", "class"], ["literal", v]];
const isNameField = (f: unknown) => {
  const t = JSON.stringify(f ?? "");
  return t.includes('"name') && !t.includes("housenumber");
};

/** **يعيد كتابةَ النمط عربيّاً** — لا يُغيّر الدخل، ويردّه كما هو إن كان عربيّاً أصلاً. */
export function arabicOnlyStyle<T extends StyleJson>(style: T): T {
  const layers = style.layers;
  if (!layers) return style;
  const symbols = layers.filter((l) => l.type === "symbol" && isNameField(l.layout?.["text-field"]));
  if (symbols.every((l) => JSON.stringify(l.layout?.["text-field"]).includes("name:nonlatin"))) {
    return style;
  }
  const expr = arabicNameExpr();
  const out: StyleLayer[] = [];
  for (const raw of layers) {
    const l = JSON.parse(JSON.stringify(raw)) as StyleLayer;
    if (l.type === "symbol" && isNameField(l.layout?.["text-field"])) {
      l.layout = { ...l.layout, "text-field": expr };
    } else {
      out.push(l);
      continue;
    }
    if (l.id === "road-label" && !layers.some((x) => x.id === "road-label-minor")) {
      const minor = JSON.parse(JSON.stringify(l)) as StyleLayer;
      minor.id = "road-label-minor";
      minor.minzoom = 15;
      minor.filter = ["!", clsIn(...MAJOR_ROADS)];
      minor.layout = { ...minor.layout, "text-padding": 4, "symbol-spacing": 280,
        "text-size": ["interpolate", ["linear"], ["zoom"], 15, 11, 17, 12, 19, 14] };
      l.filter = clsIn(...MAJOR_ROADS);
      l.layout = { ...l.layout, "text-padding": 6, "symbol-spacing": 320 };
      out.push(l, minor);
    } else if (l.id === "place-village" && !layers.some((x) => x.id === "place-neighbourhood")) {
      const hood = JSON.parse(JSON.stringify(l)) as StyleLayer;
      hood.id = "place-neighbourhood";
      hood.filter = clsIn("suburb", "neighbourhood", "quarter");
      hood.layout = { ...hood.layout, "text-padding": 3,
        "text-size": ["interpolate", ["linear"], ["zoom"], 11, 12, 14, 14, 17, 16],
        "symbol-sort-key": ["match", ["get", "class"], "suburb", 1, 2] };
      hood.paint = { ...hood.paint, "text-halo-width": 1.8 };
      l.filter = clsIn("village", "hamlet");
      l.layout = { ...l.layout,
        "symbol-sort-key": ["match", ["get", "class"], "village", 2, 3] };
      out.push(l, hood);
    } else {
      out.push(l);
    }
  }
  return { ...style, layers: out };
}

/**
 * **تُعترَض `setStyle` مرّةً واحدةً** — فتمرّ كلُّ خرائطِ اللوحة (وأيُّ خريطةٍ
 * جديدة) على التحويل دون لمس أيٍّ من منشئاتها.
 */
let styleHooked = false;
function hookArabicStyle(maplibre: typeof import("maplibre-gl")) {
  if (styleHooked) return;
  styleHooked = true;
  const proto = maplibre.Map.prototype as unknown as {
    setStyle: (style: unknown, options?: Record<string, unknown>) => unknown;
  };
  const original = proto.setStyle;
  proto.setStyle = function (this: unknown, style: unknown, options?: Record<string, unknown>) {
    const prior = options?.transformStyle as
      | ((prev: unknown, next: StyleJson) => StyleJson)
      | undefined;
    return original.call(this, style, {
      ...options,
      transformStyle: (prev: unknown, next: StyleJson) =>
        arabicOnlyStyle(prior ? prior(prev, next) : next),
    });
  };
}

export async function loadMapEngine(styleUrl: string) {
  if (!enginePromise) enginePromise = import("maplibre-gl");
  const maplibre = await enginePromise;
  hookArabicStyle(maplibre);
  // **ويُضبط قبل إنشاء أيّ خريطة** — وبعدها لا أثرَ له.
  if (!workerSet) {
    maplibre.setWorkerUrl(WORKER_URL);
    workerSet = true;
  }
  if (!rtlSet) {
    rtlSet = true;
    // **وفوريّةٌ لا كسولة** (`lazy = false`): خرائطُنا كلُّها عربيّةُ الأسماء،
    // فلا يُنتظَر أوّلُ نصٍّ — **وتُجلب قبل أن يُرسم حرفٌ مقطّع.**
    // **وسقوطُها لا يُسقط الخريطة** — تبقى الحروفُ كما كانت ويُكتب العطب.
    if (maplibre.getRTLTextPluginStatus() === "unavailable") {
      maplibre
        .setRTLTextPlugin(new URL(RTL_PLUGIN_URL, window.location.origin).href, false)
        .catch((e: unknown) => console.error("rtl-text plugin", e));
    }
  }
  if (!protocolPromise) protocolPromise = registerProtocol(maplibre, styleUrl);
  await protocolPromise;
  return maplibre;
}
