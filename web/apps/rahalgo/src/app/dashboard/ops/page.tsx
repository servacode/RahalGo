"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **مراقبةُ التشغيل — شاشةُ المراقب**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قراراتُ المالك ٢٠٢٦-١٠-٠٤ — قسمُ «مراقبة التشغيل».)
 *
 * # ما تجيب عنه، من فوق لتحت
 *
 *	١ · سطرُ حالٍ واحد — «كل شي شغّال» أخضر، أو «في مشكلة: …» أحمر وكلُّ جملةٍ تُضغط
 *	٢ · سيرُ الطلبات الآن — عدّادُ كلّ مرحلةٍ بشرط لوحة الطلبات، وجدولُ العالق
 *	    من الدالّة الواحدة نفسِها، والسائقون والطوارئ، و«اعرض على الخريطة»
 *	٣ · حالةُ النظام — كلمةٌ لكلّ جزء، والأرقامُ مطويّةٌ تحت «تفاصيل للمهندس»
 *
 * # ومن يرى ماذا (البند ٢)
 *
 * **سيرُ الطلبات لمن يملك `orders.read`** — موظّفُ العمليّات. **والتفاصيلُ
 * التقنيّةُ لمن يملك `observability.read` وحدَه** — والمحرّكُ يحرس البابين.
 *
 * # والفعلُ الوحيد هنا «أنا عليه»
 *
 * يوقف تكرارَ تذكير الطلب العالق (البند ٤) — **ولا يبدّل في الطلب شيئاً.**
 * ولا زرَّ يمسّ الخادم.
 *
 * # ولا رقمَ يُخترَع
 *
 * **ما لم يُرسله المحرّكُ يُكتب «غير معروف» لا صفراً** — والحبّةُ الخضراءُ لا
 * تبقى بعد فشل آخر قراءة: يُقال «ما قدرنا نوصل للخادم».
 */

import { useCallback, useEffect, useState } from "react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { getMessages, defaultLocale, fmtNum, fmtClockTime, fmtSpan, errorText } from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  ButtonLink,
  Card,
  EmptyState,
  IconDriver,
  IconOrder,
  IconStatus,
  IconWarning,
  ListRow,
  LoadingState,
  OrderRef,
  PageContainer,
  PageHeader,
  StatCard,
  StatGrid,
  useLiveData,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { problemPhrases } from "@/components/admin/OutageBanner";

const m = getMessages(defaultLocale);
const O = m.admin.ops;
const BOARD = m.admin.ordersPage.board.counters as Record<string, string>;
const REASONS = m.admin.ordersPage.alertReasons as Record<string, string>;

type N = number | null;

/** **عقدُ شاشة المراقب** — `GET /admin/ops/monitor`. */
interface Monitor {
  flow: Record<string, N>;
  stuck: {
    order_id: string;
    number: number;
    merchant_name: string;
    reason: string;
    since: string;
    acked_at: string | null;
    reminders: number;
  }[];
  drivers: { on_shift: N; busy: N; free: N };
  emergencies_open: N;
  system: { state: string; parts: Record<string, string>; since: string | null };
  reminder_min: number;
  missing: string[];
}

/** **عقدُ التفاصيل التقنيّة** — `GET /admin/ops/health`، قِيس حيّاً. */
interface OpsHealth {
  status?: string;
  verdict?: { state: string; parts: Record<string, { state: string; reason?: string }> };
  window_minutes?: number;
  pg?: {
    reachable?: boolean;
    ping_ms?: number | null;
    backends?: number | null;
    active?: number | null;
    idle_in_transaction?: number | null;
    longest_tx_seconds?: number | null;
    waiting_locks?: number | null;
  };
  pg_pool?: {
    max?: number;
    acquired?: number;
    idle?: number;
    total?: number;
    empty_acquire_count?: number;
    canceled_acquire_count?: number;
    acquire_duration_ms_total?: number;
  };
  redis?: {
    reachable?: boolean;
    ping_ms?: number | null;
    pool_timeouts?: number;
    total_conns?: number;
    idle_conns?: number;
  };
  runtime?: {
    uptime_seconds?: number;
    goroutines?: number;
    heap_mb?: number;
    sys_mb?: number;
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

/** ترتيبُ عدّادات السير — مفاتيحُها فلاترُ لوحة الطلبات بأعيانها. */
const FLOW = ["awaiting_accept", "awaiting_driver", "on_the_way", "at_door", "no_driver", "stuck", "emergency"] as const;
/** العاجلُ منها — يحمرّ إن لم يكن صفراً. */
const URGENT = new Set<string>(["no_driver", "stuck", "emergency"]);
const PARTS = ["api", "db", "cache", "push", "realtime"] as const;

const unknown = <span className="text-ink-muted">{O.unknown}</span>;

/** num **رقمٌ إن وُجد** — وغيابُه «غير معروف» لا صفر. */
function num(v?: number | null): React.ReactNode {
  return v == null ? unknown : fmtNum(v);
}
function ms(v?: number | null): React.ReactNode {
  return v == null ? unknown : `${fmtNum(v)} ms`;
}
function span(sec?: number | null): React.ReactNode {
  return sec == null ? unknown : fmtSpan(sec);
}

/** **كلمةُ الحال بنصٍّ ورمز** — من لا يرى اللونَ يقرأ الكلمة. */
function StateBadge({ state }: { state?: string }) {
  const st = state === "ok" || state === "slow" || state === "down" ? state : "unknown";
  const variant = { ok: "success", slow: "warning", down: "danger", unknown: "neutral" }[st] as
    | "success"
    | "warning"
    | "danger"
    | "neutral";
  const mark = { ok: "●", slow: "▲", down: "■", unknown: "—" }[st];
  return (
    <Badge variant={variant}>
      <span aria-hidden className="me-1.5">
        {mark}
      </span>
      {O.states[st]}
    </Badge>
  );
}

/** **سطرُ قيمة** — الاسمُ يميناً والرقمُ يساراً. */
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

/** **اسمُ التطبيق ونسختُه بالعربيّة** — `customer:16` ⇒ «الزبون — نسخة 16». */
function appLabel(key: string): string {
  const [app = "", v] = key.split(":");
  const name = (O.appNames as Record<string, string>)[app] ?? app;
  return O.appVersion.replace("{app}", name).replace("{v}", v ? fmtNum(Number(v)) : "");
}

function Counters({ map, labels }: { map?: Record<string, number>; labels: (k: string) => string }) {
  const keys = Object.keys(map ?? {});
  if (keys.length === 0) return <p className="py-2 text-xs text-ink-muted">{O.noneYet}</p>;
  return (
    <>
      {keys.sort().map((k) => (
        <Row key={k} label={labels(k)} value={num(map?.[k])} />
      ))}
    </>
  );
}

const WS_LABELS: Record<string, string> = {
  ws_auth_success: O.wsAuthSuccess,
  ws_auth_invalid_token: O.wsAuthInvalid,
  ws_auth_expired_access: O.wsAuthExpired,
  ws_auth_revoked_session: O.wsAuthRevoked,
  ws_auth_forbidden: O.wsAuthForbidden,
  ws_auth_backend_unavailable: O.wsAuthBackend,
};
const PUSH_LABELS: Record<string, string> = {
  push_attempted: O.pushAttempted,
  push_sent: O.pushSent,
  push_failed: O.pushFailed,
  push_no_device: O.pushNoDevice,
  push_dead_token: O.pushDeadToken,
};

export default function OpsMonitorPage() {
  const router = useRouter();
  const { capabilities, capsLoaded } = useAuth();
  const canFlow = capabilities.includes("orders.read");
  const canEngineer = capabilities.includes("observability.read");
  const canAck = capabilities.includes("orders.intervene");

  const [at, setAt] = useState<Date | null>(null);
  const [error, setError] = useState("");
  const [code, setCode] = useState(0);
  const [now, setNow] = useState(() => Date.now());
  const [acking, setAcking] = useState("");
  const [ackErr, setAckErr] = useState("");

  /** **والعطبُ يُحفظ بنصّه** — ٤٠٣ غيرُ ٤٠١ غيرُ انقطاع، والجسمُ القديمُ يبقى. */
  const guard = useCallback(async <T,>(load: () => Promise<T>): Promise<T> => {
    try {
      const d = await load();
      setError("");
      setCode(0);
      setAt(new Date());
      return d;
    } catch (err) {
      setCode(err instanceof ApiError ? err.status : 0);
      setError(errorText(err));
      throw err;
    }
  }, []);

  const mon = useLiveData<Monitor | null>(
    () => (canFlow ? guard(() => api<Monitor>("/api/v1/admin/ops/monitor")) : Promise.resolve(null)),
    ["order", "alerts", "emergency", "system"],
    [canFlow],
  );
  const eng = useLiveData<OpsHealth | null>(
    () =>
      canEngineer
        ? canFlow
          ? api<OpsHealth>("/api/v1/admin/ops/health")
          : guard(() => api<OpsHealth>("/api/v1/admin/ops/health"))
        : Promise.resolve(null),
    ["system"],
    [canEngineer, canFlow],
  );
  // **وهويّةُ النسخة مرّةً واحدة** — لا تتبدّل بين تحديثين.
  const [ident, setIdent] = useState<Identity | null>(null);
  useEffect(() => {
    if (!canEngineer) return;
    void api<Identity>("/api/v1/public/identity")
      .then(setIdent)
      .catch(() => setIdent(null));
  }, [canEngineer]);

  // **تحديثٌ محافظٌ كلَّ دقيقة** — والدقائقُ في جدول العالق تُعدّ معه.
  const reloadMon = mon.reload;
  const reloadEng = eng.reload;
  useEffect(() => {
    const t = setInterval(() => {
      setNow(Date.now());
      reloadMon();
      reloadEng();
    }, 60_000);
    return () => clearInterval(t);
  }, [reloadMon, reloadEng]);

  const refresh = () => {
    setNow(Date.now());
    reloadMon();
    reloadEng();
  };

  const ackAlert = async (id: string) => {
    setAcking(id);
    setAckErr("");
    try {
      await api(`/api/v1/admin/orders/${id}/alert-ack`, { method: "POST", body: "{}" });
      reloadMon();
    } catch (err) {
      setAckErr(errorText(err) || O.ackFailed);
    } finally {
      setAcking("");
    }
  };

  if (capsLoaded && !canFlow && !capabilities.includes("observability.read")) {
    return (
      <PageContainer>
        <Alert>{O.errForbidden}</Alert>
      </PageContainer>
    );
  }

  const data = mon.data;
  const health = eng.data;
  const pg = health?.pg;
  const pool = health?.pg_pool;
  const rd = health?.redis;
  const rt = health?.runtime;
  const id = ident;
  const loading = (canFlow ? mon.loading && !data : eng.loading && !health) && !error;

  // ── سطرُ الحال ───────────────────────────────────────────────────────
  //
  // **وفشلُ آخر قراءةٍ يُقال ولا يُخفى تحت «كل شي شغّال»** (المشكلة ٥).
  const parts = data?.system.parts ?? plainParts(health?.verdict?.parts);
  const phrases: { text: string; href: string }[] = [];
  if (data && data.stuck.length > 0) {
    phrases.push({ text: O.pStuck.replace("{n}", fmtNum(data.stuck.length)), href: "#stuck" });
  }
  if (data?.emergencies_open) {
    phrases.push({
      text: O.pEmergency.replace("{n}", fmtNum(data.emergencies_open)),
      href: "/dashboard/emergencies",
    });
  }
  for (const p of problemPhrases(parts)) phrases.push({ text: p, href: "#system" });
  const unreachable = !!error && code !== 403 && code !== 401;

  return (
    <PageContainer>
      <PageHeader
        icon={IconStatus}
        title={O.title}
        subtitle={O.subtitle}
        actions={
          <>
            <span className="text-xs text-ink-muted">
              {O.lastUpdate}: {at ? fmtClockTime(at) : loading ? O.loading : "—"}
            </span>
            <Button variant="secondary" onClick={refresh}>
              {O.refresh}
            </Button>
          </>
        }
      />

      {error && (
        <Alert>
          <span className="block font-medium">
            {code === 403 ? O.errForbidden : code === 401 ? O.errUnauthorized : O.errUnavailable}
          </span>
          <span className="mt-1 block text-xs opacity-80">{error}</span>
        </Alert>
      )}

      {loading ? (
        <LoadingState variant="stats" />
      ) : (
        <>
          {/* ── ١ · سطرُ الحال ── */}
          <Card tone={unreachable || phrases.length > 0 ? "danger" : "success"} padding="sm">
            {unreachable ? (
              <p className="flex items-center gap-2 font-bold text-danger">
                <IconWarning size={18} className="shrink-0" />
                {O.pUnreachable}
              </p>
            ) : phrases.length === 0 ? (
              <p className="flex items-center gap-2 font-bold text-success">
                <span aria-hidden>●</span>
                {O.allGood}
              </p>
            ) : (
              <p className="flex flex-wrap items-center gap-x-2 gap-y-1 font-bold text-danger">
                <IconWarning size={18} className="shrink-0" />
                <span>{O.problemPrefix}</span>
                {phrases.map((p, i) => (
                  <span key={p.text} className="flex items-center gap-2">
                    {i > 0 && <span aria-hidden>·</span>}
                    <Link href={p.href} className="underline">
                      {p.text}
                    </Link>
                  </span>
                ))}
              </p>
            )}
            <p className="mt-1 text-xs text-ink-muted">{O.auto}</p>
          </Card>

          {/* ── ٢ · سيرُ الطلبات ── */}
          {data && (
            <>
              <Card title={O.flowTitle} icon={IconOrder}>
                <p className="mb-3 text-xs text-ink-muted">{O.flowHint}</p>
                <StatGrid>
                  {FLOW.map((k) => {
                    const v = data.flow[k];
                    return (
                      <StatCard
                        key={k}
                        label={BOARD[k] ?? k}
                        value={v == null ? O.unknown : v}
                        tone={v == null || v === 0 ? "muted" : URGENT.has(k) ? "danger" : "default"}
                        emphasis={URGENT.has(k) && v != null && v > 0}
                        onClick={() => router.push(`/dashboard/orders?filter=${k}`)}
                      />
                    );
                  })}
                </StatGrid>
              </Card>

              <Card title={O.stuckTitle} icon={IconWarning} tone={data.stuck.length > 0 ? "danger" : "default"}>
                <div id="stuck" className="space-y-2">
                  <p className="text-xs text-ink-muted">
                    {data.reminder_min > 0
                      ? O.reminderEvery.replace("{n}", fmtNum(data.reminder_min))
                      : O.reminderOff}
                  </p>
                  {ackErr && <Alert>{ackErr}</Alert>}
                  {data.stuck.length === 0 ? (
                    <EmptyState title={O.stuckEmpty} />
                  ) : (
                    <ul className="space-y-2">
                      {data.stuck.map((a) => (
                        <ListRow
                          key={a.order_id}
                          leading={<OrderRef number={a.number} />}
                          title={a.merchant_name}
                          subtitle={
                            <>
                              {REASONS[a.reason] ?? a.reason}
                              {m.common.listSeparator}
                              {O.colSince}: {fmtSpan(Math.max(0, (now - Date.parse(a.since)) / 1000))}
                              {m.common.listSeparator}
                              {a.acked_at ? O.acked : O.notAcked}
                              {a.reminders > 0 &&
                                `${m.common.listSeparator}${O.reminded.replace("{n}", fmtNum(a.reminders))}`}
                            </>
                          }
                          trailing={
                            <>
                              <ButtonLink variant="secondary" href={`/dashboard/orders?id=${a.order_id}`}>
                                {O.openOrder}
                              </ButtonLink>
                              {canAck && !a.acked_at && (
                                <Button
                                  variant="danger"
                                  disabled={acking === a.order_id}
                                  onClick={() => void ackAlert(a.order_id)}
                                >
                                  {O.ack}
                                </Button>
                              )}
                            </>
                          }
                        />
                      ))}
                    </ul>
                  )}
                </div>
              </Card>

              <div className="grid min-w-0 grid-cols-1 gap-3 md:grid-cols-2">
                <Card title={O.driversTitle} icon={IconDriver}>
                  <Row label={O.onShift} value={num(data.drivers.on_shift)} />
                  <Row label={O.busy} value={num(data.drivers.busy)} />
                  <Row label={O.free} value={num(data.drivers.free)} />
                  <div className="mt-3">
                    <ButtonLink variant="secondary" href="/dashboard/opsmap">
                      {O.showMap}
                    </ButtonLink>
                  </div>
                </Card>
                <Card
                  title={O.emergenciesTitle}
                  icon={IconWarning}
                  tone={data.emergencies_open ? "danger" : "default"}
                >
                  <p className="figure">{num(data.emergencies_open)}</p>
                  <div className="mt-3">
                    <ButtonLink variant="secondary" href="/dashboard/emergencies">
                      {O.openEmergencies}
                    </ButtonLink>
                  </div>
                </Card>
              </div>
            </>
          )}

          {/* ── ٣ · حالةُ النظام ── */}
          <Card title={O.systemTitle} icon={IconStatus}>
            <div id="system" className="grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-5">
              {PARTS.map((p) => {
                const detail = health?.verdict?.parts?.[p];
                const reason = detail?.reason ? (O.reasons as Record<string, string>)[detail.reason] : "";
                return (
                  <div key={p} className="min-w-0 rounded-control border border-line-soft p-2">
                    <p className="text-xs text-ink-muted">{O.parts[p]}</p>
                    <div className="mt-1">
                      <StateBadge state={unreachable ? undefined : parts?.[p]} />
                    </div>
                    {reason && <p className="mt-1 text-xs text-ink-muted">{reason}</p>}
                  </div>
                );
              })}
            </div>

            {canEngineer && health && (
              <details className="mt-4">
                <summary className="cursor-pointer text-sm font-bold">{O.engineer}</summary>
                <p className="mt-2 text-xs text-ink-muted">
                  {O.sinceRestart.replace(
                    "{t}",
                    rt?.uptime_seconds == null ? O.unknown : fmtSpan(rt.uptime_seconds),
                  )}
                  {health.window_minutes != null &&
                    `${m.common.listSeparator}${O.windowNote.replace("{n}", fmtNum(health.window_minutes))}`}
                </p>
                <div className="mt-3 grid min-w-0 grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3">
                  <Card title={O.secServer} padding="sm">
                    <Row label={O.uptime} value={span(rt?.uptime_seconds)} />
                    <Row label={O.goroutines} value={num(rt?.goroutines)} />
                    <Row label={O.heap} value={rt?.heap_mb == null ? unknown : `${fmtNum(rt.heap_mb)} MB`} />
                    <Row label={O.sys} value={rt?.sys_mb == null ? unknown : `${fmtNum(rt.sys_mb)} MB`} />
                  </Card>
                  <Card title={O.secDb} padding="sm">
                    <Row label={O.ping} value={ms(pg?.ping_ms)} />
                    <Row label={O.backends} value={num(pg?.backends)} />
                    <Row label={O.active} value={num(pg?.active)} />
                    <Row label={O.idleInTx} value={num(pg?.idle_in_transaction)} />
                    <Row label={O.longestTx} value={span(pg?.longest_tx_seconds)} />
                    <Row label={O.waitingLocks} value={num(pg?.waiting_locks)} />
                  </Card>
                  <Card title={O.secPool} padding="sm">
                    <Row label={O.poolMax} value={num(pool?.max)} />
                    <Row label={O.poolTotal} value={num(pool?.total)} />
                    <Row label={O.poolAcquired} value={num(pool?.acquired)} />
                    <Row label={O.poolIdle} value={num(pool?.idle)} />
                    <Row label={O.poolEmpty} value={num(pool?.empty_acquire_count)} />
                    <Row label={O.poolCanceled} value={num(pool?.canceled_acquire_count)} />
                    <Row label={O.poolWaitTotal} value={ms(pool?.acquire_duration_ms_total)} />
                  </Card>
                  <Card title={O.secRedis} padding="sm">
                    <Row label={O.ping} value={ms(rd?.ping_ms)} />
                    <Row label={O.redisTotal} value={num(rd?.total_conns)} />
                    <Row label={O.redisIdle} value={num(rd?.idle_conns)} />
                    <Row label={O.redisTimeouts} value={num(rd?.pool_timeouts)} />
                  </Card>
                  <Card title={O.secWs} padding="sm">
                    <Row label={O.wsSubs} value={num(health.ws?.subscriptions)} />
                    <p className="mb-1 mt-2 text-xs font-bold">{O.wsAuthTitle}</p>
                    <Counters map={health.ws?.auth} labels={(k) => WS_LABELS[k] ?? k} />
                  </Card>
                  <Card title={O.secPush} padding="sm">
                    <Counters map={health.push} labels={(k) => PUSH_LABELS[k] ?? k} />
                  </Card>
                  <Card title={O.secClients} padding="sm">
                    <p className="mb-1 text-xs text-ink-muted">{O.clientsHint}</p>
                    <Counters map={health.clients} labels={appLabel} />
                    <Row label={O.clientsDropped} value={num(health.clients_dropped)} />
                  </Card>
                  <Card title={O.secRelease} padding="sm">
                    <Row label={O.environment} value={id?.environment ?? O.unknown} ltr />
                    <Row label={O.migration} value={id?.migration_version ?? O.unknown} ltr />
                    <Row label={O.commit} value={rt?.source_commit || id?.source_commit || O.unknown} ltr />
                    <Row label={O.buildId} value={rt?.build_id || id?.build_id || O.unknown} ltr />
                  </Card>
                </div>
              </details>
            )}
          </Card>
        </>
      )}
    </PageContainer>
  );
}

/** **كلماتُ الأجزاء من حكم المهندس** — لمن لا يرى سيرَ الطلبات. */
function plainParts(
  p?: Record<string, { state: string; reason?: string }>,
): Record<string, string> | undefined {
  if (!p) return undefined;
  const out: Record<string, string> = {};
  for (const [k, v] of Object.entries(p)) out[k] = v.state;
  return out;
}
