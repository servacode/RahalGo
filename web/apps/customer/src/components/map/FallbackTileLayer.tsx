"use client";

/**
 * طبقة بلاطات بمصادر احتياطية: عند فشل المصدر الحالي بعدة بلاطات
 * ينتقل تلقائياً للمصدر التالي — لأن خوادم OSM المجانية غير مضمونة.
 * عند النشر يكون المصدر الأول خادمنا الذاتي (NEXT_PUBLIC_TILE_URL).
 */

import { useState } from "react";
import { TileLayer } from "react-leaflet";

const SOURCES = [
  process.env.NEXT_PUBLIC_TILE_URL,
  "https://tile.openstreetmap.org/{z}/{x}/{y}.png",
  "https://a.basemaps.cartocdn.com/rastertiles/voyager/{z}/{x}/{y}.png",
  "https://tile.openstreetmap.de/{z}/{x}/{y}.png",
].filter((u): u is string => !!u);

const ERRORS_BEFORE_SWITCH = 4;

export default function FallbackTileLayer() {
  const [srcIndex, setSrcIndex] = useState(0);
  const [errors, setErrors] = useState(0);

  return (
    <TileLayer
      key={srcIndex} // إعادة إنشاء الطبقة عند تبديل المصدر
      url={SOURCES[srcIndex] ?? SOURCES[0]!}
      attribution="&copy; OpenStreetMap"
      eventHandlers={{
        tileerror: () => {
          setErrors((e) => {
            if (e + 1 >= ERRORS_BEFORE_SWITCH && srcIndex < SOURCES.length - 1) {
              setSrcIndex((i) => i + 1);
              return 0;
            }
            return e + 1;
          });
        },
      }}
    />
  );
}
