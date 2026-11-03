"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **مراقبة السائقين** (قرارُ المالك ٢٠٢٦-١١-٠٣)
 * ══════════════════════════════════════════════════════════════════════
 *
 * «بدنا نعرف كل سائق كم كيلو مشي باليوم، بطلب وبلا طلب، وهل بيتحرك بلا طلب.»
 *
 * **جدولٌ لسائقي اليوم، الأخطرُ أولاً** — وبالنقر يُفتح يومُ السائق: خريطةٌ بمسارٍ ملوَّن
 * (أخضر بطلب · أزرق للاستلام · برتقاليّ بلا طلب · رماديّ وقوف) وقصةُ اليوم بأسطر قصيرة
 * **بأسماء السائق والمتجر**. والتحليلُ كلُّه في الخادم (`internal/drivermonitor`) — هنا عرضٌ فقط.
 *
 * **وهي إشارةٌ لا دليل**: السائقُ قد يقف عند مطعمٍ ليأكل. الحكمُ للمالك بعد أن يسأله.
 */

import { useEffect, useMemo, useState } from "react";
import dynamic from "next/dynamic";
import Link from "next/link";
import {
  getMessages,
  defaultLocale,
  fmtNum,
  fmtDamascusTime,
  damascusDay,
  errorText,
} from "@rahalgo/i18n";
import {
  PageContainer,
  Button,
  Input,
  Alert,
  Badge,
  LoadingState,
  ReloadState,
  AdminEmpty,
  IconWarning,
  IconRoute,
  IconUser,
} from "@rahalgo/ui";
import { PageHero } from "@/components/admin/PageHero";
import { Tile } from "@/components/admin/DashParts";
import { api } from "@/lib/api";
import type { FeatureCollection, LayerSpec } from "@/components/admin/opsmap/canvas";
import { mapIcons, toneColor } from "@/components/admin/opsmap/palette";

const OpsMapCanvas = dynamic(
  () => import("@/components/admin/opsmap/canvas").then((x) => x.OpsMapCanvas),
  { ssr: false },
);

const m = getMessages(defaultLocale);
const T = m.admin.driverMonitor;

interface Summary {
  km_with_order: number;
  km_to_pickup: number;
  km_no_order: number;
  km_total: number;
  orders: number;
  patterns: number;
  store_stops: number;
  gaps: number;
  gap_minutes: number;
  mocked: number;
  points: number;
  stopped_minutes: number;
}
interface Row {
  driver_id: string;
  name: string;
  phone: string;
  summary: Summary;
  flagged_gaps: number;
  alert: boolean;
  score: number;
}
interface ListResp {
  day: string;
  km_alert: number;
  drivers: Row[];
}
interface Ev {
  kind: "stop_at_store" | "pattern" | "gap" | "mocked";
  at: string;
  end_at?: string;
  minutes?: number;
  lat: number;
  lng: number;
  place?: string;
  to_lat?: number;
  to_lng?: number;
  to_at?: string;
  to_minutes?: number;
  to_place?: string;
  distance_m?: number;
  returned?: boolean;
  flagged?: boolean;
  with_order?: boolean;
  count?: number;
}
type Cls = "with_order" | "to_pickup" | "no_order" | "stopped" | "gap";
interface DayResp {
  day: string;
  alert: boolean;
  driver: { id: string; name: string; phone: string };
  summary: Summary;
  path: Array<{ class: Cls; coords: Array<[number, number]> }>;
  events: Ev[];
  stores: Array<{ id: string; name: string; lat: number; lng: number }>;
}

const km = (n: number) => T.km.replace("{n}", fmtNum(Math.round(n * 10) / 10));
const mins = (n: number) => T.min.replace("{n}", fmtNum(n));

/** لون كل صنف من توكنز الخريطة (ولا لون مكتوبٌ باليد). */
function classColor(c: Cls): string {
  switch (c) {
    case "with_order":
      return toneColor("driver-available");
    case "to_pickup":
      return toneColor("order");
    case "no_order":
      return toneColor("driver-busy");
    default:
      return toneColor("driver-idle");
  }
}

/** سطر القصة لحدثٍ واحد — بأسماء السائق والمتجر. */
function line(e: Ev, driver: string): string {
  const E = T.ev;
  const time = fmtDamascusTime(e.at);
  const store = e.place ?? "";
  if (e.kind === "stop_at_store") {
    return E.stop_at_store
      .replace("{driver}", driver)
      .replace("{store}", store)
      .replace("{min}", fmtNum(e.minutes ?? 0))
      .replace("{time}", time);
  }
  if (e.kind === "pattern") {
    return E.pattern
      .replace("{driver}", driver)
      .replace("{store}", store)
      .replace("{min}", fmtNum(e.minutes ?? 0))
      .replace("{time}", time)
      .replace("{dist}", fmtNum(e.distance_m ?? 0))
      .replace("{toMin}", fmtNum(e.to_minutes ?? 0))
      .replace("{toPlace}", e.to_place ? E.patternTo.replace("{store}", e.to_place) : "")
      .replace("{toTime}", e.to_at ? fmtDamascusTime(e.to_at) : "")
      .replace("{back}", e.returned ? E.patternBack : "");
  }
  if (e.kind === "gap") {
    const why = e.with_order ? E.whyOrder : e.flagged ? E.whyMoving : E.whyFar;
    return (e.flagged ? E.gapFlagged : E.gap)
      .replace("{driver}", driver)
      .replace("{min}", fmtNum(e.minutes ?? 0))
      .replace("{time}", time)
      .replace("{place}", store ? E.gapAt.replace("{store}", store) : "")
      .replace("{why}", why);
  }
  return E.mocked.replace("{driver}", driver).replace("{n}", fmtNum(e.count ?? 0)).replace("{time}", time);
}

function fc(features: FeatureCollection["features"]): FeatureCollection {
  return { type: "FeatureCollection", features };
}
function pt(lng: number, lat: number, props: Record<string, unknown>) {
  return { type: "Feature" as const, geometry: { type: "Point", coordinates: [lng, lat] }, properties: props };
}

function DayView({ driverID, day, onBack }: { driverID: string; day: string; onBack: () => void }) {
  const [data, setData] = useState<DayResp | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const [tick, setTick] = useState(0);

  useEffect(() => {
    let alive = true;
    setData(null);
    setErr(null);
    api<DayResp>(`/api/v1/admin/driver-monitor/${driverID}?day=${day}`)
      .then((r) => alive && setData(r))
      .catch((e: unknown) => alive && setErr(errorText(e)));
    return () => {
      alive = false;
    };
  }, [driverID, day, tick]);

  const layers = useMemo<LayerSpec[]>(() => {
    if (!data) return [];
    const out: LayerSpec[] = [];
    const classes: Cls[] = ["stopped", "no_order", "to_pickup", "with_order"];
    classes.forEach((c, i) => {
      out.push({
        id: `trail-${c}`,
        kind: "line",
        order: 30 + i,
        visible: true,
        dashed: false,
        lineWidth: 4,
        color: classColor(c),
        data: fc(
          data.path
            .filter((p) => p.class === c)
            .map((p) => ({
              type: "Feature" as const,
              geometry: { type: "LineString", coordinates: p.coords.map(([la, ln]) => [ln, la]) },
              properties: {},
            })),
        ),
      });
    });
    out.push({
      id: "stores",
      kind: "symbol",
      order: 60,
      visible: true,
      color: toneColor("store-open"),
      labelField: "name",
      data: fc(data.stores.map((s) => pt(s.lng, s.lat, { id: s.id, name: s.name, icon: "store-open" }))),
    });
    const flags: FeatureCollection["features"] = [];
    for (const e of data.events) {
      if (e.kind === "pattern") {
        flags.push(pt(e.lng, e.lat, { name: data.driver.name + " — " + (e.place ?? ""), icon: "order-stuck" }));
        if (e.to_lat != null && e.to_lng != null)
          flags.push(pt(e.to_lng, e.to_lat, { name: e.to_place ?? "", icon: "order-stuck" }));
      } else if (e.kind === "gap" && e.flagged) {
        flags.push(pt(e.lng, e.lat, { name: mins(e.minutes ?? 0), icon: "driver-idle" }));
      }
    }
    out.push({
      id: "flags",
      kind: "symbol",
      order: 70,
      visible: true,
      color: toneColor("order-stuck"),
      labelField: "name",
      labelOverlap: true,
      data: fc(flags),
    });
    const last = data.path.length ? data.path[data.path.length - 1] : null;
    const end = last ? last.coords[last.coords.length - 1] : null;
    out.push({
      id: "driver",
      kind: "symbol",
      order: 80,
      visible: true,
      color: toneColor("driver-available"),
      labelField: "name",
      labelOverlap: true,
      data: fc(end ? [pt(end[1], end[0], { id: data.driver.id, name: data.driver.name, icon: "driver-available" })] : []),
    });
    return out;
  }, [data]);

  const bounds = useMemo<[number, number, number, number] | undefined>(() => {
    if (!data || !data.path.length) return undefined;
    let w = 180, s = 90, e = -180, n = -90;
    for (const p of data.path)
      for (const [la, ln] of p.coords) {
        w = Math.min(w, ln);
        e = Math.max(e, ln);
        s = Math.min(s, la);
        n = Math.max(n, la);
      }
    return [w - 0.004, s - 0.004, e + 0.004, n + 0.004];
  }, [data]);

  if (err) return <ReloadState label={err} onRetry={() => setTick((x) => x + 1)} />;
  if (!data) return <LoadingState />;
  const s = data.summary;
  const dn = data.driver.name;
  return (
    <>
      <PageHero
        icon={IconRoute}
        title={dn}
        titleExtra={data.alert ? <Badge variant="danger">{T.overLimit}</Badge> : undefined}
        actions={
          <>
            <Link className="btn-secondary text-sm" href={`/dashboard/users/${data.driver.id}`}>
              {T.profile}
            </Link>
            <Button variant="secondary" size="sm" onClick={onBack}>
              {T.back}
            </Button>
          </>
        }
      />
      <div className="mb-3 grid grid-cols-2 gap-2 sm:grid-cols-3 xl:grid-cols-6">
        <Tile icon={IconRoute} label={T.tiles.kmNoOrder} value={km(s.km_no_order)} tone={data.alert ? "danger" : undefined} />
        <Tile icon={IconRoute} label={T.tiles.kmWithOrder} value={km(s.km_with_order)} />
        <Tile icon={IconRoute} label={T.tiles.kmToPickup} value={km(s.km_to_pickup)} />
        <Tile icon={IconUser} label={T.tiles.stoppedMinutes} value={mins(s.stopped_minutes)} />
        <Tile icon={IconWarning} label={T.tiles.gapMinutes} value={mins(s.gap_minutes)} tone={s.gaps > 0 ? "warning" : undefined} />
        <Tile icon={IconWarning} label={T.tiles.mocked} value={fmtNum(s.mocked)} tone={s.mocked > 0 ? "danger" : undefined} />
      </div>
      {data.path.length === 0 ? (
        <AdminEmpty icon={IconRoute} title={T.emptyDriver} />
      ) : (
        <div className="grid grid-cols-1 gap-3 lg:grid-cols-3">
          <div className="surface relative h-[60vh] overflow-hidden lg:col-span-2">
            <OpsMapCanvas
              layers={layers}
              icons={mapIcons()}
              unavailableLabel={T.mapUnavailable}
              bounds={bounds}
              focus={
                data.path[0]?.coords[0]
                  ? { lng: data.path[0].coords[0][1], lat: data.path[0].coords[0][0], zoom: 14 }
                  : undefined
              }
            />
          </div>
          <div className="surface p-3">
            <ul className="mb-3 flex flex-wrap gap-x-3 gap-y-1 text-sm">
              {(["with_order", "to_pickup", "no_order", "stopped"] as Cls[]).map((c) => (
                <li key={c} className="flex items-center gap-1.5">
                  <span className="inline-block h-1.5 w-5 rounded-full" style={{ background: classColor(c) }} />
                  {T.legend[c]}
                </li>
              ))}
            </ul>
            <h2 className="mb-2 font-bold">{T.story}</h2>
            {data.events.length === 0 ? (
              <p className="text-sm text-ink-muted">{T.noStory}</p>
            ) : (
              <ol className="space-y-2 text-sm">
                {data.events.map((e, i) => (
                  <li
                    key={i}
                    className={`rounded-control border px-2 py-1.5 ${
                      e.kind === "pattern" || e.kind === "mocked" || (e.kind === "gap" && e.flagged)
                        ? "border-danger-edge bg-danger-tint"
                        : "border-line-soft"
                    }`}
                  >
                    {line(e, dn)}
                  </li>
                ))}
              </ol>
            )}
          </div>
        </div>
      )}
    </>
  );
}

export default function DriverMonitorPage() {
  const [day, setDay] = useState(() => damascusDay());
  const [driver, setDriver] = useState<string | null>(null);
  const [list, setList] = useState<ListResp | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const [tick, setTick] = useState(0);

  useEffect(() => {
    const sp = new URLSearchParams(window.location.search);
    const d = sp.get("day");
    if (d && /^\d{4}-\d{2}-\d{2}$/.test(d)) setDay(d);
    const id = sp.get("driver");
    if (id) setDriver(id);
  }, []);

  useEffect(() => {
    if (driver) return;
    let alive = true;
    setList(null);
    setErr(null);
    api<ListResp>(`/api/v1/admin/driver-monitor?day=${day}`)
      .then((r) => alive && setList(r))
      .catch((e: unknown) => alive && setErr(errorText(e)));
    return () => {
      alive = false;
    };
  }, [day, driver, tick]);

  return (
    <PageContainer>
      {driver ? (
        <DayView driverID={driver} day={day} onBack={() => setDriver(null)} />
      ) : (
        <>
          <PageHero
            icon={IconRoute}
            title={T.title}
            actions={
              <Input
                id="dm-day"
                type="date"
                label={T.day}
                value={day}
                max={damascusDay()}
                onChange={(e) => e.target.value && setDay(e.target.value)}
              />
            }
          />
          {err ? (
            <ReloadState label={err} onRetry={() => setTick((x) => x + 1)} />
          ) : !list ? (
            <LoadingState />
          ) : list.drivers.length === 0 ? (
            <AdminEmpty icon={IconRoute} title={T.empty} />
          ) : (
            <div className="surface overflow-x-auto">
              <table className="w-full text-sm">
                <thead>
                  <tr className="border-b border-line-soft text-start text-ink-muted">
                    <th className="p-3 text-start font-bold">{T.driver}</th>
                    <th className="p-3 text-start font-bold">{T.cols.kmNoOrder}</th>
                    <th className="p-3 text-start font-bold">{T.cols.kmWithOrder}</th>
                    <th className="p-3 text-start font-bold">{T.cols.patterns}</th>
                    <th className="p-3 text-start font-bold">{T.cols.stops}</th>
                    <th className="p-3 text-start font-bold">{T.cols.gaps}</th>
                    <th className="p-3 text-start font-bold">{T.cols.mocked}</th>
                  </tr>
                </thead>
                <tbody>
                  {list.drivers.map((r) => (
                    <tr
                      key={r.driver_id}
                      className="cursor-pointer border-b border-line-soft last:border-0 hover:bg-surface-soft"
                      onClick={() => setDriver(r.driver_id)}
                    >
                      <td className="p-3 font-bold">
                        {r.name}
                        {r.alert && (
                          <Badge variant="danger" className="ms-2">
                            {T.overLimit}
                          </Badge>
                        )}
                      </td>
                      <td className="p-3">{km(r.summary.km_no_order)}</td>
                      <td className="p-3">{km(r.summary.km_with_order + r.summary.km_to_pickup)}</td>
                      <td className="p-3">{r.summary.patterns > 0 ? <Badge variant="danger">{fmtNum(r.summary.patterns)}</Badge> : "—"}</td>
                      <td className="p-3">{r.summary.store_stops > 0 ? fmtNum(r.summary.store_stops) : "—"}</td>
                      <td className="p-3">
                        {r.summary.gaps > 0 ? (
                          <Badge variant={r.flagged_gaps > 0 ? "warning" : "neutral"}>{fmtNum(r.summary.gaps)}</Badge>
                        ) : (
                          "—"
                        )}
                      </td>
                      <td className="p-3">{r.summary.mocked > 0 ? <Badge variant="danger">{fmtNum(r.summary.mocked)}</Badge> : "—"}</td>
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
          )}
        </>
      )}
    </PageContainer>
  );
}
