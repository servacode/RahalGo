"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **لوحُ خريطة العمليات — طبقاتٌ تُوصَف لا تُبرمَج**
 * ══════════════════════════════════════════════════════════════════════
 *
 * # ولماذا سجلُّ طبقاتٍ لا مكوّنٌ لكلّ واحدة
 *
 * **عشرُ طبقاتٍ في الخريطة** (سائقون · متاجرُ · طلباتٌ · تغطيةٌ · طلباتُ
 * تغطيةٍ · فروعٌ · مناطقُ تشغيليّةٌ · مندوبون · كثافةٌ · فرص). **ومن كتب
 * لكلٍّ منها مكوّناً كتب عشرَ نسخٍ من الشيفرة نفسِها** — وأضاف الحاديةَ
 * عشرةَ بنسخةٍ ثانيةَ عشرةَ.
 *
 * **فالطبقةُ هنا وصفٌ**: معرِّفٌ · نوعٌ · بياناتٌ `GeoJSON` · لونٌ ·
 * أتُجمَّع؟ **واللوحُ يبني ما يلزم ويهدم ما زال.**
 *
 * # والرموزُ صورٌ تُرسم مرّةً (قرارُ المالك ٢٠٢٦-١٠-٠٥)
 *
 * **دائرةٌ بلون الحال وفيها رمزٌ أبيضُ من مجموعة أيقوناتنا** — تُولَّد في
 * المتصفّح عند الإقلاع بضعف الكثافة (`pixelRatio: 2`) فتبقى حادّةً، **ولا
 * نصَّ فيها فلا اتّجاهَ يُقلب.** وطبقةُ `symbol` تختار صورتَها من خاصّة
 * `icon` في المعلَم.
 *
 * # والتجميعُ لازمٌ لا زينة (البند ٣٠)
 *
 * **مئتا سائقٍ وألفُ متجرٍ نقاطٌ خامٌّ تُجمّد المتصفّح** — **والتجميعُ
 * في المصدر لا في الرسم**، فـMapLibre يفعله في العامل.
 *
 * # ولا تتحرّك الكاميرا وحدَها
 *
 * **لا `flyTo` إلّا بنقرةٍ صريحة** — رقمُ «عالق» أو فقّاعةُ تجميع. **ومن كان
 * ينظر إلى حيٍّ فانتُزع منه لأنّ بياناتٍ تبدّلت فقد ما كان يبحث عنه.**
 *
 * # ولا مصدرَ خرائطَ عموميّ
 *
 * **`resolveMapSource` يحسم** — **ولا ارتدادَ إلى مصدرٍ عموميٍّ البتّة**
 * (`TD-WEB-RASTER-SOURCE`، ٢٠٢٦-٠٨-٢١). **وغيابُ الإعداد حالُ عطلٍ
 * مضبوطة، لا خريطةٌ من بنيةٍ لا نملكها ولا نضمنها.**
 */

import type * as maplibregl from "maplibre-gl";
import { createElement, useEffect, useRef, useState, useCallback, type ComponentType } from "react";
import { flushSync } from "react-dom";
import { createRoot } from "react-dom/client";
import { resolveMapSource, MAP_ATTRIBUTION_FALLBACK } from "@rahalgo/ui/mapconfig";
import { loadMapEngine } from "@rahalgo/ui/mapengine";
import { themeColor } from "@rahalgo/ui";

/** **مركزُ الرقّة** — حيث تعمل المنصّة. */
const RAQQA: [number, number] = [39.0079, 35.9528];

export type FeatureCollection = {
  type: "FeatureCollection";
  features: Array<{
    type: "Feature";
    geometry: unknown;
    properties: Record<string, unknown>;
  }>;
};

/** **وصفُ طبقةٍ واحدة.** */
export interface LayerSpec {
  id: string;
  /**
   * `point` نقاطٌ · `symbol` رموزٌ بصورة الحال (خاصّةُ `icon`) · `bubble`
   * فقّاعاتُ عددٍ (خاصّةُ `count`) · `fill` مضلَّعاتٌ · `circle-m` دوائرُ
   * بالمتر · `heat` كثافة · `line` خطوط
   */
  kind: "point" | "symbol" | "bubble" | "fill" | "circle-m" | "heat" | "line";
  data: FeatureCollection;
  /** **اللونُ تعبيرٌ أو نصّ** — والتعبيرُ يقرأ خاصّةً من المعلَم. */
  color: unknown;
  /** **نصفُ القطر بالمتر** لطبقات `circle-m` — اسمُ خاصّة. */
  radiusField?: string;
  /** **أيُجمَّع؟** — للنقاط الكثيرة. */
  cluster?: boolean;
  /** **الوزنُ للكثافة.** */
  weightField?: string;
  visible: boolean;
  /** **ترتيبُ الرسم** — الأصغرُ أسفل. */
  order: number;
  /** **عرضُ الخطّ بالبكسل** لطبقات `fill` و`line` — افتراضُه ٢. */
  lineWidth?: number;
  /** **شفافيّةُ الحشو** لطبقات `fill` — افتراضُها ٠٫١٨. */
  fillOpacity?: number;
  /** **خطٌّ متقطّع؟** لطبقات `line` — افتراضُه نعم. */
  dashed?: boolean;
}

/** **صورةُ رمزٍ تُسجَّل في الخريطة** — اسمُها قيمةُ خاصّة `icon`. */
export interface MapIcon {
  name: string;
  color: string;
  glyph: ComponentType<{ size?: number; color?: string; strokeWidth?: number }>;
}

/** **علامةٌ بصورةٍ من الخادم** — شعارُ المكتب (لا تُرسم في المضلَّع بل فوقه). */
export interface MapMarker {
  id: string;
  lng: number;
  lat: number;
  /** **رابطُ الصورة** — وغيابُها يُظهر الحرف. */
  imageUrl?: string | null;
  /** **حرفٌ بديل** — أوّلُ حرفٍ من اسم المنصّة. */
  letter: string;
  label: string;
}

/** **طلبُ تحريكٍ صريح** — والمفتاحُ يتبدّل مع كلّ نقرة. */
export interface FlyRequest {
  lng: number;
  lat: number;
  zoom?: number;
  key: number;
}

const EMPTY: FeatureCollection = { type: "FeatureCollection", features: [] };

/** **قطرُ الرمز بالبكسل المنطقيّ** — ويُرسم بضعفه. */
const ICON_PX = 30;
const ICON_RATIO = 2;

/**
 * **يرسم رمزاً**: دائرةٌ بلون الحال، وحلقةٌ بلون الورق، ورمزٌ بلون الحرف
 * على الصلب (`on-solid`). **والرمزُ من مكوّن الأيقونة نفسِه** — يُحوَّل SVG
 * في عقدةٍ منفصلةٍ ثمّ يُرسم على لوح.
 */
async function drawIcon(icon: MapIcon): Promise<ImageData | null> {
  const size = ICON_PX * ICON_RATIO;
  const glyph = 16 * ICON_RATIO;
  const host = document.createElement("div");
  const root = createRoot(host);
  flushSync(() =>
    root.render(createElement(icon.glyph, { size: glyph, color: themeColor("on-solid"), strokeWidth: 2.25 })),
  );
  const svg = host.innerHTML;
  root.unmount();

  const img = new Image();
  img.src = `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`;
  try {
    await img.decode();
  } catch {
    return null;
  }
  const c = document.createElement("canvas");
  c.width = size;
  c.height = size;
  const g = c.getContext("2d");
  if (!g) return null;
  g.beginPath();
  g.arc(size / 2, size / 2, size / 2 - ICON_RATIO * 1.5, 0, Math.PI * 2);
  g.fillStyle = icon.color;
  g.fill();
  g.lineWidth = ICON_RATIO * 2;
  g.strokeStyle = themeColor("paper");
  g.stroke();
  g.drawImage(img, (size - glyph) / 2, (size - glyph) / 2, glyph, glyph);
  return g.getImageData(0, 0, size, size);
}

/**
 * **متر إلى بكسل** — منقولٌ من `ZonesMap` حرفاً.
 *
 * **ودائرةٌ تصف مسافةً لا حجماً** — فتكبر مع التكبير.
 */
function radiusExpression(field: string) {
  return [
    "interpolate", ["exponential", 2], ["zoom"],
    0, ["/", ["get", field], 126700],
    22, ["/", ["*", ["get", field], 4194304], 126700],
  ];
}

export function OpsMapCanvas({
  layers,
  icons = [],
  markers = [],
  flyTo,
  onFeatureClick,
  onMarkerClick,
  onMoveEnd,
  onMapClick,
  height = "h-full",
  unavailableLabel,
  focus,
}: {
  layers: LayerSpec[];
  /** **صورُ الرموز** — تُرسم مرّةً عند الإقلاع قبل أوّل طبقة. */
  icons?: MapIcon[];
  /** **علاماتٌ بصورة** — شعارُ المكتب. */
  markers?: MapMarker[];
  /** **تحريكٌ بنقرةٍ صريحة وحدَها.** */
  flyTo?: FlyRequest;
  onFeatureClick?: (layerID: string, props: Record<string, unknown>) => void;
  onMarkerClick?: (id: string) => void;
  onMoveEnd?: (bbox: [number, number, number, number], zoom: number) => void;
  /** **نقرٌ على الأرض لا على معلَم** — رسمُ المضلَّع وتحديدُ موضعِ فرع. */
  onMapClick?: (lng: number, lat: number) => void;
  height?: string;
  unavailableLabel: string;
  /** **موضعٌ تُفتح عليه الخريطة** — من «اعرض على الخريطة» في «طلبات التوسّع». */
  focus?: { lng: number; lat: number; zoom?: number };
}) {
  const box = useRef<HTMLDivElement>(null);
  const map = useRef<maplibregl.Map | null>(null);
  const engine = useRef<typeof maplibregl | null>(null);
  const [ready, setReady] = useState(false);
  const [failed, setFailed] = useState(false);
  const built = useRef<Set<string>>(new Set());
  const placed = useRef<Map<string, maplibregl.Marker>>(new Map());

  // **والمُنادياتُ في مرجعٍ** — فلا يُعاد بناءُ الخريطة كلَّما تبدّلت.
  const clickRef = useRef(onFeatureClick);
  clickRef.current = onFeatureClick;
  const markerRef = useRef(onMarkerClick);
  markerRef.current = onMarkerClick;
  const moveRef = useRef(onMoveEnd);
  moveRef.current = onMoveEnd;
  const mapClickRef = useRef(onMapClick);
  mapClickRef.current = onMapClick;
  const iconsRef = useRef(icons);
  iconsRef.current = icons;

  useEffect(() => {
    const source = resolveMapSource();
    if (!source.ok) {
      setFailed(true);
      return;
    }
    let cancelled = false;
    let instance: maplibregl.Map | null = null;
    const markersNow = placed.current;

    (async () => {
      try {
        const maplibre = await loadMapEngine(source.styleUrl);
        if (cancelled || !box.current) return;
        engine.current = maplibre as unknown as typeof maplibregl;
        instance = new maplibre.Map({
          container: box.current,
          style: source.styleUrl,
          center: focus ? [focus.lng, focus.lat] : RAQQA,
          zoom: focus?.zoom ?? 12,
          attributionControl: { customAttribution: MAP_ATTRIBUTION_FALLBACK },
        });
        instance.addControl(new maplibre.NavigationControl({ showCompass: false }), "top-left");
        map.current = instance;
        instance.on("load", async () => {
          // **والرموزُ قبل الطبقات** — طبقةٌ تطلب صورةً لم تُسجَّل ترسم فراغاً.
          for (const ic of iconsRef.current) {
            const data = await drawIcon(ic);
            if (cancelled || !instance) return;
            if (data && !instance.hasImage(ic.name)) {
              instance.addImage(ic.name, data, { pixelRatio: ICON_RATIO });
            }
          }
          if (!cancelled) setReady(true);
        });
        // **ونقرُ الأرض يُرسَل خامّاً** — والمُنادي يقرّر ما يفعل به.
        instance.on("click", (e: maplibregl.MapMouseEvent) => {
          mapClickRef.current?.(e.lngLat.lng, e.lngLat.lat);
        });
        instance.on("moveend", () => {
          if (cancelled || !instance) return;
          const b = instance.getBounds();
          moveRef.current?.(
            [b.getWest(), b.getSouth(), b.getEast(), b.getNorth()],
            instance.getZoom(),
          );
        });
      } catch {
        if (!cancelled) setFailed(true);
      }
    })();

    return () => {
      cancelled = true;
      for (const mk of markersNow.values()) mk.remove();
      markersNow.clear();
      instance?.remove();
      map.current = null;
      built.current.clear();
    };
  }, []);

  /** **يبني الطبقةَ إن لم تُبنَ، ويحدّث بياناتِها دائماً.** */
  const sync = useCallback((spec: LayerSpec) => {
    const m = map.current;
    if (!m) return;
    const srcID = `src-${spec.id}`;

    if (!built.current.has(spec.id)) {
      m.addSource(srcID, {
        type: "geojson",
        data: spec.data as never,
        ...(spec.cluster
          ? { cluster: true, clusterRadius: 48, clusterMaxZoom: 14 }
          : {}),
      } as never);

      if (spec.kind === "fill") {
        m.addLayer({
          id: spec.id, type: "fill", source: srcID,
          paint: { "fill-color": spec.color as never, "fill-opacity": spec.fillOpacity ?? 0.18 },
        } as never);
        m.addLayer({
          id: `${spec.id}-line`, type: "line", source: srcID,
          paint: { "line-color": spec.color as never, "line-width": spec.lineWidth ?? 2 },
        } as never);
      } else if (spec.kind === "line") {
        m.addLayer({
          id: spec.id, type: "line", source: srcID,
          paint: {
            "line-color": spec.color as never,
            "line-width": spec.lineWidth ?? 2,
            ...(spec.dashed === false ? {} : { "line-dasharray": [2, 2] }),
          },
        } as never);
      } else if (spec.kind === "circle-m") {
        m.addLayer({
          id: spec.id, type: "circle", source: srcID,
          paint: {
            "circle-radius": radiusExpression(spec.radiusField ?? "radius_m") as never,
            "circle-color": spec.color as never,
            "circle-opacity": 0.15,
            "circle-stroke-width": 2,
            "circle-stroke-color": spec.color as never,
          },
        } as never);
      } else if (spec.kind === "heat") {
        m.addLayer({
          id: spec.id, type: "heatmap", source: srcID,
          paint: {
            "heatmap-weight": spec.weightField
              ? (["coalesce", ["get", spec.weightField], 1] as never)
              : 1,
            "heatmap-radius": 34,
            "heatmap-opacity": 0.65,
          },
        } as never);
      } else if (spec.kind === "bubble") {
        // ── فقّاعاتُ عدد — تكبر بالعدد لا بالتكبير ─────────────
        m.addLayer({
          id: spec.id, type: "circle", source: srcID,
          paint: {
            "circle-radius": [
              "interpolate", ["linear"], ["get", "count"],
              3, 11, 20, 16, 100, 24, 500, 32,
            ] as never,
            "circle-color": spec.color as never,
            "circle-opacity": 0.72,
            "circle-stroke-width": 2,
            "circle-stroke-color": themeColor("paper"),
          },
        } as never);
        m.addLayer({
          id: `${spec.id}-label`, type: "symbol", source: srcID,
          layout: {
            "text-field": ["to-string", ["get", "count"]] as never,
            "text-size": 11,
            "text-allow-overlap": true,
          },
          paint: { "text-color": themeColor("on-solid") },
        } as never);
      } else {
        // ── نقاطٌ أو رموز · وتُجمَّع إن طُلب ──────────────────
        if (spec.cluster) {
          m.addLayer({
            id: `${spec.id}-cluster`, type: "circle", source: srcID,
            filter: ["has", "point_count"],
            paint: {
              "circle-color": spec.color as never,
              "circle-opacity": 0.85,
              "circle-radius": ["step", ["get", "point_count"], 16, 10, 22, 50, 30] as never,
              "circle-stroke-width": 2,
              "circle-stroke-color": themeColor("paper"),
            },
          } as never);
          m.addLayer({
            id: `${spec.id}-count`, type: "symbol", source: srcID,
            filter: ["has", "point_count"],
            layout: {
              "text-field": ["get", "point_count_abbreviated"] as never,
              "text-size": 12,
            },
            paint: { "text-color": themeColor("on-solid") },
          } as never);
          // **ونقرُ الفقّاعة يكبّر عليها** — حركةٌ بنقرةٍ صريحة.
          m.on("click", `${spec.id}-cluster`, (e: maplibregl.MapLayerMouseEvent) => {
            const f = e.features?.[0];
            const id = f?.properties?.cluster_id as number | undefined;
            const src = m.getSource(srcID) as maplibregl.GeoJSONSource | undefined;
            if (f == null || id == null || !src) return;
            const [lng, lat] = (f.geometry as unknown as { coordinates: [number, number] }).coordinates;
            void src.getClusterExpansionZoom(id).then((z) => m.easeTo({ center: [lng, lat], zoom: z }));
          });
        }
        if (spec.kind === "symbol") {
          m.addLayer({
            id: spec.id, type: "symbol", source: srcID,
            ...(spec.cluster ? { filter: ["!", ["has", "point_count"]] } : {}),
            layout: {
              "icon-image": ["get", "icon"] as never,
              "icon-allow-overlap": true,
              "icon-ignore-placement": true,
            },
          } as never);
        } else {
          m.addLayer({
            id: spec.id, type: "circle", source: srcID,
            ...(spec.cluster ? { filter: ["!", ["has", "point_count"]] } : {}),
            paint: {
              "circle-radius": 7,
              "circle-color": spec.color as never,
              "circle-stroke-width": 2,
              // **وحلقةٌ بلون الورق** — فتُرى النقطةُ على أيّ خلفيّة.
              "circle-stroke-color": themeColor("paper"),
            },
          } as never);
        }
      }

      // **والنقرُ يفتح اللوحةَ الجانبيّة** — لا نافذةً منبثقةً ضخمة
      // (البند ٣٧).
      m.on("click", spec.id, (e: maplibregl.MapLayerMouseEvent) => {
        const f = e.features?.[0];
        if (f) clickRef.current?.(spec.id, f.properties ?? {});
      });
      m.on("mouseenter", spec.id, () => {
        m.getCanvas().style.cursor = "pointer";
      });
      m.on("mouseleave", spec.id, () => {
        m.getCanvas().style.cursor = "";
      });

      built.current.add(spec.id);
    }

    const src = m.getSource(srcID) as maplibregl.GeoJSONSource | undefined;
    src?.setData((spec.visible ? spec.data : EMPTY) as never);

    for (const id of [spec.id, `${spec.id}-line`, `${spec.id}-cluster`, `${spec.id}-count`, `${spec.id}-label`]) {
      if (m.getLayer(id)) {
        m.setLayoutProperty(id, "visibility", spec.visible ? "visible" : "none");
      }
    }
  }, []);

  useEffect(() => {
    if (!ready) return;
    for (const spec of [...layers].sort((a, b) => a.order - b.order)) sync(spec);
  }, [layers, ready, sync]);

  // ── العلامات — تُبنى وتُهدم بحسب القائمة ──────────────────────
  useEffect(() => {
    const m = map.current;
    const lib = engine.current;
    if (!ready || !m || !lib) return;
    const want = new Set(markers.map((x) => x.id));
    for (const [id, mk] of placed.current) {
      if (!want.has(id)) {
        mk.remove();
        placed.current.delete(id);
      }
    }
    for (const x of markers) {
      placed.current.get(x.id)?.remove();
      const el = document.createElement("button");
      el.type = "button";
      el.title = x.label;
      el.setAttribute("aria-label", x.label);
      el.className =
        "grid h-11 w-11 place-items-center overflow-hidden rounded-full border-2 border-line bg-paper text-base font-bold text-on-bright shadow-lift";
      if (x.imageUrl) {
        const img = document.createElement("img");
        img.src = x.imageUrl;
        img.alt = "";
        img.className = "h-8 w-8 object-contain";
        el.appendChild(img);
      } else {
        el.textContent = x.letter;
      }
      el.addEventListener("click", (ev) => {
        ev.stopPropagation();
        markerRef.current?.(x.id);
      });
      const mk = new lib.Marker({ element: el }).setLngLat([x.lng, x.lat]).addTo(m);
      placed.current.set(x.id, mk);
    }
  }, [markers, ready]);

  // ── التحريكُ بنقرةٍ صريحة وحدَها ──────────────────────────────
  useEffect(() => {
    const m = map.current;
    if (!ready || !m || !flyTo) return;
    m.flyTo({ center: [flyTo.lng, flyTo.lat], zoom: flyTo.zoom ?? Math.max(m.getZoom(), 15) });
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [flyTo?.key, ready]);

  if (failed) {
    return (
      <div
        className={`${height} grid place-items-center rounded-card border border-dashed border-[var(--border)] bg-[var(--surface)] text-sm text-[var(--muted)]`}
        role="status"
      >
        {unavailableLabel}
      </div>
    );
  }

  return <div ref={box} className={`${height} w-full rounded-card overflow-hidden`} dir="ltr" />;
}
