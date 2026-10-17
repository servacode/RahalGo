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
}

const Ctx = createContext<LiveState | null>(null);

export function LiveShareProvider({ open, children }: { open: MyOrder[] | null; children: ReactNode }) {
  const [activeId, setActiveId] = useState("");
  const [decided, setDecided] = useState<string[]>([]);
  const [problem, setProblem] = useState<"" | "denied" | "unsupported">("");
  const filter = useRef(new LiveSendFilter());
  const watchId = useRef<number | null>(null);
  const last = useRef<Reading | null>(null);
  const tick = useRef<ReturnType<typeof setInterval> | null>(null);
  const deadline = useRef<ReturnType<typeof setTimeout> | null>(null);
  const current = useRef("");

  useEffect(() => {
    const saved = readJSON<string[]>(DECIDED_KEY, []);
    setDecided(Array.isArray(saved) ? saved.filter((x) => typeof x === "string") : []);
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
            send(r);
          },
          (e) => {
            if (e.code === e.PERMISSION_DENIED) stop("denied");
          },
          { enableHighAccuracy: true, maximumAge: 0, timeout: 20_000 },
        );
      } catch {
        stop("denied");
        return false;
      }
      tick.current = setInterval(() => {
        const r = last.current;
        if (r) send({ ...r, atMs: performance.now() });
      }, SEND_EVERY_MS);
      deadline.current = setTimeout(() => stop("timeout"), MAX_SHARE_MS);
      return true;
    },
    [send, stop, teardown],
  );

  /* **ووصل الطلبُ أو انتهى فتقف** — في لحظة القراءة لا بعد نقطةٍ تُردّ. */
  useEffect(() => {
    if (!activeId || !open) return;
    const o = open.find((x) => x.id === activeId);
    if (!o || !LIVE_STATUSES.has(o.status)) stop("ended");
  }, [open, activeId, stop]);

  useEffect(() => teardown, [teardown]);

  const value = useMemo<LiveState>(
    () => ({ activeId, decided, decide, start, stop, problem }),
    [activeId, decided, decide, start, stop, problem],
  );
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}

export function useLiveShare(): LiveState {
  const c = useContext(Ctx);
  if (!c) throw new Error("useLiveShare must be used inside LiveShareProvider");
  return c;
}
