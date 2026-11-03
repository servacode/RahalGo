/**
 * ══════════════════════════════════════════════════════════════════════
 * **معالمُ الخريطة — موضعٌ واحدٌ يُركّبها على كلّ خريطةٍ في الويب**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-١١-٠٣: «الخريطة ما فيها معالم كثيرة».)
 *
 * **من الخادم** (`/public/map/landmarks`): معالمُ قاعدتِنا — مساجدُ ومستشفياتٌ ومدارسُ وما أضافه
 * السائقون — ومتاجرُنا. **وبمفرداتِ بلاطاتِنا نفسِها** (`class`)، فتُستنسَخ طبقتا `poi-label` و
 * `poi-other-label` من النمط كما هما: **الأيقونةُ والخطُّ واللونُ والتكبيرُ نفسُها**، فلا يختلف معلمُنا
 * عن معلمِ البلاطة. **وطبقتُنا فوقَ طبقة البلاطة**، فالمكرَّرُ يحجبه التزاحمُ ولا يُرسم مرّتين.
 *
 * **وتُركَّب تحتَ مراسي العلامات** إن وُجدت — فالسائقون والطلباتُ فوقَها.
 */
import type { Map as MlMap, LayerSpecification } from "maplibre-gl";
import { apiBase } from "./runtimeconfig";

const SOURCE = "rahalgo-landmarks";
/** **طبقاتُ البلاطة المستنسَخة** — الأهمُّ من التقريب ١٥، وغيرُه من ١٦. */
const CLONED = ["poi-label", "poi-other-label"] as const;
const MARKER_ANCHOR = "rahalgo-marker-anchor";

/** **عنوانُ المعالم** — `stores:false` للخرائط التي ترسم متاجرَها طبقةً مستقلّة. */
export function landmarksUrl(stores = true): string {
  return `${apiBase()}/api/v1/public/map/landmarks?format=geojson${stores ? "" : "&stores=0"}`;
}

function layerIds(): string[] {
  return CLONED.map((id) => `rahalgo-lm-${id}`);
}

/**
 * **يركّب المعالمَ** — بعد `load` وقبل طبقات المُنادي. **ومرّتان لا تضرّان.**
 * **وحيثُ لا طبقةَ `poi-label` في النمط** (نمطٌ قديمٌ) لا يُرسم شيءٌ ولا يُرمى خطأ.
 */
export function installLandmarks(map: MlMap, opts: { stores?: boolean; visible?: boolean } = {}): void {
  try {
    if (map.getSource(SOURCE)) return;
    const layers = map.getStyle().layers as LayerSpecification[];
    const before = layers.some((l) => l.id === MARKER_ANCHOR) ? MARKER_ANCHOR : undefined;
    map.addSource(SOURCE, { type: "geojson", data: landmarksUrl(opts.stores ?? true) });
    for (const id of CLONED) {
      const src = layers.find((l) => l.id === id) as (LayerSpecification & Record<string, unknown>) | undefined;
      if (!src) continue;
      const { ["source-layer"]: _drop, ...rest } = src as Record<string, unknown>;
      void _drop;
      map.addLayer(
        {
          ...rest,
          id: `rahalgo-lm-${id}`,
          source: SOURCE,
          layout: { ...(src.layout as object), visibility: opts.visible === false ? "none" : "visible" },
        } as unknown as LayerSpecification,
        before,
      );
    }
  } catch (e) {
    // **سقوطُ المعالم لا يُسقط الخريطة.**
    console.error("landmarks", e);
  }
}

/** **يُظهر المعالمَ أو يُخفيها.** */
export function setLandmarksVisible(map: MlMap, visible: boolean): void {
  for (const id of layerIds()) {
    if (map.getLayer(id)) map.setLayoutProperty(id, "visibility", visible ? "visible" : "none");
  }
}
