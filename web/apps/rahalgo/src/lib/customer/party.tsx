"use client";

/**
 * ══════════════════════════════════════════════════════════════════════
 * **«الطرفُ الآخر يشارك موقعَه من تطبيقنا» — طلباتٌ يكون فيها الزبونُ طرفاً**
 * ══════════════════════════════════════════════════════════════════════
 *
 * (قرارُ المالك ٢٠٢٦-١١-٢٨.) **حين تطلب المنصّةُ موقعَ هاتفٍ** (مستلمُ أمانة، مستلمُ توصيلةِ
 * متجر، طلبُ مكتبٍ بالواتساب) **وكان الهاتفُ لزبونٍ مسجَّلٍ** يرى سؤالَ «نعم / لا». «نعم» موافقةٌ
 * صريحة، فيبدأ المتصفّحُ يشارك موقعَه **ما دامت الصفحةُ مفتوحة** — بلا نقرةٍ ثانية.
 *
 * **والنقاطُ**: المتصفّحُ لا يُبلغ عن جهازٍ ساكن فيُعاد آخرُ ما قُرئ كلَّ عشر ثوانٍ، **ولا أسرعَ من
 * ثماني.** ومُنع الموقعُ في المتصفّح فعنوانُه المحفوظ (بإحداثيّاته) **يُرسَل مرّةً واحدةً** وبعد «نعم» فقط.
 *
 * **ومن ضغط «إيقاف» لا تعود المشاركةُ لهالطلب على هذا المتصفّح.**
 */

import { createContext, useCallback, useContext, useEffect, useMemo, useRef, useState, type ReactNode } from "react";
import { errorText } from "@rahalgo/i18n";
import { useLiveRefresh } from "@rahalgo/ui";
import { api } from "@/lib/api";
import { useCustomer } from "./context";
import { readJSON, writeJSON } from "./storage";

export interface PartyView {
  order_id: string;
  kind: "amanah" | "delivery" | "order" | string;
  role: string;
  sender_phone?: string;
  store_name?: string;
  status: string;
  captain_assigned: boolean;
  code?: string;
  answer: "" | "yes" | "no" | string;
  shared: boolean;
  open: boolean;
}

/** **ولا أسرعَ من هذا** — مرآةُ مشاركة الطلب العاديّ. */
const MIN_GAP_MS = 8_000;
/** **ويُعاد آخرُ ما قُرئ كلَّ عشر.** */
const EVERY_MS = 10_000;
const REFRESH_MS = 15_000;
const STOPPED_KEY = "rahalgo_party_user_stopped";
const MAX_REMEMBERED = 50;

function stoppedList(): string[] {
  const cur = readJSON<string[]>(STOPPED_KEY, []);
  return Array.isArray(cur) ? cur.filter((x) => typeof x === "string") : [];
}

function errCode(err: unknown): string {
  return (err as { body?: { code?: string } })?.body?.code ?? "";
}

interface PartyState {
  /** **المفتوحةُ** — و`null` قبل أوّل قراءة. */
  orders: PartyView[] | null;
  /** **ما يُسأل عنه الآن** (البطاقةُ المفتوحة). */
  shownId: string;
  show: (id: string) => void;
  hide: () => void;
  /** **ما تجري مشاركتُه الآن.** */
  sharingIds: string[];
  answer: (id: string, answer: "yes" | "no") => Promise<string>;
  stop: (id: string) => Promise<void>;
}

const Ctx = createContext<PartyState>({
  orders: null,
  shownId: "",
  show: () => undefined,
  hide: () => undefined,
  sharingIds: [],
  answer: async () => "",
  stop: async () => undefined,
});

export function usePartyOrders(): PartyState {
  return useContext(Ctx);
}

export function PartyProvider({ children }: { children: ReactNode }) {
  const { address } = useCustomer();
  const [orders, setOrders] = useState<PartyView[] | null>(null);
  const [shownId, setShownId] = useState("");
  // **ما أُوقف بيده أو انتهى** — نسخةٌ في الذاكرة تُحدَّث مع التخزين، وعدّادٌ لإعادة الحساب.
  const stopped = useRef<Set<string>>(new Set(stoppedList()));
  const dead = useRef<Set<string>>(new Set());
  const [bump, setBump] = useState(0);
  const urlHandled = useRef<Set<string>>(new Set());

  const reload = useCallback(() => {
    if (typeof document !== "undefined" && document.visibilityState === "hidden") return;
    api<{ orders: PartyView[] }>("/api/v1/my/party-orders")
      .then((r) => setOrders((r.orders ?? []).filter((o) => o.open)))
      .catch(() => undefined); // **وقراءةٌ فشلت تُعاد بعد ربع دقيقة.**
  }, []);

  useEffect(() => {
    reload();
    const t = setInterval(reload, REFRESH_MS);
    const onShow = () => document.visibilityState === "visible" && reload();
    document.addEventListener("visibilitychange", onShow);
    return () => {
      clearInterval(t);
      document.removeEventListener("visibilitychange", onShow);
    };
  }, [reload]);

  useLiveRefresh(["party_orders"], reload);

  // **`?party=<id>` من إشعارٍ** — تُفتح بطاقتُه تلقائياً مرّةً لكلّ رقم.
  useEffect(() => {
    if (!orders) return;
    const id = new URLSearchParams(window.location.search).get("party") ?? "";
    if (!id || urlHandled.current.has(id)) return;
    const o = orders.find((x) => x.order_id === id);
    if (!o) return;
    urlHandled.current.add(id);
    if (o.answer === "") setShownId(id);
  }, [orders]);

  const replace = useCallback((v: PartyView) => {
    setOrders((cur) => {
      const rest = (cur ?? []).filter((o) => o.order_id !== v.order_id);
      return v.open ? [...rest, v] : rest;
    });
  }, []);

  const markDead = useCallback(
    (id: string) => {
      dead.current.add(id);
      setBump((n) => n + 1);
      reload();
    },
    [reload],
  );

  const answer = useCallback(
    async (id: string, ans: "yes" | "no"): Promise<string> => {
      try {
        const v = await api<PartyView>(`/api/v1/my/party-orders/${id}/answer`, {
          method: "POST",
          body: JSON.stringify({ answer: ans }),
        });
        if (ans === "yes") {
          stopped.current.delete(id);
          dead.current.delete(id);
          writeJSON(STOPPED_KEY, [...stopped.current]);
        }
        replace(v);
        setShownId("");
        return "";
      } catch (err) {
        if (errCode(err) === "live_location_inactive") {
          setShownId("");
          markDead(id);
          return "";
        }
        return errorText(err);
      }
    },
    [replace, markDead],
  );

  const stop = useCallback(
    async (id: string) => {
      stopped.current.add(id);
      writeJSON(STOPPED_KEY, [...stopped.current].slice(-MAX_REMEMBERED));
      setBump((n) => n + 1);
      try {
        await api(`/api/v1/my/party-orders/${id}/live-location`, { method: "DELETE" });
      } catch {
        /* **والإيقافُ محليٌّ نافذٌ** — المتصفّحُ لا يرسل بعده شيئاً. */
      }
      reload();
    },
    [reload],
  );

  // **ما يشارك الآن** — جوابُه «نعم» والطلبُ مفتوحٌ ولم يوقفه بيده ولم ينتهِ.
  const sharingKey = (orders ?? [])
    .filter((o) => o.open && o.answer === "yes" && !stopped.current.has(o.order_id) && !dead.current.has(o.order_id))
    .map((o) => o.order_id)
    .join(",");
  // eslint-disable-next-line react-hooks/exhaustive-deps
  const sharingIds = useMemo(() => (sharingKey ? sharingKey.split(",") : []), [sharingKey, bump]);

  // **عنوانُه المحفوظ** — مصدرُ الاحتياط الوحيد، يُقرأ وقتَ الحاجة لا يُعيد تشغيل المراقبة.
  const fallback = useRef<{ lat: number; lng: number } | null>(null);
  fallback.current =
    address && Number.isFinite(address.lat) && Number.isFinite(address.lng) && (address.lat !== 0 || address.lng !== 0)
      ? { lat: address.lat, lng: address.lng }
      : null;

  useEffect(() => {
    if (sharingIds.length === 0) return;
    const ids = sharingIds;
    const lastSent = new Map<string, number>();
    const fallbackSent = new Set<string>();
    let last: { lat: number; lng: number; acc: number | null } | null = null;
    let watchId: number | null = null;
    let timer: ReturnType<typeof setInterval> | null = null;

    const post = (id: string, lat: number, lng: number, acc: number | null) => {
      lastSent.set(id, performance.now());
      const body: Record<string, number> = { lat, lng };
      if (acc != null) body.accuracy_m = acc;
      api(`/api/v1/my/party-orders/${id}/live-location`, { method: "POST", body: JSON.stringify(body) }).catch((err) => {
        const code = errCode(err);
        if (code === "live_location_inactive" || code === "not_found") markDead(id);
        else if (code === "party_not_consented") reload();
        /* **و`rate_limited` وانقطاعُ الشبكة** — النقطةُ التاليةُ تحلّ محلّها. */
      });
    };
    const sendAll = (force: boolean) => {
      if (!last) return;
      const now = performance.now();
      for (const id of ids) {
        if (!force && now - (lastSent.get(id) ?? -Infinity) < MIN_GAP_MS) continue;
        post(id, last.lat, last.lng, last.acc);
      }
    };
    // **منعُ الموقع أو غيابُه** — عنوانُه المحفوظ مرّةً واحدةً لكلّ طلب، بلا دقّة.
    const useFallback = () => {
      const f = fallback.current;
      if (!f) return;
      for (const id of ids) {
        if (fallbackSent.has(id)) continue;
        fallbackSent.add(id);
        post(id, f.lat, f.lng, null);
      }
    };

    if (typeof navigator === "undefined" || !navigator.geolocation) {
      useFallback();
      return;
    }
    try {
      watchId = navigator.geolocation.watchPosition(
        (p) => {
          last = {
            lat: p.coords.latitude,
            lng: p.coords.longitude,
            acc: Number.isFinite(p.coords.accuracy) ? Math.round(p.coords.accuracy) : null,
          };
          sendAll(false);
        },
        (e) => {
          if (e.code === e.PERMISSION_DENIED || e.code === e.POSITION_UNAVAILABLE) {
            if (!last) useFallback();
          }
        },
        { enableHighAccuracy: true, maximumAge: 0, timeout: 20_000 },
      );
      timer = setInterval(() => sendAll(false), EVERY_MS);
    } catch {
      useFallback();
    }
    return () => {
      if (watchId != null) {
        try {
          navigator.geolocation.clearWatch(watchId);
        } catch {
          /* أُغلقت */
        }
      }
      if (timer) clearInterval(timer);
    };
  }, [sharingIds, markDead, reload]);

  const value = useMemo<PartyState>(
    () => ({ orders, shownId, show: setShownId, hide: () => setShownId(""), sharingIds, answer, stop }),
    [orders, shownId, sharingIds, answer, stop],
  );
  return <Ctx.Provider value={value}>{children}</Ctx.Provider>;
}
