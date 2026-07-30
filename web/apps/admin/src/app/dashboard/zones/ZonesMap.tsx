"use client";

/**
 * خريطة مناطق التغطية — OSM (رابط البلاطات من الإعدادات، ذاتي الاستضافة عند النشر).
 * الرسم: كل نقرة تضيف نقطة، و"إنهاء الرسم" يغلق المضلع.
 */

import { MapContainer, Polygon, Polyline, CircleMarker, useMapEvents } from "react-leaflet";
import "leaflet/dist/leaflet.css";
import FallbackTileLayer from "@/components/map/FallbackTileLayer";

// مركز مدينة الرقة
const RAQQA_CENTER: [number, number] = [35.9528, 39.0079];

export interface ZoneShape {
  id: string;
  name: string;
  polygon: [number, number][]; // [lng, lat] كما في الخادم
  delivery_fee: number;
  active: boolean;
}

function toLatLng(ring: [number, number][]): [number, number][] {
  return ring.map(([lng, lat]) => [lat, lng]);
}

function ClickCapture({ onClick }: { onClick: (lng: number, lat: number) => void }) {
  useMapEvents({
    click(e) {
      onClick(e.latlng.lng, e.latlng.lat);
    },
  });
  return null;
}

export default function ZonesMap({
  zones,
  drawing,
  draft,
  selectedID,
  onMapClick,
  onZoneClick,
}: {
  zones: ZoneShape[];
  drawing: boolean;
  draft: [number, number][];
  selectedID: string | null;
  onMapClick: (lng: number, lat: number) => void;
  onZoneClick: (id: string) => void;
}) {
  return (
    <MapContainer
      center={RAQQA_CENTER}
      zoom={13}
      className="h-full w-full"
      style={{ cursor: drawing ? "crosshair" : undefined }}
    >
      <FallbackTileLayer />
      {drawing && <ClickCapture onClick={onMapClick} />}

      {zones.map((z) => (
        <Polygon
          key={z.id}
          positions={toLatLng(z.polygon)}
          pathOptions={{
            color: z.id === selectedID ? "#F59E0B" : z.active ? "#0E7490" : "#94A3B8",
            fillOpacity: z.id === selectedID ? 0.35 : 0.18,
            weight: z.id === selectedID ? 3 : 2,
          }}
          eventHandlers={{ click: () => onZoneClick(z.id) }}
        />
      ))}

      {draft.length > 1 && (
        <Polyline positions={toLatLng(draft)} pathOptions={{ color: "#F59E0B", dashArray: "6" }} />
      )}
      {draft.map(([lng, lat], i) => (
        <CircleMarker
          key={i}
          center={[lat, lng]}
          radius={5}
          pathOptions={{ color: "#D97706", fillColor: "#F59E0B", fillOpacity: 1 }}
        />
      ))}
    </MapContainer>
  );
}
