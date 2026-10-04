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
  PageHeader,
  StatCard,
  Chips,
  DataView,
  Badge,
  Button,
  ButtonLink,
  Alert,
  Confirm,
  ViewToggle,
  useViewMode,
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
  const [view, setView] = useViewMode("expansion");
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
      <div className="mb-2 flex flex-wrap items-center justify-between gap-3">
        <PageHeader icon={IconLocation} title={T.title} />
        <ViewToggle
          view={view}
          onChange={setView}
          tableLabel={m.common.viewTable}
          cardsLabel={m.common.viewCards}
        />
      </div>
      <p className="mb-4 text-sm text-ink-muted">{T.subtitle}</p>

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

      <div className="mb-4 grid grid-cols-2 gap-3 md:grid-cols-4">
        <StatCard icon={IconUsers} label={T.people} value={s ? fmtNum(s.people) : "—"} />
        <StatCard
          icon={IconTrendUp}
          label={T.newThisWeek}
          value={s ? fmtNum(s.new_this_week) : "—"}
          tone={s && s.new_this_week > 0 ? "accent" : "muted"}
        />
        <StatCard icon={IconZones} label={T.cities} value={s ? fmtNum(s.cities) : "—"} />
        <StatCard
          icon={IconHourglass}
          label={T.waitingNotNotified}
          value={s ? fmtNum(s.waiting_not_notified) : "—"}
          tone={s && s.waiting_not_notified > 0 ? "warning" : "muted"}
          selected={filter === "waiting"}
          onClick={() => setFilter("waiting")}
        />
      </div>

      <Chips<Filter>
        className="mb-4"
        wrap
        value={filter}
        onChange={setFilter}
        items={[
          { id: "all", label: T.filterAll, count: s?.rows.length },
          { id: "waiting", label: T.filterWaiting, count: s?.waiting_not_notified },
          { id: "ready", label: T.filterReady, count: s?.ready_places },
          {
            id: "uncovered",
            label: T.filterUncovered,
            count: s?.rows.filter((r) => r.status === "uncovered").length,
          },
        ]}
      />

      <DataView
        items={rows}
        loading={data.loading && !s}
        getKey={(r) => r.key}
        columns={columns}
        view={view}
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
