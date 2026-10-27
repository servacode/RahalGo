"use client";

/**
 * **غرفةُ الطوارئ** — كلُّ الطوارئ في صندوقٍ واحد (قراراتُ المالك ٢٠٢٦-١٠-٠٤).
 *
 * # لماذا شاشةٌ لا إشعارٌ وحدَه
 *
 * الإشعارُ يمرّ في الشريط ويُقرأ مرّةً. **وطارئٌ يُقرأ ولا يُغلق طارئٌ نُسي**:
 * من سأل عنه بعد يومين لم يجد من يقول ماذا جرى.
 *
 * # والتصميمُ الاحترافيّ (٢٠٢٦-١٠-٢٧)
 *
 * رأسٌ موحّدٌ بتبويبَي «مفتوحة» و«مسكّرة» — **وعددُ المفتوحة من `/emergencies/count`
 * دائماً**، لا من صفحة التبويب الظاهر (كان يختفي على «المسكّرة»). ثلاثةُ أرقامٍ تُضغط
 * فتفلتر (مفتوحة · ما حدا استلمها · متأخرة)، **وكرتٌ لكلّ طارئٍ** بمدّته وشريطِ خطواته
 * والخطوةِ الجاية وأزرارٍ بعرضٍ كامل. **والفشلُ يُقال** — كانت القائمةُ تدور للأبد.
 */

import { useState } from "react";
import { getMessages, defaultLocale, fmtRef, fmtDateTime, fmtSpan, fmtNum, errorText } from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  ButtonLink,
  Chips,
  PageContainer,
  Pagination,
  LoadingState,
  ReloadState,
  useLiveData,
  IconWarning,
  IconPhone,
  IconStatus,
  IconCheck,
  IconHourglass,
  IconSuccess,
} from "@rahalgo/ui";
import { api } from "@/lib/api";
import { PageHero } from "@/components/admin/PageHero";
import { Tile } from "@/components/admin/DashParts";
import { kindVariant, kindTint } from "@/components/admin/emergency/kinds";

const m = getMessages(defaultLocale);
const E = m.admin.emergencies;
const R = m.admin.emergencyRoom;

interface EmergencyListRow {
  id: string;
  driver_name: string | null;
  driver_phone: string;
  order_number: number | null;
  note: string;
  created_at: string;
  resolution: string;
  resolved_at: string | null;
  resolved_by: string;
  kind: keyof typeof R.kinds;
  stage: "" | keyof typeof R.stages;
  merchant_name: string;
  acknowledged_at: string | null;
  acknowledged_by: string;
  stale: boolean;
  driver_ok: boolean;
  outcome: string;
  money_done: boolean;
}

interface Counts {
  open: number;
  unacked: number;
  stale: number;
  stale_after_min: number;
}

type Tab = "open" | "resolved";
type Only = "" | "unacked" | "stale";

const since = (iso: string) => fmtSpan(Math.max(0, (Date.now() - Date.parse(iso)) / 1000));

/** **طارئُ سائق** — له «السائقُ بخير» ومصيرُ الطلب والتعويض (`missingStep` في المحرّك). */
const driverKind = (k: string) => k !== "store_closure" && k !== "platform_halt";

/** **خطواتُ الحلّ بترتيب المحرّك** — وما قُطع منها. */
function stepsOf(x: EmergencyListRow): { key: string; label: string; done: boolean }[] {
  const out = [{ key: "ack", label: R.steps.ack, done: !!x.acknowledged_at }];
  if (driverKind(x.kind)) {
    out.push({ key: "driver_ok", label: R.steps.driverOk, done: x.driver_ok });
    if (x.order_number !== null) {
      out.push({ key: "outcome", label: R.steps.outcome, done: x.outcome !== "" });
      out.push({ key: "money", label: R.steps.money, done: x.money_done });
    }
  }
  out.push({ key: "resolve", label: R.steps.done, done: !!x.resolved_at });
  return out;
}

export default function EmergenciesPage() {
  const [page, setPage] = useState(1);
  const [tab, setTabRaw] = useState<Tab>("open");
  const [only, setOnlyRaw] = useState<Only>("");
  const [busy, setBusy] = useState("");
  const [error, setError] = useState("");

  // **ومفتاحُ الطلب مع الجواب** — فبعد تبديل التبويب لا تُعرض بياناتُ القديم تحت عنوان الجديد.
  const key = `${tab}|${only}|${page}`;
  const list = useLiveData<{
    key: string;
    emergencies: EmergencyListRow[];
    total: number;
    per_page: number;
  }>(
    async () => {
      const qs = new URLSearchParams({ page: String(page), status: tab });
      if (tab === "open" && only) qs.set("only", only);
      const r = await api<{ emergencies: EmergencyListRow[]; total: number; per_page: number }>(
        `/api/v1/admin/emergencies?${qs.toString()}`,
      );
      return { ...r, key };
    },
    ["driver", "order", "emergency"],
    [key],
  );
  const counts = useLiveData<Counts>(() => api<Counts>("/api/v1/admin/emergencies/count"), ["emergency"], []);

  function setTab(t: Tab) {
    setTabRaw(t);
    setOnlyRaw("");
    setPage(1);
  }
  function setOnly(o: Only) {
    setTabRaw("open");
    setOnlyRaw(o);
    setPage(1);
  }

  async function ack(id: string) {
    setBusy(id);
    setError("");
    try {
      await api(`/api/v1/admin/emergencies/${id}/ack`, { method: "POST", body: "{}" });
      list.reload();
      counts.reload();
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy("");
    }
  }

  const c = counts.data;
  const fresh = list.data && list.data.key === key ? list.data : null;
  const rows = fresh?.emergencies ?? [];

  const onlyChips = [
    { id: "", label: E.filterAll },
    { id: "unacked", label: E.filterUnacked },
    { id: "stale", label: E.filterStale },
  ];

  return (
    <PageContainer>
      <PageHero<Tab>
        icon={IconWarning}
        title={E.title}
        subtitle={E.hint}
        tabs={[
          { key: "open", label: E.tabOpen, icon: IconWarning, count: c?.open ?? null, warn: true },
          { key: "resolved", label: E.tabResolved, icon: IconStatus },
        ]}
        value={tab}
        onChange={setTab}
      />

      {/* **الأرقامُ الثلاثة تُضغط فتفلتر** — و«متأخرة» بمهلة الإعداد نفسِها. */}
      <div className="mb-4 grid grid-cols-1 gap-2 sm:grid-cols-3">
        <Tile
          icon={IconWarning}
          label={E.tileOpen}
          value={fmtNum(c?.open ?? 0)}
          sub={E.tileOpenSub}
          tone={(c?.open ?? 0) > 0 ? "warning" : "success"}
          onClick={() => setOnly("")}
        />
        <Tile
          icon={IconHourglass}
          label={E.tileUnacked}
          value={fmtNum(c?.unacked ?? 0)}
          sub={E.tileUnackedSub}
          tone={(c?.unacked ?? 0) > 0 ? "danger" : "muted"}
          onClick={() => setOnly("unacked")}
        />
        <Tile
          icon={IconWarning}
          label={E.tileStale}
          value={fmtNum(c?.stale ?? 0)}
          sub={E.tileStaleSub.replace("{n}", fmtNum(c?.stale_after_min ?? 10))}
          tone={(c?.stale ?? 0) > 0 ? "danger" : "muted"}
          onClick={() => setOnly("stale")}
        />
      </div>

      {tab === "open" && (
        <section className="surface mb-4 flex flex-wrap items-center gap-2 p-3">
          <span className="text-sm text-ink-muted">{E.filterLabel}:</span>
          <Chips items={onlyChips} value={only} onChange={(v) => setOnly(v as Only)} wrap />
        </section>
      )}

      {error && <Alert className="mb-3">{error}</Alert>}

      {list.error && !fresh ? (
        <section className="surface">
          <ReloadState onRetry={list.reload} label={E.loadError} />
        </section>
      ) : !fresh ? (
        <LoadingState />
      ) : rows.length === 0 ? (
        <Empty tab={tab} filtered={only !== ""} />
      ) : (
        <ul className="grid grid-cols-1 gap-3 lg:grid-cols-2">
          {rows.map((x) => (
            <EmergencyCard key={x.id} x={x} busy={busy === x.id} onAck={() => void ack(x.id)} />
          ))}
        </ul>
      )}

      {fresh && fresh.total > fresh.per_page && (
        <div className="mt-4 flex justify-center">
          <Pagination page={page} total={fresh.total} perPage={fresh.per_page} onChange={setPage} />
        </div>
      )}
    </PageContainer>
  );
}

/** **فارغٌ لكلّ تبويب** — كان «المسكّرة» الفارغُ يقول «لا بلاغات مفتوحة». */
function Empty({ tab, filtered }: { tab: Tab; filtered: boolean }) {
  const [title, hint, tint] =
    tab === "resolved"
      ? [E.emptyResolved, E.emptyResolvedHint, "bg-field text-ink-muted"]
      : filtered
        ? [E.emptyFiltered, E.emptyFilteredHint, "bg-field text-ink-muted"]
        : [E.emptyOpen, E.emptyOpenHint, "bg-success-solid text-on-solid"];
  return (
    <section className="surface flex flex-col items-center gap-3 p-8 text-center">
      <span className={`inline-flex h-16 w-16 items-center justify-center rounded-full ${tint}`}>
        {tab === "open" && !filtered ? <IconSuccess size={28} /> : <IconStatus size={28} />}
      </span>
      <h2 className="heading-section">{title}</h2>
      <p className="max-w-md text-sm text-ink-muted">{hint}</p>
    </section>
  );
}

/** **كرتُ طارئ** — من، ومن متى، ووين وصلنا، والخطوةُ الجاية، وأفعالُه بعرضٍ كامل. */
function EmergencyCard({ x, busy, onAck }: { x: EmergencyListRow; busy: boolean; onAck: () => void }) {
  const open = !x.resolved_at;
  const steps = stepsOf(x);
  const next = steps.find((s) => !s.done);
  // **واسمُ توقّف المنصّة ملخّصٌ لا تكرارٌ للشارة** — كان «توقف المنصة» مرّتين.
  const who =
    x.driver_name ||
    x.driver_phone ||
    x.merchant_name ||
    (x.kind === "platform_halt" ? x.note || E.haltSummary : R.kinds[x.kind]);
  const showNote = x.note && who !== x.note;
  return (
    <li className={`surface flex flex-col gap-3 p-4 ${x.stale ? "border border-danger" : ""}`}>
      <div className="flex items-start gap-3">
        <span
          className={`inline-flex h-11 w-11 shrink-0 items-center justify-center rounded-full ${kindTint(x.kind)}`}
        >
          <IconWarning size={20} />
        </span>
        <span className="min-w-0 flex-1">
          <span className="flex flex-wrap items-center gap-1.5">
            <Badge variant={kindVariant(x.kind)}>{R.kinds[x.kind] ?? x.kind}</Badge>
            {x.stage && <Badge>{R.stages[x.stage]}</Badge>}
          </span>
          <span className="mt-1 block truncate font-bold">{who}</span>
          {x.order_number !== null && (
            <span className="block text-2xs text-ink-muted">
              {E.orderNo} <span dir="ltr">#{fmtRef(x.order_number)}</span>
            </span>
          )}
        </span>
        <span className="shrink-0 text-end">
          <span className={`figure block ${open && x.stale ? "text-danger" : ""}`}>
            {E.since.replace("{t}", since(x.created_at))}
          </span>
          <span className="block text-2xs text-ink-muted" dir="ltr">
            {fmtDateTime(x.created_at)}
          </span>
        </span>
      </div>

      {showNote && <p className="border-s-2 border-line ps-2 text-sm">{x.note}</p>}

      {/* **شريطُ الخطوات** — المقطوعُ أخضر، والجاية بارزة، والباقي رماديّ. */}
      <ol className="flex flex-wrap gap-1.5">
        {steps.map((s, i) => (
          <li key={s.key}>
            <Badge variant={s.done ? "success" : next?.key === s.key && open ? "danger" : "neutral"}>
              {s.done && <IconCheck size={12} className="me-1 inline" />}
              {`${fmtNum(i + 1)}. ${s.label}`}
            </Badge>
          </li>
        ))}
      </ol>

      {open ? (
        <div className="space-y-0.5 text-sm">
          {next && <p className="font-bold">{E.nextStep.replace("{step}", next.label)}</p>}
          {x.acknowledged_at ? (
            <p className="text-xs text-success">{R.ackedBy.replace("{name}", x.acknowledged_by || "—")}</p>
          ) : (
            <p className={`text-xs ${x.stale ? "font-bold text-danger" : "text-warning"}`}>
              {x.stale ? R.stale.replace("{t}", since(x.created_at)) : R.unacked}
            </p>
          )}
        </div>
      ) : (
        <div className="space-y-0.5 rounded-control bg-field px-3 py-2 text-sm">
          <p>{x.resolution || E.noResolution}</p>
          <p className="text-2xs text-ink-muted">
            {E.resolvedBy.replace("{name}", x.resolved_by || "—")} ·{" "}
            <span dir="ltr">{x.resolved_at ? fmtDateTime(x.resolved_at) : ""}</span>
          </p>
        </div>
      )}

      <div className="mt-auto grid grid-cols-1 gap-2 border-t border-line-soft pt-3 sm:grid-cols-2">
        {open && !x.acknowledged_at && (
          <Button variant="danger" className="w-full sm:col-span-2" disabled={busy} onClick={onAck}>
            {R.ack}
          </Button>
        )}
        <ButtonLink
          href={`/dashboard/emergencies/${x.id}`}
          variant={open && x.acknowledged_at ? "primary" : "secondary"}
          className={`w-full ${x.driver_phone ? "" : "sm:col-span-2"}`}
        >
          {R.open}
        </ButtonLink>
        {x.driver_phone && (
          <ButtonLink href={`tel:${x.driver_phone}`} variant="secondary" className="w-full">
            <IconPhone size={15} />
            {R.callDriver}
          </ButtonLink>
        )}
      </div>
    </li>
  );
}
