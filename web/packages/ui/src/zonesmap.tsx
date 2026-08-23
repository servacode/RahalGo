"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **خريطةُ مناطق التغطية — على المحرّك نفسِه**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (إغلاقُ `TD-WEB-RASTER-SOURCE`، قرارُ المالك ٢٠٢٦-٠٨-٢١.)
 *
 * **كانت هذه الصفحةُ الثانيةَ التي تستورد `react-leaflet` مباشرةً** —
 * ولم يكشفها بحثي الأوّل عن `PickMap`. **ولذلك لا يُفترض أنّ العيبَ في
 * ملفّ واحد** (البند ١).
 *
 * **ودوائرُ المناطق تُرسم بمصدرٍ واحدٍ لا بطبقةٍ لكلّ منطقة** — فمئةُ
 * منطقةٍ لا تصير مئةَ طبقة.
 *
 * **واللونُ من الثيم لحظةَ الرسم** كما كان — والتعليقُ الأصليُّ محفوظٌ
 * في الصفحة.
 */

import { useEffect, useRef, useState } from "react";
import { resolveMapSource, MAP_ATTRIBUTION_FALLBACK } from "./mapconfig";
import { loadMapEngine } from "./mapengine";

const RAQQA_CENTER: [number, number] = [35.9528, 39.0079];

export interface ZoneShape {
  id: string;
  name: string;
  lat: number;
  lng: number;
  radius_m: number;
  delivery_fee?: number;
  active: boolean;
}

export interface ZonePalette {
  primary: string;
  accent: string;
  accentDark: string;
  muted: string;
}

/**
 * **نصفُ قطرٍ بالمتر إلى بكسل** — تعبيرُ MapLibre.
 *
 * **ودائرةُ Leaflet كانت بالمتر أصلاً**، وMapLibre ترسم بالبكسل —
 * **فيُبنى التحويلُ بالتكبير** لتبقى الدائرةُ تصف مسافةً لا حجماً.
 *
 * **والمقامُ عند خطّ عرض الرقّة**: مترٌ لكلّ بكسلٍ عند تكبير صفر
 * ≈ ١٥٦٥٤٣ × جيبِ تمام العرض ≈ ١٢٦٧٠٠ — **ثمّ يُقسَّم بضِعف التكبير.**
 */
function radiusExpression(field: string) {
  return [
    "interpolate", ["exponential", 2], ["zoom"],
    0, ["/", ["get", field], 126700],
    22, ["/", ["*", ["get", field], 4194304], 126700],
  ];
}

export function ZonesMap({
  zones,
  editing,
  draft,
  selectedID,
  onMapClick,
  onZoneClick,
  palette,
  unavailableLabel,
}: {
  zones: ZoneShape[];
  editing: boolean;
  draft: { lat: number; lng: number; radiusM: number } | null;
  selectedID: string | null;
  onMapClick: (lat: number, lng: number) => void;
  onZoneClick: (id: string) => void;
  palette: ZonePalette;
  unavailableLabel: string;
}) {
  const holder = useRef<HTMLDivElement | null>(null);
  const mapRef = useRef<import("maplibre-gl").Map | null>(null);
  const clickRef = useRef(onMapClick);
  const zoneClickRef = useRef(onZoneClick);
  const editRef = useRef(editing);
  clickRef.current = onMapClick;
  zoneClickRef.current = onZoneClick;
  editRef.current = editing;

  const [ready, setReady] = useState(false);
  const [failed, setFailed] = useState(false);
  const source = resolveMapSource();

  useEffect(() => {
    if (!source.ok) {
      setFailed(true);
      return;
    }
    let cancelled = false;
    let map: import("maplibre-gl").Map | null = null;
    let observer: ResizeObserver | null = null;
    loadMapEngine(source.styleUrl)
      .then((maplibregl) => {
        if (cancelled || !holder.current) return;
        const instance = new maplibregl.Map({
          container: holder.current,
          style: source.styleUrl,
          center: [RAQQA_CENTER[1], RAQQA_CENTER[0]],
          zoom: 13,
          attributionControl: { compact: true },
          dragRotate: false,
          pitchWithRotate: false,
        });
        map = instance;
        mapRef.current = instance;
        instance.on("load", () => {
          if (!cancelled) setReady(true);
        });
        instance.on("error", () => {
          if (!cancelled) setFailed(true);
        });
        
        // ══════════════════════════════════════════════════════════════
        // **وحجمُ الحاوية يُراقَب — وإلّا بقيت الخريطةُ فارغة**
        // ══════════════════════════════════════════════════════════════
        //
        // **المحرّكُ يقرأ الحجمَ لحظةَ الإنشاء** ويحسب منه أيَّ بلاطاتٍ
        // يطلب. **والمكوّنُ يُحمَّل ديناميّاً** فقد تكون الحاويةُ بلا
        // ارتفاعٍ حينَها — **فلا يُطلب شيءٌ ولا يُرمى خطأ.**
        //
        // (قِيس ٢٠٢٦-٠٨-٢١: ترويسةٌ ووصفٌ يُقرآن ثمّ صمتٌ تامّ.)
        const ro = new ResizeObserver(() => instance.resize());
        if (holder.current) ro.observe(holder.current);
        observer = ro;
        instance.on("click", (e) => {
          if (!editRef.current) return;
          clickRef.current(e.lngLat.lat, e.lngLat.lng);
        });
      })
      .catch(() => {
        if (!cancelled) setFailed(true);
      });
    return () => {
      cancelled = true;
      observer?.disconnect();
      map?.remove();
      mapRef.current = null;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [source.ok, source.ok ? source.styleUrl : ""]);

  // ── المناطقُ مصدرٌ واحد ─────────────────────────────────────────
  useEffect(() => {
    const map = mapRef.current;
    if (!map || !ready) return;
    const data = {
      type: "FeatureCollection" as const,
      features: zones.map((z) => ({
        type: "Feature" as const,
        id: z.id,
        geometry: { type: "Point" as const, coordinates: [z.lng, z.lat] },
        properties: {
          zid: z.id,
          radius: z.radius_m,
          color: z.id === selectedID ? palette.accent : z.active ? palette.primary : palette.muted,
          // **شهده المالك ٢٠٢٦-٠٨-١١: «شفّافةٌ جدّاً، لا أستطيع رؤيتَها»**
          // — والقيمُ كما استُقرّت يومَها.
          fill: z.id === selectedID ? 0.42 : 0.28,
          weight: z.id === selectedID ? 5 : 4,
        },
      })),
    };
    const src = map.getSource("zones") as import("maplibre-gl").GeoJSONSource | undefined;
    if (src) {
      src.setData(data);
      return;
    }
    map.addSource("zones", { type: "geojson", data });
    map.addLayer({
      id: "zones-fill",
      type: "circle",
      source: "zones",
      paint: {
        "circle-radius": radiusExpression("radius") as never,
        "circle-color": ["get", "color"],
        "circle-opacity": ["get", "fill"],
        "circle-stroke-color": ["get", "color"],
        "circle-stroke-width": ["get", "weight"],
      },
    });
    map.on("click", "zones-fill", (e) => {
      const f = e.features?.[0];
      if (f) zoneClickRef.current(String(f.properties?.zid ?? ""));
    });
    map.on("mouseenter", "zones-fill", () => {
      map.getCanvas().style.cursor = "pointer";
    });
    map.on("mouseleave", "zones-fill", () => {
      map.getCanvas().style.cursor = editRef.current ? "crosshair" : "";
    });
  }, [zones, selectedID, palette, ready]);

  // ── والمسوّدةُ تُرى كأختِها ──────────────────────────────────────
  useEffect(() => {
    const map = mapRef.current;
    if (!map || !ready) return;
    const data = {
      type: "FeatureCollection" as const,
      features: draft
        ? [
            {
              type: "Feature" as const,
              geometry: { type: "Point" as const, coordinates: [draft.lng, draft.lat] },
              properties: { radius: draft.radiusM },
            },
          ]
        : [],
    };
    const src = map.getSource("draft") as import("maplibre-gl").GeoJSONSource | undefined;
    if (src) {
      src.setData(data);
      return;
    }
    map.addSource("draft", { type: "geojson", data });
    map.addLayer({
      id: "draft-fill",
      type: "circle",
      source: "draft",
      paint: {
        "circle-radius": radiusExpression("radius") as never,
        "circle-color": palette.accent,
        "circle-opacity": 0.35,
        "circle-stroke-color": palette.accentDark,
        "circle-stroke-width": 4,
      },
    });
    map.addLayer({
      id: "draft-center",
      type: "circle",
      source: "draft",
      paint: {
        "circle-radius": 6,
        "circle-color": palette.accent,
        "circle-stroke-color": palette.accentDark,
        "circle-stroke-width": 2,
      },
    });
  }, [draft, palette, ready]);

  // ── ومؤشّرُ التحرير يتبع الحال ──────────────────────────────────
  useEffect(() => {
    const map = mapRef.current;
    if (!map || !ready) return;
    map.getCanvas().style.cursor = editing ? "crosshair" : "";
  }, [editing, ready]);

  return (
    <div className="relative h-full w-full">
      <div ref={holder} data-testid="zones-map" className="h-full w-full" />
      {failed && (
        <div className="absolute inset-0 flex flex-col items-center justify-center gap-1 bg-surface-2 px-4 text-center">
          <p className="text-sm font-bold text-ink">{unavailableLabel}</p>
          <p
            className="text-xs text-ink-soft"
            dangerouslySetInnerHTML={{ __html: MAP_ATTRIBUTION_FALLBACK }}
          />
        </div>
      )}
    </div>
  );
}
