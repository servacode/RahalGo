"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **الخريطة من السائقين** (قرارُ المالك ٢٠٢٦-١١-٠٣)
 * ══════════════════════════════════════════════════════════════════════
 *
 * نقاطُ مشي السائقين المجهولة تُظهر أين يمشون فعلاً. **طبقتان**: كثافةُ المشي،
 * وشوارعُ مرشَّحة (نقاطٌ بعيدةٌ عن كلّ طريقٍ معروفٍ وتتكرّر بأكثر من يوم).
 * **والمدير يوافق على الخطّ أو يرفضه**، والموافَقُ عليه يُصدَّر ملفّاً تقرؤه إعادةُ بناء الخريطة.
 * والكشف كلُّه في الخادم (`internal/maplearn`) — هنا عرضٌ وقرار.
 *
 * **وقسمُ «معالم السائقين»** (قرارُ المالك ٢٠٢٦-١١-٠٥) — ما يرسلونه من «أضف معلماً» وعدُّ يومهم،
 * **ويُفتح من رابط ملخّص الصبح** (`?section=landmarks&day=`).
 */

import { useCallback, useEffect, useMemo, useRef, useState } from "react";
import dynamic from "next/dynamic";
import { getMessages, defaultLocale, fmtNum, errorText } from "@rahalgo/i18n";
import {
  PageContainer,
  Button,
  Checkbox,
  Input,
  Alert,
  Badge,
  LoadingState,
  ReloadState,
  AdminEmpty,
  IconRoute,
  IconCheck,
  IconClose,
  useLiveData,
} from "@rahalgo/ui";
import { PageHero } from "@/components/admin/PageHero";
import { Tile } from "@/components/admin/DashParts";
import { api, apiFile } from "@/lib/api";
import { useCanCall } from "@/lib/policy";
import type { FeatureCollection, LayerSpec } from "@/components/admin/opsmap/canvas";
import { mapIcons, toneColor } from "@/components/admin/opsmap/palette";
import { PlacesPanel } from "./PlacesPanel";
import { LandmarksPanel } from "./LandmarksPanel";
import { OnewayPanel, SpeedsPanel } from "./RoadsPanel";

const OpsMapCanvas = dynamic(
  () => import("@/components/admin/opsmap/canvas").then((x) => x.OpsMapCanvas),
  { ssr: false },
);

const m = getMessages(defaultLocale);
const T = m.admin.mapLearning;

type Status = "pending" | "approved" | "rejected";
type Section = "streets" | "places" | "landmarks" | "oneway" | "speeds";
const SECTIONS: Section[] = ["streets", "places", "landmarks", "oneway", "speeds"];

interface Summary {
  points: number;
  unmeasured: number;
  far_points: number;
  pending: number;
  approved: number;
  rejected: number;
  running: boolean;
  can_measure: boolean;
  last_run?: { at: string; measured: number; candidates: number; failed: boolean };
}
interface Candidate {
  id: number;
  status: Status;
  name: string;
  length_m: number;
  points: number;
  days: number;
  first_day: string;
  last_day: string;
  geometry: { type: "LineString"; coordinates: Array<[number, number]> };
}

const EMPTY: FeatureCollection = { type: "FeatureCollection", features: [] };

function CandidateRow({
  c,
  active,
  canDecide,
  busy,
  onPick,
  onDecide,
}: {
  c: Candidate;
  active: boolean;
  canDecide: boolean;
  busy: boolean;
  onPick: () => void;
  onDecide: (approve: boolean, name: string) => void;
}) {
  const [name, setName] = useState(c.name);
  return (
    <li
      className={`rounded-control border px-2 py-2 text-sm ${active ? "border-accent" : "border-line-soft"}`}
    >
      <Button variant="ghost" size="sm" className="w-full justify-start text-start" onClick={onPick}>
        <span className="font-bold">{c.name || T.unnamed}</span>
        <span className="ms-2 text-ink-muted">
          {T.rowStats
            .replace("{len}", fmtNum(Math.round(c.length_m)))
            .replace("{days}", fmtNum(c.days))
            .replace("{pts}", fmtNum(c.points))}
        </span>
      </Button>
      {c.status === "pending" && canDecide && (
        <div className="mt-2 flex flex-wrap items-end gap-2">
          <Input
            id={`cand-name-${c.id}`}
            label={T.nameLabel}
            value={name}
            maxLength={80}
            onChange={(e) => setName(e.target.value)}
          />
          <Button size="sm" disabled={busy} onClick={() => onDecide(true, name)}>
            <IconCheck size={14} /> {T.approve}
          </Button>
          <Button size="sm" variant="secondary" disabled={busy} onClick={() => onDecide(false, name)}>
            <IconClose size={14} /> {T.reject}
          </Button>
        </div>
      )}
      {c.status !== "pending" && (
        <Badge variant={c.status === "approved" ? "success" : "neutral"} className="mt-2">
          {c.status === "approved" ? T.statusApproved : T.statusRejected}
        </Badge>
      )}
    </li>
  );
}

export default function MapLearningPage() {
  const canCall = useCanCall();
  const canManage = canCall("POST", "/ops-map/street-candidates/run");
  const canDecideLandmark = canCall("POST", "/ops-map/landmark-suggestions/{id}/decide");
  // **وعددُ المعالم بانتظار الموافقة على زرّ قسمها** (مراجعة ٢٠٢٦-١١-٠٥) — كانت شارةُ القائمة
  // الجانبيّة تفتح الصفحةَ على «شوارع» وزرُّ المعالم ثالثٌ بلا عدد. **والقراءةُ بصلاحيّة القائمة.**
  const canSeeLandmarks = canCall("GET", "/ops-map/landmark-suggestions");
  const { data: landmarkCount } = useLiveData<{ pending: number }>(
    () =>
      canSeeLandmarks
        ? api<{ pending: number }>("/api/v1/admin/ops-map/landmark-suggestions/count")
        : Promise.resolve({ pending: 0 }),
    ["landmark"],
    [canSeeLandmarks],
  );
  const landmarkPending = landmarkCount?.pending ?? 0;

  const [status, setStatus] = useState<Status>("pending");
  const [days, setDays] = useState(30);
  const [farOnly, setFarOnly] = useState(false);
  const [showHeat, setShowHeat] = useState(true);
  const [sum, setSum] = useState<Summary | null>(null);
  const [cands, setCands] = useState<Candidate[] | null>(null);
  const [heat, setHeat] = useState<FeatureCollection>(EMPTY);
  const [picked, setPicked] = useState<number | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const [actionErr, setActionErr] = useState<string>("");
  const [busy, setBusy] = useState(false);
  const [tick, setTick] = useState(0);
  const [section, setSection] = useState<Section>("streets");
  const [placesTick, setPlacesTick] = useState(0);
  const [placesRunning, setPlacesRunning] = useState(false);
  const [roadsTick, setRoadsTick] = useState(0);
  const [roadsRunning, setRoadsRunning] = useState(false);
  const [stores, setStores] = useState<Array<{ id: string; name: string; lat: number; lng: number }>>([]);
  const [landmarkDay, setLandmarkDay] = useState<string | undefined>(undefined);

  // **رابطُ ملخّص الصبح يفتح قسمَ المعالم على يومه** — `?section=landmarks&day=`.
  // **ومن جاء بلا قسمٍ والمعالمُ تنتظر فُتح على المعالم** — ما لم يختر قسماً بيده.
  const sectionChosen = useRef(false);
  const pickSection = useCallback((s: Section) => {
    sectionChosen.current = true;
    setSection(s);
  }, []);
  useEffect(() => {
    const sp = new URLSearchParams(window.location.search);
    const asked = sp.get("section");
    if (asked && (SECTIONS as string[]).includes(asked)) pickSection(asked as Section);
    const d = sp.get("day");
    if (d && /^\d{4}-\d{2}-\d{2}$/.test(d)) setLandmarkDay(d);
  }, [pickSection]);
  useEffect(() => {
    if (landmarkPending > 0 && !sectionChosen.current) pickSection("landmarks");
  }, [landmarkPending, pickSection]);

  useEffect(() => {
    let alive = true;
    setErr(null);
    Promise.all([
      api<Summary>("/api/v1/admin/ops-map/street-candidates/summary"),
      api<{ candidates: Candidate[] }>(`/api/v1/admin/ops-map/street-candidates?status=${status}`),
    ])
      .then(([s, c]) => {
        if (!alive) return;
        setSum(s);
        setCands(c.candidates);
      })
      .catch((e: unknown) => alive && setErr(errorText(e)));
    return () => {
      alive = false;
    };
  }, [status, tick]);

  useEffect(() => {
    let alive = true;
    api<FeatureCollection>(`/api/v1/admin/ops-map/street-candidates/heat?days=${days}&far=${farOnly ? 1 : 0}`)
      .then((r) => alive && setHeat(r))
      .catch(() => alive && setHeat(EMPTY));
    return () => {
      alive = false;
    };
  }, [days, farOnly, tick]);

  // أسماءُ المتاجر على الخريطة (طلبُ المالك) — تأتي مع رد الأماكن، والمقروءُ منه المتاجرُ وحدَها.
  useEffect(() => {
    let alive = true;
    api<{ stores: Array<{ id: string; name: string; lat: number; lng: number }> }>(
      "/api/v1/admin/ops-map/place-candidates?status=approved&source=stops",
    )
      .then((r) => alive && setStores(r.stores))
      .catch(() => alive && setStores([]));
    return () => {
      alive = false;
    };
  }, []);

  const layers = useMemo<LayerSpec[]>(() => {
    const out: LayerSpec[] = [
      {
        id: "stores",
        kind: "symbol",
        order: 20,
        visible: true,
        color: toneColor("store-open"),
        labelField: "name",
        data: {
          type: "FeatureCollection",
          features: stores.map((s) => ({
            type: "Feature" as const,
            geometry: { type: "Point", coordinates: [s.lng, s.lat] },
            properties: { id: s.id, name: s.name, icon: "store-open" },
          })),
        },
      },
      {
        id: "heat",
        kind: "heat",
        order: 10,
        visible: showHeat,
        color: toneColor("driver-busy"),
        weightField: "w",
        data: heat,
      },
    ];
    const list = cands ?? [];
    out.push({
      id: "cands",
      kind: "line",
      order: 40,
      visible: true,
      dashed: false,
      lineWidth: 4,
      color: toneColor(status === "approved" ? "driver-available" : "order"),
      data: {
        type: "FeatureCollection",
        features: list
          .filter((c) => c.id !== picked)
          .map((c) => ({ type: "Feature" as const, geometry: c.geometry, properties: { id: c.id } })),
      },
    });
    out.push({
      id: "picked",
      kind: "line",
      order: 50,
      visible: true,
      dashed: false,
      lineWidth: 7,
      color: toneColor("order-stuck"),
      data: {
        type: "FeatureCollection",
        features: list
          .filter((c) => c.id === picked)
          .map((c) => ({ type: "Feature" as const, geometry: c.geometry, properties: { id: c.id } })),
      },
    });
    return out;
  }, [heat, cands, picked, showHeat, status, stores]);

  const focus = useMemo(() => {
    const c = (cands ?? []).find((x) => x.id === picked);
    const mid = c?.geometry.coordinates[Math.floor(c.geometry.coordinates.length / 2)];
    return mid ? { lng: mid[0], lat: mid[1], zoom: 17 } : undefined;
  }, [cands, picked]);

  const decide = useCallback(async (id: number, approve: boolean, name: string) => {
    setBusy(true);
    setActionErr("");
    try {
      await api(`/api/v1/admin/ops-map/street-candidates/${id}/decide`, {
        method: "POST",
        body: JSON.stringify({ approve, name }),
      });
      setPicked(null);
      setTick((x) => x + 1);
    } catch (e) {
      setActionErr(errorText(e));
    } finally {
      setBusy(false);
    }
  }, []);

  async function run() {
    setBusy(true);
    setActionErr("");
    try {
      await api("/api/v1/admin/ops-map/street-candidates/run", { method: "POST" });
      setTick((x) => x + 1);
    } catch (e) {
      setActionErr(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  async function runPlaces() {
    setBusy(true);
    setActionErr("");
    try {
      await api("/api/v1/admin/ops-map/place-candidates/run", { method: "POST" });
      setPlacesRunning(true);
      for (let i = 0; i < 60; i++) {
        await new Promise((r) => setTimeout(r, 3000));
        const r = await api<{ running: boolean }>("/api/v1/admin/ops-map/place-candidates?status=pending");
        if (!r.running) break;
      }
    } catch (e) {
      setActionErr(errorText(e));
    } finally {
      setPlacesRunning(false);
      setBusy(false);
      setPlacesTick((x) => x + 1);
    }
  }

  async function runRoads() {
    setBusy(true);
    setActionErr("");
    try {
      await api("/api/v1/admin/ops-map/road-learning/run", { method: "POST" });
      setRoadsRunning(true);
      for (let i = 0; i < 120; i++) {
        await new Promise((r) => setTimeout(r, 3000));
        const r = await api<{ running: boolean }>("/api/v1/admin/ops-map/road-learning/summary");
        if (!r.running) break;
      }
    } catch (e) {
      setActionErr(errorText(e));
    } finally {
      setRoadsRunning(false);
      setBusy(false);
      setRoadsTick((x) => x + 1);
    }
  }

  async function downloadOneway(format: "osm" | "geojson") {
    setActionErr("");
    try {
      const res = await apiFile(`/api/v1/admin/ops-map/oneway-candidates/export?format=${format}`);
      const url = URL.createObjectURL(await res.blob());
      const a = document.createElement("a");
      a.href = url;
      a.download = `rahalgo-oneway.${format}`;
      a.click();
      URL.revokeObjectURL(url);
    } catch (e) {
      setActionErr(errorText(e));
    }
  }

  async function download(format: "osm" | "geojson") {
    setActionErr("");
    try {
      const res = await apiFile(`/api/v1/admin/ops-map/street-candidates/export?format=${format}`);
      const url = URL.createObjectURL(await res.blob());
      const a = document.createElement("a");
      a.href = url;
      a.download = `rahalgo-streets.${format}`;
      a.click();
      URL.revokeObjectURL(url);
    } catch (e) {
      setActionErr(errorText(e));
    }
  }

  return (
    <PageContainer>
      <PageHero
        icon={IconRoute}
        title={T.title}
        actions={
          canManage && (section === "oneway" || section === "speeds") ? (
            <>
              <Button variant="secondary" size="sm" disabled={busy || roadsRunning} onClick={runRoads}>
                {roadsRunning ? T.oneway.running : T.oneway.run}
              </Button>
              {section === "oneway" && (
                <>
                  <Button variant="secondary" size="sm" onClick={() => downloadOneway("osm")}>
                    {T.oneway.exportOsm}
                  </Button>
                  <Button variant="secondary" size="sm" onClick={() => downloadOneway("geojson")}>
                    {T.oneway.exportList}
                  </Button>
                </>
              )}
            </>
          ) : section === "landmarks" ? undefined : canManage && section === "places" ? (
            <Button variant="secondary" size="sm" disabled={busy || placesRunning} onClick={runPlaces}>
              {placesRunning ? T.places.running : T.places.run}
            </Button>
          ) : canManage ? (
            <>
              <Button variant="secondary" size="sm" disabled={busy || !sum?.can_measure || sum?.running} onClick={run}>
                {sum?.running ? T.running : T.run}
              </Button>
              <Button variant="secondary" size="sm" onClick={() => download("osm")}>
                {T.exportOsm}
              </Button>
              <Button variant="secondary" size="sm" onClick={() => download("geojson")}>
                {T.exportGeojson}
              </Button>
            </>
          ) : undefined
        }
      />
      {actionErr && <Alert tone="error">{actionErr}</Alert>}
      <div className="mb-3 flex flex-wrap gap-2">
        {SECTIONS.map((s) => (
          <Button key={s} size="sm" variant={section === s ? "primary" : "secondary"} onClick={() => pickSection(s)}>
            {T.sections[s]}
            {s === "landmarks" && landmarkPending > 0 && (
              <Badge variant="danger" className="ms-1">
                {fmtNum(landmarkPending)}
              </Badge>
            )}
          </Button>
        ))}
      </div>
      {section === "places" ? (
        <PlacesPanel canManage={canManage} runSignal={placesTick} />
      ) : section === "landmarks" ? (
        <LandmarksPanel key={landmarkDay ?? ""} canDecide={canDecideLandmark} initialDay={landmarkDay} />
      ) : section === "oneway" ? (
        <OnewayPanel canManage={canManage} runSignal={roadsTick} stores={stores} />
      ) : section === "speeds" ? (
        <SpeedsPanel runSignal={roadsTick} stores={stores} />
      ) : err ? (
        <ReloadState label={err} onRetry={() => setTick((x) => x + 1)} />
      ) : !sum || !cands ? (
        <LoadingState />
      ) : (
        <>
          <div className="mb-3 grid grid-cols-2 gap-2 sm:grid-cols-3 xl:grid-cols-5">
            <Tile icon={IconRoute} label={T.tiles.points} value={fmtNum(sum.points)} />
            <Tile icon={IconRoute} label={T.tiles.far} value={fmtNum(sum.far_points)} />
            <Tile icon={IconRoute} label={T.tiles.pending} value={fmtNum(sum.pending)} />
            <Tile icon={IconCheck} label={T.tiles.approved} value={fmtNum(sum.approved)} />
            <Tile icon={IconClose} label={T.tiles.rejected} value={fmtNum(sum.rejected)} />
          </div>
          {!sum.can_measure && <Alert tone="warning">{T.noRoads}</Alert>}
          <div className="grid grid-cols-1 gap-3 lg:grid-cols-3">
            <div className="surface relative h-[65vh] overflow-hidden lg:col-span-2">
              <OpsMapCanvas layers={layers} icons={mapIcons()} unavailableLabel={T.mapUnavailable} focus={focus} />
            </div>
            <div className="surface p-3">
              <div className="mb-3 flex flex-wrap items-end gap-2">
                <Checkbox id="ml-heat" label={T.heatLayer} checked={showHeat} onChange={(e) => setShowHeat(e.target.checked)} />
                <Checkbox id="ml-far" label={T.farOnly} checked={farOnly} onChange={(e) => setFarOnly(e.target.checked)} />
                {[7, 30, 90].map((d) => (
                  <Button key={d} size="sm" variant={days === d ? "primary" : "secondary"} onClick={() => setDays(d)}>
                    {T.daysN.replace("{n}", fmtNum(d))}
                  </Button>
                ))}
              </div>
              <div className="mb-3 flex flex-wrap gap-2">
                {(["pending", "approved", "rejected"] as Status[]).map((s) => (
                  <Button
                    key={s}
                    size="sm"
                    variant={status === s ? "primary" : "secondary"}
                    onClick={() => {
                      setPicked(null);
                      setStatus(s);
                    }}
                  >
                    {T.tabs[s]} {fmtNum(sum[s])}
                  </Button>
                ))}
              </div>
              {cands.length === 0 ? (
                <AdminEmpty icon={IconRoute} title={T.empty} />
              ) : (
                <ul className="max-h-[50vh] space-y-2 overflow-y-auto">
                  {cands.map((c) => (
                    <CandidateRow
                      key={c.id}
                      c={c}
                      active={c.id === picked}
                      canDecide={canManage}
                      busy={busy}
                      onPick={() => setPicked(c.id)}
                      onDecide={(approve, name) => void decide(c.id, approve, name)}
                    />
                  ))}
                </ul>
              )}
            </div>
          </div>
        </>
      )}
    </PageContainer>
  );
}
