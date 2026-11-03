/**
 * ══════════════════════════════════════════════════════════════════════
 * **أسماءُ الأحياء والشوارع المُقرَّة من عناوين الزبائن** — موضعٌ واحدٌ يُركّبها على كلّ خريطةٍ في الويب
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-١١-٠٣: «بالضبط من عناوين الزباين راح نستفيد من اسماء الاحياء والشوارع».)
 *
 * **من الخادم** (`/public/map/areas`): نقطةٌ لكلّ حيٍّ أقرّه المدير وخطٌّ لكلّ شارع. وطبقتان:
 *
 *   · **اسمُ الحيّ** — خطُّ «RahalGo Bold» أكبرُ من أسماء الطرق، من التقريب ١٢ حتى ١٧٫٥.
 *   · **اسمُ الشارع على خطّه** — «RahalGo Regular»، من التقريب ١٥.
 *
 * **والألوانُ من الثيم** (`map-label` و`map-label-halo` في كتلة `@theme static`) لا من نصّ. **والمزاحمةُ تحجب
 * المكرَّرَ مع أسماء البلاطة** (الحيُّ الموجودُ في البلاطة لا يُقترح أصلاً، والشارعُ المسمّى فيها كذلك).
 *
 * **وتُركَّب تحتَ مراسي العلامات** إن وُجدت — فالسائقون والطلباتُ فوقَها.
 */
import type { Map as MlMap, LayerSpecification } from "maplibre-gl";
import { apiBase } from "./runtimeconfig";
import { themeColor } from "./cssvar";

const SOURCE = "rahalgo-areas";
const HOOD = "rahalgo-area-hood";
const STREET = "rahalgo-area-street";
const MARKER_ANCHOR = "rahalgo-marker-anchor";

/** **عنوانُ الأسماء المُقرَّة.** */
export function areasUrl(): string {
  return `${apiBase()}/api/v1/public/map/areas`;
}

/**
 * **يركّب طبقتَي الأسماء** — بعد `load` وقبل طبقات المُنادي. **ومرّتان لا تضرّان.**
 * **وسقوطُها لا يُسقط الخريطة.**
 */
export function installAreaNames(map: MlMap, opts: { visible?: boolean } = {}): void {
  try {
    if (map.getSource(SOURCE)) return;
    const layers = map.getStyle().layers as LayerSpecification[];
    const before = layers.some((l) => l.id === MARKER_ANCHOR) ? MARKER_ANCHOR : undefined;
    const visibility = opts.visible === false ? "none" : "visible";
    const paint = {
      "text-color": themeColor("map-label"),
      "text-halo-color": themeColor("map-label-halo"),
    };
    map.addSource(SOURCE, { type: "geojson", data: areasUrl() });
    map.addLayer(
      {
        id: HOOD,
        type: "symbol",
        source: SOURCE,
        minzoom: 12,
        maxzoom: 17.5,
        filter: ["==", ["get", "kind"], "neighbourhood"],
        layout: {
          visibility,
          "text-field": ["get", "name"],
          "text-font": ["RahalGo Bold"],
          "text-size": ["interpolate", ["linear"], ["zoom"], 12, 13, 15, 17, 17, 20],
          "text-max-width": 8,
          "text-padding": 6,
        },
        paint: { ...paint, "text-halo-width": 2 },
      },
      before,
    );
    map.addLayer(
      {
        id: STREET,
        type: "symbol",
        source: SOURCE,
        minzoom: 15,
        filter: ["==", ["get", "kind"], "street"],
        layout: {
          visibility,
          "symbol-placement": "line",
          "symbol-spacing": 380,
          "text-field": ["get", "name"],
          "text-font": ["RahalGo Regular"],
          "text-size": ["interpolate", ["linear"], ["zoom"], 15, 12, 17, 14, 19, 16],
          "text-rotation-alignment": "map",
          "text-padding": 10,
        },
        paint: { ...paint, "text-halo-width": 1.8 },
      },
      before,
    );
  } catch (e) {
    console.error("area names", e);
  }
}

/** **يُظهر الأسماءَ أو يُخفيها.** */
export function setAreaNamesVisible(map: MlMap, visible: boolean): void {
  for (const id of [HOOD, STREET]) {
    if (map.getLayer(id)) map.setLayoutProperty(id, "visibility", visible ? "visible" : "none");
  }
}
