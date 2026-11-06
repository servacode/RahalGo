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
  Button,
  Input,
  Badge,
  IconAdd,
  IconDelete,
  IconClose,
  Confirm,
  themeColor,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import type {
  FeatureCollection,
  HandlePhase,
  LayerSpec,
  MapHandle,
} from "@/components/admin/opsmap/canvas";
import { circleRing, coverageMask, midpoints, openRing, outerRings, ringSelfIntersects, type Pt } from "@/components/admin/opsmap/zonegeo";
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
  /** `edit`: تعديلُ شكلٍ قائم بمقابض (٢٠٢٦-١١-٠٦) · `draw`: رسمٌ نقطةً نقطة. */
  mode: "draw" | "edit";
  /** سجلُّ التراجع في وضع التعديل — الحلقاتُ السابقة. */
  history: [number, number][][];
  /** الرأسُ المحدَّد للحذف (لمساً). */
  selected: number | null;
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
  /** **تغميقُ ما خارج المناطق الفعّالة** (٢٠٢٦-١١-٠٦). */
  const [maskOn, setMaskOn] = useState(true);
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

  const editing = drawing?.mode === "edit";
  const selfCross = useMemo(
    () => (drawing && drawing.ring.length >= 4 ? ringSelfIntersects(drawing.ring) : false),
    [drawing],
  );

  // ── الطبقات ─────────────────────────────────────────────────
  //
  // **المفعّلةُ بنفسجيّةٌ عريضة والموقوفةُ رماديّة** — **ومن رأى منطقةً
  // بلونٍ واحدٍ لم يعرف أيّها يستقبل الطلبات.**
  const layers = useMemo<LayerSpec[]>(() => {
    const on = themeColor("cta-end");
    const off = themeColor("ink-muted");
    const byActive = ["case", ["==", ["get", "active"], true], on, off];
    const ring = drawing?.ring ?? [];
    // **التغميق**: المناطقُ الفعّالة (بلا القيد تعديلُه) + الشكلُ الجاري تعديلُه إن كان صالحاً.
    const holes: Pt[][] = [];
    for (const z of zones) {
      if (!z.active || z.id === drawing?.id) continue;
      if (z.shape === "radius") holes.push(circleRing(z.lng, z.lat, z.radius_m));
      else if (z.area) holes.push(...outerRings(z.area).map(openRing));
    }
    if (editing && ring.length >= 3 && !selfCross) holes.push(ring);
    const maskGeom = maskOn ? coverageMask(holes) : null;
    return [
      {
        id: "zones-mask",
        kind: "mask",
        order: 5,
        visible: maskGeom != null,
        color: themeColor("map-label"),
        fillOpacity: 0.35,
        data: fc(
          maskGeom ? [{ type: "Feature" as const, geometry: maskGeom, properties: {} }] : [],
        ),
      },
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
        visible: ring.length > 0 && !editing,
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
  }, [zones, drawing, maskOn, editing, selfCross]);

  function startNew() {
    setSelectedID(null);
    setError("");
    setDrawing({ id: null, name: "", ring: [], mode: "draw", history: [], selected: null });
  }

  function startRedraw(z: Zone) {
    setError("");
    setDrawing({ id: z.id, name: z.name, ring: [], mode: "draw", history: [], selected: null });
  }

  /** **حلقةُ المنطقة القابلةُ للتعديل** — مضلَّعٌ واحد؛ ومتعدّدُ القطع يُعاد رسمُه. */
  function editableRing(z: Zone): [number, number][] | null {
    if (z.shape !== "polygon" || !z.area) return null;
    const rings = outerRings(z.area);
    if (rings.length !== 1) return null;
    const r = openRing(rings[0]!);
    return r.length >= 3 ? r : null;
  }

  function startEdit(z: Zone) {
    const ring = editableRing(z);
    if (!ring) return;
    setError("");
    setDrawing({ id: z.id, name: z.name, ring, mode: "edit", history: [], selected: null });
  }

  // **مقابضُ التعديل**: رأسٌ لكلّ نقطة، ومقبضٌ في منتصف كلّ ضلع.
  const handles = useMemo<MapHandle[]>(() => {
    if (!drawing || drawing.mode !== "edit") return [];
    const ring = drawing.ring;
    return [
      ...ring.map((p, i) => ({
        id: `v${i}`, lng: p[0], lat: p[1], selected: drawing.selected === i, label: Z.vertexHandle,
      })),
      ...midpoints(ring).map((p, i) => ({
        id: `m${i}`, lng: p[0], lat: p[1], mid: true, label: Z.midHandle,
      })),
    ];
  }, [drawing]);

  function onHandleMove(id: string, lng: number, lat: number, phase: HandlePhase) {
    setDrawing((d) => {
      if (!d || d.mode !== "edit") return d;
      const i = Number(id.slice(1));
      if (id[0] === "v") {
        if (!d.ring[i]) return d;
        // بدايةُ السحب تحفظ الحلقةَ للتراجع، وما بعدها يحرّك الرأس.
        if (phase === "start") return { ...d, history: [...d.history, d.ring] };
        const ring = d.ring.map((p, k) => (k === i ? ([lng, lat] as [number, number]) : p));
        return { ...d, ring };
      }
      // منتصفُ ضلع: لا شيء أثناء السحب، ويُدرَج رأسٌ جديدٌ عند الإفلات.
      if (phase !== "end") return d;
      const ring = [...d.ring];
      ring.splice(i + 1, 0, [lng, lat]);
      return { ...d, history: [...d.history, d.ring], ring, selected: null };
    });
  }

  function removeVertex(i: number) {
    if (!drawing || drawing.mode !== "edit") return;
    if (drawing.ring.length <= 3) {
      setError(Z.minThree);
      return;
    }
    setError("");
    setDrawing({
      ...drawing,
      history: [...drawing.history, drawing.ring],
      ring: drawing.ring.filter((_, k) => k !== i),
      selected: null,
    });
  }

  function undoEdit() {
    setDrawing((d) => {
      if (!d || d.history.length === 0) return d;
      return { ...d, ring: d.history[d.history.length - 1]!, history: d.history.slice(0, -1), selected: null };
    });
  }

  async function saveDrawing() {
    if (!drawing || drawing.ring.length < 3 || !drawing.name.trim()) return;
    if (ringSelfIntersects(drawing.ring)) {
      setError(Z.selfIntersects);
      return;
    }
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
      setError(
        (err as { body?: { code?: string } })?.body?.code === "invalid_geometry"
          ? Z.serverInvalid
          : errorText(err),
      );
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
    <div className="flex flex-col">
      <div className="mb-4 flex flex-wrap items-center justify-between gap-3">
        <h2 className="heading-card">{Z.title}</h2>
        <div className="flex flex-wrap items-center gap-2">
          <Button variant="secondary" onClick={() => setMaskOn((v) => !v)} aria-pressed={maskOn}>
            {maskOn ? Z.maskOff : Z.maskOn}
          </Button>
          {isAdmin && !drawing && (
            <Button onClick={startNew} className="flex items-center gap-1.5">
              <IconAdd size={16} />
              {Z.drawNew}
            </Button>
          )}
        </div>
      </div>

      {drawing && (
        <p className="mb-3 rounded-control bg-accent-tint px-3 py-2 text-sm text-accent-dark">
          {editing ? Z.editHint : drawing.id ? Z.redrawHint : Z.drawHint}
        </p>
      )}
      {error && <Alert className="mb-3">{error}</Alert>}

      {/* **الخريطةُ بارتفاعٍ محدود** (تدقيقُ ٢٠٢٦-١٠-٢٧) — كانت بارتفاع الشاشة كلّها جوّا الصفحة. */}
      <div className="flex flex-col gap-4 md:flex-row">
        <div className="h-80 flex-1 overflow-hidden rounded-card border border-line md:h-[32rem]">
          <OpsMapCanvas
            layers={layers}
            handles={handles}
            onHandleMove={onHandleMove}
            onHandleRemove={(id) => {
              if (id[0] === "v") removeVertex(Number(id.slice(1)));
            }}
            onHandleSelect={(id) => {
              if (id[0] !== "v") return;
              const i = Number(id.slice(1));
              setDrawing((d) => (d ? { ...d, selected: d.selected === i ? null : i } : d));
            }}
            unavailableLabel={m.map.unavailable}
            onMapClick={(lng, lat) => {
              if (drawing && !editing) setDrawing((d) => (d ? { ...d, ring: [...d.ring, [lng, lat]] } : d));
            }}
            onFeatureClick={(layerID, props) => {
              // **والنقرُ أثناء الرسم نقطةٌ لا اختيار** — يلتقطه `onMapClick`.
              if (drawing || layerID.startsWith("zones-draft")) return;
              if (typeof props.id === "string") setSelectedID(props.id);
            }}
          />
        </div>

        <aside className="w-full shrink-0 space-y-3 md:max-h-[32rem] md:w-80 md:overflow-y-auto">
          {/* ── محرِّرُ الرسم ─────────────────────────────────── */}
          {drawing && (
            <div className="space-y-3 surface !border-accent p-4">
              <div className="flex items-center justify-between">
                <h2 className="font-bold">{editing ? Z.editTitle : drawing.id ? Z.redraw : Z.drawNew}</h2>
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
              {selfCross && <Alert>{Z.selfIntersects}</Alert>}
              {editing && (
                <Button
                  variant="secondary"
                  disabled={drawing.selected == null}
                  onClick={() => drawing.selected != null && removeVertex(drawing.selected)}
                  className="flex items-center gap-1.5"
                >
                  <IconDelete size={15} />
                  {drawing.selected == null ? Z.selectPointHint : Z.removePoint}
                </Button>
              )}
              <div className="flex flex-wrap gap-2">
                <Button
                  onClick={saveDrawing}
                  disabled={busy || drawing.ring.length < 3 || !drawing.name.trim() || selfCross}
                >
                  {C.save}
                </Button>
                <Button
                  variant="secondary"
                  disabled={editing ? drawing.history.length === 0 : drawing.ring.length === 0}
                  onClick={() =>
                    editing ? undoEdit() : setDrawing({ ...drawing, ring: drawing.ring.slice(0, -1) })
                  }
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
                  {selected.shape === "polygon" && (
                    <Button
                      variant="secondary"
                      disabled={!editableRing(selected)}
                      title={editableRing(selected) ? undefined : Z.multiNotEditable}
                      onClick={() => startEdit(selected)}
                    >
                      {Z.editShape}
                    </Button>
                  )}
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
