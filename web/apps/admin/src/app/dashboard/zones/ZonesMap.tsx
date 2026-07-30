"use client";

/**
 * خريطة مناطق التغطية — نموذج الدوائر: مركز + نصف قطر.
 * النقر يحدد/ينقل المركز أثناء التحرير، والدائرة تُعاين حياً مع تغيير نصف القطر.
 */

import { MapContainer, Circle, CircleMarker, useMapEvents } from "react-leaflet";
import "leaflet/dist/leaflet.css";
import FallbackTileLayer from "@/components/map/FallbackTileLayer";

const RAQQA_CENTER: [number, number] = [35.9528, 39.0079];

export interface ZoneShape {
  id: string;
  name: string;
  lat: number;
  lng: number;
  radius_m: number;
  delivery_fee: number;
  active: boolean;
}

function ClickCapture({ onClick }: { onClick: (lat: number, lng: number) => void }) {
  useMapEvents({
    click(e) {
      onClick(e.latlng.lat, e.latlng.lng);
    },
  });
  return null;
}

export default function ZonesMap({
  zones,
  editing,
  draft,
  selectedID,
  onMapClick,
  onZoneClick,
}: {
  zones: ZoneShape[];
  editing: boolean;
  draft: { lat: number; lng: number; radiusM: number } | null;
  selectedID: string | null;
  onMapClick: (lat: number, lng: number) => void;
  onZoneClick: (id: string) => void;
}) {
  return (
    <MapContainer
      center={RAQQA_CENTER}
      zoom={13}
      className="h-full w-full"
      style={{ cursor: editing ? "crosshair" : undefined }}
    >
      <FallbackTileLayer />
      {editing && <ClickCapture onClick={onMapClick} />}

      {zones.map((z) => (
        <Circle
          key={z.id}
          center={[z.lat, z.lng]}
          radius={z.radius_m}
          pathOptions={{
            color: z.id === selectedID ? "#F59E0B" : z.active ? "#0E7490" : "#94A3B8",
            fillOpacity: z.id === selectedID ? 0.3 : 0.15,
            weight: z.id === selectedID ? 3 : 2,
          }}
          eventHandlers={{ click: () => onZoneClick(z.id) }}
        />
      ))}

      {draft && (
        <>
          <Circle
            center={[draft.lat, draft.lng]}
            radius={draft.radiusM}
            pathOptions={{ color: "#D97706", fillColor: "#F59E0B", fillOpacity: 0.2, dashArray: "8" }}
          />
          <CircleMarker
            center={[draft.lat, draft.lng]}
            radius={6}
            pathOptions={{ color: "#D97706", fillColor: "#F59E0B", fillOpacity: 1 }}
          />
        </>
      )}
    </MapContainer>
  );
}
