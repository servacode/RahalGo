"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **طلباتُ التوسّع — قسمٌ مستقلّ** (قرارُ المالك ٢٠٢٦-١٠-٠٤)
 * ══════════════════════════════════════════════════════════════════════
 *
 * **«مو مخفيّة تحت الخريطة»** — أرقامٌ تُضغط (أشخاص · جديدُ آخر ٧ أيام · لسا ما انبلغوا ·
 * جاهزين للإبلاغ)، وكرتٌ لكلّ مكان: اسمُه كبير وعددُ الأشخاص كبير وحالُه بلونه وأفعالُه
 * بعرض الكرت. **والملغاةُ والمرفوضةُ لا تُحسب.**
 *
 * **والإبلاغُ بزرٍّ لا آليّاً** — «بلّغ N شخص» يقول عددَهم قبل الضغط ويطلب تأكيداً.
 * **ولا زرَّ ميّتاً**: مكانٌ لا أحدَ فيه يُبلَّغ يقول «كلهم انبلغوا ✓» بدل زرٍّ رماديّ.
 *
 * **و«لسا ما انبلغوا» عينُ رقم الرئيسة** (`expansion_waiting`) — يفتحها رابطُ بطاقتها
 * بـ`?filter=waiting`، **وتفلتر بـ`waiting_accounts`** (طلباتٌ لها حسابٌ لم تُبلَّغ)،
 * فمجموعُ البطاقات يطابق الرقم (تدقيقُ ٢٠٢٦-١٠-٢٧).
 *
 * **و«حسب المدينة والمحافظة» من الردّ نفسِه** (`by_city`) — كانت لوحةٌ منفصلةٌ تعدّ الملغى
 * والمرفوض وتأخذ المدينةَ من غير مصدرها، فيفترق رقمُها عن رقم الصفحة.
 */

import { useMemo, useState } from "react";
import { getMessages, defaultLocale, fmtNum, fmtDate, errorText } from "@rahalgo/i18n";
import {
  Badge,
  Button,
  ButtonLink,
  Alert,
  Confirm,
  AdminPage,
  AdminAsync,
  AdminCard,
  FilterBar,
  FilterSelect,
  SummaryStrip,
  useLiveData,
  IconLocation,
  IconZones,
  IconSend,
  IconCheck,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useCanCall } from "@/lib/policy";

const m = getMessages(defaultLocale);
const T = m.admin.expansion;

type Status = "uncovered" | "ready" | "covered";
type Kind = "coverage_request" | "service_interest";

interface Row {
  key: string;
  kind: Kind;
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
  waiting_accounts: number;
  notifiable: number;
  status: Status;
}

interface CityAgg {
  city_id: string;
  city_name: string;
  governorate_id: string;
  governorate_name: string;
  people: number;
  requests: number;
  places: number;
}

interface Summary {
  people: number;
  new_this_week: number;
  cities: number;
  waiting_not_notified: number;
  pending_notify: number;
  ready_places: number;
  rows: Row[];
  by_city?: CityAgg[];
}

type Filter = "all" | "waiting" | "ready" | "uncovered" | "covered";
type KindFilter = "all" | Kind;

const STATUS_VARIANT: Record<Status, "neutral" | "warning" | "success"> = {
  uncovered: "neutral",
  ready: "warning",
  covered: "success",
};

const STATUS_TINT: Record<Status, string> = {
  uncovered: "bg-field text-ink-muted",
  ready: "bg-warning-solid text-on-solid",
  covered: "bg-success-solid text-on-solid",
};

const FILTERS: readonly Filter[] = ["all", "ready", "waiting", "uncovered", "covered"];

/** **الفلترُ من الرابط** — بطاقةُ الرئيسة تفتح على «لسا ما انبلغوا». */
function initialFilter(): Filter {
  if (typeof window === "undefined") return "all";
  const f = new URLSearchParams(window.location.search).get("filter");
  return (FILTERS as readonly string[]).includes(f ?? "") ? (f as Filter) : "all";
}

/** **أيدخل الصفُّ هذا الفلتر؟** — قاعدةٌ واحدةٌ للعدّاد وللبطاقات، فلا يفترقان. */
function matches(r: Row, f: Filter): boolean {
  switch (f) {
    // **وعدُّ هذا الفلتر هو رقمُ الرئيسة** — طلباتُ «أضف منطقتي» لها حسابٌ ولم تُبلَّغ.
    case "waiting":
      return r.kind === "coverage_request" && (r.waiting_accounts ?? 0) > 0;
    case "ready":
      return r.status === "ready";
    case "uncovered":
      return r.status === "uncovered";
    case "covered":
      return r.status === "covered";
    default:
      return true;
  }
}

const n = (tpl: string, v: number, p?: number) => tpl.replace("{n}", fmtNum(v)).replace("{p}", fmtNum(p ?? 0));

const FILTER_LABEL: Record<Filter, string> = {
  all: T.filterAll,
  ready: T.filterReady,
  waiting: T.filterWaiting,
  uncovered: T.filterUncovered,
  covered: T.filterCovered,
};

export default function ExpansionPage() {
  const canCall = useCanCall();
  const canNotify = canCall("POST", "/ops-map/expansion/notify");
  const canDraw = canCall("POST", "/ops-map/coverage");
  const [filter, setFilter] = useState<Filter>(initialFilter);
  const [kind, setKind] = useState<KindFilter>("all");
  const [byCityOpen, setByCityOpen] = useState(false);
  const [asking, setAsking] = useState<Row | null>(null);
  const [busy, setBusy] = useState(false);
  const [msg, setMsg] = useState<{ tone: "success" | "error"; text: string } | null>(null);

  const data = useLiveData<Summary>(() => api<Summary>("/api/v1/admin/ops-map/expansion"), []);
  const s = data.data;

  const byKind = useMemo(
    () => (s?.rows ?? []).filter((r) => kind === "all" || r.kind === kind),
    [s, kind],
  );
  const rows = useMemo(() => byKind.filter((r) => matches(r, filter)), [byKind, filter]);
  const countOf = (f: Filter) => byKind.filter((r) => matches(r, f)).length;
  const kindCount = (k: KindFilter) => (s?.rows ?? []).filter((r) => k === "all" || r.kind === k).length;

  const pick = (f: Filter) => {
    setFilter(f);
    setKind("all");
  };

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

  const kindChips = (["all", "coverage_request", "service_interest"] as const).map((k) => ({
    id: k,
    label: k === "all" ? T.kindAll : k === "coverage_request" ? T.kindCoverage : T.kindInterest,
    count: s ? kindCount(k) : undefined,
  }));

  const cities = s?.by_city ?? [];
  const filtered = filter !== "all" || kind !== "all";
  const showAll = () => {
    setFilter("all");
    setKind("all");
  };

  return (
    <AdminPage<Filter>
      icon={IconLocation}
      title={T.title}
      count={s ? (s.rows ?? []).length : null}
      tabs={FILTERS.map((f) => ({
        key: f,
        label: FILTER_LABEL[f],
        count: s ? countOf(f) : null,
        attention: f === "ready" || f === "waiting",
      }))}
      tab={filter}
      onTab={setFilter}
      notices={
        msg ? (
          <Alert tone={msg.tone} onDismiss={() => setMsg(null)}>
            {msg.text}
          </Alert>
        ) : undefined
      }
      toolbar={
        <FilterBar filtered={filtered} onClear={showAll}>
          <FilterSelect id="exp-kind" label={T.filterKind} value={kind} onChange={(v) => setKind(v as KindFilter)}>
            {kindChips.map((k) => (
              <option key={k.id} value={k.id}>
                {k.count != null ? `${k.label} (${fmtNum(k.count)})` : k.label}
              </option>
            ))}
          </FilterSelect>
        </FilterBar>
      }
      summary={
        <SummaryStrip
          items={[
            {
              key: "people",
              label: T.people,
              value: s ? fmtNum(s.people) : "—",
              meta: s ? n(T.citiesSub, s.cities) : undefined,
              onOpen: () => {
                pick("all");
                setByCityOpen(true);
              },
            },
            {
              key: "week",
              label: T.newWeek,
              value: s ? fmtNum(s.new_this_week) : "—",
              tone: s && s.new_this_week > 0 ? "attention" : "muted",
              onOpen: () => pick("all"),
            },
            {
              key: "waiting",
              label: T.waitingNotNotified,
              value: s ? fmtNum(s.waiting_not_notified) : "—",
              meta: s ? n(T.placesSub, (s.rows ?? []).filter((r) => matches(r, "waiting")).length) : undefined,
              tone: s && s.waiting_not_notified > 0 ? "attention" : "muted",
              onOpen: () => pick("waiting"),
            },
            {
              key: "ready",
              label: T.readyTile,
              value: s ? fmtNum(s.pending_notify) : "—",
              meta: s ? n(T.placesSub, s.ready_places) : undefined,
              tone: s && s.pending_notify > 0 ? "attention" : "muted",
              onOpen: () => pick("ready"),
            },
          ]}
        />
      }
    >
      <AdminAsync
        loading={data.loading}
        error={data.error}
        onRetry={data.reload}
        hasData={!!s}
        empty={!!s && rows.length === 0}
        emptyTitle={filtered ? T.emptyFilterTitle : T.emptyTitle}
        emptyIcon={IconLocation}
        emptyAction={
          filtered ? (
            <Button variant="secondary" onClick={showAll}>
              {T.showAll}
            </Button>
          ) : undefined
        }
      >
        <ul className="admin-grid">
          {rows.map((r) => (
            <PlaceCard
              key={r.key}
              r={r}
              canNotify={canNotify}
              canDraw={canDraw}
              busy={busy}
              onNotify={() => setAsking(r)}
            />
          ))}
        </ul>
      </AdminAsync>

      {/* **حسب المدينة والمحافظة** — مطويّ، ومن الردّ نفسِه فلا يفترق رقمُه عن رقم الأماكن. */}
      {cities.length > 0 && (
        <AdminCard
          title={T.byCityTitle}
          actions={
            <Button variant="secondary" size="sm" onClick={() => setByCityOpen((v) => !v)}>
              {byCityOpen ? T.byCityHide : T.byCityShow}
            </Button>
          }
        >
          {byCityOpen && (
            <ul className="admin-grid">
              {cities.map((c) => (
                <li
                  key={c.city_id || `gov:${c.governorate_id}`}
                  className="flex items-center gap-3 rounded-card border border-line-soft p-3"
                >
                  <span className="min-w-0 flex-1">
                    <span className="block truncate font-bold">{c.city_name || c.governorate_name || T.unresolved}</span>
                    <span className="block truncate text-xs text-ink-muted">
                      {c.city_name ? c.governorate_name || T.unresolved : ""}
                      {c.city_name ? " · " : ""}
                      {n(T.placesN, c.places)} · {n(T.requestsN, c.requests)}
                    </span>
                  </span>
                  <span className="shrink-0 text-end">
                    <span className="figure block leading-none">{fmtNum(c.people)}</span>
                    <span className="block text-xs text-ink-muted">{T.peopleBig}</span>
                  </span>
                </li>
              ))}
            </ul>
          )}
        </AdminCard>
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
    </AdminPage>
  );
}

/** **كرتُ مكان** — الاسمُ كبير، وعددُ الأشخاص كبير، والحالُ بلونه، والأفعالُ بعرض الكرت. */
function PlaceCard({
  r,
  canNotify,
  canDraw,
  busy,
  onNotify,
}: {
  r: Row;
  canNotify: boolean;
  canDraw: boolean;
  busy: boolean;
  onNotify: () => void;
}) {
  const focus = `${r.lat.toFixed(5)},${r.lng.toFixed(5)}`;
  const isWholeCity = r.kind === "service_interest" && r.key.startsWith("city:");
  const name = r.zone_name || (isWholeCity ? r.city_name || T.wholeCity : r.label || T.outside);
  const where = [
    !r.zone_name && !isWholeCity && r.label ? T.outside : "",
    isWholeCity ? T.wholeCity : r.city_name,
    r.governorate_name,
  ].filter(Boolean);
  const showDraw = canDraw && r.status === "uncovered" && r.kind === "coverage_request";
  const showNotify = canNotify && r.status !== "uncovered" && r.notifiable > 0;

  return (
    <li className="admin-panel flex flex-col gap-3 p-4">
      <div className="flex items-start gap-3">
        <span className={`inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-full ${STATUS_TINT[r.status]}`}>
          {r.kind === "coverage_request" ? <IconZones size={20} /> : <IconLocation size={20} />}
        </span>
        <span className="min-w-0 flex-1">
          <span className="block truncate text-base font-bold">{name}</span>
          <span className="block truncate text-xs text-ink-muted">
            {where.length > 0 ? where.join(" · ") : T.unresolved}
          </span>
        </span>
        <span className="flex shrink-0 flex-col items-end gap-1">
          <Badge variant={STATUS_VARIANT[r.status]}>{T.st[r.status]}</Badge>
          <Badge variant={r.kind === "coverage_request" ? "primary" : "info"}>
            {r.kind === "coverage_request" ? T.kindCoverage : T.kindInterest}
          </Badge>
        </span>
      </div>

      <div className="flex items-end justify-between gap-2 rounded-control bg-field px-3 py-2.5">
        <span>
          <span className="figure block leading-none">{fmtNum(r.people)}</span>
          <span className="mt-1 block text-xs text-ink-muted">
            {T.peopleBig} · {n(T.requestsN, r.requests)}
          </span>
        </span>
        <span className="text-end text-xs">
          {r.status === "ready" ? (
            <b className="text-warning">{n(T.notifyN, r.notifiable)}</b>
          ) : r.status === "covered" ? (
            <span className="inline-flex items-center gap-1 font-bold text-success">
              <IconCheck size={14} />
              {r.notified > 0 ? T.allNotified : T.noAccounts}
            </span>
          ) : r.waiting_accounts > 0 ? (
            <span className="text-ink-muted">{n(T.waitingN, r.waiting_accounts)}</span>
          ) : null}
          {r.notified > 0 && r.status !== "covered" && (
            <span className="block text-ink-muted">{n(T.notifiedN, r.notified)}</span>
          )}
        </span>
      </div>

      <div className="flex flex-wrap items-center justify-between gap-2 text-xs text-ink-muted">
        <span>
          {T.firstAt}: <span dir="ltr">{fmtDate(r.first_at)}</span>
        </span>
        <span>
          {T.lastAt}: <span dir="ltr">{fmtDate(r.last_at)}</span>
        </span>
      </div>

      <div className={`mt-auto grid grid-cols-1 gap-2 border-t border-line-soft pt-3 ${showNotify || showDraw ? "sm:grid-cols-2" : ""}`}>
        {showNotify && (
          <Button variant="primary" className="flex w-full items-center justify-center gap-1.5" disabled={busy} onClick={onNotify}>
            <IconSend size={15} />
            {n(T.notifyN, r.notifiable)}
          </Button>
        )}
        {showDraw && (
          <ButtonLink variant="primary" className="w-full justify-center" href={`/dashboard/opsmap?focus=${focus}&draw=1`}>
            {T.drawHere}
          </ButtonLink>
        )}
        <ButtonLink
          variant="secondary"
          href={`/dashboard/opsmap?focus=${focus}`}
          className="flex w-full items-center justify-center gap-1.5"
        >
          <IconLocation size={15} />
          {T.showOnMap}
        </ButtonLink>
      </div>
    </li>
  );
}
