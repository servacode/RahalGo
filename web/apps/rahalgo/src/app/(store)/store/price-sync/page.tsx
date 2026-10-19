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
import {
  getMessages,
  defaultLocale,
  errorText,
  fmtNum,
  fmtDateTime,
} from "@rahalgo/i18n";
import { Alert, Button, Card, LoadingState, Select } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useStore } from "@/lib/store/context";
import { fill } from "@/lib/customer/texts";

const m = getMessages(defaultLocale);
const P = m.store.priceSync;

interface Mapping {
  code: string;
  barcode?: string;
  name?: string;
  price: string;
  qty?: string;
  currency?: "SYP" | "USD";
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
  skipped: Issue[];
  unmatched_sample: Issue[];
}
interface Info {
  mapping: Mapping | null;
  has_token: boolean;
  last_run: { source: string; report: Report; at: string } | null;
  agent_seen_at: string | null;
  agent_version: string | null;
  sync_requested: boolean;
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
  const id = store?.id ?? "";
  const [info, setInfo] = useState<Info | null>(null);
  const [file, setFile] = useState<File | null>(null);
  const [preview, setPreview] = useState<Preview | null>(null);
  const [map, setMap] = useState<Mapping>(EMPTY);
  const [report, setReport] = useState<Report | null>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const [token, setToken] = useState("");
  const [copied, setCopied] = useState(false);

  const load = useCallback(() => {
    if (!id) return;
    api<Info>(`/api/v1/merchant/stores/${id}/price-sync`)
      .then((i) => {
        setInfo(i);
        if (i.mapping) setMap({ ...EMPTY, ...i.mapping });
      })
      .catch((e) => setError(errorText(e)));
  }, [id]);
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

  async function run(dry: boolean) {
    if (!file) return;
    if (!map.price || (!map.code && !map.name)) return setError(P.vMap);
    setBusy(true);
    setError("");
    try {
      setReport(await send(file, map, dry));
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

  async function makeToken() {
    setBusy(true);
    try {
      const r = await api<{ token: string }>(
        `/api/v1/merchant/stores/${id}/price-sync/token`,
        { method: "POST" },
      );
      setToken(r.token);
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

  return (
    <div className="flex flex-col gap-3">
      <header>
        <h2 className="heading-page">{P.title}</h2>
        <p className="text-sm text-ink-muted">{P.hint}</p>
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

      <Card>
        <div className="flex flex-col gap-2 p-4">
          <h3 className="heading-card">{P.tokenTitle}</h3>
          <p className="text-xs text-ink-muted">{P.tokenHint}</p>
          {/* **حالُ الساعي** — شغّالٌ إن سأل في آخر خمس دقائق (يسأل كلَّ دقيقة). */}
          {info?.agent_seen_at ? (
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
          {info?.agent_seen_at &&
            (info.sync_requested ? (
              <p className="text-sm text-success">{P.syncAsked}</p>
            ) : (
              <Button
                className="w-fit"
                onClick={() => void syncNow()}
                disabled={busy}
              >
                {P.syncNow}
              </Button>
            ))}
          {info?.has_token && !token && (
            <p className="text-sm text-success">{P.tokenHas}</p>
          )}
          {token && (
            <div className="flex flex-col gap-1">
              <p className="text-sm">{P.tokenShown}</p>
              <div className="flex items-center gap-2">
                <code
                  dir="ltr"
                  className="min-w-0 flex-1 break-all rounded-control bg-field p-2 text-xs"
                >
                  {token}
                </code>
                <Button
                  variant="secondary"
                  onClick={() => {
                    void navigator.clipboard
                      ?.writeText(token)
                      .then(() => setCopied(true));
                  }}
                >
                  {copied ? P.copied : P.copy}
                </Button>
              </div>
            </div>
          )}
          <Button
            variant="secondary"
            className="w-fit"
            onClick={() => void makeToken()}
            disabled={busy}
          >
            {info?.has_token ? P.tokenNew : P.tokenMake}
          </Button>
        </div>
      </Card>
    </div>
  );
}
