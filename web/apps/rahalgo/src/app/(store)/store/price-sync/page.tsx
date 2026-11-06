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
import { Alert, Button, Card, Checkbox, LoadingState, Select } from "@rahalgo/ui";
import { api, apiFile } from "@/lib/api";
import { useStore } from "@/lib/store/context";
import { fill } from "@/lib/customer/texts";
import { PageHeader } from "@/components/store/SetupParts";

const m = getMessages(defaultLocale);
const P = m.store.priceSync;

interface Mapping {
  code: string;
  barcode?: string;
  name?: string;
  price: string;
  qty?: string;
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
  const [file, setFile] = useState<File | null>(null);
  const [preview, setPreview] = useState<Preview | null>(null);
  const [map, setMap] = useState<Mapping>(EMPTY);
  const [report, setReport] = useState<Report | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");

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

  /** **«إيقاف البرنامج»** — يزيل نفسَه من الإقلاع في سؤاله التالي. */
  async function stopAgent() {
    setBusy(true);
    try {
      await api(`/api/v1/merchant/stores/${id}/price-sync/stop`, {
        method: "POST",
      });
      load();
    } catch (e) {
      setError(errorText(e));
    } finally {
      setBusy(false);
    }
  }

  if (!info && !error) return <LoadingState />;

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

  return (
    <div className="flex flex-col gap-3">
      <header>
        <PageHeader title={P.title} href="/store/menu" />
        {info?.last_run && (
          <p className="mt-1 text-xs text-ink-muted">
            {fill(P.last, { at: fmtDateTime(info.last_run.at) })}
          </p>
        )}
      </header>

      <label className="surface flex cursor-pointer flex-col items-center gap-2 rounded-card border border-dashed border-accent p-6 text-center">
        <span className="font-bold text-accent-text">
          {busy ? P.reading : P.pick}
        </span>
        {file && <span className="text-xs text-ink-muted">{file.name}</span>}
        <input
          type="file"
          accept=".xlsx,.xlsm,.csv,.txt"
          className="hidden"
          disabled={busy}
          onChange={(e) => {
            const f = e.target.files?.[0];
            if (f) void pick(f);
            e.target.value = "";
          }}
        />
      </label>

      {error && <Alert>{error}</Alert>}

      {/* **أنشئ المواد غير الموجودة أصنافاً مخفية** (٢٠٢٦-١١-٠٤) — يُبدَّل فيُعاد «جرّب» ليُرى العدد. */}
      {file && (
        <Card>
          <div className="flex flex-col gap-1 p-4">
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
            <p className="text-xs text-ink-muted">{P.createMissingHint}</p>
          </div>
        </Card>
      )}

      {preview && (
        <Card>
          <div className="flex flex-col gap-3 p-4">
            <h3 className="heading-card">{P.columnsTitle}</h3>
            <p className="text-xs text-ink-muted">
              {P.columnsHint} · {fill(P.rows, { n: fmtNum(preview.rows) })}
            </p>
            <div className="grid grid-cols-1 gap-3 sm:grid-cols-2">
              {col("code", P.colCode)}
              {col("price", P.colPrice, true)}
              {col("qty", P.colQty)}
              {col("name", P.colName)}
              {col("barcode", P.colBarcode)}
              <Select
                id="col-currency"
                label={P.currency}
                value={map.currency ?? "SYP"}
                onChange={(e) =>
                  setMap({ ...map, currency: e.target.value as "SYP" | "USD" })
                }
              >
                <option value="SYP">{P.syp}</option>
                <option value="USD">{P.usd}</option>
              </Select>
            </div>
            <div className="overflow-x-auto">
              <table className="w-full text-xs">
                <thead>
                  <tr>
                    {preview.headers.map((h, i) => (
                      <th
                        key={i}
                        className="border-b border-line-soft p-1 text-start"
                      >
                        {h}
                      </th>
                    ))}
                  </tr>
                </thead>
                <tbody>
                  {preview.sample.map((r, i) => (
                    <tr key={i}>
                      {preview.headers.map((_, j) => (
                        <td key={j} className="border-b border-line-soft p-1">
                          {r[j] ?? ""}
                        </td>
                      ))}
                    </tr>
                  ))}
                </tbody>
              </table>
            </div>
            <Button onClick={() => void run(true)} disabled={busy}>
              {P.try}
            </Button>
          </div>
        </Card>
      )}

      {report && (
        <Card tone={report.dry_run ? "default" : "success"}>
          <div className="flex flex-col gap-2 p-4 text-sm">
            <h3 className="heading-card">
              {report.dry_run ? P.dryTitle : P.doneTitle}
            </h3>
            <p>{fill(P.matched, { n: fmtNum(report.matched) })}</p>
            <p className="font-bold">
              {fill(P.priceChanged, { n: fmtNum(report.price_changed) })}
            </p>
            <p>{fill(P.madeAvailable, { n: fmtNum(report.made_available) })}</p>
            <p>
              {fill(P.madeUnavailable, { n: fmtNum(report.made_unavailable) })}
            </p>
            {report.linked_by_name > 0 && (
              <p>
                {fill(P.linkedByName, { n: fmtNum(report.linked_by_name) })}
              </p>
            )}
            {(report.created ?? 0) > 0 && (
              <p className="font-bold" data-testid="sync-created">
                {fill(report.dry_run ? P.createdDry : P.created, {
                  n: fmtNum(report.created ?? 0),
                })}
              </p>
            )}
            {(report.zero_price ?? 0) > 0 && (
              <p className="text-warning">
                {fill(P.zeroPrice, { n: fmtNum(report.zero_price ?? 0) })}
              </p>
            )}
            {(report.create_skipped ?? 0) > 0 && (
              <p className="text-ink-muted">
                {fill(P.createSkipped, { n: fmtNum(report.create_skipped ?? 0) })}
              </p>
            )}
            <p className="text-ink-muted">
              {fill(P.unchanged, { n: fmtNum(report.unchanged) })}
            </p>
            {report.unmatched > 0 && (
              <details>
                <summary className="cursor-pointer text-warning">
                  {fill(P.unmatched, { n: fmtNum(report.unmatched) })}
                </summary>
                <p className="mt-1 text-xs text-ink-muted">{P.unmatchedHint}</p>
                <ul className="mt-1 list-disc ps-5 text-xs">
                  {report.unmatched_sample.map((u) => (
                    <li key={u.row}>
                      {[u.code, u.name].filter(Boolean).join(" · ")}
                    </li>
                  ))}
                </ul>
              </details>
            )}
            {(report.link_rejected ?? 0) > 0 && (
              <details data-testid="sync-link-rejected">
                <summary className="cursor-pointer text-warning">
                  {fill(P.linkRejected, { n: fmtNum(report.link_rejected ?? 0) })}
                </summary>
                <ul className="mt-1 list-disc ps-5 text-xs">
                  {(report.link_rejected_sample ?? []).map((u) => (
                    <li key={`${u.row}-${u.reason}`}>
                      {[u.code, u.name].filter(Boolean).join(" · ")} —{" "}
                      {(P as Record<string, string>)[`why_${u.reason}`] ??
                        u.reason}
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
                      {(P as Record<string, string>)[`why_${u.reason}`] ??
                        u.reason}
                    </li>
                  ))}
                </ul>
              </div>
            )}
            {report.dry_run && (
              <Button onClick={() => void run(false)} disabled={busy}>
                {P.apply}
              </Button>
            )}
          </div>
        </Card>
      )}

      {/* **برنامجُ المزامنة — ملفٌّ واحد** (قرارُ المالك ٢٠٢٦-١٠-١٩): يُحمَّل وفيه رمزُ المتجر،
          دبل كليك على كمبيوتر الأمين وخلص. **وحالُه وخطؤه هنا** لا على كمبيوتر المحاسب. */}
      <Card>
        <div className="flex flex-col gap-2 p-4">
          <h3 className="heading-card">{P.agentTitle}</h3>
          <p className="text-sm">{P.agentSteps}</p>
          <Button
            className="w-fit"
            onClick={() => void download()}
            disabled={busy}
          >
            {busy ? P.agentDownloading : P.agentDownload}
          </Button>
          <p className="text-xs text-ink-muted">{P.agentRedownload}</p>
          {info?.stopped ? (
            <p className="text-sm text-ink-muted">{P.agentStopped}</p>
          ) : info?.agent_seen_at ? (
            <p
              className={`text-sm ${Date.now() - new Date(info.agent_seen_at).getTime() < 5 * 60_000 ? "text-success" : "text-danger"}`}
            >
              {fill(
                Date.now() - new Date(info.agent_seen_at).getTime() < 5 * 60_000
                  ? P.agentAlive
                  : P.agentDead,
                {
                  ago: fmtDateTime(info.agent_seen_at),
                },
              )}
              {info.agent_version
                ? ` · ${fill(P.agentVersion, { v: info.agent_version })}`
                : ""}
            </p>
          ) : (
            info?.has_token && (
              <p className="text-sm text-ink-muted">{P.agentNever}</p>
            )
          )}
          {info?.agent_error && !info.stopped && (
            <Alert tone="warning">
              {fill(P.agentError, { e: info.agent_error })}
            </Alert>
          )}
          {info?.has_token && !info.stopped && (
            <Select
              id="sync-interval"
              label={P.interval}
              value={String(info.interval_minutes ?? 1)}
              onChange={(e) => void saveInterval(Number(e.target.value))}
              disabled={busy}
            >
              {[1, 5, 10, 15, 30, 60].map((n) => (
                <option key={n} value={n}>
                  {n === 1
                    ? P.every1
                    : n === 60
                      ? P.every60
                      : fill(P.everyN, { n: fmtNum(n) })}
                </option>
              ))}
            </Select>
          )}
          {info?.agent_seen_at && !info.stopped && (
            <div className="flex flex-wrap gap-2">
              {info.sync_requested ? (
                <p className="text-sm text-success">{P.syncAsked}</p>
              ) : (
                <Button onClick={() => void syncNow()} disabled={busy}>
                  {P.syncNow}
                </Button>
              )}
              <Button
                variant="ghost"
                className="text-danger"
                onClick={() => void stopAgent()}
                disabled={busy}
              >
                {P.agentStop}
              </Button>
            </div>
          )}
        </div>
      </Card>
    </div>
  );
}
