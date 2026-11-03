"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **تعلّمُ الطرق: اتّجاهٌ واحد وسرعةٌ حقيقيّة** (قرارُ المالك ٢٠٢٦-١١-٠٣ — الدفعة الثالثة)
 * ══════════════════════════════════════════════════════════════════════
 *
 * قسمان فوق أثر السائقين المطابَق على حافّات الطرق ليلاً:
 *   - **اتّجاهٌ واحد**: طرقٌ يمشيها ٩٥٪ فأكثر باتّجاهٍ واحد (٢٠ عبوراً و٥ سائقين و٥ أيّام على الأقل).
 *     المدير يوافق فيُصدَّر `oneway=yes`، أو يرفض. **لا يغيّر التوجيهَ الآن.**
 *   - **السرعةُ الحقيقيّة**: لونٌ من البطيء إلى السريع بحسب ساعةٍ بتوقيت دمشق، والبطيءُ باستمرارٍ مُعلَّم.
 * وأسماءُ المتاجر ظاهرةٌ على الخريطتين (طلبُ المالك). والحسابُ كلُّه في الخادم (`internal/maplearn/roadlearn*.go`).
 */

import { useCallback, useEffect, useMemo, useState } from "react";
import dynamic from "next/dynamic";
import { getMessages, defaultLocale, fmtNum, errorText } from "@rahalgo/i18n";
import {
  Button,
  Checkbox,
  Select,
  Alert,
  Badge,
  LoadingState,
  ReloadState,
  AdminEmpty,
  IconRoute,
  IconCheck,
  IconClose,
} from "@rahalgo/ui";
import { Tile } from "@/components/admin/DashParts";
import { api } from "@/lib/api";
import type { FeatureCollection, LayerSpec } from "@/components/admin/opsmap/canvas";
import { mapIcons, toneColor } from "@/components/admin/opsmap/palette";

const OpsMapCanvas = dynamic(
  () => import("@/components/admin/opsmap/canvas").then((x) => x.OpsMapCanvas),
  { ssr: false },
);

const T = getMessages(defaultLocale).admin.mapLearning;
const O = T.oneway;
const S = T.speeds;

type Status = "pending" | "approved" | "rejected";
type Store = { id: string; name: string; lat: number; lng: number };
type Line = { type: "LineString"; coordinates: Array<[number, number]> };

interface RoadSummary {
  running: boolean;
  can_match: boolean;
  days: number;
  edges: number;
  last_day: string | null;
  last_run?: { at: string; days: number; passes: number; oneway: number; failed: boolean };
}

function storesLayer(stores: Store[], visible: boolean): LayerSpec {
  return {
    id: "stores",
    kind: "symbol",
    order: 30,
    visible,
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
  };
}

function RunInfo({ sum }: { sum: RoadSummary }) {
  return (
    <>
      {!sum.can_match && <Alert tone="warning">{O.noMatcher}</Alert>}
      {sum.last_run && (
        <Alert tone={sum.last_run.failed ? "warning" : "info"}>
          {sum.last_run.failed
            ? O.lastRunFailed
            : O.lastRun
                .replace("{days}", fmtNum(sum.last_run.days))
                .replace("{passes}", fmtNum(sum.last_run.passes))
                .replace("{oneway}", fmtNum(sum.last_run.oneway))}
        </Alert>
      )}
    </>
  );
}

// ───────────────────────── اتّجاه واحد ─────────────────────────

interface Oneway {
  id: number;
  status: Status;
  name: string;
  main_passes: number;
  opp_passes: number;
  drivers: number;
  days: number;
  share: number;
  length_m: number;
  geometry: Line;
}
interface OnewayResp {
  candidates: Oneway[];
  counts: Record<Status, number>;
}

/** اتّجاهُ الخطّ بالكلمة: من أوّل نقطةٍ إلى آخرها. */
function dirWord(g: Line): string {
  const a = g.coordinates[0];
  const b = g.coordinates[g.coordinates.length - 1];
  if (!a || !b) return "";
  const dx = (b[0] - a[0]) * Math.cos((a[1] * Math.PI) / 180);
  const dy = b[1] - a[1];
  if (Math.abs(dx) >= Math.abs(dy)) return dx >= 0 ? O.dirs.e : O.dirs.w;
  return dy >= 0 ? O.dirs.n : O.dirs.s;
}

function OnewayRow({
  c,
  active,
  canDecide,
  busy,
  onPick,
  onDecide,
}: {
  c: Oneway;
  active: boolean;
  canDecide: boolean;
  busy: boolean;
  onPick: () => void;
  onDecide: (approve: boolean) => void;
}) {
  return (
    <li className={`rounded-control border px-2 py-2 text-sm ${active ? "border-accent" : "border-line-soft"}`}>
      <Button variant="ghost" size="sm" className="w-full justify-start text-start" onClick={onPick}>
        <span className="font-bold">{c.name || O.unnamed}</span>
        <span className="ms-2 text-ink-muted">{O.toward.replace("{dir}", dirWord(c.geometry))}</span>
      </Button>
      <div className="mt-1 px-2 text-xs text-ink-muted">
        {O.stats
          .replace("{pct}", fmtNum(Math.round(c.share * 100)))
          .replace("{main}", fmtNum(c.main_passes))
          .replace("{opp}", fmtNum(c.opp_passes))
          .replace("{drivers}", fmtNum(c.drivers))
          .replace("{days}", fmtNum(c.days))}
      </div>
      {c.status === "pending" && canDecide && (
        <div className="mt-2 flex flex-wrap gap-2">
          <Button size="sm" disabled={busy} onClick={() => onDecide(true)}>
            <IconCheck size={14} /> {T.approve}
          </Button>
          <Button size="sm" variant="secondary" disabled={busy} onClick={() => onDecide(false)}>
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

export function OnewayPanel({
  canManage,
  runSignal,
  stores,
}: {
  canManage: boolean;
  runSignal: number;
  stores: Store[];
}) {
  const [status, setStatus] = useState<Status>("pending");
  const [showStores, setShowStores] = useState(true);
  const [data, setData] = useState<OnewayResp | null>(null);
  const [sum, setSum] = useState<RoadSummary | null>(null);
  const [picked, setPicked] = useState<number | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const [actionErr, setActionErr] = useState("");
  const [busy, setBusy] = useState(false);
  const [tick, setTick] = useState(0);

  useEffect(() => {
    let alive = true;
    setErr(null);
    Promise.all([
      api<OnewayResp>(`/api/v1/admin/ops-map/oneway-candidates?status=${status}`),
      api<RoadSummary>("/api/v1/admin/ops-map/road-learning/summary"),
    ])
      .then(([d, s]) => {
        if (!alive) return;
        setData(d);
        setSum(s);
      })
      .catch((e: unknown) => alive && setErr(errorText(e)));
    return () => {
      alive = false;
    };
  }, [status, tick, runSignal]);

  const list = useMemo(() => data?.candidates ?? [], [data]);

  const layers = useMemo<LayerSpec[]>(() => {
    const fc = (features: FeatureCollection["features"]): FeatureCollection => ({ type: "FeatureCollection", features });
    const lineOf = (c: Oneway) => ({ type: "Feature" as const, geometry: c.geometry, properties: { id: c.id } });
    return [
      storesLayer(stores, showStores),
      {
        id: "oneway",
        kind: "line",
        order: 40,
        visible: true,
        dashed: false,
        lineWidth: 4,
        color: toneColor(status === "approved" ? "driver-available" : "order"),
        data: fc(list.filter((c) => c.id !== picked).map(lineOf)),
      },
      {
        id: "oneway-picked",
        kind: "line",
        order: 50,
        visible: true,
        dashed: false,
        lineWidth: 7,
        color: toneColor("order-stuck"),
        data: fc(list.filter((c) => c.id === picked).map(lineOf)),
      },
    ];
  }, [list, picked, showStores, status, stores]);

  const focus = useMemo(() => {
    const c = list.find((x) => x.id === picked);
    const mid = c?.geometry.coordinates[Math.floor(c.geometry.coordinates.length / 2)];
    return mid ? { lng: mid[0], lat: mid[1], zoom: 17 } : undefined;
  }, [list, picked]);

  const decide = useCallback(async (id: number, approve: boolean) => {
    setBusy(true);
    setActionErr("");
    try {
      await api(`/api/v1/admin/ops-map/oneway-candidates/${id}/decide`, {
        method: "POST",
        body: JSON.stringify({ approve }),
      });
      setPicked(null);
      setTick((x) => x + 1);
    } catch (e) {
      setActionErr(errorText(e));
    } finally {
      setBusy(false);
    }
  }, []);

  if (err) return <ReloadState label={err} onRetry={() => setTick((x) => x + 1)} />;
  if (!data || !sum) return <LoadingState />;

  return (
    <>
      {actionErr && <Alert tone="error">{actionErr}</Alert>}
      <RunInfo sum={sum} />
      <div className="mb-3 grid grid-cols-2 gap-2 sm:grid-cols-5">
        <Tile icon={IconRoute} label={O.tiles.days} value={fmtNum(sum.days)} />
        <Tile icon={IconRoute} label={O.tiles.edges} value={fmtNum(sum.edges)} />
        <Tile icon={IconRoute} label={T.tiles.pending} value={fmtNum(data.counts.pending)} />
        <Tile icon={IconCheck} label={T.tiles.approved} value={fmtNum(data.counts.approved)} />
        <Tile icon={IconClose} label={T.tiles.rejected} value={fmtNum(data.counts.rejected)} />
      </div>
      <div className="grid grid-cols-1 gap-3 lg:grid-cols-3">
        <div className="surface relative h-[65vh] overflow-hidden lg:col-span-2">
          <OpsMapCanvas layers={layers} icons={mapIcons()} unavailableLabel={T.mapUnavailable} focus={focus} />
        </div>
        <div className="surface p-3">
          <div className="mb-3 flex flex-wrap items-end gap-2">
            <Checkbox id="ow-stores" label={T.places.storesLayer} checked={showStores} onChange={(e) => setShowStores(e.target.checked)} />
          </div>
          <div className="mb-3 flex gap-2">
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
                {T.tabs[s]} {fmtNum(data.counts[s])}
              </Button>
            ))}
          </div>
          {list.length === 0 ? (
            <AdminEmpty icon={IconRoute} title={O.empty} />
          ) : (
            <ul className="max-h-[50vh] space-y-2 overflow-y-auto">
              {list.map((c) => (
                <OnewayRow
                  key={`${c.id}-${c.status}`}
                  c={c}
                  active={c.id === picked}
                  canDecide={canManage}
                  busy={busy}
                  onPick={() => setPicked(c.id)}
                  onDecide={(approve) => void decide(c.id, approve)}
                />
              ))}
            </ul>
          )}
        </div>
      </div>
    </>
  );
}

// ───────────────────────── السرعة الحقيقية ─────────────────────────

interface SpeedFeature {
  type: "Feature";
  geometry: Line;
  properties: { name: string; length_m: number; passes: number; kmh: number; slow: boolean };
}
interface SpeedSummary {
  edges: number;
  passes: number;
  slow_edges: number;
  buckets: Array<{ hour: number; weekend: boolean; edges: number; passes: number; avg_kmh: number }>;
}

/** لونٌ من البطيء (٥ كم/س) إلى السريع (٤٠ كم/س) من خاصّة `kmh`. */
function speedRamp(): unknown {
  return [
    "interpolate",
    ["linear"],
    ["get", "kmh"],
    5,
    toneColor("order-stuck"),
    20,
    toneColor("driver-busy"),
    40,
    toneColor("driver-available"),
  ];
}

export function SpeedsPanel({ runSignal, stores }: { runSignal: number; stores: Store[] }) {
  const [hour, setHour] = useState(-1);
  const [dayKind, setDayKind] = useState<"" | "0" | "1">("");
  const [showStores, setShowStores] = useState(true);
  const [onlySlow, setOnlySlow] = useState(false);
  const [feats, setFeats] = useState<SpeedFeature[] | null>(null);
  const [sum, setSum] = useState<RoadSummary | null>(null);
  const [city, setCity] = useState<SpeedSummary | null>(null);
  const [picked, setPicked] = useState<number | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const [tick, setTick] = useState(0);

  useEffect(() => {
    let alive = true;
    setErr(null);
    Promise.all([
      api<{ features: SpeedFeature[] }>(`/api/v1/admin/ops-map/road-speeds?hour=${hour}&weekend=${dayKind}`),
      api<RoadSummary>("/api/v1/admin/ops-map/road-learning/summary"),
      api<SpeedSummary>("/api/v1/admin/ops-map/road-speeds/summary"),
    ])
      .then(([f, s, c]) => {
        if (!alive) return;
        setFeats(f.features);
        setSum(s);
        setCity(c);
      })
      .catch((e: unknown) => alive && setErr(errorText(e)));
    return () => {
      alive = false;
    };
  }, [hour, dayKind, tick, runSignal]);

  const list = useMemo(() => {
    const all = feats ?? [];
    return (onlySlow ? all.filter((f) => f.properties.slow) : all).map((f, i) => ({ f, i }));
  }, [feats, onlySlow]);

  const cityAvg = useMemo(() => {
    const rows = (city?.buckets ?? []).filter(
      (b) => (hour < 0 || b.hour === hour) && (dayKind === "" || b.weekend === (dayKind === "1")),
    );
    const passes = rows.reduce((a, b) => a + b.passes, 0);
    return passes === 0 ? 0 : rows.reduce((a, b) => a + b.avg_kmh * b.passes, 0) / passes;
  }, [city, hour, dayKind]);

  const layers = useMemo<LayerSpec[]>(() => {
    const feature = (x: { f: SpeedFeature; i: number }) => ({
      type: "Feature" as const,
      geometry: x.f.geometry,
      properties: { id: x.i, kmh: x.f.properties.kmh, name: x.f.properties.name },
    });
    return [
      storesLayer(stores, showStores),
      {
        id: "speeds",
        kind: "line",
        order: 40,
        visible: true,
        dashed: false,
        lineWidth: 4,
        color: speedRamp(),
        data: { type: "FeatureCollection", features: list.filter((x) => x.i !== picked).map(feature) },
      },
      {
        id: "speeds-slow",
        kind: "line",
        order: 45,
        visible: true,
        dashed: true,
        lineWidth: 8,
        color: toneColor("order-stuck"),
        data: {
          type: "FeatureCollection",
          features: list.filter((x) => x.f.properties.slow && x.i !== picked).map(feature),
        },
      },
      {
        id: "speeds-picked",
        kind: "line",
        order: 60,
        visible: true,
        dashed: false,
        lineWidth: 9,
        color: toneColor("order"),
        data: { type: "FeatureCollection", features: list.filter((x) => x.i === picked).map(feature) },
      },
    ];
  }, [list, picked, showStores, stores]);

  const focus = useMemo(() => {
    const x = list.find((y) => y.i === picked);
    const c = x?.f.geometry.coordinates;
    const mid = c?.[Math.floor(c.length / 2)];
    return mid ? { lng: mid[0], lat: mid[1], zoom: 17 } : undefined;
  }, [list, picked]);

  // أبطأُ الطرق أوّلاً ليراها المدير.
  const sorted = useMemo(
    () => [...list].sort((a, b) => a.f.properties.kmh - b.f.properties.kmh).slice(0, 60),
    [list],
  );

  if (err) return <ReloadState label={err} onRetry={() => setTick((x) => x + 1)} />;
  if (!feats || !sum || !city) return <LoadingState />;

  return (
    <>
      <RunInfo sum={sum} />
      <div className="mb-3 grid grid-cols-2 gap-2 sm:grid-cols-4">
        <Tile icon={IconRoute} label={S.tiles.edges} value={fmtNum(city.edges)} />
        <Tile icon={IconRoute} label={S.tiles.slow} value={fmtNum(city.slow_edges)} />
        <Tile icon={IconRoute} label={S.tiles.cityAvg} value={S.kmh.replace("{n}", fmtNum(Math.round(cityAvg * 10) / 10))} />
        <Tile icon={IconRoute} label={S.tiles.passes} value={fmtNum(city.passes)} />
      </div>
      <div className="grid grid-cols-1 gap-3 lg:grid-cols-3">
        <div className="surface relative h-[65vh] overflow-hidden lg:col-span-2">
          <OpsMapCanvas layers={layers} icons={mapIcons()} unavailableLabel={T.mapUnavailable} focus={focus} />
          <div className="absolute bottom-3 start-3 surface px-2 py-1 text-xs">
            <span style={{ color: toneColor("order-stuck") }}>{S.legendSlow}</span>
            {" ← "}
            <span style={{ color: toneColor("driver-available") }}>{S.legendFast}</span>
          </div>
        </div>
        <div className="surface p-3">
          <div className="mb-3 flex flex-wrap items-end gap-2">
            <Select id="sp-hour" label={S.hourLabel} value={String(hour)} onChange={(e) => setHour(Number(e.target.value))}>
              <option value="-1">{S.allHours}</option>
              {Array.from({ length: 24 }, (_, h) => (
                <option key={h} value={h}>
                  {S.hourN.replace("{h}", fmtNum(h))}
                </option>
              ))}
            </Select>
            <Select
              id="sp-day"
              label={S.dayLabel}
              value={dayKind}
              onChange={(e) => setDayKind(e.target.value as "" | "0" | "1")}
            >
              <option value="">{S.allDays}</option>
              <option value="0">{S.weekday}</option>
              <option value="1">{S.weekend}</option>
            </Select>
          </div>
          <div className="mb-3 flex flex-wrap gap-3">
            <Checkbox id="sp-stores" label={T.places.storesLayer} checked={showStores} onChange={(e) => setShowStores(e.target.checked)} />
            <Checkbox id="sp-slow" label={S.onlySlow} checked={onlySlow} onChange={(e) => setOnlySlow(e.target.checked)} />
          </div>
          {sorted.length === 0 ? (
            <AdminEmpty icon={IconRoute} title={S.empty} />
          ) : (
            <ul className="max-h-[50vh] space-y-2 overflow-y-auto">
              {sorted.map(({ f, i }) => (
                <li
                  key={i}
                  className={`rounded-control border px-2 py-2 text-sm ${i === picked ? "border-accent" : "border-line-soft"}`}
                >
                  <Button variant="ghost" size="sm" className="w-full justify-start text-start" onClick={() => setPicked(i)}>
                    <span className="font-bold">{f.properties.name || O.unnamed}</span>
                    <span className="ms-2 text-ink-muted">
                      {S.rowStats.replace("{kmh}", fmtNum(f.properties.kmh)).replace("{passes}", fmtNum(f.properties.passes))}
                    </span>
                  </Button>
                  {f.properties.slow && (
                    <Badge variant="warning" className="mt-1">
                      {S.slowBadge}
                    </Badge>
                  )}
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
    </>
  );
}
