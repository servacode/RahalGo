"use client";

/** خريطة صغيرة لاختيار نقطة (دبوس) — نقر لتحديد الموقع، وزر "تحديد موقعي". */

import { useRef, useState } from "react";
import { MapContainer, CircleMarker, useMapEvents } from "react-leaflet";
import type { Map as LeafletMap } from "leaflet";
import "leaflet/dist/leaflet.css";
import { IconLocateMe } from "@rahalgo/ui";
import { getMessages, defaultLocale } from "@rahalgo/i18n";
import FallbackTileLayer from "./FallbackTileLayer";

const m = getMessages(defaultLocale);
const RAQQA_CENTER: [number, number] = [35.9528, 39.0079];

function ClickCapture({ onPick }: { onPick: (lat: number, lng: number) => void }) {
  useMapEvents({
    click(e) {
      onPick(e.latlng.lat, e.latlng.lng);
    },
  });
  return null;
}

export default function PickMap({
  lat,
  lng,
  onPick,
}: {
  lat: number | null;
  lng: number | null;
  onPick: (lat: number, lng: number) => void;
}) {
  const mapRef = useRef<LeafletMap | null>(null);
  const [locating, setLocating] = useState(false);
  const center: [number, number] = lat != null && lng != null ? [lat, lng] : RAQQA_CENTER;

  function locateMe() {
    if (!navigator.geolocation) return;
    setLocating(true);
    navigator.geolocation.getCurrentPosition(
      (pos) => {
        const { latitude, longitude } = pos.coords;
        mapRef.current?.flyTo([latitude, longitude], 16);
        onPick(latitude, longitude);
        setLocating(false);
      },
      () => setLocating(false),
      { enableHighAccuracy: true, timeout: 8000 },
    );
  }

  return (
    <div className="relative">
      <MapContainer
        ref={mapRef}
        center={center}
        zoom={14}
        className="h-52 w-full"
        style={{ cursor: "crosshair" }}
      >
        <FallbackTileLayer />
        <ClickCapture onPick={onPick} />
        {lat != null && lng != null && (
          <CircleMarker
            center={[lat, lng]}
            radius={9}
            pathOptions={{ color: "#155E75", fillColor: "#0E7490", fillOpacity: 0.9, weight: 2 }}
          />
        )}
      </MapContainer>
      <button
        type="button"
        onClick={locateMe}
        disabled={locating}
        className="absolute bottom-2 end-2 z-[1000] flex items-center gap-1.5 rounded-control border border-line bg-surface px-2.5 py-1.5 text-xs font-medium text-ink shadow-sm transition-colors hover:bg-page disabled:opacity-60"
      >
        <IconLocateMe size={15} className={locating ? "animate-pulse text-primary" : "text-primary"} />
        {m.common.locateMe}
      </button>
    </div>
  );
}
