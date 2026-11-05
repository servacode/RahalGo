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
import { Box, distText } from "@/components/admin/orders/CardSections";
import { ShiftsSection, type MonitorShift } from "@/components/admin/drivermonitor/ShiftsSection";
import { useCanCall } from "@/lib/policy";
import { ContactLinks, minutesSince, useNow } from "@/components/admin/orders/BoardWidgets";
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
  /** **القيادة الفعلية** (قرار المالك ٢٠٢٦-١١-٠٥) — منها التنبيه والترتيب والأرقام كلّها. */
  drive_km_with_order: number;
  drive_km_to_pickup: number;
  drive_km_no_order: number;
  drive_km_total: number;
  /** حركة محسوبة وليست قيادة — رجفة GPS أو زحف. */
  not_driving_km: number;
}
interface Row {
  driver_id: string;
  name: string;
  phone: string;
  summary: Summary;
  flagged_gaps: number;
  alert: boolean;
  score: number;
  /** الحالة الحيّة — null لأيام ماضية. */
  live: Live | null;
  /** **ملخّصُ العدّاد لليوم** (قرارُ المالك ٢٠٢٦-١١-٠٥) — العدّاد مقابل التطبيق. */
  odometer?: OdoDay;
}
interface OdoDay {
  on_km: number | null;
  off_km: number | null;
  app_km: number | null;
  unseen_km: number | null;
  missing_close: number;
  big_gap: boolean;
  off_moved: boolean;
  shifts: number;
}
interface Live {
  state: "moving" | "stopped" | "offline";
  since: string | null;
  last_seen: string | null;
  on_shift: boolean;
  has_order: boolean;
}
interface ListResp {
  day: string;
  /** **يومُ الحساب الآن** (من ساعة فتح الدوام) — من الخادم لا من ساعة الجهاز. */
  today: string;
  day_start_hour: number;
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
  today: string;
  alert: boolean;
  driver: { id: string; name: string; phone: string };
  summary: Summary;
  path: Array<{ class: Cls; coords: Array<[number, number]> }>;
  events: Ev[];
  stores: Array<{ id: string; name: string; lat: number; lng: number }>;
  /** **طلباتُ اليوم بمسافة كلٍّ منها** — للمتجر وللزبون من القيادة الفعليّة. */
  orders: DayOrder[];
  /** **دواماتُ اليوم** — مين سكّرها وليش، والعدّاد مقابل التطبيق. */
  shifts: MonitorShift[];
  /** **ملخّصُ العدّاد لليوم.** */
  odometer?: OdoDay;
}
interface DayOrder {
  id: string;
  number: number;
  store: string;
  assigned_at: string | null;
  picked_up_at: string | null;
  end_at: string | null;
  end_kind: string;
  to_pickup_m: number;
  with_order_m: number;
  shared: boolean;
  gap_min: number;
}

/** **الطلبات والمسافة** (قرارُ المالك ٢٠٢٦-١١-٠٥) — «كم طلب، وشقد مشى بكل طلب». */
function OrdersDistance({ orders }: { orders: DayOrder[] }) {
  const O = T.ordersDist;
  const ends = O.end as Record<string, string>;
  return (
    <div className="surface mt-3 p-3">
      <h2 className="mb-2 font-bold">{O.title}</h2>
      {orders.length === 0 ? (
        <p className="text-sm text-ink-muted">{O.empty}</p>
      ) : (
        <ul className="space-y-2 text-sm">
          {orders.map((o, i) => (
            <li key={`${o.id}-${i}`} className="rounded-control border border-line-soft px-2 py-1.5">
              <span className="flex flex-wrap items-center justify-between gap-x-2 gap-y-1">
                <span className="font-bold">{O.row.replace("{n}", fmtNum(o.number)).replace("{store}", o.store || "—")}</span>
                <span className="text-xs text-ink-muted">
                  {o.assigned_at ? fmtDamascusTime(o.assigned_at) : "—"}
                  {" ← "}
                  {o.end_at ? fmtDamascusTime(o.end_at) : "—"}
                </span>
              </span>
              <span className="mt-1 flex flex-wrap items-center gap-x-3 gap-y-1">
                <span>{O.toStore.replace("{d}", distText(o.to_pickup_m))}</span>
                <span>{O.toCustomer.replace("{d}", distText(o.with_order_m))}</span>
                <span className="font-bold">{O.total.replace("{d}", distText(o.to_pickup_m + o.with_order_m))}</span>
                <Badge variant={o.end_at ? "neutral" : "accent"}>{ends[o.end_at ? o.end_kind : "open"] ?? o.end_kind}</Badge>
                {o.shared && <Badge variant="neutral">{O.shared}</Badge>}
                {o.gap_min > 0 && <Badge variant="warning">{O.gap.replace("{n}", fmtNum(o.gap_min))}</Badge>}
              </span>
            </li>
          ))}
        </ul>
      )}
    </div>
  );
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

/** **اليومُ في الرابط إن اختاره المالك** — وفارغُه يومُ الحساب الآن كما يقوله الخادم. */
const dayQuery = (day: string | null) => (day ? `?day=${day}` : "");

function DayView({ driverID, day, onBack }: { driverID: string; day: string | null; onBack: () => void }) {
  const [data, setData] = useState<DayResp | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const [tick, setTick] = useState(0);
  const canCall = useCanCall();

  useEffect(() => {
    let alive = true;
    setData(null);
    setErr(null);
    api<DayResp>(`/api/v1/admin/driver-monitor/${driverID}${dayQuery(day)}`)
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
        flags.push(pt(e.lng, e.lat, { name: data.driver.name + " — " + mins(e.minutes ?? 0), icon: "driver-idle" }));
      } else if (e.kind === "stop_at_store" && e.place) {
        flags.push(pt(e.lng, e.lat, { name: data.driver.name + " — " + e.place + " — " + mins(e.minutes ?? 0), icon: "driver-idle" }));
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
      {/* **الكيلومترات قيادةٌ فعليّة** (قرار المالك ٢٠٢٦-١١-٠٥) — والرجفة والزحف بخانةٍ لحالها. */}
      <div className="mb-1 grid grid-cols-2 gap-2 sm:grid-cols-4 xl:grid-cols-7">
        <Tile icon={IconRoute} label={T.tiles.kmNoOrder} value={km(s.drive_km_no_order)} tone={data.alert ? "danger" : undefined} />
        <Tile icon={IconRoute} label={T.tiles.kmWithOrder} value={km(s.drive_km_with_order)} />
        <Tile icon={IconRoute} label={T.tiles.kmToPickup} value={km(s.drive_km_to_pickup)} />
        <Tile icon={IconRoute} label={T.tiles.notDriving} value={km(s.not_driving_km)} />
        <Tile icon={IconUser} label={T.tiles.stoppedMinutes} value={mins(s.stopped_minutes)} />
        <Tile icon={IconWarning} label={T.tiles.gapMinutes} value={mins(s.gap_minutes)} tone={s.gaps > 0 ? "warning" : undefined} />
        <Tile icon={IconWarning} label={T.tiles.mocked} value={fmtNum(s.mocked)} tone={s.mocked > 0 ? "danger" : undefined} />
      </div>
      <p className="mb-3 text-xs text-ink-muted">{T.driveHint}</p>
      <div className="mb-3">
        <OdoLine o={data.odometer} />
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
      <ShiftsSection
        shifts={data.shifts ?? []}
        canCorrect={canCall("PATCH", "/driver-odometer/{id}")}
        onChanged={() => setTick((x) => x + 1)}
      />
      <OrdersDistance orders={data.orders ?? []} />
    </>
  );
}

const LIVE_ORDER = { moving: 0, stopped: 1, offline: 2 } as const;

/** يمشي أولاً ثم متوقف ثم غير متصل، والتنبيه يعلو داخل كل فئة، ثم الأخطر. */
function sortRows(rows: Row[]): Row[] {
  const rank = (r: Row) => (r.live ? LIVE_ORDER[r.live.state] : 0);
  return [...rows].sort((a, b) => rank(a) - rank(b) || Number(b.alert) - Number(a.alert) || b.score - a.score);
}

function LiveBadge({ live, now }: { live: Live; now: number }) {
  const L = T.live;
  if (live.state === "moving") return <Badge variant="success" className="!px-3 !py-1 !text-base !font-bold">{L.moving}</Badge>;
  if (live.state === "stopped") {
    const n = minutesSince(live.since, now);
    return (
      <Badge variant="warning" className="!px-3 !py-1 !text-base !font-bold">
        {n ? L.stopped.replace("{n}", fmtNum(n)) : L.stoppedNow}
      </Badge>
    );
  }
  return (
    <Badge variant="neutral" className="!px-3 !py-1 !text-base !font-bold">
      {live.last_seen ? L.offline.replace("{time}", fmtDamascusTime(live.last_seen)) : L.offlineNever}
    </Badge>
  );
}

function Stat({ label, value, tone }: { label: string; value: React.ReactNode; tone?: boolean }) {
  return (
    <div className="flex items-center justify-between gap-2">
      <span className="text-ink-muted">{label}</span>
      <span className={`font-bold ${tone ? "text-danger" : ""}`}>{value}</span>
    </div>
  );
}

function DriverCard({ r, now, onOpen }: { r: Row; now: number; onOpen: () => void }) {
  const L = T.live;
  const s = r.summary;
  return (
    <div
      role="button"
      tabIndex={0}
      onClick={onOpen}
      onKeyDown={(e) => (e.key === "Enter" || e.key === " ") && onOpen()}
      className={`surface cursor-pointer space-y-3 p-3 hover:bg-surface-soft ${r.alert ? "border-danger-edge" : ""}`}
    >
      <div className="flex flex-wrap items-center justify-between gap-x-2 gap-y-1">
        <span className="min-w-0 flex-1 basis-28 truncate font-bold">
          {r.name}
          {r.alert && (
            <Badge variant="danger" className="ms-2">
              {T.overLimit}
            </Badge>
          )}
        </span>
        <span className="ms-auto flex shrink-0 items-center gap-1.5">
          <span dir="ltr" className="text-xs text-ink-muted">
            {r.phone}
          </span>
          <ContactLinks phone={r.phone} callLabel={L.call} />
        </span>
      </div>
      {r.live && (
        <div className="flex flex-wrap items-center gap-2">
          <LiveBadge live={r.live} now={now} />
          <Badge variant={r.live.has_order ? "info" : "neutral"}>{r.live.has_order ? L.hasOrder : L.noOrder}</Badge>
          <Badge variant={r.live.on_shift ? "success" : "neutral"}>{r.live.on_shift ? L.onShift : L.offShift}</Badge>
        </div>
      )}
      <OdoLine o={r.odometer} />
      <Box look="plain" icon={<IconRoute size={14} />} title={T.title}>
        <div className="space-y-1 text-sm">
          <Stat label={T.cols.kmNoOrder} value={km(s.drive_km_no_order)} tone={r.alert} />
          <Stat label={T.cols.kmWithOrder} value={km(s.drive_km_with_order + s.drive_km_to_pickup)} />
          <Stat label={T.cols.patterns} value={s.patterns > 0 ? fmtNum(s.patterns) : "—"} tone={s.patterns > 0} />
          <Stat label={T.cols.stops} value={s.store_stops > 0 ? fmtNum(s.store_stops) : "—"} />
          <Stat label={T.cols.gaps} value={s.gaps > 0 ? fmtNum(s.gaps) : "—"} tone={r.flagged_gaps > 0} />
          <Stat label={T.cols.mocked} value={s.mocked > 0 ? fmtNum(s.mocked) : "—"} tone={s.mocked > 0} />
        </div>
      </Box>
    </div>
  );
}

/** **العدّاد مقابل التطبيق على كرت السائق** — والأحمرُ لفرقٍ كبير أو سكّرٍ بلا صورة أو مشيٍ خارج الدوام. */
function OdoLine({ o }: { o?: OdoDay }) {
  if (!o || o.shifts === 0) return null;
  const OL = T.odoList;
  return (
    <div className="flex flex-wrap items-center gap-2 text-sm">
      {o.on_km != null && o.app_km != null && (
        <span className="font-bold">
          {OL.line.replace("{odo}", km(o.on_km)).replace("{app}", km(o.app_km))}
        </span>
      )}
      {o.big_gap && <Badge variant="danger">{OL.bigGap}</Badge>}
      {o.missing_close > 0 && <Badge variant="danger">{OL.noClose}</Badge>}
      {o.off_km != null && o.off_km > 0 && (
        <Badge variant={o.off_moved ? "danger" : "neutral"}>{OL.off.replace("{n}", km(o.off_km))}</Badge>
      )}
    </div>
  );
}

export default function DriverMonitorPage() {
  // **فارغٌ = يومُ الحساب الآن** (من ساعة فتح الدوام لمثلها) — يقوله الخادم، فقبل التاسعة
  // صباحاً يبقى «اليوم» يومَ أمس كما في النقد وعدّاد السائق.
  const [day, setDay] = useState<string | null>(null);
  const [driver, setDriver] = useState<string | null>(null);
  const [list, setList] = useState<ListResp | null>(null);
  const [err, setErr] = useState<string | null>(null);
  const [tick, setTick] = useState(0);
  const now = useNow();
  const isToday = !list || day == null || list.day === list.today;

  // اليوم الحالي: تحديث الحالة الحيّة كل ٣٠ ثانية.
  useEffect(() => {
    if (driver || !isToday) return;
    const id = setInterval(() => setTick((x) => x + 1), 30_000);
    return () => clearInterval(id);
  }, [driver, isToday]);

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
    setErr(null);
    api<ListResp>(`/api/v1/admin/driver-monitor${dayQuery(day)}`)
      .then((r) => alive && setList(r))
      .catch((e: unknown) => alive && setErr(errorText(e)));
    return () => {
      alive = false;
    };
  }, [day, driver, tick]);

  // تبديل اليوم يفرّغ القائمة (التحديث الدوري لا يفرّغها).
  useEffect(() => setList(null), [day]);

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
                value={day ?? list?.day ?? damascusDay()}
                max={list?.today ?? damascusDay()}
                onChange={(e) => e.target.value && setDay(e.target.value)}
              />
            }
          />
          {list && (
            <p className="mb-3 text-xs text-ink-muted">
              {T.dayStarts.replace(/\{h\}/g, fmtNum(list.day_start_hour))}
            </p>
          )}
          {err ? (
            <ReloadState label={err} onRetry={() => setTick((x) => x + 1)} />
          ) : !list ? (
            <LoadingState />
          ) : list.drivers.length === 0 ? (
            <AdminEmpty icon={IconRoute} title={T.empty} />
          ) : (
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2 xl:grid-cols-3">
              {sortRows(list.drivers).map((r) => (
                <DriverCard key={r.driver_id} r={r} now={now} onOpen={() => setDriver(r.driver_id)} />
              ))}
            </div>
          )}
        </>
      )}
    </PageContainer>
  );
}
