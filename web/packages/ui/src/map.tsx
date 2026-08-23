"use client";

/**
 * خريطة اختيار الموقع — **نسخة واحدة مركزية** لكل التطبيقات.
 * كانت مكرّرة حرفياً في تطبيقين بـ11 قيمة لونية صريحة خارج التوكنز (R-09).
 *
 * تُستورد عبر مدخل فرعي خاص (`@rahalgo/ui/map`) لا من الفهرس العام: مكتبة
 * الخرائط ثقيلة وتحتاج DOM، فلا يجرّها كل تطبيق يستورد زرّاً.
 *
 *   const PickMap = dynamic(() => import("@rahalgo/ui/map").then((m) => m.PickMap), { ssr: false });
 *
 * ══════════════════════════════════════════════════════════════════════
 * **ومن راسترٍ عموميٍّ إلى متّجهٍ نملكه — إغلاقُ `TD-WEB-RASTER-SOURCE`**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-٠٨-٢١.)
 *
 * **كانت الخريطةُ على Leaflet ببلاطاتٍ راستر، وسلسلةِ ارتدادٍ تنتهي إلى
 * خوادمَ عموميّةٍ لا نملكها.** وأربعةُ أخطاءِ بلاطةٍ تكفي لينتقل الإنتاجُ
 * إلى بنيةٍ لا نملكها — **ولو كان مصدرُنا مضبوطاً.**
 *
 * **وصارت MapLibre على نمط المرحلة ٦ نفسِه**: PMTiles واحدٌ من مضيفنا،
 * وحروفٌ وأيقوناتٌ من خطّ إنتاجِ الأصول الذي أُغلق هناك — **فالويبُ
 * وأندرويد يقرآن من مصدرٍ واحد.**
 *
 * **والعقدُ العامُّ لم يتبدّل**: `PickMap` بالخصائص نفسِها،
 * **فلا صفحةَ مسّها التغيير** (البند ١٦).
 */

import { useEffect, useRef, useState } from "react";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
// **ألوانُ الخريطة من الثيم لا من لوحةٍ ثانية.**
//
// كانت تُقرأ من `tokens.ts` — **ولوحتُه شاخت**: `primary` فيها أزرقُ داكنٌ
// والثيمُ صار سماويّاً، **فكانت الخريطةُ ترسم بألوان منصّةٍ أخرى** ولا يظهر
// خطأً. (انظر `cssvar.ts`.)
import { themeColor } from "./cssvar";
import { IconLocateMe, IconAdd, IconClose } from "./icons";
import { resolveMapSource, MAP_ATTRIBUTION_FALLBACK } from "./mapconfig";
import { loadMapEngine } from "./mapengine";

const m = getMessages(defaultLocale);
const RAQQA_CENTER: [number, number] = [35.9528, 39.0079];

function ClickCapture() {
  return null;
}

/** أزرار التكبير — بديل أزرار المحرّك الافتراضية كي تتبع توكنز المنصة. */
function ZoomControls({ onZoom }: { onZoom: (delta: number) => void }) {
  const btn =
    "taparea flex h-8 w-8 items-center justify-center bg-surface text-ink transition-colors hover:bg-row-hover disabled:opacity-40";
  return (
    <div className="absolute end-2 top-2 z-[1000] overflow-hidden rounded-control border border-line elev-1">
      <button type="button" aria-label="+" className={`${btn} border-b border-line-soft`} onClick={() => onZoom(1)}>
        <IconAdd size={16} />
      </button>
      <button type="button" aria-label="−" className={btn} onClick={() => onZoom(-1)}>
        <IconClose size={16} className="rotate-45" />
      </button>
    </div>
  );
}

export function PickMap({
  lat,
  lng,
  onPick,
  height = "h-64",
  /** نصف قطر دائرة توضيحية بالمتر (مناطق التغطية) */
  radiusM,
  /** يُستدعى بعد تحديد الموقع من الجهاز — لتعبئة العنوان مثلاً */
  onLocated,
  /**
   * **إخفاءُ زرّ «تحديد موقعي»** — لخريطةٍ تُعرض ولا تُختار منها.
   *
   * (قرارُ المالك ٢٠٢٦-٠٨-٠٩: «ولا يوجد تحديد الموقع بالعرض».)
   *
   * **وخريطةُ المكتب تُقرأ لا تُضبط** — وزرٌّ يطلب موقعَ الزائر فيها **يسأل
   * إذناً بلا سبب**، ومن أعطاه لم يقع شيء.
   */
  hideLocate = false,
}: {
  lat: number | null;
  lng: number | null;
  onPick: (lat: number, lng: number) => void;
  height?: string;
  radiusM?: number;
  onLocated?: (lat: number, lng: number) => void;
  hideLocate?: boolean;
}) {
  const holder = useRef<HTMLDivElement | null>(null);
  const mapRef = useRef<import("maplibre-gl").Map | null>(null);
  const pickRef = useRef(onPick);
  pickRef.current = onPick;
  const [locating, setLocating] = useState(false);
  const [denied, setDenied] = useState(false);
  const [failed, setFailed] = useState<string | null>(null);
  const [ready, setReady] = useState(false);

  const source = resolveMapSource();
  const center: [number, number] = lat != null && lng != null ? [lat, lng] : RAQQA_CENTER;

  // ── إنشاءُ الخريطة مرّةً واحدة ───────────────────────────────────
  useEffect(() => {
    if (!source.ok) {
      // **ولا ارتدادَ إلى مصدرٍ عموميّ** (البند ١٩) — حالُ عطلٍ مضبوطة.
      setFailed(source.reason);
      return;
    }
    const node = holder.current;
    if (!node) return;
    let cancelled = false;
    let map: import("maplibre-gl").Map | null = null;
    let observer: ResizeObserver | null = null;

    loadMapEngine(source.styleUrl)
      .then((maplibregl) => {
        if (cancelled || !holder.current) return;
        const instance = new maplibregl.Map({
          container: holder.current,
          style: source.styleUrl,
          center: [center[1], center[0]],
          zoom: lat != null ? 16 : 13,
          attributionControl: { compact: true },
          // **ولا أزرارَ افتراضيّة** — أزرارُنا تتبع التوكنز.
          dragRotate: false,
          pitchWithRotate: false,
        });
        map = instance;
        mapRef.current = instance;
        instance.on("load", () => {
          if (!cancelled) setReady(true);
        });
        // **وخطأُ المصدر يُعرض ولا يُعاد** — ولا حلقةَ محاولات.
        instance.on("error", (e: unknown) => {
          const msg = String((e as { error?: { message?: string } })?.error?.message ?? "");
          if (msg.includes("style") || msg.includes("Failed to fetch")) {
            if (!cancelled) setFailed("unreachable");
          }
        });
        instance.on("click", (e) => pickRef.current(e.lngLat.lat, e.lngLat.lng));
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

      })
      .catch(() => {
        if (!cancelled) setFailed("unreachable");
      });

    return () => {
      cancelled = true;
      observer?.disconnect();
      map?.remove();
      mapRef.current = null;
    };
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [source.ok, source.ok ? source.styleUrl : ""]);

  // ── الدبّوسُ والدائرةُ — طبقتان تُحدَّثان ولا تُعاد الخريطة ──────
  useEffect(() => {
    const map = mapRef.current;
    if (!map || !ready) return;
    const data = {
      type: "FeatureCollection" as const,
      features:
        lat != null && lng != null
          ? [
              {
                type: "Feature" as const,
                geometry: { type: "Point" as const, coordinates: [lng, lat] },
                properties: {},
              },
            ]
          : [],
    };
    const src = map.getSource("pick") as import("maplibre-gl").GeoJSONSource | undefined;
    if (src) {
      src.setData(data);
      return;
    }
    map.addSource("pick", { type: "geojson", data });
    if (radiusM) {
      map.addLayer({
        id: "pick-radius",
        type: "circle",
        source: "pick",
        paint: {
          // **ونصفُ القطر بالمتر يُحوَّل بكسلاً بحسب التكبير** —
          // فالدائرةُ تصف مسافةً لا حجماً على الشاشة.
          "circle-radius": [
            "interpolate", ["exponential", 2], ["zoom"],
            10, radiusM / 152.87,
            20, (radiusM / 152.87) * 1024,
          ],
          "circle-color": themeColor("primary"),
          "circle-opacity": 0.15,
          "circle-stroke-color": themeColor("primary-dark"),
          "circle-stroke-width": 2,
        },
      });
    }
    map.addLayer({
      id: "pick-halo",
      type: "circle",
      source: "pick",
      paint: {
        "circle-radius": 14,
        "circle-color": themeColor("primary"),
        "circle-opacity": 0.18,
      },
    });
    map.addLayer({
      id: "pick-core",
      type: "circle",
      source: "pick",
      paint: {
        "circle-radius": 7,
        "circle-color": themeColor("primary"),
        "circle-stroke-color": themeColor("surface"),
        "circle-stroke-width": 3,
      },
    });
  }, [lat, lng, radiusM, ready]);

  // ── ويطير إليها إن تبدّلت من خارج الخريطة ───────────────────────
  const lastFly = useRef("");
  useEffect(() => {
    const map = mapRef.current;
    if (!map || lat == null || lng == null) return;
    const key = `${lat},${lng}`;
    if (key === lastFly.current) return;
    lastFly.current = key;
    map.flyTo({ center: [lng, lat], zoom: Math.max(map.getZoom(), 16), duration: 600 });
  }, [lat, lng]);

  function locateMe() {
    if (!navigator.geolocation) return setDenied(true);
    setDenied(false);
    setLocating(true);
    navigator.geolocation.getCurrentPosition(
      (pos) => {
        const { latitude, longitude } = pos.coords;
        mapRef.current?.flyTo({ center: [longitude, latitude], zoom: 17, duration: 600 });
        onPick(latitude, longitude);
        onLocated?.(latitude, longitude);
        setLocating(false);
      },
      () => {
        setDenied(true);
        setLocating(false);
      },
      { enableHighAccuracy: true, timeout: 10000 },
    );
  }

  return (
    <div>
      {/* **زرُّ الموقع فوق الخريطة لا داخلها.**
          كان في زاويتها السفلى بلون الحياد، فيختفي بين البلاطات — فيظلّ الزبون
          يحرّك الدبّوس بيده وهو يملك موقعه بضغطةٍ واحدة. **وأدقُّ دبّوسٍ يضعه
          الجهاز لا الإصبع**، ودقّتُه هي ما يُبلغ السائقَ البابَ.

          ولونٌ بارزٌ ممتلئ: زرٌّ محايدٌ في مشهدٍ مزدحم لا يُطلَب منه أن يُلحَظ. */}
      {!hideLocate && (
        <button
          type="button"
          onClick={locateMe}
          disabled={locating}
          className="mb-2 flex w-full items-center justify-center gap-2 rounded-control bg-accent px-4 py-2.5 text-sm font-bold text-on-bright elev-1 transition-colors hover:bg-accent-dark disabled:opacity-60"
        >
          <IconLocateMe size={17} className={locating ? "animate-pulse" : ""} />
          {locating ? m.common.loading : m.common.locateMe}
        </button>
      )}

      <div className="relative overflow-hidden rounded-card border border-line">
        <div
          ref={holder}
          data-testid="pick-map"
          className={`${height} w-full`}
          style={{ cursor: "crosshair" }}
        />

        {/* **وعطلُ المصدر يُقال ولا يُخفى** — والصفحةُ تبقى تعمل. */}
        {failed && (
          <div
            role="status"
            className={`${height} absolute inset-0 flex flex-col items-center justify-center gap-1 bg-surface-2 px-4 text-center`}
          >
            <p className="text-sm font-bold text-ink">{m.map.unavailable}</p>
            <p
              className="text-xs text-ink-soft"
              // **والنسبُ يبقى مرئيّاً ولو غاب النمط** — البند ١٧.
              dangerouslySetInnerHTML={{ __html: MAP_ATTRIBUTION_FALLBACK }}
            />
          </div>
        )}

        {!failed && <ZoomControls onZoom={(d) => mapRef.current?.zoomTo((mapRef.current?.getZoom() ?? 13) + d)} />}
        <ClickCapture />

        {denied && (
          <p className="absolute bottom-2 start-2 z-[1000] rounded-control bg-danger-fill px-2.5 py-1.5 text-xs text-on-bright">
            {m.map.denied}
          </p>
        )}
      </div>
    </div>
  );
}
