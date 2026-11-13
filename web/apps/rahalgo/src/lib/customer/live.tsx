"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **مشاركةُ الموقع المباشر مع الكابتن — من المتصفّح**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (مرآةُ `LiveShare.kt` و`LiveSendFilter.kt` و`LiveShareService.kt`.)
 *
 * **دبابيسُ الخريطة في الرقّة كثيراً ما تخطئ** — فيقف الكابتنُ بعيداً.
 * **فحين يصير الطلبُ «بالطريق» يُسأل الزبونُ مرّةً لكلّ طلب**، ومن شارك
 * أرسل متصفّحُه موقعَه **بإيقاع التطبيق نفسِه**:
 *
 *   - **الدقيقُ يُرسَل** — دقّتُه دون حدّ اللوحة
 *     (`orders.live_location_max_accuracy_m`، افتراضُه ٣٠ ويعود في الردّ).
 *   - **والتقريبيُّ يُحبَس** ويُحفظ أحسنُه، **وبعد ثلاثين ثانيةً بلا دقيقٍ
 *     يُرسَل أحسنُ ما وُجد** فيَسِمه المحرّكُ «تقريبيّاً».
 *   - **ولا أسرعَ من مرّةٍ كلَّ ثماني ثوانٍ**، **ويُعاد آخرُ ما قُرئ كلَّ عشر**
 *     (المتصفّحُ لا يُبلغ عن جهازٍ ساكن، والتطبيقُ يسأل كلَّ عشر).
 *
 * **وتقف وحدَها**: وصل الطلبُ أو انتهى (`live_location_inactive`)، أو مضت
 * خمسٌ وأربعون دقيقة، أو ضغط «وقّف المشاركة». **ولا تعود بلا ضغطةٍ منه** —
 * من أغلق الصفحةَ انتهت مشاركتُه، **والمتصفّحُ لا يشارك إلّا والصفحةُ مفتوحة.**
 */

import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { api } from "@/lib/api";
import { readJSON, writeJSON } from "./storage";
import { LIVE_STATUSES, type MyOrder } from "./types";

/** **حدُّ الدقّة الافتراضيّ** — كافتراض اللوحة. */
export const DEFAULT_MAX_ACCURACY_M = 30;
/** **ثلاثون ثانيةً بلا دقيقٍ ثمّ أحسنُ ما وُجد.** */
export const FALLBACK_MS = 30_000;
/** **ولا أسرعَ من هذا.** */
export const MIN_GAP_MS = 8_000;
/** **إيقاعُ التطبيق** — يُعاد آخرُ ما قُرئ كلَّ عشر ثوانٍ. */
export const SEND_EVERY_MS = 10_000;
/** **سقفُ المشاركة** — خمسٌ وأربعون دقيقة. */
export const MAX_SHARE_MS = 45 * 60 * 1000;
/** **متى يُسأل** — «بالطريق» وحدَها. */
export const ASK_STATUS = "on_the_way";

const DECIDED_KEY = "rahalgo_live_decided";
const AUTO_KEY = "rahalgo_live_auto";

/**
 * **«شارك موقعي المباشر مع السائق أثناء الطلب»** — تفضيلٌ على هذا المتصفّح، **مفعّلٌ
 * افتراضاً** (قرارُ المالك ٢٠٢٦-١٠-٣٠، مرآةُ `LiveAuto.kt`). وما دام مفعّلاً تبدأ المشاركةُ
 * وحدَها حين يصير الطلبُ «بالطريق» — **والإيقافُ في لافتة المشاركة.**
 */
export function liveAutoOn(): boolean {
  return readJSON<boolean>(AUTO_KEY, true) !== false;
}

export function setLiveAuto(on: boolean): void {
  writeJSON(AUTO_KEY, on);
}
const MAX_REMEMBERED = 50;

export interface Reading {
  lat: number;
  lng: number;
  accuracyM: number | null;
  atMs: number;
}

/**
 * **المرشّح — مرآةُ `LiveSendFilter` حرفاً.** يردّ ما يُرسَل الآن أو `null`.
 */
export class LiveSendFilter {
  maxAccuracyM = DEFAULT_MAX_ACCURACY_M;
  private windowStart = -1;
  private lastSent = -1;
  private best: Reading | null = null;

  precise(r: Reading): boolean {
    return r.accuracyM != null && r.accuracyM <= this.maxAccuracyM;
  }

  offer(r: Reading): Reading | null {
    if (this.windowStart < 0) this.windowStart = r.atMs;
    if (this.best == null || better(r, this.best)) this.best = r;
    if (this.lastSent >= 0 && r.atMs - this.lastSent < MIN_GAP_MS) return null;
    if (this.precise(r)) return this.sent(r, r.atMs);
    const b = this.best;
    if (this.precise(b)) return this.sent(b, r.atMs);
    if (r.atMs - this.windowStart >= FALLBACK_MS) return this.sent(b, r.atMs);
    return null;
  }

  reset(): void {
    this.windowStart = -1;
    this.lastSent = -1;
    this.best = null;
  }

  private sent(r: Reading, nowMs: number): Reading {
    this.lastSent = nowMs;
    this.windowStart = nowMs;
    this.best = null;
    return r;
  }
}

function better(a: Reading, b: Reading): boolean {
  const x = a.accuracyM ?? Number.MAX_VALUE;
  const y = b.accuracyM ?? Number.MAX_VALUE;
  return x <= y;
}

/** **الطلبُ الذي يُسأل صاحبُه الآن** — بالطريق ولم يُقرَّر فيه. */
export function pendingAsk(open: MyOrder[], decided: string[]): MyOrder | null {
  return open.find((o) => o.status === ASK_STATUS && !decided.includes(o.id)) ?? null;
}

type StopReason = "user" | "ended" | "inactive" | "timeout" | "denied";

interface LiveState {
  /** **الطلبُ الذي يُشارَك موقعُه** — وفارغٌ: لا مشاركة. */
  activeId: string;
  decided: string[];
  decide: (orderId: string) => void;
  start: (orderId: string) => boolean;
  stop: (reason?: StopReason) => void;
  /** **آخرُ سببِ توقّفٍ يُقال للزبون** — الإذنُ مرفوضٌ أو المتصفّحُ لا يدعم. */
  problem: "" | "denied" | "unsupported";
  /** **دقّةُ آخر قراءةٍ بالمتر** — تُرى في اللافتة، **و`null` قبل أوّل قراءة.** */
  accuracyM: number | null;
}

/**
 * **قفلُ الشاشة أثناء المشاركة** (ملاحظةُ المالك ٢٠٢٦-١١-١٤: «على الآيفون الموقعُ المباشرُ لا يعمل فعليّاً»).
 *
 * **سفاري يوقف `watchPosition` حين تنطفئ الشاشة أو يغيب التبويب** — لا موقعَ في الخلفيّة لصفحة ويب، وهذا
 * حدُّ آبل لا حدُّنا. **فيُطلب من المتصفّح إبقاءُ الشاشة مضاءةً** (`navigator.wakeLock`، iOS ١٦٫٤ وما بعده)
 * ما دامت المشاركة، **ويُطلب ثانيةً كلّما عادت الصفحة** (المتصفّحُ يحرّره عند الغياب). ومتصفّحٌ لا يعرفه
 * لا يُخطئ — يبقى كما كان.
 */
/** **حدُّ «تقريبيّ»** — مرآةُ `COARSE_ACCURACY_M` في الخريطة. */
export const COARSE_ACCURACY_M = 150;

/** **أهو آيفون أو آيباد؟** — والآيبادُ الحديثُ يقول إنّه ماك، **فيُعرف بشاشة اللمس.** */
export function isAppleMobile(): boolean {
  if (typeof navigator === "undefined") return false;
  const ua = navigator.userAgent || "";
  return /iPad|iPhone|iPod/.test(ua) || (navigator.platform === "MacIntel" && (navigator.maxTouchPoints ?? 0) > 1);
}

type WakeLockLike = { release: () => Promise<void> };
async function requestWake(): Promise<WakeLockLike | null> {
  try {
    const wl = (navigator as unknown as { wakeLock?: { request: (t: "screen") => Promise<WakeLockLike> } }).wakeLock;
    return wl ? await wl.request("screen") : null;
  } catch {
    return null;
  }
}

const Ctx = createContext<LiveState | null>(null);

export function LiveShareProvider({ open, children }: { open: MyOrder[] | null; children: ReactNode }) {
  const [activeId, setActiveId] = useState("");
  const [decided, setDecided] = useState<string[]>([]);
  const [problem, setProblem] = useState<"" | "denied" | "unsupported">("");
  const [accuracyM, setAccuracyM] = useState<number | null>(null);
  const wake = useRef<WakeLockLike | null>(null);
  /* **ولا تلقائيّةَ قبل قراءة ما قُرِّر** — وإلّا عادت لطلبٍ أوقفها صاحبُه. */
  const [decidedReady, setDecidedReady] = useState(false);
  const filter = useRef(new LiveSendFilter());
  const watchId = useRef<number | null>(null);
  const last = useRef<Reading | null>(null);
  const tick = useRef<ReturnType<typeof setInterval> | null>(null);
  const deadline = useRef<ReturnType<typeof setTimeout> | null>(null);
  const current = useRef("");

  useEffect(() => {
    const saved = readJSON<string[]>(DECIDED_KEY, []);
    setDecided(Array.isArray(saved) ? saved.filter((x) => typeof x === "string") : []);
    setDecidedReady(true);
  }, []);

  const decide = useCallback((orderId: string) => {
    setDecided((cur) => {
      const next = [...cur.filter((x) => x !== orderId), orderId].slice(-MAX_REMEMBERED);
      writeJSON(DECIDED_KEY, next);
      return next;
    });
  }, []);

  const teardown = useCallback(() => {
    if (watchId.current != null) {
      try {
        navigator.geolocation.clearWatch(watchId.current);
      } catch {
        /* المتصفّحُ أغلقها */
      }
    }
    watchId.current = null;
    wake.current?.release().catch(() => undefined);
    wake.current = null;
    setAccuracyM(null);
    if (tick.current) clearInterval(tick.current);
    tick.current = null;
    if (deadline.current) clearTimeout(deadline.current);
    deadline.current = null;
    last.current = null;
    filter.current.reset();
  }, []);

  const stop = useCallback(
    (reason: StopReason = "user") => {
      const id = current.current;
      current.current = "";
      teardown();
      setActiveId("");
      if (reason === "denied") setProblem("denied");
      /* **ويُمحى موقعُه في المحرّك** فيعود الكابتنُ إلى الدبّوس — **إلّا إن
         قال المحرّكُ إنّ الطلبَ ليس في طريقه**: محاه أصلاً. */
      if (id && reason !== "inactive") {
        api(`/api/v1/orders/${id}/live-location`, { method: "DELETE" }).catch(() => undefined);
      }
    },
    [teardown],
  );

  const send = useCallback(
    (r: Reading) => {
      const id = current.current;
      if (!id) return;
      const pick = filter.current.offer(r);
      if (!pick) return;
      api<{ saved: boolean; max_accuracy_m?: number | null }>(`/api/v1/orders/${id}/live-location`, {
        method: "POST",
        body: JSON.stringify({ lat: pick.lat, lng: pick.lng, accuracy_m: pick.accuracyM }),
      })
        .then((res) => {
          if (typeof res?.max_accuracy_m === "number" && res.max_accuracy_m > 0) {
            filter.current.maxAccuracyM = res.max_accuracy_m;
          }
        })
        .catch((err) => {
          const code = (err as { body?: { code?: string } })?.body?.code;
          if (code === "live_location_inactive" || code === "not_found") stop("inactive");
          /* **والشبكةُ تسقط في الشارع** — والنقطةُ التاليةُ تحلّ محلّها. */
        });
    },
    [stop],
  );

  /** **تبدأ المراقبةُ — أو تُعاد من جديد** فتُردّ القراءةُ الأولى بلا انتظار. ويردّ `false` إن رفضها المتصفّح. */
  const arm = useCallback((): boolean => {
    if (watchId.current != null) {
      try {
        navigator.geolocation.clearWatch(watchId.current);
      } catch {
        /* أُغلقت */
      }
      watchId.current = null;
    }
    try {
      watchId.current = navigator.geolocation.watchPosition(
        (p) => {
          const r: Reading = {
            lat: p.coords.latitude,
            lng: p.coords.longitude,
            accuracyM: Number.isFinite(p.coords.accuracy) ? p.coords.accuracy : null,
            atMs: performance.now(),
          };
          last.current = r;
          setAccuracyM(r.accuracyM != null ? Math.ceil(r.accuracyM) : null);
          send(r);
        },
        (e) => {
          if (e.code === e.PERMISSION_DENIED) stop("denied");
        },
        { enableHighAccuracy: true, maximumAge: 0, timeout: 20_000 },
      );
      return true;
    } catch {
      return false;
    }
  }, [send, stop]);

  const start = useCallback(
    (orderId: string): boolean => {
      if (typeof navigator === "undefined" || !("geolocation" in navigator)) {
        setProblem("unsupported");
        return false;
      }
      teardown();
      setProblem("");
      current.current = orderId;
      setActiveId(orderId);
      if (!arm()) {
        stop("denied");
        return false;
      }
      void requestWake().then((w) => {
        if (current.current === orderId) wake.current = w;
        else w?.release().catch(() => undefined);
      });
      tick.current = setInterval(() => {
        const r = last.current;
        if (r) send({ ...r, atMs: performance.now() });
      }, SEND_EVERY_MS);
      deadline.current = setTimeout(() => stop("timeout"), MAX_SHARE_MS);
      return true;
    },
    [send, stop, teardown, arm],
  );

  /* **وعادت الصفحةُ بعد غياب** — سفاري أوقف المراقبةَ والقفلَ وهي غائبة: **تُعاد المراقبةُ من جديد
     ويُطلب القفلُ ثانية**، وأوّلُ قراءةٍ تُرسَل فوراً لا بعد عشر ثوانٍ. */
  useEffect(() => {
    const onVis = () => {
      if (document.visibilityState !== "visible" || !current.current) return;
      arm();
      const id = current.current;
      void requestWake().then((w) => {
        if (current.current === id) {
          wake.current?.release().catch(() => undefined);
          wake.current = w;
        } else w?.release().catch(() => undefined);
      });
    };
    document.addEventListener("visibilitychange", onVis);
    return () => document.removeEventListener("visibilitychange", onVis);
  }, [arm]);

  /* **والمشاركةُ التلقائيّة** — طلبٌ «بالطريق» لم يُقرَّر فيه والتفضيلُ مفعّل: تبدأ بلا سؤال. */
  useEffect(() => {
    if (!decidedReady || activeId || !open || !liveAutoOn()) return;
    const o = pendingAsk(open, decided);
    if (!o) return;
    decide(o.id);
    start(o.id);
  }, [decidedReady, open, decided, activeId, decide, start]);

  /* **ووصل الطلبُ أو انتهى فتقف** — في لحظة القراءة لا بعد نقطةٍ تُردّ. */
  useEffect(() => {
    if (!activeId || !open) return;
    const o = open.find((x) => x.id === activeId);
    if (!o || !LIVE_STATUSES.has(o.status)) stop("ended");
  }, [open, activeId, stop]);

  useEffect(() => teardown, [teardown]);

  const value = useMemo<LiveState>(
    () => ({ activeId, decided, decide, start, stop, problem, accuracyM }),
    [activeId, decided, decide, start, stop, problem, accuracyM],
  );
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}

export function useLiveShare(): LiveState {
  const c = useContext(Ctx);
  if (!c) throw new Error("useLiveShare must be used inside LiveShareProvider");
  return c;
}
