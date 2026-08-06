"use client";

/**
 * خريطة اختيار الموقع — **نسخة واحدة مركزية** لكل التطبيقات.
 * كانت مكرّرة حرفياً في تطبيقين بـ11 قيمة لونية صريحة خارج التوكنز (R-09).
 *
 * تُستورد عبر مدخل فرعي خاص (`@rahalgo/ui/map`) لا من الفهرس العام: مكتبة
 * الخرائط ثقيلة وتحتاج DOM، فلا يجرّها كل تطبيق يستورد زرّاً.
 *
 *   const PickMap = dynamic(() => import("@rahalgo/ui/map").then((m) => m.PickMap), { ssr: false });
 */

import { useEffect, useRef, useState } from "react";
import { MapContainer, CircleMarker, Circle, TileLayer, useMapEvents, useMap } from "react-leaflet";
import type { Map as LeafletMap } from "leaflet";
import "leaflet/dist/leaflet.css";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
// **ألوانُ الخريطة من الثيم لا من لوحةٍ ثانية.**
//
// كانت تُقرأ من `tokens.ts` — **ولوحتُه شاخت**: `primary` فيها أزرقُ داكنٌ
// والثيمُ صار سماويّاً، **فكانت الخريطةُ ترسم بألوان منصّةٍ أخرى** ولا يظهر
// خطأً. (انظر `cssvar.ts`.)
import { themeColor } from "./cssvar";
import { IconLocateMe, IconAdd, IconClose } from "./icons";

const m = getMessages(defaultLocale);
const RAQQA_CENTER: [number, number] = [35.9528, 39.0079];

/**
 * مصادر البلاطات بترتيب أفضلية — عند فشل المصدر ينتقل تلقائياً للتالي،
 * لأن خوادم OSM المجانية غير مضمونة. الأول عند النشر خادمنا الذاتي.
 */
const TILE_SOURCES = [
  process.env.NEXT_PUBLIC_TILE_URL,
  "https://tile.openstreetmap.org/{z}/{x}/{y}.png",
  "https://a.basemaps.cartocdn.com/rastertiles/voyager/{z}/{x}/{y}.png",
  "https://tile.openstreetmap.de/{z}/{x}/{y}.png",
].filter((u): u is string => !!u);

const ERRORS_BEFORE_SWITCH = 4;

export function FallbackTileLayer() {
  const [srcIndex, setSrcIndex] = useState(0);
  const [errors, setErrors] = useState(0);
  return (
    <TileLayer
      key={srcIndex}
      url={TILE_SOURCES[srcIndex] ?? TILE_SOURCES[0]!}
      attribution="&copy; OpenStreetMap"
      eventHandlers={{
        tileerror: () =>
          setErrors((e) => {
            if (e + 1 >= ERRORS_BEFORE_SWITCH && srcIndex < TILE_SOURCES.length - 1) {
              setSrcIndex((i) => i + 1);
              return 0;
            }
            return e + 1;
          }),
      }}
    />
  );
}

function ClickCapture({ onPick }: { onPick: (lat: number, lng: number) => void }) {
  useMapEvents({
    click: (e) => onPick(e.latlng.lat, e.latlng.lng),
  });
  return null;
}

/** يحرّك الخريطة عندما يتغيّر الموقع من خارجها (بحث عنوان مثلاً). */
function FlyTo({ lat, lng, zoom }: { lat: number | null; lng: number | null; zoom: number }) {
  const map = useMap();
  const last = useRef<string>("");
  useEffect(() => {
    if (lat == null || lng == null) return;
    const key = `${lat},${lng}`;
    if (key === last.current) return;
    last.current = key;
    map.flyTo([lat, lng], Math.max(map.getZoom(), zoom), { duration: 0.6 });
  }, [lat, lng, zoom, map]);
  return null;
}

/** أزرار التكبير — بديل أزرار Leaflet الافتراضية كي تتبع توكنز المنصة. */
function ZoomControls() {
  const map = useMap();
  const btn =
    "taparea flex h-8 w-8 items-center justify-center bg-surface text-ink transition-colors hover:bg-page disabled:opacity-40";
  return (
    <div className="absolute end-2 top-2 z-[1000] overflow-hidden rounded-control border border-line elev-1">
      <button type="button" aria-label="+" className={`${btn} border-b border-line`} onClick={() => map.zoomIn()}>
        <IconAdd size={16} />
      </button>
      <button type="button" aria-label="−" className={btn} onClick={() => map.zoomOut()}>
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
}: {
  lat: number | null;
  lng: number | null;
  onPick: (lat: number, lng: number) => void;
  height?: string;
  radiusM?: number;
  onLocated?: (lat: number, lng: number) => void;
}) {
  const mapRef = useRef<LeafletMap | null>(null);
  const [locating, setLocating] = useState(false);
  const [denied, setDenied] = useState(false);
  const center: [number, number] = lat != null && lng != null ? [lat, lng] : RAQQA_CENTER;

  function locateMe() {
    if (!navigator.geolocation) return setDenied(true);
    setDenied(false);
    setLocating(true);
    navigator.geolocation.getCurrentPosition(
      (pos) => {
        const { latitude, longitude } = pos.coords;
        mapRef.current?.flyTo([latitude, longitude], 17, { duration: 0.6 });
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
      <button
        type="button"
        onClick={locateMe}
        disabled={locating}
        className="mb-2 flex w-full items-center justify-center gap-2 rounded-control bg-accent px-4 py-2.5 text-sm font-bold text-on-bright elev-1 transition-colors hover:bg-accent-dark disabled:opacity-60"
      >
        <IconLocateMe size={17} className={locating ? "animate-pulse" : ""} />
        {locating ? m.common.loading : m.common.locateMe}
      </button>

    <div className="relative overflow-hidden rounded-card border border-line">
      <MapContainer
        ref={mapRef}
        center={center}
        zoom={lat != null ? 16 : 13}
        zoomControl={false}
        className={`${height} w-full`}
        style={{ cursor: "crosshair" }}
      >
        <FallbackTileLayer />
        <ClickCapture onPick={onPick} />
        <FlyTo lat={lat} lng={lng} zoom={16} />
        <ZoomControls />
        {lat != null && lng != null && (
          <>
            {radiusM ? (
              <Circle
                center={[lat, lng]}
                radius={radiusM}
                pathOptions={{
                  color: themeColor("primary-dark"),
                  fillColor: themeColor("primary"),
                  fillOpacity: 0.15,
                  weight: 2,
                }}
              />
            ) : null}
            {/* هالة خارجية + نواة: دبوس واضح على أي خلفية بلاطات */}
            <CircleMarker
              center={[lat, lng]}
              radius={14}
              pathOptions={{
                color: themeColor("primary"),
                fillColor: themeColor("primary"),
                fillOpacity: 0.18,
                weight: 0,
              }}
            />
            <CircleMarker
              center={[lat, lng]}
              radius={7}
              pathOptions={{
                color: themeColor("surface"),
                fillColor: themeColor("primary"),
                fillOpacity: 1,
                weight: 3,
              }}
            />
          </>
        )}
      </MapContainer>


      {denied && (
        <p className="absolute bottom-2 start-2 z-[1000] rounded-control bg-danger-fill px-2.5 py-1.5 text-xs text-on-bright">
          {m.map.denied}
        </p>
      )}
      </div>
    </div>
  );
}
