"use client";

/**
 * **صحّةُ الدفتر** — فحوصُ `moneycheck` نفسُها من المحرّك (`fininv`)، **كلٌّ أخضرُ أو
 * أحمر**، والخرقُ بصفوفه الأولى. لا فحصَ ثانٍ يُكتب للشاشة.
 */

import { useRef, useState } from "react";
import { fmtDateTime } from "@rahalgo/i18n";
import { Alert, Badge, Button, LoadingState, ReloadState, IconShieldCheck, IconLoading, useLiveData } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { T } from "./shared";

const H = T.health;

interface Check {
  id: string;
  family: string;
  name: string;
  why: string;
  ok: boolean;
  count: number;
  error?: string;
  cols?: string[];
  sample?: string[][];
}

interface Health {
  checks: Check[];
  failing: number;
  total: number;
  checked_at: string;
  source_commit?: string;
}

/**
 * **تقريرٌ نصّيٌّ للمطوّر** (مراقبةُ المنصّة — الدفعةُ الثالثة ٢٠٢٦-١٠-١٧) — الفحوصُ
 * الساقطةُ بمعرّفها واسمها وعددِ صفوفها وأوّلِ صفوفها JSON، ووقتُ الفحص ونسخةُ
 * الخادم. **ويُنسخ ولو كان كلُّه أخضر** — فالتقريرُ يقول ذلك صراحةً.
 */
function healthReport(data: Health): string {
  const R = H.report;
  const lines = [
    R.title,
    `${R.at}: ${data.checked_at}`,
    `${R.commit}: ${data.source_commit || R.noCommit}`,
    `${R.summary}: ${data.total} · ${R.failing}: ${data.failing}`,
    "",
  ];
  const bad = data.checks.filter((c) => !c.ok);
  if (bad.length === 0) lines.push(R.allGreen);
  for (const c of bad) {
    lines.push(`[${c.id}] ${c.name}`, `${R.count}: ${c.count}`);
    if (c.error) lines.push(`${R.error}: ${c.error}`);
    if (c.sample && c.sample.length > 0) {
      lines.push(`${R.rows}:`);
      for (const row of c.sample) {
        const obj = c.cols ? Object.fromEntries(c.cols.map((k, i) => [k, row[i]])) : row;
        lines.push(JSON.stringify(obj));
      }
    }
    lines.push("");
  }
  return lines.join("\n").trimEnd();
}

/** **زرُّ «انسخ للمطوّر»** — الحافظةُ أوّلاً، وإلّا مربّعٌ مخفيٌّ يُحدَّد ويُنسخ، وإلّا يظهر المربّعُ ليُنسخ باليد. */
function CopyReport({ data }: { data: Health }) {
  const box = useRef<HTMLTextAreaElement>(null);
  const [state, setState] = useState<"idle" | "copied" | "failed">("idle");
  const text = healthReport(data);
  const fallback = () => {
    const el = box.current;
    if (!el) return false;
    el.value = text;
    el.select();
    try {
      return document.execCommand("copy");
    } catch {
      return false;
    }
  };
  const onCopy = () => {
    const done = () => setState("copied");
    const fail = () => setState(fallback() ? "copied" : "failed");
    if (navigator.clipboard?.writeText) {
      navigator.clipboard.writeText(text).then(done, fail);
    } else {
      fail();
    }
  };
  return (
    <>
      <Button variant="secondary" onClick={onCopy}>
        {state === "copied" ? H.copied : H.copy}
      </Button>
      {state === "failed" && <span className="text-xs text-danger">{H.copyFailed}</span>}
      <textarea
        ref={box}
        readOnly
        dir="ltr"
        aria-hidden={state !== "failed"}
        tabIndex={-1}
        defaultValue={text}
        className={state === "failed" ? "h-40 w-full rounded-control border border-line bg-field p-2 font-mono text-2xs text-ink" : "sr-only"}
      />
    </>
  );
}

export function HealthTab() {
  const { data, error, reload, setData } = useLiveData<Health>(
    () => api(`/api/v1/admin/treasury/health`),
    [],
  );
  // **إعادةُ الفحص تُرى** (قرارُ المالك ٢٠٢٦-١٠-٢٦) — كلُّ فحصٍ يصير «بالانتظار» ثمّ
  // يتحوّل صحّاً أو خطأً. **ونصفُ ثانيةٍ على الأقلّ** كي لا يومض الانتظارُ فلا يُرى.
  const [running, setRunning] = useState(false);
  const [runFailed, setRunFailed] = useState(false);
  const rerun = async () => {
    setRunning(true);
    setRunFailed(false);
    const started = performance.now();
    try {
      const d = await api<Health>(`/api/v1/admin/treasury/health`);
      const wait = 700 - (performance.now() - started);
      if (wait > 0) await new Promise((r) => setTimeout(r, wait));
      setData(d);
    } catch {
      setRunFailed(true);
    } finally {
      setRunning(false);
    }
  };
  if (error && !data) return <ReloadState onRetry={reload} />;
  if (!data) return <LoadingState />;
  // **الفحوصُ بمواضيعها** (قرارُ المالك ٢٠٢٦-١٠-٢٦: «تصير واضحة أكثر ومفهومة») — عشراتُ الأسطر
  // تُقرأ بمواضيعَ بأسماءٍ مفهومة، **والموضوعُ الذي فيه خرقٌ يُفتح أوّلاً.**
  const groups = new Map<string, Check[]>();
  for (const c of data.checks) {
    const g = groups.get(c.family) ?? [];
    g.push(c);
    groups.set(c.family, g);
  }
  const fam = H.families as Record<string, { t: string; d: string }>;
  const ordered = [...groups.entries()].sort(
    (x, y) => Number(x[1].every((c) => c.ok)) - Number(y[1].every((c) => c.ok)) || x[0].localeCompare(y[0]),
  );
  const okAll = data.failing === 0;
  return (
    <div className="space-y-4">
      {/* **الحالُ بنظرة** — دائرةٌ خضراءُ أو حمراء، وجملةٌ تقول ما يعنيه الفحص. */}
      <section className={`surface flex flex-wrap items-center gap-4 p-5 ${running || okAll ? "" : "border-danger"}`}>
        <span
          className={`inline-flex h-16 w-16 shrink-0 items-center justify-center rounded-full ${
            running ? "bg-field text-ink-muted" : okAll ? "bg-success-solid text-on-solid" : "bg-danger-solid text-on-solid"
          }`}
        >
          {running ? <IconLoading size={30} className="animate-spin" /> : <IconShieldCheck size={30} />}
        </span>
        <div className="min-w-0 flex-1 space-y-1">
          <h2 className={`heading-section ${running ? "text-ink-muted" : okAll ? "text-success" : "text-danger"}`}>
            {running ? H.running : okAll ? H.allGreen : `${H.failing}: ${data.failing}`}
          </h2>
          {runFailed && <p className="text-sm text-danger">{H.runFailed}</p>}
          <p className="text-sm text-ink-muted">{H.what}</p>
          <p className="text-xs text-ink-muted">
            {H.checks}: <span dir="ltr">{data.total}</span> · {H.passed}:{" "}
            <span dir="ltr">{data.total - data.failing}</span> · {H.lastRun}{" "}
            <span dir="ltr">{fmtDateTime(data.checked_at)}</span>
          </p>
        </div>
        <div className="flex shrink-0 flex-wrap items-center gap-2">
          <Button onClick={rerun} disabled={running}>
            {running ? H.running : H.run}
          </Button>
          <CopyReport data={data} />
        </div>
      </section>

      <div className="grid grid-cols-1 gap-3 lg:grid-cols-2">
        {ordered.map(([key, list]) => {
          const bad = list.filter((c) => !c.ok).length;
          const info = fam[key];
          return (
            <details key={key} open={bad > 0} className="surface group overflow-hidden">
              <summary className="flex cursor-pointer list-none items-center gap-3 p-4">
                <span
                  className={`inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full font-bold ${
                    running ? "bg-field text-ink-muted" : bad > 0 ? "bg-danger-solid text-on-solid" : "bg-success-solid text-on-solid"
                  }`}
                >
                  {running ? <IconLoading size={18} className="animate-spin" /> : bad > 0 ? "!" : "\u2713"}
                </span>
                <span className="min-w-0 flex-1">
                  <span className="block font-bold">{info?.t ?? key}</span>
                  <span className="block truncate text-xs text-ink-muted">{info?.d ?? ""}</span>
                </span>
                <span
                  className={`shrink-0 rounded-full px-2.5 py-0.5 text-xs font-bold tabular-nums ${
                    !running && bad > 0 ? "bg-danger-solid text-on-solid" : "bg-field text-ink-muted"
                  }`}
                  dir="ltr"
                >
                  {running ? "\u2026" : `${list.length - bad}/${list.length}`}
                </span>
                <span className="shrink-0 text-xs text-ink-muted transition-transform group-open:rotate-180">{"▾"}</span>
              </summary>
              <ul className="space-y-1.5 border-t border-line-soft p-3">
                {list.map((c) => (
                  <li key={c.id} className={`rounded-control px-3 py-2 text-sm ${running || c.ok ? "bg-field" : "bg-danger-tint"}`}>
                    <div className="flex items-center gap-2">
                      {running ? (
                        <IconLoading size={12} className="animate-spin text-ink-muted" />
                      ) : (
                        <span className={`text-xs ${c.ok ? "text-success" : "text-danger"}`}>{c.ok ? "\u2713" : "\u2717"}</span>
                      )}
                      <span className={`min-w-0 flex-1 ${running ? "text-ink-muted" : ""}`}>{c.name}</span>
                      {running && <span className="text-2xs text-ink-muted">{H.waiting}</span>}
                      {!running && !c.ok && <Badge variant="danger">{c.error ? H.error : `${c.count} ${H.rows}`}</Badge>}
                    </div>
                    {!running && !c.ok && <p className="mt-1 text-xs text-ink-muted">{c.why}</p>}
                    {!running && c.error && (
                      <p dir="ltr" className="mt-1 text-2xs text-danger">
                        {c.error}
                      </p>
                    )}
                    {!running && !c.ok && c.sample && c.sample.length > 0 && (
                      <div className="mt-1 overflow-x-auto">
                        <table className="text-2xs" dir="ltr">
                          {c.cols && (
                            <thead>
                              <tr>
                                {c.cols.map((k) => (
                                  <th key={k} className="px-1.5 text-start text-ink-muted">
                                    {k}
                                  </th>
                                ))}
                              </tr>
                            </thead>
                          )}
                          <tbody>
                            {c.sample.map((row, i) => (
                              <tr key={i}>
                                {row.map((v, j) => (
                                  <td key={j} className="px-1.5">
                                    {v}
                                  </td>
                                ))}
                              </tr>
                            ))}
                          </tbody>
                        </table>
                      </div>
                    )}
                  </li>
                ))}
              </ul>
            </details>
          );
        })}
      </div>
    </div>
  );
}
