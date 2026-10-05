"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **مناطقُ التغطية — تُرسم باليد** (قرارُ المالك ٢٠٢٦-١٠-٠٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * «بدنا رسم بالإيد، نرسم ع كيفنا المنطقة» — **والدائرةُ لا تلحق حدودَ
 * الرقّة**: نهرٌ وأطرافٌ وحيٌّ خارجَها نصلُه وآخرُ داخلَها لا نصله.
 *
 * **فالشاشةُ تعرض المناطقَ كلَّها** (المرسومةَ والدوائرَ القديمة) من باب
 * خريطة العمليات نفسِه، **وتُنشئ مضلَّعاً جديداً أو تعيد رسمَ قائمٍ مكانَه**
 * — بالمعرّف نفسِه، فطلباتُه القديمةُ لا تُشير إلى عدم.
 *
 * **ولا دوائرَ جديدة**: القديمةُ تُعرض وتُطفأ وتُحذف، **أو يُعاد رسمُها
 * فتصير مضلَّعاً.**
 */

import { useCallback, useEffect, useMemo, useState } from "react";
import dynamic from "next/dynamic";
import { getMessages, defaultLocale, errorText } from "@rahalgo/i18n";
import {
  Alert,
  PageHeader,
  Button,
  Input,
  Badge,
  IconAdd,
  IconDelete,
  IconZones,
  IconClose,
  Confirm,
  themeColor,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import type { FeatureCollection, LayerSpec } from "@/components/admin/opsmap/canvas";
import ZoneHoursCard from "./ZoneHours";

const OpsMapCanvas = dynamic(
  () => import("@/components/admin/opsmap/canvas").then((x) => x.OpsMapCanvas),
  { ssr: false },
);

const m = getMessages(defaultLocale);
const Z = m.admin.zones;
const C = m.admin.opsMap.coverage;

interface Zone {
  id: string;
  name: string;
  shape: "radius" | "polygon";
  active: boolean;
  lat: number;
  lng: number;
  radius_m: number;
  area?: unknown;
  min_order: number;
  city_id?: string;
}

/** **ما يُرسم الآن** — منطقةٌ جديدة، أو إعادةُ رسمِ قائمةٍ مكانَها. */
interface Drawing {
  id: string | null;
  name: string;
  ring: [number, number][];
}

const fc = (features: FeatureCollection["features"]): FeatureCollection => ({
  type: "FeatureCollection",
  features,
});

export default function ZonesPanel() {
  const { can } = useAuth();
  // **ورسمُ المناطق جغرافيا** — `settings.general.manage`.
  const isAdmin = can("settings.general.manage");

  const [zones, setZones] = useState<Zone[]>([]);
  const [error, setError] = useState("");
  const [selectedID, setSelectedID] = useState<string | null>(null);
  const [drawing, setDrawing] = useState<Drawing | null>(null);
  const [busy, setBusy] = useState(false);
  /** **المنطقةُ المرشَّحةُ للحذف** — تنتظر تأكيداً من نافذة المنصّة. */
  const [pendingDelete, setPendingDelete] = useState<Zone | null>(null);

  const load = useCallback(async () => {
    try {
      const r = await api<{ zones: Zone[] }>("/api/v1/admin/ops-map/coverage");
      setZones(r.zones ?? []);
      setError("");
    } catch (err) {
      setError(errorText(err));
    }
  }, []);

  useEffect(() => {
    void load();
  }, [load]);

  const selected = zones.find((z) => z.id === selectedID) ?? null;

  // ── الطبقات ─────────────────────────────────────────────────
  //
  // **المفعّلةُ بنفسجيّةٌ عريضة والموقوفةُ رماديّة** — **ومن رأى منطقةً
  // بلونٍ واحدٍ لم يعرف أيّها يستقبل الطلبات.**
  const layers = useMemo<LayerSpec[]>(() => {
    const on = themeColor("cta-end");
    const off = themeColor("ink-muted");
    const byActive = ["case", ["==", ["get", "active"], true], on, off];
    const ring = drawing?.ring ?? [];
    return [
      {
        id: "zones-radius",
        kind: "circle-m",
        radiusField: "radius_m",
        order: 10,
        visible: true,
        color: byActive,
        data: fc(
          zones
            .filter((z) => z.shape === "radius" && z.id !== drawing?.id)
            .map((z) => ({
              type: "Feature" as const,
              geometry: { type: "Point", coordinates: [z.lng, z.lat] },
              properties: { id: z.id, active: z.active, radius_m: z.radius_m },
            })),
        ),
      },
      {
        id: "zones-polygon",
        kind: "fill",
        order: 11,
        visible: true,
        color: byActive,
        lineWidth: 3,
        data: fc(
          zones
            .filter((z) => z.shape === "polygon" && z.area && z.id !== drawing?.id)
            .map((z) => ({
              type: "Feature" as const,
              geometry: z.area,
              properties: { id: z.id, active: z.active },
            })),
        ),
      },
      {
        id: "zones-draft",
        kind: "line",
        order: 12,
        visible: ring.length > 1,
        color: on,
        lineWidth: 4,
        data: fc(
          ring.length > 1
            ? [{
                type: "Feature" as const,
                geometry: { type: "LineString", coordinates: [...ring, ring[0]] },
                properties: {},
              }]
            : [],
        ),
      },
      // **ونقاطُ الرسم تُرى** — ومن لا يرى أين نقر نقر مرّتين.
      {
        id: "zones-draft-points",
        kind: "point",
        order: 13,
        visible: ring.length > 0,
        color: on,
        data: fc(
          ring.map((p, i) => ({
            type: "Feature" as const,
            geometry: { type: "Point", coordinates: p },
            properties: { i },
          })),
        ),
      },
    ];
  }, [zones, drawing]);

  function startNew() {
    setSelectedID(null);
    setError("");
    setDrawing({ id: null, name: "", ring: [] });
  }

  function startRedraw(z: Zone) {
    setError("");
    setDrawing({ id: z.id, name: z.name, ring: [] });
  }

  async function saveDrawing() {
    if (!drawing || drawing.ring.length < 3 || !drawing.name.trim()) return;
    setBusy(true);
    setError("");
    try {
      if (drawing.id) {
        // **وإعادةُ الرسم تحفظ ما سوى الشكل كما هو** — الحالُ والمدينةُ
        // والحدُّ؛ **والبابُ يكتب ما يصله، فما لا يُرسل يُمحى.**
        const z = zones.find((x) => x.id === drawing.id);
        await api(`/api/v1/admin/ops-map/coverage/${drawing.id}`, {
          method: "PUT",
          body: JSON.stringify({
            name: drawing.name.trim(),
            ring: drawing.ring,
            min_order: z?.min_order ?? 0,
            active: z?.active ?? true,
            city_id: z?.city_id ?? null,
          }),
        });
        setSelectedID(drawing.id);
      } else {
        const r = await api<{ id: string }>("/api/v1/admin/ops-map/coverage", {
          method: "POST",
          body: JSON.stringify({ name: drawing.name.trim(), ring: drawing.ring, min_order: 0 }),
        });
        setSelectedID(r.id);
      }
      setDrawing(null);
      await load();
    } catch (err) {
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  }

  async function toggleActive(z: Zone) {
    setError("");
    try {
      await api(`/api/v1/admin/ops-map/coverage/${z.id}/active`, {
        method: "POST",
        body: JSON.stringify({ active: !z.active }),
      });
      await load();
    } catch (err) {
      setError(errorText(err));
    }
  }

  /* **وحذفُ منطقةٍ يُسأل عنه بنافذة المنصّة** — و`confirm()` الأصليّةُ نافذةُ
     نظامٍ بخطّه ولغته وأزرارِها التي لا تقول فعلَها. (شكوى المالك
     ٢٠٢٦-٠٨-١٠.) */
  async function deleteZone(z: Zone) {
    setPendingDelete(null);
    setError("");
    try {
      await api(`/api/v1/admin/zones/${z.id}`, { method: "DELETE" });
      setSelectedID(null);
      await load();
    } catch (err) {
      setError(errorText(err));
    }
  }

  return (
    <div className="flex h-[calc(100vh-6rem)] flex-col lg:h-[calc(100vh-3rem)]">
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <PageHeader icon={IconZones} title={Z.title} />
        {isAdmin && !drawing && (
          <Button onClick={startNew} className="flex items-center gap-1.5">
            <IconAdd size={16} />
            {Z.drawNew}
          </Button>
        )}
      </div>

      {drawing && (
        <p className="mb-3 rounded-control bg-accent-tint px-3 py-2 text-sm text-accent-dark">
          {drawing.id ? Z.redrawHint : Z.drawHint}
        </p>
      )}
      {error && <Alert className="mb-3">{error}</Alert>}

      <div className="flex min-h-0 flex-1 flex-col gap-4 md:flex-row">
        <div className="min-h-64 flex-1 overflow-hidden rounded-card border border-line">
          <OpsMapCanvas
            layers={layers}
            unavailableLabel={m.map.unavailable}
            onMapClick={(lng, lat) => {
              if (drawing) setDrawing((d) => (d ? { ...d, ring: [...d.ring, [lng, lat]] } : d));
            }}
            onFeatureClick={(layerID, props) => {
              // **والنقرُ أثناء الرسم نقطةٌ لا اختيار** — يلتقطه `onMapClick`.
              if (drawing || layerID.startsWith("zones-draft")) return;
              if (typeof props.id === "string") setSelectedID(props.id);
            }}
          />
        </div>

        <aside className="w-full shrink-0 space-y-3 overflow-y-auto md:w-80">
          {/* ── محرِّرُ الرسم ─────────────────────────────────── */}
          {drawing && (
            <div className="space-y-3 surface !border-accent p-4">
              <div className="flex items-center justify-between">
                <h2 className="font-bold">{drawing.id ? Z.redraw : Z.drawNew}</h2>
                <button
                  onClick={() => setDrawing(null)}
                  className="text-ink-muted hover:text-ink"
                  aria-label={C.cancel}
                >
                  <IconClose size={18} />
                </button>
              </div>
              <Input
                id="z-name"
                label={Z.zoneName}
                required
                value={drawing.name}
                onChange={(e) => setDrawing({ ...drawing, name: e.target.value })}
                placeholder={Z.zoneNamePlaceholder}
              />
              <p className="text-sm text-ink-muted">
                <Badge variant="primary">{drawing.ring.length}</Badge> {Z.points}
                {drawing.ring.length < 3 && <> · {C.needThree}</>}
              </p>
              <div className="flex flex-wrap gap-2">
                <Button
                  onClick={saveDrawing}
                  disabled={busy || drawing.ring.length < 3 || !drawing.name.trim()}
                >
                  {C.save}
                </Button>
                <Button
                  variant="secondary"
                  disabled={drawing.ring.length === 0}
                  onClick={() => setDrawing({ ...drawing, ring: drawing.ring.slice(0, -1) })}
                >
                  {C.undo}
                </Button>
                <Button variant="secondary" onClick={() => setDrawing(null)}>
                  {C.cancel}
                </Button>
              </div>
            </div>
          )}

          {/* ── المنطقةُ المختارة ─────────────────────────────── */}
          {!drawing && selected && (
            <div className="space-y-3 surface !border-accent p-4">
              <div className="flex items-center justify-between gap-2">
                <h2 className="font-bold">{selected.name}</h2>
                <Badge variant={selected.active ? "success" : "danger"}>
                  {selected.active ? Z.active : Z.inactive}
                </Badge>
              </div>
              <p className="text-sm text-ink-muted">
                {selected.shape === "polygon"
                  ? C.polygon
                  : `${C.radius} · ${(selected.radius_m / 1000).toFixed(1)} ${Z.km}`}
              </p>
              {isAdmin && (
                <div className="flex flex-wrap gap-2">
                  <Button variant="secondary" onClick={() => startRedraw(selected)}>
                    {Z.redraw}
                  </Button>
                  <Button
                    variant={selected.active ? "danger" : "secondary"}
                    onClick={() => void toggleActive(selected)}
                  >
                    {selected.active ? C.disable : C.enable}
                  </Button>
                  <Button variant="ghost" onClick={() => setPendingDelete(selected)}>
                    <IconDelete size={15} className="text-danger" />
                  </Button>
                </div>
              )}
              {/* **ووقتُ المنطقة صفةٌ من صفاتها** (`ZH`، ٢٠٢٦-٠٩-١٤). */}
              <ZoneHoursCard zoneID={selected.id} may={isAdmin} />
            </div>
          )}

          {/* ── قائمةُ المناطق ────────────────────────────────── */}
          {zones.length === 0 && !drawing && (
            <p className="surface p-6 text-center text-sm text-ink-muted">{Z.empty}</p>
          )}
          {zones.map((z) => (
            <button
              key={z.id}
              onClick={() => !drawing && setSelectedID(z.id)}
              className={`w-full rounded-card border p-3 text-start transition-colors ${
                z.id === selectedID
                  ? "border-accent bg-accent-tint"
                  : "border-line bg-surface hover:border-primary-edge"
              }`}
            >
              <div className="flex items-center justify-between">
                <span className="font-bold">{z.name}</span>
                <Badge variant={z.active ? "success" : "danger"}>
                  {z.active ? Z.active : Z.inactive}
                </Badge>
              </div>
              <p className="mt-1 text-sm text-ink-muted">
                {z.shape === "polygon"
                  ? C.polygon
                  : `${C.radius} · ${(z.radius_m / 1000).toFixed(1)} ${Z.km}`}
              </p>
            </button>
          ))}
        </aside>
      </div>

      <Confirm
        open={!!pendingDelete}
        title={Z.deleteConfirm}
        body={pendingDelete?.name}
        confirmLabel={m.common.delete}
        onConfirm={() => pendingDelete && void deleteZone(pendingDelete)}
        onCancel={() => setPendingDelete(null)}
      />
    </div>
  );
}
