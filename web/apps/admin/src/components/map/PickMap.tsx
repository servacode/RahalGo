"use client";

/** خريطة صغيرة لاختيار نقطة (دبوس) — تُستخدم في نافذة المتجر وغيرها. */

import { MapContainer, CircleMarker, useMapEvents } from "react-leaflet";
import "leaflet/dist/leaflet.css";
import FallbackTileLayer from "./FallbackTileLayer";

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
  const center: [number, number] = lat != null && lng != null ? [lat, lng] : RAQQA_CENTER;
  return (
    <MapContainer center={center} zoom={14} className="h-52 w-full" style={{ cursor: "crosshair" }}>
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
  );
}
