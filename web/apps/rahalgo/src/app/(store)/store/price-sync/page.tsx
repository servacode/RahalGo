"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **‏/store/price-sync — مزامنةُ الأسعار والكميّات من الأمين** (قرارُ المالك ٢٠٢٦-١٠-١٩)
 * ══════════════════════════════════════════════════════════════════════
 *
 * المحاسبُ يصدّر «قائمة المواد» ويرفعها. **أوّلَ مرّةٍ يختار الأعمدة** فتُحفظ، **وبعدها
 * يرفع فيرى «جرّب» مباشرةً ثمّ «طبّق».** والقلبُ في المحرّك (`catalog.SyncPrices`).
 *
 * **ورمزُ الساعي هنا** — للمولات التي تقبل برنامجَ المزامنة على كمبيوتر الأمين.
 */

import { useCallback, useEffect, useState } from "react";
import { useRouter } from "next/navigation";
import {
  getMessages,
  defaultLocale,
  errorText,
  fmtNum,
  fmtDateTime,
} from "@rahalgo/i18n";
import { Alert, Button, Checkbox, Confirm, IconCheck, IconRotate, IconUpload, Kpi, KpiGrid, LoadingState, Pill, Select } from "@rahalgo/ui";
import { api, apiFile } from "@/lib/api";
import { useStore } from "@/lib/store/context";
import { fill } from "@/lib/customer/texts";
import { PageHeader } from "@/components/store/SetupParts";
import { PriceSyncSections } from "@/components/store/PriceSyncSections";
import { agentState, lastRealSync } from "@/components/store/AmeenStatus";

const m = getMessages(defaultLocale);
const P = m.store.priceSync;
const K = m.store.console;

interface Mapping {
  code: string;
  barcode?: string;
  name?: string;
  price: string;
  qty?: string;
  /** **عمود المجموعة** (٢٠٢٦-١١-٠٧) — اختياري؛ بيحدد قسم الصنف الجديد. */
  group?: string;
  currency?: "SYP" | "USD";
  /** **أنشئ غير الموجود أصنافاً مخفيّة** (٢٠٢٦-١١-٠٤) — مُطفأٌ افتراضاً. */
  create_missing?: boolean;
}
interface Issue {
  row: number;
  code?: string;
  name?: string;
  reason: string;
}
interface Report {
  dry_run: boolean;
  rows: number;
  matched: number;
  price_changed: number;
  made_available: number;
  made_unavailable: number;
  linked_by_name: number;
  unchanged: number;
  unmatched: number;
  created?: number;
  created_zero_price?: number;
  create_skipped?: number;
  /** **وين راحت الأصناف الجديدة** (٢٠٢٦-١١-٠٧). */
  created_by_name?: number;
  created_by_group?: number;
  created_pending?: number;
  /** **من الأمين وكميتها موجودة بس مخفية لأنو بلا صورة** (٢٠٢٦-١١-٠٧). */
  kept_hidden_no_photo?: number;
  zero_price?: number;
  skipped: Issue[];
  unmatched_sample: Issue[];
  /** **رقمٌ أو باركودٌ لم يُحفظ** (٢٠٢٦-١١-٠٥) — وصفُّه طُبّق، فلا يُذكر في «لم تُطبَّق». */
  link_rejected?: number;
  link_rejected_sample?: Issue[];
}
interface Info {
  mapping: Mapping | null;
  has_token: boolean;
  last_run: { source: string; report: Report; at: string } | null;
  agent_seen_at: string | null;
  agent_version: string | null;
  sync_requested: boolean;
  interval_minutes: number;
  agent_error: string | null;
  stopped: boolean;
  /** **موقوفة مؤقّتاً** (٢٠٢٦-١١-٠٧) — البرنامجُ باقٍ على الكمبيوتر. */
  paused: boolean;
  /** **كم صنفاً أنشأه الأمين** — لزرّ «احذف الأصناف المستوردة» (٢٠٢٦-١١-٠٨). */
  imported_count?: number;
}
interface Preview {
  headers: string[];
  sample: string[][];
  rows: number;
}

const EMPTY: Mapping = {
  code: "",
  barcode: "",
  name: "",
  price: "",
  qty: "",
  group: "",
  currency: "SYP",
};

export default function PriceSyncPage() {
  const { store } = useStore();
  const router = useRouter();
  const id = store?.id ?? "";
  /** **بلا برنامج الأمين لا صفحةَ هنا** (٢٠٢٦-١١-٠٦) — ويرجع إلى القائمة. */
  const noAmeen = !!store && store.uses_ameen !== true;
  useEffect(() => {
    if (noAmeen) router.replace("/store/menu");
  }, [noAmeen, router]);
  const [info, setInfo] = useState<Info | null>(null);
  const [askUninstall, setAskUninstall] = useState(false);
  const [askDeleteImported, setAskDeleteImported] = useState(false);
  const [deletedN, setDeletedN] = useState<number | null>(null);
  const [file, setFile] = useState<File | null>(null);
  const [preview, setPreview] = useState<Preview | null>(null);
  const [map, setMap] = useState<Mapping>(EMPTY);
  const [report, setReport] = useState<Report | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  /** **بعد كل تطبيق** تنقرا «توزيع الأقسام» من جديد. */
  const [applied, setApplied] = useState(0);

  const load = useCallback(() => {
    if (!id || noAmeen) return;
    api<Info>(`/api/v1/merchant/stores/${id}/price-sync`)
      .then((i) => {
        setInfo(i);
        if (i.mapping) setMap({ ...EMPTY, ...i.mapping });
      })
      .catch((e) => setError(errorText(e)));
  }, [id, noAmeen]);
  useEffect(load, [load]);

  async function send(f: File, mapping: Mapping | null, dry: boolean) {
    const body = new FormData();
    body.append("file", f);
    if (mapping) body.append("mapping", JSON.stringify(mapping));
    if (dry) body.append("dry_run", "1");
    return api<Report & Preview>(
      `/api/v1/merchant/stores/${id}/price-sync/file`,
      { method: "POST", body },
    );
  }

  /** **ملفٌّ اختير** — باختيارٍ محفوظٍ «جرّب» مباشرةً، وإلّا الأعمدةُ أوّلاً. */
  async function pick(f: File) {
    setFile(f);
    setReport(null);
    setPreview(null);
    setError("");
    setBusy(true);
    try {
      if (info?.mapping) {
        try {
          setReport(await send(f, map, true));
          return;
        } catch (e) {
          // **أعمدةٌ تبدّلت** ⇒ تُختار من جديد.
          if ((e as Error).message !== "price_sync_mapping") throw e;
        }
      }
      const p = await send(f, null, false);
      setPreview({ headers: p.headers, sample: p.sample, rows: p.rows });
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  async function run(dry: boolean, use: Mapping = map) {
    if (!file) return;
    if (!use.price || (!use.code && !use.name)) return setError(P.vMap);
    setBusy(true);
    setError("");
    try {
      setReport(await send(file, use, dry));
      if (!dry) {
        setPreview(null);
        load();
        setApplied((n) => n + 1);
      }
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  /** **كلّ كم دقيقة** (قرارُ المالك ٢٠٢٦-١٠-١٩) — تصل الساعيَ في سؤاله التالي. */
  async function saveInterval(n: number) {
    setBusy(true);
    try {
      await api(`/api/v1/merchant/stores/${id}/price-sync/interval`, {
        method: "PUT",
        body: JSON.stringify({ interval_minutes: n }),
      });
      setInfo((i) => (i ? { ...i, interval_minutes: n } : i));
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  /** **«مزامنة الآن»** — يراها الساعي في سؤاله التالي (دقيقة). */
  async function syncNow() {
    setBusy(true);
    try {
      await api(`/api/v1/merchant/stores/${id}/price-sync/request`, {
        method: "POST",
      });
      load();
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  /** **البرنامجُ برمز المتجر** — يُحمَّل ملفّاً واحداً (`RahalGoSync.exe`). */
  async function download() {
    setBusy(true);
    setError("");
    try {
      const res = await apiFile(
        `/api/v1/merchant/stores/${id}/price-sync/agent`,
      );
      const url = URL.createObjectURL(await res.blob());
      const a = document.createElement("a");
      a.href = url;
      a.download = "RahalGoSync.exe";
      a.click();
      setTimeout(() => URL.revokeObjectURL(url), 10_000);
      load();
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  /** **«احذف البرنامج من الكمبيوتر»** — يزيل نفسَه من الإقلاع في سؤاله التالي، وبعدها تحميلٌ جديد. */
  async function stopAgent() {
    setBusy(true);
    try {
      await api(`/api/v1/merchant/stores/${id}/price-sync/stop`, {
        method: "POST",
      });
      setAskUninstall(false);
      load();
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  /**
   * **«احذف الأصناف المستوردة من الأمين»** (قرارُ المالك ٢٠٢٦-١١-٠٨) — ما أنشأته المزامنةُ وحدَه،
   * والمُضافُ باليد يبقى. **وبرنامجٌ يعمل يُنشئها من جديد** — فالتأكيدُ يقول ذلك.
   */
  async function deleteImported() {
    setBusy(true);
    try {
      const r = await api<{ deleted: number }>(`/api/v1/merchant/stores/${id}/price-sync/delete-imported`, {
        method: "POST",
      });
      setAskDeleteImported(false);
      setDeletedN(r.deleted);
      setApplied((n) => n + 1);
      load();
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  /** **«إيقاف المزامنة» / «شغّل المزامنة»** (٢٠٢٦-١١-٠٧) — مؤقّت، بلا تحميلٍ جديد. */
  async function setPaused(paused: boolean) {
    setBusy(true);
    try {
      await api(
        `/api/v1/merchant/stores/${id}/price-sync/${paused ? "pause" : "resume"}`,
        { method: "POST" },
      );
      load();
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  if (!info && !error) return <LoadingState variant="list" rows={4} />;

  const col = (key: keyof Mapping, label: string, required = false) => (
    <Select
      id={`col-${key}`}
      label={label + (required ? " *" : "")}
      value={(map[key] as string) ?? ""}
      onChange={(e) => setMap({ ...map, [key]: e.target.value })}
    >
      <option value="">{P.none}</option>
      {(preview?.headers ?? []).filter(Boolean).map((h) => (
        <option key={h} value={h}>
          {h}
        </option>
      ))}
    </Select>
  );

  if (noAmeen) return null;

  const st = agentState(info);
  const last = lastRealSync(info);
  const linked = !!info?.agent_seen_at && !info.stopped;
  /** **التحميلُ هو الفعلُ الأوّلُ** حين لا برنامجَ يعمل بعد، وثانويٌّ حين يعمل. */
  const needsInstall = st.key === "unlinked" || st.key === "never" || st.key === "stopped";

  return (
    <div className="console-page" data-testid="store-price-sync">
      <PageHeader
        title={P.title}
        href="/store/menu"
        topLevel
        subtitle={info?.last_run ? fill(P.last, { at: fmtDateTime(info.last_run.at) }) : undefined}
      />

      {/* ══ **حالةُ برنامج الأمين أوّلُ ما في الصفحة** (ملاحظةُ المالك ٢٠٢٦-١١-٠٧) —
          الحالةُ وآخرُ مزامنةٍ فعليّةٍ وأفعالُها كلُّها في لوحٍ واحدٍ فوق العمودين. ══ */}
      <section
        className="panel overflow-hidden"
        aria-label={P.agentTitle}
        data-testid="sync-agent"
        data-state={st.key}
      >
        <div className="flex items-start gap-4 p-4 lg:p-5">
          <span className="icon-chip size-12 rounded-2xl lg:size-14" data-tone={st.tone} aria-hidden>
            <IconRotate size={26} />
          </span>
          <div className="flex min-w-0 flex-1 flex-col gap-2">
            <span className="flex flex-wrap items-center gap-2">
              <h2 className="console-h2">{P.agentTitle}</h2>
              <Pill tone={st.tone}>{st.label}</Pill>
            </span>
            {st.key === "error" ? (
              <div className="rounded-2xl border border-danger-edge bg-danger-tint p-3 text-sm" role="alert">
                <span className="block font-bold text-danger">{st.detail}</span>
                <span className="block text-ink">{st.why}</span>
              </div>
            ) : st.key === "paused" ? (
              <Alert tone="warning">
                <span className="block font-bold">{P.agentPaused}</span>
                <span className="block">{P.agentPausedHint}</span>
              </Alert>
            ) : (
              <p
                className={`text-sm ${st.key === "alive" ? "text-success" : st.key === "dead" ? "text-danger" : "text-ink-muted"}`}
              >
                {st.detail}
              </p>
            )}
            {last && <p className="console-meta">{last}</p>}
            {needsInstall && <p className="text-sm text-ink-muted">{P.agentSteps}</p>}
          </div>
        </div>

        <div className="flex flex-col gap-3 border-t border-line-soft bg-field p-4 lg:flex-row lg:flex-wrap lg:items-end lg:px-5">
          <div className="flex flex-wrap gap-2">
            {linked &&
              (info?.paused ? (
                <Button className="min-h-11" onClick={() => void setPaused(false)} disabled={busy}>
                  {P.agentResume}
                </Button>
              ) : (
                <>
                  {info?.sync_requested ? (
                    <p className="flex min-h-11 items-center gap-2 px-2 text-sm font-bold text-success">
                      <IconCheck size={18} />
                      {P.syncAsked}
                    </p>
                  ) : (
                    <Button className="min-h-11" onClick={() => void syncNow()} disabled={busy}>
                      {P.syncNow}
                    </Button>
                  )}
                  <Button className="min-h-11" variant="secondary" onClick={() => void setPaused(true)} disabled={busy}>
                    {P.agentStop}
                  </Button>
                </>
              ))}
            <Button
              className="min-h-11"
              variant={needsInstall ? "primary" : "secondary"}
              onClick={() => void download()}
              disabled={busy}
            >
              {busy ? P.agentDownloading : P.agentDownload}
            </Button>
          </div>
          {info?.has_token && !info.stopped && (
            <div className="lg:w-60">
              <Select
                id="sync-interval"
                label={P.interval}
                value={String(info.interval_minutes ?? 1)}
                onChange={(e) => void saveInterval(Number(e.target.value))}
                disabled={busy}
              >
                {[1, 5, 10, 15, 30, 60].map((n) => (
                  <option key={n} value={n}>
                    {n === 1 ? P.every1 : n === 60 ? P.every60 : fill(P.everyN, { n: fmtNum(n) })}
                  </option>
                ))}
              </Select>
            </div>
          )}
          <div className="flex min-w-0 flex-1 flex-col gap-1 lg:items-end">
            <p className="console-meta">{P.agentRedownload}</p>
            {/* **الحذفُ من الكمبيوتر أقلُّ بروزاً ومؤكَّد** — بعده تحميلٌ جديد (٢٠٢٦-١١-٠٧). */}
            {linked && (
              <button
                type="button"
                className="flex min-h-11 w-fit items-center text-xs text-danger underline disabled:opacity-60"
                onClick={() => setAskUninstall(true)}
                disabled={busy}
              >
                {P.agentUninstall}
              </button>
            )}
          </div>
        </div>
        <Confirm
          open={askUninstall}
          title={P.agentUninstallQ}
          body={P.agentUninstallBody}
          confirmLabel={P.agentUninstall}
          busy={busy}
          onConfirm={() => void stopAgent()}
          onCancel={() => setAskUninstall(false)}
        />
      </section>

      {error && <Alert>{error}</Alert>}

      {/* ══ عمودان على الحاسوب: الملفّ والأعمدة والتقرير | توزيع الأقسام. جوّالاً متتابعان. ══ */}
      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 md:items-start md:gap-5">
        <div className="flex min-w-0 flex-col gap-4">
          <label className="panel flex min-h-32 cursor-pointer flex-col items-center justify-center gap-3 dropzone p-6 text-center transition-colors hover:bg-row-hover focus-within:outline-2 focus-within:outline-offset-2 focus-within:outline-accent">
            <span className="icon-chip size-12 rounded-2xl" data-tone="accent" aria-hidden>
              <IconUpload size={24} />
            </span>
            <span className="font-bold text-accent-text">{busy ? P.reading : P.pick}</span>
            {file && (
              <span className="console-meta" dir="ltr">
                {file.name}
              </span>
            )}
            <input
              type="file"
              accept=".xlsx,.xlsm,.csv,.txt"
              className="sr-only"
              disabled={busy}
              onChange={(e) => {
                const f = e.target.files?.[0];
                if (f) void pick(f);
                e.target.value = "";
              }}
            />
          </label>

          {/* **أنشئ المواد غير الموجودة أصنافاً مخفية** (٢٠٢٦-١١-٠٤) — يُبدَّل فيُعاد «جرّب» ليُرى العدد. */}
          {file && (
            <div className="panel flex flex-col gap-1 p-4">
              <Checkbox
                id="sync-create-missing"
                data-testid="sync-create-missing"
                checked={!!map.create_missing}
                disabled={busy}
                onChange={(e) => {
                  const next = { ...map, create_missing: e.target.checked };
                  setMap(next);
                  if (report || info?.mapping) void run(true, next);
                }}
                label={<span className="font-bold">{P.createMissing}</span>}
              />
              <p className="console-meta">{P.createMissingHint}</p>
            </div>
          )}

          {preview && (
            <div className="panel flex flex-col gap-3 p-4">
              <h3 className="console-h2">{P.columnsTitle}</h3>
              <p className="console-meta">
                {P.columnsHint} · {fill(P.rows, { n: fmtNum(preview.rows) })}
              </p>
              <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
                {col("code", P.colCode)}
                {col("price", P.colPrice, true)}
                {col("qty", P.colQty)}
                {col("name", P.colName)}
                {col("barcode", P.colBarcode)}
                {col("group", P.colGroup)}
                <Select
                  id="col-currency"
                  label={P.currency}
                  value={map.currency ?? "SYP"}
                  onChange={(e) => setMap({ ...map, currency: e.target.value as "SYP" | "USD" })}
                >
                  <option value="SYP">{P.syp}</option>
                  <option value="USD">{P.usd}</option>
                </Select>
              </div>
              <div className="overflow-x-auto rounded-control border border-line-soft">
                <table className="console-table text-xs">
                  <thead>
                    <tr>
                      {preview.headers.map((h, i) => (
                        <th key={i} className="text-start">
                          {h}
                        </th>
                      ))}
                    </tr>
                  </thead>
                  <tbody>
                    {preview.sample.map((r, i) => (
                      <tr key={i}>
                        {preview.headers.map((_, j) => (
                          <td key={j}>{r[j] ?? ""}</td>
                        ))}
                      </tr>
                    ))}
                  </tbody>
                </table>
              </div>
              <Button className="min-h-11" onClick={() => void run(true)} disabled={busy}>
                {P.try}
              </Button>
            </div>
          )}

          {report && (
            <div className="panel flex flex-col gap-3 p-4 text-sm" data-testid="sync-report">
              <h3 className="console-h2">{report.dry_run ? P.dryTitle : P.doneTitle}</h3>
              <KpiGrid>
                <Kpi label={K.rpMatched} value={report.matched} />
                <Kpi label={K.rpChanged} value={report.price_changed} tone={report.price_changed > 0 ? "accent" : undefined} />
                <Kpi label={K.rpMadeAvailable} value={report.made_available} tone={report.made_available > 0 ? "good" : undefined} />
                <Kpi
                  label={K.rpMadeUnavailable}
                  value={report.made_unavailable}
                  tone={report.made_unavailable > 0 ? "attention" : undefined}
                />
              </KpiGrid>
              {report.linked_by_name > 0 && <p>{fill(P.linkedByName, { n: fmtNum(report.linked_by_name) })}</p>}
              {(report.created ?? 0) > 0 && (
                <p className="font-bold" data-testid="sync-created">
                  {fill(report.dry_run ? P.createdDry : P.created, { n: fmtNum(report.created ?? 0) })}
                </p>
              )}
              {(report.created ?? 0) > 0 && (
                <p className="console-meta">
                  {[
                    fill(P.createdByName, { n: fmtNum(report.created_by_name ?? 0) }),
                    (report.created_by_group ?? 0) > 0
                      ? fill(P.createdByGroup, { n: fmtNum(report.created_by_group ?? 0) })
                      : "",
                    fill(P.createdPending, { n: fmtNum(report.created_pending ?? 0) }),
                  ]
                    .filter(Boolean)
                    .join(" · ")}
                </p>
              )}
              {(report.kept_hidden_no_photo ?? 0) > 0 && (
                <p className="text-warning" data-testid="sync-kept-hidden">
                  {fill(P.keptHiddenNoPhoto, { n: fmtNum(report.kept_hidden_no_photo ?? 0) })}
                </p>
              )}
              {(report.zero_price ?? 0) > 0 && (
                <p className="text-warning">{fill(P.zeroPrice, { n: fmtNum(report.zero_price ?? 0) })}</p>
              )}
              {(report.create_skipped ?? 0) > 0 && (
                <p className="text-ink-muted">{fill(P.createSkipped, { n: fmtNum(report.create_skipped ?? 0) })}</p>
              )}
              <p className="text-ink-muted">{fill(P.unchanged, { n: fmtNum(report.unchanged) })}</p>
              {report.unmatched > 0 && (
                <details>
                  <summary className="flex min-h-11 cursor-pointer items-center text-warning">
                    {fill(P.unmatched, { n: fmtNum(report.unmatched) })}
                  </summary>
                  <p className="console-meta mt-1">{P.unmatchedHint}</p>
                  <ul className="mt-1 list-disc ps-5 text-xs">
                    {report.unmatched_sample.map((u) => (
                      <li key={u.row}>{[u.code, u.name].filter(Boolean).join(" · ")}</li>
                    ))}
                  </ul>
                </details>
              )}
              {(report.link_rejected ?? 0) > 0 && (
                <details data-testid="sync-link-rejected">
                  <summary className="flex min-h-11 cursor-pointer items-center text-warning">
                    {fill(P.linkRejected, { n: fmtNum(report.link_rejected ?? 0) })}
                  </summary>
                  <ul className="mt-1 list-disc ps-5 text-xs">
                    {(report.link_rejected_sample ?? []).map((u) => (
                      <li key={`${u.row}-${u.reason}`}>
                        {[u.code, u.name].filter(Boolean).join(" · ")} —{" "}
                        {(P as Record<string, string>)[`why_${u.reason}`] ?? u.reason}
                      </li>
                    ))}
                  </ul>
                </details>
              )}
              {report.skipped.length > 0 && (
                <div className="text-xs">
                  <p className="font-bold">{P.skipped}</p>
                  <ul className="list-disc ps-5">
                    {report.skipped.map((u) => (
                      <li key={u.row}>
                        {[u.code, u.name].filter(Boolean).join(" · ")} —{" "}
                        {(P as Record<string, string>)[`why_${u.reason}`] ?? u.reason}
                      </li>
                    ))}
                  </ul>
                </div>
              )}
              {report.dry_run && (
                <Button className="min-h-11" onClick={() => void run(false)} disabled={busy}>
                  {P.apply}
                </Button>
              )}
            </div>
          )}
        </div>

        {/* **توزيع الأقسام** (٢٠٢٦-١١-٠٧) — كل صنف يروح على قسمو. */}
        {id && <PriceSyncSections storeId={id} refreshKey={applied} />}
      </div>

      {/* **حذفُ المستورد — آخرَ الصفحة وبتأكيد** (٢٠٢٦-١١-٠٨): لمن غلط بالاستيراد أو خلّص تجربة. */}
      {((info?.imported_count ?? 0) > 0 || deletedN !== null) && (
        <section className="panel flex flex-col gap-2 p-4 lg:flex-row lg:items-center lg:justify-between">
          <div className="flex min-w-0 flex-col gap-1">
            <p className="font-bold text-ink">{P.importedTitle}</p>
            <p className="console-meta">
              {deletedN !== null
                ? fill(P.importedDeleted, { n: fmtNum(deletedN) })
                : fill(P.importedHint, { n: fmtNum(info?.imported_count ?? 0) })}
            </p>
          </div>
          {(info?.imported_count ?? 0) > 0 && (
            <Button
              variant="danger"
              className="min-h-11 shrink-0"
              onClick={() => setAskDeleteImported(true)}
              disabled={busy}
            >
              {P.importedDelete}
            </Button>
          )}
          <Confirm
            open={askDeleteImported}
            title={fill(P.importedDeleteQ, { n: fmtNum(info?.imported_count ?? 0) })}
            body={P.importedDeleteBody}
            confirmLabel={P.importedDelete}
            busy={busy}
            onConfirm={() => void deleteImported()}
            onCancel={() => setAskDeleteImported(false)}
          />
        </section>
      )}
    </div>
  );
}
