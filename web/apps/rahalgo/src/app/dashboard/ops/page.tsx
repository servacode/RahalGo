"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **مراقبةُ التشغيل — شاشةُ صحّةٍ تُقرأ ولا تُفعل**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (دورةُ ٢٠٢٦-٠٩-١٢، بعد إغلاق `70B`.)
 *
 * # ما تجيب عنه
 *
 *	هل النظامُ يعمل؟ · أين المشكلة؟ · القاعدةُ سليمة؟ · الذاكرةُ سليمة؟
 *	الاتّصالُ اللحظيُّ يعمل؟ · ما الإصدارُ العامل؟ · أمؤشّراتُ تدهّور؟
 *
 * # وما ليست
 *
 * **ليست تحليلاً ماليّاً ولا تجاريّاً، ولا مركزَ تحكّم، ولا لوحةَ إعادةِ
 * تشغيل، ولا طرفيّةَ خادم.** **ولا زرَّ يُبدّل شيئاً في الإنتاج** —
 * ويحرسه فحصٌ دائم.
 *
 * # ومصدرُ كلّ رقمٍ هنا
 *
 * **`GET /api/v1/admin/ops/health` وحدَه** — **ولا حسابَ صحّةٍ ثانٍ في
 * الويب.** **ولا حقلٌ يُخترَع**: ما لم يُرسله المحرّكُ يُكتب «غيرُ
 * معروضٍ في هذا العقد» لا صفراً ولا «سليم».
 *
 * **والإصدارُ من `/public/identity`** — بابٌ قائمٌ يحمل البيئةَ ورقمَ
 * الهجرة، **وهما ليسا في عقد الصحّة** (قِيس). **وليس بابَ رصدٍ ثانياً**:
 * هويّةُ إصدارٍ تقرؤها الصفحاتُ كلُّها أصلاً.
 *
 * # ولماذا «غيرُ معروض» لا «سليم»
 *
 * **وحقلٌ غائبٌ ليس حقلاً سليماً** — **ومن كتب «سليم» لغياب خطأٍ كذب
 * على من يقرأ عند العطب.** **فالحكمُ من المحرّك**: `status` و
 * `reachable`، ولا استنتاجَ فوقهما.
 */

import { useCallback, useEffect, useState } from "react";
import { getMessages, defaultLocale, fmtNum, errorText } from "@rahalgo/i18n";
import { Alert, Badge, Button, Card, IconStatus } from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth";

const m = getMessages(defaultLocale);
const O = m.admin.ops;

/** **عقدُ الصحّة كما يردّه المحرّكُ** — قِيس حيّاً، لا مُفترَض. */
interface OpsHealth {
  status?: string;
  pg?: {
    reachable?: boolean;
    ping_ms?: number;
    backends?: number;
    active?: number;
    idle_in_transaction?: number;
    longest_tx_seconds?: number;
    waiting_locks?: number;
  };
  pg_pool?: {
    max?: number;
    acquired?: number;
    idle?: number;
    total?: number;
    constructing?: number;
    empty_acquire_count?: number;
    canceled_acquire_count?: number;
    acquire_duration_ms_total?: number;
  };
  redis?: {
    reachable?: boolean;
    ping_ms?: number;
    pool_hits?: number;
    pool_misses?: number;
    pool_timeouts?: number;
    total_conns?: number;
    idle_conns?: number;
  };
  runtime?: {
    uptime_seconds?: number;
    goroutines?: number;
    heap_mb?: number;
    sys_mb?: number;
    gc_count?: number;
    source_commit?: string;
    build_id?: string;
  };
  ws?: { subscriptions?: number; auth?: Record<string, number> };
  push?: Record<string, number>;
  clients?: Record<string, number>;
  clients_dropped?: number;
}

interface Identity {
  environment?: string;
  source_commit?: string;
  build_id?: string;
  migration_version?: string;
}

type State = "ok" | "degraded" | "down" | "unknown";

/**
 * **حالةٌ بنصٍّ ورمزٍ لا بلونٍ وحدَه** — من لا يرى اللونَ يقرأ الكلمة.
 *
 * **والحبّةُ من `@rahalgo/ui`** — ولا تُبنى باليد.
 */
function StatusPill({ state, label }: { state: State; label?: string }) {
  const text =
    label ??
    { ok: O.statusOk, degraded: O.statusDegraded, down: O.statusDown, unknown: O.statusUnknown }[
      state
    ];
  const mark = { ok: "●", degraded: "▲", down: "■", unknown: "—" }[state];
  const variant = {
    ok: "success",
    degraded: "warning",
    down: "danger",
    unknown: "neutral",
  }[state] as "success" | "warning" | "danger" | "neutral";
  return (
    <Badge variant={variant}>
      <span aria-hidden className="me-1.5">
        {mark}
      </span>
      {text}
    </Badge>
  );
}

/** **سطرُ قيمةٍ** — والرمزُ التقنيُّ يسارٌ والاسمُ يمين. */
function Row({ label, value, ltr }: { label: string; value: React.ReactNode; ltr?: boolean }) {
  return (
    <div className="flex items-baseline justify-between gap-3 border-b border-line-soft py-1.5 last:border-0">
      <span className="shrink-0 text-xs text-ink-muted">{label}</span>
      <span
        className={`min-w-0 break-all text-sm font-medium ${ltr ? "font-mono text-[11px]" : ""}`}
        dir={ltr ? "ltr" : undefined}
      >
        {value}
      </span>
    </div>
  );
}

/** **قسمٌ بعنوانٍ وحالة** — ويلتفّ ولا ينزلق. */
function Section({
  title,
  state,
  children,
}: {
  title: string;
  state?: State;
  children: React.ReactNode;
}) {
  return (
    <Card className="min-w-0">
      <div className="mb-2 flex flex-wrap items-center justify-between gap-2">
        <h2 className="heading-card">{title}</h2>
        {state && <StatusPill state={state} />}
      </div>
      <div className="min-w-0">{children}</div>
    </Card>
  );
}

/** **مدّةٌ بالعربيّة** — ولا «٦٢٦١ ثانية» لمن يسأل «منذ متى؟». */
function human(sec?: number): string {
  if (sec == null) return O.notExposed;
  const d = Math.floor(sec / 86400);
  const h = Math.floor((sec % 86400) / 3600);
  const mi = Math.floor((sec % 3600) / 60);
  const parts: string[] = [];
  if (d > 0) parts.push(`${fmtNum(d)} ${O.days}`);
  if (h > 0) parts.push(`${fmtNum(h)} ${O.hours}`);
  if (parts.length === 0 || (d === 0 && mi > 0)) parts.push(`${fmtNum(mi)} ${O.minutes}`);
  return parts.join(" · ");
}

/** num **رقمٌ إن وُجد** — **وغيابُه يُقال، ولا يُكتب صفراً.** */
function num(v?: number): React.ReactNode {
  return v == null ? <span className="text-ink-muted">{O.notExposed}</span> : fmtNum(v);
}

function ms(v?: number): React.ReactNode {
  return v == null ? <span className="text-ink-muted">{O.notExposed}</span> : `${fmtNum(v)} ms`;
}

/** counters **خريطةُ عدّادات** — بمحدودِ التنوّع كما يرسلها المحرّك. */
function Counters({ map, labels }: { map?: Record<string, number>; labels: Record<string, string> }) {
  const keys = Object.keys(map ?? {});
  if (keys.length === 0) {
    return <p className="py-2 text-xs text-ink-muted">{O.noneYet}</p>;
  }
  return (
    <>
      {keys.sort().map((k) => (
        <Row key={k} label={labels[k] ?? k} value={num(map?.[k])} />
      ))}
    </>
  );
}

export default function OpsHealthPage() {
  const { capabilities, capsLoaded } = useAuth();
  const [health, setHealth] = useState<OpsHealth | null>(null);
  const [ident, setIdent] = useState<Identity | null>(null);
  const [at, setAt] = useState<Date | null>(null);
  const [busy, setBusy] = useState(true);
  const [error, setError] = useState("");
  const [code, setCode] = useState(0);

  const load = useCallback(async () => {
    setBusy(true);
    try {
      const h = await api<OpsHealth>("/api/v1/admin/ops/health");
      setHealth(h);
      setError("");
      setCode(0);
      setAt(new Date());
    } catch (err) {
      // **والجسمُ القديمُ يبقى معروضاً** — **وشاشةٌ تُمحى عند أوّل
      // انقطاعٍ تُفقد من يراقب آخرَ ما رآه.**
      setCode(err instanceof ApiError ? err.status : 0);
      setError(errorText(err));
    } finally {
      setBusy(false);
    }
  }, []);

  // **والهويّةُ مرّةً واحدةً** — لا تتبدّل بين تحديثين.
  useEffect(() => {
    void api<Identity>("/api/v1/public/identity")
      .then(setIdent)
      .catch(() => setIdent(null));
  }, []);

  // **وتحديثٌ محافظٌ كلَّ دقيقة** — **ولا استجدائَ للباب**: ثانيةٌ
  // واحدةٌ تضاعف حملَ قراءةٍ تمسّ القاعدةَ والذاكرةَ في كلّ نداء.
  useEffect(() => {
    void load();
    const t = setInterval(() => void load(), 60_000);
    return () => clearInterval(t);
  }, [load]);

  // ── والقدرةُ تُقاس بعد وصولها لا قبله ──────────────────────────────
  if (capsLoaded && !capabilities.includes("observability.read")) {
    return (
      <div className="p-4">
        <Alert>{O.errForbidden}</Alert>
      </div>
    );
  }

  const pg = health?.pg;
  const pool = health?.pg_pool;
  const rd = health?.redis;
  const rt = health?.runtime;

  const overall: State =
    health?.status === "ok" ? "ok" : health?.status === "degraded" ? "degraded" : "unknown";
  const sub = (reachable?: boolean): State =>
    reachable == null ? "unknown" : reachable ? "ok" : "down";

  return (
    <div className="min-w-0 p-4">
      <div className="mb-3 flex flex-wrap items-center justify-between gap-3">
        <div className="min-w-0">
          <h1 className="heading-page flex items-center gap-2">
            <IconStatus size={18} />
            {O.title}
          </h1>
          <p className="text-xs text-ink-muted">{O.subtitle}</p>
        </div>
        <div className="flex flex-wrap items-center gap-2">
          <span className="text-xs text-ink-muted">
            {O.lastUpdate}:{" "}
            {at ? at.toLocaleTimeString("ar", { hour12: false }) : busy ? O.loading : "—"}
          </span>
          <Button variant="secondary" onClick={() => void load()} disabled={busy}>
            {O.refresh}
          </Button>
        </div>
      </div>
      <p className="mb-3 text-[11px] text-ink-muted">
        {O.readOnly} · {O.auto}
      </p>

      {/* ══════════════════════════════════════════════════════════
          **والعطبُ يُقال بنصّه** — ٤٠٣ غيرُ ٤٠١ غيرُ انقطاع.

          **ولا تُطوى الشاشةُ لتدهّورِ نظامٍ فرعيّ** — تُطوى إن لم
          يصل الجوابُ أصلاً، وحينها يبقى آخرُ ما وصل معروضاً. */}
      {error && (
        <Alert className="mb-3">
          <span className="block font-medium">
            {code === 403
              ? O.errForbidden
              : code === 401
                ? O.errUnauthorized
                : O.errUnavailable}
          </span>
          <span className="mt-1 block text-xs opacity-80">{error}</span>
          {health && <span className="mt-1 block text-xs">{O.errHint}</span>}
        </Alert>
      )}

      <div className="grid min-w-0 grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3">
        <Section title={O.overall} state={overall}>
          <Row label={O.uptime} value={human(rt?.uptime_seconds)} />
          <Row label={O.environment} value={ident?.environment ?? O.notExposed} ltr />
          <Row label={O.migration} value={ident?.migration_version ?? O.notExposed} ltr />
        </Section>

        <Section title={O.secApi}>
          <Row label={O.goroutines} value={num(rt?.goroutines)} />
          <Row label={O.heap} value={rt?.heap_mb == null ? O.notExposed : `${fmtNum(rt.heap_mb)} MB`} />
          <Row label={O.sys} value={rt?.sys_mb == null ? O.notExposed : `${fmtNum(rt.sys_mb)} MB`} />
          <Row label={O.gc} value={num(rt?.gc_count)} />
        </Section>

        <Section title={O.secDb} state={sub(pg?.reachable)}>
          <Row label={O.ping} value={ms(pg?.ping_ms)} />
          <Row label={O.backends} value={num(pg?.backends)} />
          <Row label={O.active} value={num(pg?.active)} />
          <Row label={O.idleInTx} value={num(pg?.idle_in_transaction)} />
          <Row label={O.longestTx} value={human(pg?.longest_tx_seconds)} />
          <Row label={O.waitingLocks} value={num(pg?.waiting_locks)} />
        </Section>

        <Section title={O.secPool}>
          <Row label={O.poolMax} value={num(pool?.max)} />
          <Row label={O.poolTotal} value={num(pool?.total)} />
          <Row label={O.poolAcquired} value={num(pool?.acquired)} />
          <Row label={O.poolIdle} value={num(pool?.idle)} />
          <Row label={O.poolConstructing} value={num(pool?.constructing)} />
          <Row label={O.poolEmpty} value={num(pool?.empty_acquire_count)} />
          <Row label={O.poolCanceled} value={num(pool?.canceled_acquire_count)} />
          <Row label={O.poolWaitTotal} value={ms(pool?.acquire_duration_ms_total)} />
        </Section>

        <Section title={O.secRedis} state={sub(rd?.reachable)}>
          <Row label={O.ping} value={ms(rd?.ping_ms)} />
          <Row label={O.redisTotal} value={num(rd?.total_conns)} />
          <Row label={O.redisIdle} value={num(rd?.idle_conns)} />
          <Row label={O.redisHits} value={num(rd?.pool_hits)} />
          <Row label={O.redisMisses} value={num(rd?.pool_misses)} />
          <Row label={O.redisTimeouts} value={num(rd?.pool_timeouts)} />
        </Section>

        <Section title={O.secWs}>
          <Row label={O.wsSubs} value={num(health?.ws?.subscriptions)} />
          <p className="mt-2 mb-1 text-xs font-bold">{O.wsAuthTitle}</p>
          <Counters
            map={health?.ws?.auth}
            labels={{
              ws_auth_success: O.wsAuthSuccess,
              ws_auth_invalid_token: O.wsAuthInvalid,
              ws_auth_expired_access: O.wsAuthExpired,
              ws_auth_revoked_session: O.wsAuthRevoked,
              ws_auth_forbidden: O.wsAuthForbidden,
              ws_auth_backend_unavailable: O.wsAuthBackend,
            }}
          />
        </Section>

        <Section title={O.secPush}>
          <Counters
            map={health?.push}
            labels={{
              push_attempted: O.pushAttempted,
              push_sent: O.pushSent,
              push_failed: O.pushFailed,
              push_no_device: O.pushNoDevice,
              push_dead_token: O.pushDeadToken,
            }}
          />
        </Section>

        <Section title={O.clients}>
          <Counters map={health?.clients} labels={{}} />
          <Row label={O.clientsDropped} value={num(health?.clients_dropped)} />
        </Section>

        <Section title={O.secRelease}>
          <Row label={O.commit} value={rt?.source_commit ?? ident?.source_commit ?? O.notExposed} ltr />
          <Row label={O.buildId} value={rt?.build_id ?? ident?.build_id ?? O.notExposed} ltr />
        </Section>
      </div>
    </div>
  );
}
