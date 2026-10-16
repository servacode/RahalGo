"use client";

/**
 * **صحّةُ الدفتر** — فحوصُ `moneycheck` نفسُها من المحرّك (`fininv`)، **كلٌّ أخضرُ أو
 * أحمر**، والخرقُ بصفوفه الأولى. لا فحصَ ثانٍ يُكتب للشاشة.
 */

import { useRef, useState } from "react";
import { fmtDateTime } from "@rahalgo/i18n";
import { Alert, Badge, Button, LoadingState, ReloadState, IconShieldCheck, useLiveData } from "@rahalgo/ui";
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
  const { data, error, reload } = useLiveData<Health>(
    () => api(`/api/v1/admin/treasury/health`),
    [],
  );
  if (error && !data) return <ReloadState onRetry={reload} />;
  if (!data) return <LoadingState />;
  const sorted = [...data.checks].sort((a, b) => Number(a.ok) - Number(b.ok));
  return (
    <div className="space-y-3">
      <div className="flex flex-wrap items-center gap-3">
        {data.failing === 0 ? (
          <Alert tone="success">{H.allGreen}</Alert>
        ) : (
          <Alert>
            {H.failing}: <span dir="ltr">{data.failing}</span>
          </Alert>
        )}
        <span className="text-xs text-ink-muted">
          {H.checks}: <span dir="ltr">{data.total}</span> · <span dir="ltr">{fmtDateTime(data.checked_at)}</span>
        </span>
        <Button variant="secondary" onClick={reload}>
          {H.run}
        </Button>
        <CopyReport data={data} />
      </div>
      <ul className="divide-y divide-line surface">
        {sorted.map((c) => (
          <li key={c.id} className="space-y-1 px-3 py-2 text-sm">
            <div className="flex flex-wrap items-center gap-2">
              <IconShieldCheck size={16} className={c.ok ? "text-success" : "text-danger"} />
              <span dir="ltr" className="text-xs text-ink-muted">
                {c.id}
              </span>
              <span className="font-medium">{c.name}</span>
              <Badge variant={c.ok ? "success" : "danger"}>
                {c.error ? H.error : c.ok ? H.ok : `${H.bad} · ${c.count} ${H.rows}`}
              </Badge>
            </div>
            {!c.ok && <p className="text-xs text-ink-muted">{c.why}</p>}
            {c.error && (
              <p dir="ltr" className="text-2xs text-danger">
                {c.error}
              </p>
            )}
            {!c.ok && c.sample && c.sample.length > 0 && (
              <div className="overflow-x-auto">
                <table className="text-2xs" dir="ltr">
                  {c.cols && (
                    <thead>
                      <tr>
                        {c.cols.map((h) => (
                          <th key={h} className="px-1 text-start text-ink-muted">
                            {h}
                          </th>
                        ))}
                      </tr>
                    </thead>
                  )}
                  <tbody>
                    {c.sample.map((row, i) => (
                      <tr key={i}>
                        {row.map((v, j) => (
                          <td key={j} className="px-1">
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
    </div>
  );
}
