"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **أماكنُ وأسامي** (قرارُ المالك ٢٠٢٦-١١-٠٣ — الدفعة الثانية)
 * ══════════════════════════════════════════════════════════════════════
 *
 * مرشَّحو معالمَ من مصدرين: وقفاتُ السائقين المتكرّرة عند مكانٍ بلا معلم، وأسماءٌ ذكرها زبونان فأكثر
 * في عناوينهم عند النقطة نفسِها. **المدير يسمّي ويوافق** فيدخل المعلمُ «قرب المعلم»، أو يرفض.
 * **وأسماءُ المتاجر ظاهرةٌ على الخريطة** (طلبُ المالك) — فيعرف من أيّ متجرٍ قريبٍ هذا المكان.
 * والكشفُ كلُّه في الخادم (`internal/maplearn/places*.go`) — هنا عرضٌ وقرار.
 */

import { useCallback, useEffect, useMemo, useState } from "react";
import dynamic from "next/dynamic";
import { getMessages, defaultLocale, fmtNum, errorText } from "@rahalgo/i18n";
import {
  Button,
  Checkbox,
  Input,
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
const P = T.places;

type Status = "pending" | "approved" | "rejected";
type Source = "" | "stops" | "addresses";
const KINDS = ["mosque", "pharmacy", "school", "hospital", "shop", "roundabout", "other"] as const;
type Kind = (typeof KINDS)[number];

interface Place {
  id: number;
  source: "stops" | "addresses";
  status: Status;
  name: string;
  kind: string;
  lat: number;
  lng: number;
  visits: number;
  drivers: number;
  customers: number;
  days: number;
  evidence: number;
  samples: string[];
  first_seen: string | null;
  last_seen: string | null;
}
interface Store {
  id: string;
  name: string;
  lat: number;
  lng: number;
}
interface Resp {
  candidates: Place[];
  counts: Record<Status, number>;
  stores: Store[];
  running: boolean;
  last_run?: { at: string; stop_places: number; name_places: number; failed: boolean };
}

const pt = (lng: number, lat: number, properties: Record<string, unknown>) => ({
  type: "Feature" as const,
  geometry: { type: "Point", coordinates: [lng, lat] },
  properties,
});

function asKind(k: string): Kind {
  return (KINDS as readonly string[]).includes(k) ? (k as Kind) : "other";
}

function PlaceRow({
  c,
  active,
  canDecide,
  busy,
  onPick,
  onDecide,
}: {
  c: Place;
  active: boolean;
  canDecide: boolean;
  busy: boolean;
  onPick: () => void;
  onDecide: (approve: boolean, name: string, kind: Kind) => void;
}) {
  const [name, setName] = useState(c.name);
  const [kind, setKind] = useState<Kind>(asKind(c.kind));
  const stats =
    c.source === "stops"
      ? P.stopStats
          .replace("{visits}", fmtNum(c.visits))
          .replace("{drivers}", fmtNum(c.drivers))
          .replace("{days}", fmtNum(c.days))
      : P.addrStats.replace("{customers}", fmtNum(c.customers)).replace("{evidence}", fmtNum(c.evidence));
  return (
    <li className={`rounded-control border px-2 py-2 text-sm ${active ? "border-accent" : "border-line-soft"}`}>
      <Button variant="ghost" size="sm" className="w-full justify-start text-start" onClick={onPick}>
        <span className="font-bold">{c.name || P.unnamed}</span>
        <span className="ms-2 text-ink-muted">{stats}</span>
      </Button>
      <div className="mt-1 flex flex-wrap items-center gap-2 px-2 text-xs text-ink-muted">
        <Badge variant="neutral">{P.sourceBadge[c.source]}</Badge>
        {c.first_seen && c.last_seen && (
          <span>{P.seen.replace("{first}", c.first_seen).replace("{last}", c.last_seen)}</span>
        )}
      </div>
      {c.samples.length > 0 && (
        <div className="mt-1 px-2 text-xs">
          <div className="text-ink-muted">{P.samples}</div>
          <ul className="list-disc ps-4">
            {c.samples.map((s) => (
              <li key={s}>{s}</li>
            ))}
          </ul>
        </div>
      )}
      {c.status === "pending" && canDecide && (
        <div className="mt-2 flex flex-wrap items-end gap-2">
          <Input
            id={`place-name-${c.id}`}
            label={P.nameLabel}
            value={name}
            maxLength={80}
            onChange={(e) => setName(e.target.value)}
          />
          <Select id={`place-kind-${c.id}`} label={P.kindLabel} value={kind} onChange={(e) => setKind(asKind(e.target.value))}>
            {KINDS.map((k) => (
              <option key={k} value={k}>
                {P.kinds[k]}
              </option>
            ))}
          </Select>
          <Button size="sm" disabled={busy || name.trim().length < 2} onClick={() => onDecide(true, name, kind)}>
            <IconCheck size={14} /> {T.approve}
          </Button>
          <Button size="sm" variant="secondary" disabled={busy} onClick={() => onDecide(false, name, kind)}>
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

export function PlacesPanel({ canManage, runSignal }: { canManage: boolean; runSignal: number }) {
  const [status, setStatus] = useState<Status>("pending");
  const [source, setSource] = useState<Source>("");
  const [showStores, setShowStores] = useState(true);
  const [data, setData] = useState<Resp | null>(null);
  const [picked, setPicked] = useState<number | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const [actionErr, setActionErr] = useState("");
  const [busy, setBusy] = useState(false);
  const [tick, setTick] = useState(0);

  useEffect(() => {
    let alive = true;
    setErr(null);
    api<Resp>(`/api/v1/admin/ops-map/place-candidates?status=${status}&source=${source}`)
      .then((r) => alive && setData(r))
      .catch((e: unknown) => alive && setErr(errorText(e)));
    return () => {
      alive = false;
    };
  }, [status, source, tick, runSignal]);

  const list = useMemo(() => data?.candidates ?? [], [data]);

  const layers = useMemo<LayerSpec[]>(() => {
    const fc = (features: FeatureCollection["features"]): FeatureCollection => ({ type: "FeatureCollection", features });
    return [
      {
        id: "stores",
        kind: "symbol",
        order: 30,
        visible: showStores,
        color: toneColor("store-open"),
        labelField: "name",
        data: fc((data?.stores ?? []).map((s) => pt(s.lng, s.lat, { id: s.id, name: s.name, icon: "store-open" }))),
      },
      {
        id: "places",
        kind: "symbol",
        order: 50,
        visible: true,
        color: toneColor("order"),
        labelField: "name",
        labelOverlap: true,
        data: fc(
          list
            .filter((c) => c.id !== picked)
            .map((c) =>
              pt(c.lng, c.lat, {
                id: c.id,
                name: c.name || P.unnamed,
                icon: c.status === "approved" ? "driver-available" : "order",
              }),
            ),
        ),
      },
      {
        id: "place-picked",
        kind: "symbol",
        order: 60,
        visible: true,
        color: toneColor("order-stuck"),
        labelField: "name",
        labelOverlap: true,
        data: fc(
          list
            .filter((c) => c.id === picked)
            .map((c) => pt(c.lng, c.lat, { id: c.id, name: c.name || P.unnamed, icon: "order-stuck" })),
        ),
      },
    ];
  }, [data, list, picked, showStores]);

  const focus = useMemo(() => {
    const c = list.find((x) => x.id === picked);
    return c ? { lng: c.lng, lat: c.lat, zoom: 18 } : undefined;
  }, [list, picked]);

  const decide = useCallback(async (id: number, approve: boolean, name: string, kind: Kind) => {
    setBusy(true);
    setActionErr("");
    try {
      await api(`/api/v1/admin/ops-map/place-candidates/${id}/decide`, {
        method: "POST",
        body: JSON.stringify({ approve, name, kind }),
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
  if (!data) return <LoadingState />;

  return (
    <>
      {actionErr && <Alert tone="error">{actionErr}</Alert>}
      {data.last_run && (
        <Alert tone={data.last_run.failed ? "warning" : "info"}>
          {data.last_run.failed
            ? P.lastRunFailed
            : P.lastRun
                .replace("{stops}", fmtNum(data.last_run.stop_places))
                .replace("{names}", fmtNum(data.last_run.name_places))}
        </Alert>
      )}
      <div className="mb-3 grid grid-cols-2 gap-2 sm:grid-cols-3">
        <Tile icon={IconRoute} label={P.tiles.pending} value={fmtNum(data.counts.pending)} />
        <Tile icon={IconCheck} label={P.tiles.approved} value={fmtNum(data.counts.approved)} />
        <Tile icon={IconClose} label={P.tiles.rejected} value={fmtNum(data.counts.rejected)} />
      </div>
      <div className="grid grid-cols-1 gap-3 lg:grid-cols-3">
        <div className="surface relative h-[65vh] overflow-hidden lg:col-span-2">
          <OpsMapCanvas layers={layers} icons={mapIcons()} unavailableLabel={T.mapUnavailable} focus={focus} />
        </div>
        <div className="surface p-3">
          <div className="mb-3 flex flex-wrap items-end gap-2">
            <Checkbox id="pl-stores" label={P.storesLayer} checked={showStores} onChange={(e) => setShowStores(e.target.checked)} />
            {(["", "stops", "addresses"] as Source[]).map((s) => (
              <Button
                key={s || "all"}
                size="sm"
                variant={source === s ? "primary" : "secondary"}
                onClick={() => {
                  setPicked(null);
                  setSource(s);
                }}
              >
                {P.sources[s === "" ? "all" : s]}
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
                {T.tabs[s]} {fmtNum(data.counts[s])}
              </Button>
            ))}
          </div>
          {list.length === 0 ? (
            <AdminEmpty icon={IconRoute} title={P.empty} />
          ) : (
            <ul className="max-h-[50vh] space-y-2 overflow-y-auto">
              {list.map((c) => (
                <PlaceRow
                  key={`${c.id}-${c.status}`}
                  c={c}
                  active={c.id === picked}
                  canDecide={canManage}
                  busy={busy}
                  onPick={() => setPicked(c.id)}
                  onDecide={(approve, name, kind) => void decide(c.id, approve, name, kind)}
                />
              ))}
            </ul>
          )}
        </div>
      </div>
    </>
  );
}
