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
 */

import { useCallback, useEffect, useMemo, useState } from "react";
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
} from "@rahalgo/ui";
import { PageHero } from "@/components/admin/PageHero";
import { Tile } from "@/components/admin/DashParts";
import { api, apiFile } from "@/lib/api";
import { useCanCall } from "@/lib/policy";
import type { FeatureCollection, LayerSpec } from "@/components/admin/opsmap/canvas";
import { toneColor } from "@/components/admin/opsmap/palette";

const OpsMapCanvas = dynamic(
  () => import("@/components/admin/opsmap/canvas").then((x) => x.OpsMapCanvas),
  { ssr: false },
);

const m = getMessages(defaultLocale);
const T = m.admin.mapLearning;

type Status = "pending" | "approved" | "rejected";

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

  const layers = useMemo<LayerSpec[]>(() => {
    const out: LayerSpec[] = [
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
  }, [heat, cands, picked, showHeat, status]);

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
          canManage ? (
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
      {err ? (
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
              <OpsMapCanvas layers={layers} icons={[]} unavailableLabel={T.mapUnavailable} focus={focus} />
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
