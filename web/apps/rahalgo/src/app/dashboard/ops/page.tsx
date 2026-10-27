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
 *	    (تصميمٌ احترافيّ (المالك ٢٠٢٦-١٠-٢٦): رأسٌ موحّدٌ ودائرةُ حالٍ كبيرة، والأرقامُ
 *	    بطاقاتٌ ملوّنة، والعالقُ بطاقاتٌ لا جدول.)
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
import { getMessages, defaultLocale, fmtNum, fmtClockTime, fmtDateTime, fmtSpan, errorText } from "@rahalgo/i18n";
import {
  Alert,
  Badge,
  Button,
  ButtonLink,
  IconBell,
  IconCheck,
  IconDriver,
  IconHourglass,
  IconLocation,
  IconMoto,
  IconSettings,
  IconShieldCheck,
  IconShieldX,
  IconStatus,
  IconStore,
  IconSuccess,
  IconWarning,
  LoadingState,
  OrderRef,
  PageContainer,
  useLiveData,
} from "@rahalgo/ui";
import { api, ApiError } from "@/lib/api";
import { useAuth } from "@/lib/auth";
import { problemPhrases } from "@/components/admin/OutageBanner";
import { PageHero } from "@/components/admin/PageHero";
import { Panel, Tile, type TileTone } from "@/components/admin/DashParts";

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

/** **أخطاءُ الخادم مجمَّعةً** — `GET /admin/monitoring/errors` (مراقبةُ المنصّة ٢٠٢٦-١٠-٠٩). */
interface ServerErrors {
  total: number;
  last_hour: number;
  groups: { route: string; code: number; count: number; last_seen: string }[];
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
const FLOW = ["new", "awaiting_accept", "awaiting_driver", "on_the_way", "at_door", "no_driver", "stuck", "emergency"] as const;
/** العاجلُ منها — يحمرّ إن لم يكن صفراً. */
const URGENT = new Set<string>(["no_driver", "stuck", "emergency"]);
const PARTS = ["api", "db", "cache", "push", "realtime"] as const;

// ── تصميمٌ احترافيّ (المالك ٢٠٢٦-١٠-٢٦) ──────────────────────────────
/** أيقونةُ كلّ مرحلةٍ في سير الطلبات. */
const FLOW_ICON: Record<(typeof FLOW)[number], typeof IconStore> = {
  new: IconBell,
  awaiting_accept: IconStore,
  awaiting_driver: IconHourglass,
  on_the_way: IconMoto,
  at_door: IconLocation,
  no_driver: IconDriver,
  stuck: IconWarning,
  emergency: IconShieldX,
};
/** الانتظارُ برتقاليّ — والعاجلُ أحمر، والصفرُ والمجهولُ رماديّان. */
const WAITING = new Set<string>(["new", "awaiting_accept", "awaiting_driver"]);
function flowTone(k: string, v: N | undefined): TileTone {
  if (v == null || v === 0) return "muted";
  if (URGENT.has(k)) return "danger";
  if (WAITING.has(k)) return "warning";
  return "default";
}
type St = "ok" | "slow" | "down" | "unknown";
const stateOf = (s?: string): St => (s === "ok" || s === "slow" || s === "down" ? s : "unknown");
const DOT: Record<St, string> = {
  ok: "bg-success",
  slow: "bg-warning",
  down: "bg-danger",
  unknown: "bg-line",
};

/** **ما يعدّه الرقمُ إن التبس** — «طوارئ» هنا طلبات، وبطاقةُ الطوارئ بلاغات. */
const FLOW_SUB: Partial<Record<(typeof FLOW)[number], string>> = {
  new: O.newFlowSub,
  emergency: O.emergencyFlowSub,
};
/** **أقسامٌ لم تُقرأ** — من حقل `missing`، ولا طمأنةَ مكانها. */
const MISSING = O.missingSections as Record<string, string>;

/** **فارغٌ مطمئن** — دائرةٌ خضراءُ صغيرةٌ وجملة. */
function AllClear({ text }: { text: string }) {
  return (
    <p className="flex items-center gap-2 text-sm font-bold text-success">
      <span className="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-success-solid text-on-solid">
        <IconSuccess size={14} />
      </span>
      {text}
    </p>
  );
}

/** **كتلةُ تفاصيلٍ للمهندس** — عنوانٌ بنقطة حالٍ إن عُرفت، ثمّ أسطرُ القيم. */
function Block({ title, state, children }: { title: string; state?: boolean; children: React.ReactNode }) {
  return (
    <div className="min-w-0 rounded-card border border-line-soft p-3">
      <h3 className="heading-card mb-2 flex items-center gap-2">
        {state != null && (
          <span className={`h-2.5 w-2.5 shrink-0 rounded-full ${state ? "bg-success" : "bg-danger"}`} />
        )}
        {title}
      </h3>
      {children}
    </div>
  );
}

const unknown = <span className="text-ink-muted">{O.unknown}</span>;

/** num **رقمٌ إن وُجد** — وغيابُه «غير معروف» لا صفر. */
function num(v?: number | null): React.ReactNode {
  return v == null ? unknown : fmtNum(v);
}
function ms(v?: number | null): React.ReactNode {
  return v == null ? unknown : O.unitMs.replace("{n}", fmtNum(v));
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
  // **زرُّ التحديث يقول إنّه يعمل** — يُطفأ مع أوّل جوابٍ أو عطب.
  const [refreshing, setRefreshing] = useState(false);
  const [tick, setTick] = useState(0);

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
    } finally {
      setTick((n) => n + 1);
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
  // **وأخطاءُ الخادم لمن يرى التفاصيل التقنيّة** — تُقرأ مع كلّ تحديث.
  // **وفشلُ قراءتها يُقال** — لا تختفي الكتلةُ فيُظنّ ألّا أخطاء.
  const errs = useLiveData<ServerErrors | "failed" | null>(
    () =>
      canEngineer
        ? api<ServerErrors>("/api/v1/admin/monitoring/errors").catch(() => "failed" as const)
        : Promise.resolve(null),
    ["system"],
    [canEngineer],
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
  const reloadErrs = errs.reload;
  useEffect(() => {
    const t = setInterval(() => {
      setNow(Date.now());
      reloadMon();
      reloadEng();
      reloadErrs();
    }, 60_000);
    return () => clearInterval(t);
  }, [reloadMon, reloadEng, reloadErrs]);

  useEffect(() => {
    setRefreshing(false);
  }, [tick]);
  useEffect(() => {
    if (!refreshing) return;
    const t = setTimeout(() => setRefreshing(false), 15_000);
    return () => clearTimeout(t);
  }, [refreshing]);

  const refresh = () => {
    setRefreshing(true);
    setNow(Date.now());
    reloadMon();
    reloadEng();
    reloadErrs();
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
  // **صفحةُ الطوارئ تحتاج صلاحيّتها** — ومن لا يملكها يُنقل لبطاقتها هنا.
  const canEmergency = capabilities.includes("emergencies.manage");
  const missing = data?.missing ?? [];
  const stuckMissing = missing.includes("stuck");
  const phrases: { text: string; href: string }[] = [];
  if (data && data.stuck.length > 0) {
    phrases.push({ text: O.pStuck.replace("{n}", fmtNum(data.stuck.length)), href: "#stuck" });
  }
  if (data?.emergencies_open) {
    phrases.push({
      text: O.pEmergency.replace("{n}", fmtNum(data.emergencies_open)),
      href: canEmergency ? "/dashboard/emergencies" : "#emergencies",
    });
  }
  for (const p of problemPhrases(parts)) phrases.push({ text: p, href: "#system" });
  const unreachable = !!error && code !== 403 && code !== 401;

  const bad = unreachable || phrases.length > 0;
  // **قسمٌ لم يُقرأ لا يُطمئن** — الدائرةُ برتقاليّةٌ والجملةُ تقول ما غاب.
  const partial = !bad && missing.length > 0;
  const missingText = missing.map((k) => MISSING[k] ?? k).join(" / ");
  const partsOk = !unreachable && PARTS.every((p) => parts?.[p] === "ok");

  return (
    <PageContainer>
      {/* **رأسٌ موحّد** — تصميمٌ احترافيّ (المالك ٢٠٢٦-١٠-٢٦): آخرُ تحديثٍ وزرُّه على الطرف. */}
      <PageHero
        icon={IconStatus}
        title={O.title}
        subtitle={O.subtitle}
        actions={
          <>
            <span className="text-xs text-ink-muted">
              {O.lastUpdate}: {at ? fmtClockTime(at) : loading ? O.loading : "—"}
            </span>
            <Button variant="secondary" disabled={refreshing} onClick={refresh}>
              {refreshing ? O.refreshing : O.refresh}
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
          {/* ── ١ · الحالُ بنظرة ── تصميمٌ احترافيّ (المالك ٢٠٢٦-١٠-٢٦): دائرةٌ كبيرةٌ خضراءُ أو حمراء،
              وجملُ المشكلة تُضغط فتنقل إليها. */}
          <section
            className={`surface flex flex-wrap items-center gap-4 p-5 ${
              bad ? "border-danger" : partial ? "border-warning" : ""
            }`}
          >
            <span
              className={`inline-flex h-16 w-16 shrink-0 items-center justify-center rounded-full text-on-solid ${
                bad ? "bg-danger-solid" : partial ? "bg-warning-solid" : "bg-success-solid"
              }`}
            >
              {bad || partial ? <IconWarning size={30} /> : <IconShieldCheck size={30} />}
            </span>
            <div className="min-w-0 flex-1 space-y-1">
              {unreachable ? (
                <h2 className="heading-section text-danger">{O.pUnreachable}</h2>
              ) : partial ? (
                <h2 className="heading-section text-warning">
                  {O.missingPrefix} {missingText}
                </h2>
              ) : phrases.length === 0 ? (
                <h2 className="heading-section text-success">{O.allGood}</h2>
              ) : (
                <h2 className="heading-section flex flex-wrap items-center gap-x-2 gap-y-1 text-danger">
                  <span>{O.problemPrefix}</span>
                  {phrases.map((p, i) => (
                    <span key={p.text} className="flex items-center gap-2">
                      {i > 0 && <span aria-hidden>·</span>}
                      <Link href={p.href} className="underline">
                        {p.text}
                      </Link>
                    </span>
                  ))}
                </h2>
              )}
              <p className="text-sm text-ink-muted">{bad ? O.badLine : partial ? O.missingLine : O.okLine}</p>
              {bad && missing.length > 0 && (
                <p className="text-sm font-bold text-warning">
                  {O.missingPrefix} {missingText}
                </p>
              )}
              <p className="text-xs text-ink-muted">{O.auto}</p>
            </div>
          </section>

          {/* ── ٢ · سيرُ الطلبات ── */}
          {data && (
            <>
              {/* **كلُّ رقمٍ بابٌ** — والصفرُ رماديٌّ، والانتظارُ برتقاليّ، والعاجلُ أحمر (المالك ٢٠٢٦-١٠-٢٦). */}
              <Panel title={O.flowTitle} note={O.flowHint}>
                <div className="grid grid-cols-2 gap-2 sm:grid-cols-3 lg:grid-cols-4">
                  {FLOW.map((k) => {
                    const v = data.flow[k];
                    return (
                      <Tile
                        key={k}
                        icon={FLOW_ICON[k]}
                        label={BOARD[k] ?? k}
                        value={v == null ? O.unknown : fmtNum(v)}
                        sub={FLOW_SUB[k]}
                        tone={flowTone(k, v)}
                        onClick={() => router.push(`/dashboard/orders?filter=${k}`)}
                      />
                    );
                  })}
                </div>
              </Panel>

              {/* **العالقُ بطاقات** — رقمٌ ومتجرٌ وسببٌ ومدّة، وزرّا الفتح و«أنا عليه». */}
              <section
                id="stuck"
                className={`surface p-4 ${
                  data.stuck.length > 0 ? "border-danger" : stuckMissing ? "border-warning" : ""
                }`}
              >
                <h2 className="heading-card mb-1 flex flex-wrap items-center gap-2">
                  <IconWarning size={18} className={data.stuck.length > 0 ? "text-danger" : "text-ink-muted"} />
                  {O.stuckTitle}
                  {data.stuck.length > 0 && (
                    <span className="rounded-full bg-danger-solid px-2 py-0.5 text-xs font-bold text-on-solid">
                      {O.stuckCount.replace("{n}", fmtNum(data.stuck.length))}
                    </span>
                  )}
                </h2>
                <p className="mb-3 text-xs text-ink-muted">
                  {data.reminder_min > 0 ? O.reminderEvery.replace("{n}", fmtNum(data.reminder_min)) : O.reminderOff}
                </p>
                {ackErr && <Alert>{ackErr}</Alert>}
                {stuckMissing ? (
                  <p className="flex items-center gap-2 text-sm font-bold text-warning">
                    <span className="inline-flex h-7 w-7 shrink-0 items-center justify-center rounded-full bg-warning-solid text-on-solid">
                      <IconWarning size={14} />
                    </span>
                    {O.stuckUnreadable}
                  </p>
                ) : data.stuck.length === 0 ? (
                  <AllClear text={O.stuckEmpty} />
                ) : (
                  <ul className="grid grid-cols-1 gap-2 lg:grid-cols-2">
                    {data.stuck.map((a) => (
                      <li
                        key={a.order_id}
                        className={`flex flex-col gap-3 rounded-card border p-3 ${
                          a.acked_at ? "border-line-soft" : "border-danger bg-danger-tint"
                        }`}
                      >
                        <div className="flex items-start gap-3">
                          {/* **المستلَمُ يهدأ لونُه** — فيبين بنظرةٍ أيُّ عالقٍ لسّا ما حدا عليه. */}
                          <span
                            className={`inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-full text-on-solid ${
                              a.acked_at ? "bg-warning-solid" : "bg-danger-solid"
                            }`}
                          >
                            {a.acked_at ? <IconCheck size={18} /> : <IconHourglass size={18} />}
                          </span>
                          <div className="min-w-0 flex-1 space-y-1">
                            <div className="flex flex-wrap items-center gap-2">
                              <OrderRef number={a.number} />
                              <span className="min-w-0 truncate font-bold">{a.merchant_name}</span>
                            </div>
                            <div className="flex flex-wrap items-center gap-1.5">
                              <Badge variant="danger">{REASONS[a.reason] ?? a.reason}</Badge>
                              <Badge variant={a.acked_at ? "success" : "warning"}>
                                {a.acked_at
                                  ? O.ackedSince.replace(
                                      "{t}",
                                      fmtSpan(Math.max(0, (now - Date.parse(a.acked_at)) / 1000)),
                                    )
                                  : O.notAcked}
                              </Badge>
                            </div>
                            <p className="text-xs text-ink-muted">
                              {O.stuckFor.replace("{t}", fmtSpan(Math.max(0, (now - Date.parse(a.since)) / 1000)))}
                              {a.reminders > 0 &&
                                `${m.common.listSeparator}${O.reminded.replace("{n}", fmtNum(a.reminders))}`}
                            </p>
                          </div>
                        </div>
                        {/* **أزرارٌ بعرض البطاقة** — و«أنا عليه» أساسيٌّ لا أحمر: هو استلامٌ لا حذف. */}
                        <div className={`grid gap-2 ${canAck && !a.acked_at ? "grid-cols-2" : "grid-cols-1"}`}>
                          <ButtonLink
                            variant="secondary"
                            className="w-full justify-center"
                            href={`/dashboard/orders?id=${a.order_id}`}
                          >
                            {O.openOrder}
                          </ButtonLink>
                          {canAck && !a.acked_at && (
                            <Button
                              variant="primary"
                              className="w-full"
                              disabled={acking === a.order_id}
                              onClick={() => void ackAlert(a.order_id)}
                            >
                              {O.ack}
                            </Button>
                          )}
                        </div>
                        {canAck && !a.acked_at && <p className="text-2xs text-ink-muted">{O.ackHint}</p>}
                      </li>
                    ))}
                  </ul>
                )}
              </section>

              <div className="grid min-w-0 grid-cols-1 gap-4 md:grid-cols-2">
                {/* **السائقون** — ثلاثُ دوائرَ بأرقامها، والخريطةُ تحتها. */}
                <Panel title={O.driversTitle}>
                  <ul className="space-y-1.5">
                    {(
                      [
                        [O.onShift, data.drivers.on_shift, "bg-accent text-on-bright"],
                        [O.busy, data.drivers.busy, "bg-warning-solid text-on-solid"],
                        [O.free, data.drivers.free, "bg-success-solid text-on-solid"],
                      ] as const
                    ).map(([label, v, tint]) => (
                      <li key={label} className="flex items-center gap-3 rounded-control bg-field px-3 py-2">
                        <span
                          className={`inline-flex h-9 w-9 shrink-0 items-center justify-center rounded-full ${
                            v ? tint : "border border-line text-ink-muted"
                          }`}
                        >
                          <IconDriver size={16} />
                        </span>
                        <span className="min-w-0 flex-1 text-sm">{label}</span>
                        <span className={`figure leading-none ${v ? "" : "text-ink-muted"}`}>{num(v)}</span>
                      </li>
                    ))}
                  </ul>
                  <div className="mt-3">
                    <ButtonLink variant="secondary" className="w-full justify-center" href="/dashboard/opsmap">
                      {O.showMap}
                    </ButtonLink>
                  </div>
                </Panel>

                {/* **الطوارئ** — دائرةٌ كبيرةٌ بالعدد، حمراءُ إن كان فيها شي. */}
                <section
                  id="emergencies"
                  className={`surface p-4 ${data.emergencies_open ? "border-danger" : ""}`}
                >
                  <h2 className="heading-card mb-1">{O.emergenciesTitle}</h2>
                  <p className="mb-3 text-xs text-ink-muted">{O.emergenciesCardNote}</p>
                  <div className="flex items-center gap-4">
                    <span
                      className={`inline-flex h-16 w-16 shrink-0 items-center justify-center rounded-full ${
                        data.emergencies_open ? "bg-danger-solid text-on-solid" : "bg-field text-ink-muted"
                      }`}
                    >
                      <span className="figure leading-none">{num(data.emergencies_open)}</span>
                    </span>
                    <p
                      className={`min-w-0 flex-1 text-sm font-bold ${
                        data.emergencies_open ? "text-danger" : data.emergencies_open === 0 ? "text-success" : "text-ink-muted"
                      }`}
                    >
                      {data.emergencies_open
                        ? O.emergenciesSome
                        : data.emergencies_open === 0
                          ? O.emergenciesNone
                          : O.unknown}
                    </p>
                  </div>
                  {/* **الزرُّ لمن يفتح الصفحة** — ومن لا يملك الطوارئ يرى طلباتها في لوحة الطلبات. */}
                  <div className="mt-3">
                    {canEmergency ? (
                      <ButtonLink variant="secondary" className="w-full justify-center" href="/dashboard/emergencies">
                        {O.openEmergencies}
                      </ButtonLink>
                    ) : (
                      <ButtonLink
                        variant="secondary"
                        className="w-full justify-center"
                        href="/dashboard/orders?filter=emergency"
                      >
                        {O.openEmergencyOrders}
                      </ButtonLink>
                    )}
                  </div>
                </section>
              </div>
            </>
          )}

          {/* ── ٣ · حالةُ النظام ── سطرٌ لكلّ جزءٍ بنقطةٍ ملوّنةٍ وكلمة (المالك ٢٠٢٦-١٠-٢٦). */}
          <section id="system" className="surface p-4">
            <h2 className="heading-card mb-1">{O.systemTitle}</h2>
            <p className="mb-3 text-xs text-ink-muted">{O.systemNote}</p>
            {partsOk && <AllClear text={O.systemOk} />}
            <ul className="mt-2 grid grid-cols-1 gap-2 sm:grid-cols-2 lg:grid-cols-5">
              {PARTS.map((p) => {
                const detail = health?.verdict?.parts?.[p];
                const reason = detail?.reason ? (O.reasons as Record<string, string>)[detail.reason] : "";
                const st = unreachable ? undefined : parts?.[p];
                return (
                  <li key={p} className="flex min-w-0 items-start gap-2.5 rounded-control bg-field px-3 py-2.5">
                    <span className={`mt-1.5 h-2.5 w-2.5 shrink-0 rounded-full ${DOT[stateOf(st)]}`} />
                    <span className="min-w-0 flex-1">
                      <span className="block text-sm font-bold">{O.parts[p]}</span>
                      <span className="mt-1 block">
                        <StateBadge state={st} />
                      </span>
                      {reason && <span className="mt-1 block text-xs text-ink-muted">{reason}</span>}
                    </span>
                  </li>
                );
              })}
            </ul>

            {canEngineer && health && (
              <details className="group mt-4 rounded-card border border-line-soft">
                <summary className="flex cursor-pointer list-none items-center gap-2 p-3 text-sm font-bold">
                  <IconSettings size={16} className="text-ink-muted" />
                  <span className="flex-1">{O.engineer}</span>
                  <span className="text-xs text-ink-muted transition-transform group-open:rotate-180">{"▾"}</span>
                </summary>
                <div className="border-t border-line-soft p-3">
                  <p className="text-xs text-ink-muted">
                    {O.sinceRestart.replace(
                      "{t}",
                      rt?.uptime_seconds == null ? O.unknown : fmtSpan(rt.uptime_seconds),
                    )}
                    {health.window_minutes != null &&
                      `${m.common.listSeparator}${O.windowNote.replace("{n}", fmtNum(health.window_minutes))}`}
                  </p>
                  <div className="mt-3 grid min-w-0 grid-cols-1 gap-3 md:grid-cols-2 xl:grid-cols-3">
                    <Block title={O.secServer}>
                      <Row label={O.uptime} value={span(rt?.uptime_seconds)} />
                      <Row label={O.goroutines} value={num(rt?.goroutines)} />
                      <Row label={O.heap} value={rt?.heap_mb == null ? unknown : O.unitMb.replace("{n}", fmtNum(rt.heap_mb))} />
                      <Row label={O.sys} value={rt?.sys_mb == null ? unknown : O.unitMb.replace("{n}", fmtNum(rt.sys_mb))} />
                    </Block>
                    <Block title={O.secDb} state={pg?.reachable}>
                      <Row label={O.ping} value={ms(pg?.ping_ms)} />
                      <Row label={O.backends} value={num(pg?.backends)} />
                      <Row label={O.active} value={num(pg?.active)} />
                      <Row label={O.idleInTx} value={num(pg?.idle_in_transaction)} />
                      <Row label={O.longestTx} value={span(pg?.longest_tx_seconds)} />
                      <Row label={O.waitingLocks} value={num(pg?.waiting_locks)} />
                    </Block>
                    <Block title={O.secPool}>
                      <Row label={O.poolMax} value={num(pool?.max)} />
                      <Row label={O.poolTotal} value={num(pool?.total)} />
                      <Row label={O.poolAcquired} value={num(pool?.acquired)} />
                      <Row label={O.poolIdle} value={num(pool?.idle)} />
                      <Row label={O.poolEmpty} value={num(pool?.empty_acquire_count)} />
                      <Row label={O.poolCanceled} value={num(pool?.canceled_acquire_count)} />
                      <Row label={O.poolWaitTotal} value={ms(pool?.acquire_duration_ms_total)} />
                    </Block>
                    <Block title={O.secRedis} state={rd?.reachable}>
                      <Row label={O.ping} value={ms(rd?.ping_ms)} />
                      <Row label={O.redisTotal} value={num(rd?.total_conns)} />
                      <Row label={O.redisIdle} value={num(rd?.idle_conns)} />
                      <Row label={O.redisTimeouts} value={num(rd?.pool_timeouts)} />
                    </Block>
                    <Block title={O.secWs}>
                      <Row label={O.wsSubs} value={num(health.ws?.subscriptions)} />
                      <p className="mb-1 mt-2 text-xs font-bold">{O.wsAuthTitle}</p>
                      <Counters map={health.ws?.auth} labels={(k) => WS_LABELS[k] ?? k} />
                    </Block>
                    <Block title={O.secPush}>
                      <Counters map={health.push} labels={(k) => PUSH_LABELS[k] ?? k} />
                    </Block>
                    <Block title={O.secClients}>
                      <p className="mb-1 text-xs text-ink-muted">{O.clientsHint}</p>
                      <Counters map={health.clients} labels={appLabel} />
                      <Row label={O.clientsDropped} value={num(health.clients_dropped)} />
                    </Block>
                    {errs.data === "failed" ? (
                      <Block title={O.secErrors}>
                        <p className="py-2 text-sm font-bold text-warning">{O.errorsUnreadable}</p>
                      </Block>
                    ) : (
                      errs.data && (
                        <Block title={O.secErrors} state={errs.data.groups.length === 0}>
                          {errs.data.groups.length === 0 ? (
                            <AllClear text={O.errorsNone} />
                          ) : (
                            <>
                              <p className="mb-2 text-xs text-ink-muted">
                                {O.errorsTotal
                                  .replace("{n}", fmtNum(errs.data.total))
                                  .replace("{h}", fmtNum(errs.data.last_hour))}
                              </p>
                              {/* **المسارُ بسطرٍ لاتينيٍّ ينكسر** — والعددُ وآخرُ مرّةٍ (بيومها) تحته. */}
                              {errs.data.groups.slice(0, 10).map((g) => (
                                <div
                                  key={`${g.route}|${g.code}`}
                                  className="min-w-0 border-b border-line-soft py-1.5 last:border-0"
                                >
                                  <p dir="ltr" className="break-all text-start font-mono text-[11px]">
                                    {`${g.route} (${g.code})`}
                                  </p>
                                  <p className="mt-0.5 text-xs text-ink-muted">
                                    {O.errorsCount.replace("{n}", fmtNum(g.count))}
                                    {m.common.listSeparator}
                                    {O.errorsLast.replace("{t}", fmtDateTime(g.last_seen))}
                                  </p>
                                </div>
                              ))}
                            </>
                          )}
                        </Block>
                      )
                    )}
                    <Block title={O.secRelease}>
                      <Row label={O.environment} value={id?.environment ?? O.unknown} ltr />
                      <Row label={O.migration} value={id?.migration_version ?? O.unknown} ltr />
                      <Row label={O.commit} value={rt?.source_commit || id?.source_commit || O.unknown} ltr />
                      <Row label={O.buildId} value={rt?.build_id || id?.build_id || O.unknown} ltr />
                    </Block>
                  </div>
                </div>
              </details>
            )}
          </section>
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
