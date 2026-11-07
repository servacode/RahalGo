"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **حالةُ برنامج الأمين — مصدرٌ واحدٌ لصفحة المزامنة وبطاقة الرئيسيّة**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (ملاحظةُ المالك ٢٠٢٦-١١-٠٧: «قسم برنامج الأمين والمزامنة لازم يكون بالأعلى مو بالأسفل».)
 *
 * **الحالاتُ الستّ** كما هي في المحرّك ولا شيءَ جديد: محذوف · موقوفٌ مؤقّتاً · متّصلٌ ولا
 * يزامن (أحمر + السبب) · يعمل · غير متّصل · لم يتّصل بعد. **والحسابُ هنا وحدَه** فلا
 * تُكتب الشروطُ في مكانين فتفترق (الرئيسيّةُ تقول «يعمل» والصفحةُ تقول غير ذلك).
 * **قراءةٌ فقط** — لا يغيّر شيئاً في المزامنة.
 */

import { useEffect, useState } from "react";
import Link from "next/link";
import { getMessages, defaultLocale, fmtDateTime } from "@rahalgo/i18n";
import { ButtonLink, IconRotate, Pill } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useStore } from "@/lib/store/context";
import { fill } from "@/lib/customer/texts";

const m = getMessages(defaultLocale);
const P = m.store.priceSync;
const K = m.store.console;

/** **ما يلزم من ردّ المحرّك لتحديد الحالة** — والصفحةُ تمدّه بما تحتاجه. */
export interface SyncInfoBase {
  has_token: boolean;
  last_run: { source: string; at: string } | null;
  agent_seen_at: string | null;
  agent_version: string | null;
  agent_error: string | null;
  stopped: boolean;
  paused: boolean;
}

export type AgentKey = "stopped" | "paused" | "error" | "alive" | "dead" | "never" | "unlinked";

export interface AgentState {
  key: AgentKey;
  /** نبرةُ الشارة — وبجانبها نصٌّ دائماً، فلا يعتمد المعنى على اللون. */
  badge: "success" | "danger" | "warning" | "neutral";
  tone: "good" | "danger" | "attention" | undefined;
  label: string;
  /** سطرٌ يشرح الحالةَ (آخر اتصال · السبب …). */
  detail: string;
  /** سببُ الخطأ حين يكون البرنامجُ متّصلاً ولا يزامن. */
  why?: string;
}

/** **خمسُ دقائق** — بعدها يُعدّ البرنامجُ غيرَ متّصل (كما كانت الصفحة). */
const ALIVE_MS = 5 * 60_000;

export function agentState(info: SyncInfoBase | null, now: number = Date.now()): AgentState {
  if (!info || !info.has_token) {
    return { key: "unlinked", badge: "neutral", tone: undefined, label: K.stUnlinked, detail: K.stUnlinkedHint };
  }
  if (info.stopped) {
    return { key: "stopped", badge: "neutral", tone: undefined, label: K.stStopped, detail: P.agentStopped };
  }
  if (info.paused) {
    return { key: "paused", badge: "warning", tone: "attention", label: K.stPaused, detail: P.agentPausedHint };
  }
  if (info.agent_seen_at && info.agent_error) {
    return {
      key: "error",
      badge: "danger",
      tone: "danger",
      label: K.stError,
      detail: fill(P.agentNotSyncing, { ago: fmtDateTime(info.agent_seen_at) }),
      why: fill(P.agentNotSyncingWhy, { e: info.agent_error }),
    };
  }
  if (info.agent_seen_at) {
    const alive = now - new Date(info.agent_seen_at).getTime() < ALIVE_MS;
    const ver = info.agent_version ? ` · ${fill(P.agentVersion, { v: info.agent_version })}` : "";
    return {
      key: alive ? "alive" : "dead",
      badge: alive ? "success" : "danger",
      tone: alive ? "good" : "danger",
      label: alive ? K.stAlive : K.stDead,
      detail: fill(alive ? P.agentAlive : P.agentDead, { ago: fmtDateTime(info.agent_seen_at) }) + ver,
    };
  }
  return { key: "never", badge: "neutral", tone: undefined, label: K.stNever, detail: P.agentNever };
}

/** **آخرُ مزامنةٍ فعليّةٍ من البرنامج** (لا من ملفٍّ رُفع باليد) أو الفارغ. */
export function lastRealSync(info: SyncInfoBase | null): string {
  if (!info?.agent_seen_at || info.stopped) return "";
  return info.last_run && info.last_run.source !== "file"
    ? fill(P.agentLastSync, { at: fmtDateTime(info.last_run.at) })
    : P.agentNoSyncYet;
}

/**
 * **بطاقةُ الرئيسيّة** — حالةُ البرنامج في صفٍّ واحدٍ قرب الأعلى، وتفتح صفحةَ المزامنة.
 * لا تظهر إلّا لمتجرٍ يستعمل الأمين، وتبقى صامتةً إن تعذّرت القراءة (لا تُسقط الرئيسيّة).
 */
export function AmeenStatusTile({ className = "" }: { className?: string }) {
  const { store } = useStore();
  const id = store?.id ?? "";
  const on = store?.uses_ameen === true;
  const [info, setInfo] = useState<SyncInfoBase | null | undefined>(undefined);

  useEffect(() => {
    if (!id || !on) return;
    let live = true;
    const read = () =>
      api<SyncInfoBase>(`/api/v1/merchant/stores/${id}/price-sync`)
        .then((i) => live && setInfo(i))
        .catch(() => live && setInfo((old) => (old === undefined ? null : old)));
    void read();
    // **كلَّ دقيقة** — تصل الحالةُ من الساعي في سؤاله، والبطاقةُ لا تُترك على «يعمل» وقد انقطع.
    const t = setInterval(() => void read(), 60_000);
    return () => {
      live = false;
      clearInterval(t);
    };
  }, [id, on]);

  if (!on || info === undefined) return null;
  const st = agentState(info);
  const last = lastRealSync(info);
  return (
    <section
      className={`panel flex flex-wrap items-center gap-3 p-4 lg:px-5 ${className}`}
      aria-label={K.ameenTile}
      data-testid="ameen-status-tile"
      data-state={st.key}
    >
      <span className="icon-chip size-11" data-tone={st.tone} aria-hidden>
        <IconRotate size={22} />
      </span>
      <div className="flex min-w-0 flex-1 flex-col gap-0.5">
        <span className="flex flex-wrap items-center gap-2">
          <h2 className="console-h2">{K.ameenTile}</h2>
          <Pill tone={st.tone}>{st.label}</Pill>
        </span>
        <p className={`console-meta line-clamp-2 lg:line-clamp-none ${st.key === "error" ? "!text-danger" : ""}`}>
          {st.key === "error" && st.why ? `${st.detail} — ${st.why}` : st.detail}
        </p>
        {last && st.key !== "stopped" && <p className="console-meta hidden sm:block">{last}</p>}
      </div>
      <ButtonLink Link={Link} href="/store/price-sync" variant="secondary" className="min-h-11 w-full shrink-0 sm:w-auto">
        {K.ameenOpen}
      </ButtonLink>
    </section>
  );
}
