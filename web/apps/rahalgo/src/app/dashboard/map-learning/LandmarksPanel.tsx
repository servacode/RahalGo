"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **معالمُ السائقين** (قرارُ المالك ٢٠٢٦-١١-٠٥)
 * ══════════════════════════════════════════════════════════════════════
 *
 * «السائقين عم يقولو عم يبعثو معالم بس انا ماعم يوصلني شي» — كانت تُحفظ بانتظار الموافقة
 * وطبقتُها في خريطة العمليّات مطفأةٌ ومخبّأة. **فصار لها مكانُها هنا**: يومُ المنصّة بمربّعاته،
 * وكلُّ سائقٍ كم أرسل وكم انحسب من المطلوب (ومن اشتغل ولم يرسل شيئاً)، وطابورُ الموافقة.
 * **والعدُّ والتكرارُ كلُّه في الخادم** (`server/landmark_quota.go`) — هنا عرضٌ وقرار.
 */

import { useCallback, useEffect, useMemo, useState } from "react";
import Link from "next/link";
import { getMessages, defaultLocale, fmtNum, fmtDateTime, fmtDamascusTime, errorText } from "@rahalgo/i18n";
import {
  Button,
  Checkbox,
  Input,
  Alert,
  Badge,
  ReloadState,
  DataView,
  ViewToggle,
  useViewMode,
  useLiveData,
  type DataColumn,
  IconRoute,
  IconCheck,
  IconClose,
  IconLocation,
  IconUsers,
  IconTarget,
  IconHourglass,
  IconWarning,
} from "@rahalgo/ui";
import { Tile } from "@/components/admin/DashParts";
import { api } from "@/lib/api";

const T = getMessages(defaultLocale).admin.mapLearning;
const L = T.landmarks;
const KINDS = T.places.kinds as Record<string, string>;

type Status = "pending" | "approved" | "rejected" | "duplicate";
const STATUSES: Status[] = ["pending", "approved", "rejected", "duplicate"];
const PAGE = 100;

interface DriverDay {
  driver_id: string;
  name: string;
  sent: number;
  new: number;
  duplicate: number;
  pending: number;
  approved: number;
  rejected: number;
  worked: boolean;
  completed: boolean;
  reminded_hour_at: string | null;
  reminded_shift_end_at: string | null;
}
interface Daily {
  day: string;
  today: string;
  quota: number;
  totals: {
    sent: number;
    new: number;
    duplicate: number;
    pending: number;
    approved: number;
    rejected: number;
    drivers: number;
    senders: number;
    completed: number;
    incomplete: number;
  };
  drivers: DriverDay[];
}
interface Sugg {
  id: number;
  name: string;
  kind: string;
  lat: number;
  lng: number;
  created_at: string;
  suggested_by_name: string;
  status: Status;
  platform_day: string;
  point_from_ping: boolean;
  verified: boolean;
  dup_source: string | null;
  dup_name: string | null;
  dup_distance_m: number | null;
}
interface Page {
  suggestions: Sugg[];
  total: number;
  has_more: boolean;
}

/** **«على الخريطة»** — خريطةُ العمليّات على المعلم وطبقتُه مشغّلةٌ لهذه الفتحة وحدَها. */
const onMapHref = (s: Sugg) => `/dashboard/opsmap?focus=${s.lat},${s.lng}&layer=landmarks`;

function DriverProgress({ d, quota }: { d: DriverDay; quota: number }) {
  if (quota <= 0) return <span className="text-ink-muted">{L.quotaOff}</span>;
  if (d.completed) return <Badge variant="success">{L.done}</Badge>;
  return (
    <Badge variant={d.new === 0 ? "danger" : "warning"}>
      {L.progress.replace("{n}", fmtNum(d.new)).replace("{quota}", fmtNum(quota))}
    </Badge>
  );
}

function Reminders({ d }: { d: DriverDay }) {
  if (!d.reminded_hour_at && !d.reminded_shift_end_at) return <span className="text-ink-muted">—</span>;
  return (
    <span className="flex flex-col gap-0.5 text-xs">
      {d.reminded_hour_at && <span>{L.remindedHour.replace("{time}", fmtDamascusTime(d.reminded_hour_at))}</span>}
      {d.reminded_shift_end_at && (
        <span>{L.remindedShiftEnd.replace("{time}", fmtDamascusTime(d.reminded_shift_end_at))}</span>
      )}
    </span>
  );
}

function LocationBadge({ s }: { s: Sugg }) {
  if (s.point_from_ping) return <Badge variant="info">{L.fixedFromPing}</Badge>;
  if (!s.verified) return <Badge variant="neutral">{L.unverified}</Badge>;
  return <Badge variant="success">{L.verified}</Badge>;
}

export function LandmarksPanel({ canDecide, initialDay }: { canDecide: boolean; initialDay?: string }) {
  const [day, setDay] = useState(initialDay ?? "");
  const [status, setStatus] = useState<Status>("pending");
  const [dayOnly, setDayOnly] = useState(false);
  const [items, setItems] = useState<Sugg[]>([]);
  const [total, setTotal] = useState(0);
  const [hasMore, setHasMore] = useState(false);
  const [queueErr, setQueueErr] = useState<string | null>(null);
  const [queueLoading, setQueueLoading] = useState(true);
  const [actionErr, setActionErr] = useState("");
  const [busy, setBusy] = useState<number | null>(null);
  const [tick, setTick] = useState(0);
  const [driversView, setDriversView] = useViewMode("map-learning-landmark-drivers");
  const [queueView, setQueueView] = useViewMode("map-learning-landmark-queue", "cards");

  // **يومُ المنصّة من الخادم** — من التاسعة للتاسعة، فلا يُحسب في المتصفّح بيوم التقويم.
  const daily = useLiveData<Daily>(
    () => api<Daily>(`/api/v1/admin/ops-map/landmarks/daily${day ? `?day=${day}` : ""}`),
    ["landmark"],
    [day, tick],
  );
  useEffect(() => {
    if (!day && daily.data?.day) setDay(daily.data.day);
  }, [day, daily.data]);

  const query = useCallback(
    (offset: number) => {
      const p = new URLSearchParams({ status, limit: String(PAGE), offset: String(offset) });
      if (dayOnly && day) p.set("day", day);
      return api<Page>(`/api/v1/admin/ops-map/landmark-suggestions?${p.toString()}`);
    },
    [status, dayOnly, day],
  );

  useEffect(() => {
    let alive = true;
    setQueueErr(null);
    setQueueLoading(true);
    query(0)
      .then((r) => {
        if (!alive) return;
        setItems(r.suggestions);
        setTotal(r.total);
        setHasMore(r.has_more);
      })
      .catch((e: unknown) => alive && setQueueErr(errorText(e)))
      .finally(() => alive && setQueueLoading(false));
    return () => {
      alive = false;
    };
  }, [query, tick]);

  async function more() {
    try {
      const r = await query(items.length);
      setItems((x) => [...x, ...r.suggestions]);
      setTotal(r.total);
      setHasMore(r.has_more);
    } catch (e) {
      setActionErr(errorText(e));
    }
  }

  async function decide(id: number, approve: boolean) {
    setBusy(id);
    setActionErr("");
    try {
      await api(`/api/v1/admin/ops-map/landmark-suggestions/${id}/decide`, {
        method: "POST",
        body: JSON.stringify({ approve }),
      });
      setTick((x) => x + 1);
    } catch (e) {
      setActionErr(errorText(e));
    } finally {
      setBusy(null);
    }
  }

  const quota = daily.data?.quota ?? 0;
  const driverCols = useMemo<DataColumn<DriverDay>[]>(
    () => [
      {
        id: "name",
        header: L.cols.driver,
        primary: true,
        icon: <IconUsers size={14} />,
        cell: (d) => (
          <span className="font-bold">
            {d.name || L.noName}
            {d.worked && d.sent === 0 && (
              <Badge variant="danger" className="ms-2">
                {L.workedNoSend}
              </Badge>
            )}
          </span>
        ),
      },
      { id: "completed", header: L.cols.completed, cell: (d) => <DriverProgress d={d} quota={quota} /> },
      { id: "sent", header: L.cols.sent, cell: (d) => fmtNum(d.sent) },
      { id: "new", header: L.cols.new, cell: (d) => fmtNum(d.new) },
      { id: "duplicate", header: L.cols.duplicate, cell: (d) => fmtNum(d.duplicate) },
      { id: "pending", header: L.cols.pending, cell: (d) => fmtNum(d.pending) },
      { id: "approved", header: L.cols.approved, cell: (d) => fmtNum(d.approved) },
      { id: "rejected", header: L.cols.rejected, cell: (d) => fmtNum(d.rejected) },
      { id: "reminder", header: L.cols.reminder, cell: (d) => <Reminders d={d} /> },
    ],
    [quota],
  );

  const queueCols = useMemo<DataColumn<Sugg>[]>(
    () => [
      {
        id: "name",
        header: L.q.name,
        primary: true,
        icon: <IconLocation size={14} />,
        cell: (s) => (
          <span className="font-bold">
            {s.name}
            <Badge variant="accent" className="ms-2">
              {KINDS[s.kind] ?? s.kind}
            </Badge>
          </span>
        ),
      },
      { id: "driver", header: L.q.driver, cell: (s) => s.suggested_by_name || L.noName },
      { id: "at", header: L.q.at, cell: (s) => fmtDateTime(s.created_at) },
      { id: "location", header: L.q.location, cell: (s) => <LocationBadge s={s} /> },
      {
        id: "dup",
        header: L.q.dup,
        hide: (s) => s.status !== "duplicate",
        cell: (s) =>
          s.dup_name ? (
            <span>
              {L.dupOf.replace("{name}", s.dup_name).replace("{d}", fmtNum(s.dup_distance_m ?? 0))}
              {s.dup_source && (
                <Badge variant="neutral" className="ms-2">
                  {(L.dupSource as Record<string, string>)[s.dup_source] ?? s.dup_source}
                </Badge>
              )}
            </span>
          ) : (
            "—"
          ),
      },
    ],
    [],
  );

  const t = daily.data?.totals;
  return (
    <>
      {actionErr && <Alert tone="error">{actionErr}</Alert>}
      <div className="mb-3 flex flex-wrap items-end gap-2">
        <Input
          id="lm-day"
          type="date"
          label={L.day}
          value={day}
          max={daily.data?.today}
          onChange={(e) => e.target.value && setDay(e.target.value)}
        />
      </div>
      {daily.error && !daily.data ? (
        <ReloadState label={L.loadFailed} onRetry={() => setTick((x) => x + 1)} />
      ) : (
        <>
          <div className="mb-3 grid grid-cols-2 gap-2 sm:grid-cols-4 xl:grid-cols-7">
            <Tile icon={IconRoute} label={L.tiles.sent} value={t ? fmtNum(t.sent) : "—"} />
            <Tile icon={IconCheck} label={L.tiles.new} value={t ? fmtNum(t.new) : "—"} />
            <Tile icon={IconWarning} label={L.tiles.duplicate} value={t ? fmtNum(t.duplicate) : "—"} />
            <Tile icon={IconHourglass} label={L.tiles.pending} value={t ? fmtNum(t.pending) : "—"} />
            <Tile icon={IconCheck} label={L.tiles.approved} value={t ? fmtNum(t.approved) : "—"} />
            <Tile icon={IconClose} label={L.tiles.rejected} value={t ? fmtNum(t.rejected) : "—"} />
            <Tile
              icon={IconTarget}
              label={L.tiles.completed.replace("{quota}", fmtNum(quota))}
              value={
                t && quota > 0
                  ? L.tiles.completedValue.replace("{n}", fmtNum(t.completed)).replace("{of}", fmtNum(t.drivers))
                  : L.quotaOff
              }
              tone={t && quota > 0 && t.incomplete > 0 ? "warning" : undefined}
            />
          </div>
          <div className="mb-2 flex items-center justify-between gap-2">
            <h2 className="heading-section">{L.driversTitle}</h2>
            <ViewToggle view={driversView} onChange={setDriversView} tableLabel={L.viewTable} cardsLabel={L.viewCards} />
          </div>
          <div className="mb-6">
            <DataView
              items={daily.data?.drivers ?? []}
              loading={daily.loading}
              getKey={(d) => d.driver_id}
              columns={driverCols}
              view={driversView}
              empty={L.emptyDrivers}
            />
          </div>
        </>
      )}

      <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
        <h2 className="heading-section">
          {L.queueTitle} <span className="text-sm text-ink-muted">{L.total.replace("{n}", fmtNum(total))}</span>
        </h2>
        <ViewToggle view={queueView} onChange={setQueueView} tableLabel={L.viewTable} cardsLabel={L.viewCards} />
      </div>
      <div className="mb-3 flex flex-wrap items-end gap-2">
        {STATUSES.map((s) => (
          <Button key={s} size="sm" variant={status === s ? "primary" : "secondary"} onClick={() => setStatus(s)}>
            {L.status[s]}
          </Button>
        ))}
        <Checkbox id="lm-day-only" label={L.dayOnly} checked={dayOnly} onChange={(e) => setDayOnly(e.target.checked)} />
      </div>
      {queueErr ? (
        <ReloadState label={queueErr} onRetry={() => setTick((x) => x + 1)} />
      ) : (
        <>
          <DataView
            items={items}
            loading={queueLoading}
            getKey={(s) => String(s.id)}
            columns={queueCols}
            view={queueView}
            empty={L.emptyQueue}
            actions={(s) => (
              <div className="flex flex-wrap gap-2">
                {s.status === "pending" && canDecide && (
                  <>
                    <Button size="sm" disabled={busy === s.id} onClick={() => void decide(s.id, true)}>
                      <IconCheck size={14} /> {T.approve}
                    </Button>
                    <Button size="sm" variant="secondary" disabled={busy === s.id} onClick={() => void decide(s.id, false)}>
                      <IconClose size={14} /> {T.reject}
                    </Button>
                  </>
                )}
                <Link className="btn-secondary text-sm" href={onMapHref(s)}>
                  {L.onMap}
                </Link>
              </div>
            )}
          />
          {hasMore && (
            <div className="mt-3 flex justify-center">
              <Button variant="secondary" size="sm" onClick={() => void more()}>
                {L.loadMore}
              </Button>
            </div>
          )}
        </>
      )}
    </>
  );
}
