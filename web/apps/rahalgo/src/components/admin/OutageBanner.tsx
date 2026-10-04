"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **شريطُ تعطّل الخادم — أعلى كلّ صفحةٍ في اللوحة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-١٠-٠٤ — قسمُ «مراقبة التشغيل»، البند ٣.)
 *
 * **يظهر لكلّ موظّف** حين يقول المحرّكُ إنّ جزءاً منه بطيءٌ أو واقف، **أو حين لا
 * يصل الجوابُ أصلاً.** ويختفي وحدَه حين يعود كلُّ شيء — البثُّ «system» يعيد
 * القراءة، **ونبضةٌ كلَّ دقيقةٍ تكشف الخادمَ الذي لا يبثّ لأنّه واقف.**
 *
 * **كلماتٌ بلا أرقام** — والتفاصيلُ في «مراقبة التشغيل» لمن يملكها.
 */

import { useEffect } from "react";
import Link from "next/link";
import { getMessages, defaultLocale, fmtSpan } from "@rahalgo/i18n";
import { useLiveData, IconWarning } from "@rahalgo/ui";
import { api } from "@/lib/api";

const m = getMessages(defaultLocale);
const B = m.admin.outageBanner;
const O = m.admin.ops;

type Part = keyof typeof O.problems;

interface SystemStatus {
  state: string;
  parts: Record<string, string>;
  since: string | null;
}

/** **جملةُ المشكلة لكلّ جزءٍ ليس سليماً** — بترتيب العرض. */
export function problemPhrases(parts: Record<string, string> | undefined): string[] {
  const out: string[] = [];
  for (const k of Object.keys(O.problems) as Part[]) {
    const st = parts?.[k];
    if (st === "slow" || st === "down") out.push(O.problems[k][st]);
  }
  return out;
}

export function OutageBanner({ canOpen }: { canOpen: boolean }) {
  const status = useLiveData<SystemStatus>(
    () => api<SystemStatus>("/api/v1/admin/ops/status"),
    ["system"],
  );
  const reload = status.reload;
  useEffect(() => {
    const t = setInterval(() => reload(), 60_000);
    return () => clearInterval(t);
  }, [reload]);

  const s = status.data;
  const unreachable = status.error;
  const phrases = s && s.state !== "ok" ? problemPhrases(s.parts) : [];
  if (!unreachable && phrases.length === 0) return null;

  const since =
    s?.since && !unreachable
      ? B.since.replace("{t}", fmtSpan(Math.max(0, (Date.now() - Date.parse(s.since)) / 1000)))
      : "";
  return (
    <div
      role="alert"
      className="mb-4 rounded-card border border-danger-edge bg-danger-tint p-3 text-sm text-danger"
    >
      <p className="flex flex-wrap items-center gap-2 font-bold">
        <IconWarning size={18} className="shrink-0" />
        <span className="min-w-0 flex-1">
          {B.title}
          {since && `${m.common.listSeparator}${since}`}
        </span>
        {canOpen && (
          <Link href="/dashboard/ops" className="text-xs font-normal underline">
            {B.open}
          </Link>
        )}
      </p>
      <p className="mt-1 text-ink">
        {unreachable ? B.unreachable : phrases.join(m.common.listSeparator)}
      </p>
    </div>
  );
}
