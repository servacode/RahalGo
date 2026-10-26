"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **طلباتُ التوسّع — قسمٌ مستقلّ** (قرارُ المالك ٢٠٢٦-١٠-٠٤)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **«مو مخفيّة تحت الخريطة»** — أرقامٌ (أشخاص · جديدُ الأسبوع · مدن)،
 * وجدولٌ محافظة ← مدينة ← منطقة بعدد الأشخاص والطلبات وأوّلِ وآخرِ طلبٍ
 * والحال، **والملغاةُ لا تُحسب.**
 *
 * **والإبلاغُ بزرٍّ لا آليّاً** — «بلّغ المنتظرين الآن» يقول عددَهم قبل
 * الضغط، ويطلب تأكيداً. **والتذكيرُ** في رأس الصفحة حين تكون منطقةٌ
 * مغطّاةٌ ومنتظروها لم يُبلَّغوا.
 *
 * **و«ينتظرون ولم يُبلَّغوا»** عينُ رقم الصفحة الرئيسة
 * (`expansion_waiting`) — يفتحها رابطُ بطاقتها بـ`?filter=waiting`.
 */

import { useMemo, useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtDate, errorText } from "@rahalgo/i18n";
import {
  DataView,
  Badge,
  Button,
  ButtonLink,
  Alert,
  Confirm,
  useLiveData,
  type DataColumn,
  IconLocation,
  IconUsers,
  IconTrendUp,
  IconZones,
  IconHourglass,
  IconSend,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useCanCall } from "@/lib/policy";
import PlaceDemandPanel from "@/components/admin/PlaceDemandPanel";

const m = getMessages(defaultLocale);
const T = m.admin.expansion;

type Status = "uncovered" | "ready" | "covered";

interface Row {
  key: string;
  kind: "coverage_request" | "service_interest";
  governorate_id: string;
  governorate_name: string;
  city_id: string;
  city_name: string;
  zone_id: string;
  zone_name: string;
  label: string;
  people: number;
  requests: number;
  first_at: string;
  last_at: string;
  lat: number;
  lng: number;
  waiting: number;
  notified: number;
  notifiable: number;
  status: Status;
}

interface Summary {
  people: number;
  new_this_week: number;
  cities: number;
  waiting_not_notified: number;
  pending_notify: number;
  ready_places: number;
  rows: Row[];
}

type Filter = "all" | "waiting" | "ready" | "uncovered";

const STATUS_VARIANT: Record<Status, "neutral" | "warning" | "success"> = {
  uncovered: "neutral",
  ready: "warning",
  covered: "success",
};

/** **الفلترُ من الرابط** — بطاقةُ الرئيسة تفتح على «ينتظرون ولم يُبلَّغوا». */
function initialFilter(): Filter {
  if (typeof window === "undefined") return "all";
  const f = new URLSearchParams(window.location.search).get("filter");
  return f === "waiting" || f === "ready" || f === "uncovered" ? f : "all";
}

const n = (tpl: string, v: number, p?: number) =>
  tpl.replace("{n}", fmtNum(v)).replace("{p}", fmtNum(p ?? 0));

export default function ExpansionPage() {
  const canPlaces = useCanCall()("GET", "/ops-map/coverage-demand/places");
  const [filter, setFilter] = useState<Filter>(initialFilter);
  const [asking, setAsking] = useState<Row | null>(null);
  const [busy, setBusy] = useState(false);
  const [msg, setMsg] = useState<{ tone: "success" | "error"; text: string } | null>(null);

  const data = useLiveData<Summary>(() => api<Summary>("/api/v1/admin/ops-map/expansion"), []);
  const meta = useLiveData<{ permissions: string[] }>(
    () => api<{ permissions: string[] }>("/api/v1/admin/ops-map/meta"),
    [],
  );
  const canNotify = !!meta.data?.permissions.includes("MANAGE_COVERAGE");

  const rows = useMemo(() => {
    const all = data.data?.rows ?? [];
    switch (filter) {
      // **وعدُّ هذا الفلتر هو رقمُ الرئيسة** — طلباتُ «أضف منطقتي» غيرُ المُبلَّغة.
      case "waiting":
        return all.filter((r) => r.kind === "coverage_request" && r.waiting > 0);
      case "ready":
        return all.filter((r) => r.status === "ready");
      case "uncovered":
        return all.filter((r) => r.status === "uncovered");
      default:
        return all;
    }
  }, [data.data, filter]);

  const s = data.data;

  const notify = async (r: Row) => {
    setBusy(true);
    try {
      const out = await api<{ notified: number }>("/api/v1/admin/ops-map/expansion/notify", {
        method: "POST",
        body: JSON.stringify({ key: r.key }),
      });
      setMsg({ tone: "success", text: n(T.done, out.notified) });
      data.reload();
    } catch (e) {
      setMsg({ tone: "error", text: errorText(e) });
    } finally {
      setBusy(false);
      setAsking(null);
    }
  };

  const columns: DataColumn<Row>[] = [
    {
      id: "gov",
      header: T.governorate,
      cell: (r) => r.governorate_name || <span className="text-ink-muted">{T.unresolved}</span>,
    },
    {
      id: "city",
      header: T.city,
      cell: (r) => r.city_name || <span className="text-ink-muted">{T.unresolved}</span>,
    },
    {
      id: "zone",
      header: T.zone,
      icon: <IconZones />,
      cell: (r) =>
        r.zone_name ? (
          <span className="font-medium">{r.zone_name}</span>
        ) : (
          <span className="flex flex-col">
            <span className="text-ink-muted">
              {r.kind === "service_interest" && r.key.startsWith("city:") ? T.wholeCity : T.outside}
            </span>
            {r.label && <span className="text-xs text-ink-muted">{r.label}</span>}
          </span>
        ),
    },
    {
      id: "kind",
      header: T.kind,
      cell: (r) => (
        <Badge variant={r.kind === "coverage_request" ? "primary" : "info"}>
          {r.kind === "coverage_request" ? T.kindCoverage : T.kindInterest}
        </Badge>
      ),
    },
    {
      id: "people",
      header: T.peopleCol,
      icon: <IconUsers />,
      cell: (r) => <span className="tabular-nums">{fmtNum(r.people)}</span>,
    },
    {
      id: "requests",
      header: T.requests,
      cell: (r) => <span className="tabular-nums">{fmtNum(r.requests)}</span>,
    },
    {
      id: "first",
      header: T.firstAt,
      cell: (r) => <span className="text-xs text-ink-muted">{fmtDate(r.first_at)}</span>,
    },
    {
      id: "last",
      header: T.lastAt,
      cell: (r) => <span className="text-xs text-ink-muted">{fmtDate(r.last_at)}</span>,
    },
    {
      id: "status",
      header: T.status,
      cell: (r) => (
        <span className="flex flex-col gap-1">
          <Badge variant={STATUS_VARIANT[r.status]}>{T.st[r.status]}</Badge>
          {r.status === "ready" && (
            <span className="text-xs text-warning">{n(T.notifyCount, r.notifiable)}</span>
          )}
        </span>
      ),
    },
  ];

  const focus = (r: Row) => `${r.lat.toFixed(5)},${r.lng.toFixed(5)}`;

  return (
    <div>
      {/* **رأسٌ واحدٌ مرتّب** (قرارُ المالك ٢٠٢٦-١٠-٢٦: «تصميم احترافي كالسابق»): العنوانُ بأيقونته،
          والأرقامُ الأربعةُ بصفٍّ صغير، وتحتها الفلاترُ تبويباتٌ بسطرٍ واحدٍ بعدّها. */}
      <section className="surface mb-4 overflow-hidden">
        <div className="flex items-center gap-3 p-4">
          <span className="inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-full bg-accent text-on-bright">
            <IconLocation size={22} />
          </span>
          <div className="min-w-0">
            <h1 className="heading-page">{T.title}</h1>
            <p className="text-xs text-ink-muted">{T.subtitle}</p>
          </div>
        </div>
        <div className="grid grid-cols-2 gap-2 px-4 pb-4 md:grid-cols-4">
          {(
            [
              [IconUsers, T.people, s?.people, ""],
              [IconTrendUp, T.newThisWeek, s?.new_this_week, s && s.new_this_week > 0 ? "bg-accent text-on-bright" : ""],
              [IconZones, T.cities, s?.cities, ""],
              [IconHourglass, T.waitingNotNotified, s?.waiting_not_notified, s && s.waiting_not_notified > 0 ? "bg-warning-solid text-on-solid" : ""],
            ] as const
          ).map(([Icon, label, v, tone]) => (
            <div key={label} className="flex items-center gap-3 rounded-card border border-line-soft p-3">
              <span className={`inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full ${tone || "bg-field text-ink-muted"}`}>
                <Icon size={17} />
              </span>
              <span className="min-w-0">
                <span className="figure block leading-none">{v == null ? "—" : fmtNum(v)}</span>
                <span className="block truncate text-xs text-ink-muted">{label}</span>
              </span>
            </div>
          ))}
        </div>
        <nav className="flex gap-1 overflow-x-auto border-t border-line-soft px-2" role="tablist">
          {(
            [
              ["all", T.filterAll, s?.rows.length, IconLocation],
              ["waiting", T.filterWaiting, s?.waiting_not_notified, IconHourglass],
              ["ready", T.filterReady, s?.ready_places, IconSend],
              ["uncovered", T.filterUncovered, s?.rows.filter((r) => r.status === "uncovered").length, IconZones],
            ] as const
          ).map(([id, label, count, Icon]) => {
            const on = filter === id;
            const warn = (id === "waiting" || id === "ready") && (count ?? 0) > 0;
            return (
              <button
                key={id}
                type="button"
                role="tab"
                aria-selected={on}
                onClick={() => setFilter(id)}
                className={`relative inline-flex shrink-0 items-center gap-2 px-3 py-3 text-sm font-bold transition-colors ${
                  on ? "text-ink" : "text-ink-muted hover:text-ink"
                }`}
              >
                <Icon size={16} />
                {label}
                <span
                  className={`min-w-[1.5rem] rounded-full px-1.5 py-0.5 text-center text-2xs tabular-nums ${
                    warn ? "bg-warning-solid text-on-solid" : on ? "bg-accent text-on-bright" : "bg-field text-ink-muted"
                  }`}
                >
                  {count == null ? "…" : fmtNum(count)}
                </span>
                {on && <span className="absolute inset-x-2 bottom-0 h-0.5 rounded-full bg-accent" />}
              </button>
            );
          })}
        </nav>
      </section>

      {data.error && <Alert className="mb-4">{m.errors.internal}</Alert>}
      {msg && (
        <Alert tone={msg.tone} className="mb-4" onDismiss={() => setMsg(null)}>
          {msg.text}
        </Alert>
      )}

      {/* **التذكير** — منطقةٌ مغطّاةٌ ومنتظروها لم يُبلَّغوا. */}
      {s && s.pending_notify > 0 && (
        <Alert tone="warning" title={T.reminderTitle} className="mb-4">
          <button
            type="button"
            className="underline underline-offset-2"
            onClick={() => setFilter("ready")}
          >
            {n(T.reminderBody, s.pending_notify, s.ready_places)}
          </button>
        </Alert>
      )}

      <DataView
        items={rows}
        loading={data.loading && !s}
        getKey={(r) => r.key}
        columns={columns}
        view="cards"
        empty={filter === "all" ? T.empty : T.emptyFilter}
        actions={(r) => (
          <>
            <ButtonLink
              variant="secondary"
              href={`/dashboard/opsmap?focus=${focus(r)}`}
              className="flex items-center gap-1.5"
            >
              <IconLocation size={15} />
              {T.showOnMap}
            </ButtonLink>
            {canNotify && r.status === "uncovered" && r.kind === "coverage_request" && (
              <ButtonLink variant="secondary" href={`/dashboard/opsmap?focus=${focus(r)}&draw=1`}>
                {T.drawHere}
              </ButtonLink>
            )}
            {canNotify && r.status !== "uncovered" && (
              <Button
                variant={r.notifiable > 0 ? "primary" : "secondary"}
                disabled={busy || r.notifiable === 0}
                onClick={() => setAsking(r)}
                className="flex items-center gap-1.5"
                title={r.notifiable > 0 ? n(T.notifyCount, r.notifiable) : T.notifyNone}
              >
                <IconSend size={15} />
                {T.notifyNow}
                <span className="tabular-nums">({fmtNum(r.notifiable)})</span>
              </Button>
            )}
          </>
        )}
      />

      {/* **وطلبُ التوسّع بالمكان — مدينةً ومحافظةً** (قرارُ المالك ٢٠٢٦-١٠-٢٥: نُقل من خريطة العمليّات
          إلى قسمه). وبقدرة بابه. */}
      {canPlaces && (
        <div className="mt-4">
          <PlaceDemandPanel />
        </div>
      )}

      <Confirm
        open={!!asking}
        title={T.confirmTitle}
        body={asking ? n(T.confirmBody, asking.notifiable) : undefined}
        confirmLabel={T.confirm}
        tone="primary"
        busy={busy}
        onConfirm={() => asking && void notify(asking)}
        onCancel={() => setAsking(null)}
      />
    </div>
  );
}
