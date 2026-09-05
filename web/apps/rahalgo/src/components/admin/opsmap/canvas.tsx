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
 * # والتجميعُ لازمٌ لا زينة (البند ٣٠)
 *
 * **مئتا سائقٍ وألفُ متجرٍ نقاطٌ خامٌّ تُجمّد المتصفّح** — **والتجميعُ
 * في المصدر لا في الرسم**، فـMapLibre يفعله في العامل.
 *
 * # ولا مصدرَ خرائطَ عموميّ
 *
 * **`resolveMapSource` يحسم** — **ولا ارتدادَ إلى مصدرٍ عموميٍّ البتّة**
 * (`TD-WEB-RASTER-SOURCE`، ٢٠٢٦-٠٨-٢١). **وغيابُ الإعداد حالُ عطلٍ
 * مضبوطة، لا خريطةٌ من بنيةٍ لا نملكها ولا نضمنها.**
 */

import type * as maplibregl from "maplibre-gl";
import { useEffect, useRef, useState, useCallback } from "react";
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
  /** `point` نقاطٌ · `fill` مضلَّعاتٌ · `circle-m` دوائرُ بالمتر · `heat` كثافة */
  kind: "point" | "fill" | "circle-m" | "heat" | "line";
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
}

const EMPTY: FeatureCollection = { type: "FeatureCollection", features: [] };

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
  onFeatureClick,
  onMoveEnd,
  height = "h-full",
  unavailableLabel,
}: {
  layers: LayerSpec[];
  onFeatureClick?: (layerID: string, props: Record<string, unknown>) => void;
  onMoveEnd?: (bbox: [number, number, number, number], zoom: number) => void;
  height?: string;
  unavailableLabel: string;
}) {
  const box = useRef<HTMLDivElement>(null);
  const map = useRef<maplibregl.Map | null>(null);
  const [ready, setReady] = useState(false);
  const [failed, setFailed] = useState(false);
  const built = useRef<Set<string>>(new Set());

  // **والمُنادياتُ في مرجعٍ** — فلا يُعاد بناءُ الخريطة كلَّما تبدّلت.
  const clickRef = useRef(onFeatureClick);
  clickRef.current = onFeatureClick;
  const moveRef = useRef(onMoveEnd);
  moveRef.current = onMoveEnd;

  useEffect(() => {
    const source = resolveMapSource();
    if (!source.ok) {
      setFailed(true);
      return;
    }
    let cancelled = false;
    let instance: maplibregl.Map | null = null;

    (async () => {
      try {
        const maplibre = await loadMapEngine(source.styleUrl);
        if (cancelled || !box.current) return;
        instance = new maplibre.Map({
          container: box.current,
          style: source.styleUrl,
          center: RAQQA,
          zoom: 12,
          attributionControl: { customAttribution: MAP_ATTRIBUTION_FALLBACK },
        });
        instance.addControl(new maplibre.NavigationControl({ showCompass: false }), "top-left");
        map.current = instance;
        instance.on("load", () => {
          if (!cancelled) setReady(true);
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
          paint: { "fill-color": spec.color as never, "fill-opacity": 0.18 },
        } as never);
        m.addLayer({
          id: `${spec.id}-line`, type: "line", source: srcID,
          paint: { "line-color": spec.color as never, "line-width": 2 },
        } as never);
      } else if (spec.kind === "line") {
        m.addLayer({
          id: spec.id, type: "line", source: srcID,
          paint: { "line-color": spec.color as never, "line-width": 2, "line-dasharray": [2, 2] },
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
      } else {
        // ── نقاطٌ · وتُجمَّع إن طُلب ──────────────────────────
        if (spec.cluster) {
          m.addLayer({
            id: `${spec.id}-cluster`, type: "circle", source: srcID,
            filter: ["has", "point_count"],
            paint: {
              "circle-color": spec.color as never,
              "circle-opacity": 0.85,
              "circle-radius": ["step", ["get", "point_count"], 16, 10, 22, 50, 30] as never,
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
        }
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

    for (const id of [spec.id, `${spec.id}-line`, `${spec.id}-cluster`, `${spec.id}-count`]) {
      if (m.getLayer(id)) {
        m.setLayoutProperty(id, "visibility", spec.visible ? "visible" : "none");
      }
    }
  }, []);

  useEffect(() => {
    if (!ready) return;
    for (const spec of [...layers].sort((a, b) => a.order - b.order)) sync(spec);
  }, [layers, ready, sync]);

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
